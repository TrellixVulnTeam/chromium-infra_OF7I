// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dut

import (
	"context"
	"fmt"
	"infra/cmd/crosfleet/internal/common"
	"infra/cmd/crosfleet/internal/site"
	"infra/cmdsupport/cmdlib"
	ufspb "infra/unifiedfleet/api/v1/models"
	ufsapi "infra/unifiedfleet/api/v1/rpc"
	ufsutil "infra/unifiedfleet/app/util"
	"strings"

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

	var infoList []string
	for _, hostname := range args {
		info, err := dutInfo(ctx, ufsClient, c.json, hostname)
		if err != nil {
			return err
		}
		infoList = append(infoList, info)
	}
	if c.json {
		fmt.Fprintf(a.GetOut(), "[%s]\n", strings.Join(infoList, ",\n"))
	} else {
		fmt.Fprintf(a.GetOut(), "%s\n\n", strings.Join(infoList, "\n\n"))
	}
	return nil
}

// dutInfo returns pretty-printed information about the DUT with the given
// hostname. There is no newline at the end of the info string.
func dutInfo(ctx context.Context, ufsClient ufsapi.FleetClient, json bool, hostname string) (string, error) {
	ctx = contextWithOSNamespace(ctx)
	correctedHostname := correctedHostname(hostname)
	labSetup, err := ufsClient.GetMachineLSE(ctx, &ufsapi.GetMachineLSERequest{
		Name: ufsutil.AddPrefix(ufsutil.MachineLSECollection, correctedHostname),
	})
	if err != nil {
		return "", err
	}
	machine, err := ufsClient.GetMachine(ctx, &ufsapi.GetMachineRequest{
		Name: ufsutil.AddPrefix(ufsutil.MachineCollection, labSetup.GetMachines()[0]),
	})
	if err != nil {
		return "", err
	}
	if json {
		return dutInfoAsJSON(labSetup, machine), nil
	} else {
		return dutInfoAsBashVariables(labSetup, machine), nil
	}
}

func dutInfoAsBashVariables(labSetup *ufspb.MachineLSE, machine *ufspb.Machine) string {
	dut := labSetup.GetChromeosMachineLse().GetDeviceLse().GetDut()

	info := fmt.Sprintf(`DUT_HOSTNAME=%s.cros
MODEL=%s
BOARD=%s`,
		dut.Hostname,
		machine.GetChromeosMachine().GetModel(),
		machine.GetChromeosMachine().GetBuildTarget())

	servo := dut.GetPeripherals().GetServo()
	if servo == nil {
		return info
	}
	info += fmt.Sprintf(`
SERVO_HOSTNAME=%s
SERVO_PORT=%d
SERVO_SERIAL=%s`,
		servo.GetServoHostname(),
		servo.GetServoPort(),
		servo.GetServoSerial())

	return info
}

func dutInfoAsJSON(labSetup *ufspb.MachineLSE, machine *ufspb.Machine) string {
	return fmt.Sprintf(`{"MachineLSE": %s,
"Machine": %s}`, protoJSON(labSetup), protoJSON(machine))
}
