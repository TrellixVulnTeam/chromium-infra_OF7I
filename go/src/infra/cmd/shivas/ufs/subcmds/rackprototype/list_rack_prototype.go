// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package rackprototype

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
	ufsUtil "infra/unifiedfleet/app/util"
)

// ListRackLSEPrototypeCmd list all RackLSEPrototypes.
var ListRackLSEPrototypeCmd = &subcommands.Command{
	UsageLine: "rack-prototype [Filters...]",
	ShortDesc: "List all rack prototypes",
	LongDesc:  cmdhelp.ListRackLSEPrototypeLongDesc,
	CommandRun: func() subcommands.CommandRun {
		c := &listRackLSEPrototype{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.Flags.IntVar(&c.pageSize, "n", 0, cmdhelp.ListPageSizeDesc)
		c.Flags.BoolVar(&c.json, "json", false, `print output in JSON format`)
		c.Flags.StringVar(&c.labFilter, "lab", "", "lab name to filter the results.\n"+
			"acs - ACS lab rack prototypes\n"+
			"atl - ATL lab rack prototypes\n"+
			"browser - Browser lab rack prototypes")
		return c
	},
}

type listRackLSEPrototype struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
	envFlags  site.EnvFlags
	pageSize  int
	json      bool
	labFilter string
}

func (c *listRackLSEPrototype) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *listRackLSEPrototype) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	if err := c.validateArgs(); err != nil {
		return err
	}
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
		return utils.PrintListJSONFormat(ctx, ic, printRackLSEPrototypes, c.json, int32(c.pageSize), ufsUtil.FormatLabFilter(c.labFilter))
	}
	return utils.PrintListTableFormat(ctx, ic, printRackLSEPrototypes, c.json, int32(c.pageSize), ufsUtil.FormatLabFilter(c.labFilter), utils.RacklseprototypeTitle)
}

func printRackLSEPrototypes(ctx context.Context, ic ufsAPI.FleetClient, json bool, pageSize int32, pageToken, filter string) (string, error) {
	req := &ufsAPI.ListRackLSEPrototypesRequest{
		PageSize:  pageSize,
		PageToken: pageToken,
		Filter:    filter,
	}
	res, err := ic.ListRackLSEPrototypes(ctx, req)
	if err != nil {
		return "", err
	}
	if json {
		utils.PrintRackLSEPrototypesJSON(res.RackLSEPrototypes)
	} else {
		utils.PrintRackLSEPrototypes(res.RackLSEPrototypes)
	}
	return res.GetNextPageToken(), nil
}

func (c *listRackLSEPrototype) validateArgs() error {
	if c.labFilter != "" && !ufsUtil.IsValidFilter(c.labFilter) {
		return cmdlib.NewUsageError(c.Flags, "Please provide a correct filter\n"+
			"acs - ACS lab rack prototypes\n"+
			"atl - ATL lab rack prototypes\n"+
			"browser - Browser lab rack prototytpes")

	}
	return nil
}
