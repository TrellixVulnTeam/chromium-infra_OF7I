// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package host

import (
	"context"

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

// GetHostCmd get host by given name.
var GetHostCmd = &subcommands.Command{
	UsageLine: "host {Host name}",
	ShortDesc: "Get host details by name",
	LongDesc: `Get host details by name.

Example:

shivas get host {Host name}
Gets the host and prints the output in JSON format.`,
	CommandRun: func() subcommands.CommandRun {
		c := &getHost{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.outputFlags.Register(&c.Flags)
		return c
	},
}

type getHost struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	outputFlags site.OutputFlags
}

func (c *getHost) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *getHost) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
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

	machinelse, err := ic.GetMachineLSE(ctx, &ufsAPI.GetMachineLSERequest{
		Name: ufsUtil.AddPrefix(ufsUtil.MachineLSECollection, args[0]),
	})
	if err != nil {
		return err
	}
	machinelse.Name = ufsUtil.RemovePrefix(machinelse.Name)
	if utils.FullMode(c.outputFlags.Full()) {
		return c.printFull(ctx, ic, machinelse)
	}
	return c.print(machinelse)
}

func (c *getHost) printFull(ctx context.Context, ic ufsAPI.FleetClient, lse *ufspb.MachineLSE) error {
	var nic *ufspb.Nic
	if lse.GetNic() != "" {
		if nic, _ = ic.GetNic(ctx, &ufsAPI.GetNicRequest{
			Name: ufsUtil.AddPrefix(ufsUtil.NicCollection, lse.GetNic()),
		}); nic != nil {
			nic.Name = ufsUtil.RemovePrefix(nic.Name)
		}
	}
	var machine *ufspb.Machine
	if len(lse.GetMachines()) >= 1 {
		if machine, _ = ic.GetMachine(ctx, &ufsAPI.GetMachineRequest{
			Name: ufsUtil.AddPrefix(ufsUtil.MachineCollection, lse.GetMachines()[0]),
		}); machine != nil {
			machine.Name = ufsUtil.RemovePrefix(machine.Name)
		}
	}
	dhcp, _ := ic.GetDHCPConfig(ctx, &ufsAPI.GetDHCPConfigRequest{
		Hostname: lse.GetHostname(),
	})

	if c.outputFlags.Tsv() {
		utils.PrintTSVHostFull(lse, machine, dhcp)
		return nil
	}
	utils.PrintTitle(utils.MachineLSETFullitle)
	utils.PrintMachineLSEFull(lse, machine, dhcp)
	return nil
}

func (c *getHost) print(lse *ufspb.MachineLSE) error {
	if c.outputFlags.JSON() {
		utils.PrintProtoJSON(lse, !c.outputFlags.NoEmit())
		return nil
	}
	if c.outputFlags.Tsv() {
		utils.PrintTSVMachineLSEs([]*ufspb.MachineLSE{lse}, false)
		return nil
	}
	utils.PrintTitle(utils.MachineLSETitle)
	utils.PrintMachineLSEs([]*ufspb.MachineLSE{lse}, false)
	return nil
}

func (c *getHost) validateArgs() error {
	if c.Flags.NArg() == 0 {
		return cmdlib.NewUsageError(c.Flags, "Please provide the host name or deployed host hostname.")
	}
	return nil
}
