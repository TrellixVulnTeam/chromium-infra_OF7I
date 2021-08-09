// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package subcmds

import (
	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/cli"

	"infra/cros/cmd/satlab/internal/components/dut"
)

// DeleteBase is the placeholder for the delete command.
type deleteBase struct {
	subcommands.CommandRunBase
}

// DeleteCmd contains the usage and implementation for the delete command.
var DeleteCmd = &subcommands.Command{
	UsageLine: "delete <sub-command>",
	CommandRun: func() subcommands.CommandRun {
		c := &deleteBase{}
		return c
	},
}

// DeleteApp is an application for the delete commands. Control is transferred here
// when consuming the "delete" subcommand.
type deleteApp struct {
	cli.Application
}

// Run transfers control to the subcommands of delete.
func (c *deleteBase) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	d := a.(*cli.Application)
	return subcommands.Run(&deleteApp{*d}, args)
}

// GetCommands lists the subcommands of delete.
func (c deleteApp) GetCommands() []*subcommands.Command {
	return []*subcommands.Command{
		subcommands.CmdHelp,
		dut.DeleteDUTCmd,
	}
}
