// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package switches

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

// AddSwitchCmd add Switch in the lab.
var AddSwitchCmd = &subcommands.Command{
	UsageLine: "add-switch [Options...]",
	ShortDesc: "Add a switch by name",
	LongDesc:  cmdhelp.AddSwitchLongDesc,
	CommandRun: func() subcommands.CommandRun {
		c := &addSwitch{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.Flags.StringVar(&c.newSpecsFile, "f", "", cmdhelp.SwitchFileText)
		c.Flags.StringVar(&c.rackName, "r", "", "name of the rack to associate the switch")
		c.Flags.BoolVar(&c.interactive, "i", false, "enable interactive mode for input")
		return c
	},
}

type addSwitch struct {
	subcommands.CommandRunBase
	authFlags    authcli.Flags
	envFlags     site.EnvFlags
	rackName     string
	newSpecsFile string
	interactive  bool
}

func (c *addSwitch) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *addSwitch) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
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
	var s ufspb.Switch
	if c.interactive {
		c.rackName = utils.GetSwitchInteractiveInput(ctx, ic, &s, false)
	} else {
		if err = utils.ParseJSONFile(c.newSpecsFile, &s); err != nil {
			return err
		}
	}
	res, err := ic.CreateSwitch(ctx, &ufsAPI.CreateSwitchRequest{
		Switch:   &s,
		SwitchId: s.GetName(),
		Rack:     c.rackName,
	})
	if err != nil {
		return err
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	utils.PrintProtoJSON(res)
	fmt.Println()
	return nil
}

func (c *addSwitch) validateArgs() error {
	if !c.interactive {
		if c.newSpecsFile == "" {
			return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nNeither JSON input file specified nor in interactive mode to accept input.")
		}
		if c.rackName == "" {
			return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nRack name parameter is required to associate the switch with a rack.")
		}
	}
	return nil
}
