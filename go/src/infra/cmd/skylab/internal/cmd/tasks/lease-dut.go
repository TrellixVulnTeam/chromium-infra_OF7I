// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package tasks

import (
	"context"
	"fmt"
	"infra/cmd/skylab/internal/cmd/utils"
	"strings"
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

// maxTasksPerModel is the maximum number of tasks that are allowed to be executing
// at the same time for a given model.
const maxTasksPerModel = 2

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
		c.Flags.StringVar(&c.model, "model", "", "Leases may optionally target a model instead of a hostname.")
		// We allow arbitrary dimensions to be passed in via the -dims flag.
		// e.g. -dims a=4,b=7
		c.Flags.Var(dimsVar{data: c}, "dims", "List of additional dimensions in format key1=value1,key2=value2,... .")
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
	dims         map[string]string
}

// dimsVar is a handle to leaseDutRun that implements the Value interface
// and allows the dims map to be modified.
type dimsVar struct {
	data *leaseDutRun
}

// String returns the default value for dimensions represented as a string.
// The default value is an empty map, which stringifies to an empty string.
func (d dimsVar) String() string {
	return ""
}

// Set populates the dims map with comma-delimited key-value pairs.
// Setting the dims map always succeeds, regardless of what string is given.
func (d dimsVar) Set(newval string) error {
	if len(d.data.dims) == 0 {
		d.data.dims = make(map[string]string)
	}
	// strings.Split, if given an empty string, will produce a
	// slice containing a single string.
	if len(newval) > 0 {
		for _, entry := range strings.Split(newval, ",") {
			key, val, err := splitKeyVal(entry)
			if err != nil {
				return err
			}
			d.data.dims[key] = val
		}
	}
	return nil
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

	ctx := cli.GetContext(a, c, env)

	leaseDuration := time.Duration(c.leaseMinutes) * time.Minute

	if hasOneHostname {
		oldhost := args[0]
		host := skycmdlib.FixSuspiciousHostname(oldhost)
		if host != oldhost {
			fmt.Fprintf(a.GetErr(), "correcting (%s) to (%s)\n", oldhost, host)
		}
		return c.leaseDutByHostname(ctx, a, leaseDuration, host)
	}
	return c.leaseDUTByModel(ctx, a, leaseDuration, c.model)
}

// leaseDutByHostname leases a DUT by hostname and schedules a follow-up repair task
func (c *leaseDutRun) leaseDutByHostname(ctx context.Context, a subcommands.Application, leaseDuration time.Duration, host string) error {
	client, err := c.newSwarmingClient(ctx)
	if err != nil {
		return errors.Annotate(err, "failed to create Swarming client").Err()
	}

	e := c.envFlags.Env()
	creator := utils.TaskCreator{
		Client:      client,
		Environment: e,
	}
	id, err := creator.LeaseByHostnameTask(ctx, host, int(leaseDuration.Seconds()), c.leaseReason)
	if err != nil {
		return err
	}
	fmt.Fprintf(a.GetOut(), "Created lease task for host %s: %s\n", host, swarming.TaskURL(e.SwarmingService, id))
	scheduleRepairTaskForLater(ctx, &creator, a, leaseDuration, host)
	fmt.Fprintf(a.GetOut(), "Waiting for task to start; lease isn't active yet\n")

	if err := c.waitForTaskStart(ctx, client, id); err != nil {
		return err
	}
	// TODO(ayatane): The time printed here may be off by the poll interval above.
	fmt.Fprintf(a.GetOut(), "DUT leased until %s\n", time.Now().Add(leaseDuration).Format(time.RFC1123))
	return nil
}

func (c *leaseDutRun) leaseDUTByModel(ctx context.Context, a subcommands.Application, leaseDuration time.Duration, model string) error {
	client, err := c.newSwarmingClient(ctx)
	if err != nil {
		return errors.Annotate(err, "failed to create Swarming client").Err()
	}

	tasks, err := client.GetActiveLeaseTasksForModel(ctx, model)
	if err != nil {
		return errors.Annotate(err, "computing existing leases").Err()
	}
	if len(tasks) > maxTasksPerModel {
		return fmt.Errorf("number of active tasks %d for model (%s) exceeds cap %d", len(tasks), model, maxTasksPerModel)
	}

	e := c.envFlags.Env()
	creator := utils.TaskCreator{
		Client:      client,
		Environment: e,
	}
	id, err := creator.LeaseByModelTask(ctx, model, c.dims, int(leaseDuration.Seconds()), c.leaseReason)
	if err != nil {
		return err
	}
	fmt.Fprintf(a.GetOut(), "Created lease task for model %s: %s\n", model, swarming.TaskURL(e.SwarmingService, id))

	if err := c.waitForTaskStart(ctx, client, id); err != nil {
		return err
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

// newSwarmingClient creates a new swarming client.
func (c *leaseDutRun) newSwarmingClient(ctx context.Context) (*swarming.Client, error) {
	h, err := cmdlib.NewHTTPClient(ctx, &c.authFlags)
	if err != nil {
		return nil, err
	}
	e := c.envFlags.Env()
	client, err := swarming.New(ctx, h, e.SwarmingService)
	if err != nil {
		return nil, err
	}
	return client, nil
}

// waitForTaskStart waits for the task with the given id to start.
func (c *leaseDutRun) waitForTaskStart(ctx context.Context, client *swarming.Client, id string) error {
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
			time.Sleep(10 * time.Second)
		case "RUNNING":
			return nil
		default:
			return errors.Reason("Got unexpected task state %#v", s).Err()
		}
	}
}

// splitKeyVal splits a string with "=" into two key-value pairs,
// and returns an error if this is impossible.
// Strings with multiple "=" values are considered malformed.
func splitKeyVal(s string) (string, string, error) {
	res := strings.Split(s, "=")
	switch len(res) {
	case 0, 1:
		return "", "", fmt.Errorf(`string (%s) does not contain a key and value`, s)
	case 2:
		return res[0], res[1], nil
	}
	return "", "", fmt.Errorf(`string (%s) contains more than too many %d "=" chars`, s, len(res))
}
