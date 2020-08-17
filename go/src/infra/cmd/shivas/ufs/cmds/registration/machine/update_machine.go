// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package machine

import (
	"fmt"
	"strings"

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

// UpdateMachineCmd update Machine by given name.
var UpdateMachineCmd = &subcommands.Command{
	UsageLine: "update-machine [Options...]",
	ShortDesc: "Update a machine(Hardware asset: ChromeBook, Bare metal server, Macbook.) by name",
	LongDesc:  cmdhelp.UpdateMachineLongDesc,
	CommandRun: func() subcommands.CommandRun {
		c := &updateMachine{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.commonFlags.Register(&c.Flags)

		c.Flags.StringVar(&c.newSpecsFile, "f", "", cmdhelp.MachineFileText)
		c.Flags.BoolVar(&c.interactive, "i", false, "enable interactive mode for input")

		c.Flags.StringVar(&c.machineName, "name", "", "the name of the machine to update")
		c.Flags.StringVar(&c.zoneName, "zone", "", fmt.Sprintf("the name of the zone to add the machine to. Valid zone strings: [%s]. ", strings.Join(ufsUtil.ValidZoneStr(), ", ")))
		c.Flags.StringVar(&c.rackName, "rack", "", "the rack to add the machine to. "+cmdhelp.ClearFieldHelpText)
		c.Flags.StringVar(&c.platform, "platform", "", "the platform of this machine. "+cmdhelp.ClearFieldHelpText)
		c.Flags.StringVar(&c.kvm, "kvm", "", "the name of the kvm that this machine uses. "+cmdhelp.ClearFieldHelpText)
		c.Flags.StringVar(&c.deploymentTicket, "ticket", "", "the deployment ticket for this machine. "+cmdhelp.ClearFieldHelpText)
		c.Flags.StringVar(&c.serialNumber, "serial", "", "the serial number for this machine. "+cmdhelp.ClearFieldHelpText)
		c.Flags.StringVar(&c.tags, "tags", "", "comma separated tags. You can only append/add new tags here. "+cmdhelp.ClearFieldHelpText)
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
	deploymentTicket string
	tags             string
	serialNumber     string
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
		} else {
			c.parseArgs(&machine)
		}
	}
	machine.Name = ufsUtil.AddPrefix(ufsUtil.MachineCollection, machine.Name)
	res, err := ic.UpdateMachine(ctx, &ufsAPI.UpdateMachineRequest{
		Machine: &machine,
		UpdateMask: utils.GetUpdateMask(&c.Flags, map[string]string{
			"zone":     "zone",
			"rack":     "rack",
			"platform": "platform",
			"kvm":      "kvm",
			"ticket":   "ticket",
			"tags":     "tags",
			"serial":   "serialNumber",
		}),
	})
	if err != nil {
		return err
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	utils.PrintProtoJSON(res)
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
	if c.platform != "" || c.deploymentTicket != "" || c.kvm != "" {
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
	}
}

func (c *updateMachine) validateArgs() error {
	if c.newSpecsFile != "" && c.interactive {
		return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe interactive & JSON mode cannot be specified at the same time.")
	}
	if c.zoneName != "" && !ufsUtil.IsUFSZone(ufsUtil.RemoveZonePrefix(c.zoneName)) {
		return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n%s is not a valid zone name, please check help info for '-zone'.", c.zoneName)
	}
	if c.newSpecsFile == "" && !c.interactive {
		if c.machineName == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n'-name' is required, no mode ('-f' or '-i') is specified.")
		}
		if c.zoneName == "" && c.rackName == "" && c.tags == "" && c.platform == "" && c.deploymentTicket == "" && c.kvm == "" && c.serialNumber == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nNothing to update. Please provide any field to update")
		}
	}
	return nil
}
