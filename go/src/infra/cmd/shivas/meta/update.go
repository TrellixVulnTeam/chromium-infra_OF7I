// Copyright 2018 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package meta

import (
	"fmt"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"

	"infra/cmd/shivas/site"
	"infra/libs/cipd"
)

// cipd reference tag for the dev version. Reference tag for prod version is prod.
var devVersionRefTag string = "latest"

// Update subcommand: Update shivas tool.
var Update = &subcommands.Command{
	UsageLine: "upgrade", // we have a separate update cmd under UFS, so changing this to upgrade.
	ShortDesc: "upgrade shivas",
	LongDesc: `Upgrade shivas.

This is a thin wrapper around CIPD.`,
	CommandRun: func() subcommands.CommandRun {
		c := &updateRun{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)

		c.Flags.BoolVar(&c.dev, "dev", false, "Upgrade to dev version")
		return c
	},
}

type updateRun struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags

	dev bool
}

func (c *updateRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		fmt.Fprintf(a.GetErr(), "%s: %s\n", a.GetName(), err)
		return 1
	}
	return 0
}

func (c *updateRun) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	if c.dev {
		if err := cipd.UpdatePackage(site.CipdInstalledPath, devVersionRefTag, a.GetOut(), a.GetErr()); err != nil {
			return err
		}
	} else {
		if err := cipd.UpdatePackageToProd(site.CipdInstalledPath, a.GetOut(), a.GetErr()); err != nil {
			return err
		}
	}
	fmt.Fprintf(a.GetErr(), "%s: You may need to run shivas login again after the upgrade\n", a.GetName())
	fmt.Fprintf(a.GetErr(), "%s: Run shivas whoami to check login status\n", a.GetName())
	return nil
}
