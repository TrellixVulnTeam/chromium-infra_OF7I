// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package rack

import (
	"fmt"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/flag"
	"go.chromium.org/luci/grpc/prpc"

	"infra/cmd/shivas/cmdhelp"
	"infra/cmd/shivas/site"
	"infra/cmd/shivas/utils"
	"infra/cmdsupport/cmdlib"
	ufspb "infra/unifiedfleet/api/v1/models"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	ufsUtil "infra/unifiedfleet/app/util"
)

// UpdateRackCmd update Rack by given name.
var UpdateRackCmd = &subcommands.Command{
	UsageLine: "rack [Options...]",
	ShortDesc: "Update a rack",
	LongDesc:  cmdhelp.UpdateRackLongDesc,
	CommandRun: func() subcommands.CommandRun {
		c := &updateRack{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.commonFlags.Register(&c.Flags)

		c.Flags.StringVar(&c.newSpecsFile, "f", "", cmdhelp.RackFileText)

		c.Flags.StringVar(&c.rackName, "name", "", "the name of the rack to update")
		c.Flags.StringVar(&c.zoneName, "zone", "", cmdhelp.ZoneHelpText)
		c.Flags.IntVar(&c.bbNum, "bbnum", 0, "indicate the breadboard number of the rack. To clear this field set it to -1.")
		c.Flags.IntVar(&c.capacity, "capacity_ru", 0, "indicate the size of the rack in rack units (U). "+"To clear this field set it to -1.")
		c.Flags.Var(flag.StringSlice(&c.tags), "tag", "Name(s) of tag(s). Can be specified multiple times. "+cmdhelp.ClearFieldHelpText)
		c.Flags.StringVar(&c.state, "state", "", cmdhelp.StateHelp)
		return c
	},
}

type updateRack struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags

	newSpecsFile string

	rackName string
	zoneName string
	capacity int
	tags     []string
	bbNum    int
	state    string
}

func (c *updateRack) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *updateRack) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
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
	var rack ufspb.Rack
	if c.newSpecsFile != "" {
		if err = utils.ParseJSONFile(c.newSpecsFile, &rack); err != nil {
			return err
		}
		rack.Realm = ufsUtil.ToUFSRealm(rack.GetLocation().GetZone().String())
	} else {
		c.parseArgs(&rack)
	}
	if err := utils.PrintExistingRack(ctx, ic, rack.Name); err != nil {
		return err
	}
	rack.Name = ufsUtil.AddPrefix(ufsUtil.RackCollection, rack.Name)
	if !ufsUtil.ValidateTags(rack.Tags) {
		return fmt.Errorf(ufsAPI.InvalidTags)
	}
	res, err := ic.UpdateRack(ctx, &ufsAPI.UpdateRackRequest{
		Rack: &rack,
		UpdateMask: utils.GetUpdateMask(&c.Flags, map[string]string{
			"zone":        "zone",
			"capacity_ru": "capacity",
			"tag":         "tags",
			"bbnum":       "bbnum",
			"state":       "resourceState",
		}),
	})
	if err != nil {
		return err
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	fmt.Println("The rack after update:")
	utils.PrintProtoJSON(res, !utils.NoEmitMode(false))
	fmt.Println("Successfully updated the rack: ", res.Name)
	return nil
}

func (c *updateRack) parseArgs(rack *ufspb.Rack) {
	rack.Name = c.rackName
	rack.ResourceState = ufsUtil.ToUFSState(c.state)
	rack.Location = &ufspb.Location{}
	if c.zoneName == utils.ClearFieldValue {
		rack.GetLocation().Zone = ufsUtil.ToUFSZone("")
	} else {
		rack.GetLocation().Zone = ufsUtil.ToUFSZone(c.zoneName)
	}
	if ufsUtil.ContainsAnyStrings(c.tags, utils.ClearFieldValue) {
		rack.Tags = nil
	} else {
		rack.Tags = c.tags
	}
	if c.bbNum == -1 {
		rack.Bbnum = 0
	} else {
		rack.Bbnum = int32(c.bbNum)
	}
	if c.capacity == -1 {
		rack.CapacityRu = 0
	} else {
		rack.CapacityRu = int32(c.capacity)
	}
	rack.Realm = ufsUtil.ToUFSRealm(rack.GetLocation().GetZone().String())
}

func (c *updateRack) validateArgs() error {
	if c.newSpecsFile != "" {
		if c.rackName != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON input file is already specified. '-name' cannot be specified at the same time.")
		}
		if c.zoneName != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON input file is already specified. '-zone' cannot be specified at the same time.")
		}
		if c.capacity != 0 {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON input file is already specified. '-capacity_ru' cannot be specified at the same time.")
		}
		if c.bbNum != 0 {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON input file is already specified. '-bbnum' cannot be specified at the same time.")
		}
		if len(c.tags) > 0 {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe interactive/JSON mode is specified. '-tag' cannot be specified at the same time.")
		}
		if c.state != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON input file is already specified. '-state' cannot be specified at the same time.")
		}
	} else {
		if c.rackName == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n'-name' is required, no mode ('-f') is specified.")
		}
		if c.zoneName == "" && c.capacity == 0 && c.bbNum == 0 && len(c.tags) == 0 && c.state == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nNothing to update. Please provide any field to update")
		}
		if c.zoneName != "" && !ufsUtil.IsUFSZone(ufsUtil.RemoveZonePrefix(c.zoneName)) {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n%s is not a valid zone name, please check help info for '-zone'.", c.zoneName)
		}
		if c.state != "" && !ufsUtil.IsUFSState(ufsUtil.RemoveStatePrefix(c.state)) {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n%s is not a valid state, please check help info for '-state'.", c.state)
		}
	}
	return nil
}
