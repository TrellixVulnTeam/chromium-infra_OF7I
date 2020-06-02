// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package resources

import (
	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/cli"
	s "infra/cmd/labtool/ufs/subcmds/registration/switches"
)

type switches struct {
	subcommands.CommandRunBase
}

// SwitchCmd contains switch command specification
var SwitchCmd = &subcommands.Command{
	UsageLine: "switch <sub-command>",
	ShortDesc: "operations for Switch",
	LongDesc:  "Example operations for Switch - Create/Update/List/Get/Delete",
	CommandRun: func() subcommands.CommandRun {
		c := &switches{}
		return c
	},
}

type switchesApp struct {
	cli.Application
}

// Run implementing subcommands.CommandRun interface
func (c *switches) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	d := a.(*cli.Application)
	return subcommands.Run(&switchesApp{*d}, args)
}

// GetCommands lists all the subcommands under switch
func (c switchesApp) GetCommands() []*subcommands.Command {
	return []*subcommands.Command{
		s.AddSwitchCmd,
		s.UpdateSwitchCmd,
	}
}

// GetName is cli.Application interface implementation
func (c switchesApp) GetName() string {
	return "switch"
}
