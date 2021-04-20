// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package tasks

import (
	"fmt"
	"infra/cmd/mallet/internal/bb"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"

	"infra/cmd/mallet/internal/site"
	"infra/cmdsupport/cmdlib"
)

// Recovery subcommand: Recovering the devices.
var Recovery = &subcommands.Command{
	UsageLine: "recovery",
	ShortDesc: "Recovery the DUT",
	LongDesc:  "Recovery the DUT.",
	CommandRun: func() subcommands.CommandRun {
		c := &recoveryRun{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		return c
	},
}

type recoveryRun struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
}

func (c *recoveryRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *recoveryRun) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	ctx := cli.GetContext(a, c, env)

	bc, err := bb.NewClient(ctx, c.authFlags)
	if err != nil {
		return err
	}
	var unit string
	if len(args) == 0 {
		return errors.New("Repair target is not specified")
	} else {
		unit = args[0]
	}

	taskID, err := bc.ScheduleRepair(ctx, unit)
	if err != nil {
		return err
	}
	fmt.Fprintf(a.GetOut(), "Created repair task for %s: %s\n", unit, bc.BuildURL(taskID))
	return nil
}
