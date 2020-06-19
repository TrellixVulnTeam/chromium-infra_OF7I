// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package machine

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
)

// ListMachineCmd list all Machine.
var ListMachineCmd = &subcommands.Command{
	UsageLine: "ls",
	ShortDesc: "list all Machines",
	LongDesc:  cmdhelp.ListMachineLongDesc,
	CommandRun: func() subcommands.CommandRun {
		c := &listMachine{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.Flags.IntVar(&c.pageSize, "n", 100, cmdhelp.ListPageSizeDesc)
		c.Flags.BoolVar(&c.json, "json", false, `print output in JSON format`)
		return c
	},
}

type listMachine struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
	envFlags  site.EnvFlags
	pageSize  int
	json      bool
}

func (c *listMachine) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}
func (c *listMachine) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
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
		req := &UfleetAPI.ListMachinesRequest{
			PageSize:  int32(c.pageSize),
			PageToken: pageToken,
		}
		res, err := ic.ListMachines(ctx, req)
		if err != nil {
			return err
		}
		if c.json {
			utils.PrintMachinesJSON(res.Machines)
		} else {
			utils.PrintMachines(res.Machines)
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
