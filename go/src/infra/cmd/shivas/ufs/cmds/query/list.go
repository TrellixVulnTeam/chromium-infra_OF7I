// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package query

import (
	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/cli"

	"infra/cmd/shivas/ufs/subcmds/machineprototype"
)

type list struct {
	subcommands.CommandRunBase
}

// ListCmd contains list command specification
var ListCmd = &subcommands.Command{
	UsageLine: "ls <sub-command>",
	ShortDesc: "List entries for hardware asset/deployed asset/configuration",
	LongDesc: `List entries for hardware asset(machine/rack/kvm/rpm/switch/drac/nic)
or deployed machine/rack or configuration(machine-prototype/rack-prototype/chromeplatform)`,
	CommandRun: func() subcommands.CommandRun {
		c := &list{}
		return c
	},
}

type listApp struct {
	cli.Application
}

// Run implementing subcommands.CommandRun interface
func (c *list) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	d := a.(*cli.Application)
	return subcommands.Run(&listApp{*d}, args)
}

// GetCommands lists all the subcommands under list
func (c listApp) GetCommands() []*subcommands.Command {
	return []*subcommands.Command{
		subcommands.CmdHelp,
		machineprototype.ListMachineLSEPrototypeCmd,
	}
}

// GetName is cli.Application interface implementation
func (c listApp) GetName() string {
	return "list"
}
