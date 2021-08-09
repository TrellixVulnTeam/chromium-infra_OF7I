// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package subcmds

import (
	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/cli"

	"infra/cros/cmd/satlab/internal/components/dut"
)

// GetBase is a placeholder command for "get".
type getBase struct {
	subcommands.CommandRunBase
}

// GetCmd is a placeholder command for get.
var GetCmd = &subcommands.Command{
	UsageLine: "get <sub-command>",
	CommandRun: func() subcommands.CommandRun {
		c := &getBase{}
		return c
	},
}

// GetApp is an application tha tholds the get subcommands.
type getApp struct {
	cli.Application
}

// Run transfers control to a subcommand.
func (c *getBase) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	d := a.(*cli.Application)
	return subcommands.Run(&getApp{*d}, args)
}

// GetCommands lists the available subcommands.
func (c getApp) GetCommands() []*subcommands.Command {
	return []*subcommands.Command{
		subcommands.CmdHelp,
		dut.GetDUTCmd,
	}
}
