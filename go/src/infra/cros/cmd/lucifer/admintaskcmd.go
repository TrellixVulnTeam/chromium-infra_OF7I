// Copyright 2018 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/subcommands"
	"github.com/pkg/errors"

	"infra/cros/cmd/lucifer/internal/api"
	"infra/cros/cmd/lucifer/internal/autotest/atutil"
	"infra/cros/cmd/lucifer/internal/event"
	"infra/cros/cmd/lucifer/internal/flagx"
)

type adminTaskCmd struct {
	commonOpts
	host     string
	taskType atutil.AdminTaskType
}

func (adminTaskCmd) Name() string {
	return "admintask"
}
func (adminTaskCmd) Synopsis() string {
	return "Run an admin task"
}
func (adminTaskCmd) Usage() string {
	return `admintask [FLAGS]

lucifer admintask runs an admin task against a host.
Updating the status of a running job is delegated to the calling
process.  Status update events are printed to stdout, and the calling
process should perform the necessary updates.
`
}

func (c *adminTaskCmd) SetFlags(f *flag.FlagSet) {
	c.commonOpts.Register(f)
	c.taskType = atutil.Verify
	f.StringVar(&c.host, "host", "",
		"Host on which to run task")
	f.Var(flagx.TaskType(&c.taskType, flagx.RejectNoTask), "task",
		"Task to run (default verify)")
}

func (c *adminTaskCmd) Execute(ctx context.Context, f *flag.FlagSet, args ...interface{}) subcommands.ExitStatus {
	if err := c.innerExecute(ctx, f, args...); err != nil {
		fmt.Fprintf(os.Stderr, "lucifer: %s\n", err)
		switch err := err.(type) {
		case exitError:
			return err.ExitStatus()
		default:
			return subcommands.ExitFailure
		}
	}
	return subcommands.ExitSuccess
}

func (c *adminTaskCmd) innerExecute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) error {
	if err := c.validateFlags(); err != nil {
		return err
	}
	ctx, res, err := commonSetup(ctx, c.commonOpts)
	if err != nil {
		return err
	}
	defer res.Close()

	ac := res.apiClient()
	t := c.adminTask()
	s := ac.Step(t.Type.String())
	defer s.Close()
	if err := runTask(ctx, ac, c.mainJob(), t); err != nil {
		s.Printf("Error running admin task: %s", err)
		s.Exception()
		return err
	}
	return nil
}

func (c *adminTaskCmd) validateFlags() error {
	errs := make([]error, 0, 5)
	if c.abortSock == "" {
		errs = append(errs, errors.New("-abortsock must be provided"))
	}
	if c.host == "" {
		errs = append(errs, errors.New("-host must be provided"))
	}
	if c.resultsDir == "" {
		errs = append(errs, errors.New("-resultsdir must be provided"))
	}
	if len(errs) > 0 {
		return usageError{fmt.Errorf("Errors occurred during argument parsing: %s", errs)}
	}
	return nil
}

func (c *adminTaskCmd) mainJob() *atutil.MainJob {
	return &atutil.MainJob{
		AutotestConfig:   c.autotestConfig(),
		ResultsDir:       c.resultsDir,
		UseLocalHostInfo: true,
	}
}

func (c *adminTaskCmd) adminTask() *atutil.AdminTask {
	return &atutil.AdminTask{
		Type:       c.taskType,
		Host:       c.host,
		ResultsDir: filepath.Join(c.resultsDir, "admintask"),
	}
}

var taskEvents = map[atutil.AdminTaskType]struct {
	pass event.Event
	fail event.Event
}{
	atutil.Cleanup: {event.HostClean, event.HostNeedsRepair},
	atutil.Repair:  {event.HostReady, event.HostFailedRepair},
	atutil.Reset:   {event.HostClean, event.HostNeedsRepair},
	atutil.Verify:  {event.HostReady, event.HostNeedsRepair},
}

func runTask(ctx context.Context, ac *api.Client, m *atutil.MainJob, t *atutil.AdminTask) (err error) {
	event.Send(event.Starting)
	defer event.Send(event.Completed)
	te := taskEvents[t.Type]
	defer func() {
		var e event.Event
		if err == nil {
			e = te.pass
		} else {
			e = te.fail
		}
		sendHostStatus(ctx, ac, []string{t.Host}, e)
	}()
	_, err = atutil.RunAutoserv(ctx, m, t, ac.Logger().RawWriter())
	if err != nil {
		return fmt.Errorf("task %s failed: %s", t.Type, err)
	}
	return nil
}
