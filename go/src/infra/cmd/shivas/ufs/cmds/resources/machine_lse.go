// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package resources

import (
	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/cli"
	m "infra/cmd/shivas/ufs/subcmds/inventory/machinelse"
)

type machinelse struct {
	subcommands.CommandRunBase
}

// MachinelseCmd contains machinelse command specification
var MachinelseCmd = &subcommands.Command{
	UsageLine: "machinelse <sub-command>",
	ShortDesc: "operations for MachineLSE(Machine Lab Setup Environment)",
	LongDesc:  "Example operations for MachineLSE - Create/Update/List/Get/Delete",
	CommandRun: func() subcommands.CommandRun {
		c := &machinelse{}
		return c
	},
}

type machinelseApp struct {
	cli.Application
}

// Run implementing subcommands.CommandRun interface
func (c *machinelse) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	d := a.(*cli.Application)
	return subcommands.Run(&machinelseApp{*d}, args)
}

// GetCommands lists all the subcommands under machinelse
func (c machinelseApp) GetCommands() []*subcommands.Command {
	return []*subcommands.Command{
		m.GetMachinelseCmd,
	}
}

// GetName is cli.Application interface implementation
func (c machinelseApp) GetName() string {
	return "machinelse"
}
