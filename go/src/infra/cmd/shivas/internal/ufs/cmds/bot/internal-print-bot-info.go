// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmds

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/grpc/prpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"infra/cmd/shivas/site"
	"infra/cmd/shivas/utils"
	suUtil "infra/cmd/shivas/utils/schedulingunit"
	"infra/cmdsupport/cmdlib"
	"infra/cros/dutstate"
	"infra/libs/skylab/inventory/swarming"
	"infra/libs/skylab/inventory/swarming/attacheddevice"
	ufspb "infra/unifiedfleet/api/v1/models"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	ufsUtil "infra/unifiedfleet/app/util"
)

// PrintBotInfo subcommand: Print Swarming dimensions for a DUT.
var PrintBotInfo = &subcommands.Command{
	UsageLine: "internal-print-bot-info DUT hostname/Asset tag",
	ShortDesc: "print Swarming bot info for a DUT",
	LongDesc: `Print Swarming bot info for a DUT.

For internal use only.`,
	CommandRun: func() subcommands.CommandRun {
		c := &printBotInfoRun{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.commonFlags.Register(&c.Flags)

		c.Flags.BoolVar(&c.byHostname, "by-hostname", false, "Lookup by hostname instead of ID/Asset tag.")
		return c
	},
}

type printBotInfoRun struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags

	byHostname bool
}

func (c *printBotInfoRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		fmt.Fprintf(a.GetErr(), "%s: %s\n", a.GetName(), err)
		return 1
	}
	return 0
}

func (c *printBotInfoRun) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	if len(args) != 1 {
		return cmdlib.NewUsageError(c.Flags, "exactly one DUT hostname must be provided")
	}
	ctx := cli.GetContext(a, c, env)
	hc, err := cmdlib.NewHTTPClient(ctx, &c.authFlags)
	if err != nil {
		return err
	}
	e := c.envFlags.Env()
	ns, err := c.envFlags.Namespace()
	if err != nil {
		// Set namespace to OS namespace for whatever errors.
		ns = ufsUtil.OSNamespace
	}
	if c.commonFlags.Verbose() {
		fmt.Printf("Using UnifiedFleet service %s (namespace %s)\n", e.UnifiedFleetService, ns)
	}
	ufsClient := ufsAPI.NewFleetPRPCClient(&prpc.Client{
		C:       hc,
		Host:    e.UnifiedFleetService,
		Options: site.DefaultPRPCOptions,
	})
	stderr := a.GetErr()
	r := func(e error) { fmt.Fprintf(stderr, "sanitize dimensions: %s\n", err) }
	var bi *botInfo

	if ns == ufsUtil.BrowserNamespace {
		ctx = utils.SetupContext(ctx, ufsUtil.BrowserNamespace)
		if bi, err = getBrowserBotInfo(ctx, ufsClient, args[0]); err != nil {
			return err
		}
	} else {
		ctx = utils.SetupContext(ctx, ufsUtil.OSNamespace)
		if bi, err = getOSBotInfo(ctx, ufsClient, args[0], c.byHostname, r); err != nil {
			return err
		}
	}

	// Post-processing
	enc, err := json.Marshal(bi)
	if err != nil {
		return err
	}
	a.GetOut().Write(enc)
	return nil
}

type botInfo struct {
	Dimensions swarming.Dimensions
	State      botState
}

type botState map[string][]string

func getBrowserBotInfo(ctx context.Context, client ufsAPI.FleetClient, id string) (*botInfo, error) {
	// id is the hostname by default for browser bots
	resp, err := getDeviceData(ctx, client, id, true)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, errors.New(fmt.Sprintf("no browser device data for host %q", id))
		}
		return nil, err
	}
	var state string
	var zone string
	if resp.GetBrowserDeviceData().GetHost() != nil {
		state = dutstate.ConvertFromUFSState(resp.GetBrowserDeviceData().GetHost().GetResourceState()).String()
		zone = resp.GetBrowserDeviceData().GetHost().GetZone()
	} else {
		state = dutstate.ConvertFromUFSState(resp.GetBrowserDeviceData().GetVm().GetResourceState()).String()
		zone = resp.GetBrowserDeviceData().GetVm().GetZone()
	}
	return &botInfo{
		Dimensions: map[string][]string{
			"ufs_state": {state},
			// Duplicate state to dut_state to reuse analytics logic built for ChromeOS lab
			"dut_state": {state},
			"ufs_zone":  {zone},
		},
	}, nil
}

