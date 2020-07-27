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

// AddKVMCmd add KVM in the lab.
var AddKVMCmd = &subcommands.Command{
	UsageLine: "add-kvm [Options...]",
	ShortDesc: "Add a kvm to UFS",
	LongDesc:  cmdhelp.AddKVMLongDesc,
	CommandRun: func() subcommands.CommandRun {
		c := &addKVM{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.commonFlags.Register(&c.Flags)
		c.Flags.StringVar(&c.newSpecsFile, "f", "", cmdhelp.KVMFileText)
		c.Flags.BoolVar(&c.interactive, "i", false, "enable interactive mode for input")

		c.Flags.StringVar(&c.rackName, "rack", "", "name of the rack to associate the kvm")
		c.Flags.StringVar(&c.kvmName, "name", "", "the name of the kvm to add")
		c.Flags.StringVar(&c.macAddress, "mac-address", "", "the name of the kvm to add")
		c.Flags.StringVar(&c.platform, "platform", "", "the name of the kvm to add")
		return c
	},
}

type addKVM struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags

	newSpecsFile string
	interactive  bool

	rackName   string
	kvmName    string
	macAddress string
	platform   string
}

func (c *addKVM) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *addKVM) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
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
		fmt.Printf("Using UFS service %s\n", e.UnifiedFleetService)
	}
	ic := ufsAPI.NewFleetPRPCClient(&prpc.Client{
		C:       hc,
		Host:    e.UnifiedFleetService,
		Options: site.DefaultPRPCOptions,
	})

	var kvm ufspb.KVM
	if c.interactive {
		c.rackName = utils.GetKVMInteractiveInput(ctx, ic, &kvm, false)
	} else {
		if c.newSpecsFile != "" {
			if err = utils.ParseJSONFile(c.newSpecsFile, &kvm); err != nil {
				return err
			}
		} else {
			c.parseArgs(&kvm)
		}
	}
	res, err := ic.CreateKVM(ctx, &ufsAPI.CreateKVMRequest{
		KVM:   &kvm,
		KVMId: kvm.GetName(),
		Rack:  c.rackName,
	})
	if err != nil {
		return err
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	utils.PrintProtoJSON(res)
	fmt.Printf("Successfully added the kvm %s to rack %s\n", res.Name, c.rackName)
	return nil
}

func (c *addKVM) parseArgs(kvm *ufspb.KVM) {
	kvm.Name = c.kvmName
	kvm.ChromePlatform = c.platform
	kvm.MacAddress = c.macAddress
}

func (c *addKVM) validateArgs() error {
	if c.newSpecsFile != "" || c.interactive {
		if c.kvmName != "" {
			return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nThe interactive/JSON mode is specified. '-name' cannot be specified at the same time.")
		}
		if c.platform != "" {
			return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nThe interactive/JSON mode is specified. '-platform' cannot be specified at the same time.")
		}
		if c.macAddress != "" {
			return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nThe interactive/JSON mode is specified. '-mac-address' cannot be specified at the same time.")
		}
	}
	if c.newSpecsFile != "" {
		if c.interactive {
			return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nThe interactive & JSON mode cannot be specified at the same time.")
		}
		if c.rackName == "" {
			return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nRack name (-rack) is required for JSON mode.")
		}
	}
	if c.newSpecsFile == "" && !c.interactive {
		if c.kvmName == "" {
			return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nNo mode ('-f' or '-i') is specified, so '-name' is required.")
		}
		if c.platform == "" {
			return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nNo mode ('-f' or '-i') is specified, so '-platform' is required.")
		}
		if c.macAddress == "" {
			return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nNo mode ('-f' or '-i') is specified, so '-mac-address' is required.")
		}
		if c.rackName == "" {
			return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nRack name (-rack) is required.")
		}
	}
	return nil
}
