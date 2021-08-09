// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package subcmds

import (
	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/cli"

	"infra/cros/cmd/satlab/internal/components/dut"
)

// AddBase is the type for the add placeholder command.
type addBase struct {
	subcommands.CommandRunBase
}

// AddCmd is the add placeholder command.
var AddCmd = &subcommands.Command{
	UsageLine: "add <sub-command>",
	CommandRun: func() subcommands.CommandRun {
		c := &addBase{}
		return c
	},
}

// AddApp is the placeholder application for the add command.
type addApp struct {
	cli.Application
}

// Run transfers control to the add subcommands.
func (c *addBase) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	d := a.(*cli.Application)
	return subcommands.Run(&addApp{*d}, args)
}

// GetCommands lists the add subcommands.
func (c addApp) GetCommands() []*subcommands.Command {
	return []*subcommands.Command{
		subcommands.CmdHelp,
		dut.AddDUTCmd,
	}
}