func getOSBotInfo(ctx context.Context, client ufsAPI.FleetClient, id string, byHostname bool, r swarming.ReportFunc) (*botInfo, error) {
	resp, err := getDeviceData(ctx, client, id, byHostname)
	if err != nil {
		return nil, err
	}
	if resp.GetResourceType() == ufsAPI.GetDeviceDataResponse_RESOURCE_TYPE_SCHEDULING_UNIT {
		return getSUBotInfo(ctx, client, resp.GetSchedulingUnit(), r)
	}
	botDimensions, err := getBotDimensions(ctx, client, resp, r)
	if err != nil {
		return nil, err
	}
	botState, err := getBotState(resp)
	if err != nil {
		return nil, err
	}
	return &botInfo{Dimensions: botDimensions, State: botState}, nil
}

func getSUBotInfo(ctx context.Context, client ufsAPI.FleetClient, su *ufspb.SchedulingUnit, r swarming.ReportFunc) (*botInfo, error) {
	var dutsDims []swarming.Dimensions
	for _, hostname := range su.GetMachineLSEs() {
		resp, err := getDeviceData(ctx, client, hostname, true)
		if err != nil {
			return nil, err
		}
		botDimensions, err := getBotDimensions(ctx, client, resp, r)
		if err != nil {
			return nil, err
		}
		dutsDims = append(dutsDims, botDimensions)
	}
	return &botInfo{
		Dimensions: suUtil.SchedulingUnitDimensions(su, dutsDims),
		State:      suUtil.SchedulingUnitBotState(su),
	}, nil
}

func getDeviceData(ctx context.Context, client ufsAPI.FleetClient, id string, byHostname bool) (*ufsAPI.GetDeviceDataResponse, error) {
	req := &ufsAPI.GetDeviceDataRequest{}
	if byHostname {
		req.Hostname = id
	} else {
		req.DeviceId = id
	}
	return client.GetDeviceData(ctx, req)
}

func getBotState(deviceData *ufsAPI.GetDeviceDataResponse) (botState, error) {
	switch deviceData.GetResourceType() {
	case ufsAPI.GetDeviceDataResponse_RESOURCE_TYPE_CHROMEOS_DEVICE:
		return getDUTBotState(deviceData.GetChromeOsDeviceData()), nil
	case ufsAPI.GetDeviceDataResponse_RESOURCE_TYPE_ATTACHED_DEVICE:
		return getAttachedDeviceBotState(deviceData.GetAttachedDeviceData()), nil
	}
	return nil, fmt.Errorf("get bot state: invalid device type (%s)", deviceData.GetResourceType())
}

func getDUTBotState(deviceData *ufspb.ChromeOSDeviceData) botState {
	d := deviceData.GetDutV1()
	s := make(botState)
	for _, kv := range d.GetCommon().GetAttributes() {
		k, v := kv.GetKey(), kv.GetValue()
		s[k] = append(s[k], v)
	}
	s["storage_state"] = []string{d.GetCommon().GetLabels().GetPeripherals().GetStorageState().String()[len("HARDWARE_"):]}
	s["servo_usb_state"] = []string{d.GetCommon().GetLabels().GetPeripherals().GetServoUsbState().String()[len("HARDWARE_"):]}
	s["battery_state"] = []string{d.GetCommon().GetLabels().GetPeripherals().GetBatteryState().String()[len("HARDWARE_"):]}
	s["wifi_state"] = []string{d.GetCommon().GetLabels().GetPeripherals().GetWifiState().String()[len("HARDWARE_"):]}
	s["bluetooth_state"] = []string{d.GetCommon().GetLabels().GetPeripherals().GetBluetoothState().String()[len("HARDWARE_"):]}
	s["rpm_state"] = []string{d.GetCommon().GetLabels().GetPeripherals().GetRpmState().String()}
	s["lab_config_version_index"] = []string{deviceData.GetLabConfig().GetUpdateTime().AsTime().Format(ufsUtil.TimestampBasedVersionKeyFormat)}
	s["dut_state_version_index"] = []string{deviceData.GetDutState().GetUpdateTime().AsTime().Format(ufsUtil.TimestampBasedVersionKeyFormat)}
	return s
}

