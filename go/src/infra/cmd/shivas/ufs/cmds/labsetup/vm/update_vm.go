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

// UpdateVMCmd update VM on a host.
var UpdateVMCmd = &subcommands.Command{
	UsageLine: "update-vm [Options...]",
	ShortDesc: "Update a VM on a host",
	LongDesc: `Update a VM on a host

Examples:
shivas update-vm -f vm.json -h {Hostname}
Update a VM on a host by reading a JSON file input.`,
	CommandRun: func() subcommands.CommandRun {
		c := &updateVM{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.Flags.StringVar(&c.newSpecsFile, "f", "", cmdhelp.VMFileText)

		c.Flags.StringVar(&c.hostName, "host", "", "name of the vm")
		c.Flags.StringVar(&c.vmName, "name", "", "name of the host that this VM is running on")
		c.Flags.StringVar(&c.vlanName, "vlan", "", "name of the vlan to assign this vm to")
		c.Flags.BoolVar(&c.deleteVlan, "delete-vlan", false, "if deleting the ip assignment for the vm")
		c.Flags.StringVar(&c.ip, "ip", "", "the ip to assign the vm to")
		c.Flags.StringVar(&c.state, "state", "", cmdhelp.StateHelp)
		return c
	},
}

type updateVM struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
	envFlags  site.EnvFlags

	newSpecsFile string

	hostName   string
	vmName     string
	vlanName   string
	deleteVlan bool
	ip         string
	state      string
}

func (c *updateVM) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *updateVM) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
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

	// Parse the josn input
	var vm ufspb.VM
	if c.newSpecsFile != "" {
		if err = utils.ParseJSONFile(c.newSpecsFile, &vm); err != nil {
			return err
		}
	} else {
		c.parseArgs(&vm)
	}

	var nwOpt *ufsAPI.NetworkOption
	if c.deleteVlan || c.vlanName != "" || c.ip != "" {
		nwOpt = &ufsAPI.NetworkOption{
			Delete: c.deleteVlan,
			Vlan:   c.vlanName,
			Ip:     c.ip,
		}
	}
	var s ufspb.State
	if c.state != "" {
		s = ufsUtil.ToUFSState(c.state)
	}
	vm.Name = ufsUtil.AddPrefix(ufsUtil.VMCollection, vm.Name)
	res, err := ic.UpdateVM(ctx, &ufsAPI.UpdateVMRequest{
		Vm:            &vm,
		MachineLSEId:  c.hostName,
		NetworkOption: nwOpt,
		State:         s,
	})
	if err != nil {
		return errors.Annotate(err, "Unable to update the VM on the host").Err()
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	utils.PrintProtoJSON(res)
	if c.deleteVlan {
		fmt.Printf("Successfully deleted vlan of vm %s\n", vm.Name)
	}
	if c.vlanName != "" || c.ip != "" {
		// Log the assigned IP
		if dhcp, err := ic.GetDHCPConfig(ctx, &ufsAPI.GetDHCPConfigRequest{
			Hostname: vm.Name,
		}); err == nil {
			utils.PrintProtoJSON(dhcp)
			fmt.Println("Successfully added dhcp config to vm: ", vm.Name)
		}
	}
	return nil
}

func (c *updateVM) parseArgs(vm *ufspb.VM) {
	vm.Name = c.vmName
	vm.Hostname = c.vmName
}

func (c *updateVM) validateArgs() error {
	if c.hostName == "" {
		return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n'-host' is required to update the VM on a host")
	}
	if c.newSpecsFile == "" {
		if c.vmName == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n'-name' is required, no mode ('-f') is specified.")
		}
		if c.vlanName == "" && !c.deleteVlan && c.ip == "" && c.state == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\none of ['-delete-vlan', '-vlan', '-ip', '-state'] is required, no mode ('-f') is specified.")
		}
	}
	if c.state != "" && !ufsUtil.IsUFSState(c.state) {
		return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n%s is not a valid state, please check help info for '-state'.", c.state)
	}
	return nil
}
