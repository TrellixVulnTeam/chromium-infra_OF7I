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

// suiteCmdName is the name of the `crosfleet run suite` command.
const suiteCmdName = "suite"

var suite = &subcommands.Command{
	UsageLine: fmt.Sprintf("%s [FLAGS...] SUITE_NAME", suiteCmdName),
	ShortDesc: "runs a test suite",
	LongDesc: `Launches a suite task with the given suite name.

You must supply -board and -pool.

This command does not wait for the task to start running.

This command's behavior is subject to change without notice.
Do not build automation around this subcommand.`,
	CommandRun: func() subcommands.CommandRun {
		c := &suiteRun{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.printer.Register(&c.Flags)
		c.testCommonFlags.register(&c.Flags)
		return c
	},
}

type suiteRun struct {
	subcommands.CommandRunBase
	testCommonFlags
	authFlags authcli.Flags
	envFlags  common.EnvFlags
	printer   common.CLIPrinter
}

func (c *suiteRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *suiteRun) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	bbService := c.envFlags.Env().BuildbucketService
	ctx := cli.GetContext(a, c, env)
	if err := c.validateAndAutocompleteFlags(ctx, &c.Flags, suiteCmdName, bbService, c.authFlags, c.printer); err != nil {
		return err
	}
	ctpBBClient, err := buildbucket.NewClient(ctx, c.envFlags.Env().CTPBuilder, c.envFlags.Env().BuildbucketService, c.authFlags)
	if err != nil {
		return err
	}

	ufsClient, err := ufs.NewUFSClient(ctx, c.envFlags.Env().UFSService, &c.authFlags)
	if err != nil {
		return err
	}

	fleetValidationResults, err := c.verifyFleetTestsPolicy(ctx, ufsClient, suiteCmdName, args, true)
	if err != nil {
		return err
	}
	if err = checkAndPrintFleetValidationErrors(*fleetValidationResults, c.printer, suiteCmdName); err != nil {
		return err
	}
	if fleetValidationResults.testValidationErrors != nil {
		c.models = fleetValidationResults.validModels
		args = fleetValidationResults.validTests
	}

	testLauncher := ctpRunLauncher{
		mainArgsTag: testOrSuiteNamesTag(args),
		printer:     c.printer,
		cmdName:     suiteCmdName,
		bbClient:    ctpBBClient,
		testPlan:    testPlanForSuites(args),
		cliFlags:    &c.testCommonFlags,
		exitEarly:   c.exitEarly,
	}
	return testLauncher.launchAndOutputTests(ctx)
}

// testPlanForSuites constructs a Test Platform test plan for the given tests.
func testPlanForSuites(suiteNames []string) *test_platform.Request_TestPlan {
	testPlan := test_platform.Request_TestPlan{}
	for _, suiteName := range suiteNames {
		suiteRequest := &test_platform.Request_Suite{Name: suiteName}
		testPlan.Suite = append(testPlan.Suite, suiteRequest)
	}
	return &testPlan
}
