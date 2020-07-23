// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package rack

import (
	"fmt"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/grpc/prpc"

	"infra/cmd/shivas/site"
	"infra/cmd/shivas/utils"
	"infra/cmdsupport/cmdlib"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	ufsUtil "infra/unifiedfleet/app/util"
)

// GetRackCmd get rack by given name.
var GetRackCmd = &subcommands.Command{
	UsageLine: "rack {Rack Name}",
	ShortDesc: "Get rack details by name.",
	LongDesc: `Get rack details by name

Example:

shivas get rack {Rack Name}
Gets the rack and prints the output in JSON format.

shivas get rack {Rack Name}
Gets the rack and prints the output in JSON format.`,
	CommandRun: func() subcommands.CommandRun {
		c := &getRack{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.commonFlags.Register(&c.Flags)
		return c
	},
}

type getRack struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags
}

func (c *getRack) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *getRack) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
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
	if c.commonFlags.Verbose() {
		fmt.Printf("Using UnifiedFleet service %s\n", e.UnifiedFleetService)
	}
	rackRes, err := ic.GetRack(ctx, &ufsAPI.GetRackRequest{
		Name: ufsUtil.AddPrefix(ufsUtil.RackCollection, args[0]),
	})
	if err != nil {
		return err
	}
	rackRes.Name = ufsUtil.RemovePrefix(rackRes.Name)
	utils.PrintProtoJSON(rackRes)
	fmt.Println()
	return nil
}

func (c *getRack) validateArgs() error {
	if c.Flags.NArg() == 0 {
		return cmdlib.NewUsageError(c.Flags, "Please provide the rack name.")
	}
	return nil
}
