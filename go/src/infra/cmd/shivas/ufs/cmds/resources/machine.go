// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package resources

import (
	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/cli"
	m "infra/cmd/shivas/ufs/subcmds/registration/machine"
)

type machine struct {
	subcommands.CommandRunBase
}

// MachineCmd contains machine command specification
var MachineCmd = &subcommands.Command{
	UsageLine: "machine <sub-command>",
	ShortDesc: "operations for Machine",
	LongDesc:  "Example operations for Machine - Create/Update/List/Get/Delete",
	CommandRun: func() subcommands.CommandRun {
		c := &machine{}
		return c
	},
}

type machineApp struct {
	cli.Application
}

// Run implementing subcommands.CommandRun interface
func (c *machine) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	d := a.(*cli.Application)
	return subcommands.Run(&machineApp{*d}, args)
}

// GetCommands lists all the subcommands under machine
func (c machineApp) GetCommands() []*subcommands.Command {
	return []*subcommands.Command{
		m.GetMachineCmd,
		m.ListMachineCmd,
	}
}

// GetName is cli.Application interface implementation
func (c machineApp) GetName() string {
	return "machine"
}
