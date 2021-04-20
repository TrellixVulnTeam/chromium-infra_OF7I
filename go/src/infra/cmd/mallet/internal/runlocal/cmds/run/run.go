// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package run

import (
	"github.com/maruel/subcommands"

	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"

	"infra/cmd/shivas/site"
)

type run struct {
	subcommands.CommandRunBase
}

// RunCmd runs a command locally
var RunCmd = &subcommands.Command{
	UsageLine: "run <sub-command>",
	ShortDesc: "Tools for locally running jobs",
	LongDesc:  "Tools for locally running jobs",
	CommandRun: func() subcommands.CommandRun {
		c := &run{}
		return c
	},
}

type runApp struct {
	cli.Application
	authFlags authcli.Flags
	envFlags  site.EnvFlags
}

func (c *run) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	d := a.(*cli.Application)
	ra := &runApp{
		Application: *d,
	}
	return subcommands.Run(ra, args)
}

func (c runApp) GetCommands() []*subcommands.Command {
	return []*subcommands.Command{
		subcommands.CmdHelp,
		LabpackCmd,
	}
}

func (c runApp) GetName() string {
	return "run"
}
