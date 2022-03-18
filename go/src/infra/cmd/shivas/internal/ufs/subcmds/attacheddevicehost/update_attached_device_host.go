// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package attacheddevicehost

import (
	"context"
	"flag"
	"fmt"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
	luciflag "go.chromium.org/luci/common/flag"
	"go.chromium.org/luci/grpc/prpc"

	"infra/cmd/shivas/cmdhelp"
	"infra/cmd/shivas/site"
	"infra/cmd/shivas/utils"
	"infra/cmdsupport/cmdlib"
	ufspb "infra/unifiedfleet/api/v1/models"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	ufsUtil "infra/unifiedfleet/app/util"
)

// UpdateAttachedDeviceHostCmd updates the attached device host for a given name.
var UpdateAttachedDeviceHostCmd = &subcommands.Command{
	UsageLine:  "attached-device-host ...",
	ShortDesc:  "Update attached device host details by filters",
	LongDesc:   cmdhelp.UpdateADHText,
	CommandRun: updateADHCommandRun,
}

// UpdateADHCmd is an alias to UpdateAttachedDeviceHostCmd
var UpdateADHCmd = &subcommands.Command{
	UsageLine:  "adh ...",
	ShortDesc:  "Update attached device host details by filters",
	LongDesc:   cmdhelp.UpdateADHText,
	CommandRun: updateADHCommandRun,
}

func updateADHCommandRun() subcommands.CommandRun {
	c := &updateAttachedDeviceHost{}
	c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
	c.envFlags.Register(&c.Flags)
	c.commonFlags.Register(&c.Flags)

	c.Flags.StringVar(&c.newSpecsFile, "f", "", cmdhelp.AttachedDeviceMachineLSEFileText)

	c.Flags.StringVar(&c.hostName, "name", "", "name of the host")
	c.Flags.StringVar(&c.machineName, "machine", "", "name of the machine that the host is connected to")
	c.Flags.StringVar(&c.osVersion, "os", "", "name of the os version of the host")
	c.Flags.BoolVar(&c.schedulable, "schedulable", true, "whether this host should be exposed to the scheduling layer or not. bool value true or false.")
	c.Flags.StringVar(&c.state, "state", "", cmdhelp.StateHelp)

	c.Flags.StringVar(&c.assocHostname, "associated-hostname", "", "the hostname of the device that the attached device is attached to")
	c.Flags.StringVar(&c.assocHostPort, "associated-hostport", "", "the port used of the device that the attached device is attached to")

	c.Flags.StringVar(&c.description, "desc", "", "description for the lse. "+cmdhelp.ClearFieldHelpText)
	c.Flags.Var(luciflag.StringSlice(&c.tags), "tag", "Name(s) of tag(s). Can be specified multiple times. "+cmdhelp.ClearFieldHelpText)

	return c
}

type updateAttachedDeviceHost struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags
	outputFlags site.OutputFlags

	newSpecsFile string

	hostName      string
	machineName   string
	osVersion     string
	schedulable   bool
	state         string
	assocHostname string
	assocHostPort string
	description   string
	tags          []string

	// For use in determining if a flag is set
	flagInputs map[string]bool
}

func (c *updateAttachedDeviceHost) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *updateAttachedDeviceHost) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	c.flagInputs = make(map[string]bool)
	c.Flags.Visit(func(f *flag.Flag) {
		c.flagInputs[f.Name] = true
	})

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
	machinelse := &ufspb.MachineLSE{}
	if c.newSpecsFile != "" {
		if err = utils.ParseJSONFile(c.newSpecsFile, machinelse); err != nil {
			return err
		}
		if machinelse.GetMachines() == nil || len(machinelse.GetMachines()) <= 0 {
			return errors.New(fmt.Sprintf("machines field is empty in json. It is a required parameter for json input."))
		}
	} else {
		c.parseArgs(machinelse)
	}

	_, err = utils.PrintExistingAttachedDeviceHost(ctx, ic, machinelse.Name)
	if err != nil {
		return err
	}

	machinelse.Name = ufsUtil.AddPrefix(ufsUtil.MachineLSECollection, machinelse.Name)
	if !ufsUtil.ValidateTags(machinelse.Tags) {
		return fmt.Errorf(ufsAPI.InvalidTags)
	}

	if err = c.validateAttachedDeviceLse(ctx, ic, machinelse); err != nil {
		return err
	}

	res, err := ic.UpdateMachineLSE(ctx, &ufsAPI.UpdateMachineLSERequest{
		MachineLSE: machinelse,
		UpdateMask: utils.GetUpdateMask(&c.Flags, map[string]string{
			"machine":             "machines",
			"os":                  "osVersion",
			"associated-hostname": "assocHostname",
			"associated-hostport": "assocHostPort",
			"schedulable":         "schedulable",
			"tag":                 "tags",
			"desc":                "description",
			"state":               "resourceState",
		}),
	})
	if err != nil {
		return err
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	c.printRes(ctx, ic, res)
	return nil
}

func (c *updateAttachedDeviceHost) printRes(ctx context.Context, ic ufsAPI.FleetClient, res *ufspb.MachineLSE) {
	fmt.Println("The attached device host after update:")
	utils.PrintProtoJSON(res, !utils.NoEmitMode(false))
	fmt.Println("Successfully updated the attached device host: ", res.Name)
}

