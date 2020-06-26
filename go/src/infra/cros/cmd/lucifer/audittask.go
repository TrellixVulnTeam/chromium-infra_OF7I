// Copyright 2020 The Chromium OS Authors. All rights reserved.
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
	"infra/cros/cmd/lucifer/internal/autotest"
	"infra/cros/cmd/lucifer/internal/event"
	"infra/cros/cmd/lucifer/internal/flagx"
	"infra/cros/cmd/lucifer/internal/osutil"
)

type auditTaskCmd struct {
	commonOpts
	name    string
	host    string
	actions []string
}

func (auditTaskCmd) Name() string {
	return "audittask"
}

func (auditTaskCmd) Synopsis() string {
	return "Run an audit task"
}

func (auditTaskCmd) Usage() string {
	return `audittask [FLAGS]

lucifer audittask runs actions to audit a host.
Updating the status of a running job is delegated to the calling
process.  Status update events are printed to stdout, and the calling
process should perform the necessary updates.
`
}

func (c *auditTaskCmd) SetFlags(f *flag.FlagSet) {
	c.commonOpts.Register(f)
	f.StringVar(&c.host, "host", "", "Host on which to run audit task")
	f.Var(flagx.CommaList(&c.actions), "actions", "Host actions to execute.")
}

func (c *auditTaskCmd) Execute(ctx context.Context, f *flag.FlagSet, args ...interface{}) subcommands.ExitStatus {
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

func (c *auditTaskCmd) innerExecute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) error {
	if err := c.validateFlags(); err != nil {
		return err
	}
	ctx, res, err := commonSetup(ctx, c.commonOpts)
	if err != nil {
		return err
	}
	defer res.Close()

	ac := res.apiClient()
	var errors []error
	for _, a := range c.actions {
		if err := c.runAction(ctx, ac, a); err != nil {
			// Allows to run all actions and collect errors.
			errors = append(errors, err)
		}
	}
	if len(errors) > 0 {
		sendHostStatus(ctx, ac, []string{c.host}, event.HostNeedsReset)
		return fmt.Errorf("errors %s", errors)
	}
	return nil
}

func (c *auditTaskCmd) runAction(ctx context.Context, ac *api.Client, a string) error {
	s := ac.Logger().Step(a)
	defer s.Close()

	resultsDir := filepath.Join(c.resultsDir, a)
	if err := os.MkdirAll(resultsDir, 0777); err != nil {
		s.Printf("Run action %s: %s", a, err)
		s.Exception()
		return errors.Errorf("audit run action %s: %s", a, err)
	}

	args := autotest.AuditTaskArgs{
		Hostname:     c.host,
		ResultsDir:   resultsDir,
		HostInfoFile: c.hostInfoStorePath(c.host),
		Actions:      []string{a},
	}
	cmd := autotest.AuditTaskCommand(c.autotestConfig(), &args)
	cmd.Stdout = ac.Logger().RawWriter()
	cmd.Stderr = ac.Logger().RawWriter()

	if err := wrapRunError(osutil.RunWithAbort(ctx, cmd)); err != nil {
		s.Printf("Error running %#v command: %s", a, err)
		s.Exception()
		return errors.Errorf("audit run action %s: %s", a, err)
	}
	return nil
}

func (c *auditTaskCmd) validateFlags() error {
	var errs []error
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
