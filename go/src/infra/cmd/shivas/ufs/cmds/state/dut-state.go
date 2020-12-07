// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package state

import (
	"fmt"
	"time"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/grpc/prpc"

	"infra/cmd/shivas/site"
	"infra/cmd/shivas/utils"
	"infra/cmdsupport/cmdlib"
	"infra/cros/dutstate"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	ufsUtil "infra/unifiedfleet/app/util"
)

// DutStateCmd subcommand: get State of the DUT from UFS.
var DutStateCmd = &subcommands.Command{
	UsageLine: "dut-state [FLAGS...]",
	ShortDesc: "read state for a DUT",
	LongDesc:  `read DUT state from UFS.`,
	CommandRun: func() subcommands.CommandRun {
		c := &dutStateCmdRun{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.commonFlags.Register(&c.Flags)
		return c
	},
}

type dutStateCmdRun struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags
}

// Run implements the subcommands.CommandRun interface.
func (c *dutStateCmdRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *dutStateCmdRun) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	if err := c.validateArgs(args); err != nil {
		return err
	}
	ctx := cli.GetContext(a, c, env)
	ctx = utils.SetupContext(ctx, ufsUtil.OSNamespace)
	hc, err := cmdlib.NewHTTPClient(ctx, &c.authFlags)
	if err != nil {
		return err
	}
	e := c.envFlags.Env()
	if c.commonFlags.Verbose() {
		fmt.Printf("Using UnifiedFleet service %s\n", e.UnifiedFleetService)
	}
	ufsClient := ufsAPI.NewFleetPRPCClient(&prpc.Client{
		C:       hc,
		Host:    e.UnifiedFleetService,
		Options: site.DefaultPRPCOptions,
	})
	host := args[0]
	i := dutstate.Read(ctx, ufsClient, host)
	fmt.Printf("%s: %s\n", host, i.State.String())
	if c.commonFlags.Verbose() {
		fmt.Printf("Updated at:%s \n", time.Unix(i.Time, 0))
	}
	return nil
}

func (c *dutStateCmdRun) validateArgs(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("Expected an args to provide a host")
	}
	if len(args) > 1 {
		return fmt.Errorf("Expected an args to provide only one host: %s", args)
	}
	return nil
}
