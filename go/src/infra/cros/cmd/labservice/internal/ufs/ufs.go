// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package ufs

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	labapi "go.chromium.org/chromiumos/config/go/test/lab/api"

	ufspb "infra/unifiedfleet/api/v1/models"
	lab "infra/unifiedfleet/api/v1/models/chromeos/lab"
	ufsapi "infra/unifiedfleet/api/v1/rpc"
)

// GetDutTopology returns a DutTopology constructed from UFS.
// The returned error, if any, has gRPC status information.
func GetDutTopology(ctx context.Context, c ufsapi.FleetClient, id string) (*labapi.DutTopology, error) {
	// Assume the "scheduling unit" is a single DUT first.
	// If we can't find the DUT, try to look up the scheduling
	// unit for multiple DUT setups.
	dd, err := c.GetChromeOSDeviceData(ctx, &ufsapi.GetChromeOSDeviceDataRequest{Hostname: id})
	if err != nil {
		if s, ok := status.FromError(err); ok && s.Code() == codes.NotFound {
			return getSchedulingUnitDutTopology(ctx, c, id)
		} else {
			// Use the gRPC status from the UFS call.
			return nil, err
		}
	}
	lse := dd.GetLabConfig()
	d, err := makeDutProto(lse)
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "ID %q: %s", id, err)
	}
	dt := makeTopology(id)
	dt.Duts = append(dt.Duts, d)
	return dt, nil
}

func makeTopology(id string) *labapi.DutTopology {
	return &labapi.DutTopology{
		Id: &labapi.DutTopology_Id{Value: id},
	}
}

// getSchedulingUnitDutTopology returns a DutTopology constructed from
// UFS for a scheduling unit.
// The returned error, if any, has gRPC status information.
// You should use getDutTopology instead of this.
func getSchedulingUnitDutTopology(ctx context.Context, c ufsapi.FleetClient, id string) (*labapi.DutTopology, error) {
	resp, err := c.GetSchedulingUnit(ctx, &ufsapi.GetSchedulingUnitRequest{Name: id})
	if err != nil {
		// Use the gRPC status from the UFS call.
		return nil, err
	}
	dt := makeTopology(id)
	for _, name := range resp.GetMachineLSEs() {
		lse, err := c.GetMachineLSE(ctx, &ufsapi.GetMachineLSERequest{Name: name})
		if err != nil {
			return nil, status.Errorf(codes.FailedPrecondition, "%s", err)
		}
		d, err := makeDutProto(lse)
		if err != nil {
			return nil, status.Errorf(codes.FailedPrecondition, "ID %q: %s", id, err)
		}
		dt.Duts = append(dt.Duts, d)
	}
	return dt, nil
}

// makeDutProto makes a DutTopology Dut protobuf.
func makeDutProto(lse *ufspb.MachineLSE) (*labapi.Dut, error) {
	hostname := lse.GetHostname()
	if hostname == "" {
		return nil, errors.New("make dut proto: empty hostname")
	}
	croslse := lse.GetChromeosMachineLse()
	if croslse == nil {
		return nil, errors.New("make dut proto: empty chromeos_machine_lse")
	}
	dlse := croslse.GetDeviceLse()
	if dlse == nil {
		return nil, errors.New("make dut proto: empty device_lse")
	}
	d := dlse.GetDut()
	if d == nil {
		return nil, errors.New("make dut proto: empty dut")
	}
	p := d.GetPeripherals()
	if p == nil {
		return nil, errors.New("make dut proto: empty peripherals")
	}

	return &labapi.Dut{
		Id: &labapi.Dut_Id{Value: hostname},
		DutType: &labapi.Dut_Chromeos{
			Chromeos: &labapi.Dut_ChromeOS{
				Ssh: &labapi.IpEndpoint{
					Address: hostname,
					Port:    22,
				},
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
	res := []labapi.Chameleon_Peripheral{}
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
	ret := []*labapi.Cable{}
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
