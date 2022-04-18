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

// GetStableVersionCmd is a command for the GetStableVersion RPC.
// It intentionally performs no validation of its own and forwards requests to CSA as is.
var GetStableVersionCmd = &subcommands.Command{
	UsageLine: `get-stable-version`,
	ShortDesc: `Get the stable version`,
	CommandRun: func() subcommands.CommandRun {
		r := &getStableVersionRun{}

		r.authFlags.Register(&r.Flags, site.DefaultAuthOptions)
		r.envFlags.Register(&r.Flags)
		r.commonFlags.Register(&r.Flags)

		// Key info.
		r.Flags.StringVar(&r.board, "board", "", "the board or build target")
		r.Flags.StringVar(&r.model, "model", "", "the model")
		r.Flags.StringVar(&r.hostname, "hostname", "", "the hostname")
		return r
	},
}

// GetStableVersionRun is the command for adminclient get-stable-version.
type getStableVersionRun struct {
	subcommands.CommandRunBase

	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags

	board    string
	model    string
	hostname string
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
	newHostname, err := preprocessHostname(c.commonFlags, c.hostname, nil, nil)
	if err != nil {
		return errors.Annotate(err, "get stable version").Err()
	}
	c.hostname = newHostname

	hc, err := cmdlib.NewHTTPClient(ctx, &c.authFlags)
	if err != nil {
		return errors.Annotate(err, "get stable version").Err()
	}

	invWithSVClient := fleet.NewInventoryPRPCClient(
		&prpc.Client{
			C:       hc,
			Host:    c.envFlags.GetCrosAdmService(),
			Options: site.DefaultPRPCOptions,
		},
	)
	resp, err := invWithSVClient.GetStableVersion(ctx, &fleet.GetStableVersionRequest{
		BuildTarget: c.board,
		Model:       c.model,
		Hostname:    c.hostname,
		// Mark ourselves as a satlab informational query so we always get the satlab versions.
		SatlabInformationalQuery: true,
	})
	if err != nil {
		return errors.Annotate(err, "get stable version").Err()
	}
	out, err := protojson.MarshalOptions{
		Indent: "  ",
	}.Marshal(resp)
	if err != nil {
		return errors.Annotate(err, "get stable version").Err()
	}
	fmt.Fprintf(a.GetOut(), "%s\n", out)
	return nil
}
