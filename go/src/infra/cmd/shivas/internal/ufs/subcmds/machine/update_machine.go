// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package machine

import (
	"fmt"
	"os"

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

// UpdateMachineCmd update Machine by given name.
var UpdateMachineCmd = &subcommands.Command{
	UsageLine: "machine [Options...]",
	ShortDesc: "Update a machine(Hardware asset: ChromeBook, Bare metal server, Macbook.)",
	LongDesc:  cmdhelp.UpdateMachineLongDesc,
	CommandRun: func() subcommands.CommandRun {
		c := &updateMachine{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.commonFlags.Register(&c.Flags)

		c.Flags.StringVar(&c.newSpecsFile, "f", "", cmdhelp.MachineFileText)
		c.Flags.BoolVar(&c.interactive, "i", false, "enable interactive mode for input")

		c.Flags.StringVar(&c.machineName, "name", "", "the name of the machine to update")
		c.Flags.StringVar(&c.zoneName, "zone", "", cmdhelp.ZoneHelpText)
		c.Flags.StringVar(&c.rackName, "rack", "", "the rack to add the machine to. "+cmdhelp.ClearFieldHelpText)
		c.Flags.StringVar(&c.platform, "platform", "", "the platform of this machine. "+cmdhelp.ClearFieldHelpText)
		c.Flags.StringVar(&c.kvm, "kvm", "", "the name of the kvm that this machine uses. "+cmdhelp.ClearFieldHelpText)
		c.Flags.StringVar(&c.kvmPort, "kvm-port", "", "the port of the kvm that this machine uses"+cmdhelp.ClearFieldHelpText)
		c.Flags.StringVar(&c.deploymentTicket, "ticket", "", "the deployment ticket for this machine. "+cmdhelp.ClearFieldHelpText)
		c.Flags.StringVar(&c.serialNumber, "serial", "", "the serial number for this machine. "+cmdhelp.ClearFieldHelpText)
		c.Flags.StringVar(&c.tags, "tags", "", "comma separated tags. You can only append/add new tags here. "+cmdhelp.ClearFieldHelpText)
		c.Flags.StringVar(&c.state, "state", "", cmdhelp.StateHelp)
		c.Flags.StringVar(&c.description, "desc", "", "description for the machine. "+cmdhelp.ClearFieldHelpText)

		return c
	},
}

type updateMachine struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags

	newSpecsFile string
	interactive  bool

	machineName      string
	zoneName         string
	rackName         string
	platform         string
	kvm              string
	kvmPort          string
	deploymentTicket string
	tags             string
	serialNumber     string
	state            string
	description      string
}

func (c *updateMachine) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *updateMachine) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
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
	if c.interactive {
		utils.GetMachineInteractiveInput(ctx, ic, &machine, true)
	} else {
		if c.newSpecsFile != "" {
			if err = utils.ParseJSONFile(c.newSpecsFile, &machine); err != nil {
				return err
			}
			machine.Realm = ufsUtil.ToUFSRealm(machine.GetLocation().GetZone().String())
		} else {
			c.parseArgs(&machine)
		}
	}
	existingMachine, err := utils.PrintExistingMachine(ctx, ic, machine.Name)
	if err != nil {
		return err
	}
	// Check HWID for non-partial update
	// TODO(anushruth): Check for file type when implementing mcsv support.
	if c.newSpecsFile != "" && existingMachine.GetChromeosMachine() != nil && existingMachine.GetChromeosMachine().GetHwid() != machine.GetChromeosMachine().GetHwid() {
		newHWID := machine.GetChromeosMachine().GetHwid()
		if newHWID == "" {
			return fmt.Errorf("users cannot update hwid to empty string manually")
		}
		prompt := utils.CLIPrompt(a.GetOut(), os.Stdin, false)
		if prompt != nil && !prompt(fmt.Sprintf("HWID can only be used by Fleet Admins. Are you sure you want to modify the HWID to %s?", newHWID)) {
			return nil
		}
	}
	machine.Name = ufsUtil.AddPrefix(ufsUtil.MachineCollection, machine.Name)
	if !ufsUtil.ValidateTags(machine.Tags) {
		return fmt.Errorf(ufsAPI.InvalidTags)
	}
	res, err := ic.UpdateMachine(ctx, &ufsAPI.UpdateMachineRequest{
		Machine: &machine,
		UpdateMask: utils.GetUpdateMask(&c.Flags, map[string]string{
			"zone":     "zone",
			"rack":     "rack",
			"platform": "platform",
			"kvm":      "kvm",
			"kvm-port": "kvmport",
			"ticket":   "deploymentTicket",
			"tags":     "tags",
			"serial":   "serialNumber",
			"state":    "resourceState",
			"desc":     "description",
		}),
	})
	if err != nil {
		return err
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	fmt.Println("The machine after update:")
	utils.PrintProtoJSON(res, !utils.NoEmitMode(false))
	fmt.Println("Successfully updated the machine: ", res.Name)
	return nil
}

