// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package machinelse

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

// AddMachinelseCmd add MachineLSE to the system.
var AddMachinelseCmd = &subcommands.Command{
	UsageLine: "add",
	ShortDesc: "add MachineLSE by name",
	LongDesc: `add MachineLSE by name.
	./shivas machinelse add -j -f machinelse.json
	Adds a MachineLSE by reading a JSON file input.

	./shivas machinelse -i
	Adds a MachineLSE by reading input through interactive mode.`,
	CommandRun: func() subcommands.CommandRun {
		c := &addMachinelse{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.Flags.StringVar(&c.newSpecsFile, "f", "",
			`Path to a file containing Machine specification in JSON format.
This file must contain one MachineLSE JSON message.
Example Browser Lab MachineLSE: {
	"hostname": "Dell",
	"chromeBrowserMachineLse": {
		"vms": [
			{
				"name": "Windows Server 2008",
				"osVersion": {
					"value": "Server edition 2008",
					"description": "Windows Server"
				},
				"macAddress": "23.1.34.5",
				"hostname": "WinVM"
			}
		]
	},
	"machines": [
		"machine-test-123"
	],
}

 Example OS Lab MachineLSE: {
	"hostname": "ChromeOSSmaus",
	"chromeosMachineLse": {
		"dut": {
			"config": {
				"hostname": "chromes-Samus6",
				"peripherals": {
					"servo": {
						"servoHostname": "servo-12",
						"servoPort": 23,
						"servoSerial": "23456",
						"servoType": "v3"
					},
					"chameleon": {
						"chameleonPeripherals": [
							"CHAMELEON_TYPE_VGA"
						],
						"audioBoard": true
					},
					"rpm": {
						"powerunitOutlet": "0"
					},
					"connectedCamera": [
						{
							"cameraType": "CAMERA_PTZPRO2"
						}
					],
					"audio": {
						"audioBox": true,
						"atrus": true
					},
					"wifi": {
						"wificell": true,
						"antennaConn": "CONN_CONDUCTIVE",
						"router": "ROUTER_802_11AX"
					},
					"touch": {
						"mimo": true
					},
					"carrier": "ATT",
					"camerabox": true,
					"chaos": true,
					"cable": [
						{
							"type": "CABLE_USBAUDIO"
						},
						{
							"type": "CABLE_USBPRINTING"
						},
						{
							"type": "CABLE_HDMIAUDIO"
						}
					],
					"cameraboxInfo": {
						"facing": "FACING_BACK"
					}
				},
				"pools": [
					"DUT_ACS_POOL"
				]
			},
			"rpmInterface": {
				"rpm": "rpm-1",
				"port": 23
			},
			"networkDeviceInterface": {
				"switch": "switch-1",
				"port": 23
			}
		}
	},
	"machines": [
		"machine-test-123"
	],
}

The protobuf definition of MachineLSE is part of
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/unifiedfleet/api/v1/proto/machine_lse.proto#24`)
		c.Flags.BoolVar(&c.json, "j", false, `interpret the input file as a JSON file.`)
		c.Flags.BoolVar(&c.interactive, "i", false, "enable interactive mode for input")
		return c
	},
}

type addMachinelse struct {
	subcommands.CommandRunBase
	authFlags    authcli.Flags
	envFlags     site.EnvFlags
	newSpecsFile string
	json         bool
	interactive  bool
}

func (c *addMachinelse) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *addMachinelse) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
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
	var machinelse fleet.MachineLSE
	if c.interactive {
		//utils.GetMachinelseInteractiveInput(ctx, ic, &machinelse)
	} else {
		err = utils.ParseJSONFile(c.newSpecsFile, &machinelse)
		if err != nil {
			return err
		}
	}
	res, err := ic.CreateMachineLSE(ctx, &UfleetAPI.CreateMachineLSERequest{
		MachineLSE:   &machinelse,
		MachineLSEId: machinelse.GetName(),
	})
	if err != nil {
		return err
	}
	res.Name = UfleetUtil.RemovePrefix(res.Name)
	utils.PrintProtoJSON(res)
	return nil
}

func (c *addMachinelse) validateArgs() error {
	if !c.interactive && c.newSpecsFile == "" {
		return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nNeither JSON input file specified nor in interactive mode to accept input.")
	}
	return nil
}
