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
	"io"

	"github.com/golang/protobuf/proto"
	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/grpc/prpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
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
	return printDUTInfo(ctx, &c.authFlags, a.GetOut(), c.envFlags.Env().UFSService, c.json, args...)
}

// printDUTInfo pretty-prints information about the DUT with the given hostname
// to the command line.
func printDUTInfo(ctx context.Context, authFlags *authcli.Flags, w io.Writer, ufsService string, json bool, hostnames ...string) error {
	ctx = contextWithOSNamespace(ctx)
	ufsClient, err := newUFSClient(ctx, ufsService, authFlags)
	if err != nil {
		return err
	}
	for _, hostname := range hostnames {
		correctedHostname := correctedHostname(hostname)
		labSetup, err := ufsClient.GetMachineLSE(ctx, &ufsapi.GetMachineLSERequest{
			Name: ufsutil.AddPrefix(ufsutil.MachineLSECollection, correctedHostname),
		})
		if err != nil {
			return err
		}
		machine, err := ufsClient.GetMachine(ctx, &ufsapi.GetMachineRequest{
			Name: ufsutil.AddPrefix(ufsutil.MachineCollection, labSetup.GetMachines()[0]),
		})
		if err != nil {
			return err
		}
		if json {
			printDUTInfoAsJSON(labSetup, machine, w)
		} else {
			printDUTInfoAsBashVariables(labSetup, machine, w)
		}
		fmt.Fprintf(w, "\n")
	}
	return nil
}

// newUFSClient returns a new client to interact with the Unified Fleet System.
func newUFSClient(ctx context.Context, ufsService string, authFlags *authcli.Flags) (ufsapi.FleetClient, error) {
	httpClient, err := cmdlib.NewHTTPClient(ctx, authFlags)
	if err != nil {
		return nil, err
	}
	return ufsapi.NewFleetPRPCClient(&prpc.Client{
		C:       httpClient,
		Host:    ufsService,
		Options: site.DefaultPRPCOptions,
	}), nil
}

func printDUTInfoAsBashVariables(labSetup *ufspb.MachineLSE, machine *ufspb.Machine, w io.Writer) {
	dut := labSetup.GetChromeosMachineLse().GetDeviceLse().GetDut()
	fmt.Fprintf(w, "DUT_HOSTNAME=%s.cros\n", dut.Hostname)
	fmt.Fprintf(w, "MODEL=%s\n", machine.GetChromeosMachine().GetModel())
	fmt.Fprintf(w, "BOARD=%s\n", machine.GetChromeosMachine().GetBuildTarget())

	servo := dut.GetPeripherals().GetServo()
	if servo == nil {
		return
	}
	fmt.Fprintf(w, "SERVO_HOSTNAME=%s\n", servo.GetServoHostname())
	fmt.Fprintf(w, "SERVO_PORT=%d\n", servo.GetServoPort())
	fmt.Fprintf(w, "SERVO_SERIAL=%s\n", servo.GetServoSerial())
	return
}

func printDUTInfoAsJSON(labSetup *ufspb.MachineLSE, machine *ufspb.Machine, w io.Writer) {
	fmt.Fprintf(w, "{\"MachineLSE\":\t%s,\n", protoJSON(labSetup))
	fmt.Fprintf(w, "\"Machine\":\t%s}\n", protoJSON(machine))
	return
}

func protoJSON(message proto.Message) []byte {
	marshalOpts := protojson.MarshalOptions{
		EmitUnpopulated: false,
		Indent:          "\t",
	}
	json, err := marshalOpts.Marshal(proto.MessageV2(message))
	if err != nil {
		panic("Failed to marshal JSON")
	}
	return json
}

func contextWithOSNamespace(ctx context.Context) context.Context {
	osMetadata := metadata.Pairs(ufsutil.Namespace, ufsutil.OSNamespace)
	return metadata.NewOutgoingContext(ctx, osMetadata)
}
