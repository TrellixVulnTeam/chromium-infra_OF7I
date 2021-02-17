// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dut

import (
	"fmt"
	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"infra/cmd/crosfleet/internal/site"
)

var release = &subcommands.Command{
	UsageLine: "release HOST [HOST...]",
	ShortDesc: "release DUTs which were previously leased via 'dut lease'",
	LongDesc: `Release DUTs which were previously leased via 'dut lease'.

This command's behavior is subject to change without notice.
Do not build automation around this subcommand.`,
	CommandRun: func() subcommands.CommandRun {
		c := &releaseRun{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		return c
	},
}

type releaseRun struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
}

func (c *releaseRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	fmt.Fprintf(a.GetOut(), "In real life this would cancel a lease.\n")
	return 0
}
