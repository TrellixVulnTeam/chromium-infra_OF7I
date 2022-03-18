// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package attacheddevicemachine

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

// UpdateAttachedDeviceMachineCmd updates the attached device machine for a given name.
var UpdateAttachedDeviceMachineCmd = &subcommands.Command{
	UsageLine:  "attached-device-machine ...",
	ShortDesc:  "Update attached device machine details by filters",
	LongDesc:   cmdhelp.UpdateADMText,
	CommandRun: updateADMCommandRun,
}

// UpdateADMCmd is an alias to UpdateAttachedDeviceMachineCmd
var UpdateADMCmd = &subcommands.Command{
	UsageLine:  "adm ...",
	ShortDesc:  "Update attached device machine details by filters",
	LongDesc:   cmdhelp.UpdateADMText,
	CommandRun: updateADMCommandRun,
}

func updateADMCommandRun() subcommands.CommandRun {
	c := &updateAttachedDeviceMachine{}
	c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
	c.envFlags.Register(&c.Flags)
	c.commonFlags.Register(&c.Flags)

	c.Flags.StringVar(&c.newSpecsFile, "f", "", cmdhelp.ADMFileText)

	c.Flags.StringVar(&c.machineName, "name", "", "The name of the attached device machine to add.")
	c.Flags.StringVar(&c.zoneName, "zone", "", cmdhelp.ZoneHelpText)
	c.Flags.StringVar(&c.rackName, "rack", "", "The rack to add the attached device machine to. "+cmdhelp.ClearFieldHelpText)
	c.Flags.StringVar(&c.serialNumber, "serial", "", "The serial number for this attached device machine.")
	c.Flags.StringVar(&c.manufacturer, "man", "", "The manufacturer for this attached device machine. "+cmdhelp.ClearFieldHelpText)
	c.Flags.StringVar(&c.deviceType, "devicetype", "", "The device type for this attached device machine. "+cmdhelp.AttachedDeviceTypeHelpText)
	c.Flags.StringVar(&c.buildTarget, "build-target", "", "The build target for this attached device machine. "+cmdhelp.ClearFieldHelpText)
	c.Flags.StringVar(&c.model, "model", "", "The model for this attached device machine. "+cmdhelp.ClearFieldHelpText)
	c.Flags.StringVar(&c.state, "state", "", cmdhelp.StateHelp)
	c.Flags.Var(flag.StringSlice(&c.tags), "tag", "Name(s) of tag(s). Can be specified multiple times. "+cmdhelp.ClearFieldHelpText)
	return c
}

type updateAttachedDeviceMachine struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags
	outputFlags site.OutputFlags

	newSpecsFile string

	machineName  string
	zoneName     string
	rackName     string
	tags         []string
	serialNumber string
	manufacturer string
	deviceType   string
	buildTarget  string
	model        string
	state        string
}

func (c *updateAttachedDeviceMachine) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *updateAttachedDeviceMachine) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
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

	var machine ufspb.Machine
	if c.newSpecsFile != "" {
		if err = utils.ParseJSONFile(c.newSpecsFile, &machine); err != nil {
			return err
		}
		machine.Realm = ufsUtil.ToUFSRealm(machine.GetLocation().GetZone().String())
	} else {
		c.parseArgs(&machine)
	}
	_, err = utils.PrintExistingAttachedDeviceMachine(ctx, ic, machine.Name)
	if err != nil {
		return err
	}

	machine.Name = ufsUtil.AddPrefix(ufsUtil.MachineCollection, machine.Name)
	if !ufsUtil.ValidateTags(machine.Tags) {
		return fmt.Errorf(ufsAPI.InvalidTags)
	}
	res, err := ic.UpdateMachine(ctx, &ufsAPI.UpdateMachineRequest{
		Machine: &machine,
		UpdateMask: utils.GetUpdateMask(&c.Flags, map[string]string{
			"zone":         "zone",
			"rack":         "rack",
			"tag":          "tags",
			"serial":       "serialNumber",
			"state":        "resourceState",
			"man":          "admManufacturer",
			"devicetype":   "admDeviceType",
			"build-target": "admBuildTarget",
			"model":        "admModel",
		}),
	})
	if err != nil {
		return err
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	fmt.Println("The attached device machine after update:")
	utils.PrintProtoJSON(res, !utils.NoEmitMode(false))
	fmt.Println("Successfully updated the attached device machine: ", res.Name)
	return nil
}

