// Copyright 2018 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package meta

import (
	"fmt"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"

	"infra/cmd/shivas/site"
	"infra/libs/cros/cipd"
)

// Update subcommand: Update shivas tool.
var Update = &subcommands.Command{
	UsageLine: "update",
	ShortDesc: "update shivas",
	LongDesc: `Update shivas.

This is a thin wrapper around CIPD.`,
	CommandRun: func() subcommands.CommandRun {
		c := &updateRun{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		return c
	},
}

type updateRun struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
}

func (c *updateRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		fmt.Fprintf(a.GetErr(), "%s: %s\n", a.GetName(), err)
		return 1
	}
	return 0
}

func (c *updateRun) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	if err := cipd.UpdatePackage(site.CipdInstalledPath, a.GetOut(), a.GetErr()); err != nil {
		return err
	}
	fmt.Fprintf(a.GetErr(), "%s: You may need to run shivas login again after the update\n", a.GetName())
	fmt.Fprintf(a.GetErr(), "%s: Run shivas whoami to check login status\n", a.GetName())
	return nil
}
