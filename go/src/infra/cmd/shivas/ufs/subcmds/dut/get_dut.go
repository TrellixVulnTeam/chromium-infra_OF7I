// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dut

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/grpc/prpc"

	"infra/cmd/shivas/site"
	"infra/cmd/shivas/utils"
	"infra/cmdsupport/cmdlib"
	ufspb "infra/unifiedfleet/api/v1/models"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	ufsUtil "infra/unifiedfleet/app/util"
)

// GetDutCmd get host by given name.
var GetDutCmd = &subcommands.Command{
	UsageLine: "dut ...",
	ShortDesc: "Get dut details by filters",
	LongDesc: `Get dut details by filters.

Example:

shivas get dut {name1}

Gets the ChromeOS DUT and prints the output in user-specified format.`,
	CommandRun: func() subcommands.CommandRun {
		c := &getDut{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.outputFlags.Register(&c.Flags)
		c.commonFlags.Register(&c.Flags)

		return c
	},
}

type getDut struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	outputFlags site.OutputFlags
	commonFlags site.CommonFlags
}

func (c *getDut) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *getDut) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	ctx := cli.GetContext(a, c, env)
	ns, err := c.envFlags.Namespace()
	if err != nil {
		return err
	}
	ctx = utils.SetupContext(ctx, ns)
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

	res, err := ic.GetMachineLSE(ctx, &ufsAPI.GetMachineLSERequest{
		Name: ufsUtil.AddPrefix(ufsUtil.MachineLSECollection, args[0]),
	})
	if err != nil {
		return err
	}

	emit := !utils.NoEmitMode(c.outputFlags.NoEmit())
	if c.outputFlags.JSON() {
		utils.PrintDutsJSON([]proto.Message{res}, emit)
	} else {
		printDutShort(ctx, ic, []proto.Message{res}, false)
	}
	return nil
}

func printDutShort(ctx context.Context, ic ufsAPI.FleetClient, msgs []proto.Message, tsv bool) error {
	machineMap := make(map[string]*ufspb.Machine, 0)
	lses := make([]*ufspb.MachineLSE, len(msgs))
	for i, r := range msgs {
		lses[i] = r.(*ufspb.MachineLSE)
		lses[i].Name = ufsUtil.RemovePrefix(lses[i].Name)
		if len(lses[i].GetMachines()) == 0 {
			fmt.Println("Invalid host ", lses[i].Name)
			continue
		}
		res, _ := ic.GetMachine(ctx, &ufsAPI.GetMachineRequest{
			Name: ufsUtil.AddPrefix(ufsUtil.MachineCollection, lses[i].GetMachines()[0]),
		})
		machineMap[lses[i].Name] = res
	}
	utils.PrintDutsShort(lses, machineMap)
	return nil
}
