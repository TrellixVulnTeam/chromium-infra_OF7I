// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package tasks

import (
	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/errors"

	"infra/cmdsupport/cmdlib"
)

// Run executes the provisioning for requested devices.
func Run(authOpts auth.Options) *subcommands.Command {
	return &subcommands.Command{
		UsageLine: "run",
		ShortDesc: "Run provisioning on devices",
		LongDesc:  "Run provisioning on devices",
		CommandRun: func() subcommands.CommandRun {
			c := &runCmd{}
			c.authFlags.Register(&c.Flags, authOpts)
			return c
		},
	}
}

type runCmd struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
}

// Run executes the tool.
func (c *runCmd) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *runCmd) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	return errors.Reason("run is not implemented").Err()
}
