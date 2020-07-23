// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package rack

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

// UpdateRackCmd update Rack by given name.
var UpdateRackCmd = &subcommands.Command{
	UsageLine: "update-rack [Options...]",
	ShortDesc: "Update a rack by name",
	LongDesc:  cmdhelp.UpdateRackLongDesc,
	CommandRun: func() subcommands.CommandRun {
		c := &updateRack{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.Flags.StringVar(&c.newSpecsFile, "f", "", cmdhelp.RackFileText)
		c.Flags.BoolVar(&c.interactive, "i", false, "enable interactive mode for input")
		return c
	},
}

type updateRack struct {
	subcommands.CommandRunBase
	authFlags    authcli.Flags
	envFlags     site.EnvFlags
	newSpecsFile string
	interactive  bool
}

func (c *updateRack) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *updateRack) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
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
	var rack ufspb.Rack
	if c.interactive {
		return errors.New("Interactive mode for this " +
			"command is not yet implemented yet. Use JSON input mode.")
	}
	if err = utils.ParseJSONFile(c.newSpecsFile, &rack); err != nil {
		return err
	}
	rack.Name = ufsUtil.AddPrefix(ufsUtil.RackCollection, rack.Name)
	res, err := ic.UpdateRack(ctx, &ufsAPI.UpdateRackRequest{
		Rack: &rack,
	})
	if err != nil {
		return err
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	utils.PrintProtoJSON(res)
	fmt.Println()
	return nil
}

func (c *updateRack) validateArgs() error {
	if !c.interactive && c.newSpecsFile == "" {
		return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nNeither JSON input file specified nor in interactive mode to accept input.")
	}
	return nil
}
