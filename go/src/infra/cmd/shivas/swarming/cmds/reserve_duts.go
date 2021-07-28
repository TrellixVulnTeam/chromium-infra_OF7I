// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package tasks

import (
	"fmt"
	"os/user"

	"github.com/maruel/subcommands"

	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"

	"infra/cmd/shivas/site"
	"infra/libs/swarming"
)

type reserveDuts struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
	envFlags  site.EnvFlags

	manualRepair bool
}

// ReserveDutsCmd contains reserve-dut command specification
var ReserveDutsCmd = &subcommands.Command{
	UsageLine: "reserve-duts",
	ShortDesc: "Reserve the DUT by name",
	LongDesc: `Reserve the DUT by name.
	./shivas reserve <dut_name>
	Schedule a swarming Reserve task to the DUT to set the state to RESERVED to prevent scheduling tasks and tests to the DUT.
	Reserved DUT does not have expiration time and can be changed by scheduling any admin task on it.`,
	CommandRun: func() subcommands.CommandRun {
		c := &reserveDuts{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.Flags.BoolVar(&c.manualRepair, "manual-repair", false, "Reserve the dut for manual repair.")
		return c
	},
}

// Run represent runner for reserve command
func (c *reserveDuts) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		fmt.Fprintf(a.GetErr(), "%s: %s\n", a.GetName(), err)
		return 1
	}
	return 0
}

func (c *reserveDuts) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	if len(args) == 0 {
		return errors.Reason("at least one hostname has to be provided").Err()
	}
	user, err := user.Current()
	if err != nil {
		return err
	}
	ctx := cli.GetContext(a, c, env)
	e := c.envFlags.Env()
	creator, err := swarming.NewTaskCreator(ctx, &c.authFlags, e.SwarmingService)
	if err != nil {
		return err
	}

	successMap := make(map[string]*swarming.TaskInfo)
	errorMap := make(map[string]error)
	for _, host := range args {
		// TODO(crbug/1128496): update state directly in the UFS without creating the swarming task
		var task *swarming.TaskInfo
		if c.manualRepair {
			task, err = creator.SetManualRepair(ctx, e.SwarmingServiceAccount, host, user.Username)
		} else {
			task, err = creator.ReserveDUT(ctx, e.SwarmingServiceAccount, host, user.Username)
		}
		if err != nil {
			errorMap[host] = err
		} else {
			successMap[host] = task
		}
	}
	creator.PrintResults(a.GetOut(), successMap, errorMap)
	return nil
}
