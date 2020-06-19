// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package inventory

import (
	"fmt"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"

	skycmdlib "infra/cmd/skylab/internal/cmd/cmdlib"
	inv "infra/cmd/skylab/internal/inventory"
	"infra/cmd/skylab/internal/site"
	"infra/cmdsupport/cmdlib"
)

// UpdateLabstation subcommand: update a labstation to inventory.
var UpdateLabstation = &subcommands.Command{
	UsageLine: "update-labstation [FLAGS...]",
	ShortDesc: "update a labstation",
	LongDesc:  `Update a labstation to the inventory.`,
	CommandRun: func() subcommands.CommandRun {
		c := &updateLabstationRun{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.Flags.StringVar(&c.servoToDelete, "delete-servo-serial", "", "the serial number of the servo to be deleted")
		return c
	},
}

type updateLabstationRun struct {
	subcommands.CommandRunBase
	authFlags     authcli.Flags
	envFlags      skycmdlib.EnvFlags
	servoToDelete string
}

// Run implements the subcommands.CommandRun interface.
func (c *updateLabstationRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *updateLabstationRun) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	ctx := cli.GetContext(a, c, env)
	hc, err := cmdlib.NewHTTPClient(ctx, &c.authFlags)
	if err != nil {
		return err
	}

	e := c.envFlags.Env()
	ic := inv.NewInventoryClient(hc, e)
	hostname := args[0]
	res, err := ic.UpdateLabstations(ctx, hostname, c.servoToDelete)
	if err != nil {
		return err
	}
	fmt.Printf("Successfully delete servo %s for labstation %s\n", c.servoToDelete, hostname)
	fmt.Printf("The left servos for this labstation are:\n")
	for _, servo := range res.GetLabstation().GetLabstation().GetServos() {
		if servo.GetServoSerial() == c.servoToDelete {
			fmt.Printf("Warning: servo %s is not deleted, stop printing...\n", c.servoToDelete)
			break
		} else {
			fmt.Printf("%v\n", servo)
		}
	}
	return nil
}
