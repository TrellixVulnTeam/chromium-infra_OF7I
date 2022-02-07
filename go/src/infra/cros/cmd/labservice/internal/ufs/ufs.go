// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package ufs

import (
	"context"
	"errors"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	labapi "go.chromium.org/chromiumos/config/go/test/lab/api"

	ufspb "infra/unifiedfleet/api/v1/models"
	lab "infra/unifiedfleet/api/v1/models/chromeos/lab"
	ufsapi "infra/unifiedfleet/api/v1/rpc"
)

type DeviceType int64

const (
	ChromeOSDevice DeviceType = iota
	AndroidDevice
)

type deviceInfo struct {
	deviceType DeviceType
	machine    *ufspb.Machine
	machineLse *ufspb.MachineLSE
}

// GetDutTopology returns a DutTopology constructed from UFS.
// The returned error, if any, has gRPC status information.
func GetDutTopology(ctx context.Context, c ufsapi.FleetClient, id string) (*labapi.DutTopology, error) {
	deviceInfos, err := getAllDevicesInfo(ctx, c, id)
	if err != nil {
		return nil, err
	}
	dt := &labapi.DutTopology{
		Id: &labapi.DutTopology_Id{Value: id},
	}
	for _, deviceInfo := range deviceInfos {
		d, err := makeDutProto(deviceInfo)
		if err != nil {
			return nil, status.Errorf(codes.FailedPrecondition, "ID %q: %s", id, err)
		}
		dt.Duts = append(dt.Duts, d)
	}
	return dt, nil
}

// getAllDevicesInfo fetches inventory entry of all DUTs / attached devices by a resource name.
func getAllDevicesInfo(ctx context.Context, client ufsapi.FleetClient, resourceName string) ([]*deviceInfo, error) {
	resp, err := getDeviceData(ctx, client, resourceName)
	if err != nil {
		return nil, err
	}
	if resp.GetResourceType() == ufsapi.GetDeviceDataResponse_RESOURCE_TYPE_SCHEDULING_UNIT {
		return getSchedulingUnitInfo(ctx, client, resp.GetSchedulingUnit().GetMachineLSEs())
	}
	deviceInfos, err := appendDeviceInfo([]*deviceInfo{}, resp)
	if err != nil {
		return nil, fmt.Errorf("%w for %s", err, resourceName)
	}
	return deviceInfos, nil
}

// getSchedulingUnitInfo fetches device info for every DUT / attached device in the scheduling unit.
func getSchedulingUnitInfo(ctx context.Context, client ufsapi.FleetClient, hostnames []string) ([]*deviceInfo, error) {
	// Get device info for every DUT / attached device in the scheduling unit.
	var deviceInfos []*deviceInfo
	for _, hostname := range hostnames {
		resp, err := getDeviceData(ctx, client, hostname)
		if err != nil {
			return nil, err
		}
		if deviceInfos, err = appendDeviceInfo(deviceInfos, resp); err != nil {
			return nil, fmt.Errorf("%w for %s", err, hostname)
		}
	}
	return deviceInfos, nil
}

