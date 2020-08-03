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

// UpdateDracCmd Update drac by given name.
var UpdateDracCmd = &subcommands.Command{
	UsageLine: "update-drac [Options...]",
	ShortDesc: "Update a drac by name",
	LongDesc:  cmdhelp.UpdateDracLongDesc,
	CommandRun: func() subcommands.CommandRun {
		c := &updateDrac{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.Flags.StringVar(&c.newSpecsFile, "f", "", cmdhelp.DracFileText)
		c.Flags.BoolVar(&c.interactive, "i", false, "enable interactive mode for input")

		c.Flags.StringVar(&c.machineName, "machine", "", "name of the machine to update the association of the drac to a different machine")
		c.Flags.StringVar(&c.dracName, "name", "", "name of the drac to update")
		c.Flags.StringVar(&c.vlanName, "vlan", "", "the vlan to assign the drac to")
		c.Flags.BoolVar(&c.deleteVlan, "delete-vlan", false, "if deleting the ip assignment for the drac")
		c.Flags.StringVar(&c.ip, "ip", "", "the ip to assign the drac to")
		return c
	},
}

type updateDrac struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
	envFlags  site.EnvFlags

	newSpecsFile string
	interactive  bool

	machineName string
	vlanName    string
	dracName    string
	deleteVlan  bool
	ip          string
}

func (c *updateDrac) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *updateDrac) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	if err := c.validateArgs(); err != nil {
		return err
	}
	ctx := cli.GetContext(a, c, env)
	ctx = utils.SetupContext(ctx)
	hc, err := cmdlib.NewHTTPClient(ctx, &c.authFlags)
	if err != nil {
		return err
	}
	e := c.envFlags.Env()
	ic := ufsAPI.NewFleetPRPCClient(&prpc.Client{
		C:       hc,
		Host:    e.UnifiedFleetService,
		Options: site.DefaultPRPCOptions,
	})
	var drac ufspb.Drac
	if c.interactive {
		c.machineName = utils.GetDracInteractiveInput(ctx, ic, &drac, true)
	} else {
		if c.newSpecsFile != "" {
			err = utils.ParseJSONFile(c.newSpecsFile, &drac)
			if err != nil {
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
				"Are you sure you want to update the drac to a different machine?",
				c.machineName, oldDracName, drac.Name, c.machineName, oldDracName)) {
				return nil
			}
		}
	}
	drac.Name = ufsUtil.AddPrefix(ufsUtil.DracCollection, drac.Name)
	res, err := ic.UpdateDrac(ctx, &ufsAPI.UpdateDracRequest{
		Drac:    &drac,
		Machine: c.machineName,
		NetworkOption: &ufsAPI.NetworkOption{
			Vlan:   c.vlanName,
			Delete: c.deleteVlan,
			Ip:     c.ip,
		},
	})
	if err != nil {
		return err
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	utils.PrintProtoJSON(res)
	if c.deleteVlan {
		fmt.Printf("Successfully deleted vlan of drac %s\n", res.Name)
	}
	if c.vlanName != "" {
		// Log the assigned IP
		if dhcp, err := ic.GetDHCPConfig(ctx, &ufsAPI.GetDHCPConfigRequest{
			Hostname: res.Name,
		}); err == nil {
			utils.PrintProtoJSON(dhcp)
			fmt.Println("Successfully added dhcp config to drac: ", res.Name)
		}
	}
	return nil
}

func (c *updateDrac) parseArgs(drac *ufspb.Drac) {
	drac.Name = c.dracName
}

func (c *updateDrac) validateArgs() error {
	if c.newSpecsFile != "" {
		if c.interactive {
			return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nThe interactive & JSON mode cannot be specified at the same time.")
		}
	}
	if c.newSpecsFile == "" && !c.interactive {
		if c.dracName == "" {
			return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nNo mode ('-f' or '-i') is specified, so '-name' is required.")
		}
		if c.vlanName == "" && !c.deleteVlan && c.ip == "" {
			return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nNo mode ('-f' or '-i') is specified, so one of ['-delete-vlan', '-vlan', '-ip'] is required.")
		}
	}
	return nil
}
