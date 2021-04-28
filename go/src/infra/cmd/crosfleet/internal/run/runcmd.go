// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package run

import (
	"infra/cmd/crosfleet/internal/common"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/cli"
)

var testApplication = &cli.Application{
	Name:  "crosfleet run",
	Title: "Run tests.",
	Commands: []*subcommands.Command{
		backfill,
		test,
		suite,
		testplan,
		subcommands.CmdHelp,
	},
}

// CmdRun is the parent command for all `crosfleet run <subcommand>` commands.
var CmdRun = &subcommands.Command{
	UsageLine: "run <subcommand>",
	ShortDesc: "runs tests and other executable tasks on DUTs in ChromeOS hardware labs",
	LongDesc: `Runs individual tests, test suites, or custom test plan files, depending on the subcommand given.

Run 'crosfleet run' to see list of all subcommands.`,
	CommandRun: func() subcommands.CommandRun {
		c := &runCmd{}
		c.Init()
		return c
	},
}

type runCmd struct {
	common.Flags
}

func (c *runCmd) Run(a subcommands.Application, args []string, _ subcommands.Env) int {
	status := subcommands.Run(testApplication, args)
	if status == 0 {
		common.PrintCrosfleetUIPrompt(a)
	}
	return status
}
