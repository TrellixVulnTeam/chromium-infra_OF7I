// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package registration

import (
	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/cli"
)

type register struct {
	subcommands.CommandRunBase
}

// RegisterCmd contains register command specification
var RegisterCmd = &subcommands.Command{
	UsageLine: "register <sub-command>",
	ShortDesc: "Register a hardware asset(machine/rack/kvm/rpm/switch/drac/nic)",
	LongDesc:  "Register a machine/rack/kvm/rpm/switch/drac/nic",
	CommandRun: func() subcommands.CommandRun {
		c := &register{}
		return c
	},
}

type registerApp struct {
	cli.Application
}

// Run implementing subcommands.CommandRun interface
func (c *register) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	d := a.(*cli.Application)
	return subcommands.Run(&registerApp{*d}, args)
}

// GetCommands lists all the subcommands under register
func (c registerApp) GetCommands() []*subcommands.Command {
	return []*subcommands.Command{
		subcommands.CmdHelp,
	}
}

// GetName is cli.Application interface implementation
func (c registerApp) GetName() string {
	return "register"
}
