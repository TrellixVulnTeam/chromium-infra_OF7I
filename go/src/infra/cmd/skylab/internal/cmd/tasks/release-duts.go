// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package tasks

import (
	"context"
	"fmt"
	swarming_api "go.chromium.org/luci/common/api/swarming/swarming/v1"
	"io"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"

	skycmdlib "infra/cmd/skylab/internal/cmd/cmdlib"
	"infra/cmd/skylab/internal/site"
	"infra/cmdsupport/cmdlib"
	"infra/libs/skylab/swarming"
)

// ReleaseDuts subcommand: Release a DUT previously leased via LeaseDuts.
var ReleaseDuts = &subcommands.Command{
	UsageLine: "release-duts HOST [HOST...]",
	ShortDesc: "release DUTs which are previously leased via lease-dut [DEPRECATED--please use crosfleet (go/crosfleet-cli)]",
	LongDesc: `[DEPRECATED--please use crosfleet (go/crosfleet-cli)]

release DUTs which are previously leased via lease-dut.

This subcommand's behavior is subject to change without notice.
Do not build automation around this subcommand.`,
	CommandRun: func() subcommands.CommandRun {
		c := &releaseDutsRun{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		return c
	},
}

type releaseDutsRun struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
	envFlags  skycmdlib.EnvFlags
}

func (c *releaseDutsRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *releaseDutsRun) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	if c.Flags.NArg() == 0 {
		return cmdlib.NewUsageError(c.Flags, "must specify at least 1 DUT")
	}

	ctx := cli.GetContext(a, c, env)
	h, err := cmdlib.NewHTTPClient(ctx, &c.authFlags)
	if err != nil {
		return errors.Annotate(err, "failed to create http client").Err()
	}
	e := c.envFlags.Env()
	client, err := swarming.NewClient(h, e.SwarmingService)
	if err != nil {
		return errors.Annotate(err, "failed to create Swarming client").Err()
	}

	return c.cancelLeaseTasks(ctx, a.GetOut(), client)
}

func (c *releaseDutsRun) cancelLeaseTasks(ctx context.Context, w io.Writer, client *swarming.Client) error {
	hostnames := c.Flags.Args()
	errs := make(errors.MultiError, 0)
	for _, h := range hostnames {
		fmt.Fprintf(w, "Canceling lease task for host: %s\n", h)
		err := c.cancelLeaseTaskForHost(ctx, w, client, h)
		if err != nil {
			fmt.Fprintf(w, "Failed to cancel: %s\n", err.Error())
			errs = append(errs, err)
			continue
		}
	}
	if errs.First() != nil {
		return errs
	}
	return nil
}

func (c *releaseDutsRun) cancelLeaseTaskForHost(ctx context.Context, w io.Writer, client *swarming.Client, hostname string) error {
	ic, err := getUFSClient(ctx, &c.authFlags, c.envFlags.Env())
	if err != nil {
		return err
	}
	model, err := getModelForHost(ctx, ic, hostname)
	if err != nil {
		return err
	}
	// Not all lease tasks are directly tagged with the hostname,
	// so we search for all active leases of the associated model,
	// then iterate through the results to find lease(s) with the
	// hostname's bot ID.
	leaseTasksForModel, err := client.GetActiveLeaseTasksForModel(ctx, model)
	if err != nil {
		return err
	}
	var leaseTasksForHost []*swarming_api.SwarmingRpcsTaskResult
	botID, err := client.DutNameToBotID(ctx, hostname)
	if err != nil {
		return err
	}
	for _, t := range leaseTasksForModel {
		if t.BotId == botID {
			leaseTasksForHost = append(leaseTasksForHost, t)
		}
	}

	if len(leaseTasksForHost) == 0 {
		fmt.Fprintf(w, "Found no lease tasks for host %s\n", hostname)
		return nil
	}
	for _, t := range leaseTasksForHost {
		err = client.CancelTask(ctx, t.TaskId)
		if err != nil {
			return err
		}
		fmt.Fprintf(w, "Successfully killed task %s, DUT %s is released\n", t.TaskId, hostname)
	}
	return nil
}
