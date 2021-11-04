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

// CreateAction is a CLI command that creates an action on the Karte server.
var CreateAction = &subcommands.Command{
	UsageLine: `create-action`,
	ShortDesc: "create action",
	LongDesc:  "Create an action on the karte server.",
	CommandRun: func() subcommands.CommandRun {
		r := &createActionRun{}
		r.authFlags.Register(&r.Flags, site.DefaultAuthOptions)
		// TODO(gregorynisbet): add envFlags
		r.Flags.StringVar(&r.kind, "kind", "", "The action kind")
		r.Flags.StringVar(&r.swarmingTaskID, "task-id", "", "The ID of the swarming task")
		r.Flags.StringVar(&r.assetTag, "asset-tag", "", "The asset tag")
		r.Flags.StringVar(&r.failReason, "fail-reason", "", "The fail reason")
		return r
	},
}

// CreateActionRun runs create-action.
type createActionRun struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
	// Action fields
	kind           string
	swarmingTaskID string
	assetTag       string
	// TODO(gregorynisbet): Support times.
	// startTime      string
	// stopTime       string
	// TODO(gregorynisbet): Support status.
	// status     string
	failReason string
	// TODO(gregorynisbet): Support times.
	// sealTime string
}

// NontrivialActionFields counts the number of fields in the action to be created with a non-default value.
func (c *createActionRun) nontrivialActionFields() int {
	tally := 0
	if c.kind != "" {
		tally++
	}
	if c.swarmingTaskID != "" {
		tally++
	}
	if c.assetTag != "" {
		tally++
	}
	if c.failReason != "" {
		tally++
	}
	return tally
}

// Run creates an action and returns an exit status.
func (c *createActionRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	ctx := cli.GetContext(a, c, env)
	if err := c.innerRun(ctx, a, args, env); err != nil {
		fmt.Fprintf(a.GetErr(), "%s: %s\n", a.GetName(), err)
		return 1
	}
	return 0
}

// InnerRun creates an action and returns an error.
func (c *createActionRun) innerRun(ctx context.Context, a subcommands.Application, args []string, env subcommands.Env) error {
	if len(args) != 0 {
		return fmt.Errorf("positional arguments are not accepted")
	}
	tally := c.nontrivialActionFields()
	if tally == 0 {
		return fmt.Errorf("refusing to create empty action")
	}
	authOptions, err := c.authFlags.Options()
	if err != nil {
		return errors.Annotate(err, "create action").Err()
	}
	kClient, err := client.NewClient(ctx, client.DevConfig(authOptions))
	// TODO(gregorynisbet): Factor this into a separate function.
	action := &kartepb.Action{}
	action.Kind = c.kind
	action.SwarmingTaskId = c.swarmingTaskID
	action.AssetTag = c.assetTag
	action.FailReason = c.failReason
	out, err := kClient.CreateAction(ctx, &kartepb.CreateActionRequest{Action: action})
	if err != nil {
		return errors.Annotate(err, "create action").Err()
	}
	fmt.Fprintf(a.GetOut(), "%s\n", out.String())
	return nil
}
