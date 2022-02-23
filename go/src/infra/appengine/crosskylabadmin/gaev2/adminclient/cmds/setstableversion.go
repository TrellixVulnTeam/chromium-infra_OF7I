// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmds

import (
	"context"
	"fmt"
	fleet "infra/appengine/crosskylabadmin/api/fleet/v1"
	"infra/appengine/crosskylabadmin/site"
	"infra/cmdsupport/cmdlib"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/grpc/prpc"
	"google.golang.org/protobuf/encoding/protojson"
)

// SetStableVersion is a command for the GetStableVersion RPC.
// It intentionally performs no validation of its own and forwards requests to CSA as is.
var SetStableVersion = &subcommands.Command{
	UsageLine: `set-stable-version`,
	ShortDesc: `Set the stable version for satlab devices`,
	CommandRun: func() subcommands.CommandRun {
		r := &setStableVersionRun{}
		r.crOSAdminRPCRun.Register(&r.Flags)
		r.authFlags.Register(&r.Flags, site.DefaultAuthOptions)
		r.Flags.StringVar(&r.board, "board", "", "the board or build target")
		r.Flags.StringVar(&r.model, "model", "", "the model")
		r.Flags.StringVar(&r.hostname, "hostname", "", "the hostname")
		r.Flags.StringVar(&r.os, "os", "", "the OS version")
		r.Flags.StringVar(&r.fw, "fw", "", "the firmware version")
		r.Flags.StringVar(&r.fwImage, "fw-image", "", "the firmware image version")
		return r
	},
}

// SetStableVersionRun is the command for adminclient set-stable-version.
type setStableVersionRun struct {
	crOSAdminRPCRun
	authFlags authcli.Flags
	board     string
	model     string
	hostname  string
	os        string
	fw        string
	fwImage   string
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

// InnerRun creates a client, sends a GetStableVersion request, and prints the response.
func (c *setStableVersionRun) innerRun(ctx context.Context, a subcommands.Application, args []string, env subcommands.Env) error {
	hc, err := cmdlib.NewHTTPClient(ctx, &c.authFlags)
	if err != nil {
		return errors.Annotate(err, "set stable version").Err()
	}

	host, err := c.GetHost()
	if err != nil {
		return errors.Annotate(err, "set stable version").Err()
	}

	options, err := c.GetOptions()
	if err != nil {
		return errors.Annotate(err, "set stable version").Err()
	}

	invWithSVClient := fleet.NewInventoryPRPCClient(
		&prpc.Client{
			C:       hc,
			Host:    host,
			Options: options,
		},
	)

	req := &fleet.SetSatlabStableVersionRequest{}
	if c.hostname == "" {
		req.Strategy = &fleet.SetSatlabStableVersionRequest_SatlabHostnameStrategy{
			SatlabHostnameStrategy: &fleet.SatlabHostnameStrategy{
				Hostname: c.hostname,
			},
		}
	} else {
		req.Strategy = &fleet.SetSatlabStableVersionRequest_SatlabBoardAndModelStrategy{
			SatlabBoardAndModelStrategy: &fleet.SatlabBoardAndModelStrategy{
				Board: c.board,
				Model: c.model,
			},
		}
	}
	req.CrosVersion = c.os
	req.FirmwareVersion = c.fw
	req.FirmwareImage = c.fwImage

	resp, err := invWithSVClient.SetSatlabStableVersion(ctx, req)
	if err != nil {
		return errors.Annotate(err, "set stable version").Err()
	}
	out, err := protojson.Marshal(resp)
	if err != nil {
		return errors.Annotate(err, "set stable version").Err()
	}
	fmt.Fprintf(a.GetOut(), "%s\n", out)
	return nil
}
