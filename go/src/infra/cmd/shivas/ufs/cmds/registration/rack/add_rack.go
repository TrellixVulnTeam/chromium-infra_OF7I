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
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
)

// AddRackCmd add Rack to the system.
var AddRackCmd = &subcommands.Command{
	UsageLine: "add-rack [Options...]",
	ShortDesc: "Add a rack by name",
	LongDesc:  cmdhelp.AddRackLongDesc,
	CommandRun: func() subcommands.CommandRun {
		c := &addRack{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.Flags.StringVar(&c.newSpecsFile, "f", "", cmdhelp.RackRegistrationFileText)
		c.Flags.BoolVar(&c.interactive, "i", false, "enable interactive mode for input")
		return c
	},
}

type addRack struct {
	subcommands.CommandRunBase
	authFlags    authcli.Flags
	envFlags     site.EnvFlags
	newSpecsFile string
	interactive  bool
}

func (c *addRack) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *addRack) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
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
	var rackRegistrationReq ufsAPI.RackRegistrationRequest
	if c.interactive {
		return errors.New("Interactive mode for this " +
			"command is not yet implemented yet. Use JSON input mode.")
	}
	if err = utils.ParseJSONFile(c.newSpecsFile, &rackRegistrationReq); err != nil {
		return err
	}
	res, err := ic.RackRegistration(ctx, &rackRegistrationReq)
	if err != nil {
		return err
	}
	utils.PrintProtoJSON(res)
	fmt.Println()
	return nil
}

func (c *addRack) validateArgs() error {
	if !c.interactive && c.newSpecsFile == "" {
		return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nNeither JSON input file specified nor in interactive mode to accept input.")
	}
	return nil
}
