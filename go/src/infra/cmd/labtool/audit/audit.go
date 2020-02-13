// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package audit

import (
	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/cli"
)

type audit struct {
	subcommands.CommandRunBase
}

// AuditCmd contains audit command specification
var AuditCmd = &subcommands.Command{
	UsageLine: "audit <sub-command>",
	ShortDesc: "Tools for auditing the lab",
	LongDesc:  "Collection of tools for use in lab auditing",
	CommandRun: func() subcommands.CommandRun {
		c := &audit{}
		return c
	},
}

type auditApp struct {
	cli.Application
}

// Run implementing subcommands.CommandRun interface
func (c *audit) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	d := a.(*cli.Application)
	return subcommands.Run(&auditApp{*d}, args)
}

// GetCommands lists all the subcommands under audit
func (c auditApp) GetCommands() []*subcommands.Command {
	return []*subcommands.Command{
		subcommands.CmdHelp,
		ScannerCmd,
		LoglsCmd,
	}
}

// GetName is cli.Application interface implementation
func (c auditApp) GetName() string {
	return "audit"
}
