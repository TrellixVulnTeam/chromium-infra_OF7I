// Copyright 2020 The Chromium Authors. All rights reserved.
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

// Verify subcommand: Verify hosts.
var Verify = &subcommands.Command{
	UsageLine: "verify [HOST...]",
	ShortDesc: "create verify tasks",
	LongDesc: `Create verify tasks.

This command does not wait for the tasks to start running.`,
	CommandRun: func() subcommands.CommandRun {
		c := &verifyRun{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.Flags.IntVar(&c.expirationMins, "expiration-mins", 10, "The expiration minutes of the request.")
		return c
	},
}

type verifyRun struct {
	subcommands.CommandRunBase
	authFlags      authcli.Flags
	envFlags       skycmdlib.EnvFlags
	expirationMins int
}

func (c *verifyRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		fmt.Fprintf(a.GetErr(), "%s: %s\n", a.GetName(), err)
		return 1
	}
	return 0
}

func (c *verifyRun) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	if c.expirationMins >= dayInMinutes {
		return cmdlib.NewUsageError(c.Flags, "Expiration minutes (%d minutes) cannot exceed 1 day [%d minutes]", c.expirationMins, dayInMinutes)
	}

	return errors.Reason("not implemeneted yet").Err()
}
