// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmds

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
	"infra/appengine/crosskylabadmin/site"
	"infra/cmdsupport/cmdlib"
)

// GetStableVersion is a command for the GetStableVersion RPC.
// It intentionally performs no validation of its own and forwards requests to CSA as is.
var GetStableVersion = &subcommands.Command{
	UsageLine: `get-stable-version`,
	ShortDesc: `Get the stable version`,
	CommandRun: func() subcommands.CommandRun {
		r := &getStableVersionRun{}
		r.authFlags.Register(&r.Flags, site.DefaultAuthOptions)
		r.Flags.StringVar(&r.board, "board", "", "the board or build target")
		r.Flags.StringVar(&r.model, "model", "", "the model")
		r.Flags.StringVar(&r.hostname, "hostname", "", "the hostname")
		return r
	},
}

// GetStableVersionRun is the command for adminclient get-stable-version.
type getStableVersionRun struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
	board     string
	model     string
	hostname  string
}

// Run runs the command, prints the error if there is one, and returns an exit status.
func (c *getStableVersionRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	ctx := cli.GetContext(a, c, env)
	if err := c.innerRun(ctx, a, args, env); err != nil {
		fmt.Fprintf(a.GetErr(), "%s: %s\n", a.GetName(), err)
		return 1
	}
	return 0
}

// InnerRun creates a client, sends a GetStableVersion request, and prints the response.
func (c *getStableVersionRun) innerRun(ctx context.Context, a subcommands.Application, args []string, env subcommands.Env) error {
	hc, err := cmdlib.NewHTTPClient(ctx, &c.authFlags)
	if err != nil {
		return errors.Annotate(err, "get stable version").Err()
	}

	// TODO(gregorynisbet): Only set insecure to true if we are looking at a localhost-like address.
	var options prpc.Options = *site.DefaultPRPCOptions
	options.Insecure = true

	invWithSVClient := fleet.NewInventoryPRPCClient(
		&prpc.Client{
			C: hc,
			// TODO(gregorynisbet): Do not hardcode the CrOSSkylabAdmin server.
			Host:    "127.0.0.1:8800",
			Options: &options,
		},
	)
	resp, err := invWithSVClient.GetStableVersion(ctx, &fleet.GetStableVersionRequest{
		BuildTarget: c.board,
		Model:       c.model,
		Hostname:    c.hostname,
	})
	if err != nil {
		return errors.Annotate(err, "get stable version").Err()
	}
	out, err := protojson.Marshal(resp)
	if err != nil {
		return errors.Annotate(err, "get stable version").Err()
	}
	fmt.Fprintf(a.GetOut(), "%s\n", out)
	return nil
}
