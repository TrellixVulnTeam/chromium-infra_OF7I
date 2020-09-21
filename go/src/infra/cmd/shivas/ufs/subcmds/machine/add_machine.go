// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package machine

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

// AddMachineCmd add Machine to the system.
var AddMachineCmd = &subcommands.Command{
	UsageLine: "machine [Options...]",
	ShortDesc: "Add a machine(Hardware asset: ChromeBook, Bare metal server, Macbook.)",
	LongDesc:  cmdhelp.AddMachineLongDesc,
	CommandRun: func() subcommands.CommandRun {
		c := &addMachine{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.commonFlags.Register(&c.Flags)

		c.Flags.StringVar(&c.newSpecsFile, "f", "", cmdhelp.MachineRegistrationFileText)

		c.Flags.StringVar(&c.machineName, "name", "", "the name of the machine to add")
		c.Flags.StringVar(&c.zoneName, "zone", "", cmdhelp.ZoneHelpText)
		c.Flags.StringVar(&c.rackName, "rack", "", "the rack to add the machine to")
		c.Flags.StringVar(&c.platform, "platform", "", "the platform of this machine")
		c.Flags.StringVar(&c.kvm, "kvm", "", "the name of the kvm that this machine uses")
		c.Flags.StringVar(&c.kvmPort, "kvm-port", "", "the port of the kvm that this machine uses")
		c.Flags.StringVar(&c.deploymentTicket, "ticket", "", "the deployment ticket for this machine")
		c.Flags.StringVar(&c.serialNumber, "serial", "", "the serial number for this machine")
		c.Flags.StringVar(&c.tags, "tags", "", "comma separated tags. You can only append/add new tags here.")
		return c
	},
}

type addMachine struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags

	newSpecsFile string

	machineName      string
	zoneName         string
	rackName         string
	platform         string
	kvm              string
	kvmPort          string
	deploymentTicket string
	tags             string
	serialNumber     string
}

func (c *addMachine) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *addMachine) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
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

	res, err := ic.MachineRegistration(ctx, &machineRegistrationReq)
	if err != nil {
		return err
	}
	utils.PrintProtoJSON(res, !utils.NoEmitMode(false))
	fmt.Println("Successfully added the machine: ", res.GetName())
	return nil
}

func (c *addMachine) parseArgs(req *ufsAPI.MachineRegistrationRequest) {
	ufsZone := ufsUtil.ToUFSZone(c.zoneName)
	req.Machine = &ufspb.Machine{
		Name: c.machineName,
		Location: &ufspb.Location{
			Zone: ufsZone,
			Rack: c.rackName,
		},
		Realm:        ufsUtil.ToUFSRealm(c.zoneName),
		Tags:         utils.GetStringSlice(c.tags),
		SerialNumber: c.serialNumber,
	}
	if ufsUtil.IsInBrowserZone(ufsZone.String()) {
		req.Machine.Device = &ufspb.Machine_ChromeBrowserMachine{
			ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{
				DisplayName:      c.machineName,
				ChromePlatform:   c.platform,
				DeploymentTicket: c.deploymentTicket,
				KvmInterface: &ufspb.KVMInterface{
					Kvm:      c.kvm,
					PortName: c.kvmPort,
				},
				RpmInterface: &ufspb.RPMInterface{},
			},
		}
	} else {
		req.Machine.Device = &ufspb.Machine_ChromeosMachine{
			ChromeosMachine: &ufspb.ChromeOSMachine{},
		}
	}
}

func (c *addMachine) validateArgs() error {
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
	} else {
		if c.machineName == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n'-name' is required, no mode ('-f') is setup.")
		}
		if c.zoneName == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n'-zone' is required, no mode ('-f') is setup.")
		}
		if !ufsUtil.IsUFSZone(ufsUtil.RemoveZonePrefix(c.zoneName)) {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n%s is not a valid zone name, please check help info for '-zone'.", c.zoneName)
		}
	}
	return nil
}
