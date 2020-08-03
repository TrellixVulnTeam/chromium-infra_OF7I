// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package vm

import (
	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/grpc/prpc"

	"infra/cmd/shivas/cmdhelp"
	"infra/cmd/shivas/site"
	"infra/cmd/shivas/utils"
	"infra/cmdsupport/cmdlib"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	ufsUtil "infra/unifiedfleet/app/util"
)

// ListVMCmd list all VMs on a host.
var ListVMCmd = &subcommands.Command{
	UsageLine: "vm -h {Hostname}",
	ShortDesc: "List all VMs on a host",
	LongDesc:  cmdhelp.ListVMLongDesc,
	CommandRun: func() subcommands.CommandRun {
		c := &listVM{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.outputFlags.Register(&c.Flags)
		c.Flags.StringVar(&c.hostname, "h", "", "hostname of the host to list the VMs")
		return c
	},
}

type listVM struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	outputFlags site.OutputFlags
	hostname    string
}

func (c *listVM) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *listVM) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
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

	// Print the VMs
	existingVMs := machinelse.GetChromeBrowserMachineLse().GetVms()
	if c.outputFlags.JSON() {
		utils.PrintVMsJSON(existingVMs)
	} else {
		utils.PrintVMs(existingVMs)
	}
	return nil
}

func (c *listVM) validateArgs() error {
	if c.hostname == "" {
		return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nHostname parameter is required to list the VMs on the host")
	}
	return nil
}
