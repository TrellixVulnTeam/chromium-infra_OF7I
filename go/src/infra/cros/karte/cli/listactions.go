// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cli

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"

	kartepb "infra/cros/karte/api"
	"infra/cros/karte/client"
	"infra/cros/karte/internal/errors"
	"infra/cros/karte/internal/site"
)

// DefaultPageSize is the number of actions to request in each batch.
const defaultPageSize = 100

// ListActions command lists all actions in Karte.
var ListActions = &subcommands.Command{
	UsageLine: `list-actions`,
	ShortDesc: "list actions",
	LongDesc:  "List all actions on the karte server.",
	CommandRun: func() subcommands.CommandRun {
		r := &listActionsRun{}
		r.authFlags.Register(&r.Flags, site.DefaultAuthOptions)
		r.Flags.StringVar(&r.filter, "filter", "", "Karte query command")
		// TODO(gregorynisbet): add envFlags.
		return r
	},
}

type listActionsRun struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
	filter    string
}

// Run runs listactions and returns an exit status.
func (c *listActionsRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	ctx := cli.GetContext(a, c, env)
	if err := c.innerRun(ctx, a, args, env); err != nil {
		fmt.Fprintf(a.GetErr(), "%s: %s\n", a.GetName(), err)
		return 1
	}
	return 0
}

func (c *listActionsRun) innerRun(ctx context.Context, a subcommands.Application, args []string, env subcommands.Env) error {
	if len(args) > 0 {
		return fmt.Errorf("list-actions: positional arguments not supported")
	}
	authOptions, err := c.authFlags.Options()
	if err != nil {
		return errors.Annotate(err, "inner run").Err()
	}
	kClient, err := client.NewClient(ctx, client.DevConfig(authOptions))
	res, err := kClient.ListActions(ctx, &kartepb.ListActionsRequest{
		PageSize: defaultPageSize,
		Filter:   c.filter,
	})
	if err != nil {
		return errors.Annotate(err, "inner run").Err()
	}
	b, err := json.MarshalIndent(res, "", "    ")
	if err != nil {
		return errors.Annotate(err, "inner run").Err()
	}
	fmt.Fprintf(a.GetOut(), "%s\n", string(b))
	return errors.Annotate(err, "inner run").Err()
}
