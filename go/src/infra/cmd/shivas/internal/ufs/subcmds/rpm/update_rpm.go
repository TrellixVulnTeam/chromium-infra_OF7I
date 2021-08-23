// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package rpm

import (
	"fmt"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/grpc/prpc"

	"infra/cmd/shivas/cmdhelp"
	"infra/cmd/shivas/site"
	"infra/cmd/shivas/utils"
	"infra/cmdsupport/cmdlib"
	ufspb "infra/unifiedfleet/api/v1/models"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	ufsUtil "infra/unifiedfleet/app/util"
)

// UpdateRPMCmd Update rpm by given name.
var UpdateRPMCmd = &subcommands.Command{
	UsageLine: "rpm [Options...]",
	ShortDesc: "Update a rpm on a rack",
	LongDesc:  cmdhelp.UpdateRPMLongDesc,
	CommandRun: func() subcommands.CommandRun {
		c := &updateRPM{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.commonFlags.Register(&c.Flags)

		c.Flags.StringVar(&c.newSpecsFile, "f", "", cmdhelp.RPMFileText)

		c.Flags.StringVar(&c.vlanName, "vlan", "", "the vlan to assign the rpm to")
		c.Flags.BoolVar(&c.deleteVlan, "delete-vlan", false, "if deleting the ip assignment for the rpm")
		c.Flags.StringVar(&c.ip, "ip", "", "the ip to assign the rpm to")

		c.Flags.StringVar(&c.rackName, "rack", "", "name of the rack to associate the rpm.")
		c.Flags.StringVar(&c.rpmName, "name", "", "the name of the rpm to update")
		c.Flags.StringVar(&c.macAddress, "mac", "", "the mac address of the rpm to update"+cmdhelp.ClearFieldHelpText)
		c.Flags.StringVar(&c.description, "desc", "", "the description of the switch to update ."+cmdhelp.ClearFieldHelpText)
		c.Flags.IntVar(&c.capacity, "capacity", 0, "indicate how many ports this switch support. "+"To clear this field set it to -1.")
		c.Flags.StringVar(&c.tags, "tags", "", "comma separated tags. You can only append/add new tags here. "+cmdhelp.ClearFieldHelpText)
		c.Flags.StringVar(&c.state, "state", "", cmdhelp.StateHelp)

		return c
	},
}

type updateRPM struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags

	newSpecsFile string

	rackName    string
	vlanName    string
	rpmName     string
	deleteVlan  bool
	ip          string
	macAddress  string
	description string
	capacity    int
	tags        string
	state       string
}

func (c *updateRPM) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *updateRPM) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
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
	var rpm ufspb.RPM
	if c.newSpecsFile != "" {
		if err = utils.ParseJSONFile(c.newSpecsFile, &rpm); err != nil {
			return err
		}
		if rpm.GetRack() == "" {
			return errors.New(fmt.Sprintf("rack field is empty in json. It is a required parameter for json input."))
		}
	} else {
		c.parseArgs(&rpm)
	}
	if err := utils.PrintExistingRPM(ctx, ic, rpm.Name); err != nil {
		return err
	}
	rpm.Name = ufsUtil.AddPrefix(ufsUtil.RPMCollection, rpm.Name)
	if !ufsUtil.ValidateTags(rpm.Tags) {
		return fmt.Errorf(ufsAPI.InvalidTags)
	}
	res, err := ic.UpdateRPM(ctx, &ufsAPI.UpdateRPMRequest{
		RPM: &rpm,
		NetworkOption: &ufsAPI.NetworkOption{
			Vlan:   c.vlanName,
			Delete: c.deleteVlan,
			Ip:     c.ip,
		},
		UpdateMask: utils.GetUpdateMask(&c.Flags, map[string]string{
			"rack":     "rack",
			"mac":      "macAddress",
			"capacity": "capacity",
			"tags":     "tags",
			"desc":     "description",
			"state":    "resourceState",
		}),
	})
	if err != nil {
		return err
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	fmt.Println("The rpm after update:")
	utils.PrintProtoJSON(res, !utils.NoEmitMode(false))
	if c.deleteVlan {
		fmt.Printf("Successfully deleted vlan of rpm %s\n", res.Name)
	}
	if c.vlanName != "" {
		// Log the assigned IP
		if dhcp, err := ic.GetDHCPConfig(ctx, &ufsAPI.GetDHCPConfigRequest{
			Hostname: res.Name,
		}); err == nil {
			utils.PrintProtoJSON(dhcp, false)
			fmt.Println("Successfully added dhcp config to rpm: ", res.Name)
		}
	}
	return nil
}

func (c *updateRPM) parseArgs(rpm *ufspb.RPM) {
	rpm.Name = c.rpmName
	rpm.Rack = c.rackName
	rpm.ResourceState = ufsUtil.ToUFSState(c.state)
	if c.macAddress == utils.ClearFieldValue {
		rpm.MacAddress = ""
	} else {
		rpm.MacAddress = c.macAddress
	}
	if c.description == utils.ClearFieldValue {
		rpm.Description = ""
	} else {
		rpm.Description = c.description
	}
	if c.capacity == -1 {
		rpm.CapacityPort = 0
	} else {
		rpm.CapacityPort = int32(c.capacity)
	}
	if c.tags == utils.ClearFieldValue {
		rpm.Tags = nil
	} else {
		rpm.Tags = utils.GetStringSlice(c.tags)
	}
}

func (c *updateRPM) validateArgs() error {
	if c.newSpecsFile != "" {
		if c.rpmName != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON mode is specified. '-name' cannot be specified at the same time.")
		}
		if c.macAddress != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON mode is specified. '-mac' cannot be specified at the same time.")
		}
		if c.tags != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON mode is specified. '-tags' cannot be specified at the same time.")
		}
		if c.rackName != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON mode is specified. '-rack' cannot be specified at the same time.")
		}
		if c.state != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON mode is specified. '-state' cannot be specified at the same time.")
		}
		if c.capacity != 0 {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON mode is specified. '-capacity' cannot be specified at the same time.")
		}
		if c.description != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON mode is specified. '-desc' cannot be specified at the same time.")
		}
	}
	if c.newSpecsFile == "" {
		if c.rpmName == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n'-name' is required, no mode ('-f') is specified")
		}
		if c.vlanName == "" && !c.deleteVlan && c.ip == "" &&
			c.rackName == "" && c.macAddress == "" && c.tags == "" &&
			c.state == "" && c.capacity == 0 && c.description == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nNothing to update. Please provide any field to update")
		}
		if c.state != "" && !ufsUtil.IsUFSState(ufsUtil.RemoveStatePrefix(c.state)) {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n%s is not a valid state, please check help info for '-state'.", c.state)
		}
	}
	return nil
}
