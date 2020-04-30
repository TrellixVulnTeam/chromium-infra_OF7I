// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sync"

	"github.com/pkg/errors"
	"go.opencensus.io/trace"

	"infra/cros/cmd/lucifer/internal/api"
	"infra/cros/cmd/lucifer/internal/autotest"
	"infra/cros/cmd/lucifer/internal/autotest/atutil"
	"infra/cros/cmd/lucifer/internal/event"
)

func doProvisioningStep(ctx context.Context, c *testCmd, ac *api.Client) (err error) {
	ctx, span := trace.StartSpan(ctx, "Provision")
	defer span.End()
	s := ac.Step("Provision")
	defer s.Close()
	defer func() {
		if err != nil {
			s.Printf("Error: %s", err)
			s.Exception()
		}
	}()
	event.Send(event.Provisioning)
	var wg sync.WaitGroup
	errs := make(chan error, len(c.hosts))
	for _, h := range c.hosts {
		wg.Add(1)
		go func(h string) {
			defer wg.Done()
			errs <- provision(ctx, c, h)
		}(h)
	}
	wg.Wait()
	close(errs)
	for err = range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

func doPrejobStep(ctx context.Context, c *testCmd, ac *api.Client) (err error) {
	ctx, span := trace.StartSpan(ctx, "Pre-job")
	defer span.End()
	if len(c.hosts) != 1 {
		return errors.New("Prejob task only supported for one host")
	}
	t := c.adminTask(c.hosts[0])
	t.Type = c.prejobTask
	s := ac.Step(t.Type.String())
	defer s.Close()
	defer func() {
		if err != nil {
			s.Printf("Error: %s", err)
			s.Exception()
		}
	}()
	if err := runPrejobTask(ctx, c.mainJob(), t, ac); err != nil {
		return errors.Wrap(err, "run prejob task")
	}
	return nil
}

func doRunningStep(ctx context.Context, c *testCmd, ac *api.Client) (err error) {
	ctx, span := trace.StartSpan(ctx, "Test")
	defer span.End()
	s := ac.Step("Test")
	defer s.Close()
	defer func() {
		if err != nil {
			s.Printf("Error: %s", err)
			s.Exception()
		}
	}()
	event.Send(event.Running)
	if c.isHostless() {
		log.Printf("Running as hostless job")
		t := c.hostlessTest()
		r, err := atutil.RunAutoserv(ctx, c.mainJob(), t, s.RawWriter())
		return handleTest(r, err)
	}
	log.Printf("Running as host job")
	t := c.hostTest()
	sendHostStatus(ctx, ac, c.hosts, event.HostRunning)
	r, err := atutil.RunAutoserv(ctx, c.mainJob(), t, s.RawWriter())
	err = handleTest(r, err)
	doGatheringSubstep(ctx, c.autotestConfig(), ac, t, r)
	updateHostStatus(ctx, ac, c, r)
	return err
}

func doGatheringSubstep(ctx context.Context, c autotest.Config, ac *api.Client, t *atutil.HostTest, r *atutil.Result) {
	ctx, span := trace.StartSpan(ctx, "Gather")
	defer span.End()
	s := ac.Step("Gather")
	defer s.Close()
	event.Send(event.Gathering)
	if r.Signaled() {
		if err := collectCrashinfo(c, t.Hosts, t.ResultsDir); err != nil {
			s.Printf("Error during crashinfo collection: %s", err)
			s.Exception()
		}
	}
}

// collectCrashinfo runs autoserv to collect crashinfo files.
func collectCrashinfo(c autotest.Config, hosts []string, resultsDir string) error {
	cmd := autotest.AutoservCommand(c, &autotest.AutoservArgs{
		Hosts:              hosts,
		ResultsDir:         resultsDir,
		UseExistingResults: true,
		CollectCrashinfo:   true,
	})
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func doParsingStep(ctx context.Context, c *testCmd, ac *api.Client) (err error) {
	ctx, span := trace.StartSpan(ctx, "Parse")
	defer span.End()
	event.Send(event.Parsing)
	var d string
	s := ac.Step("Upload results to TKO")
	defer s.Close()
	defer func() {
		if err != nil {
			s.Printf("Error: %s", err)
			s.Exception()
		}
	}()
	if c.isHostless() {
		d = c.hostlessTest().ResultsDir
	} else {
		d = c.hostTest().ResultsDir
	}
	n, err := atutil.TKOParse(c.autotestConfig(), d, c.tkoLevel(), s.RawWriter())
	if err != nil {
		return errors.Wrap(err, "run tko/parse")
	}
	for i := 0; i < n; i++ {
		event.SendWithMsg(event.TestFailed, "test")
	}
	if n > 0 {
		s.Printf("%d tests failed", n)
	}
	return nil
}

func updateHostStatus(ctx context.Context, ac *api.Client, c *testCmd, r *atutil.Result) {
	if !c.runReset && !r.Success() {
		sendHostStatus(ctx, ac, c.hosts, event.HostNeedsReset)
		return
	}
	shouldReboot := r.Aborted ||
		c.rebootAfter == "always" ||
		c.rebootAfter == "if_all_tests_passed" && r.Success() ||
		r.TestsFailed > 0
	if shouldReboot {
		sendHostStatus(ctx, ac, c.hosts, event.HostNeedsCleanup)
	} else {
		sendHostStatus(ctx, ac, c.hosts, event.HostReady)
	}
}

// provision provisions a single host.  This function is intended to
// be run in a goroutine and cannot modify its pointer arguments.
func provision(ctx context.Context, c *testCmd, host string) (err error) {
	p := c.provisionJob(host)
	// TODO(ayatane): LogDog output is single threaded, so
	// gathering output from potentially multiple provisions in
	// parallel is non-trivial, so just discard for now.
	_, err = atutil.RunAutoserv(ctx, c.mainJob(), p, ioutil.Discard)
	if err != nil {
		event.SendWithMsg(event.HostNeedsRepair, p.Host)
		if _, err2 := atutil.TKOParse(c.autotestConfig(), p.ResultsDir, c.tkoLevel(),
			ioutil.Discard); err == nil {
			err = err2
		}
		return err
	}
	event.SendWithMsg(event.HostReadyToRun, p.Host)
	return nil
}

// handleTest handles the result from an autoserv test.
func handleTest(r *atutil.Result, err error) error {
	if err != nil {
		event.SendWithMsg(event.TestFailed, "autoserv")
		if r.Exit == 0 {
			// Set exit status for calculating status events in gathering.
			r.Exit = 1
		}
		return err
	}
	event.SendWithMsg(event.TestPassed, "autoserv")
	return nil
}

var testTaskEvents = map[atutil.AdminTaskType]struct {
	pass event.Event
	fail event.Event
}{
	atutil.Cleanup: {event.HostClean, event.HostNeedsRepair},
	atutil.Reset:   {event.HostClean, event.HostNeedsRepair},
	atutil.Verify:  {event.HostReady, event.HostNeedsRepair},
}

func runPrejobTask(ctx context.Context, a *atutil.MainJob, t *atutil.AdminTask, ac *api.Client) (err error) {
	te := testTaskEvents[t.Type]
	defer func() {
		if err == nil {
			sendHostStatus(ctx, ac, []string{t.Host}, te.pass)
		} else {
			sendHostStatus(ctx, ac, []string{t.Host}, te.fail)
		}
	}()
	_, err = atutil.RunAutoserv(ctx, a, t, ac.Logger().RawWriter())
	if err != nil {
		return fmt.Errorf("task %s failed: %s", t.Type, err)
	}
	return nil
}
