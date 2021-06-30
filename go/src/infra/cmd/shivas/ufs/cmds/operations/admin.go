// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package operations

import (
	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/cli"

	"infra/cmd/shivas/ufs/subcmds/cron"
)

type admin struct {
	subcommands.CommandRunBase
}

// AddCmd contains add command specification
var AdminCmd = &subcommands.Command{
	UsageLine: "admin <sub-command>",
	ShortDesc: "Admin operations. You may lack the permission to use these",
	LongDesc:  `Run admin operations on UFS. Available operations are [cron]`,
	CommandRun: func() subcommands.CommandRun {
		c := &admin{}
		return c
	},
}

type adminApp struct {
	cli.Application
}

// Run implementing subcommands.CommandRun interface
func (c *admin) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	d := a.(*cli.Application)
	return subcommands.Run(&adminApp{*d}, args)
}

// GetCommands lists all the subcommands under add
func (c *adminApp) GetCommands() []*subcommands.Command {
	return []*subcommands.Command{
		subcommands.CmdHelp,
		cron.CronCmd,
	}
}

// GetName is cli.Application interface implementation
func (c *adminApp) GetName() string {
	return "admin"
}
