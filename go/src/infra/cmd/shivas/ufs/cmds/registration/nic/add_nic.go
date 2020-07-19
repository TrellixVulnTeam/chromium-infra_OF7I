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
		c.Flags.StringVar(&c.newSpecsFile, "f", "", cmdhelp.NicFileText)
		c.Flags.StringVar(&c.machineName, "m", "", "name of the machine to associate the nic")
		c.Flags.BoolVar(&c.interactive, "i", false, "enable interactive mode for input")
		return c
	},
}

type addNic struct {
	subcommands.CommandRunBase
	authFlags    authcli.Flags
	envFlags     site.EnvFlags
	machineName  string
	newSpecsFile string
	interactive  bool
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
	var nic ufspb.Nic
	if c.interactive {
		c.machineName = utils.GetNicInteractiveInput(ctx, ic, &nic, false)
	} else {
		if err = utils.ParseJSONFile(c.newSpecsFile, &nic); err != nil {
			return err
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
	fmt.Println()
	return nil
}

func (c *addNic) validateArgs() error {
	if !c.interactive {
		if c.newSpecsFile == "" {
			return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nNeither JSON input file specified nor in interactive mode to accept input.")
		}
		if c.machineName == "" {
			return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nMachine name parameter is required to associate the nic with a machine.")
		}
	}
	return nil
}
