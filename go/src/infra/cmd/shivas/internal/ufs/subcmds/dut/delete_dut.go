// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dut

import (
	"context"
	"fmt"
	"os"

	"github.com/golang/protobuf/proto"
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

// DeleteDUTCmd delete a dut on a machine.
var DeleteDUTCmd = &subcommands.Command{
	UsageLine: "dut {Hostname}...",
	ShortDesc: "Delete a DUT/labstation",
	LongDesc: `Delete a DUT/labstation.

Example:
shivas delete dut {Hostname}

shivas delete dut {Hostname1} {Hostname2}

Deletes the DUT(s).`,
	CommandRun: func() subcommands.CommandRun {
		c := &deleteDUT{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		return c
	},
}

type deleteDUT struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
	envFlags  site.EnvFlags
}

func (c *deleteDUT) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *deleteDUT) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	if err := c.validateArgs(); err != nil {
		return err
	}
	ctx := cli.GetContext(a, c, env)
	ctx = utils.SetupContext(ctx, ufsUtil.OSNamespace)
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
	prompt := utils.CLIPrompt(a.GetOut(), os.Stdin, false)
	if prompt != nil && !prompt(fmt.Sprintf("Are you sure you want to delete the dut(s): %s", args)) {
		return nil
	}
	duts := utils.ConcurrentGet(ctx, ic, args, c.getSingle)
	fmt.Fprintln(a.GetOut(), "\nDUT(s) before deletion:")
	utils.PrintMachineLSEsJSON(duts, true)
	pass, fail := utils.ConcurrentDelete(ctx, ic, args, c.deleteSingle)
	fmt.Fprintln(a.GetOut(), "\nSuccessfully deleted DUT(s):")
	fmt.Fprintln(a.GetOut(), pass)
	fmt.Fprintln(a.GetOut(), "\nFailed to delete DUT(s):")
	fmt.Fprintln(a.GetOut(), fail)
	return nil
}

func (c *deleteDUT) getSingle(ctx context.Context, ic ufsAPI.FleetClient, name string) (proto.Message, error) {
	return ic.GetMachineLSE(ctx, &ufsAPI.GetMachineLSERequest{
		Name: ufsUtil.AddPrefix(ufsUtil.MachineLSECollection, name),
	})
}

func (c *deleteDUT) deleteSingle(ctx context.Context, ic ufsAPI.FleetClient, name string) error {
	_, err := ic.DeleteMachineLSE(ctx, &ufsAPI.DeleteMachineLSERequest{
		Name: ufsUtil.AddPrefix(ufsUtil.MachineLSECollection, name),
	})
	return err
}

func (c *deleteDUT) validateArgs() error {
	if c.Flags.NArg() == 0 {
		return cmdlib.NewUsageError(c.Flags, "Please provide the hostname of the dut to delete.")
	}
	return nil
}
