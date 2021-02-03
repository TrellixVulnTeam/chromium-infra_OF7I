// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// TODO(gregorynisbet)
// This command has been removed. Its only purpose now is to emit an error
// message suggesting an alternative to the removed functionality. After
// enough time has passed, please completely remove the file.

package internalcmds

import (
	"fmt"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"

	skycmdlib "infra/cmd/skylab/internal/cmd/cmdlib"
	"infra/cmd/skylab/internal/site"
	"infra/cmdsupport/cmdlib"
)

// DutStableVersion subcommand: Stable versions for DUT.
var DutStableVersion = &subcommands.Command{
	UsageLine: "stable-version HOSTNAME",
	ShortDesc: "Stable versions for DUT",
	LongDesc: `Stable versions for DUT.

For internal use only.`,
	CommandRun: func() subcommands.CommandRun {
		c := &dutStableVersionRun{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		return c
	},
}

type dutStableVersionRun struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
	envFlags  skycmdlib.EnvFlags
}

func (c *dutStableVersionRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		fmt.Fprintf(a.GetErr(), "%s: %s\n", a.GetName(), err)
		return 1
	}
	return 0
}

func (c *dutStableVersionRun) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	return cmdlib.NewUsageError(
		c.Flags,
		"skylab stable-version has been removed! Use shivas get -dut -host-info-store instead.",
	)
}
