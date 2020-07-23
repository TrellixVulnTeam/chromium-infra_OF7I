// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package rack

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

// ListRackCmd list all Racks.
var ListRackCmd = &subcommands.Command{
	UsageLine: "rack",
	ShortDesc: "List all racks",
	LongDesc:  cmdhelp.ListRackLongDesc,
	CommandRun: func() subcommands.CommandRun {
		c := &listRack{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.Flags.IntVar(&c.pageSize, "n", 0, cmdhelp.ListPageSizeDesc)
		c.Flags.BoolVar(&c.json, "json", false, `print output in JSON format`)
		return c
	},
}

type listRack struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
	envFlags  site.EnvFlags
	pageSize  int
	json      bool
}

func (c *listRack) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *listRack) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
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
	if c.json {
		return utils.PrintListJSONFormat(ctx, ic, printRacks, c.json, int32(c.pageSize), "")
	}
	return utils.PrintListTableFormat(ctx, ic, printRacks, c.json, int32(c.pageSize), "", utils.RackTitle)
}

func printRacks(ctx context.Context, ic ufsAPI.FleetClient, json bool, pageSize int32, pageToken, filter string) (string, error) {
	req := &ufsAPI.ListRacksRequest{
		PageSize:  pageSize,
		PageToken: pageToken,
	}
	res, err := ic.ListRacks(ctx, req)
	if err != nil {
		return "", err
	}
	if json {
		utils.PrintRacksJSON(res.Racks)
	} else {
		utils.PrintRacks(res.Racks)
	}
	return res.GetNextPageToken(), nil
}
