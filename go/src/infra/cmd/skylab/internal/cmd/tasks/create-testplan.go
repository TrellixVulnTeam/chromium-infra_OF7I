// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package tasks

import (
	"fmt"
	"os"

	"github.com/maruel/subcommands"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"

	"infra/cmd/skylab/internal/bb"
	skycmdlib "infra/cmd/skylab/internal/cmd/cmdlib"
	"infra/cmd/skylab/internal/site"
	"infra/cmdsupport/cmdlib"
)

// CreateTestPlan subcommand: create a testplan task.
var CreateTestPlan = &subcommands.Command{
	UsageLine: `create-testplan [FLAGS...]`,
	ShortDesc: "create a testplan task",
	LongDesc: `Create a testplan task. [DEPRECATED--please use crosfleet (go/crosfleet-cli)]

[DEPRECATED--please use crosfleet (go/crosfleet-cli)]

This command is more general than create-test or create-suite. The supplied
testplan should conform to the TestPlan proto as defined here:
https://chromium.googlesource.com/chromiumos/infra/proto/+/master/src/test_platform/request.proto

You must supply -board, -image, -pool, and -plan-file.

This command does not wait for the task to start running.`,
	CommandRun: func() subcommands.CommandRun {
		c := &createTestPlanRun{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.createRunCommon.Register(&c.Flags)
		c.Flags.StringVar(&c.testplanPath, "plan-file", "", "Path to jsonpb-encoded test plan.")
		c.Flags.StringVar(&c.legacySuite, "legacy-suite", "", "DEPRECATED. Has no effect.")
		return c
	},
}

type createTestPlanRun struct {
	subcommands.CommandRunBase
	createRunCommon
	authFlags    authcli.Flags
	envFlags     skycmdlib.EnvFlags
	testplanPath string
	// TODO(crbug.com/1161539) Delete once cbuildbot stops setting this on all
	// branches.
	legacySuite string
}

// validateArgs ensures that the command line arguments are valid.
func (c *createTestPlanRun) validateArgs() error {
	if err := c.createRunCommon.ValidateArgs(c.Flags); err != nil {
		return err
	}
	if c.testplanPath == "" {
		return cmdlib.NewUsageError(c.Flags, "missing -plan-file")
	}

	return nil
}

func (c *createTestPlanRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *createTestPlanRun) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	if err := c.validateArgs(); err != nil {
		return err
	}

	ctx := cli.GetContext(a, c, env)
	client, err := bb.NewClient(ctx, c.envFlags.Env().CTPBuilderInfo, c.authFlags)
	if err != nil {
		return err
	}

	recipeArgs, err := c.RecipeArgs(c.buildTags())
	if err != nil {
		return err
	}

	testPlan, err := c.readTestPlan(c.testplanPath)
	if err != nil {
		return err
	}

	recipeArgs.TestPlan = testPlan

	req, err := recipeArgs.TestPlatformRequest()
	if err != nil {
		return err
	}
	m := map[string]*test_platform.Request{"default": req}
	buildID, err := client.ScheduleCTPBuild(ctx, m, c.buildTags())
	if err != nil {
		return err
	}
	return printScheduledTaskJSON(a.GetOut(), "cros_test_platform", fmt.Sprintf("%d", buildID), client.BuildURL(buildID))
}

func (c *createTestPlanRun) readTestPlan(path string) (*test_platform.Request_TestPlan, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, errors.Annotate(err, "read test plan").Err()
	}
	defer file.Close()

	testPlan := test_platform.Request_TestPlan{}
	if err := cmdlib.JSONPBUnmarshaller.Unmarshal(file, &testPlan); err != nil {
		return nil, errors.Annotate(err, "read test plan").Err()
	}

	return &testPlan, nil
}

func (c *createTestPlanRun) buildTags() []string {
	return append(c.createRunCommon.BuildTags(), "skylab-tool:create-testplan")
}
