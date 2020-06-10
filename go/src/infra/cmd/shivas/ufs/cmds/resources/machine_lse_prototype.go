// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package resources

import (
	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/cli"
	m "infra/cmd/shivas/ufs/subcmds/configuration/machinelseprototype"
)

type machinelsePrototype struct {
	subcommands.CommandRunBase
}

// MachinelsePrototypeCmd contains machinelsePrototype command specification
var MachinelsePrototypeCmd = &subcommands.Command{
	UsageLine: "machinelseprototype <sub-command>",
	ShortDesc: "operations for MachineLSEPrototype(Machine Lab Setup Environment Prototype)",
	LongDesc:  "Example operations for MachineLSEPrototype - Create/Update/List/Get/Delete",
	CommandRun: func() subcommands.CommandRun {
		c := &machinelsePrototype{}
		return c
	},
}

type machinelsePrototypeApp struct {
	cli.Application
}

// Run implementing subcommands.CommandRun interface
func (c *machinelsePrototype) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	d := a.(*cli.Application)
	return subcommands.Run(&machinelsePrototypeApp{*d}, args)
}

// GetCommands lists all the subcommands under machinelsePrototype
func (c machinelsePrototypeApp) GetCommands() []*subcommands.Command {
	return []*subcommands.Command{
		m.AddMachinelsePrototypeCmd,
		m.UpdateMachinelsePrototypeCmd,
		m.GetMachinelsePrototypeCmd,
		m.DeleteMachinelsePrototypeCmd,
	}
}

// GetName is cli.Application interface implementation
func (c machinelsePrototypeApp) GetName() string {
	return "machinelseprototype"
}
