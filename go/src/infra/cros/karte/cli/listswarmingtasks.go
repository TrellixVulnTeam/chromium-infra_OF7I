// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cli

import (
	"context"
	"fmt"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"

	kartepb "infra/cros/karte/api"
	"infra/cros/karte/client"
	"infra/cros/karte/internal/errors"
	"infra/cros/karte/internal/site"
)

// ListSwarmingTasks lists the swarming tasks in the recent past.
var ListSwarmingTasks = &subcommands.Command{
	UsageLine: `list-swarming-tasks`,
	ShortDesc: `list recent swarming tasks`,
	LongDesc:  `Take the most recent 1000 actions and show which swarming tasks executed them.`,
	CommandRun: func() subcommands.CommandRun {
		r := &listSwarmingTasksRun{}
		r.authFlags.Register(&r.Flags, site.DefaultAuthOptions)
		return r
	},
}

// ListSwarmingTasksRun is the list-tasks command.
type listSwarmingTasksRun struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
}

// Run logs an error message if there is one and returns an exit status.
func (c *listSwarmingTasksRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	ctx := cli.GetContext(a, c, env)
	if err := c.innerRun(ctx, a, args, env); err != nil {
		fmt.Fprintf(a.GetErr(), "%s: %s\n", a.GetName(), err)
		return 1
	}
	return 0
}

// ListSwarmingTasksNumEvents is the number of events that we look at when listing recent swarming tasks.
const listSwarmingTasksNumEvents = 1000

// InnerRun is the main implementation of list-swarming-tasks. List-swarming-tasks looks a constant number of actions into the past
// and then tallies how many of them belong to each swarming task ID.
func (c *listSwarmingTasksRun) innerRun(ctx context.Context, a subcommands.Application, args []string, env subcommands.Env) error {
	if len(args) > 0 {
		return fmt.Errorf("list-actions: positional arguments not supported")
	}
	authOptions, err := c.authFlags.Options()
	if err != nil {
		return errors.Annotate(err, "inner run").Err()
	}
	kClient, err := client.NewClient(ctx, client.DevConfig(authOptions))
	if err != nil {
		return errors.Annotate(err, "inner run").Err()
	}
	// Just list the swarming tasks regardless. We *could* only check for plan:... events in Karte, but this is optimistic.
	res, err := kClient.ListActions(ctx, &kartepb.ListActionsRequest{
		PageSize: listSwarmingTasksNumEvents,
		Filter:   "",
	})
	if err != nil {
		return errors.Annotate(err, "inner run").Err()
	}
	var swarmingTasks []string
	swarmingTasksSeenTally := make(map[string]int)
	for _, action := range res.GetActions() {
		if _, ok := swarmingTasksSeenTally[action.GetSwarmingTaskId()]; !ok {
			swarmingTasks = append(swarmingTasks, action.GetSwarmingTaskId())
		}
		swarmingTasksSeenTally[action.GetSwarmingTaskId()]++
	}
	for _, taskID := range swarmingTasks {
		fmt.Fprintf(a.GetOut(), "%q: %d\n", taskID, swarmingTasksSeenTally[taskID])
	}
	return nil
}
