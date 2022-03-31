// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package operations

import (
	"infra/cmd/shivas/internal/ufs/subcmds/peripherals"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/cli"
)

// ReplaceCmd contains rename command specification
var ReplaceCmd = &subcommands.Command{
	UsageLine: "replace <sub-command>",
	ShortDesc: "Replace a resource/entity",
	LongDesc: `Replace a single or set of
	peripheral-wifi
	bluetooth-peers`,
	CommandRun: func() subcommands.CommandRun {
		c := &replace{}
		return c
	},
}

type replace struct {
	subcommands.CommandRunBase
}

// Run implementing subcommands.CommandRun interface
func (c *replace) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	d := a.(*cli.Application)
	return subcommands.Run(&replaceApp{*d}, args)
}

type replaceApp struct {
	cli.Application
}

// GetCommands lists all the subcommands under rename
func (c replaceApp) GetCommands() []*subcommands.Command {
	return []*subcommands.Command{
		peripherals.ReplaceBluetoothPeersCmd,
		peripherals.ReplacePeripheralWifiCmd,
	}
}

// GetName is cli.Application interface implementation
func (c replaceApp) GetName() string {
	return "replace"
}
