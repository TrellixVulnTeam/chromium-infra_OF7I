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
	"infra/cmd/shivas/site"
	"infra/cmd/shivas/utils"
	"infra/cmdsupport/cmdlib"
	fleet "infra/unifiedfleet/api/v1/proto"
	UfleetAPI "infra/unifiedfleet/api/v1/rpc"
)

// AddSwitchCmd add Switch to the system.
var AddSwitchCmd = &subcommands.Command{
	UsageLine: "add",
	ShortDesc: "add Switch by name",
	LongDesc: `add switch by name.
	./shivas switch add -j -f switch.json
	Adds a Switch by reading a JSON file input.

	./shivas switch -i
	Adds a Switch by reading input through interactive mode.`,
	CommandRun: func() subcommands.CommandRun {
		c := &addSwitch{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.Flags.StringVar(&c.newSpecsFile, "f", "",
			`Path to a file containing Switch specification in JSON format.
This file must contain one Switch JSON message.
 Example : {
	"name": "switch-test-example",
	"capacityPort": 456
 }

The protobuf definition of Switch is part of
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/unifiedfleet/api/v1/proto/peripherals.proto#71`)
		c.Flags.BoolVar(&c.json, "j", false, `interpret the input file as a JSON file.`)
		c.Flags.BoolVar(&c.interactive, "i", false, "enable interactive mode for input")
		return c
	},
}

type addSwitch struct {
	subcommands.CommandRunBase
	authFlags    authcli.Flags
	envFlags     site.EnvFlags
	newSpecsFile string
	json         bool
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
	var s fleet.Switch
	if c.interactive {
		utils.GetSwitchInteractiveInput(&s)
	} else {
		err = utils.ParseJSONFile(c.newSpecsFile, &s)
		if err != nil {
			return err
		}
	}
	e := c.envFlags.Env()
	fmt.Printf("Using UnifiedFleet service %s\n", e.UnifiedFleetService)
	ic := UfleetAPI.NewFleetPRPCClient(&prpc.Client{
		C:       hc,
		Host:    e.UnifiedFleetService,
		Options: site.DefaultPRPCOptions,
	})
	res, err := ic.CreateSwitch(ctx, &UfleetAPI.CreateSwitchRequest{
		Switch:   &s,
		SwitchId: s.GetName(),
	})
	if err != nil {
		return err
	}
	utils.PrintProtoJSON(res)
	return nil
}

func (c *addSwitch) validateArgs() error {
	if !c.interactive && c.newSpecsFile == "" {
		return cmdlib.NewUsageError(c.Flags, "Wrong usage!!")
	}
	return nil
}
