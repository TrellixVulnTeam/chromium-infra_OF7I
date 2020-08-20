// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package kvm

import (
	"context"
	"fmt"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/grpc/prpc"

	"infra/cmd/shivas/site"
	"infra/cmd/shivas/utils"
	"infra/cmdsupport/cmdlib"
	ufspb "infra/unifiedfleet/api/v1/proto"
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
		c.commonFlags.Register(&c.Flags)
		c.outputFlags.Register(&c.Flags)
		return c
	},
}

type getKVM struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags
	outputFlags site.OutputFlags
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

	res, err := ic.GetKVM(ctx, &ufsAPI.GetKVMRequest{
		Name: ufsUtil.AddPrefix(ufsUtil.KVMCollection, args[0]),
	})
	if err != nil {
		return err
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	if utils.FullMode(c.outputFlags.Full()) {
		return c.printFull(ctx, ic, res)
	}
	return c.print(res)
}

func (c *getKVM) printFull(ctx context.Context, ic ufsAPI.FleetClient, kvm *ufspb.KVM) error {
	dhcp, _ := ic.GetDHCPConfig(ctx, &ufsAPI.GetDHCPConfigRequest{
		Hostname: kvm.GetName(),
	})
	if c.outputFlags.Tsv() {
		utils.PrintTSVKVMFull(kvm, dhcp)
		return nil
	}
	utils.PrintTitle(utils.KvmFullTitle)
	utils.PrintKVMFull(kvm, dhcp)
	return nil
}

func (c *getKVM) print(kvm *ufspb.KVM) error {
	if c.outputFlags.JSON() {
		utils.PrintProtoJSON(kvm, !c.outputFlags.NoEmit())
		return nil
	}
	if c.outputFlags.Tsv() {
		utils.PrintTSVKVMs([]*ufspb.KVM{kvm}, false)
		return nil
	}
	utils.PrintTitle(utils.KvmTitle)
	utils.PrintKVMs([]*ufspb.KVM{kvm}, false)
	return nil
}

func (c *getKVM) validateArgs() error {
	if c.Flags.NArg() == 0 {
		return cmdlib.NewUsageError(c.Flags, "Please provide the kvm name.")
	}
	return nil
}
