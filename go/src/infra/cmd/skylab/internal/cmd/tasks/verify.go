// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package tasks

import (
	"fmt"
	"strings"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"

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
	hosts := args
	if len(args) == 0 {
		hosts = []string{"host1", "host2", "..."}
	}
	return cmdlib.NewUsageError(
		c.Flags,
		"skylab verify has been removed! Please use:\n\nshivas repair-duts -verify "+strings.Join(hosts, " ")+"\n",
	)
}
