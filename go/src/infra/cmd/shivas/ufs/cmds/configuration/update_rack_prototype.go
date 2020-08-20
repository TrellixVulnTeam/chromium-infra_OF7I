// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package configuration

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

// UpdateRackLSEPrototypeCmd update RackLSEPrototype by given name.
var UpdateRackLSEPrototypeCmd = &subcommands.Command{
	UsageLine: "update-rack-prototype",
	ShortDesc: "Update prototype for rack deployment",
	LongDesc:  cmdhelp.UpdateRackLSEPrototypeLongDesc,
	CommandRun: func() subcommands.CommandRun {
		c := &updateRackLSEPrototype{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.Flags.StringVar(&c.newSpecsFile, "f", "", cmdhelp.RackLSEPrototypeFileText)
		c.Flags.BoolVar(&c.interactive, "i", false, "enable interactive mode for input")
		return c
	},
}

type updateRackLSEPrototype struct {
	subcommands.CommandRunBase
	authFlags    authcli.Flags
	envFlags     site.EnvFlags
	newSpecsFile string
	interactive  bool
}

func (c *updateRackLSEPrototype) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *updateRackLSEPrototype) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
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
	fmt.Printf("Using UnifiedFleet service %s\n", e.UnifiedFleetService)
	ic := ufsAPI.NewFleetPRPCClient(&prpc.Client{
		C:       hc,
		Host:    e.UnifiedFleetService,
		Options: site.DefaultPRPCOptions,
	})
	var racklsePrototype ufspb.RackLSEPrototype
	if c.interactive {
		utils.GetRacklsePrototypeInteractiveInput(ctx, ic, &racklsePrototype, true)
	} else {
		err = utils.ParseJSONFile(c.newSpecsFile, &racklsePrototype)
		if err != nil {
			return err
		}
	}
	if err := utils.PrintExistingRackPrototype(ctx, ic, racklsePrototype.Name); err != nil {
		return err
	}
	racklsePrototype.Name = ufsUtil.AddPrefix(ufsUtil.RackLSEPrototypeCollection, racklsePrototype.Name)
	res, err := ic.UpdateRackLSEPrototype(ctx, &ufsAPI.UpdateRackLSEPrototypeRequest{
		RackLSEPrototype: &racklsePrototype,
	})
	if err != nil {
		return err
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	fmt.Println("The rack lse prototype after update:")
	utils.PrintProtoJSON(res, false)
	fmt.Println()
	return nil
}

func (c *updateRackLSEPrototype) validateArgs() error {
	if !c.interactive && c.newSpecsFile == "" {
		return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nNeither JSON input file specified nor in interactive mode to accept input.")
	}
	return nil
}