func (c *updateAttachedDeviceHost) parseArgs(lse *ufspb.MachineLSE) {
	lse.Lse = &ufspb.MachineLSE_AttachedDeviceLse{
		AttachedDeviceLse: &ufspb.AttachedDeviceLSE{
			OsVersion: &ufspb.OSVersion{},
		},
	}
	lse.Name = c.hostName
	lse.Hostname = c.hostName

	if c.machineName != "" {
		lse.Machines = []string{c.machineName}
	}

	if c.osVersion == utils.ClearFieldValue {
		lse.GetAttachedDeviceLse().GetOsVersion().Value = ""
	} else if c.osVersion != "" {
		lse.GetAttachedDeviceLse().GetOsVersion().Value = c.osVersion
	}

	if c.flagInputs["schedulable"] {
		lse.Schedulable = c.schedulable
	}

	lse.ResourceState = ufsUtil.ToUFSState(c.state)

	if c.assocHostname == utils.ClearFieldValue {
		lse.GetAttachedDeviceLse().AssociatedHostname = ""
	} else if c.assocHostname != "" {
		lse.GetAttachedDeviceLse().AssociatedHostname = c.assocHostname
	}
	if c.assocHostPort == utils.ClearFieldValue {
		lse.GetAttachedDeviceLse().AssociatedHostPort = ""
	} else if c.assocHostPort != "" {
		lse.GetAttachedDeviceLse().AssociatedHostPort = c.assocHostPort
	}

	if c.description == utils.ClearFieldValue {
		lse.Description = ""
	} else if c.description != "" {
		lse.Description = c.description
	}
	if ufsUtil.ContainsAnyStrings(c.tags, utils.ClearFieldValue) {
		lse.Tags = nil
	} else if len(c.tags) > 0 {
		lse.Tags = c.tags
	}
}

func (c *updateAttachedDeviceHost) validateArgs() error {
	if c.newSpecsFile != "" {
		if c.hostName != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON mode is specified. '-name' cannot be specified at the same time.")
		}
		if c.machineName != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON mode is specified. '-machine' cannot be specified at the same time.")
		}
		if c.osVersion != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON mode is specified. '-os' cannot be specified at the same time.")
		}
		if c.flagInputs["schedulable"] {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON mode is specified. '-schedulable' cannot be specified at the same time.")
		}
		if c.state != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON mode is specified. '-state' cannot be specified at the same time.")
		}
		if c.assocHostname != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON mode is specified. '-associated-hostname' cannot be specified at the same time.")
		}
		if c.assocHostPort != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON mode is specified. '-associated-hostport' cannot be specified at the same time.")
		}
		if c.description != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON mode is specified. '-desc' cannot be specified at the same time.")
		}
		if len(c.tags) > 0 {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON mode is specified. '-tag' cannot be specified at the same time.")
		}
	} else {
		if c.hostName == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n'-name' is required, no mode ('-f') is specified.")
		}
		if c.machineName == utils.ClearFieldValue {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n'-machine' can only be used to replace a machine, not remove.")
		}
		if c.state != "" && !ufsUtil.IsUFSState(ufsUtil.RemoveStatePrefix(c.state)) {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n%s is not a valid state, please check help info for '-state'.", c.state)
		}
		if c.osVersion == "" && c.assocHostname == "" && c.assocHostPort == "" &&
			c.description == "" && len(c.tags) == 0 && c.state == "" && !c.flagInputs["schedulable"] {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nNothing to update. Please provide any field to update")
		}
	}
	return nil
}

func (c *updateAttachedDeviceHost) validateAttachedDeviceLse(ctx context.Context, ic ufsAPI.FleetClient, machinelse *ufspb.MachineLSE) error {
	if machinelse.GetMachines() != nil || len(machinelse.GetMachines()) > 0 {
		machine, err := ic.GetMachine(ctx, &ufsAPI.GetMachineRequest{Name: ufsUtil.AddPrefix(ufsUtil.MachineCollection, machinelse.GetMachines()[0])})
		if err != nil {
			return errors.New(fmt.Sprintf("fail to find attached device machine %s", machinelse.GetMachines()[0]))
		}
		if machine.GetAttachedDevice() == nil {
			return errors.New(fmt.Sprintf("machine is not an attached device %s", machinelse.GetMachines()[0]))
		}
	}

	if machinelse.GetAttachedDeviceLse().GetAssociatedHostname() != "" {
		_, err := ic.GetMachineLSE(ctx, &ufsAPI.GetMachineLSERequest{Name: ufsUtil.AddPrefix(ufsUtil.MachineLSECollection, machinelse.GetAttachedDeviceLse().GetAssociatedHostname())})
		if err != nil {
			return errors.New(fmt.Sprintf("fail to find associated host %s", machinelse.GetAttachedDeviceLse().GetAssociatedHostname()))
		}
	}

	if !ufsUtil.ValidateTags(machinelse.Tags) {
		return fmt.Errorf(ufsAPI.InvalidTags)
	}

	return nil
}
