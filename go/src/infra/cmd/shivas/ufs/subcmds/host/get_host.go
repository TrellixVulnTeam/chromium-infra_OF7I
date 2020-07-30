// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package host

import (
	"context"
	"fmt"

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

		c.Flags.BoolVar(&c.full, "full", false, "get the full information of a host")
		return c
	},
}

type getHost struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
	envFlags  site.EnvFlags

	full bool
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

	if c.full {
		lse, machine, nic, nicDhcp, err := getFull(ctx, ic, args[0])
		if err != nil {
			return err
		}
		utils.PrintProtoJSON(lse)
		if machine != nil {
			utils.PrintProtoJSON(machine)
		}
		if nic != nil {
			utils.PrintProtoJSON(nic)
		}
		if nicDhcp != nil {
			utils.PrintProtoJSON(nicDhcp)
		}
		return nil
	}

	machinelse, err := ic.GetMachineLSE(ctx, &ufsAPI.GetMachineLSERequest{
		Name: ufsUtil.AddPrefix(ufsUtil.MachineLSECollection, args[0]),
	})
	if err != nil {
		return err
	}
	machinelse.Name = ufsUtil.RemovePrefix(machinelse.Name)
	utils.PrintProtoJSON(machinelse)
	return nil
}

func getFull(ctx context.Context, ic ufsAPI.FleetClient, lseName string) (*ufspb.MachineLSE, *ufspb.Machine, *ufspb.Nic, *ufspb.DHCPConfig, error) {
	lse, err := ic.GetMachineLSE(ctx, &ufsAPI.GetMachineLSERequest{
		Name: ufsUtil.AddPrefix(ufsUtil.MachineLSECollection, lseName),
	})
	if err != nil {
		return nil, nil, nil, nil, err
	}
	lse.Name = ufsUtil.RemovePrefix(lse.Name)
	var nic *ufspb.Nic
	if lse.GetNic() != "" {
		nic, err = ic.GetNic(ctx, &ufsAPI.GetNicRequest{
			Name: ufsUtil.AddPrefix(ufsUtil.NicCollection, lse.GetNic()),
		})
		if err != nil {
			return nil, nil, nil, nil, fmt.Errorf("Fail to get this host's nic by name %s", lse.GetNic())
		}
		nic.Name = ufsUtil.RemovePrefix(nic.Name)
	}
	var machine *ufspb.Machine
	if len(lse.GetMachines()) >= 1 {
		machine, err = ic.GetMachine(ctx, &ufsAPI.GetMachineRequest{
			Name: ufsUtil.AddPrefix(ufsUtil.MachineCollection, lse.GetMachines()[0]),
		})
		if err != nil {
			return nil, nil, nil, nil, fmt.Errorf("Fail to get this host's machine by name %s", lse.GetMachines()[0])
		}
	}
	dhcp, err := ic.GetDHCPConfig(ctx, &ufsAPI.GetDHCPConfigRequest{
		Hostname: lse.GetHostname(),
	})
	if ufsUtil.IsNotFoundError(err) {
		return lse, machine, nic, nil, nil
	}
	if err != nil {
		return nil, nil, nil, nil, err
	}
	return lse, machine, nic, dhcp, nil
}

func (c *getHost) validateArgs() error {
	if c.Flags.NArg() == 0 {
		return cmdlib.NewUsageError(c.Flags, "Please provide the host name or deployed host hostname.")
	}
	return nil
}
