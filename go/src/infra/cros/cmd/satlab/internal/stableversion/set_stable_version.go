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

// If allowSetModelBoard is true, then the user is allowed to create new entries for a host&model.
// If allowSetModelBoard is false, then the user is blocked from creating new entries a host&model.
//
// We set this variable to false. We want to force users to use the per-host override so that it is
// easier to replace the stable version implementation with a new service behind the scenes without
// a change in user-facing behavior.
const allowSetModelBoard = false

var SetStableVersionCmd = &subcommands.Command{
	UsageLine: `set-stable-version`,
	ShortDesc: `Set the stable version using {board, model} or {hostname}.`,
	CommandRun: func() subcommands.CommandRun {
		r := &setStableVersionRun{}

		r.authFlags.Register(&r.Flags, site.DefaultAuthOptions)
		r.envFlags.Register(&r.Flags)
		r.commonFlags.Register(&r.Flags)

		if allowSetModelBoard {
			r.Flags.StringVar(&r.board, "board", "", `the board or build target (used with model)`)
			r.Flags.StringVar(&r.model, "model", "", `the model (used with board)`)
		}
		r.Flags.StringVar(&r.hostname, "hostname", "", `the hostname (used by itself)`)
		r.Flags.StringVar(&r.os, "os", "", `the OS version to set (no change if empty)`)
		r.Flags.StringVar(&r.fw, "fw", "", `the firmware version to set (no change if empty)`)
		r.Flags.StringVar(&r.fwImage, "fwImage", "", `the firmware image version to set (no change if empty)`)

		return r
	},
}

// SetStableVersionRun is the command for adminclient set-stable-version.
type setStableVersionRun struct {
	subcommands.CommandRunBase

	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags

	board    string
	model    string
	hostname string
	os       string
	fw       string
	fwImage  string
}

// Run runs the command, prints the error if there is one, and returns an exit status.
func (c *setStableVersionRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	ctx := cli.GetContext(a, c, env)
	if err := c.innerRun(ctx, a, args, env); err != nil {
		fmt.Fprintf(a.GetErr(), "%s: %s\n", a.GetName(), err)
		return 1
	}
	return 0
}

// ProduceRequest creates a request that can be used as a key to set the stable version.
// If the command line arguments do not unambiguously indicate how to create such a request, we fail.
//
func (c *setStableVersionRun) produceRequest(ctx context.Context, a subcommands.Application, args []string, env subcommands.Env) (*fleet.SetSatlabStableVersionRequest, error) {
	req := &fleet.SetSatlabStableVersionRequest{}
	useHostnameStrategy := c.hostname != ""
	useBoardModelStrategy := allowSetModelBoard && (c.board != "") && (c.model != "")

	// Validate and populate the strategy field of the request.
	if err := func() error {
		if useHostnameStrategy {
			if useBoardModelStrategy {
				return errors.Reason("board and model should not be set if hostname is provided").Err()
			}
			req.Strategy = &fleet.SetSatlabStableVersionRequest_SatlabHostnameStrategy{
				SatlabHostnameStrategy: &fleet.SatlabHostnameStrategy{
					Hostname: c.hostname,
				},
			}
			return nil
		} // Hostname strategy not set.
		if !useBoardModelStrategy {
			return errors.Reason("must provide hostname").Err()
		}
		req.Strategy = &fleet.SetSatlabStableVersionRequest_SatlabBoardAndModelStrategy{
			SatlabBoardAndModelStrategy: &fleet.SatlabBoardAndModelStrategy{
				Board: c.board,
				Model: c.model,
			},
		}
		return nil
	}(); err != nil {
		return nil, err
	}

	// TODO(gregorynisbet): Consider adding validation here instead of on the server side
	req.CrosVersion = c.os
	req.FirmwareVersion = c.fw
	req.FirmwareImage = c.fwImage

	return req, nil
}

// InnerRun creates a client, sends a GetStableVersion request, and prints the response.
func (c *setStableVersionRun) innerRun(ctx context.Context, a subcommands.Application, args []string, env subcommands.Env) error {
	newHostname, err := preprocessHostname(c.commonFlags, c.hostname, nil, nil)
	if err != nil {
		return errors.Annotate(err, "set stable version").Err()
	}
	c.hostname = newHostname

	req, err := c.produceRequest(ctx, a, args, env)
	if err != nil {
		return errors.Annotate(err, "set stable version").Err()
	}

	hc, err := cmdlib.NewHTTPClient(ctx, &c.authFlags)
	if err != nil {
		return errors.Annotate(err, "set stable version").Err()
	}

	invWithSVClient := fleet.NewInventoryPRPCClient(
		&prpc.Client{
			C:       hc,
			Host:    c.envFlags.GetCrosAdmService(),
			Options: site.DefaultPRPCOptions,
		},
	)

	resp, err := invWithSVClient.SetSatlabStableVersion(ctx, req)
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
