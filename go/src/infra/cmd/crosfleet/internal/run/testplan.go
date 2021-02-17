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

var testplan = &subcommands.Command{
	UsageLine: "testplan [FLAGS...]",
	ShortDesc: "runs a test plan",
	LongDesc: `Launches a test plan task for a given test plan file.

You must supply -board, -image, -pool, and -plan-file.

This command is more general than "run test" or "run suite". The supplied
test plan should conform to the TestPlan proto as defined here:
https://chromium.googlesource.com/chromiumos/infra/proto/+/master/src/test_platform/request.proto

This command does not wait for the task to start running.

This command's behavior is subject to change without notice.
Do not build automation around this subcommand.`,
	CommandRun: func() subcommands.CommandRun {
		c := &planRun{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		return c
	},
}

type planRun struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
}

func (c *planRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	fmt.Fprintf(a.GetOut(), "In real life this would run a test plan.\n")
	return 0
}
