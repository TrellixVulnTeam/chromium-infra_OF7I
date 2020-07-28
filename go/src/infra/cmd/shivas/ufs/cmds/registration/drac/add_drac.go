// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package drac

import (
	"fmt"
	"os"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/grpc/prpc"

	"infra/cmd/shivas/cmdhelp"
	"infra/cmd/shivas/site"
	"infra/cmd/shivas/utils"
	"infra/cmdsupport/cmdlib"
	ufspb "infra/unifiedfleet/api/v1/proto"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	ufsUtil "infra/unifiedfleet/app/util"
)

// AddDracCmd add Drac in the lab.
var AddDracCmd = &subcommands.Command{
	UsageLine: "add-drac [Options...]",
	ShortDesc: "Add a drac by name",
	LongDesc:  cmdhelp.AddDracLongDesc,
	CommandRun: func() subcommands.CommandRun {
		c := &addDrac{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.commonFlags.Register(&c.Flags)
		c.Flags.StringVar(&c.newSpecsFile, "f", "", cmdhelp.DracFileText)
		c.Flags.BoolVar(&c.interactive, "i", false, "enable interactive mode for input")

		c.Flags.StringVar(&c.machineName, "machine", "", "name of the machine to associate the drac")
		c.Flags.StringVar(&c.dracName, "name", "", "the name of the drac to add")
		c.Flags.StringVar(&c.macAddress, "mac-address", "", "the mac address of the drac to add")
		c.Flags.StringVar(&c.switchName, "switch", "", "the name of the switch that this drac is connected to")
		c.Flags.IntVar(&c.port, "switch-port", 0, "the port of the switch that this drac is connected to")
		return c
	},
}

type addDrac struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags

	newSpecsFile string
	interactive  bool

	machineName string
	dracName    string
	macAddress  string
	switchName  string
	port        int
}

func (c *addDrac) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *addDrac) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
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

	var drac ufspb.Drac
	if c.interactive {
		c.machineName = utils.GetDracInteractiveInput(ctx, ic, &drac, false)
	} else {
		if c.newSpecsFile != "" {
			if err := utils.ParseJSONFile(c.newSpecsFile, &drac); err != nil {
				return err
			}
		} else {
			c.parseArgs(&drac)
		}
	}
	if c.machineName != "" {
		machine, err := ic.GetMachine(ctx, &ufsAPI.GetMachineRequest{
			Name: ufsUtil.AddPrefix(ufsUtil.MachineCollection, c.machineName),
		})
		if err != nil {
			return errors.Annotate(err, "Machine %s not found", c.machineName).Err()
		}
		machine.Name = ufsUtil.RemovePrefix(machine.Name)
		if oldDracName := machine.GetChromeBrowserMachine().GetDrac(); oldDracName != "" && oldDracName != drac.Name {
			prompt := utils.CLIPrompt(a.GetOut(), os.Stdin, false)
			if !prompt(fmt.Sprintf("Machine %s is already associated with drac %s. "+
				"If you associate this drac %s with the machine %s, "+
				"the old drac %s will be deleted from the system. "+
				"A drac cannot exist in the system without being associated to a machine.\n"+
				"Are you sure you want to add the drac to this machine?",
				c.machineName, oldDracName, drac.Name, c.machineName, oldDracName)) {
				return nil
			}
		}
	}
	res, err := ic.CreateDrac(ctx, &ufsAPI.CreateDracRequest{
		Drac:    &drac,
		DracId:  drac.GetName(),
		Machine: c.machineName,
	})
	if err != nil {
		return err
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	utils.PrintProtoJSON(res)
	fmt.Printf("Successfully added the drac %s to machine %s\n", res.Name, c.machineName)
	return nil
}

func (c *addDrac) parseArgs(drac *ufspb.Drac) {
	drac.Name = c.dracName
	drac.MacAddress = c.macAddress
	drac.SwitchInterface = &ufspb.SwitchInterface{
		Switch: c.switchName,
		Port:   int32(c.port),
	}
}

func (c *addDrac) validateArgs() error {
	if c.newSpecsFile != "" || c.interactive {
		if c.dracName != "" {
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
		if c.dracName == "" {
			return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nNo mode ('-f' or '-i') is specified, so '-name' is required.")
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
