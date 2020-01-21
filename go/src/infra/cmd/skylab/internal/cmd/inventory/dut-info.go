// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package inventory

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"

	invV2Api "infra/appengine/cros/lab_inventory/api/v1"
	fleet "infra/appengine/crosskylabadmin/api/fleet/v1"
	skycmdlib "infra/cmd/skylab/internal/cmd/cmdlib"
	"infra/cmd/skylab/internal/site"
	"infra/cmdsupport/cmdlib"
	"infra/libs/skylab/inventory"
)

// DutInfo subcommand: Get DUT inventory information
var DutInfo = &subcommands.Command{
	UsageLine: "dut-info [-json] [-full] HOSTNAME",
	ShortDesc: "print Device Under Test inventory information",
	LongDesc: `Print Device Under Test inventory information.

By default, only the most commonly used information is printed in a
human-readable format. This format may change without prior notice.

If you need a stable output format, use -json, which dumps a JSON-encoded
serialization of the inventory.DeviceUnderTest protobuf.

The JSON-encoding for protobuf messages is described at
https://developers.google.com/protocol-buffers/docs/proto3#json

The protobuf definition of inventory.DeviceUnderTest is part of
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/
    libs/skylab/inventory/device.proto`,
	CommandRun: func() subcommands.CommandRun {
		c := &dutInfoRun{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)

		c.Flags.BoolVar(&c.asJSON, "json", false, "Print inventory as JSON-encoded protobuf. Implies -full.")
		c.Flags.BoolVar(&c.full, "full", false, "Print full DUT information, including less frequently used fields.")
		c.Flags.BoolVar(&c.v2, "v2", false, "[INTERNAL ONLY] Use ChromeOS Lab inventory v2 service.")
		return c
	},
}

type dutInfoRun struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
	envFlags  skycmdlib.EnvFlags

	asJSON bool
	full   bool
	v2     bool
}

func (c *dutInfoRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, errors.Annotate(err, "dut-info").Err())
		return 1
	}
	return 0
}

func (c *dutInfoRun) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	if len(args) != 1 {
		return cmdlib.NewUsageError(c.Flags, "want 1 argument, have %d", len(args))
	}

	ctx := cli.GetContext(a, c, env)
	hc, err := cmdlib.NewHTTPClient(ctx, &c.authFlags)
	if err != nil {
		return err
	}
	e := c.envFlags.Env()
	ic := NewInventoryClient(hc, e, c.v2)
	dut, err := ic.GetDutInfo(ctx, args[0], true)

	if err != nil {
		return err
	}

	bw := bufio.NewWriter(os.Stdout)
	defer bw.Flush()
	switch {
	case c.asJSON:
		return printProtoJSON(bw, dut)
	case c.full:
		return printHumanizedInfoFull(bw, dut)
	default:
		return printHumanizedInfoShort(bw, dut)
	}
}

func (client *inventoryClientV1) GetDutInfo(ctx context.Context, id string, byHostname bool) (*inventory.DeviceUnderTest, error) {
	req := &fleet.GetDutInfoRequest{Id: id}
	if byHostname {
		req = &fleet.GetDutInfoRequest{Hostname: id}
	}
	resp, err := client.ic.GetDutInfo(ctx, req)
	if err != nil {
		return nil, errors.Annotate(err, "get dutinfo for %s", id).Err()
	}
	var dut inventory.DeviceUnderTest
	if err := proto.Unmarshal(resp.Spec, &dut); err != nil {
		return nil, errors.Annotate(err, "get dutinfo for %s", id).Err()
	}
	return &dut, nil
}

