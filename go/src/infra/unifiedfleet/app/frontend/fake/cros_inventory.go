// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package fake

import (
	"context"

	"github.com/golang/protobuf/proto"
	device "go.chromium.org/chromiumos/infra/proto/go/device"
	lab "go.chromium.org/chromiumos/infra/proto/go/lab"
	manufacturing "go.chromium.org/chromiumos/infra/proto/go/manufacturing"
	"go.chromium.org/luci/common/errors"
	"google.golang.org/grpc"

	invV2Api "infra/appengine/cros/lab_inventory/api/v1"
)

// InventoryClient mocks the inventory v2 client
type InventoryClient struct {
}

var mockDUT = &lab.ChromeOSDevice_Dut{
	Dut: &lab.DeviceUnderTest{
		Hostname: "chromeos2-test_host",
		Pools:    []string{"DUT_POOL_QUOTA", "hotrod"},
		Peripherals: &lab.Peripherals{
			Servo: &lab.Servo{
				ServoHostname: "test_servo",
				ServoPort:     int32(9999),
				ServoSerial:   "test_servo_serial",
				ServoType:     "v3",
			},
			Chameleon: &lab.Chameleon{
				ChameleonPeripherals: []lab.ChameleonType{
					lab.ChameleonType_CHAMELEON_TYPE_DP,
					lab.ChameleonType_CHAMELEON_TYPE_HDMI,
				},
				AudioBoard: true,
			},
			Rpm: &lab.RPM{
				PowerunitName:   "test_power_unit_name",
				PowerunitOutlet: "test_power_unit_outlet",
			},
			ConnectedCamera: []*lab.Camera{
				{
					CameraType: lab.CameraType_CAMERA_HUDDLY,
				},
				{
					CameraType: lab.CameraType_CAMERA_PTZPRO2,
				},
			},
			Audio: &lab.Audio{
				AudioBox: true,
				Atrus:    true,
			},

			Touch: &lab.Touch{
				Mimo: true,
			},
			Carrier:   "att",
			Camerabox: false,
			CameraboxInfo: &lab.Camerabox{
				Facing: lab.Camerabox_FACING_BACK,
			},
			Chaos: true,
			Cable: []*lab.Cable{
				{
					Type: lab.CableType_CABLE_AUDIOJACK,
				},
				{
					Type: lab.CableType_CABLE_USBAUDIO,
				},
				{
					Type: lab.CableType_CABLE_USBPRINTING,
				},
				{
					Type: lab.CableType_CABLE_HDMIAUDIO,
				},
			},
			SmartUsbhub: true,
		},
	},
}

var mockLabstation = &lab.ChromeOSDevice_Labstation{
	Labstation: &lab.Labstation{
		Hostname: "test_servo",
		Servos: []*lab.Servo{{
			ServoHostname: "test_servo",
			ServoPort:     int32(9999),
			ServoSerial:   "test_servo_serial",
			ServoType:     "v3",
		}},
		Rpm: &lab.RPM{
			PowerunitName:   "test_power_unit_name",
			PowerunitOutlet: "test_power_unit_outlet3",
		},
		Pools: []string{"labstation_main"},
	},
}

var mockLabConfig = &invV2Api.ListCrosDevicesLabConfigResponse_LabConfig{
	Config: &lab.ChromeOSDevice{
		Id: &lab.ChromeOSDeviceID{
			Value: "test_dut",
		},
		SerialNumber: "test_serial",
		ManufacturingId: &manufacturing.ConfigID{
			Value: "test_hwid",
		},
		DeviceConfigId: &device.ConfigId{
			PlatformId: &device.PlatformId{
				Value: "coral",
			},
			ModelId: &device.ModelId{
				Value: "test_model",
			},
			VariantId: &device.VariantId{
				Value: "test_variant",
			},
		},
	},
}

var mockDutState = &lab.DutState{
	Id: &lab.ChromeOSDeviceID{
		Value: "",
	},
	Servo:                  lab.PeripheralState_WORKING,
	StorageState:           lab.HardwareState_HARDWARE_NORMAL,
	WorkingBluetoothBtpeer: 1,
	Cr50Phase:              lab.DutState_CR50_PHASE_PVT,
}

