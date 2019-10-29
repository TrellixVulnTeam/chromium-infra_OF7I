// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmd

import (
	"context"
	"fmt"

	"github.com/maruel/subcommands"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"

	"infra/cmd/skylab/internal/bb"
	"infra/cmd/skylab/internal/cmd/recipe"
	"infra/cmd/skylab/internal/site"
)

// CreateSuite subcommand: create a suite task.
var CreateSuite = &subcommands.Command{
	UsageLine: "create-suite [FLAGS...] SUITE_NAME",
	ShortDesc: "create a suite task",
	LongDesc:  "Create a suite task, with the given suite name.",
	CommandRun: func() subcommands.CommandRun {
		c := &createSuiteRun{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.createRunCommon.Register(&c.Flags)
		c.Flags.BoolVar(&c.orphan, "orphan", false, "Deprecated, do not use.")
		c.Flags.BoolVar(&c.json, "json", false, "Format output as JSON")
		c.Flags.StringVar(&c.taskName, "task-name", "", "Optional name to be used for the Swarming task.")
		return c
	},
}

type createSuiteRun struct {
	subcommands.CommandRunBase
	createRunCommon
	authFlags authcli.Flags
	envFlags  envFlags
	orphan    bool
	json      bool
	taskName  string
}

func (c *createSuiteRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		PrintError(a.GetErr(), err)
		return 1
	}
	return 0
}

func (c *createSuiteRun) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	if err := c.validateArgs(); err != nil {
		return err
	}

	ctx := cli.GetContext(a, c, env)
	suiteName := c.Flags.Arg(0)

	return c.innerRunBB(ctx, a, suiteName)
}

func (c *createSuiteRun) validateArgs() error {
	if err := c.createRunCommon.ValidateArgs(c.Flags); err != nil {
		return err
	}

	if c.Flags.NArg() == 0 {
		return NewUsageError(c.Flags, "missing suite name")
	}

	if c.orphan {
		return errors.Reason("-orphan is deprecated").Err()
	}

	return nil
}

func (c *createSuiteRun) innerRunBB(ctx context.Context, a subcommands.Application, suiteName string) error {
	client, err := bb.NewClient(ctx, c.envFlags.Env(), c.authFlags)
	if err != nil {
		return err
	}

	req, err := c.testPlatformRequest(suiteName)
	if err != nil {
		return err
	}
	buildID, err := client.ScheduleBuild(ctx, req, c.buildTags(suiteName))
	if err != nil {
		return err
	}
	buildURL := client.BuildURL(buildID)
	if c.json {
		return printScheduledTaskJSON(a.GetOut(), "cros_test_platform", fmt.Sprintf("%d", buildID), buildURL)
	}
	fmt.Fprintf(a.GetOut(), "Created request at %s\n", buildURL)
	return nil
}

func (c *createSuiteRun) testPlatformRequest(suite string) (*test_platform.Request, error) {
	recipeArgs, err := c.RecipeArgs(c.buildTags(suite))
	if err != nil {
		return nil, err
	}
	recipeArgs.TestPlan = recipe.NewTestPlanForSuites(suite)
	return recipeArgs.TestPlatformRequest()
}

func (c *createSuiteRun) buildTags(suiteName string) []string {
	return append(c.createRunCommon.BuildTags(), "skylab-tool:create-suite", fmt.Sprintf("suite:%s", suiteName))
}
