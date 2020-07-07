// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package labsetup

import (
	"fmt"
	"os"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/grpc/prpc"

	"infra/cmd/shivas/site"
	"infra/cmd/shivas/utils"
	"infra/cmdsupport/cmdlib"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	ufsUtil "infra/unifiedfleet/app/util"
)

// AbandonMachineCmd deletes machinelse by hostname.
var AbandonMachineCmd = &subcommands.Command{
	UsageLine: "abandon-machine {Machine Hostname}",
	ShortDesc: "Abandon/Delete a deployed machine in the lab",
	LongDesc: `Abandon/Delete a deployed machine(DUT, Labstation, DevServer, Caching Server or a VM Server) in the lab.

Example:
shivas abandon-machine {Machine Hostname}
Deletes the deployed machine.`,
	CommandRun: func() subcommands.CommandRun {
		c := &abandonMachine{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		return c
	},
}

type abandonMachine struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
	envFlags  site.EnvFlags
}

func (c *abandonMachine) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *abandonMachine) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	if err := c.validateArgs(); err != nil {
		return err
	}
	ctx := cli.GetContext(a, c, env)
	ctx = utils.SetupContext(ctx)
	hc, err := cmdlib.NewHTTPClient(ctx, &c.authFlags)
	if err != nil {
		return err
	}
	prompt := utils.CLIPrompt(a.GetOut(), os.Stdin, false)
	if !prompt(fmt.Sprintf("Are you sure you want to delete the deployed machine: %s", args[0])) {
		return nil
	}
	e := c.envFlags.Env()
	fmt.Printf("Using UnifiedFleet service %s\n", e.UnifiedFleetService)
	ic := ufsAPI.NewFleetPRPCClient(&prpc.Client{
		C:       hc,
		Host:    e.UnifiedFleetService,
		Options: site.DefaultPRPCOptions,
	})
	_, err = ic.DeleteMachineLSE(ctx, &ufsAPI.DeleteMachineLSERequest{
		Name: ufsUtil.AddPrefix(ufsUtil.MachineLSECollection, args[0]),
	})
	if err == nil {
		fmt.Fprintln(a.GetOut(), args[0], "deleted successfully.")
		return nil
	}
	return err
}

func (c *abandonMachine) validateArgs() error {
	if c.Flags.NArg() == 0 {
		return cmdlib.NewUsageError(c.Flags, "Please provide the hostname of the deployed machine to delete.")
	}
	return nil
}