func (c *updateAttachedDeviceMachine) parseArgs(machine *ufspb.Machine) {
	machine.Device = &ufspb.Machine_AttachedDevice{
		AttachedDevice: &ufspb.AttachedDevice{},
	}
	machine.Name = c.machineName
	machine.Location = &ufspb.Location{}
	if c.zoneName == utils.ClearFieldValue {
		machine.GetLocation().Zone = ufsUtil.ToUFSZone("")
	} else {
		machine.GetLocation().Zone = ufsUtil.ToUFSZone(c.zoneName)
	}
	if c.rackName == utils.ClearFieldValue {
		machine.GetLocation().Rack = ""
	} else {
		machine.GetLocation().Rack = c.rackName
	}
	if ufsUtil.ContainsAnyStrings(c.tags, utils.ClearFieldValue) {
		machine.Tags = nil
	} else {
		machine.Tags = c.tags
	}
	if c.serialNumber == utils.ClearFieldValue {
		machine.SerialNumber = ""
	} else {
		machine.SerialNumber = c.serialNumber
	}
	machine.ResourceState = ufsUtil.ToUFSState(c.state)

	// Attached Device Machine specific masks
	if c.manufacturer != "" || c.deviceType != "" || c.buildTarget != "" || c.model != "" {
		machine.GetAttachedDevice().DeviceType = ufsUtil.ToUFSAttachedDeviceType(c.deviceType)
		if c.manufacturer == utils.ClearFieldValue {
			machine.GetAttachedDevice().Manufacturer = ""
		} else {
			machine.GetAttachedDevice().Manufacturer = c.manufacturer
		}
		if c.buildTarget == utils.ClearFieldValue {
			machine.GetAttachedDevice().BuildTarget = ""
		} else {
			machine.GetAttachedDevice().BuildTarget = c.buildTarget
		}
		if c.model == utils.ClearFieldValue {
			machine.GetAttachedDevice().Model = ""
		} else {
			machine.GetAttachedDevice().Model = c.model
		}
	}
	machine.Realm = ufsUtil.ToUFSRealm(machine.GetLocation().GetZone().String())
}

func (c *updateAttachedDeviceMachine) validateArgs() error {
	if c.newSpecsFile != "" {
		if c.machineName != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON input file is already specified. '-name' cannot be specified at the same time.")
		}
		if c.rackName != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON input file is already specified. '-rack' cannot be specified at the same time.")
		}
		if c.zoneName != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON input file is already specified. '-zone' cannot be specified at the same time.")
		}
		if c.serialNumber != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON mode is specified. '-serial' cannot be specified at the same time.")
		}
		if c.manufacturer != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON mode is specified. '-man' cannot be specified at the same time.")
		}
		if c.deviceType != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON mode is specified. '-devicetype' cannot be specified at the same time.")
		}
		if c.buildTarget != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON mode is specified. '-target' cannot be specified at the same time.")
		}
		if c.model != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON mode is specified. '-model' cannot be specified at the same time.")
		}
		if len(c.tags) > 0 {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON mode is specified. '-tags' cannot be specified at the same time.")
		}
	} else {
		if c.machineName == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n'-name' is required, no mode ('-f') is setup.")
		}
		if c.zoneName == "" && c.manufacturer == "" && c.serialNumber == "" &&
			c.deviceType == "" && c.buildTarget == "" && c.model == "" && c.rackName == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nNothing to update. Please provide any field to update")
		}

		if c.deviceType != "" && !ufsUtil.IsAttachedDeviceType(c.deviceType) {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n%s is not a valid attached device type, please check help info for '-devicetype'.", c.deviceType)
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
