// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package nic

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

// AddNicCmd add Nic in the lab.
var AddNicCmd = &subcommands.Command{
	UsageLine: "add-nic [Options...]",
	ShortDesc: "Add a nic by name",
	LongDesc:  cmdhelp.AddNicLongDesc,
	CommandRun: func() subcommands.CommandRun {
		c := &addNic{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.commonFlags.Register(&c.Flags)
		c.Flags.StringVar(&c.newSpecsFile, "f", "", cmdhelp.NicFileText)
		c.Flags.BoolVar(&c.interactive, "i", false, "enable interactive mode for input")

		c.Flags.StringVar(&c.machineName, "machine", "", "name of the machine to associate the nic")
		c.Flags.StringVar(&c.nicName, "name", "", "the name of the nic to add")
		c.Flags.StringVar(&c.macAddress, "mac-address", "", "the mac address of the nic to add")
		c.Flags.StringVar(&c.switchName, "switch", "", "the name of the switch that this nic is connected to")
		c.Flags.IntVar(&c.port, "switch-port", 0, "the port of the switch that this nic is connected to")
		return c
	},
}

type addNic struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags

	newSpecsFile string
	interactive  bool

	machineName string
	nicName     string
	macAddress  string
	switchName  string
	port        int
}

func (c *addNic) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *addNic) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
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

	var nic ufspb.Nic
	if c.interactive {
		c.machineName = utils.GetNicInteractiveInput(ctx, ic, &nic, false)
	} else {
		if c.newSpecsFile != "" {
			if err = utils.ParseJSONFile(c.newSpecsFile, &nic); err != nil {
				return err
			}
		} else {
			c.parseArgs(&nic)
		}
	}
	res, err := ic.CreateNic(ctx, &ufsAPI.CreateNicRequest{
		Nic:     &nic,
		NicId:   nic.GetName(),
		Machine: c.machineName,
	})
	if err != nil {
		return err
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	utils.PrintProtoJSON(res)
	fmt.Printf("Successfully added the nic %s to machine %s\n", res.Name, c.machineName)
	return nil
}

func (c *addNic) parseArgs(nic *ufspb.Nic) {
	nic.Name = c.nicName
	nic.MacAddress = c.macAddress
	nic.SwitchInterface = &ufspb.SwitchInterface{
		Switch: c.switchName,
		Port:   int32(c.port),
	}
}

func (c *addNic) validateArgs() error {
	if c.newSpecsFile != "" || c.interactive {
		if c.nicName != "" {
			return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nThe interactive/JSON mode is specified. '-name' cannot be specified at the same time.")
		}
		if c.switchName != "" {
			return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nThe interactive/JSON mode is specified. '-switch' cannot be specified at the same time.")
		}
		if c.macAddress != "" {
			return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nThe interactive/JSON mode is specified. '-mac-address' cannot be specified at the same time.")
		}
	}
	if c.newSpecsFile != "" {
		if c.interactive {
			return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nThe interactive & JSON mode cannot be specified at the same time.")
		}
		if c.machineName == "" {
			return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nMachine name (-machine) is required for JSON mode.")
		}
	}
	if c.newSpecsFile == "" && !c.interactive {
		if c.nicName == "" {
			return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nNo mode ('-f or '-i') is specified, so '-name' is required.")
		}
		if c.switchName == "" {
			return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nNo mode ('-f' or '-i') is specified, so '-switch' is required.")
		}
		if c.macAddress == "" {
			return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nNo mode ('-f' or '-i') is specified, so '-mac-address' is required.")
		}
		if c.machineName == "" {
			return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nMachine name (-machine) is required.")
		}
		if c.port == 0 {
			return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nNo mode ('-f' or '-i'), so '-switch-port' is required.")
		}
	}
	return nil
}
