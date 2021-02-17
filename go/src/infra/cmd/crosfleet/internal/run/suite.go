// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package run

import (
	"fmt"
	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"infra/cmd/crosfleet/internal/site"
)

var suite = &subcommands.Command{
	UsageLine: "suite [FLAGS...] SUITE_NAME",
	ShortDesc: "runs a test suite",
	LongDesc: `Launches a suite task with the given suite name.

You must supply -board, -image, and -pool.

This command does not wait for the task to start running.

This command's behavior is subject to change without notice.
Do not build automation around this subcommand.`,
	CommandRun: func() subcommands.CommandRun {
		c := &suiteRun{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		return c
	},
}

type suiteRun struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
}

func (c *suiteRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	fmt.Fprintf(a.GetOut(), "In real life this would launch a test suite.\n")
	return 0
}