// getDeviceData fetches a device entry.
func getDeviceData(ctx context.Context, client ufsapi.FleetClient, id string) (*ufsapi.GetDeviceDataResponse, error) {
	resp, err := client.GetDeviceData(ctx, &ufsapi.GetDeviceDataRequest{Hostname: id})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// appendDeviceData appends a device data response to the list of responses
// after validation. Returns error if the device type is different from ChromeOs
// or Android device.
func appendDeviceInfo(deviceInfos []*deviceInfo, resp *ufsapi.GetDeviceDataResponse) ([]*deviceInfo, error) {
	switch resp.GetResourceType() {
	case ufsapi.GetDeviceDataResponse_RESOURCE_TYPE_CHROMEOS_DEVICE:
		return append(deviceInfos, &deviceInfo{
			deviceType: ChromeOSDevice,
			machine:    resp.GetChromeOsDeviceData().GetMachine(),
			machineLse: resp.GetChromeOsDeviceData().GetLabConfig(),
		}), nil
	case ufsapi.GetDeviceDataResponse_RESOURCE_TYPE_ATTACHED_DEVICE:
		return append(deviceInfos, &deviceInfo{
			deviceType: AndroidDevice,
			machine:    resp.GetAttachedDeviceData().GetMachine(),
			machineLse: resp.GetAttachedDeviceData().GetLabConfig(),
		}), nil
	}
	return nil, fmt.Errorf("append device info: invalid device type (%s)", resp.GetResourceType())
}

// makeDutProto makes a DutTopology Dut protobuf.
func makeDutProto(di *deviceInfo) (*labapi.Dut, error) {
	switch di.deviceType {
	case ChromeOSDevice:
		return makeChromeOsDutProto(di)
	case AndroidDevice:
		return makeAndroidDutProto(di)
	}
	return nil, errors.New("make dut proto: invalid device type for " + di.machineLse.GetHostname())
}

// makeChromeOsDutProto populates DutTopology proto for ChromeOS device.
func makeChromeOsDutProto(di *deviceInfo) (*labapi.Dut, error) {
	lse := di.machineLse
	hostname := lse.GetHostname()
	if hostname == "" {
		return nil, errors.New("make chromeos dut proto: empty hostname")
	}
	croslse := lse.GetChromeosMachineLse()
	if croslse == nil {
		return nil, errors.New("make chromeos dut proto: empty chromeos_machine_lse")
	}
	dlse := croslse.GetDeviceLse()
	if dlse == nil {
		return nil, errors.New("make chromeos dut proto: empty device_lse")
	}
	d := dlse.GetDut()
	if d == nil {
		return nil, errors.New("make chromeos dut proto: empty dut")
	}
	p := d.GetPeripherals()
	if p == nil {
		return nil, errors.New("make chromeos dut proto: empty peripherals")
	}

	return &labapi.Dut{
		Id: &labapi.Dut_Id{Value: hostname},
		DutType: &labapi.Dut_Chromeos{
			Chromeos: &labapi.Dut_ChromeOS{
				Ssh: &labapi.IpEndpoint{
					Address: hostname,
					Port:    22,
				},
				DutModel:  getDutModel(di),
				Servo:     getServo(p),
				Chameleon: getChameleon(p),
				Audio:     getAudio(p),
				Wifi:      getWifi(p),
				Touch:     getTouch(p),
				Camerabox: getCamerabox(p),
				Cables:    getCables(p),
			},
		},
	}, nil
}

// makeAndroidDutProto populates DutTopology proto for Android device.
func makeAndroidDutProto(di *deviceInfo) (*labapi.Dut, error) {
	machine := di.machine
	lse := di.machineLse
	hostname := lse.GetHostname()
	if hostname == "" {
		return nil, errors.New("make android dut proto: empty hostname")
	}
	androidLse := lse.GetAttachedDeviceLse()
	if androidLse == nil {
		return nil, errors.New("make android dut proto: empty attached_device_lse")
	}
	associatedHostname := androidLse.GetAssociatedHostname()
	if associatedHostname == "" {
		return nil, errors.New("make android dut proto: empty associated_hostname")
	}
	serialNumber := machine.GetSerialNumber()
	if serialNumber == "" {
		return nil, errors.New("make android dut proto: empty serial_number")
	}
	return &labapi.Dut{
		Id: &labapi.Dut_Id{Value: hostname},
		DutType: &labapi.Dut_Android_{
			Android: &labapi.Dut_Android{
				AssociatedHostname: &labapi.IpEndpoint{
					Address: associatedHostname,
				},
				Name:         hostname,
				SerialNumber: serialNumber,
				DutModel:     getDutModel(di),
			},
		},
	}, nil
}

func getDutModel(di *deviceInfo) *labapi.DutModel {
	machine := di.machine
	if di.deviceType == ChromeOSDevice {
		return &labapi.DutModel{
			BuildTarget: machine.GetChromeosMachine().GetBuildTarget(),
			ModelName:   machine.GetChromeosMachine().GetModel(),
		}
	}
	return &labapi.DutModel{
		BuildTarget: machine.GetAttachedDevice().GetBuildTarget(),
		ModelName:   machine.GetAttachedDevice().GetModel(),
	}
}

func getServo(p *lab.Peripherals) *labapi.Servo {
	s := p.GetServo()
	if s != nil && s.GetServoHostname() != "" {
		return &labapi.Servo{
			ServodAddress: &labapi.IpEndpoint{
				Address: s.GetServoHostname(),
				Port:    s.GetServoPort(),
			},
		}
	}
	return nil
}

func getChameleon(p *lab.Peripherals) *labapi.Chameleon {
	c := p.GetChameleon()
	if c == nil {
		return nil
	}
	return &labapi.Chameleon{
		AudioBoard:  c.GetAudioBoard(),
		Peripherals: mapChameleonPeripherals(p, c),
	}
}

func mapChameleonPeripherals(p *lab.Peripherals, c *lab.Chameleon) []labapi.Chameleon_Peripheral {
	var res []labapi.Chameleon_Peripheral
	for _, cp := range c.GetChameleonPeripherals() {
		m := labapi.Chameleon_PREIPHERAL_UNSPECIFIED
		switch cp {
		case lab.ChameleonType_CHAMELEON_TYPE_INVALID:
			m = labapi.Chameleon_PREIPHERAL_UNSPECIFIED
		case lab.ChameleonType_CHAMELEON_TYPE_DP:
			m = labapi.Chameleon_DP
		case lab.ChameleonType_CHAMELEON_TYPE_DP_HDMI:
			m = labapi.Chameleon_DP_HDMI
		case lab.ChameleonType_CHAMELEON_TYPE_VGA:
			m = labapi.Chameleon_VGA
		case lab.ChameleonType_CHAMELEON_TYPE_HDMI:
			m = labapi.Chameleon_HDMI
			// TODO(ivanbrovkovich): are these not mapped to anything?
			// BT_BLE_HID
			// BT_A2DP_SINK
			// BT_PEER
		}
		res = append(res, m)
	}
	return res
}

func getAudio(p *lab.Peripherals) *labapi.Audio {
	a := p.GetAudio()
	if a == nil {
		return nil
	}
	return &labapi.Audio{
		AudioBox: a.AudioBox,
		Atrus:    a.Atrus,
	}
}

func getWifi(p *lab.Peripherals) *labapi.Wifi {
	res := &labapi.Wifi{}
	w := p.GetWifi()
	if p.GetChaos() {
		res.Environment = labapi.Wifi_ROUTER_802_11AX
	} else if w != nil && w.GetWificell() {
		res.Environment = labapi.Wifi_WIFI_CELL
	} else if w != nil && w.GetRouter() == lab.Wifi_ROUTER_802_11AX {
		res.Environment = labapi.Wifi_ROUTER_802_11AX
	} else if w != nil {
		res.Environment = labapi.Wifi_STANDARD
	}
	// TODO(ivanbrovkovich): Do we still get antenna for Chaos and wificell?
	if w != nil {
		res.Antenna = &labapi.WifiAntenna{
			Connection: mapWifiAntenna(w.GetAntennaConn()),
		}
	}
	return res
}

func mapWifiAntenna(wa lab.Wifi_AntennaConnection) labapi.WifiAntenna_Connection {
	switch wa {
	case lab.Wifi_CONN_UNKNOWN:
		return labapi.WifiAntenna_CONNECTION_UNSPECIFIED
	case lab.Wifi_CONN_CONDUCTIVE:
		return labapi.WifiAntenna_CONDUCTIVE
	case lab.Wifi_CONN_OTA:
		return labapi.WifiAntenna_OTA
	}
	return labapi.WifiAntenna_CONNECTION_UNSPECIFIED
}

func getTouch(p *lab.Peripherals) *labapi.Touch {
	if t := p.GetTouch(); t != nil {
		return &labapi.Touch{
			Mimo: t.GetMimo(),
		}
	}
	return nil
}

func getCamerabox(p *lab.Peripherals) *labapi.Camerabox {
	if !p.GetCamerabox() {
		return nil
	}
	cb := p.GetCameraboxInfo()
	return &labapi.Camerabox{
		Facing: mapCameraFacing(cb.Facing),
	}
}

func mapCameraFacing(cf lab.Camerabox_Facing) labapi.Camerabox_Facing {
	switch cf {
	case lab.Camerabox_FACING_UNKNOWN:
		return labapi.Camerabox_FACING_UNSPECIFIED
	case lab.Camerabox_FACING_BACK:
		return labapi.Camerabox_BACK
	case lab.Camerabox_FACING_FRONT:
		return labapi.Camerabox_FRONT
	}
	return labapi.Camerabox_FACING_UNSPECIFIED
}

func getCables(p *lab.Peripherals) []*labapi.Cable {
	var ret []*labapi.Cable
	for _, c := range p.GetCable() {
		ret = append(ret, &labapi.Cable{
			Type: mapCables(c.GetType()),
		})
	}
	return ret
}

func mapCables(ct lab.CableType) labapi.Cable_Type {
	switch ct {
	case lab.CableType_CABLE_INVALID:
		return labapi.Cable_TYPE_UNSPECIFIED
	case lab.CableType_CABLE_AUDIOJACK:
		return labapi.Cable_AUDIOJACK
	case lab.CableType_CABLE_USBAUDIO:
		return labapi.Cable_USBAUDIO
	case lab.CableType_CABLE_USBPRINTING:
		return labapi.Cable_USBPRINTING
	case lab.CableType_CABLE_HDMIAUDIO:
		return labapi.Cable_HDMIAUDIO
	}
	return labapi.Cable_TYPE_UNSPECIFIED
}
