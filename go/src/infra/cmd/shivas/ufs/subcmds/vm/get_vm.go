// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package vm

import (
	"context"
	"fmt"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/grpc/prpc"

	"infra/cmd/shivas/site"
	"infra/cmd/shivas/utils"
	"infra/cmdsupport/cmdlib"
	ufspb "infra/unifiedfleet/api/v1/proto"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	ufsUtil "infra/unifiedfleet/app/util"
)

// GetVMCmd get VM by given name.
var GetVMCmd = &subcommands.Command{
	UsageLine: "vm -h {Hostname} {VM name}",
	ShortDesc: "Get VM details by name",
	LongDesc: `Get VM details by name.

Example:

shivas get vm -h {Hostname} {VM name}
Gets the vm and prints the output in JSON format.`,
	CommandRun: func() subcommands.CommandRun {
		c := &getVM{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.outputFlags.Register(&c.Flags)
		c.Flags.StringVar(&c.hostname, "host", "", "hostname of the host to get the VM")
		return c
	},
}

type getVM struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	outputFlags site.OutputFlags
	hostname    string
}

func (c *getVM) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *getVM) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	if err := c.validateArgs(); err != nil {
		return err
	}
	ctx := cli.GetContext(a, c, env)
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

	// Get the host machineLSE
	machinelse, err := ic.GetMachineLSE(ctx, &ufsAPI.GetMachineLSERequest{
		Name: ufsUtil.AddPrefix(ufsUtil.MachineLSECollection, c.hostname),
	})
	if err != nil {
		return errors.Annotate(err, "No host with hostname %s found", c.hostname).Err()
	}
	machinelse.Name = ufsUtil.RemovePrefix(machinelse.Name)

	// Check if VM exists on the host MachineLSE and print
	var vm *ufspb.VM
	existingVMs := machinelse.GetChromeBrowserMachineLse().GetVms()
	for _, v := range existingVMs {
		if v.Name == args[0] {
			vm = v
		}
	}
	if vm == nil {
		return errors.New(fmt.Sprintf("VM %s does not exist on the host %s", args[0], machinelse.Name))
	}
	if c.outputFlags.Full() {
		return c.printFull(ctx, ic, vm)
	}
	return c.print(vm)
}

func (c *getVM) printFull(ctx context.Context, ic ufsAPI.FleetClient, vm *ufspb.VM) error {
	dhcp, _ := ic.GetDHCPConfig(ctx, &ufsAPI.GetDHCPConfigRequest{
		Hostname: vm.GetName(),
	})
	s, _ := ic.GetState(ctx, &ufsAPI.GetStateRequest{
		ResourceName: ufsUtil.AddPrefix(ufsUtil.VMCollection, vm.GetName()),
	})
	if !c.outputFlags.Tsv() {
		utils.PrintTitle(utils.VMFullTitle)
	}
	utils.PrintVMFull(vm, dhcp, s)
	return nil
}

func (c *getVM) print(vm *ufspb.VM) error {
	if c.outputFlags.JSON() {
		utils.PrintProtoJSON(vm)
	} else {
		if !c.outputFlags.Tsv() {
			utils.PrintTitle(utils.VMTitle)
		}
		utils.PrintVMs([]*ufspb.VM{vm}, false)
	}
	return nil
}

func (c *getVM) validateArgs() error {
	if c.Flags.NArg() == 0 {
		return cmdlib.NewUsageError(c.Flags, "Please provide the VM name.")
	}
	if c.hostname == "" {
		return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nHostname parameter is required to get the VM on the host")
	}
	return nil
}
