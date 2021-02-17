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

var lease = &subcommands.Command{
	UsageLine: "lease HOST\n\tcrosfleet dut lease -model MODEL",
	ShortDesc: "lease DUT for debugging",
	LongDesc: `Lease DUT for debugging.

This command's behavior is subject to change without notice.
Do not build automation around this subcommand.`,
	CommandRun: func() subcommands.CommandRun {
		c := &leaseRun{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		return c
	},
}

type leaseRun struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
}

func (c *leaseRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	fmt.Fprintf(a.GetOut(), "In real life this would start a lease.\n")
	return 0
}
