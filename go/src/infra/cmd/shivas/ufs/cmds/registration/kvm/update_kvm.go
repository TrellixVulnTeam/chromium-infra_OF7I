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

	"infra/cmd/shivas/cmdhelp"
	"infra/cmd/shivas/site"
	"infra/cmd/shivas/utils"
	"infra/cmdsupport/cmdlib"
	ufspb "infra/unifiedfleet/api/v1/proto"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	ufsUtil "infra/unifiedfleet/app/util"
)

// UpdateKVMCmd Update kvm by given name.
var UpdateKVMCmd = &subcommands.Command{
	UsageLine: "update-kvm [Options...]",
	ShortDesc: "Update a kvm by name",
	LongDesc:  cmdhelp.UpdateKVMLongDesc,
	CommandRun: func() subcommands.CommandRun {
		c := &updateKVM{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.Flags.StringVar(&c.newSpecsFile, "f", "", cmdhelp.KVMFileText)
		c.Flags.BoolVar(&c.interactive, "i", false, "enable interactive mode for input")

		c.Flags.StringVar(&c.rackName, "rack", "", "name of the rack to associate the kvm")
		c.Flags.StringVar(&c.kvmName, "name", "", "the name of the kvm")
		c.Flags.StringVar(&c.vlanName, "vlan", "", "the vlan to assign the kvm to")
		c.Flags.BoolVar(&c.deleteVlan, "delete-vlan", false, "if deleting the ip assignment for the kvm")
		return c
	},
}

type updateKVM struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
	envFlags  site.EnvFlags

	newSpecsFile string
	interactive  bool

	rackName   string
	vlanName   string
	kvmName    string
	deleteVlan bool
}

func (c *updateKVM) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *updateKVM) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
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
	var kvm ufspb.KVM
	if c.interactive {
		c.rackName = utils.GetKVMInteractiveInput(ctx, ic, &kvm, true)
	} else {
		if c.newSpecsFile != "" {
			if err = utils.ParseJSONFile(c.newSpecsFile, &kvm); err != nil {
				return err
			}
		} else {
			c.parseArgs(&kvm)
		}
	}
	kvm.Name = ufsUtil.AddPrefix(ufsUtil.KVMCollection, kvm.Name)
	res, err := ic.UpdateKVM(ctx, &ufsAPI.UpdateKVMRequest{
		KVM:  &kvm,
		Rack: c.rackName,
		NetworkOption: &ufsAPI.NetworkOption{
			Vlan:   c.vlanName,
			Delete: c.deleteVlan,
		},
	})
	if err != nil {
		return err
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	utils.PrintProtoJSON(res)
	if c.deleteVlan {
		fmt.Printf("Successfully deleted vlan of kvm %s\n", res.Name)
	}
	if c.vlanName != "" {
		// Log the assigned IP
		if dhcp, err := ic.GetDHCPConfig(ctx, &ufsAPI.GetDHCPConfigRequest{
			Hostname: res.Name,
		}); err == nil {
			utils.PrintProtoJSON(dhcp)
			fmt.Println("Successfully added dhcp config to kvm: ", res.Name)
		}
	}
	return nil
}

func (c *updateKVM) parseArgs(kvm *ufspb.KVM) {
	kvm.Name = c.kvmName
}

func (c *updateKVM) validateArgs() error {
	if c.newSpecsFile != "" && c.interactive {
		return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nThe interactive & JSON mode cannot be specified at the same time.")
	}
	if c.newSpecsFile == "" && !c.interactive {
		if c.kvmName == "" {
			return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nNo mode ('-f' or '-i') is specified, so '-name' is required.")
		}
		if c.vlanName == "" && !c.deleteVlan {
			return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nNo mode ('-f' or '-i') is specified, so one of ['-delete-vlan', '-vlan'] is required.")
		}
	}
	return nil
}
