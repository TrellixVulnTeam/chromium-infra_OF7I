// Copyright 2018 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package tasks

import (
	"fmt"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/errors"

	skycmdlib "infra/cmd/skylab/internal/cmd/cmdlib"
	"infra/cmd/skylab/internal/site"
	"infra/cmdsupport/cmdlib"
)

// Audit subcommand: Audit hosts.
var Audit = &subcommands.Command{
	UsageLine: "audit [HOST...]",
	ShortDesc: "create audit tasks",
	LongDesc: `Create audit tasks.

This command does not wait for the tasks to start running.`,
	CommandRun: func() subcommands.CommandRun {
		c := &auditRun{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.Flags.IntVar(&c.expirationMins, "expiration-mins", 10, "The expiration minutes of the repair request.")
		c.Flags.BoolVar(&c.skipVerifyServoUSB, "skip-verify-servo-usb", false, "Do not run verifyer for servo usb drive.")
		c.Flags.BoolVar(&c.skipVerifyDUTStorage, "skip-verify-dut-storage", false, "Do not run verifyer for DUT storage.")
		return c
	},
}

type auditRun struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
	envFlags  skycmdlib.EnvFlags

	expirationMins       int
	skipVerifyServoUSB   bool
	skipVerifyDUTStorage bool
}

func (c *auditRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		fmt.Fprintf(a.GetErr(), "%s: %s\n", a.GetName(), err)
		return 1
	}
	return 0
}

func (c *auditRun) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	if c.expirationMins >= dayInMinutes {
		return cmdlib.NewUsageError(c.Flags, "Expiration minutes (%d minutes) cannot exceed 1 day [%d minutes]", c.expirationMins, dayInMinutes)
	}
	if len(args) == 0 {
		return errors.Reason("at least one host has to provided").Err()
	}

	return errors.Reason("not implemented yet").Err()
}
