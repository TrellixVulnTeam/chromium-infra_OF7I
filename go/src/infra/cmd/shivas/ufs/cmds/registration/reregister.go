// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package registration

import (
	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/cli"
)

type reregister struct {
	subcommands.CommandRunBase
}

// ReregisterCmd contains reregister command specification
var ReregisterCmd = &subcommands.Command{
	UsageLine: "reregister <sub-command>",
	ShortDesc: "Reregister/Update a hardware asset",
	LongDesc:  "Reregister/Update a machine/rack/kvm/rpm/switch/drac/nic",
	CommandRun: func() subcommands.CommandRun {
		c := &reregister{}
		return c
	},
}

type reregisterApp struct {
	cli.Application
}

// Run implementing subcommands.CommandRun interface
func (c *reregister) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	d := a.(*cli.Application)
	return subcommands.Run(&reregisterApp{*d}, args)
}

// GetCommands lists all the subcommands under reregister
func (c reregisterApp) GetCommands() []*subcommands.Command {
	return []*subcommands.Command{
		subcommands.CmdHelp,
	}
}

// GetName is cli.Application interface implementation
func (c reregisterApp) GetName() string {
	return "reregister"
}
