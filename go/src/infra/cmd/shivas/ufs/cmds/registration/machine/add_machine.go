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

// AddMachineCmd add Machine to the system.
var AddMachineCmd = &subcommands.Command{
	UsageLine: "add-machine [Options...]",
	ShortDesc: "Create a machine(Hardware asset: ChromeBook, Bare metal server, Macbook.) to UFS",
	LongDesc:  cmdhelp.AddMachineLongDesc,
	CommandRun: func() subcommands.CommandRun {
		c := &addMachine{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.commonFlags.Register(&c.Flags)
		c.Flags.StringVar(&c.newSpecsFile, "f", "", cmdhelp.MachineRegistrationFileText)

		c.Flags.StringVar(&c.machineName, "name", "", "the name of the machine to add")
		c.Flags.StringVar(&c.labName, "lab", "", fmt.Sprintf("the name of the lab to add the machine to. Valid lab strings: [%s]", strings.Join(utils.ValidLabStr(), ", ")))
		c.Flags.StringVar(&c.rackName, "rack", "", "the rack to add the machine to")
		c.Flags.StringVar(&c.platform, "platform", "", "the platform of this machine")
		c.Flags.StringVar(&c.kvm, "kvm", "", "the name of the kvm that this machine uses")
		c.Flags.StringVar(&c.deploymentTicket, "ticket", "", "the deployment ticket for this machine")
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
	labName          string
	rackName         string
	platform         string
	kvm              string
	deploymentTicket string
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
		ufsLab := utils.ToUFSLab(c.labName)
		machineRegistrationReq.GetMachine().GetLocation().Lab = ufsLab
		machineRegistrationReq.GetMachine().Realm = utils.ToUFSRealm(ufsLab.String())
	} else {
		c.parseArgs(&machineRegistrationReq)
	}

	res, err := ic.MachineRegistration(ctx, &machineRegistrationReq)
	if err != nil {
		return err
	}
	utils.PrintProtoJSON(res)
	fmt.Println("Successfully added the machine: ", machineRegistrationReq.GetMachine().GetName())
	return nil
}

func (c *addMachine) parseArgs(req *ufsAPI.MachineRegistrationRequest) {
	ufsLab := utils.ToUFSLab(c.labName)
	req.Machine = &ufspb.Machine{
		Name: c.machineName,
		Location: &ufspb.Location{
			Lab:  ufsLab,
			Rack: c.rackName,
		},
		Realm: utils.ToUFSRealm(c.labName),
	}
	if ufsUtil.IsInBrowserLab(ufsLab.String()) {
		req.Machine.Device = &ufspb.Machine_ChromeBrowserMachine{
			ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{
				DisplayName:      c.machineName,
				ChromePlatform:   c.platform,
				DeploymentTicket: c.deploymentTicket,
				KvmInterface: &ufspb.KVMInterface{
					Kvm: c.kvm,
				},
			},
		}
	} else {
		req.Machine.Device = &ufspb.Machine_ChromeosMachine{
			ChromeosMachine: &ufspb.ChromeOSMachine{},
		}
	}
}

func (c *addMachine) validateArgs() error {
	if c.labName == "" {
		return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\n'-lab' is required.")
	}
	if !utils.IsUFSLab(c.labName) {
		return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\n%s is not a valid lab name, please check help info for '-lab'.", c.labName)
	}
	if c.newSpecsFile != "" {
		if c.machineName != "" {
			return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nThe JSON input file is already specified. '-name' cannot be specified at the same time.")
		}

	} else {
		if c.machineName == "" {
			return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nNo mode ('-f') is setup, so '-name' is required.")
		}
	}
	return nil
}
