// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package stableversion

import (
	"context"
	"fmt"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/grpc/prpc"
	"google.golang.org/protobuf/encoding/protojson"

	fleet "infra/appengine/crosskylabadmin/api/fleet/v1"
	"infra/cmdsupport/cmdlib"
	"infra/cros/cmd/satlab/internal/site"
)

// DeleteStableVersionCmd deletes a stable version entry.
var DeleteStableVersionCmd = &subcommands.Command{
	UsageLine: `delete-stable-version`,
	ShortDesc: `Delete the stable version using {board, model} or {hostname}. This only deletes satlab entries.`,
	CommandRun: func() subcommands.CommandRun {
		r := &deleteStableVersionRun{}

		r.authFlags.Register(&r.Flags, site.DefaultAuthOptions)
		r.envFlags.Register(&r.Flags)
		r.commonFlags.Register(&r.Flags)

		r.Flags.StringVar(&r.board, "board", "", `the board or build target (used with model)`)
		r.Flags.StringVar(&r.model, "model", "", `the model (used with board)`)
		r.Flags.StringVar(&r.hostname, "hostname", "", `the hostname (used by itself)`)

		return r
	},
}

// DeleteStableVersionRun is the command for adminclient set-stable-version.
type deleteStableVersionRun struct {
	subcommands.CommandRunBase

	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags

	board    string
	model    string
	hostname string
}

// Run runs the command, prints the error if there is one, and returns an exit status.
func (c *deleteStableVersionRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	ctx := cli.GetContext(a, c, env)
	if err := c.innerRun(ctx, a, args, env); err != nil {
		fmt.Fprintf(a.GetErr(), "%s: %s\n", a.GetName(), err)
		return 1
	}
	return 0
}

// InnerRun creates a client, sends a DeleteStableVersion request, and prints the response.
func (c *deleteStableVersionRun) innerRun(ctx context.Context, a subcommands.Application, args []string, env subcommands.Env) error {
	newHostname, err := preprocessHostname(c.commonFlags, c.hostname, nil, nil)
	if err != nil {
		return errors.Annotate(err, "set stable version").Err()
	}
	c.hostname = newHostname

	hc, err := cmdlib.NewHTTPClient(ctx, &c.authFlags)
	if err != nil {
		return errors.Annotate(err, "delete stable version").Err()
	}

	host := c.envFlags.GetCrosAdmService()

	options := site.DefaultPRPCOptions

	invWithSVClient := fleet.NewInventoryPRPCClient(
		&prpc.Client{
			C:       hc,
			Host:    host,
			Options: options,
		},
	)

	req := &fleet.DeleteSatlabStableVersionRequest{}
	if c.hostname == "" {
		req.Strategy = &fleet.DeleteSatlabStableVersionRequest_SatlabBoardModelDeletionCriterion{
			SatlabBoardModelDeletionCriterion: &fleet.SatlabBoardModelDeletionCriterion{
				Board: c.board,
				Model: c.model,
			},
		}
	} else {
		req.Strategy = &fleet.DeleteSatlabStableVersionRequest_SatlabHostnameDeletionCriterion{
			SatlabHostnameDeletionCriterion: &fleet.SatlabHostnameDeletionCriterion{
				Hostname: c.hostname,
			},
		}
	}

	resp, err := invWithSVClient.DeleteSatlabStableVersion(ctx, req)
	if err != nil {
		return errors.Annotate(err, "delete stable version").Err()
	}
	out, err := protojson.MarshalOptions{
		Indent: "    ",
	}.Marshal(resp)
	if err != nil {
		return errors.Annotate(err, "delete stable version").Err()
	}
	fmt.Fprintf(a.GetOut(), "%s\n", out)
	return nil
}
