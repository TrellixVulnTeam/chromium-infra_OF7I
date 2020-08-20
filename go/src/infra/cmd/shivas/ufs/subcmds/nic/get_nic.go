// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package nic

import (
	"context"
	"fmt"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/grpc/prpc"

	"infra/cmd/shivas/site"
	"infra/cmd/shivas/utils"
	"infra/cmdsupport/cmdlib"
	ufspb "infra/unifiedfleet/api/v1/proto"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	ufsUtil "infra/unifiedfleet/app/util"
)

// GetNicCmd get nic by given name.
var GetNicCmd = &subcommands.Command{
	UsageLine: "nic {Nic Name}",
	ShortDesc: "Get nic details by name",
	LongDesc: `Get nic details by name.

Example:

shivas get nic {Nic Name}
Gets the nic and prints the output in JSON format.`,
	CommandRun: func() subcommands.CommandRun {
		c := &getNic{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.commonFlags.Register(&c.Flags)
		c.outputFlags.Register(&c.Flags)
		return c
	},
}

type getNic struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags
	outputFlags site.OutputFlags
}

func (c *getNic) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *getNic) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
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
	if c.commonFlags.Verbose() {
		fmt.Printf("Using UnifiedFleet service %s\n", e.UnifiedFleetService)
	}
	ic := ufsAPI.NewFleetPRPCClient(&prpc.Client{
		C:       hc,
		Host:    e.UnifiedFleetService,
		Options: site.DefaultPRPCOptions,
	})
	res, err := ic.GetNic(ctx, &ufsAPI.GetNicRequest{
		Name: ufsUtil.AddPrefix(ufsUtil.NicCollection, args[0]),
	})
	if err != nil {
		return err
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	if utils.FullMode(c.outputFlags.Full()) {
		return c.printFull(ctx, ic, res)
	}
	return c.print(res)
}

func (c *getNic) printFull(ctx context.Context, ic ufsAPI.FleetClient, nic *ufspb.Nic) error {
	machine, _ := ic.GetMachine(ctx, &ufsAPI.GetMachineRequest{
		Name: ufsUtil.AddPrefix(ufsUtil.MachineCollection, nic.GetMachine()),
	})
	if machine != nil {
		machine.Name = ufsUtil.RemovePrefix(machine.Name)
	}
	res2, _ := ic.ListMachineLSEs(ctx, &ufsAPI.ListMachineLSEsRequest{
		Filter: ufsUtil.MachineFilterName + "=" + machine.Name,
	})
	var lse *ufspb.MachineLSE
	if res2 != nil && len(res2.GetMachineLSEs()) > 0 {
		if res2.GetMachineLSEs()[0].GetNic() == nic.GetName() {
			lse = res2.GetMachineLSEs()[0]
			lse.Name = ufsUtil.RemovePrefix(lse.Name)
		}
	}
	var dhcp *ufspb.DHCPConfig
	if lse != nil {
		dhcp, _ = ic.GetDHCPConfig(ctx, &ufsAPI.GetDHCPConfigRequest{
			Hostname: lse.GetName(),
		})
	}
	// JSON mode is disabled for full mode for now
	if c.outputFlags.Tsv() {
		utils.PrintTSVNicFull(nic, dhcp)
		return nil
	}
	utils.PrintTitle(utils.NicFullTitle)
	utils.PrintNicFull(nic, dhcp)
	return nil
}

func (c *getNic) print(nic *ufspb.Nic) error {
	if c.outputFlags.JSON() {
		utils.PrintProtoJSON(nic, c.outputFlags.Emit())
	} else {
		if c.outputFlags.Tsv() {
			utils.PrintTSVNics([]*ufspb.Nic{nic}, false)
		} else {
			utils.PrintTitle(utils.NicTitle)
			utils.PrintNics([]*ufspb.Nic{nic}, false)
		}
	}
	return nil
}

func (c *getNic) validateArgs() error {
	if c.Flags.NArg() == 0 {
		return cmdlib.NewUsageError(c.Flags, "Please provide the nic name.")
	}
	return nil
}
