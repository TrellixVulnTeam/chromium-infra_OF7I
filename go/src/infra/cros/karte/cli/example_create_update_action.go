// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cli

import (
	"context"
	"fmt"
	kartepb "infra/cros/karte/api"
	"infra/cros/karte/client"
	"infra/cros/karte/internal/errors"
	"infra/cros/karte/internal/scalars"
	"infra/cros/karte/internal/site"
	"time"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
)

// ExampleCreateUpdateAction is a CLI command that creates an action on the Karte server.
var ExampleCreateUpdateAction = &subcommands.Command{
	UsageLine: `example-create-update-action`,
	ShortDesc: "create and update an action",
	LongDesc:  "Create and update an action on the karte server.",
	CommandRun: func() subcommands.CommandRun {
		r := &exampleCreateUpdateActionRun{}
		r.authFlags.Register(&r.Flags, site.DefaultAuthOptions)
		return r
	},
}

// exampleCreateUpdateActionRun runs create-action.
type exampleCreateUpdateActionRun struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
}

// Run creates an action and returns an exit status.
func (c *exampleCreateUpdateActionRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	ctx := cli.GetContext(a, c, env)
	if err := c.innerRun(ctx, a, args, env); err != nil {
		fmt.Fprintf(a.GetErr(), "%s: %s\n", a.GetName(), err)
		return 1
	}
	return 0
}

// innerRun creates an action and returns an error.
func (c *exampleCreateUpdateActionRun) innerRun(ctx context.Context, a subcommands.Application, args []string, env subcommands.Env) error {
	if len(args) != 0 {
		return fmt.Errorf("positional arguments are not accepted")
	}
	authOptions, err := c.authFlags.Options()
	if err != nil {
		return errors.Annotate(err, "create action").Err()
	}
	kClient, err := client.NewClient(ctx, client.DevConfig(authOptions))
	action := &kartepb.Action{
		Kind:       "d1375f6a-9a0a-41d5-91ff-6763dbb774cf",
		FailReason: "c13d7fe6-ba1d-4d8f-9292-a320b4519587",
		StartTime:  scalars.ConvertTimeToTimestampPtr(time.Now()),
	}
	out, err := kClient.CreateAction(ctx, &kartepb.CreateActionRequest{Action: action})
	if err != nil {
		return errors.Annotate(err, "create action").Err()
	}
	fmt.Fprintf(a.GetOut(), "%s\n", out.String())

	action = out
	action.FailReason = "1f6929e9-74d9-4b68-8e42-06b3862410fa"
	action.StopTime = scalars.ConvertTimeToTimestampPtr(time.Now())
	out, err = kClient.UpdateAction(ctx, &kartepb.UpdateActionRequest{
		Action: action,
	})
	if err != nil {
		return errors.Annotate(err, "update action").Err()
	}
	fmt.Fprintf(a.GetOut(), "%s\n", out.String())
	return nil
}
