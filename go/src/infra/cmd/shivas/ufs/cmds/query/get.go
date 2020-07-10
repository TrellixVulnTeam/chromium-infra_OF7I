// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package query

import (
	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/cli"

	"infra/cmd/shivas/ufs/subcmds/machineprototype"
)

type get struct {
	subcommands.CommandRunBase
}

// GetCmd contains get command specification
var GetCmd = &subcommands.Command{
	UsageLine: "get <sub-command>",
	ShortDesc: "Get details of asset/configuration by name or deployed machine/rack by hostname",
	LongDesc: `Get details of an asset(machine/rack/kvm/rpm/switch/drac/nic)
or configuration(machine-prototype/rack-prototype/chromeplatform) by name or a deployed machine/rack by hostname.`,
	CommandRun: func() subcommands.CommandRun {
		c := &get{}
		return c
	},
}

type getApp struct {
	cli.Application
}

// Run implementing subcommands.CommandRun interface
func (c *get) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	d := a.(*cli.Application)
	return subcommands.Run(&getApp{*d}, args)
}

// GetCommands lists all the subcommands under get
func (c getApp) GetCommands() []*subcommands.Command {
	return []*subcommands.Command{
		subcommands.CmdHelp,
		machineprototype.GetMachineLSEPrototypeCmd,
	}
}

// GetName is cli.Application interface implementation
func (c getApp) GetName() string {
	return "get"
}
