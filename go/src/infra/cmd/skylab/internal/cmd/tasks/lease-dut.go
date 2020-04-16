// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package tasks

import (
	"context"
	"fmt"
	"infra/cmd/skylab/internal/cmd/utils"
	"time"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"

	skycmdlib "infra/cmd/skylab/internal/cmd/cmdlib"
	"infra/cmd/skylab/internal/site"
	"infra/cmd/skylab/internal/userinput"
	"infra/cmdsupport/cmdlib"
	"infra/libs/skylab/swarming"
)

const dayInMinutes = 24 * 60

// LeaseDut subcommand: Lease a DUT for debugging.
var LeaseDut = &subcommands.Command{
	UsageLine: "lease-dut HOST\n\tskylab lease-dut -model MODEL",
	ShortDesc: "lease DUT for debugging",
	LongDesc: `Lease DUT for debugging.

This subcommand's behavior is subject to change without notice.
Do not build automation around this subcommand.`,
	CommandRun: func() subcommands.CommandRun {
		c := &leaseDutRun{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		// use a float so that large values passed on the command line are NOT wrapped.
		c.Flags.Float64Var(&c.leaseMinutes, "minutes", 60, "Duration of lease.")
		c.Flags.StringVar(&c.leaseReason, "reason", "", "The reason to perform this lease, it must match crbug.com/NNNN or b/NNNN.")
		// TODO(gregorynisbet):
		// if a model is provided, then we necessarily target DUT_POOL_QUOTA and only repair-failed DUTs until
		// a better policy can be implemented.
		c.Flags.StringVar(&c.model, "model", "", "Leases may optionally target a model instead of a hostname")
		return c
	},
}

type leaseDutRun struct {
	subcommands.CommandRunBase
	authFlags    authcli.Flags
	envFlags     skycmdlib.EnvFlags
	leaseMinutes float64
	leaseReason  string
	model        string
}

func (c *leaseDutRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *leaseDutRun) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	hasOneHostname := len(args) == 1
	hasModel := c.model != ""
	if !hasOneHostname && !hasModel {
		return cmdlib.NewUsageError(c.Flags, "exactly one host or one model required")
	}
	if c.leaseMinutes < 0 {
		return cmdlib.NewUsageError(c.Flags, fmt.Sprintf("minutes to lease (%d) cannot be negative", int64(c.leaseMinutes)))
	}
	if c.leaseMinutes >= dayInMinutes {
		return cmdlib.NewUsageError(c.Flags, "Lease duration (%d minutes) cannot exceed 1 day [%d minutes]", int64(c.leaseMinutes), dayInMinutes)
	}
	if len(c.leaseReason) > 30 {
		return cmdlib.NewUsageError(c.Flags, "the lease reason is limited in 30 characters")
	}
	if userinput.ValidBug(c.leaseReason) {
		return cmdlib.NewUsageError(c.Flags, "the lease reason must match crbug.com/NNNN or b/NNNN")
	}
	host := skycmdlib.FixSuspiciousHostname(args[0])
	if host != args[0] {
		fmt.Fprintf(a.GetErr(), "correcting (%s) to (%s)\n", args[0], host)
	}

	ctx := cli.GetContext(a, c, env)

	leaseDuration := time.Duration(c.leaseMinutes) * time.Minute

	if hasOneHostname {
		return c.leaseDutByHostname(ctx, a, leaseDuration, host)
	}
	panic("lease by model not yet implemented")
}

// leaseDutByHostname leases a DUT by hostname and schedules a follow-up repair task
func (c *leaseDutRun) leaseDutByHostname(ctx context.Context, a subcommands.Application, leaseDuration time.Duration, host string) error {
	h, err := cmdlib.NewHTTPClient(ctx, &c.authFlags)
	if err != nil {
		return errors.Annotate(err, "failed to create http client").Err()
	}
	e := c.envFlags.Env()
	client, err := swarming.New(ctx, h, e.SwarmingService)
	if err != nil {
		return errors.Annotate(err, "failed to create Swarming client").Err()
	}

	creator := utils.TaskCreator{
		Client:      client,
		Environment: e,
	}
	id, err := creator.LeaseTask(ctx, host, int(leaseDuration.Seconds()), c.leaseReason)
	if err != nil {
		return err
	}
	fmt.Fprintf(a.GetOut(), "Created lease task for host %s: %s\n", host, swarming.TaskURL(e.SwarmingService, id))
	scheduleRepairTaskForLater(ctx, &creator, a, leaseDuration, host)
	fmt.Fprintf(a.GetOut(), "Waiting for task to start; lease isn't active yet\n")
poll:
	for {
		result, err := client.GetTaskState(ctx, id)
		if err != nil {
			return err
		}
		if len(result.States) != 1 {
			return errors.Reason("Got unexpected task states: %#v; expected one state", result.States).Err()
		}
		switch s := result.States[0]; s {
		case "PENDING":
			time.Sleep(time.Duration(10) * time.Second)
		case "RUNNING":
			break poll
		default:
			return errors.Reason("Got unexpected task state %#v", s).Err()
		}
	}
	// TODO(ayatane): The time printed here may be off by the poll interval above.
	fmt.Fprintf(a.GetOut(), "DUT leased until %s\n", time.Now().Add(leaseDuration).Format(time.RFC1123))
	return nil
}

func scheduleRepairTaskForLater(ctx context.Context, creator *utils.TaskCreator, a subcommands.Application, leaseDuration time.Duration, host string) {
	// expirde time is time for lease + expiration time of lease_task + additional time for any delays
	expirationSec := int(leaseDuration.Seconds()) + 900
	id, err := creator.RepairTask(ctx, host, nil, expirationSec)
	if err == nil {
		fmt.Fprintf(a.GetOut(), "Created repair task %s\n", swarming.TaskURL(creator.Environment.SwarmingService, id))
	} else {
		fmt.Fprint(a.GetOut(), "Error: Repair task was not created\n")
	}
}
