// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cli

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/jsonpb"
	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"

	kartepb "infra/cros/karte/api"
	"infra/cros/karte/client"
	"infra/cros/karte/internal/errors"
	"infra/cros/karte/internal/site"
)

// PersistSingleAction is a command that persists a single action to BigQuery.
// Since this command is destructive, it only affects the dev instance of Karte.
var PersistSingleAction = &subcommands.Command{
	UsageLine: `persist-single-action [id]`,
	ShortDesc: `Persist a single action to BigQuery in an ad hoc way`,
	LongDesc:  ``,
	CommandRun: func() subcommands.CommandRun {
		r := &persistSingleActionRun{}
		r.authFlags.Register(&r.Flags, site.DefaultAuthOptions)
		return r
	},
}

// persistSingleActionRun runs the command.
type persistSingleActionRun struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
}

// Run logs an error message if there is one and returns an exit status.
func (c *persistSingleActionRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	ctx := cli.GetContext(a, c, env)
	if err := c.innerRun(ctx, a, args, env); err != nil {
		fmt.Fprintf(a.GetErr(), "%s: %s\n", a.GetName(), err)
		return 1
	}
	return 0
}

// innerRun validates arguments and performs the RPC call.
func (c *persistSingleActionRun) innerRun(ctx context.Context, a subcommands.Application, args []string, env subcommands.Env) error {
	if len(args) == 0 {
		return errors.Reason("persist single action: need an action").Err()
	}
	if len(args) > 1 {
		return errors.Reason("persist single action: multiple actions not supported").Err()
	}
	authOptions, err := c.authFlags.Options()
	if err != nil {
		return errors.Annotate(err, "inner run").Err()
	}
	kClient, err := client.NewClient(ctx, client.DevConfig(authOptions))
	if err != nil {
		return errors.Annotate(err, "inner run").Err()
	}
	res, err := kClient.PersistAction(ctx, &kartepb.PersistActionRequest{
		ActionId: args[0],
	})
	marshalIndent := jsonpb.Marshaler{
		Indent: "  ",
	}
	return errors.Annotate(marshalIndent.Marshal(a.GetErr(), res), "marshal JSON").Err()
}
