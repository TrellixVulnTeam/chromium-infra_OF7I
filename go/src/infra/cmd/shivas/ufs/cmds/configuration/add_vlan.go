// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package configuration

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
	ufspb "infra/unifiedfleet/api/v1/proto"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	ufsUtil "infra/unifiedfleet/app/util"
)

// AddVlanCmd add a vlan to the system.
var AddVlanCmd = &subcommands.Command{
	UsageLine: "add-vlan",
	ShortDesc: "Add vlan configuration for browser machine",
	LongDesc:  cmdhelp.AddVlanLongDesc,
	CommandRun: func() subcommands.CommandRun {
		c := &addVlan{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.commonFlags.Register(&c.Flags)

		c.Flags.StringVar(&c.name, "name", "", "name of the vlan")
		c.Flags.StringVar(&c.cidrBlock, "cidr-block", "", "the cidr block of the vlan")
		c.Flags.StringVar(&c.description, "desc", "", "description for the vlan")
		return c
	},
}

type addVlan struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags

	name        string
	cidrBlock   string
	description string
}

func (c *addVlan) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *addVlan) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	if err := c.validateArgs(); err != nil {
		return err
	}
	ctx := cli.GetContext(a, c, env)
	hc, err := cmdlib.NewHTTPClient(ctx, &c.authFlags)
	if err != nil {
		return err
	}
	e := c.envFlags.Env()
	if c.commonFlags.Verbose() {
		fmt.Printf("Using UFS service %s\n", e.UnifiedFleetService)
	}
	ic := ufsAPI.NewFleetPRPCClient(&prpc.Client{
		C:       hc,
		Host:    e.UnifiedFleetService,
		Options: site.DefaultPRPCOptions,
	})
	var vlan ufspb.Vlan
	c.parseArgs(&vlan)
	res, err := ic.CreateVlan(ctx, &ufsAPI.CreateVlanRequest{
		Vlan:   &vlan,
		VlanId: vlan.GetName(),
	})
	if err != nil {
		return err
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	utils.PrintProtoJSON(res)
	fmt.Printf("Successfully added the vlan %s\n", res.Name)
	return nil
}

func (c *addVlan) parseArgs(vlan *ufspb.Vlan) {
	vlan.Name = c.name
	vlan.VlanAddress = c.cidrBlock
	vlan.Description = c.description
}

func (c *addVlan) validateArgs() error {
	if c.name == "" {
		return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n'-name' is required.")
	}
	if c.cidrBlock == "" {
		return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n'-cidr-block' is required.")
	}
	return nil
}
