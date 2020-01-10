// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package validateconfig

import (
	"errors"
	"fmt"

	"go.chromium.org/luci/auth/client/authcli"

	"github.com/maruel/subcommands"
	"infra/cmd/stable_version2/internal/cmd"
	"infra/cmd/stable_version2/internal/site"
	vc "infra/libs/cros/stableversion/validateconfig"
)

// Cmd is the top-level runnable for the validate-config subcommand of stable_version2
var Cmd = &subcommands.Command{
	UsageLine: `validate-config /path/to/stable_versions.cfg`,
	ShortDesc: "check that a stable_versions.cfg file is well-formed",
	LongDesc: `check that a stable_versions.cfg file is well-formed.

This command exists solely to validate a stable_versions.cfg file.
Its intended consumer is a submit hook that runs in the infra/config repo.
`,
	CommandRun: func() subcommands.CommandRun {
		c := &command{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)

		return c
	},
}

type command struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
}

func (c *command) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmd.PrintError(a.GetErr(), err)
		return 1
	}
	return 0
}

func (c *command) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	if len(args) == 0 {
		return errors.New("need at least one file")
	}
	if len(args) > 1 {
		return errors.New("validating multiple files not yet supported")
	}
	if err := vc.InspectFile(args[0]); err != nil {
		return err
	}
	fmt.Printf("%s\n", vc.FileSeemsLegit)
	return nil
}
