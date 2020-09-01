// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package kvm

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

// UpdateKVMCmd Update kvm by given name.
var UpdateKVMCmd = &subcommands.Command{
	UsageLine: "kvm [Options...]",
	ShortDesc: "Update a kvm on a rack",
	LongDesc:  cmdhelp.UpdateKVMLongDesc,
	CommandRun: func() subcommands.CommandRun {
		c := &updateKVM{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.commonFlags.Register(&c.Flags)

		c.Flags.StringVar(&c.newSpecsFile, "f", "", cmdhelp.KVMFileText)
		c.Flags.BoolVar(&c.interactive, "i", false, "enable interactive mode for input")

		c.Flags.StringVar(&c.vlanName, "vlan", "", "the vlan to assign the kvm to")
		c.Flags.BoolVar(&c.deleteVlan, "delete-vlan", false, "if deleting the ip assignment for the kvm")
		c.Flags.StringVar(&c.ip, "ip", "", "the ip to assign the kvm to")

		c.Flags.StringVar(&c.rackName, "rack", "", "name of the rack to associate the kvm.")
		c.Flags.StringVar(&c.kvmName, "name", "", "the name of the kvm to update")
		c.Flags.StringVar(&c.macAddress, "mac-address", "", "the mac address of the kvm to update")
		c.Flags.StringVar(&c.platform, "platform", "", "the platform of the kvm to update. "+cmdhelp.ClearFieldHelpText)
		c.Flags.StringVar(&c.tags, "tags", "", "comma separated tags. You can only append/add new tags here. "+cmdhelp.ClearFieldHelpText)
		c.Flags.StringVar(&c.description, "desc", "", "description for the kvm. "+cmdhelp.ClearFieldHelpText)
		c.Flags.StringVar(&c.state, "state", "", cmdhelp.StateHelp)

		return c
	},
}

type updateKVM struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags

	newSpecsFile string
	interactive  bool

	rackName    string
	vlanName    string
	kvmName     string
	deleteVlan  bool
	ip          string
	macAddress  string
	platform    string
	tags        string
	description string
	state       string
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
		utils.GetKVMInteractiveInput(ctx, ic, &kvm, true)
	} else {
		if c.newSpecsFile != "" {
			if err = utils.ParseJSONFile(c.newSpecsFile, &kvm); err != nil {
				return err
			}
			if kvm.GetRack() == "" {
				return errors.New(fmt.Sprintf("rack field is empty in json. It is a required parameter for json input."))
			}
		} else {
			c.parseArgs(&kvm)
		}
	}
	if err := utils.PrintExistingKVM(ctx, ic, kvm.Name); err != nil {
		return err
	}
	kvm.Name = ufsUtil.AddPrefix(ufsUtil.KVMCollection, kvm.Name)
	res, err := ic.UpdateKVM(ctx, &ufsAPI.UpdateKVMRequest{
		KVM: &kvm,
		NetworkOption: &ufsAPI.NetworkOption{
			Vlan:   c.vlanName,
			Delete: c.deleteVlan,
			Ip:     c.ip,
		},
		UpdateMask: utils.GetUpdateMask(&c.Flags, map[string]string{
			"rack":        "rack",
			"platform":    "platform",
			"mac-address": "macAddress",
			"tags":        "tags",
			"desc":        "description",
			"state":       "resourceState",
		}),
	})
	if err != nil {
		return err
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	fmt.Println("The kvm after update:")
	utils.PrintProtoJSON(res, !utils.NoEmitMode(false))
	if c.deleteVlan {
		fmt.Printf("Successfully deleted vlan of kvm %s\n", res.Name)
	}
	if c.vlanName != "" {
		// Log the assigned IP
		if dhcp, err := ic.GetDHCPConfig(ctx, &ufsAPI.GetDHCPConfigRequest{
			Hostname: res.Name,
		}); err == nil {
			utils.PrintProtoJSON(dhcp, false)
			fmt.Println("Successfully added dhcp config to kvm: ", res.Name)
		}
	}
	return nil
}

func (c *updateKVM) parseArgs(kvm *ufspb.KVM) {
	kvm.Name = c.kvmName
	kvm.Rack = c.rackName
	kvm.ResourceState = ufsUtil.ToUFSState(c.state)
	kvm.MacAddress = c.macAddress
	if c.platform == utils.ClearFieldValue {
		kvm.ChromePlatform = ""
	} else {
		kvm.ChromePlatform = c.platform
	}
	if c.tags == utils.ClearFieldValue {
		kvm.Tags = nil
	} else {
		kvm.Tags = utils.GetStringSlice(c.tags)
	}
	if c.description == utils.ClearFieldValue {
		kvm.Description = ""
	} else {
		kvm.Description = c.description
	}
}

func (c *updateKVM) validateArgs() error {
	if c.newSpecsFile != "" && c.interactive {
		return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe interactive & JSON mode cannot be specified at the same time.")
	}
	if c.newSpecsFile != "" || c.interactive {
		if c.kvmName != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe interactive/JSON mode is specified. '-name' cannot be specified at the same time.")
		}
		if c.platform != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe interactive/JSON mode is specified. '-platform' cannot be specified at the same time.")
		}
		if c.macAddress != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe interactive/JSON mode is specified. '-mac-address' cannot be specified at the same time.")
		}
		if c.tags != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe interactive/JSON mode is specified. '-tags' cannot be specified at the same time.")
		}
		if c.rackName != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe interactive/JSON mode is specified. '-rack' cannot be specified at the same time.")
		}
		if c.description != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe interactive/JSON mode is specified. '-desc' cannot be specified at the same time.")
		}
		if c.state != "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nThe JSON input file is already specified. '-state' cannot be specified at the same time.")
		}
	}
	if c.newSpecsFile == "" && !c.interactive {
		if c.kvmName == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n'-name' is required, no mode ('-f' or '-i') is specified")
		}
		if c.vlanName == "" && !c.deleteVlan && c.ip == "" &&
			c.rackName == "" && c.platform == "" && c.description == "" &&
			c.macAddress == "" && c.tags == "" && c.state == "" {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nNothing to update. Please provide any field to update")
		}
		if c.state != "" && !ufsUtil.IsUFSState(ufsUtil.RemoveStatePrefix(c.state)) {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n%s is not a valid state, please check help info for '-state'.", c.state)
		}
	}
	return nil
}
