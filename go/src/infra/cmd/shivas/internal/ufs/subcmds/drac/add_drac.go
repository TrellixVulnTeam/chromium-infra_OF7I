// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package drac

import (
	"fmt"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/grpc/prpc"

	"infra/cmd/shivas/cmdhelp"
	"infra/cmd/shivas/site"
	"infra/cmd/shivas/utils"
	"infra/cmdsupport/cmdlib"
	ufspb "infra/unifiedfleet/api/v1/models"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	ufsUtil "infra/unifiedfleet/app/util"
)

// AddDracCmd add Drac in the lab.
var AddDracCmd = &subcommands.Command{
	UsageLine: "drac [Options...]",
	ShortDesc: "Add a drac to a machine",
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
		c.Flags.StringVar(&c.macAddress, "mac", "", "the mac address of the drac to add")
		c.Flags.StringVar(&c.switchName, "switch", "", "the name of the switch that this drac is connected to")
		c.Flags.StringVar(&c.switchPort, "switch-port", "", "the port of the switch that this drac is connected to")
		c.Flags.StringVar(&c.tags, "tags", "", "comma separated tags. You can only append/add new tags here.")
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
	switchPort  string
	tags        string
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

	var drac ufspb.Drac
	if c.interactive {
		utils.GetDracInteractiveInput(ctx, ic, &drac, false)
	} else {
		if c.newSpecsFile != "" {
			if err := utils.ParseJSONFile(c.newSpecsFile, &drac); err != nil {
				return err
			}
			if drac.GetMachine() == "" {
				return errors.New(fmt.Sprintf("machine field is empty in json. It is a required parameter for json input."))
			}
		} else {
			c.parseArgs(&drac)
		}
	}
	if !ufsUtil.ValidateTags(drac.Tags) {
		return fmt.Errorf(ufsAPI.InvalidTags)
	}
	res, err := ic.CreateDrac(ctx, &ufsAPI.CreateDracRequest{
		Drac:   &drac,
		DracId: drac.GetName(),
	})
	if err != nil {
		return err
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	utils.PrintProtoJSON(res, !utils.NoEmitMode(false))
	fmt.Printf("Successfully added the drac %s to machine %s\n", res.Name, res.GetMachine())
	return nil
}

func (c *addDrac) parseArgs(drac *ufspb.Drac) {
	drac.Name = c.dracName
	drac.Machine = c.machineName
	drac.MacAddress = c.macAddress
	drac.SwitchInterface = &ufspb.SwitchInterface{
		Switch:   c.switchName,
		PortName: c.switchPort,
	}
	drac.Tags = utils.GetStringSlice(c.tags)
}

func (c *addDrac) validateArgs() error {
	if c.newSpecsFile != "" && c.interactive {
		return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe interactive & JSON mode cannot be specified at the same time.")
	}
	if c.newSpecsFile != "" || c.interactive {
		if c.dracName != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe interactive/JSON mode is specified. '-name' cannot be specified at the same time.")
		}
		if c.switchName != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe interactive/JSON mode is specified. '-switch' cannot be specified at the same time.")
		}
		if c.switchPort != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe interactive/JSON mode is specified. '-switch-port' cannot be specified at the same time.")
		}
		if c.macAddress != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe interactive/JSON mode is specified. '-mac' cannot be specified at the same time.")
		}
		if c.tags != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe interactive/JSON mode is specified. '-tags' cannot be specified at the same time.")
		}
		if c.machineName != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe interactive/JSON mode is specified. '-machine' cannot be specified at the same time.")
		}
	}
	if c.newSpecsFile == "" && !c.interactive {
		if c.dracName == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n'-name' is required, no mode ('-f' or '-i') is specified.")
		}
		if c.macAddress == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n'-mac' is required, no mode ('-f' or '-i') is specified.")
		}
		if c.machineName == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nMachine name (-machine) is required.")
		}
	}
	return nil
}
