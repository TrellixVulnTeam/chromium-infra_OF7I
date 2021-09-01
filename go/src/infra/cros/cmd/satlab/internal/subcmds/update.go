// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package subcmds

import (
	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/cli"

	"infra/cros/cmd/satlab/internal/components/dut"
)

// UpdateBase is the type for the add placeholder command.
type updateBase struct {
	subcommands.CommandRunBase
}

// UpdateCmd is the add placeholder command.
var UpdateCmd = &subcommands.Command{
	UsageLine: "update <sub-command>",
	CommandRun: func() subcommands.CommandRun {
		c := &updateBase{}
		return c
	},
}

// UpdateApp is the placeholder application for the update command.
type updateApp struct {
	cli.Application
}

// Run transfers control to the add subcommands.
func (c *updateBase) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	d := a.(*cli.Application)
	return subcommands.Run(&updateApp{*d}, args)
}

// GetCommands lists the add subcommands.
func (c updateApp) GetCommands() []*subcommands.Command {
	return []*subcommands.Command{
		subcommands.CmdHelp,
		dut.UpdateDUTCmd,
	}
}
