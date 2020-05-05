// Copyright 2019 The Chromium OS Authors. All rights reserved.
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
	"infra/cros/cmd/lucifer/internal/autotest/atutil"
	"infra/cros/cmd/lucifer/internal/autotest/dutprep"
	"infra/cros/cmd/lucifer/internal/event"
	"infra/cros/cmd/lucifer/internal/flagx"
	"infra/cros/cmd/lucifer/internal/osutil"
)

type deployTaskCmd struct {
	commonOpts
	host    string
	actions []dutprep.Action
}

func (deployTaskCmd) Name() string {
	return "deploytask"
}
func (deployTaskCmd) Synopsis() string {
	return "Run a deploy task"
}
func (deployTaskCmd) Usage() string {
	return `deploytask [FLAGS]

lucifer deploytask runs a task to deploy a host.
Updating the status of a running job is delegated to the calling
process.  Status update events are printed to stdout, and the calling
process should perform the necessary updates.
`
}

func (c *deployTaskCmd) SetFlags(f *flag.FlagSet) {
	c.commonOpts.Register(f)
	f.StringVar(&c.host, "host", "",
		"Host on which to run deploy task")
	f.Var(flagx.DeployActionList(&c.actions), "actions",
		"Host preparation actions to execute.")
}

func (c *deployTaskCmd) Execute(ctx context.Context, f *flag.FlagSet, args ...interface{}) subcommands.ExitStatus {
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

func (c *deployTaskCmd) innerExecute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) error {
	if err := c.validateFlags(); err != nil {
		return err
	}
	ctx, res, err := commonSetup(ctx, c.commonOpts)
	if err != nil {
		return err
	}
	defer res.Close()

	ac := res.apiClient()
	for _, a := range dutprep.SortActions(c.actions) {
		if err := c.runDeployAction(ctx, ac, a); err != nil {
			fmt.Fprintf(os.Stderr, "deploy failed: %s\n", err)
			sendHostStatus(ctx, ac, []string{c.host}, event.HostNeedsDeploy)
			return err
		}
	}
	return c.runRepair(ctx, ac)
}

func (c *deployTaskCmd) runDeployAction(ctx context.Context, ac *api.Client, a dutprep.Action) error {
	s := ac.Logger().Step(a.String())
	defer s.Close()

	resultsDir := filepath.Join(c.resultsDir, a.String())
	if err := os.MkdirAll(resultsDir, 0777); err != nil {
		s.Printf("Error creating directory %s: %s", resultsDir, err)
		s.Exception()
		return err
	}

	args := autotest.DutPreparationArgs{
		Hostname:     c.host,
		ResultsDir:   resultsDir,
		HostInfoFile: c.hostInfoStorePath(c.host),
		Actions:      []string{a.Arg()},
	}
	cmd := autotest.DutPreparationCommand(c.autotestConfig(), &args)
	cmd.Stdout = ac.Logger().RawWriter()
	cmd.Stderr = ac.Logger().RawWriter()

	if err := wrapRunError(osutil.RunWithAbort(ctx, cmd)); err != nil {
		s.Printf("Error running dut preparation command: %s", err)
		s.Exception()
		return err
	}
	return nil
}

func (c *deployTaskCmd) runRepair(ctx context.Context, ac *api.Client) error {
	t := c.repairTask()
	s := ac.Logger().Step(t.Type.String())
	defer s.Close()
	if err := runTask(ctx, ac, c.mainJob(), t); err != nil {
		s.Printf("Error running repair task: %s", err)
		s.Exception()
		return err
	}
	return nil
}

func (c *deployTaskCmd) validateFlags() error {
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

func (c *deployTaskCmd) mainJob() *atutil.MainJob {
	return &atutil.MainJob{
		AutotestConfig:   c.autotestConfig(),
		ResultsDir:       c.resultsDir,
		UseLocalHostInfo: true,
	}
}

func (c *deployTaskCmd) repairTask() *atutil.AdminTask {
	return &atutil.AdminTask{
		Type:       atutil.Repair,
		Host:       c.host,
		ResultsDir: filepath.Join(c.resultsDir, "repair"),
	}
}
