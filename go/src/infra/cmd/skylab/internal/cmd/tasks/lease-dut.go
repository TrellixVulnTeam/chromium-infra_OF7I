// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package tasks

import (
	"context"
	"fmt"
	"time"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	swarming_api "go.chromium.org/luci/common/api/swarming/swarming/v1"
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
	UsageLine: "lease-dut HOST",
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
		return c
	},
}

type leaseDutRun struct {
	subcommands.CommandRunBase
	authFlags    authcli.Flags
	envFlags     skycmdlib.EnvFlags
	leaseMinutes float64
	leaseReason  string
}

func (c *leaseDutRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *leaseDutRun) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	if len(args) != 1 {
		return cmdlib.NewUsageError(c.Flags, "exactly one host required")
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
	h, err := cmdlib.NewHTTPClient(ctx, &c.authFlags)
	if err != nil {
		return errors.Annotate(err, "failed to create http client").Err()
	}
	e := c.envFlags.Env()
	client, err := swarming.New(ctx, h, e.SwarmingService)
	if err != nil {
		return errors.Annotate(err, "failed to create Swarming client").Err()
	}

	lt := leaseTask{
		host:     host,
		duration: time.Duration(c.leaseMinutes) * time.Minute,
		reason:   c.leaseReason,
	}
	id, err := createLeaseTask(ctx, client, e, lt)
	if err != nil {
		return err
	}
	fmt.Fprintf(a.GetOut(), "Created lease task for host %s: %s\n", host, swarming.TaskURL(e.SwarmingService, id))
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
	fmt.Fprintf(a.GetOut(), "DUT leased until %s\n", time.Now().Add(lt.duration).Format(time.RFC1123))
	return nil
}

type leaseTask struct {
	host     string
	duration time.Duration
	reason   string
}

func createLeaseTask(ctx context.Context, t *swarming.Client, e site.Environment, lt leaseTask) (taskID string, err error) {
	c := []string{"/bin/sh", "-c", `while true; do sleep 60; echo Zzz...; done`}
	slices := []*swarming_api.SwarmingRpcsTaskSlice{{
		ExpirationSecs: 600,
		Properties: &swarming_api.SwarmingRpcsTaskProperties{
			Command: c,
			Dimensions: []*swarming_api.SwarmingRpcsStringPair{
				{Key: "pool", Value: "ChromeOSSkylab"},
				{Key: "dut_name", Value: lt.host},
			},
			// lt.duration.Seconds() is guaranteed to be much less than -1 + 2^63
			ExecutionTimeoutSecs: int64(lt.duration.Seconds()),
		},
	}}
	r := &swarming_api.SwarmingRpcsNewTaskRequest{
		Name: "lease task",
		Tags: []string{
			"pool:ChromeOSSkylab",
			"skylab-tool:lease",
			// This quota account specifier is only relevant for DUTs that are
			// in the prod skylab DUT_POOL_QUOTA pool; it is irrelevant and
			// harmless otherwise.
			"qs_account:leases",
			fmt.Sprintf("dut-name:%s", lt.host),
			fmt.Sprintf("lease-reason:%s", lt.reason),
		},
		TaskSlices:     slices,
		Priority:       10,
		ServiceAccount: e.ServiceAccount,
	}
	ctx, cf := context.WithTimeout(ctx, 60*time.Second)
	defer cf()
	resp, err := t.CreateTask(ctx, r)
	if err != nil {
		return "", errors.Annotate(err, "failed to create task").Err()
	}
	return resp.TaskId, nil
}
