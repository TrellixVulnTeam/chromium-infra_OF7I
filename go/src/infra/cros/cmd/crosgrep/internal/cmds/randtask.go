// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmds

import (
	"context"
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

// RandTask is a command that returns a random task that ran in the past hour.
var RandTask = &subcommands.Command{
	UsageLine: `rand-task`,
	ShortDesc: "get a random task",
	LongDesc:  "Get a random task.",
	CommandRun: func() subcommands.CommandRun {
		c := &randTaskCmd{}
		c.InitFlags()
		return c
	},
}

// RandTaskCmd is a command that returns a random task that ran in the past hour.
type randTaskCmd struct {
	base.Command
}

// Run parses arguments and runs a command.
func (c *randTaskCmd) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	ctx := cli.GetContext(a, c, env)
	ctx = logging.SetContextVerbosity(ctx, c.Verbose())
	if err := c.innerRun(ctx, a, args, env); err != nil {
		logging.Errorf(ctx, "%s: %s\n", a.GetName(), err)
		return 1
	}
	return 0
}

// InnerRun is the main implementation of the brand-task command.
func (c *randTaskCmd) innerRun(ctx context.Context, a subcommands.Application, args []string, env subcommands.Env) error {
	client, err := bigquery.NewClient(ctx, c.GetBQProject())
	if err != nil {
		return errors.Annotate(err, "rand-task: getting BigQuery client for project %q", c.GetBQProject()).Err()
	}
	it, err := query.RunRandTaskQuery(
		ctx,
		client,
		&query.RandTaskParams{},
	)
	if err != nil {
		return errors.Annotate(err, "rand-task: get result set").Err()
	}
	for {
		var item map[string]bigquery.Value
		err := it.Next(&item)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return errors.Annotate(err, "rand-task: extracting item from result set").Err()
		}
		fmt.Fprintf(a.GetOut(), "%#v\n", item)
	}
	return nil
}
