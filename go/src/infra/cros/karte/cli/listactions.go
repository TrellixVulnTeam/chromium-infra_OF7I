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

	"infra/cros/karte/client"
	"infra/cros/karte/site"
)

// ListActions command lists all actions in Karte.
var ListActions = &subcommands.Command{
	UsageLine: `list-actions`,
	ShortDesc: "list actions",
	LongDesc:  "List all actions on the karte server.",
	CommandRun: func() subcommands.CommandRun {
		r := &listActionsRun{}
		r.authFlags.Register(&r.Flags, site.DefaultAuthOptions)
		// TODO(gregorynisbet): add envFlags.
		return r
	},
}

type listActionsRun struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
}

// Run runs listactions and returns an exit status.
func (c *listActionsRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	ctx := cli.GetContext(a, c, env)
	if err := c.innerRun(ctx, args, env); err != nil {
		fmt.Fprintf(a.GetErr(), "%s: %s\n", a.GetName(), err)
		return 1
	}
	return 0
}

func (c *listActionsRun) innerRun(ctx context.Context, args []string, env subcommands.Env) error {
	authOptions, err := c.authFlags.Options()
	if err != nil {
		return err
	}
	_, err = client.NewClient(ctx, client.LocalConfig(authOptions))
	// TODO(gregorynisbet): Expand innerRun so it that lists the actions instead
	// of checking whether it was able to instantiate the client or not.
	return err
}
