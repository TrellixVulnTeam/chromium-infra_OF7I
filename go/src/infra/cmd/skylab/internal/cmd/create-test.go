// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmd

import (
	"fmt"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"

	"infra/cmd/skylab/internal/bb"
	"infra/cmd/skylab/internal/cmd/cmdlib"
	"infra/cmd/skylab/internal/cmd/recipe"
	"infra/cmd/skylab/internal/site"
)

// CreateTest subcommand: create a test task.
var CreateTest = &subcommands.Command{
	UsageLine: `create-test [FLAGS...] TEST_NAME [TEST_NAME...]`,
	ShortDesc: "create a test task",
	LongDesc: `Create a test task.

You must supply -pool, -image, and one of -board or -model.

This command does not wait for the task to start running.`,
	CommandRun: func() subcommands.CommandRun {
		c := &createTestRun{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.createRunCommon.Register(&c.Flags)
		c.Flags.BoolVar(&c.client, "client-test", false, "(Deprecated).")
		c.Flags.StringVar(&c.testArgs, "test-args", "", "Test arguments string (meaning depends on test).")
		c.Flags.StringVar(&c.parentTaskID, "parent-task-run-id", "", "For internal use only. Task run ID of the parent (suite) task to this test. Note that this must be a run ID (i.e. not ending in 0).")

		return c
	},
}

type createTestRun struct {
	subcommands.CommandRunBase
	createRunCommon
	authFlags    authcli.Flags
	envFlags     cmdlib.EnvFlags
	client       bool
	testArgs     string
	parentTaskID string
}

// validateArgs ensures that the command line arguments are
func (c *createTestRun) validateArgs() error {
	if err := c.createRunCommon.ValidateArgs(c.Flags); err != nil {
		return err
	}

	if c.Flags.NArg() == 0 {
		return cmdlib.NewUsageError(c.Flags, "missing test name")
	}

	return nil
}

func (c *createTestRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a.GetErr(), err)
		return 1
	}
	return 0
}

func (c *createTestRun) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	if err := c.validateArgs(); err != nil {
		return err
	}

	return c.innerRunBB(a, args, env)
}

func (c *createTestRun) innerRunBB(a subcommands.Application, args []string, env subcommands.Env) error {
	if err := c.validateForBB(); err != nil {
		return err
	}

	ctx := cli.GetContext(a, c, env)
	e := c.envFlags.Env()
	client, err := bb.NewClient(ctx, e, c.authFlags)
	if err != nil {
		return err
	}

	recipeArg, err := c.RecipeArgs(c.buildTags())
	if err != nil {
		return err
	}
	recipeArg.TestPlan = recipe.NewTestPlanForAutotestTests(c.testArgs, args...)

	req, err := recipeArg.TestPlatformRequest()
	if err != nil {
		return err
	}
	buildID, err := client.ScheduleLegacyBuild(ctx, req, c.buildTags())
	if err != nil {
		return err
	}
	fmt.Fprintf(a.GetOut(), "Created request at %s\n", client.BuildURL(buildID))
	return nil
}

func (c *createTestRun) validateForBB() error {
	// TODO(akeshet): support for all of these arguments, or deprecate them.
	if c.parentTaskID != "" {
		return errors.Reason("parent task id not yet supported in -bb mode").Err()
	}
	if len(c.provisionLabels) != 0 {
		return errors.Reason("freeform provisionable labels not yet supported in -bb mode").Err()
	}
	return nil
}

func (c *createTestRun) buildTags() []string {
	return append(c.createRunCommon.BuildTags(), "skylab-tool:create-test")
}
