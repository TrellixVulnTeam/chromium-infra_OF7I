// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmds

import (
	"fmt"
	"os"

	"cloud.google.com/go/bigquery"
	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"

	"infra/cros/cmd/crosgrep/internal/swarming"
	"infra/cros/cmd/crosgrep/internal/swarming/logging"
)

// CrosgrepBQProjectEnvvar is a variable to use for bigquery project.
const crosgrepBQProjectEnvvar = "CROSGREP_BQ_PROJECT"

// ListAllTasks is a command that lists some swarming tasks in an
// arbitrary way.
var ListAllTasks = &subcommands.Command{
	UsageLine: `list-all-tasks`,
	ShortDesc: "List all the swarming tasks.",
	LongDesc:  "List all the swarming tasks.",
	CommandRun: func() subcommands.CommandRun {
		c := &listAllTasksCmd{}
		c.Flags.StringVar(&c.bqProject, "bq-project", "", "BigQuery Project for use in queries, falls back to CROSGREP_BQ_PROJECT envvar")
		c.Flags.BoolVar(&c.verbose, "verbose", false, `Set the verbosity.`)
		c.Flags.StringVar(&c.model, "model", "", "The model to search.")
		return c
	},
}

// ListAllTasksCmd is a command that lists all tasks.
// TODO(gregorynisbet): Create common subcommand base internal to crosgrep.
type listAllTasksCmd struct {
	subcommands.CommandRunBase
	verbose   bool
	bqProject string
	model     string
}

// GetBQProject returns the cloud project for bigquery explicitly specified on the command line
// or taken from the CROSGREP_BQ_PROJECT environment variable if no flag is provided.
func (c *listAllTasksCmd) getBQProject() string {
	if c.bqProject == "" {
		return os.Getenv(crosgrepBQProjectEnvvar)
	}
	return c.bqProject
}

// Run is the main entrypoint for the search command.
func (c listAllTasksCmd) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		fmt.Fprintf(a.GetErr(), "%s: %s\n", a.GetName(), err)
		return 1
	}
	return 0
}

// InnerRun is the implementation of the search command.
func (c *listAllTasksCmd) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	ctx := cli.GetContext(a, c, env)
	ctx = logging.SetContextVerbosity(ctx, c.verbose)
	client, err := bigquery.NewClient(ctx, c.getBQProject())
	if err != nil {
		return errors.Annotate(err, "getting bigquery client").Err()
	}
	vals, err := swarming.ExtractValues(ctx, client, c.model)
	if err != nil {
		return errors.Annotate(err, "extracting values from query").Err()
	}
	for _, row := range vals {
		fmt.Fprintf(a.GetOut(), "%#v\n", row)
	}
	return nil
}
