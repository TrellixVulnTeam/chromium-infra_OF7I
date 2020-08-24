// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package switches

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

// GetSwitchCmd get Switch by given name.
var GetSwitchCmd = &subcommands.Command{
	UsageLine: "switch {Switch Name}",
	ShortDesc: "get switch details by name",
	LongDesc: `get switch details by name.

Example:
shivas get switch {Switch Name}
Gets the switch and prints the output in JSON format.`,
	CommandRun: func() subcommands.CommandRun {
		c := &getSwitch{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.commonFlags.Register(&c.Flags)
		c.outputFlags.Register(&c.Flags)
		return c
	},
}

type getSwitch struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags
	outputFlags site.OutputFlags
}

func (c *getSwitch) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *getSwitch) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
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
		fmt.Printf("Using UnifiedFleet service %s\n", e.UnifiedFleetService)
	}
	ic := ufsAPI.NewFleetPRPCClient(&prpc.Client{
		C:       hc,
		Host:    e.UnifiedFleetService,
		Options: site.DefaultPRPCOptions,
	})
	res, err := ic.GetSwitch(ctx, &ufsAPI.GetSwitchRequest{
		Name: ufsUtil.AddPrefix(ufsUtil.SwitchCollection, args[0]),
	})
	if err != nil {
		return err
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	if utils.FullMode(c.outputFlags.Full()) {
		return c.printFull(ctx, ic, res)
	}
	return c.print(res)
}

func (c *getSwitch) printFull(ctx context.Context, ic ufsAPI.FleetClient, sw *ufspb.Switch) error {
	resp, err := ic.ListNics(ctx, &ufsAPI.ListNicsRequest{
		Filter: "switch=" + sw.GetName(),
	})
	nics := resp.GetNics()
	if err != nil && c.commonFlags.Verbose() {
		fmt.Printf("Failed to get nic for the switch: %s", err)
	}
	for _, nic := range nics {
		nic.Name = ufsUtil.RemovePrefix(nic.Name)
	}
	resp2, err := ic.ListDracs(ctx, &ufsAPI.ListDracsRequest{
		Filter: "switch=" + sw.GetName(),
	})
	dracs := resp2.GetDracs()
	if err != nil && c.commonFlags.Verbose() {
		fmt.Printf("Failed to get drac for the switch: %s", err)
	}
	for _, drac := range dracs {
		drac.Name = ufsUtil.RemovePrefix(drac.Name)
	}
	if c.outputFlags.JSON() {
		// TODO: print nics/dracs json
		utils.PrintProtoJSON(sw, !utils.NoEmitMode(c.outputFlags.NoEmit()))
	}
	if c.outputFlags.Tsv() {
		utils.PrintTSVSwitchFull(sw, nics, dracs)
		return nil
	}
	utils.PrintTitle(utils.SwitchFullTitle)
	utils.PrintSwitchFull(sw, nics, dracs)
	return nil
}

func (c *getSwitch) print(sw *ufspb.Switch) error {
	if c.outputFlags.JSON() {
		utils.PrintProtoJSON(sw, !utils.NoEmitMode(c.outputFlags.NoEmit()))
		return nil
	}
	if c.outputFlags.Tsv() {
		utils.PrintTSVSwitches([]*ufspb.Switch{sw}, false)
		return nil
	}
	utils.PrintTitle(utils.SwitchTitle)
	utils.PrintSwitches([]*ufspb.Switch{sw}, false)
	return nil
}

func (c *getSwitch) validateArgs() error {
	if c.Flags.NArg() == 0 {
		return cmdlib.NewUsageError(c.Flags, "Please provide the switch name")
	}
	return nil
}
