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

	"infra/cmd/shivas/cmdhelp"
	"infra/cmd/shivas/site"
	"infra/cmd/shivas/utils"
	"infra/cmdsupport/cmdlib"
	ufspb "infra/unifiedfleet/api/v1/models"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	ufsUtil "infra/unifiedfleet/app/util"
)

// UpdateVlanCmd update Vlan by given name.
var UpdateVlanCmd = &subcommands.Command{
	UsageLine: "vlan",
	ShortDesc: "Update vlan configuration for browser machine",
	LongDesc:  cmdhelp.UpdateVlanLongDesc,
	CommandRun: func() subcommands.CommandRun {
		c := &updateVlan{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.commonFlags.Register(&c.Flags)

		c.Flags.StringVar(&c.name, "name", "", "name of the vlan")
		c.Flags.StringVar(&c.description, "desc", "", "description for the vlan. "+cmdhelp.ClearFieldHelpText)
		c.Flags.StringVar(&c.state, "state", "", cmdhelp.StateHelp)
		c.Flags.StringVar(&c.zones, "zone", "", "comma separated zones, You can only append/add new zones here or clean it out."+cmdhelp.ZoneHelpText+cmdhelp.ClearFieldHelpText)
		c.Flags.StringVar(&c.reservedIPs, "reserved_ips", "", "comma separated ips. You can only append/add new ips here. "+cmdhelp.ClearFieldHelpText)
		c.Flags.StringVar(&c.freeStartIP, "start-ip", "", "the start IPv4 string of the vlan's free DHCP range. "+cmdhelp.ClearFieldHelpText)
		c.Flags.StringVar(&c.freeEndIP, "end-ip", "", "the end IPv4 string of the vlan's free DHCP range. "+cmdhelp.ClearFieldHelpText)
		c.Flags.StringVar(&c.tags, "tags", "", "comma separated tags. You can only append/add new tags here. "+cmdhelp.ClearFieldHelpText)
		return c
	},
}

type updateVlan struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags

	name        string
	description string
	state       string
	reservedIPs string
	zones       string
	freeStartIP string
	freeEndIP   string
	tags        string
}

func (c *updateVlan) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *updateVlan) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	if err := c.validateArgs(); err != nil {
		return err
	}
	ctx := cli.GetContext(a, c, env)
	ns, err := c.envFlags.Namespace()
	if err != nil {
		return err
	}
	ctx = utils.SetupContext(ctx, ns)
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
	if !ufsUtil.ValidateTags(vlan.Tags) {
		return fmt.Errorf(ufsAPI.InvalidTags)
	}
	if err := utils.PrintExistingVlan(ctx, ic, vlan.Name); err != nil {
		return err
	}
	vlan.Name = ufsUtil.AddPrefix(ufsUtil.VlanCollection, vlan.Name)
	res, err := ic.UpdateVlan(ctx, &ufsAPI.UpdateVlanRequest{
		Vlan: &vlan,
		UpdateMask: utils.GetUpdateMask(&c.Flags, map[string]string{
			"desc":         "description",
			"state":        "resourceState",
			"reserved_ips": "reserved_ips",
			"zone":         "zones",
			"start-ip":     "free_start_ip",
			"end-ip":       "free_end_ip",
			"tags":         "tags",
		}),
	})
	if err != nil {
		return err
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	fmt.Println("The vlan after update:")
	utils.PrintProtoJSON(res, !utils.NoEmitMode(false))
	fmt.Printf("Successfully updated the vlan %s\n", res.Name)
	return nil
}

func (c *updateVlan) parseArgs(vlan *ufspb.Vlan) {
	vlan.Name = c.name
	vlan.ResourceState = ufsUtil.ToUFSState(c.state)
	if c.description == utils.ClearFieldValue {
		vlan.Description = ""
	} else {
		vlan.Description = c.description
	}
	if c.reservedIPs == utils.ClearFieldValue {
		vlan.ReservedIps = nil
	} else {
		vlan.ReservedIps = utils.GetStringSlice(c.reservedIPs)
	}
	if c.zones == utils.ClearFieldValue {
		vlan.Zones = nil
	} else {
		zones := utils.GetStringSlice(c.zones)
		ufsZones := make([]ufspb.Zone, len(zones))
		for i, z := range zones {
			ufsZones[i] = ufsUtil.ToUFSZone(z)
		}
		vlan.Zones = ufsZones
	}
	if c.freeStartIP == utils.ClearFieldValue {
		vlan.FreeStartIpv4Str = ""
	} else {
		vlan.FreeStartIpv4Str = c.freeStartIP
	}
	if c.freeEndIP == utils.ClearFieldValue {
		vlan.FreeEndIpv4Str = ""
	} else {
		vlan.FreeEndIpv4Str = c.freeEndIP
	}
	if c.tags == utils.ClearFieldValue {
		vlan.Tags = nil
	} else {
		vlan.Tags = utils.GetStringSlice(c.tags)
	}
}

func (c *updateVlan) validateArgs() error {
	if c.name == "" {
		return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n'-name' is required, no mode ('-f' or '-i') is specified.")
	}
	if c.state == "" && c.description == "" && c.reservedIPs == "" && c.zones == "" && c.freeEndIP == "" && c.freeStartIP == "" && c.tags == "" {
		return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nNothing to update. Please provide any field to update")
	}
	if c.state != "" && !ufsUtil.IsUFSState(ufsUtil.RemoveStatePrefix(c.state)) {
		return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n%s is not a valid state, please check help info for '-state'.", c.state)
	}
	if c.zones != "" && c.zones != utils.ClearFieldValue {
		zones := utils.GetStringSlice(c.zones)
		for _, zone := range zones {
			if !ufsUtil.IsUFSZone(ufsUtil.RemoveZonePrefix(zone)) {
				cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n%s is not a valid zone name, please check help info for '-zone'.", zone)
			}
		}
	}
	return nil
}
