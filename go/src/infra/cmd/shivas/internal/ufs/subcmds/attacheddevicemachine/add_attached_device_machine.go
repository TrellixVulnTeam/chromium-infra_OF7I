// Copyright 2020 The Chromium Authors. All rights reserved.
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

// AddAttachedDeviceMachineCmd gets the attached device machine for a given name.
var AddAttachedDeviceMachineCmd = &subcommands.Command{
	UsageLine:  "attached-device-machine ...",
	ShortDesc:  "Add attached device machine details by filters",
	LongDesc:   cmdhelp.AddADMText,
	CommandRun: addADMCommandRun,
}

// AddADMCmd is an alias to AddAttachedDeviceMachineCmd
var AddADMCmd = &subcommands.Command{
	UsageLine:  "adm ...",
	ShortDesc:  "Add attached device machine details by filters",
	LongDesc:   cmdhelp.AddADMText,
	CommandRun: addADMCommandRun,
}

func addADMCommandRun() subcommands.CommandRun {
	c := &addAttachedDeviceMachine{}
	c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
	c.envFlags.Register(&c.Flags)
	c.commonFlags.Register(&c.Flags)

	c.Flags.StringVar(&c.newSpecsFile, "f", "", cmdhelp.ADMRegistrationFileText)

	c.Flags.StringVar(&c.machineName, "name", "", "The name of the attached device machine to add.")
	c.Flags.StringVar(&c.zoneName, "zone", "", cmdhelp.ZoneHelpText)
	c.Flags.StringVar(&c.rackName, "rack", "", "The rack to add the attached device machine to.")
	c.Flags.StringVar(&c.serialNumber, "serial", "", "The serial number for this attached device machine.")
	c.Flags.StringVar(&c.manufacturer, "man", "", "The manufacturer for this attached device machine.")
	c.Flags.StringVar(&c.deviceType, "devicetype", "", "The device type for this attached device machine. "+cmdhelp.AttachedDeviceTypeHelpText)
	c.Flags.StringVar(&c.buildTarget, "build-target", "", "The build target for this attached device machine.")
	c.Flags.StringVar(&c.model, "model", "", "The model for this attached device machine.")
	c.Flags.Var(flag.StringSlice(&c.tags), "tag", "Name(s) of tag(s). Can be specified multiple times.")
	return c
}

type addAttachedDeviceMachine struct {
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
}

func (c *addAttachedDeviceMachine) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *addAttachedDeviceMachine) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
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

	var machineRegistrationReq ufsAPI.MachineRegistrationRequest
	if c.newSpecsFile != "" {
		if err = utils.ParseJSONFile(c.newSpecsFile, &machineRegistrationReq); err != nil {
			return err
		}
		ufsZone := machineRegistrationReq.GetMachine().GetLocation().GetZone()
		machineRegistrationReq.GetMachine().Realm = ufsUtil.ToUFSRealm(ufsZone.String())
	} else {
		c.parseArgs(&machineRegistrationReq)
	}

	if !ufsUtil.ValidateTags(machineRegistrationReq.Machine.Tags) {
		return fmt.Errorf(ufsAPI.InvalidTags)
	}

	res, err := ic.MachineRegistration(ctx, &machineRegistrationReq)
	if err != nil {
		return err
	}
	utils.PrintProtoJSON(res, !utils.NoEmitMode(false))
	fmt.Println("Successfully added the attached device machine: ", res.GetName())
	return nil
}

func (c *addAttachedDeviceMachine) parseArgs(req *ufsAPI.MachineRegistrationRequest) {
	ufsZone := ufsUtil.ToUFSZone(c.zoneName)
	req.Machine = &ufspb.Machine{
		Name: c.machineName,
		Location: &ufspb.Location{
			Zone: ufsZone,
			Rack: c.rackName,
		},
		Realm:        ufsUtil.ToUFSRealm(c.zoneName),
		Tags:         c.tags,
		SerialNumber: c.serialNumber,
		Device: &ufspb.Machine_AttachedDevice{
			AttachedDevice: &ufspb.AttachedDevice{
				Manufacturer: c.manufacturer,
				DeviceType:   ufsUtil.ToUFSAttachedDeviceType(c.deviceType),
				BuildTarget:  c.buildTarget,
				Model:        c.model,
			},
		},
	}
}

func (c *addAttachedDeviceMachine) validateArgs() error {
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
		if c.zoneName == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n'-zone' is required, no mode ('-f') is setup.")
		}
		if c.serialNumber == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n'-serial' is required, no mode ('-f') is setup.")
		}
		if c.manufacturer == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n'-man' is required, no mode ('-f') is setup.")
		}
		if c.deviceType == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n'-devicetype' is required, no mode ('-f') is setup.")
		}
		if c.deviceType != "" && !ufsUtil.IsAttachedDeviceType(c.deviceType) {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n%s is not a valid attached device type, please check help info for '-devicetype'.", c.deviceType)
		}
		if c.buildTarget == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n'-target' is required, no mode ('-f') is setup.")
		}
		if c.model == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n'-model' is required, no mode ('-f') is setup.")
		}
		if !ufsUtil.IsUFSZone(ufsUtil.RemoveZonePrefix(c.zoneName)) {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n%s is not a valid zone name, please check help info for '-zone'.", c.zoneName)
		}
	}
	return nil
}