// GetDutInfo gets the dut information from inventory v2 service.
func (client *inventoryClientV2) GetDutInfo(ctx context.Context, id string, byHostname bool) (*inventory.DeviceUnderTest, error) {
	devID := &invV2Api.DeviceID{Id: &invV2Api.DeviceID_ChromeosDeviceId{ChromeosDeviceId: id}}
	if byHostname {
		devID = &invV2Api.DeviceID{Id: &invV2Api.DeviceID_Hostname{Hostname: id}}
	}
	rsp, err := client.ic.GetCrosDevices(ctx, &invV2Api.GetCrosDevicesRequest{
		Ids: []*invV2Api.DeviceID{devID},
	})
	if err != nil {
		return nil, errors.Annotate(err, "[v2] get dutinfo for %s", id).Err()
	}
	if len(rsp.FailedDevices) > 0 {
		result := rsp.FailedDevices[0]
		return nil, errors.Reason("[v2] failed to get device %s: %s", result.Hostname, result.ErrorMsg).Err()
	}
	return invV2Api.AdaptToV1DutSpec(rsp.Data[0])
}

// printHumanizedInfoShort prints the most commonly used dut information in a
// human-readable format.
//
// This function modifies dut to remove the already printed information.
func printHumanizedInfoShort(w io.Writer, dut *inventory.DeviceUnderTest) (err error) {
	tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
	defer func() {
		err = tw.Flush()
	}()

	c := dut.GetCommon()
	fmt.Fprintf(tw, "Hostname:\t%s\n", c.GetHostname())
	c.Hostname = nil
	fmt.Fprintf(tw, "Inventory Id:\t%s\n", c.GetId())
	c.Id = nil

	l := c.GetLabels()
	if l.GetModel() != "" {
		fmt.Fprintf(tw, "Model:\t%s\n", l.GetModel())
		l.Model = nil
	}
	if l.GetBoard() != "" {
		fmt.Fprintf(tw, "Board:\t%s\n", l.GetBoard())
		l.Board = nil
	}
	if l.GetReferenceDesign() != "" {
		fmt.Fprintf(tw, "ReferenceDesign:\t%s\n", l.GetReferenceDesign())
		l.ReferenceDesign = nil
	}
	if len(l.GetVariant()) > 0 {
		fmt.Fprintf(tw, "Variant:\t%s\n", strings.Join(l.GetVariant(), ", "))
		l.Variant = nil
	}

	c, sa := extractServoAttributes(c)
	if len(sa) > 0 {
		fmt.Fprintf(tw, "Servo attributes:\n")
		for k, v := range sa {
			fmt.Fprintf(tw, "\t%s\t%s\n", k, v)
		}
	}
	return nil
}

// printHumanizedInfoFull prints all the dut information in a human-readable
// format.
func printHumanizedInfoFull(w io.Writer, dut *inventory.DeviceUnderTest) error {
	if err := printHumanizedInfoShort(w, dut); err != nil {
		return err
	}
	fmt.Fprintf(w, "\nOther inventory data:\n")
	// TODO(pprabhu) Use printProtoJSON once all protobuf fields have been made
	// optional. There is no clean way to skip printing Hostname and other
	// required fields currently.
	return printProtoText(w, dut)
}

func printProtoText(w io.Writer, dut *inventory.DeviceUnderTest) error {
	return proto.MarshalText(w, dut)
}

func printProtoJSON(w io.Writer, dut *inventory.DeviceUnderTest) error {
	m := jsonpb.Marshaler{
		EnumsAsInts: false,
		Indent:      "\t",
	}
	return m.Marshal(w, dut)
}

var servoAttributeKeys = map[string]bool{
	"servo_host":   true,
	"servo_port":   true,
	"servo_serial": true,
}

func extractServoAttributes(c *inventory.CommonDeviceSpecs) (*inventory.CommonDeviceSpecs, map[string]string) {
	sa := make(map[string]string)
	others := make([]*inventory.KeyValue, 0, len(c.GetAttributes()))
	for _, kv := range c.GetAttributes() {
		if servoAttributeKeys[*kv.Key] {
			sa[*kv.Key] = *kv.Value
		} else {
			others = append(others, kv)
		}
	}
	c.Attributes = others
	return c, sa
}
