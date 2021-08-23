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

	"infra/cmd/shivas/site"
	"infra/cmd/shivas/utils"
	"infra/cmdsupport/cmdlib"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	ufsUtil "infra/unifiedfleet/app/util"
)

// RenameNicCmd rename nic by given name.
var RenameNicCmd = &subcommands.Command{
	UsageLine: "nic ...",
	ShortDesc: "Rename nic with new name",
	LongDesc: `Rename nic with new name.

Example:

shivas rename nic -name {oldName} -new-name {newName}

Renames the nic and prints the output in the user-specified format.`,
	CommandRun: func() subcommands.CommandRun {
		c := &renameNic{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.commonFlags.Register(&c.Flags)
		c.outputFlags.Register(&c.Flags)

		c.Flags.StringVar(&c.name, "name", "", "the name of the nic to rename")
		c.Flags.StringVar(&c.newName, "new-name", "", "the new name of the nic")
		return c
	},
}

type renameNic struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags
	outputFlags site.OutputFlags

	name    string
	newName string
}

func (c *renameNic) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *renameNic) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
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
	if err := utils.PrintExistingNic(ctx, ic, c.name); err != nil {
		return err
	}
	res, err := ic.RenameNic(ctx, &ufsAPI.RenameNicRequest{
		Name:    ufsUtil.AddPrefix(ufsUtil.NicCollection, c.name),
		NewName: ufsUtil.AddPrefix(ufsUtil.NicCollection, c.newName),
	})
	if err != nil {
		return err
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	fmt.Println("The nic after renaming:")
	utils.PrintProtoJSON(res, !utils.NoEmitMode(false))
	fmt.Println("Successfully renamed the nic to: ", res.GetName())
	return nil
}

func (c *renameNic) validateArgs() error {
	if c.name == "" {
		return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n'-name' is required")
	}
	if c.newName == "" {
		return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n'-new-name' is required")
	}
	return nil
}
