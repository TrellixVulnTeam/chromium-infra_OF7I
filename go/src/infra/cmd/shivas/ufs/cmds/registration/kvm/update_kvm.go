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
		c.Flags.StringVar(&c.rackName, "r", "", "name of the rack to associate the kvm")
		c.Flags.BoolVar(&c.interactive, "i", false, "enable interactive mode for input")
		return c
	},
}

type updateKVM struct {
	subcommands.CommandRunBase
	authFlags    authcli.Flags
	envFlags     site.EnvFlags
	rackName     string
	newSpecsFile string
	interactive  bool
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
		if err = utils.ParseJSONFile(c.newSpecsFile, &kvm); err != nil {
			return err
		}
	}
	kvm.Name = ufsUtil.AddPrefix(ufsUtil.KVMCollection, kvm.Name)
	res, err := ic.UpdateKVM(ctx, &ufsAPI.UpdateKVMRequest{
		KVM:  &kvm,
		Rack: c.rackName,
	})
	if err != nil {
		return err
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	utils.PrintProtoJSON(res)
	fmt.Println()
	return nil
}

func (c *updateKVM) validateArgs() error {
	if !c.interactive && c.newSpecsFile == "" {
		return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nNeither JSON input file specified nor in interactive mode to accept input.")
	}
	return nil
}
