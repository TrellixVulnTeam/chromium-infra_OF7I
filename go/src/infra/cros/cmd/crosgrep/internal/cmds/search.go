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
	"go.chromium.org/luci/common/data/text"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"

	"infra/cros/cmd/crosgrep/internal/swarming"
)

// maxSwarmingResults set to 10 temporarily to limit the amount of
// time it takes queries to run.
const maxSwarmingResults = 10

// environment variable to use for bigquer project
const crosgrepBqProjectEnvvar = "CROSGREP_BQ_PROJECT"

// ListAllTasks is a command that lists some swarming tasks in an
// arbitrary way.
var ListAllTasks = &subcommands.Command{
	UsageLine: `list-all-tasks`,
	ShortDesc: "List all the swarming tasks.",
	LongDesc:  "List all the swarming tasks.",
	CommandRun: func() subcommands.CommandRun {
		c := &listAllTasksCmd{}
		c.Flags.StringVar(&c.bqProject, "bq-project", "", "BigQuery Project for use in queries, falls back to CROSGREP_BQ_PROJECT envvar")
		c.logLevel = logging.Info
		c.Flags.Var(&c.logLevel, "log-level", text.Doc(`
		Log level, valid options are "debug", "info", "warning", "error". Default is "info".
		`))
		return c
	},
}

type listAllTasksCmd struct {
	subcommands.CommandRunBase
	logLevel  logging.Level
	bqProject string
}

// getBqProject returns the cloud project for bigquery explicitly specified on the command line
// or taken from the CROSGREP_BQ_PROJECT environment variable if no flag is provided.
func (c *listAllTasksCmd) getBqProject() string {
	if c.bqProject == "" {
		return os.Getenv(crosgrepBqProjectEnvvar)
	}
	return c.bqProject
}

func (c *listAllTasksCmd) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		fmt.Fprintf(a.GetErr(), "%s: %s\n", a.GetName(), err)
		return 1
	}
	return 0
}

func (c *listAllTasksCmd) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	ctx := cli.GetContext(a, c, env)
	logging.SetLevel(ctx, c.logLevel)
	client, err := bigquery.NewClient(ctx, c.getBqProject())
	if err != nil {
		return errors.Annotate(err, "getting bigquery client").Err()
	}
	vals, err := swarming.ExtractNValues(ctx, client, maxSwarmingResults)
	if err != nil {
		return errors.Annotate(err, "extracting values from query").Err()
	}
	for _, row := range vals {
		fmt.Fprintf(a.GetOut(), "%#v\n", row)
	}
	return nil
}
