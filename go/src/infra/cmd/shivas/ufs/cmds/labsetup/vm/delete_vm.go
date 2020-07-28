// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package vm

import (
	"fmt"
	"os"

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

// DeleteVMCmd deletes vm on a host.
var DeleteVMCmd = &subcommands.Command{
	UsageLine: "delete-vm -host {Hostname} {VM name}",
	ShortDesc: "Delete a VM on a host",
	LongDesc: `Delete a VM on a host.

Example:
shivas delete-vm -host {Hostname} {VM name}
Deletes the VM on a host.`,
	CommandRun: func() subcommands.CommandRun {
		c := &deleteVM{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.Flags.StringVar(&c.hostname, "host", "", "hostname of the host to delete the VM")
		return c
	},
}

type deleteVM struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
	envFlags  site.EnvFlags
	hostname  string
}

func (c *deleteVM) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *deleteVM) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	if err := c.validateArgs(); err != nil {
		return err
	}
	ctx := cli.GetContext(a, c, env)
	hc, err := cmdlib.NewHTTPClient(ctx, &c.authFlags)
	if err != nil {
		return err
	}
	prompt := utils.CLIPrompt(a.GetOut(), os.Stdin, false)
	if !prompt(fmt.Sprintf("Are you sure you want to delete the VM: %s", args[0])) {
		return nil
	}
	e := c.envFlags.Env()
	ic := ufsAPI.NewFleetPRPCClient(&prpc.Client{
		C:       hc,
		Host:    e.UnifiedFleetService,
		Options: site.DefaultPRPCOptions,
	})

	// Get the host MachineLSE
	machinelse, err := ic.GetMachineLSE(ctx, &ufsAPI.GetMachineLSERequest{
		Name: ufsUtil.AddPrefix(ufsUtil.MachineLSECollection, c.hostname),
	})
	if err != nil {
		return errors.Annotate(err, "No host with hostname %s found", c.hostname).Err()
	}
	machinelse.Name = ufsUtil.RemovePrefix(machinelse.Name)

	// Check if the VM does not exist on the host MachineLSE
	existingVMs := machinelse.GetChromeBrowserMachineLse().GetVms()
	if !utils.CheckExistsVM(existingVMs, args[0]) {
		return errors.New(fmt.Sprintf("VM %s does not exist on the host %s", args[0], machinelse.Name))
	}
	existingVMs = utils.RemoveVM(existingVMs, args[0])
	machinelse.GetChromeBrowserMachineLse().Vms = existingVMs

	// Update the host MachineLSE host
	machinelse.Name = ufsUtil.AddPrefix(ufsUtil.MachineLSECollection, machinelse.Name)
	res, err := ic.UpdateMachineLSE(ctx, &ufsAPI.UpdateMachineLSERequest{
		MachineLSE: machinelse,
	})
	if err == nil {
		fmt.Fprintf(a.GetOut(), "Updated host after deleting vm %s:\n", args[0])
		utils.PrintProtoJSON(res)
		fmt.Fprintln(a.GetOut(), args[0], "is deleted successfully.")
		return nil
	}
	return errors.Annotate(err, "Unable to delete the VM on the host").Err()
}

func (c *deleteVM) validateArgs() error {
	if c.Flags.NArg() == 0 {
		return cmdlib.NewUsageError(c.Flags, "Please provide the name of the VM to delete.")
	}
	if c.hostname == "" {
		return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\n'-host' is required to delete the VM on the host")
	}
	return nil
}
