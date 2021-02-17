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

var test = &subcommands.Command{
	UsageLine: "test [FLAGS...] TEST_NAME [TEST_NAME...]",
	ShortDesc: "runs an individual test",
	LongDesc: `Launches an individual test task with the given test name.

You must supply -board, -image, and -pool.

This command does not wait for the task to start running.

This command's behavior is subject to change without notice.
Do not build automation around this subcommand.`,
	CommandRun: func() subcommands.CommandRun {
		c := &testRun{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		return c
	},
}

type testRun struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
}

func (c *testRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	fmt.Fprintf(a.GetOut(), "In real life this would test an individual test.\n")
	return 0
}
