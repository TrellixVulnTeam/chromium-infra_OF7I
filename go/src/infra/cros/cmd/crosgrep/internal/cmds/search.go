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

// ListAllTasks is a command that lists some swarming tasks for the past hour.
var ListAllTasks = &subcommands.Command{
	UsageLine: `list-all-tasks`,
	ShortDesc: "List all the swarming tasks.",
	LongDesc:  "List all the swarming tasks.",
	CommandRun: func() subcommands.CommandRun {
		c := &listAllTasksCmd{}
		c.InitFlags()
		c.Flags.StringVar(&c.model, "model", "", "The model to search.")
		return c
	},
}

// ListAllTasksCmd is a command object that contains the values of command line arguments.
// TODO(gregorynisbet): Create common subcommand base internal to crosgrep.
type listAllTasksCmd struct {
	base.Command
	model string
}

// Run is the main entrypoint for the list all tasks command. It returns an exit status.
func (c listAllTasksCmd) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		fmt.Fprintf(a.GetErr(), "%s: %s\n", a.GetName(), err)
		return 1
	}
	return 0
}

// InnerRun has the main logic for the list all tasks command. It returns a go-level error.
func (c *listAllTasksCmd) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	ctx := cli.GetContext(a, c, env)
	ctx = logging.SetContextVerbosity(ctx, c.Verbose())
	client, err := bigquery.NewClient(ctx, c.GetBQProject())
	if err != nil {
		return errors.Annotate(err, "getting bigquery client").Err()
	}
	it, err := query.RunTaskQuery(
		ctx,
		client,
		&query.TaskQueryParams{
			Model: c.model,
		},
	)
	if err != nil {
		return errors.Annotate(err, "extracting values from query").Err()
	}
	for {
		var item map[string]bigquery.Value
		err := it.Next(&item)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		fmt.Fprintf(a.GetOut(), "%#v\n", item)
	}
	return nil
}
