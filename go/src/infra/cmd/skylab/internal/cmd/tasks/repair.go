// Copyright 2018 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package tasks

import (
	"fmt"

	"github.com/google/uuid"
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

	attemptID := uuid.New().String()
	creator := &utils.TaskCreator{
		Client:      client,
		Environment: e,
	}
	tags := []string{
		fmt.Sprintf("repairAttemptID:%s", attemptID),
	}

	expirationSec := c.expirationMins * 60
	for _, host := range args {
		id, err := creator.RepairTask(ctx, host, tags, expirationSec)
		if err != nil {
			return err
		}
		fmt.Fprintf(a.GetOut(), "Created Swarming task %s for host %s\n", swarming.TaskURL(e.SwarmingService, id), host)
	}
	fmt.Fprintf(a.GetOut(), "Batch repair task URL: %s\n", swarming.TaskListURLForTags(e.SwarmingService, tags))
	return nil
}
