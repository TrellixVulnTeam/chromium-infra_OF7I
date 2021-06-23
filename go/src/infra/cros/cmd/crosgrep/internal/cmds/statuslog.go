// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmds

import (
	"fmt"

	"cloud.google.com/go/bigquery"
	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
	"google.golang.org/api/iterator"

	"infra/cros/cmd/crosgrep/internal/base"
	"infra/cros/cmd/crosgrep/internal/swarming/logging"
	"infra/cros/cmd/crosgrep/internal/swarming/query"
)

// StatusLog gets the status.log associated with a specific swarming task if a
// task ID is provided. Otherwise, it returns the status log associated with
// an arbitrary swarming task.
var StatusLog = &subcommands.Command{
	UsageLine: `status-log`,
	ShortDesc: "get the status log",
	LongDesc:  "Get the status log for a specified task or an arbitrary log if no task is specified.",
	CommandRun: func() subcommands.CommandRun {
		c := &statusLogCmd{}
		c.InitFlags()
		c.Flags.StringVar(&c.taskID, "task-id", "", "The task ID to search")
		return c
	},
}

// StatusLogCmd holds the arguments needed to get the status log of a task.
// There are the crosgrep common arguments and the task ID.
type statusLogCmd struct {
	base.Command
	taskID string
}

// Run is a wrapper around the main entrypoint for the status log command.
func (c *statusLogCmd) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		fmt.Fprintf(a.GetErr(), "%s: %s\n", a.GetName(), err)
		return 1
	}
	return 0
}

// InnerRun is the main entrypoint for the status-log commad.
func (c *statusLogCmd) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	ctx := cli.GetContext(a, c, env)
	ctx = logging.SetContextVerbosity(ctx, c.Verbose())
	client, err := bigquery.NewClient(ctx, c.GetBQProject())
	if err != nil {
		return errors.Annotate(err, "status-log: getting bigquery client").Err()
	}
	it, err := query.RunStatusLogQuery(
		ctx,
		client,
		&query.GetStatusLogParams{
			SwarmingTaskID: c.taskID,
		},
	)
	if err != nil {
		return errors.Annotate(err, "getting result set").Err()
	}
	for {
		var item map[string]bigquery.Value
		err := it.Next(&item)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return errors.Annotate(err, "status-log: extracting item from result set").Err()
		}
		fmt.Fprintf(a.GetOut(), "%#v\n", item)
	}
	return nil
}
