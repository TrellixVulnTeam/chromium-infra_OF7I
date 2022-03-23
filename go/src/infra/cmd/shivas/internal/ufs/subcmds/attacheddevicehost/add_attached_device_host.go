// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package attacheddevicehost

import (
	"context"
	"fmt"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
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

// AddAttachedDeviceHostCmd adds a host to an attached device machine.
var AddAttachedDeviceHostCmd = &subcommands.Command{
	UsageLine:  "attached-device-host ...",
	ShortDesc:  "Add an attached device host on a machine",
	LongDesc:   cmdhelp.AddADHText,
	CommandRun: addADHCommandRun,
}

// AddADHCmd adds a host to an attached device machine.
var AddADHCmd = &subcommands.Command{
	UsageLine:  "adh ...",
	ShortDesc:  "Add an attached device host on a machine",
	LongDesc:   cmdhelp.AddADHText,
	CommandRun: addADHCommandRun,
}

func addADHCommandRun() subcommands.CommandRun {
	c := &addAttachedDeviceHost{}
	c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
	c.envFlags.Register(&c.Flags)
	c.commonFlags.Register(&c.Flags)
	c.Flags.StringVar(&c.newSpecsFile, "f", "", cmdhelp.AttachedDeviceMachineLSEFileText)

	c.Flags.StringVar(&c.hostName, "name", "", "name of the host")
	c.Flags.StringVar(&c.machineName, "machine", "", "name of the machine to associate the host")
	c.Flags.StringVar(&c.manufacturer, "man", "", "name of the manufacturer of the host")
	c.Flags.StringVar(&c.osVersion, "os", "", "name of the os version of the host")
	c.Flags.StringVar(&c.assocHostname, "associated-hostname", "", "the hostname of the device that the attached device is attached to")
	c.Flags.StringVar(&c.assocHostPort, "associated-hostport", "", "the port used of the device that the attached device is attached to")
	c.Flags.BoolVar(&c.schedulable, "schedulable", true, "whether this host should be exposed to the scheduling layer or not (default to true)")
	c.Flags.Var(flag.StringSlice(&c.tags), "tag", "Name(s) of tag(s). Can be specified multiple times. "+cmdhelp.ClearFieldHelpText)
	return c
}

type addAttachedDeviceHost struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags

	newSpecsFile string

	hostName      string
	machineName   string
	manufacturer  string
	osVersion     string
	assocHostname string
	assocHostPort string
	schedulable   bool
	tags          []string
}

func (c *addAttachedDeviceHost) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *addAttachedDeviceHost) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
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

	var machinelse ufspb.MachineLSE
	if c.newSpecsFile != "" {
		if err = utils.ParseJSONFile(c.newSpecsFile, &machinelse); err != nil {
			return err
		}
	} else {
		c.parseArgs(&machinelse)
	}

	if err = c.validateAttachedDeviceLse(ctx, ic, &machinelse); err != nil {
		return err
	}

	req := &ufsAPI.CreateMachineLSERequest{
		MachineLSE:   &machinelse,
		MachineLSEId: machinelse.GetName(),
	}

	res, err := ic.CreateMachineLSE(ctx, req)
	if err != nil {
		return err
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	c.printRes(ctx, ic, res)
	return nil
}

func (c *addAttachedDeviceHost) printRes(ctx context.Context, ic ufsAPI.FleetClient, res *ufspb.MachineLSE) {
	fmt.Println("The newly added host:")
	utils.PrintProtoJSON(res, !utils.NoEmitMode(false))
}

func (c *addAttachedDeviceHost) validateAttachedDeviceLse(ctx context.Context, ic ufsAPI.FleetClient, machinelse *ufspb.MachineLSE) error {
	if machinelse.GetMachines() == nil || len(machinelse.GetMachines()) <= 0 {
		return errors.New(fmt.Sprintf("machines field is empty in json. It is a required parameter for json input."))
	}

	machine, err := ic.GetMachine(ctx, &ufsAPI.GetMachineRequest{Name: ufsUtil.AddPrefix(ufsUtil.MachineCollection, machinelse.GetMachines()[0])})
	if err != nil {
		return errors.New(fmt.Sprintf("fail to find attached device machine %s", machinelse.GetMachines()[0]))
	}
	if machine.GetAttachedDevice() == nil {
		return errors.New(fmt.Sprintf("machine is not an attached device %s", machinelse.GetMachines()[0]))
	}

	_, err = ic.GetMachineLSE(ctx, &ufsAPI.GetMachineLSERequest{Name: ufsUtil.AddPrefix(ufsUtil.MachineLSECollection, machinelse.GetAttachedDeviceLse().GetAssociatedHostname())})
	if err != nil {
		return errors.New(fmt.Sprintf("fail to find associated host %s", machinelse.GetAttachedDeviceLse().GetAssociatedHostname()))
	}

	if !ufsUtil.ValidateTags(machinelse.Tags) {
		return fmt.Errorf(ufsAPI.InvalidTags)
	}

	return nil
}

func (c *addAttachedDeviceHost) parseArgs(lse *ufspb.MachineLSE) {
	lse.Hostname = c.hostName
	lse.Name = c.hostName
	lse.Machines = []string{c.machineName}
	lse.Manufacturer = c.manufacturer
	lse.Schedulable = c.schedulable
	lse.Tags = c.tags
	lse.Lse = &ufspb.MachineLSE_AttachedDeviceLse{
		AttachedDeviceLse: &ufspb.AttachedDeviceLSE{
			OsVersion: &ufspb.OSVersion{
				Value: c.osVersion,
			},
			AssociatedHostname: c.assocHostname,
			AssociatedHostPort: c.assocHostPort,
		},
	}
}

func (c *addAttachedDeviceHost) validateArgs() error {
	if c.newSpecsFile != "" {
		if c.hostName != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON mode is specified. '-name' cannot be specified at the same time.")
		}
		if c.machineName != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON mode is specified. '-machine' cannot be specified at the same time.")
		}
		if c.manufacturer != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON mode is specified. '-man' cannot be specified at the same time.")
		}
		if c.osVersion != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON mode is specified. '-os' cannot be specified at the same time.")
		}
		if c.assocHostname != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON mode is specified. '-associated-hostname' cannot be specified at the same time.")
		}
		if c.assocHostPort != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON mode is specified. '-associated-hostport' cannot be specified at the same time.")
		}
		if len(c.tags) > 0 {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON mode is specified. '-tag' cannot be specified at the same time.")
		}
	} else {
		if c.hostName == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n'-name' is required, no mode ('-f') is specified.")
		}
		if c.machineName == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n'-machine' is required, no mode ('-f') is specified.")
		}
		if c.assocHostname == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n'-associated-hostname' is required, no mode ('-f') is specified.")
		}
	}
	return nil
}