func getAttachedDeviceBotState(deviceData *ufsAPI.AttachedDeviceData) botState {
	s := make(botState)
	s["lab_config_version_index"] = []string{deviceData.GetLabConfig().GetUpdateTime().AsTime().Format(ufsUtil.TimestampBasedVersionKeyFormat)}
	s["dut_state_version_index"] = []string{deviceData.GetDutState().GetUpdateTime().AsTime().Format(ufsUtil.TimestampBasedVersionKeyFormat)}
	return s
}

func getBotDimensions(ctx context.Context, client ufsAPI.FleetClient, deviceData *ufsAPI.GetDeviceDataResponse, r swarming.ReportFunc) (swarming.Dimensions, error) {
	switch deviceData.GetResourceType() {
	case ufsAPI.GetDeviceDataResponse_RESOURCE_TYPE_CHROMEOS_DEVICE:
		dutState := dutstate.Read(ctx, client, deviceData.GetChromeOsDeviceData().GetLabConfig().GetName())
		return getDUTBotDimensions(deviceData.GetChromeOsDeviceData(), dutState, r), nil
	case ufsAPI.GetDeviceDataResponse_RESOURCE_TYPE_ATTACHED_DEVICE:
		dutState := dutstate.Read(ctx, client, deviceData.GetAttachedDeviceData().GetLabConfig().GetName())
		return getAttachedDeviceBotDimensions(deviceData.GetAttachedDeviceData(), dutState, r), nil
	}
	return nil, fmt.Errorf("append bot dimensions: invalid device type (%s)", deviceData.GetResourceType())
}

func getDUTBotDimensions(deviceData *ufspb.ChromeOSDeviceData, ds dutstate.Info, r swarming.ReportFunc) swarming.Dimensions {
	c := deviceData.GetDutV1().GetCommon()
	dims := swarming.Convert(c.GetLabels())
	dims["dut_id"] = []string{c.GetId()}
	dims["dut_name"] = []string{c.GetHostname()}
	if v := c.GetHwid(); v != "" {
		dims["hwid"] = []string{v}
	}
	if v := c.GetSerialNumber(); v != "" {
		dims["serial_number"] = []string{v}
	}
	if v := c.GetLocation(); v != nil {
		location := fmt.Sprintf("%s-row%d-rack%d-host%d",
			v.GetLab().GetName(), v.GetRow(), v.GetRack(), v.GetHost())
		dims["location"] = []string{location}
	}
	dims["dut_state"] = []string{string(ds.State)}
	dims["ufs_zone"] = []string{deviceData.GetLabConfig().GetZone()}
	swarming.Sanitize(dims, r)
	return dims
}

func getAttachedDeviceBotDimensions(deviceData *ufsAPI.AttachedDeviceData, ds dutstate.Info, r swarming.ReportFunc) swarming.Dimensions {
	dims := attacheddevice.Convert(deviceData)
	if v := deviceData.GetMachine().GetLocation(); v != nil {
		location := fmt.Sprintf("%s-aisle%s-row%s-rack%s-racknumber%s-shelf%s-position%s",
			v.GetAisle(), v.GetZone(), v.GetRow(), v.GetRack(), v.GetRackNumber(), v.GetShelf(), v.GetPosition())
		dims["location"] = []string{location}
	}
	dims["dut_state"] = []string{string(ds.State)}
	dims["ufs_zone"] = []string{deviceData.GetLabConfig().GetZone()}
	swarming.Sanitize(dims, r)
	return dims
}
