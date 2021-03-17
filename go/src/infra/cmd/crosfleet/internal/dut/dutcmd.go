// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dut

import (
	"infra/cmd/crosfleet/internal/common"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/cli"
)

var dutApplication = &cli.Application{
	Name:  "crosfleet dut",
	Title: "Interact with DUTs.",
	Commands: []*subcommands.Command{
		info,
		lease,
		abandon,
		subcommands.CmdHelp,
	},
}

// CmdDut is the parent command for all `crosfleet dut <subcommand>` commands.
var CmdDut = &subcommands.Command{
	UsageLine: "dut <subcommand>",
	ShortDesc: "interacts directly with DUTs in ChromeOS hardware labs",
	LongDesc: `Issues subcommands directly to DUTs.

Run 'crosfleet dut' to see list of all subcommands.`,
	CommandRun: func() subcommands.CommandRun {
		c := &dutCmdRun{}
		c.Init()
		return c
	},
}

type dutCmdRun struct {
	common.Flags
}

func (c *dutCmdRun) Run(a subcommands.Application, args []string, _ subcommands.Env) int {
	status := subcommands.Run(dutApplication, args)
	if status == 0 {
		common.PrintCrosfleetUIPrompt(a)
	}
	return status
}
