// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package drac

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

// GetDracCmd get drac by given name.
var GetDracCmd = &subcommands.Command{
	UsageLine: "drac {Drac Name}",
	ShortDesc: "Get drac details by name",
	LongDesc: `Get drac details by name.

Example:

shivas get drac {Drac Name}
Gets the drac and prints the output in JSON format.`,
	CommandRun: func() subcommands.CommandRun {
		c := &getDrac{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.commonFlags.Register(&c.Flags)
		c.outputFlags.Register(&c.Flags)
		return c
	},
}

type getDrac struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags
	outputFlags site.OutputFlags
}

func (c *getDrac) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *getDrac) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
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
	res, err := ic.GetDrac(ctx, &ufsAPI.GetDracRequest{
		Name: ufsUtil.AddPrefix(ufsUtil.DracCollection, args[0]),
	})
	if err != nil {
		return err
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	if c.outputFlags.Full() {
		return c.printFull(ctx, ic, res)
	}
	return c.print(res)
}

func (c *getDrac) printFull(ctx context.Context, ic ufsAPI.FleetClient, drac *ufspb.Drac) error {
	machine, _ := ic.GetMachine(ctx, &ufsAPI.GetMachineRequest{
		Name: ufsUtil.AddPrefix(ufsUtil.MachineCollection, drac.GetMachine()),
	})
	if machine != nil {
		machine.Name = ufsUtil.RemovePrefix(machine.Name)
	}
	dhcp, _ := ic.GetDHCPConfig(ctx, &ufsAPI.GetDHCPConfigRequest{
		Hostname: drac.Name,
	})
	// JSON mode is disabled for full mode for now
	if !c.outputFlags.Tsv() {
		utils.PrintTitle(utils.DracFullTitle)
	}
	utils.PrintDracFull(drac, machine, dhcp)
	return nil
}

func (c *getDrac) print(drac *ufspb.Drac) error {
	if c.outputFlags.JSON() {
		utils.PrintProtoJSON(drac)
	} else {
		if c.outputFlags.Tsv() {
			utils.PrintTSVDracs([]*ufspb.Drac{drac}, false)
		} else {
			utils.PrintTitle(utils.DracTitle)
			utils.PrintDracs([]*ufspb.Drac{drac}, false)
		}
	}
	return nil
}

func (c *getDrac) getFull(ctx context.Context, ic ufsAPI.FleetClient, name string) (*ufspb.Drac, *ufspb.DHCPConfig, error) {
	res, err := ic.GetDrac(ctx, &ufsAPI.GetDracRequest{
		Name: ufsUtil.AddPrefix(ufsUtil.DracCollection, name),
	})
	if err != nil {
		return nil, nil, err
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	dhcp, err := ic.GetDHCPConfig(ctx, &ufsAPI.GetDHCPConfigRequest{
		Hostname: res.Name,
	})
	if ufsUtil.IsNotFoundError(err) || dhcp == nil {
		return res, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	return res, dhcp, nil
}

func (c *getDrac) validateArgs() error {
	if c.Flags.NArg() == 0 {
		return cmdlib.NewUsageError(c.Flags, "Please provide the drac name.")
	}
	return nil
}
