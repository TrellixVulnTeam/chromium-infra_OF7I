// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmds

import (
	"context"
	"fmt"
	"log"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/grpc/prpc"
	"google.golang.org/protobuf/encoding/protojson"

	fleet "infra/appengine/crosskylabadmin/api/fleet/v1"
	"infra/appengine/crosskylabadmin/site"
	"infra/cmdsupport/cmdlib"
)

// DeleteStableVersion is a command for the DeleteStableVersion RPC.
// It intentionally performs no validation of its own and forwards requests to CSA as is.
var DeleteStableVersion = &subcommands.Command{
	UsageLine: `delete-stable-version`,
	ShortDesc: `Delete the satlab stable version. Non-satlab stable versions cannot be deleted.`,
	CommandRun: func() subcommands.CommandRun {
		r := &deleteStableVersionRun{}
		r.crOSAdminRPCRun.Register(&r.Flags)
		r.authFlags.Register(&r.Flags, site.DefaultAuthOptions)
		r.Flags.StringVar(&r.board, "board", "", "the board or build target")
		r.Flags.StringVar(&r.model, "model", "", "the model")
		r.Flags.StringVar(&r.hostname, "hostname", "", "the hostname")
		return r
	},
}

// DeleteStableVersionRun is the command for adminclient get-stable-version.
type deleteStableVersionRun struct {
	crOSAdminRPCRun
	authFlags authcli.Flags
	board     string
	model     string
	hostname  string
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

// InnerRun creates a client, sends a GetStableVersion request, and prints the response.
func (c *deleteStableVersionRun) innerRun(ctx context.Context, a subcommands.Application, args []string, env subcommands.Env) error {
	hc, err := cmdlib.NewHTTPClient(ctx, &c.authFlags)
	if err != nil {
		return errors.Annotate(err, "delete stable version").Err()
	}

	host, err := c.GetHost()
	if err != nil {
		return errors.Annotate(err, "delete stable version").Err()
	}

	options, err := c.GetOptions()
	if err != nil {
		return errors.Annotate(err, "delete stable version").Err()
	}

	invWithSVClient := fleet.NewInventoryPRPCClient(
		&prpc.Client{
			C: hc,
			// TODO(gregorynisbet): Do not hardcode the CrOSSkylabAdmin server.
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

	out, err := protojson.Marshal(req)
	if err != nil {
		return errors.Annotate(err, "delete stable version").Err()
	}
	log.Printf("Request Body: %s\n", out)

	resp, err := invWithSVClient.DeleteSatlabStableVersion(ctx, req)
	if err != nil {
		return errors.Annotate(err, "delete stable version").Err()
	}
	out, err = protojson.Marshal(resp)
	if err != nil {
		return errors.Annotate(err, "delete stable version").Err()
	}
	fmt.Fprintf(a.GetOut(), "%s\n", out)
	return nil
}
