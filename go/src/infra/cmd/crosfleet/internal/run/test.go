// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package run

import (
	"fmt"
	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"infra/cmd/crosfleet/internal/common"
	"infra/cmd/crosfleet/internal/site"
	"infra/cmdsupport/cmdlib"
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
		c.envFlags.Register(&c.Flags)
		c.testCommonFlags.Register(&c.Flags)
		c.Flags.StringVar(&c.testArgs, "test-args", "", "Test arguments string (meaning depends on test).")
		return c
	},
}

type testRun struct {
	subcommands.CommandRunBase
	testCommonFlags
	authFlags authcli.Flags
	envFlags  common.EnvFlags
	testArgs  string
}

func (c *testRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *testRun) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	if err := c.validateArgs(&c.Flags, "test name"); err != nil {
		return err
	}
	fmt.Fprintf(a.GetOut(), "In real life this would run an individual test.\nFlags registered: %v\nArgs ergistered: %v\n", c.Flags, args)
	return nil
}
