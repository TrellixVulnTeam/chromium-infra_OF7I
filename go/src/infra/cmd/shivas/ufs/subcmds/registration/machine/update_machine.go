// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package machine

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

// UpdateMachineCmd set Machine by given name.
var UpdateMachineCmd = &subcommands.Command{
	UsageLine: "set",
	ShortDesc: "update Machine by name",
	LongDesc: `update Machine by name.
	./shivas machine set -j -f machine.json
	Updates a Machine by reading a JSON file input.

	./shivas machine -i
	Updates a Machine by reading input through interactive mode.`,
	CommandRun: func() subcommands.CommandRun {
		c := &updateMachine{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.Flags.StringVar(&c.newSpecsFile, "f", "",
			`Path to a file containing Switch specification in JSON format.
This file must contain one Machine JSON message.
Example Browser Lab Machine: {
	"name": "machine-BROWSERLAB-example",
	"location": {
		"lab": "LAB_DATACENTER_MTV97",
		"rack": "RackName"
	},
	"chromeBrowserMachine": {
		"displayName": "ax105-34-230",
		"chromePlatform": "Dell R230",
		"nic": "ax105-34-230-eth0",
		"kvmInterface": {
			"kvm": "kvm.mtv97",
			"port": 34
		},
		"rpmInterface": {
			"rpm": "rpm.mtv97",
			"port": 65
		},
		"networkDeviceInterface": {
			"switch": "sq005.mtv97",
			"port": 98
		},
		"drac": "ax105-34-230-drac",
		"deploymentTicket": "846026"
	},
	"realm": "Browserlab"
 }

 Example OS Lab Machine: {
	"name": "machine-OSLAB-example",
	"location": {
		"lab": "LAB_CHROME_ATLANTA",
		"aisle": "1",
		"row": "2",
		"rack": "Rack-42",
		"rackNumber": "42",
		"shelf": "3",
		"position": "5"
	},
	"chromeosMachine": {
	},
	"realm": "OSlab"
}

The protobuf definition of Machine is part of
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/unifiedfleet/api/v1/proto/machine.proto#19`)
		c.Flags.BoolVar(&c.json, "j", false, `interpret the input file as a JSON file.`)
		c.Flags.BoolVar(&c.interactive, "i", false, "enable interactive mode for input")
		return c
	},
}

type updateMachine struct {
	subcommands.CommandRunBase
	authFlags    authcli.Flags
	envFlags     site.EnvFlags
	newSpecsFile string
	json         bool
	interactive  bool
}

func (c *updateMachine) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *updateMachine) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
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
	var machine fleet.Machine
	if c.interactive {
		utils.GetMachineInteractiveInput(ctx, ic, &machine)
	} else {
		err = utils.ParseJSONFile(c.newSpecsFile, &machine)
		if err != nil {
			return err
		}
	}
	machine.Name = UfleetUtil.AddPrefix(UfleetUtil.MachineCollection, machine.Name)
	res, err := ic.UpdateMachine(ctx, &UfleetAPI.UpdateMachineRequest{
		Machine: &machine,
	})
	if err != nil {
		return err
	}
	res.Name = UfleetUtil.RemovePrefix(res.Name)
	utils.PrintProtoJSON(res)
	return nil
}

func (c *updateMachine) validateArgs() error {
	if !c.interactive && c.newSpecsFile == "" {
		return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nNeither JSON input file specified nor in interactive mode to accept input.")
	}
	return nil
}
