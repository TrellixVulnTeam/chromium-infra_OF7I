// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package meta

import (
	"fmt"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/errors"

	"infra/cros/cmd/satlab/internal/site"
	"infra/libs/cipd"
)

// cipd reference tag for the dev version. Reference tag for prod version is prod.
var devVersionRefTag string = "latest"

// Update subcommand: Update satlab tool.
var Update = &subcommands.Command{
	UsageLine: "upgrade", // we have a separate update cmd under UFS, so changing this to upgrade.
	ShortDesc: "upgrade satlab",
	LongDesc: `Upgrade satlab.

This is a thin wrapper around CIPD.`,
	CommandRun: func() subcommands.CommandRun {
		c := &updateRun{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)

		c.Flags.BoolVar(&c.dev, "dev", false, "Upgrade to dev version")
		return c
	},
}

// UpdateRun is the update command for satlab.
type updateRun struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags

	dev bool
}

// Run updates satlab and returns an exit status.
func (c *updateRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		fmt.Fprintf(a.GetErr(), "%s: %s\n", a.GetName(), err)
		return 1
	}
	return 0
}

// InnerRun is the implementation of the run command.
func (c *updateRun) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	if c.dev {
		if err := cipd.UpdatePackage(site.CipdInstalledPath, devVersionRefTag, a.GetOut(), a.GetErr()); err != nil {
			return errors.Annotate(err, "update").Err()
		}
	} else {
		if err := cipd.UpdatePackageToProd(site.CipdInstalledPath, a.GetOut(), a.GetErr()); err != nil {
			return errors.Annotate(err, "update").Err()
		}
	}
	fmt.Fprintf(a.GetErr(), "%s: You may need to run satlab login again after the upgrade\n", a.GetName())
	fmt.Fprintf(a.GetErr(), "%s: Run satlab whoami to check login status\n", a.GetName())
	return nil
}
