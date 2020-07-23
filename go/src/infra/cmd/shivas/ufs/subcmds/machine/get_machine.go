// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package machine

import (
	"fmt"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/grpc/prpc"

	"infra/cmd/shivas/site"
	"infra/cmd/shivas/utils"
	"infra/cmdsupport/cmdlib"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	ufsUtil "infra/unifiedfleet/app/util"
)

// GetMachineCmd get machine by given name.
var GetMachineCmd = &subcommands.Command{
	UsageLine: "machine {Machine Name/Hostname}",
	ShortDesc: "Get machine details by name or get deployed machine details by hostname",
	LongDesc: `Get machine details by name or get deployed machine details by hostname.

Example:

shivas get machine {Machine Name}
Gets the machine and prints the output in JSON format.

shivas get machine {Machine Hostname}
Gets the deployed machine and prints the output in JSON format.`,
	CommandRun: func() subcommands.CommandRun {
		c := &getMachine{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.commonFlags.Register(&c.Flags)
		return c
	},
}

type getMachine struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags
}

func (c *getMachine) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *getMachine) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
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
	if c.commonFlags.Verbose() {
		fmt.Printf("Using UnifiedFleet service %s\n", e.UnifiedFleetService)
	}
	ic := ufsAPI.NewFleetPRPCClient(&prpc.Client{
		C:       hc,
		Host:    e.UnifiedFleetService,
		Options: site.DefaultPRPCOptions,
	})
	machineRes, err := ic.GetMachine(ctx, &ufsAPI.GetMachineRequest{
		Name: ufsUtil.AddPrefix(ufsUtil.MachineCollection, args[0]),
	})
	if err != nil {
		// If get Machine fails, check for get MachineLSE
		machinelseRes, err := ic.GetMachineLSE(ctx, &ufsAPI.GetMachineLSERequest{
			Name: ufsUtil.AddPrefix(ufsUtil.MachineLSECollection, args[0]),
		})
		if err != nil {
			return errors.Annotate(err, "Given argument is neither a machine name nor a machine hostname").Err()
		}
		machinelseRes.Name = ufsUtil.RemovePrefix(machinelseRes.Name)
		utils.PrintProtoJSON(machinelseRes)
		fmt.Println()
		return nil
	}
	machineRes.Name = ufsUtil.RemovePrefix(machineRes.Name)
	utils.PrintProtoJSON(machineRes)
	fmt.Println()
	return nil
}

func (c *getMachine) validateArgs() error {
	if c.Flags.NArg() == 0 {
		return cmdlib.NewUsageError(c.Flags, "Please provide the machine name or deployed machine hostname.")
	}
	return nil
}
