// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package host

import (
	"context"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/grpc/prpc"

	"infra/cmd/shivas/cmdhelp"
	"infra/cmd/shivas/site"
	"infra/cmd/shivas/utils"
	"infra/cmdsupport/cmdlib"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
)

// ListHostCmd list all hosts.
var ListHostCmd = &subcommands.Command{
	UsageLine: "host",
	ShortDesc: "List all hosts",
	LongDesc:  cmdhelp.ListHostLongDesc,
	CommandRun: func() subcommands.CommandRun {
		c := &listHost{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.Flags.IntVar(&c.pageSize, "n", 0, cmdhelp.ListPageSizeDesc)
		return c
	},
}

type listHost struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
	envFlags  site.EnvFlags
	pageSize  int
}

func (c *listHost) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *listHost) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	ctx := cli.GetContext(a, c, env)
	hc, err := cmdlib.NewHTTPClient(ctx, &c.authFlags)
	if err != nil {
		return err
	}
	e := c.envFlags.Env()
	ic := ufsAPI.NewFleetPRPCClient(&prpc.Client{
		C:       hc,
		Host:    e.UnifiedFleetService,
		Options: site.DefaultPRPCOptions,
	})
	// MachineLSE has large number of fields. Print only JSON format always.
	return utils.PrintListJSONFormat(ctx, ic, printMachineLSEs, true, int32(c.pageSize), "")
}

func printMachineLSEs(ctx context.Context, ic ufsAPI.FleetClient, json bool, pageSize int32, pageToken, filter string) (string, error) {
	req := &ufsAPI.ListMachineLSEsRequest{
		PageSize:  pageSize,
		PageToken: pageToken,
	}
	res, err := ic.ListMachineLSEs(ctx, req)
	if err != nil {
		return "", err
	}
	utils.PrintMachineLSEsJSON(res.MachineLSEs)
	return res.GetNextPageToken(), nil
}
