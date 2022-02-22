// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package run

import (
	"fmt"

	"infra/cmd/crosfleet/internal/buildbucket"
	"infra/cmd/crosfleet/internal/common"
	"infra/cmd/crosfleet/internal/site"
	"infra/cmd/crosfleet/internal/ufs"
	"infra/cmdsupport/cmdlib"

	"github.com/maruel/subcommands"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
)

// testCmdName is the name of the `crosfleet run test` command.
const testCmdName = "test"

var test = &subcommands.Command{
	UsageLine: fmt.Sprintf("%s [FLAGS...] TEST_NAME [TEST_NAME...]", testCmdName),
	ShortDesc: "runs an individual test",
	LongDesc: `Launches an individual test task with the given test name.

You must supply -board and -pool.

This command does not wait for the task to start running.

This command's behavior is subject to change without notice.
Do not build automation around this subcommand.`,
	CommandRun: func() subcommands.CommandRun {
		c := &testRun{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.printer.Register(&c.Flags)
		c.testCommonFlags.register(&c.Flags)
		c.Flags.StringVar(&c.testArgs, "test-args", "", "Test arguments string (meaning depends on test).")
		return c
	},
}

type testRun struct {
	subcommands.CommandRunBase
	testCommonFlags
	authFlags authcli.Flags
	envFlags  common.EnvFlags
	printer   common.CLIPrinter
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
	bbService := c.envFlags.Env().BuildbucketService
	ctx := cli.GetContext(a, c, env)
	if err := c.validateAndAutocompleteFlags(ctx, &c.Flags, testCmdName, bbService, c.authFlags, c.printer); err != nil {
		return err
	}
	ctpBBClient, err := buildbucket.NewClient(ctx, c.envFlags.Env().CTPBuilder, bbService, c.authFlags)
	if err != nil {
		return err
	}

	ufsClient, err := ufs.NewUFSClient(ctx, c.envFlags.Env().UFSService, &c.authFlags)
	if err != nil {
		return err
	}

	fleetValidationResults, err := c.verifyFleetTestsPolicy(ctx, ufsClient, testCmdName, args, true)
	if err != nil {
		return err
	}
	if err = checkAndPrintFleetValidationErrors(*fleetValidationResults, c.printer, testCmdName); err != nil {
		return err
	}
	if fleetValidationResults.testValidationErrors != nil {
		c.models = fleetValidationResults.validModels
		args = fleetValidationResults.validTests
	}

	testLauncher := ctpRunLauncher{
		mainArgsTag: testOrSuiteNamesTag(args),
		printer:     c.printer,
		cmdName:     testCmdName,
		bbClient:    ctpBBClient,
		testPlan:    testPlanForTests(c.testArgs, args),
		cliFlags:    &c.testCommonFlags,
		exitEarly:   c.exitEarly,
	}
	return testLauncher.launchAndOutputTests(ctx)
}

// testPlanForTests constructs a Test Platform test plan for the given tests.
func testPlanForTests(testArgs string, testNames []string) *test_platform.Request_TestPlan {
	// Due to crbug/984103, the first autotest arg gets dropped somewhere between here and
	// when autotest reads the args. Add a dummy arg to prevent this bug for now.
	// TODO(crbug/984103): Remove the dummy arg once the underlying bug is fixed.
	if testArgs != "" {
		testArgs = "dummy=crbug/984103 " + testArgs
	}
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
