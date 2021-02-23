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

// suiteCmdName is the name of the `crosfleet run suite` command.
const suiteCmdName = "suite"

var suite = &subcommands.Command{
	UsageLine: fmt.Sprintf("%s [FLAGS...] SUITE_NAME", suiteCmdName),
	ShortDesc: "runs a test suite",
	LongDesc: `Launches a suite task with the given suite name.

You must supply -board, -image, and -pool.

This command does not wait for the task to start running.

This command's behavior is subject to change without notice.
Do not build automation around this subcommand.`,
	CommandRun: func() subcommands.CommandRun {
		c := &suiteRun{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.testCommonFlags.register(&c.Flags)
		return c
	},
}

type suiteRun struct {
	subcommands.CommandRunBase
	testCommonFlags
	authFlags authcli.Flags
	envFlags  common.EnvFlags
}

func (c *suiteRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *suiteRun) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	if err := c.validateArgs(&c.Flags, suiteCmdName); err != nil {
		return err
	}
	testPlan := testPlanForSuites(args)
	suiteNamesLabel := testOrSuiteNamesLabel(args)
	buildTags := c.buildTags(testCmdName, suiteNamesLabel)

	ctx := cli.GetContext(a, c, env)
	bbClient, err := buildbucket.NewClient(ctx, c.envFlags.Env().CTPBuilderInfo, c.authFlags)
	if err != nil {
		return err
	}
	buildID, err := launchCTPBuild(ctx, bbClient, testPlan, buildTags, &c.testCommonFlags)
	if err != nil {
		return err
	}
	fmt.Fprintf(a.GetOut(), "Running %s at %s\n", suiteCmdName, bbClient.BuildURL(buildID))
	return nil
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
