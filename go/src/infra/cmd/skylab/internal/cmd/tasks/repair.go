// Copyright 2018 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package tasks

import (
	"fmt"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"

	skycmdlib "infra/cmd/skylab/internal/cmd/cmdlib"
	"infra/cmd/skylab/internal/cmd/utils"
	"infra/cmd/skylab/internal/site"
	"infra/cmdsupport/cmdlib"
	"infra/libs/skylab/swarming"
)

// Repair subcommand: Repair hosts.
var Repair = &subcommands.Command{
	UsageLine: "repair [HOST...]",
	ShortDesc: "create repair tasks",
	LongDesc: `Create repair tasks.

This command does not wait for the task to start running.`,
	CommandRun: func() subcommands.CommandRun {
		c := &repairRun{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.Flags.IntVar(&c.expirationMins, "expiration-mins", 10, "The expiration minutes of the repair request.")
		return c
	},
}

type repairRun struct {
	subcommands.CommandRunBase
	authFlags      authcli.Flags
	envFlags       skycmdlib.EnvFlags
	expirationMins int
}

func (c *repairRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		fmt.Fprintf(a.GetErr(), "%s: %s\n", a.GetName(), err)
		return 1
	}
	return 0
}

func (c *repairRun) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	if c.expirationMins >= dayInMinutes {
		return cmdlib.NewUsageError(c.Flags, "Expiration minutes (%d minutes) cannot exceed 1 day [%d minutes]", c.expirationMins, dayInMinutes)
	}
	if len(args) == 0 {
		return errors.Reason("at least one host has to provided").Err()
	}

	ctx := cli.GetContext(a, c, env)
	creator, err := utils.NewTaskCreator(ctx, &c.authFlags, c.envFlags)
	if err != nil {
		return err
	}

	for _, host := range args {
		dutName := skycmdlib.FixSuspiciousHostname(host)
		if dutName != host {
			fmt.Fprintf(a.GetErr(), "correcting (%s) to (%s)\n", host, dutName)
		}
		id, err := creator.RepairTask(ctx, dutName, c.expirationMins*60)
		if err != nil {
			return err
		}
		fmt.Fprintf(a.GetOut(), "Created Swarming task %s for host %s\n", swarming.TaskURL(creator.Environment.SwarmingService, id), dutName)
	}
	if len(args) > 1 {
		fmt.Fprintf(a.GetOut(), "\nBatch tasks URL: %s\n\n", creator.SessionTasksURL())
	}
	return nil
}
