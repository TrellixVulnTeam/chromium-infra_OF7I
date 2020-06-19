// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package machinelseprototype

import (
	"fmt"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/grpc/prpc"
	"infra/cmd/shivas/cmdhelp"
	"infra/cmd/shivas/site"
	"infra/cmd/shivas/utils"
	"infra/cmdsupport/cmdlib"
	UfleetAPI "infra/unifiedfleet/api/v1/rpc"
	Ufleetds "infra/unifiedfleet/app/model/datastore"
)

// ListMachinelsePrototypeCmd list all MachineLSEPrototype.
var ListMachinelsePrototypeCmd = &subcommands.Command{
	UsageLine: "ls",
	ShortDesc: "list all MachineLSEPrototypes",
	LongDesc:  cmdhelp.ListMachinelsePrototypeLongDesc,
	CommandRun: func() subcommands.CommandRun {
		c := &listMachinelsePrototype{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.Flags.IntVar(&c.pageSize, "n", 100, cmdhelp.ListPageSizeDesc)
		c.Flags.BoolVar(&c.json, "json", false, `print output in JSON format`)
		c.Flags.StringVar(&c.labFilter, "lab", "", "lab name to filter the results.\n"+
			"acs for ACS lab MachineLSEPrototypes\n"+
			"atl for ATL lab MachineLSEPrototypes\n"+
			"browser for Browser lab MachineLSEPrototytpes")
		return c
	},
}

type listMachinelsePrototype struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
	envFlags  site.EnvFlags
	pageSize  int
	json      bool
	labFilter string
}

func (c *listMachinelsePrototype) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *listMachinelsePrototype) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	if err := c.validateArgs(); err != nil {
		return err
	}
	ctx := cli.GetContext(a, c, env)
	ctx = utils.SetupContext(ctx)
	hc, err := cmdlib.NewHTTPClient(ctx, &c.authFlags)
	if err != nil {
		return err
	}
	e := c.envFlags.Env()
	fmt.Printf("Using UnifiedFleet service %s\n", e.UnifiedFleetService)
	ic := UfleetAPI.NewFleetPRPCClient(&prpc.Client{
		C:       hc,
		Host:    e.UnifiedFleetService,
		Options: site.DefaultPRPCOptions,
	})
	var pageToken string
	for {
		req := &UfleetAPI.ListMachineLSEPrototypesRequest{
			PageSize:  int32(c.pageSize),
			PageToken: pageToken,
			Filter:    Ufleetds.Lab + Ufleetds.FilterConditionSeparator + c.labFilter,
		}
		res, err := ic.ListMachineLSEPrototypes(ctx, req)
		if err != nil {
			return err
		}
		if c.json {
			utils.PrintMachineLSEPrototypesJSON(res.MachineLSEPrototypes)
		} else {
			utils.PrintMachineLSEPrototypes(res.MachineLSEPrototypes)
		}
		pageToken = res.GetNextPageToken()
		nextPage, err := utils.GetNextPage(pageToken)
		if err != nil {
			return err
		}
		if !nextPage {
			return nil
		}
	}
}

func (c *listMachinelsePrototype) validateArgs() error {
	if c.labFilter != "" {
		if c.labFilter != Ufleetds.ATL &&
			c.labFilter != Ufleetds.ACS &&
			c.labFilter != Ufleetds.Browser {
			return cmdlib.NewUsageError(c.Flags, "Please provide a correct filter\n"+
				"acs for ACS lab MachineLSEPrototypes\n"+
				"atl for ATL lab MachineLSEPrototypes\n"+
				"browser for Browser lab MachineLSEPrototytpes")
		}
	}
	return nil
}
