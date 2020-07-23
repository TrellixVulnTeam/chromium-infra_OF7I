// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package host

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

// AddHostCmd add a host to the machine.
var AddHostCmd = &subcommands.Command{
	UsageLine: "add-host [Options..]",
	ShortDesc: "Add a host(DUT, Labstation, Dev Server, Caching Server, VM Server, Host OS...) on a machine",
	LongDesc:  cmdhelp.AddHostLongDesc,
	CommandRun: func() subcommands.CommandRun {
		c := &addHost{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.Flags.StringVar(&c.newSpecsFile, "f", "", cmdhelp.MachineLSEFileText)
		c.Flags.StringVar(&c.machineName, "m", "", "name of the machine to associate the host")
		c.Flags.BoolVar(&c.interactive, "i", false, "enable interactive mode for input")
		return c
	},
}

type addHost struct {
	subcommands.CommandRunBase
	authFlags    authcli.Flags
	envFlags     site.EnvFlags
	machineName  string
	newSpecsFile string
	interactive  bool
}

func (c *addHost) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *addHost) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
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
	var machinelse ufspb.MachineLSE
	if c.interactive {
		return errors.New("Interactive mode for this " +
			"command is not yet implemented yet. Use JSON input mode.")
		//TODO(eshwarn): add interactive input
		//utils.GetMachinelseInteractiveInput(ctx, ic, &machinelse, false)
	}
	if err = utils.ParseJSONFile(c.newSpecsFile, &machinelse); err != nil {
		return err
	}
	res, err := ic.CreateMachineLSE(ctx, &ufsAPI.CreateMachineLSERequest{
		MachineLSE:   &machinelse,
		MachineLSEId: machinelse.GetName(),
		Machines:     []string{c.machineName},
	})
	if err != nil {
		return err
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	utils.PrintProtoJSON(res)
	fmt.Println()
	return nil
}

func (c *addHost) validateArgs() error {
	if !c.interactive {
		if c.newSpecsFile == "" {
			return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nNeither JSON input file specified nor in interactive mode to accept input.")
		}
		if c.machineName == "" {
			return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nMachine name parameter is required to associate the host with a machine.")
		}
	}
	return nil
}
