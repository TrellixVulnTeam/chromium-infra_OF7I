// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package vm

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
	ufspb "infra/unifiedfleet/api/v1/proto"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	ufsUtil "infra/unifiedfleet/app/util"
)

// AddVMCmd add a vm on a host.
var AddVMCmd = &subcommands.Command{
	UsageLine: "add-vm [Options..]",
	ShortDesc: "Add a VM on a host",
	LongDesc: `Add a VM on a host

Examples:
shivas add-vm -f vm.json -h {Hostname}
Add a VM on a host by reading a JSON file input.`,
	CommandRun: func() subcommands.CommandRun {
		c := &addVM{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.Flags.StringVar(&c.newSpecsFile, "f", "", cmdhelp.VMFileText)
		c.Flags.StringVar(&c.hostname, "h", "", "hostname of the host to add the VM")
		return c
	},
}

type addVM struct {
	subcommands.CommandRunBase
	authFlags    authcli.Flags
	envFlags     site.EnvFlags
	hostname     string
	newSpecsFile string
}

func (c *addVM) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *addVM) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
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

	// Parse input json
	var vm ufspb.VM
	if err = utils.ParseJSONFile(c.newSpecsFile, &vm); err != nil {
		return err
	}

	// Get the host machineLSE
	machinelse, err := ic.GetMachineLSE(ctx, &ufsAPI.GetMachineLSERequest{
		Name: ufsUtil.AddPrefix(ufsUtil.MachineLSECollection, c.hostname),
	})
	if err != nil {
		return errors.Annotate(err, "No host with hostname %s found", c.hostname).Err()
	}
	machinelse.Name = ufsUtil.RemovePrefix(machinelse.Name)

	// Check if VM already exists on the host MachineLSE
	existingVMs := machinelse.GetChromeBrowserMachineLse().GetVms()
	if utils.CheckExistsVM(existingVMs, vm.Name) {
		return errors.New(fmt.Sprintf("VM %s already exists on the host %s", vm.Name, machinelse.Name))
	}
	existingVMs = append(existingVMs, &vm)
	machinelse.GetChromeBrowserMachineLse().Vms = existingVMs

	// Update the host MachineLSE with new VM
	machinelse.Name = ufsUtil.AddPrefix(ufsUtil.MachineLSECollection, machinelse.Name)
	res, err := ic.UpdateMachineLSE(ctx, &ufsAPI.UpdateMachineLSERequest{
		MachineLSE: machinelse,
	})
	if err != nil {
		return errors.Annotate(err, "Unable to add the VM on the host").Err()
	}
	utils.PrintProtoJSON(res)
	fmt.Println()
	return nil
}

func (c *addVM) validateArgs() error {
	if c.newSpecsFile == "" {
		return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nNo JSON input file specified")
	}
	if c.hostname == "" {
		return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nHostname parameter is required to add the VM on the host")
	}
	return nil
}
