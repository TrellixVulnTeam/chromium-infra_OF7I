// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package vlan

import (
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

// GetVlanCmd get vlan by given name.
var GetVlanCmd = &subcommands.Command{
	UsageLine: "vlan {Vlan Name}",
	ShortDesc: "Get vlan details by name",
	LongDesc: `Get vlan details by name.

Example:

shivas get vlan {Vlan Name}
Gets the vlan and prints the output.`,
	CommandRun: func() subcommands.CommandRun {
		c := &getVlan{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.commonFlags.Register(&c.Flags)
		c.outputFlags.Register(&c.Flags)
		return c
	},
}

type getVlan struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags
	outputFlags site.OutputFlags
}

func (c *getVlan) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *getVlan) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
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
	res, err := ic.GetVlan(ctx, &ufsAPI.GetVlanRequest{
		Name: ufsUtil.AddPrefix(ufsUtil.VlanCollection, args[0]),
	})
	if err != nil {
		return err
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	return c.print(res)
}

func (c *getVlan) print(vlan *ufspb.Vlan) error {
	if c.outputFlags.JSON() {
		utils.PrintProtoJSON(vlan, !utils.NoEmitMode(c.outputFlags.NoEmit()))
		return nil
	}
	if c.outputFlags.Tsv() {
		utils.PrintTSVVlans([]*ufspb.Vlan{vlan}, false)
		return nil
	}
	utils.PrintTitle(utils.VlanTitle)
	utils.PrintVlans([]*ufspb.Vlan{vlan}, false)
	return nil
}

func (c *getVlan) validateArgs() error {
	if c.Flags.NArg() == 0 {
		return cmdlib.NewUsageError(c.Flags, "Please provide the vlan name.")
	}
	return nil
}