func (c *updateMachine) parseArgs(machine *ufspb.Machine) {
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
	if c.tags == utils.ClearFieldValue {
		machine.Tags = nil
	} else {
		machine.Tags = utils.GetStringSlice(c.tags)
	}
	if c.serialNumber == utils.ClearFieldValue {
		machine.SerialNumber = ""
	} else {
		machine.SerialNumber = c.serialNumber
	}
	machine.ResourceState = ufsUtil.ToUFSState(c.state)
	if c.platform != "" || c.deploymentTicket != "" || c.kvm != "" || c.kvmPort != "" || c.description != "" {
		machine.Device = &ufspb.Machine_ChromeBrowserMachine{
			ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{
				DisplayName:  c.machineName,
				KvmInterface: &ufspb.KVMInterface{},
			},
		}
		if c.platform == utils.ClearFieldValue {
			machine.GetChromeBrowserMachine().ChromePlatform = ""
		} else {
			machine.GetChromeBrowserMachine().ChromePlatform = c.platform
		}
		if c.deploymentTicket == utils.ClearFieldValue {
			machine.GetChromeBrowserMachine().DeploymentTicket = ""
		} else {
			machine.GetChromeBrowserMachine().DeploymentTicket = c.deploymentTicket
		}
		if c.kvm == utils.ClearFieldValue {
			machine.GetChromeBrowserMachine().GetKvmInterface().Kvm = ""
		} else {
			machine.GetChromeBrowserMachine().GetKvmInterface().Kvm = c.kvm
		}
		if c.kvmPort == utils.ClearFieldValue {
			machine.GetChromeBrowserMachine().GetKvmInterface().PortName = ""
		} else {
			machine.GetChromeBrowserMachine().GetKvmInterface().PortName = c.kvmPort
		}
		if c.description == utils.ClearFieldValue {
			machine.GetChromeBrowserMachine().Description = ""
		} else {
			machine.GetChromeBrowserMachine().Description = c.description
		}
	}
	machine.Realm = ufsUtil.ToUFSRealm(machine.GetLocation().GetZone().String())
}

func (c *updateMachine) validateArgs() error {
	if c.newSpecsFile != "" && c.interactive {
		return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe interactive & JSON mode cannot be specified at the same time.")
	}
	if c.newSpecsFile != "" || c.interactive {
		if c.machineName != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON input file is already specified. '-name' cannot be specified at the same time.")
		}
		if c.rackName != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON input file is already specified. '-rack' cannot be specified at the same time.")
		}
		if c.zoneName != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON input file is already specified. '-zone' cannot be specified at the same time.")
		}
		if c.platform != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON mode is specified. '-platform' cannot be specified at the same time.")
		}
		if c.kvm != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON input file is already specified. '-kvm' cannot be specified at the same time.")
		}
		if c.deploymentTicket != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON input file is already specified. '-ticket' cannot be specified at the same time.")
		}
		if c.serialNumber != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON mode is specified. '-serial' cannot be specified at the same time.")
		}
		if c.tags != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON mode is specified. '-tags' cannot be specified at the same time.")
		}
		if c.state != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON input file is already specified. '-state' cannot be specified at the same time.")
		}
		if c.description != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe interactive/JSON mode is specified. '-desc' cannot be specified at the same time.")
		}
	}
	if c.newSpecsFile == "" && !c.interactive {
		if c.machineName == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n'-name' is required, no mode ('-f' or '-i') is specified.")
		}
		if c.zoneName == "" && c.rackName == "" && c.state == "" &&
			c.tags == "" && c.platform == "" && c.deploymentTicket == "" &&
			c.kvm == "" && c.kvmPort == "" && c.serialNumber == "" && c.description == "" {
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