// ListCrosDevicesLabConfig mock the invV2Api.InventoryClient's ListCrosDevicesLabConfig
func (ic *InventoryClient) ListCrosDevicesLabConfig(ctx context.Context, in *invV2Api.ListCrosDevicesLabConfigRequest, opts ...grpc.CallOption) (*invV2Api.ListCrosDevicesLabConfigResponse, error) {
	cameraDUT := proto.Clone(mockDUT.Dut).(*lab.DeviceUnderTest)
	// Put it into acs lab
	cameraDUT.Hostname = "chromeos3-test_host"
	cameraDUT.Peripherals.Camerabox = true
	lcCamera := proto.Clone(mockLabConfig).(*invV2Api.ListCrosDevicesLabConfigResponse_LabConfig)
	lcCamera.Config.Device = &lab.ChromeOSDevice_Dut{Dut: cameraDUT}
	lcCamera.Config.Id.Value = "mock_camera_dut_id"
	cameraState := proto.Clone(mockDutState).(*lab.DutState)
	cameraState.Servo = lab.PeripheralState_SERVOD_ISSUE
	lcCamera.State = cameraState
	lcCamera.State.Id.Value = "mock_camera_dut_id"

	wifiDUT := proto.Clone(mockDUT.Dut).(*lab.DeviceUnderTest)
	// Put it into acs lab
	wifiDUT.Hostname = "chromeos5-test_host"
	wifiDUT.Peripherals.Wifi = &lab.Wifi{
		Wificell:    true,
		AntennaConn: lab.Wifi_CONN_CONDUCTIVE,
		Router:      lab.Wifi_ROUTER_802_11AX,
	}
	lcWifi := proto.Clone(mockLabConfig).(*invV2Api.ListCrosDevicesLabConfigResponse_LabConfig)
	lcWifi.Config.Device = &lab.ChromeOSDevice_Dut{Dut: wifiDUT}
	lcWifi.Config.Id.Value = "mock_wifi_dut_id"
	wifiState := proto.Clone(mockDutState).(*lab.DutState)
	wifiState.Servo = lab.PeripheralState_NOT_CONNECTED
	lcWifi.State = wifiState
	lcWifi.State.Id.Value = "mock_wifi_dut_id"

	return &invV2Api.ListCrosDevicesLabConfigResponse{
		LabConfigs: []*invV2Api.ListCrosDevicesLabConfigResponse_LabConfig{GetMockDUT(), GetMockLabstation(), lcCamera, lcWifi},
	}, nil
}

// DeviceConfigsExists mock the device config exists request.
func (ic *InventoryClient) DeviceConfigsExists(ctx context.Context, in *invV2Api.DeviceConfigsExistsRequest, opts ...grpc.CallOption) (*invV2Api.DeviceConfigsExistsResponse, error) {
	resp := make(map[int32]bool)
	for idx, config := range in.GetConfigIds() {
		if pid := config.GetPlatformId(); pid != nil && pid.GetValue() == "test" {
			if mid := config.GetModelId(); mid != nil && mid.GetValue() == "test" {
				resp[int32(idx)] = true
			}
		} else {
			resp[int32(idx)] = false
		}
	}
	return &invV2Api.DeviceConfigsExistsResponse{
		Exists: resp,
	}, nil
}

// GetManufacturingConfig mocks the GetManufaturingConfig api from InvV2.
func (ic *InventoryClient) GetManufacturingConfig(ctx context.Context, in *invV2Api.GetManufacturingConfigRequest, opts ...grpc.CallOption) (*manufacturing.Config, error) {
	if in.GetName() == "test" || in.GetName() == "test-no-server" {
		return &manufacturing.Config{
			ManufacturingId: &manufacturing.ConfigID{Value: "test"},
		}, nil
	}
	return nil, errors.New("No manufacturing config found")
}

// GetDeviceConfig mocks the GetDeviceConfig api from InvV2.
func (ic *InventoryClient) GetDeviceConfig(ctx context.Context, in *invV2Api.GetDeviceConfigRequest, opts ...grpc.CallOption) (*device.Config, error) {
	if in.GetConfigId().GetPlatformId().GetValue() == "test" && in.GetConfigId().GetModelId().GetValue() == "test" {
		return &device.Config{
			Id: &device.ConfigId{
				PlatformId: &device.PlatformId{Value: "test"},
				ModelId:    &device.ModelId{Value: "test"},
			},
		}, nil
	}
	return nil, errors.New("No device config found")
}

// GetHwidData mocks the GetHwidData api from InvV2.
func (ic *InventoryClient) GetHwidData(ctx context.Context, in *invV2Api.GetHwidDataRequest, opts ...grpc.CallOption) (*invV2Api.HwidData, error) {
	if in.GetName() == "test" {
		return &invV2Api.HwidData{Sku: "test", Variant: "test"}, nil
	}
	return nil, errors.New("No Hwid data found")
}

// GetMockDUT returns a mock lab config with ChromeOS DUT
func GetMockDUT() *invV2Api.ListCrosDevicesLabConfigResponse_LabConfig {
	lc := proto.Clone(mockLabConfig).(*invV2Api.ListCrosDevicesLabConfigResponse_LabConfig)
	lc.Config.Device = mockDUT
	lc.Config.Id.Value = "mock_dut_id"
	lc.State = mockDutState
	lc.State.Id.Value = "mock_dut_id"
	return lc
}

// GetMockLabstation returns a mock lab config with ChromeOS Labstation
func GetMockLabstation() *invV2Api.ListCrosDevicesLabConfigResponse_LabConfig {
	lc := proto.Clone(mockLabConfig).(*invV2Api.ListCrosDevicesLabConfigResponse_LabConfig)
	lc.Config.Device = mockLabstation
	lc.Config.Id.Value = "mock_labstation_id"
	return lc
}
