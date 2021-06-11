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
	"google.golang.org/api/iterator"

	"infra/cros/cmd/crosgrep/internal/swarming/logging"
	"infra/cros/cmd/crosgrep/internal/swarming/queries"
)

// BrokenBy is a command that identifies the last successful task to run
// on a given device.
// TODO(gregorynisbet): change broken-by to check whether the device is currently broken
// and apply some heuristics to find guesses for the task which broke it.
var BrokenBy = &subcommands.Command{
	UsageLine: `broken-by`,
	ShortDesc: `find last successful task on DUT`,
	LongDesc:  `find last successful task on DUT.`,
	CommandRun: func() subcommands.CommandRun {
		c := &brokenByCmd{}
		c.Flags.StringVar(&c.bqProject, "bq-project", "", "BigQuery Project for use in queries, falls back to CROSGREP_BQ_PROJECT envvar")
		c.Flags.BoolVar(&c.verbose, "verbose", false, `Set the verbosity of diagnostic messages.`)
		c.Flags.StringVar(&c.botID, "bot-id", "", "The bot ID to search for.")
		return c
	},
}

// BrokenByCmd is the command that searches for the last successful task to run on a device.
// TODO(gregorynisbet): Create common subcommand base internal to crosgrep.
type brokenByCmd struct {
	subcommands.CommandRunBase
	verbose   bool
	bqProject string
	botID     string
}

// GetBQProject returns the cloud project for bigquery explicitly specified on the command line
// or taken from the CROSGREP_BQ_PROJECT environment variable if no flag is provided.
func (c *brokenByCmd) getBQProject() string {
	if c.bqProject == "" {
		return os.Getenv(crosgrepBQProjectEnvvar)
	}
	return c.bqProject
}

// Run parses arguments and runs a command.
func (c *brokenByCmd) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		fmt.Fprintf(a.GetErr(), "%s: %s\n", a.GetName(), err)
		return 1
	}
	return 0
}

// InnerRun is the main implementation of the broken-by command.
func (c *brokenByCmd) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	ctx := cli.GetContext(a, c, env)
	ctx = logging.SetContextVerbosity(ctx, c.verbose)
	client, err := bigquery.NewClient(ctx, c.getBQProject())
	if err != nil {
		return errors.Annotate(err, "getting bigquery client").Err()
	}
	it, err := queries.RunBrokenBy(
		ctx,
		client,
		&queries.BrokenByParams{
			BotID: c.botID,
			// TODO(gregorynisbet): Replace the placeholder value with a value from the command line.
			StartTime: 0,
			// TODO(gregorynisbet): Replace the placeholder value with a value from the command line.
			EndTime: 0,
		},
	)
	if err != nil {
		return errors.Annotate(err, "get result set").Err()
	}
	for {
		var item map[string]bigquery.Value
		err := it.Next(&item)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return errors.Annotate(err, "extracting item from result set").Err()
		}
		fmt.Fprintf(a.GetOut(), "%#v\n", item)
	}
	return nil
}
