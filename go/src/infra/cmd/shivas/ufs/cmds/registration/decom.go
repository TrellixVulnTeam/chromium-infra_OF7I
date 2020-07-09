// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package registration

import (
	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/cli"

	"infra/cmd/shivas/ufs/subcmds/machine"
	"infra/cmd/shivas/ufs/subcmds/switches"
)

type decom struct {
	subcommands.CommandRunBase
}

// DecomCmd contains decom command specification
var DecomCmd = &subcommands.Command{
	UsageLine: "decom <sub-command>",
	ShortDesc: "Decommission/Delete a hardware asset",
	LongDesc:  "Decommission/Delete a machine/rack/kvm/rpm/switch/drac/nic",
	CommandRun: func() subcommands.CommandRun {
		c := &decom{}
		return c
	},
}

type decomApp struct {
	cli.Application
}

// Run implementing subcommands.CommandRun interface
func (c *decom) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	d := a.(*cli.Application)
	return subcommands.Run(&decomApp{*d}, args)
}

// GetCommands lists all the subcommands under decom
func (c decomApp) GetCommands() []*subcommands.Command {
	return []*subcommands.Command{
		subcommands.CmdHelp,
		machine.DecomMachineCmd,
		switches.DecomSwitchCmd,
	}
}

// GetName is cli.Application interface implementation
func (c decomApp) GetName() string {
	return "decom"
}
