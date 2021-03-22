// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dut

import (
	"context"
	"fmt"
	"infra/cmd/crosfleet/internal/common"
	dutinfopb "infra/cmd/crosfleet/internal/proto"
	"infra/cmd/crosfleet/internal/site"
	"infra/cmdsupport/cmdlib"
	ufsapi "infra/unifiedfleet/api/v1/rpc"
	ufsutil "infra/unifiedfleet/app/util"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
)

const (
	infoCmdName = "info"
)

var info = &subcommands.Command{
	UsageLine: fmt.Sprintf("%s HOSTNAME [HOSTNAME...]", infoCmdName),
	ShortDesc: "print DUT information",
	LongDesc: `Print DUT information from UFS.

This command's behavior is subject to change without notice.
Do not build automation around this subcommand.`,
	CommandRun: func() subcommands.CommandRun {
		c := &infoRun{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.Flags.BoolVar(&c.json, "json", false, "Format output as JSON.")
		return c
	},
}

type infoRun struct {
	subcommands.CommandRunBase
	json      bool
	authFlags authcli.Flags
	envFlags  common.EnvFlags
}

func (c *infoRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *infoRun) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	if len(args) == 0 {
		return fmt.Errorf("missing DUT hostname arg")
	}
	ctx := cli.GetContext(a, c, env)
	ufsClient, err := newUFSClient(ctx, c.envFlags.Env().UFSService, &c.authFlags)
	if err != nil {
		return err
	}

	var infoList dutinfopb.DUTInfoList
	for _, hostname := range args {
		info, err := getDutInfo(ctx, ufsClient, hostname)
		if err != nil {
			return err
		}
		// If outputting the command as JSON, collect all DUT info in a proto
		// message first, then print together as one JSON object.
		// Otherwise, just print each separately from this loop.
		if c.json {
			infoList.DUTs = append(infoList.DUTs, info)
		} else {
			fmt.Fprintf(a.GetOut(), "%s\n\n", dutInfoAsBashVariables(info))
		}
	}
	if c.json {
		fmt.Fprintf(a.GetOut(), "%s\n", protoJSON(&infoList))
	}
	return nil
}

// getDutInfo returns information about the DUT with the given hostname. There
// is no newline at the end of the info string.
func getDutInfo(ctx context.Context, ufsClient ufsapi.FleetClient, hostname string) (*dutinfopb.DUTInfo, error) {
	ctx = contextWithOSNamespace(ctx)
	correctedHostname := correctedHostname(hostname)
	labSetup, err := ufsClient.GetMachineLSE(ctx, &ufsapi.GetMachineLSERequest{
		Name: ufsutil.AddPrefix(ufsutil.MachineLSECollection, correctedHostname),
	})
	if err != nil {
		return nil, err
	}
	machine, err := ufsClient.GetMachine(ctx, &ufsapi.GetMachineRequest{
		Name: ufsutil.AddPrefix(ufsutil.MachineCollection, labSetup.GetMachines()[0]),
	})
	if err != nil {
		return nil, err
	}
	return &dutinfopb.DUTInfo{
		Hostname: correctedHostname,
		LabSetup: labSetup,
		Machine:  machine,
	}, nil
}

// dutInfoAsBashVariables returns a pretty-printed string containing info about
// the given DUT formatted as bash variables.
func dutInfoAsBashVariables(info *dutinfopb.DUTInfo) string {
	infoString := fmt.Sprintf(`DUT_HOSTNAME=%s.cros
MODEL=%s
BOARD=%s`,
		info.GetHostname(),
		info.GetMachine().GetChromeosMachine().GetModel(),
		info.GetMachine().GetChromeosMachine().GetBuildTarget())

	servo := info.GetLabSetup().GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetServo()
	if servo == nil {
		return infoString
	}
	infoString += fmt.Sprintf(`
SERVO_HOSTNAME=%s
SERVO_PORT=%d
SERVO_SERIAL=%s`,
		servo.GetServoHostname(),
		servo.GetServoPort(),
		servo.GetServoSerial())

	return infoString
}
