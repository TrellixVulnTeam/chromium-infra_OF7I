// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package kvm

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

// GetKVMCmd get kvm by given name.
var GetKVMCmd = &subcommands.Command{
	UsageLine: "kvm {KVM Name}",
	ShortDesc: "Get kvm details by name",
	LongDesc: `Get kvm details by name.

Example:

shivas get kvm {KVM Name}
Gets the kvm and prints the output in JSON format.`,
	CommandRun: func() subcommands.CommandRun {
		c := &getKVM{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		return c
	},
}

type getKVM struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
	envFlags  site.EnvFlags
}

func (c *getKVM) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *getKVM) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
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
	res, err := ic.GetKVM(ctx, &ufsAPI.GetKVMRequest{
		Name: ufsUtil.AddPrefix(ufsUtil.KVMCollection, args[0]),
	})
	if err != nil {
		return err
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	utils.PrintProtoJSON(res)
	fmt.Println()
	return nil
}

func (c *getKVM) validateArgs() error {
	if c.Flags.NArg() == 0 {
		return cmdlib.NewUsageError(c.Flags, "Please provide the kvm name.")
	}
	return nil
}
