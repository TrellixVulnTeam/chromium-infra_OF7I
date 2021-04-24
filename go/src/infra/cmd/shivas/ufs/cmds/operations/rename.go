// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package operations

import (
	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/cli"

	"infra/cmd/shivas/ufs/subcmds/asset"
	"infra/cmd/shivas/ufs/subcmds/dut"
	"infra/cmd/shivas/ufs/subcmds/labstation"
	"infra/cmd/shivas/ufs/subcmds/machine"
	"infra/cmd/shivas/ufs/subcmds/nic"
	"infra/cmd/shivas/ufs/subcmds/switches"
)

type rename struct {
	subcommands.CommandRunBase
}

// RenameCmd contains rename command specification
var RenameCmd = &subcommands.Command{
	UsageLine: "rename <sub-command>",
	ShortDesc: "Rename a resource/entity",
	LongDesc:  `Rename a machine/nic/switch`,
	CommandRun: func() subcommands.CommandRun {
		c := &rename{}
		return c
	},
}

type renameApp struct {
	cli.Application
}

// Run implementing subcommands.CommandRun interface
func (c *rename) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	d := a.(*cli.Application)
	return subcommands.Run(&renameApp{*d}, args)
}

// GetCommands lists all the subcommands under rename
func (c renameApp) GetCommands() []*subcommands.Command {
	return []*subcommands.Command{
		subcommands.CmdHelp,
		dut.RenameDUTCmd,
		asset.RenameAssetCmd,
		labstation.RenameLabstationCmd,
		machine.RenameMachineCmd,
		nic.RenameNicCmd,
		switches.RenameSwitchCmd,
	}
}

// GetName is cli.Application interface implementation
func (c renameApp) GetName() string {
	return "rename"
}
