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
		c.Flags.StringVar(&c.machineName, "m", "", "name of the machine to update the association of the drac to a different machine")
		c.Flags.BoolVar(&c.interactive, "i", false, "enable interactive mode for input")
		return c
	},
}

type updateDrac struct {
	subcommands.CommandRunBase
	authFlags    authcli.Flags
	envFlags     site.EnvFlags
	machineName  string
	newSpecsFile string
	interactive  bool
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
		err = utils.ParseJSONFile(c.newSpecsFile, &drac)
		if err != nil {
			return err
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
	})
	if err != nil {
		return err
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	utils.PrintProtoJSON(res)
	fmt.Println()
	return nil
}

func (c *updateDrac) validateArgs() error {
	if !c.interactive && c.newSpecsFile == "" {
		return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nNeither JSON input file specified nor in interactive mode to accept input.")
	}
	return nil
}
