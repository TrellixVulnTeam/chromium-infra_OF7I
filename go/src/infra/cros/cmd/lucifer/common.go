// Copyright 2018 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/google/subcommands"
	"github.com/pkg/errors"

	"infra/cros/cmd/lucifer/internal/abortsock"
	"infra/cros/cmd/lucifer/internal/api"
	"infra/cros/cmd/lucifer/internal/logdog"
	"infra/cros/cmd/lucifer/internal/metrics"
	"infra/cros/cmd/lucifer/internal/osutil"
)

// exitError interface is for errors that can be returned from
// subcommands with more detail.
type exitError interface {
	error
	ExitStatus() subcommands.ExitStatus
}

type usageError struct {
	error
}

func (e usageError) ExitStatus() subcommands.ExitStatus {
	return subcommands.ExitUsageError
}

// commonOpts contains common command options.
type commonOpts struct {
	abortSock   string
	autotestDir string
	gcpProject  string
	logdogFile  string
	resultsDir  string
}

// Register adds flags for common options.
func (c *commonOpts) Register(f *flag.FlagSet) {
	f.StringVar(&c.abortSock, "abortsock", "",
		"Abort socket")
	f.StringVar(&c.autotestDir, "autotestdir", "/usr/local/autotest",
		"Autotest directory")
	f.StringVar(&c.gcpProject, "gcp-project", "chromeos-lucifer",
		"GCP project")
	f.StringVar(&c.logdogFile, "logdog-file", "",
		"File for LogDog output")
	f.StringVar(&c.resultsDir, "resultsdir", "",
		"Results directory")
}

const hostInfoSubDir = "host_info_store"

func (c *commonOpts) hostInfoStorePath(host string) string {
	return filepath.Join(c.resultsDir, hostInfoSubDir, fmt.Sprintf("%s.store", host))
}

// Setup sets up common resources for Lucifer commands.  If the
// returned error is nil, the caller is responsible for calling Close
// on the returned commonResources.
func commonSetup(ctx context.Context, c commonOpts) (ctx2 context.Context, res *commonResources, err error) {
	res = &commonResources{}
	defer func(res io.Closer) {
		if err != nil {
			res.Close()
		}
	}(res)

	// Set up abort socket.
	s, err := abortsock.Open(c.abortSock)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "error opening abort socket")
	}
	res.abortsock = s
	ctx = s.AttachContext(ctx)

	// Set up LogDog output.
	if c.logdogFile != "" {
		f, err := os.OpenFile(c.logdogFile, os.O_WRONLY, 0666)
		if err != nil {
			return ctx, nil, errors.Wrap(err, "error opening logdog file")
		}
		log.Printf("LogDog file/pipe is %s", c.logdogFile)
		res.logdog = f
		log.Print("Switching log output to LogDog")
		log.SetOutput(f)
	}

	// Set up metrics.
	ctx, res.metrics = metrics.Setup(ctx, res.Logger(), metrics.Config{
		GCPProject: c.gcpProject,
	})
	return ctx, res, nil
}

// commonResources wraps common resources that need closing.
type commonResources struct {
	// Things to close.
	metrics   *metrics.Client
	abortsock *abortsock.AbortSock
	logdog    io.WriteCloser
}

// apiClient returns an API client.
func (c *commonResources) apiClient() *api.Client {
	return api.NewClient(c.Logger(), c.metrics)
}

// Logger returns a Logger that supports LogDog features.  If LogDog
// is not set up, suitable text formatted logs are printed to stderr
// instead.
func (c *commonResources) Logger() logdog.Logger {
	if c.logdog == nil {
		return logdog.NewTextLogger(os.Stderr)
	}
	return logdog.NewLogger(c.logdog)
}

// Close closes all resources and returns the first error encountered.
func (c *commonResources) Close() error {
	var err error
	if c.metrics != nil {
		if err2 := c.metrics.Close(); err == nil {
			err = err2
		}
		c.metrics = nil
	}
	if c.logdog != nil {
		log.Print("LogDog logs end here")
		log.SetOutput(os.Stderr)
		log.Print("Switching log output back from LogDog")
		if err2 := c.logdog.Close(); err == nil {
			err = err2
		}
		c.logdog = nil
	}
	if c.abortsock != nil {
		if err2 := c.abortsock.Close(); err == nil {
			err = err2
		}
		c.abortsock = nil
	}
	return err
}

// verifySkylabFlags check the cmd flags. Returns error if they were not
// provisioned as Skylab job.
func verifySkylabFlags(t testCmd) error {
	if t.level != "SKYLAB_PROVISION" {
		return errors.New("this command only accepts Skylab tests")
	}
	return nil
}

func wrapRunError(r osutil.RunResult, err error) error {
	var reason error
	switch {
	case !r.Started:
		reason = fmt.Errorf("process failed to start")
	case r.Aborted:
		reason = fmt.Errorf("process aborted")
	case r.ExitStatus != 0:
		reason = fmt.Errorf("process fail with exit code %d", r.ExitStatus)
	}
	switch {
	case reason == nil:
		return err
	case err == nil:
		return reason
	default:
		return fmt.Errorf("%s: %s", reason, err)
	}
}
