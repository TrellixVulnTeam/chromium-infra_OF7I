// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmd

import (
	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/flag"

	"infra/cmd/skylab/internal/cmd/cmdlib"
	"infra/cmd/skylab/internal/site"
)

const showTaskLimit = 5

const rerunTagKey = "skylab-tool"
const rerunTagVal = "rerun-tasks"

// RerunTasks subcommand.
var RerunTasks = &subcommands.Command{
	UsageLine: "rerun-tasks [-task-id TASK_ID...] [-tag TAG...]",
	ShortDesc: "Deprecated, do not use.",
	LongDesc:  `Create copies of tasks to run again.`,
	CommandRun: func() subcommands.CommandRun {
		c := &rerunTasksRun{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.Flags.BoolVar(&c.outputJSON, "output-json", false, "Format output as JSON.")
		c.Flags.Var(flag.StringSlice(&c.taskIds), "task-id", "Swarming task ids for locating tests to retry. If it's a retry task which is kicked off by rerun-tasks command, it won't be retried. May be specified multiple times.")
		c.Flags.Var(flag.StringSlice(&c.tags), "tag", "Tasks that match all these tags (and that were not already a retry task) will be retried. Task-id and tag cannot be both specified. May be specified multiple times.")
		c.Flags.BoolVar(&c.includePassed, "include-passed", false, "If true, rerun tasks even if they passed the first time. Only apply to tasks matched by tags.")
		c.Flags.BoolVar(&c.dryRun, "dry-run", false, "Print tasks that would be rerun, but don't actually rerun them.")
		c.Flags.BoolVar(&c.preserveParent, "preserve-parent", false, "Preserve the parent task ID of retried tasks. This should be used only within the context of a suite retrying its own children.")
		return c
	},
}

type rerunTasksRun struct {
	subcommands.CommandRunBase
	authFlags      authcli.Flags
	envFlags       cmdlib.EnvFlags
	outputJSON     bool
	taskIds        []string
	tags           []string
	includePassed  bool
	dryRun         bool
	preserveParent bool
}

func (c *rerunTasksRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a.GetErr(), err)
		return 1
	}
	return 0
}

func (c *rerunTasksRun) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	return errors.New("rerun-tasks subcommand no longer supported")
}
