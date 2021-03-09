// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package tasks

import (
	"fmt"

	"github.com/maruel/subcommands"

	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"

	"infra/cmd/shivas/site"
	"infra/libs/skylab/worker"
	"infra/libs/swarming"
)

type repairDuts struct {
	subcommands.CommandRunBase
	authFlags      authcli.Flags
	envFlags       site.EnvFlags
	expirationMins int
	onlyVerify     bool
}

// RepairDutsCmd contains repair-duts command specification
var RepairDutsCmd = &subcommands.Command{
	UsageLine: "repair-duts",
	ShortDesc: "Repair the DUT by name",
	LongDesc: `Repair the DUT by name.
	./shivas repair <dut_name1> ...
	Schedule a swarming Repair task to the DUT to try to recover/verify it.`,
	CommandRun: func() subcommands.CommandRun {
		c := &repairDuts{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.Flags.BoolVar(&c.onlyVerify, "verify", false, "Run only verify actions.")
		c.Flags.IntVar(&c.expirationMins, "expiration-mins", 10, "The expiration minutes of the repair request.")
		return c
	},
}

// Run represent runner for reserve command
func (c *repairDuts) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		fmt.Fprintf(a.GetErr(), "%s: %s\n", a.GetName(), err)
		return 1
	}
	return 0
}

func (c *repairDuts) innerRun(a subcommands.Application, args []string, env subcommands.Env) (err error) {
	if len(args) == 0 {
		return errors.Reason("at least one hostname has to be provided").Err()
	}
	ctx := cli.GetContext(a, c, env)
	e := c.envFlags.Env()
	creator, err := swarming.NewTaskCreator(ctx, &c.authFlags, e.SwarmingService)
	if err != nil {
		return err
	}
	creator.LogdogService = e.LogdogService
	successMap := make(map[string]*swarming.TaskInfo)
	errorMap := make(map[string]error)
	for _, host := range args {
		creator.GenerateLogdogTaskCode()
		cmd := &worker.Command{TaskName: c.taskName()}
		cmd.LogDogAnnotationURL = creator.LogdogURL()
		var task *swarming.TaskInfo
		if c.onlyVerify {
			task, err = creator.VerifyTask(ctx, e.SwarmingServiceAccount, host, c.expirationMins*60, cmd.Args(), cmd.LogDogAnnotationURL)
		} else {
			task, err = creator.RepairTask(ctx, e.SwarmingServiceAccount, host, c.expirationMins*60, cmd.Args(), cmd.LogDogAnnotationURL)
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

func (c *repairDuts) taskName() string {
	if c.onlyVerify {
		return "admin_verify"
	}
	return "admin_repair"
}
