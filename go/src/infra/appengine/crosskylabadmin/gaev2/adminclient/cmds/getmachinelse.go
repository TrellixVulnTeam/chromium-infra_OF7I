// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmds

import (
	"context"
	"fmt"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"

	"infra/appengine/crosskylabadmin/internal/ufs"
	"infra/appengine/crosskylabadmin/site"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
)

// GetMachineLSE calls the GetMachineLSE RPC of UFS the way that CrOSSkylabAdmin would.
var GetMachineLSE = &subcommands.Command{
	UsageLine: `get-machine-lse`,
	ShortDesc: `Get the stable version`,
	CommandRun: func() subcommands.CommandRun {
		r := &getMachineLSERun{}
		r.authFlags.Register(&r.Flags, site.DefaultAuthOptions)
		r.Flags.StringVar(&r.ufs, "ufs", "", "the UFS server")
		r.Flags.StringVar(&r.name, "name", "", "the device name")
		return r
	},
}

// GetMachineLSERun runs the get-machine-lse command.
type getMachineLSERun struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
	name      string
	ufs       string
}

// Run runs the command and returns an exit status.
func (c *getMachineLSERun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	ctx := cli.GetContext(a, c, env)
	if err := c.innerRun(ctx, a, args, env); err != nil {
		fmt.Fprintf(a.GetErr(), "%s: %s\n", a.GetName(), err)
		return 1
	}
	return 0
}

// InnerRun runs the command and returns an error.
func (c *getMachineLSERun) innerRun(ctx context.Context, a subcommands.Application, args []string, env subcommands.Env) error {
	authOptions, err := c.authFlags.Options()
	if err != nil {
		return errors.Annotate(err, "get machine lse: authenticating").Err()
	}
	hc, err := auth.NewAuthenticator(ctx, auth.InteractiveLogin, authOptions).Client()
	if err != nil {
		return errors.Annotate(err, "get machine lse").Err()
	}
	client, err := ufs.NewClient(ctx, hc, c.ufs)
	if err != nil {
		return errors.Annotate(err, "get machine lse: creating client").Err()
	}
	req := &ufsAPI.GetMachineLSERequest{
		Name: c.name,
	}
	jsonMarshaler.Marshal(a.GetErr(), req)
	res, err := client.GetMachineLSE(ctx, req)
	if err != nil {
		return errors.Annotate(err, "get machine lse: inner run").Err()
	}
	jsonMarshaler.Marshal(a.GetOut(), res)
	return nil
}
