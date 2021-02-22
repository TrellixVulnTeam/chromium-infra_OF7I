// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package run

import (
	"fmt"

	"github.com/maruel/subcommands"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"infra/cmd/crosfleet/internal/buildbucket"
	"infra/cmd/crosfleet/internal/common"
	"infra/cmd/crosfleet/internal/site"
	"infra/cmdsupport/cmdlib"
)

// testCmdName is the name of the `crosfleet run test` command.
var testCmdName = "test"

var test = &subcommands.Command{
	UsageLine: fmt.Sprintf("%s [FLAGS...] TEST_NAME [TEST_NAME...]", testCmdName),
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
	if err := c.validateArgs(&c.Flags, testCmdName); err != nil {
		return err
	}
	testPlan := testPlanForTests(c.testArgs, args)
	testNamesLabel := testOrSuiteNamesLabel(args)
	buildTags := c.buildTags(testCmdName, testNamesLabel)

	ctx := cli.GetContext(a, c, env)
	bbClient, err := buildbucket.NewClient(ctx, c.envFlags.Env().CTPBuilderInfo, c.authFlags)
	if err != nil {
		return err
	}
	buildID, err := launchCTPBuild(ctx, bbClient, testPlan, buildTags, &c.testCommonFlags)
	if err != nil {
		return err
	}
	fmt.Fprintf(a.GetOut(), "Running %s at %s\n", testCmdName, bbClient.BuildURL(buildID))
	return nil
}

// testPlanForTests constructs a Test Platform test plan for the given tests.
func testPlanForTests(testArgs string, testNames []string) *test_platform.Request_TestPlan {
	testPlan := &test_platform.Request_TestPlan{}
	for _, testName := range testNames {
		testRequest := &test_platform.Request_Test{
			Harness: &test_platform.Request_Test_Autotest_{
				Autotest: &test_platform.Request_Test_Autotest{
					Name:     testName,
					TestArgs: testArgs,
				},
			},
		}
		testPlan.Test = append(testPlan.Test, testRequest)
	}
	return testPlan
}
