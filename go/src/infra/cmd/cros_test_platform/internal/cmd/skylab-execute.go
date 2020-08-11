// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmd

import (
	"context"
	"fmt"

	"github.com/maruel/subcommands"

	"go.chromium.org/luci/common/cli"

	"infra/cmd/cros_test_platform/luciexe/execute"
)

// SkylabExecute subcommand: Run a set of enumerated tests against skylab backend.
var SkylabExecute = &subcommands.Command{
	UsageLine: "skylab-execute -input_json /path/to/input.json -output_json /path/to/output.json",
	ShortDesc: "Run a set of enumerated tests against skylab backend.",
	LongDesc:  `Run a set of enumerated tests against skylab backend.`,
	CommandRun: func() subcommands.CommandRun {
		c := &skylabExecuteRun{}
		c.Flags.StringVar(&c.inputPath, "input_json", "", "Path to JSON ExecuteRequests to read.")
		c.Flags.StringVar(&c.outputPath, "output_json", "", "Path to JSON ExecuteResponses to write.")
		return c
	},
}

type skylabExecuteRun struct {
	subcommands.CommandRunBase
	inputPath  string
	outputPath string
}

func (c *skylabExecuteRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.validateArgs(); err != nil {
		fmt.Fprintln(a.GetErr(), err.Error())
		c.Flags.Usage()
		return exitCode(err)
	}

	ctx := cli.GetContext(a, c, env)
	ctx = setupLogging(ctx)
	err := c.innerRun(ctx, args, env)
	if err != nil {
		logApplicationError(ctx, a, err)
	}
	return exitCode(err)
}

func (c *skylabExecuteRun) validateArgs() error {
	if c.inputPath == "" {
		return fmt.Errorf("-input_json not specified")
	}

	if c.outputPath == "" {
		return fmt.Errorf("-output_json not specified")
	}

	return nil
}

func (c *skylabExecuteRun) innerRun(ctx context.Context, args []string, env subcommands.Env) error {
	return execute.Run(ctx, execute.Args{
		InputPath:      c.inputPath,
		OutputPath:     c.outputPath,
		SwarmingTaskID: env["SWARMING_TASK_ID"].Value,
	})
}
