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
		c.InitFlags()
		c.Flags.StringVar(&c.botID, "bot-id", "", "The bot ID to search for.")
		return c
	},
}

// BrokenByCmd is the command that searches for the last successful task to run on a device.
// TODO(gregorynisbet): Create common subcommand base internal to crosgrep.
type brokenByCmd struct {
	base.Command
	botID string
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
	client, err := bigquery.NewClient(ctx, c.GetBQProject())
	if err != nil {
		return errors.Annotate(err, "broken-by: getting bigquery client").Err()
	}
	it, err := queries.RunBrokenBy(
		ctx,
		client,
		&queries.BrokenByParams{
			BotID: c.botID,
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
			return errors.Annotate(err, "broken-by: extracting item from result set").Err()
		}
		fmt.Fprintf(a.GetOut(), "%#v\n", item)
	}
	return nil
}
