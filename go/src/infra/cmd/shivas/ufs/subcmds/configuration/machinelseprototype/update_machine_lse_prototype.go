// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package machinelseprototype

import (
	"fmt"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/grpc/prpc"
	"infra/cmd/shivas/site"
	"infra/cmd/shivas/utils"
	"infra/cmdsupport/cmdlib"
	fleet "infra/unifiedfleet/api/v1/proto"
	UfleetAPI "infra/unifiedfleet/api/v1/rpc"
	UfleetUtil "infra/unifiedfleet/app/util"
)

// UpdateMachinelsePrototypeCmd set MachineLSEPrototype by given name.
var UpdateMachinelsePrototypeCmd = &subcommands.Command{
	UsageLine: "set",
	ShortDesc: "update MachineLSEPrototype by name",
	LongDesc: `update MachineLSEPrototype by name.
	./shivas machinelseprototype set -j -f machinelseprototype.json
	Updates a MachineLSEPrototype by reading a JSON file input.

	./shivas machinelseprototype -i
	Updates a MachineLSEPrototype by reading input through interactive mode.`,
	CommandRun: func() subcommands.CommandRun {
		c := &updateMachinelsePrototype{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.Flags.StringVar(&c.newSpecsFile, "f", "",
			`Path to a file containing MachineLSEPrototype specification in JSON format.
This file must contain one MachineLSEPrototype JSON message.
Example MachineLsePrototype:
{
	"name": "browser-lab:vm",
	"peripheralRequirements": [
		{
			"peripheralType": "PERIPHERAL_TYPE_SWITCH",
			"min": 5,
			"max": 7
		}
	],
	"occupiedCapacityRu": 32,
	"virtualRequirements": [
		{
			"virtualType": "VIRTUAL_TYPE_VM",
			"min": 3,
			"max": 4
		}
	]
}

The protobuf definition of MachineLSEPrototype is part of
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/unifiedfleet/api/v1/proto/lse_prototype.proto#29`)
		c.Flags.BoolVar(&c.json, "j", false, `interpret the input file as a JSON file.`)
		c.Flags.BoolVar(&c.interactive, "i", false, "enable interactive mode for input")
		return c
	},
}

type updateMachinelsePrototype struct {
	subcommands.CommandRunBase
	authFlags    authcli.Flags
	envFlags     site.EnvFlags
	newSpecsFile string
	json         bool
	interactive  bool
}

func (c *updateMachinelsePrototype) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *updateMachinelsePrototype) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	if err := c.validateArgs(); err != nil {
		return err
	}
	ctx := cli.GetContext(a, c, env)
	hc, err := cmdlib.NewHTTPClient(ctx, &c.authFlags)
	if err != nil {
		return err
	}
	e := c.envFlags.Env()
	fmt.Printf("Using UnifiedFleet service %s\n", e.UnifiedFleetService)
	ic := UfleetAPI.NewFleetPRPCClient(&prpc.Client{
		C:       hc,
		Host:    e.UnifiedFleetService,
		Options: site.DefaultPRPCOptions,
	})
	var machinelsePrototype fleet.MachineLSEPrototype
	if c.interactive {
		utils.GetMachinelsePrototypeInteractiveInput(ctx, ic, &machinelsePrototype, true)
	} else {
		err = utils.ParseJSONFile(c.newSpecsFile, &machinelsePrototype)
		if err != nil {
			return err
		}
	}
	machinelsePrototype.Name = UfleetUtil.AddPrefix(UfleetUtil.MachineLSEPrototypeCollection, machinelsePrototype.Name)
	res, err := ic.UpdateMachineLSEPrototype(ctx, &UfleetAPI.UpdateMachineLSEPrototypeRequest{
		MachineLSEPrototype: &machinelsePrototype,
	})
	if err != nil {
		return err
	}
	res.Name = UfleetUtil.RemovePrefix(res.Name)
	utils.PrintProtoJSON(res)
	return nil
}

func (c *updateMachinelsePrototype) validateArgs() error {
	if !c.interactive && c.newSpecsFile == "" {
		return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nNeither JSON input file specified nor in interactive mode to accept input.")
	}
	return nil
}
