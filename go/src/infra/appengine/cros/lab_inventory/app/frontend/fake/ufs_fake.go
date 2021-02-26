// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package fake

import (
	"context"

	"github.com/golang/protobuf/proto"
	"go.chromium.org/luci/common/errors"
	"google.golang.org/grpc"

	ufspb "infra/unifiedfleet/api/v1/models"
	lab "infra/unifiedfleet/api/v1/models/chromeos/lab"
	ufsapi "infra/unifiedfleet/api/v1/rpc"
	ufsutil "infra/unifiedfleet/app/util"
)

// FleetClient mocks the UFS client
type FleetClient struct {
}

var mockDUT = &ufspb.MachineLSE{
	Name:     "test-dut",
	Hostname: "test-dut",
	Machines: []string{"test-machine-dut"},
	Lse: &ufspb.MachineLSE_ChromeosMachineLse{
		ChromeosMachineLse: &ufspb.ChromeOSMachineLSE{
			ChromeosLse: &ufspb.ChromeOSMachineLSE_DeviceLse{
				DeviceLse: &ufspb.ChromeOSDeviceLSE{
					Device: &ufspb.ChromeOSDeviceLSE_Dut{
						Dut: &lab.DeviceUnderTest{
							Hostname: "test-dut",
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
										lab.ChameleonType_CHAMELEON_TYPE_BT_BLE_HID,
										lab.ChameleonType_CHAMELEON_TYPE_BT_PEER,
									},
									AudioBoard: true,
								},
								Rpm: &lab.OSRPM{
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
					},
				},
			},
		},
	},
	Zone:          "ZONE_CHROMEOS6",
	ResourceState: ufspb.State_STATE_REGISTERED,
}

var mockLabstation = &ufspb.MachineLSE{
	Name:     "test-labstation",
	Hostname: "test-labstation",
	Machines: []string{"test-machine-labstation"},
	Lse: &ufspb.MachineLSE_ChromeosMachineLse{
		ChromeosMachineLse: &ufspb.ChromeOSMachineLSE{
			ChromeosLse: &ufspb.ChromeOSMachineLSE_DeviceLse{
				DeviceLse: &ufspb.ChromeOSDeviceLSE{
					Device: &ufspb.ChromeOSDeviceLSE_Labstation{
						Labstation: &lab.Labstation{
							Hostname: "test-labstation",
							Servos: []*lab.Servo{{
								ServoHostname: "test-labstation",
								ServoPort:     int32(9999),
								ServoSerial:   "test_servo_serial",
								ServoType:     "v3",
							}},
							Rpm: &lab.OSRPM{
								PowerunitName:   "test_power_unit_name",
								PowerunitOutlet: "test_power_unit_outlet3",
							},
							Pools: []string{"labstation_main"},
						},
					},
				},
			},
		},
	},
}

var mockDutStateForDUT = &lab.DutState{
	Id: &lab.ChromeOSDeviceID{
		Value: "test-machine-dut",
	},
	Servo:                  lab.PeripheralState_WORKING,
	StorageState:           lab.HardwareState_HARDWARE_NORMAL,
	WorkingBluetoothBtpeer: 1,
	Cr50Phase:              lab.DutState_CR50_PHASE_PVT,
	Hostname:               "test-dut",
}

// MockDUT2 for testing UpdateDutState
var MockDUT2 = &ufspb.MachineLSE{
	Name:     "test-dut-2",
	Hostname: "test-dut-2",
	Machines: []string{"test-machine-dut-2"},
	Lse: &ufspb.MachineLSE_ChromeosMachineLse{
		ChromeosMachineLse: &ufspb.ChromeOSMachineLSE{
			ChromeosLse: &ufspb.ChromeOSMachineLSE_DeviceLse{
				DeviceLse: &ufspb.ChromeOSDeviceLSE{
					Device: &ufspb.ChromeOSDeviceLSE_Dut{
						Dut: &lab.DeviceUnderTest{
							Hostname: "test-dut-2",
							Peripherals: &lab.Peripherals{
								Servo: &lab.Servo{
									ServoType: "invalid",
									ServoTopology: &lab.ServoTopology{
										Main: &lab.ServoTopologyItem{
											Type: "invalid",
										},
									},
								},
								SmartUsbhub: false,
							},
						},
					},
				},
			},
		},
	},
}

// MockMachineForDUT2 for testing UpdateDutState
var MockMachineForDUT2 = &ufspb.Machine{
	Name:         "test-machine-dut-2",
	SerialNumber: "invalid",
	Device: &ufspb.Machine_ChromeosMachine{
		ChromeosMachine: &ufspb.ChromeOSMachine{
			Sku:  "invalid",
			Hwid: "invalid",
		},
	},
}

// MockDutStateForDUT2 for testing UpdateDutState
var MockDutStateForDUT2 = &lab.DutState{
	Id: &lab.ChromeOSDeviceID{
		Value: "test-machine-dut-2",
	},
	Servo:                  lab.PeripheralState_UNKNOWN,
	StorageState:           lab.HardwareState_HARDWARE_UNKNOWN,
	WorkingBluetoothBtpeer: 0,
	Cr50Phase:              lab.DutState_CR50_PHASE_INVALID,
	Hostname:               "test-dut-2",
}

var mockDutStateForLabstation = &lab.DutState{
	Id: &lab.ChromeOSDeviceID{
		Value: "test-machine-labstation",
	},
	Servo:                  lab.PeripheralState_WORKING,
	StorageState:           lab.HardwareState_HARDWARE_NORMAL,
	WorkingBluetoothBtpeer: 1,
	Cr50Phase:              lab.DutState_CR50_PHASE_PVT,
	Hostname:               "test-labstation",
}

var mockMachineForDUT = &ufspb.Machine{
	Name:         "test-machine-dut",
	SerialNumber: "test-machine-dut-serial",
	Device: &ufspb.Machine_ChromeosMachine{
		ChromeosMachine: &ufspb.ChromeOSMachine{
			Model:       "testdutmodel",
			BuildTarget: "testdutplatform",
			Sku:         "testdutvariant",
		},
	},
}

var mockMachineForLabStation = &ufspb.Machine{
	Name:         "test-machine-labstation",
	SerialNumber: "test-machine-labstation-serial",
	Device: &ufspb.Machine_ChromeosMachine{
		ChromeosMachine: &ufspb.ChromeOSMachine{
			Model:       "testlabstationmodel",
			BuildTarget: "testlabstationplatform",
			Sku:         "testlabstationvairant",
		},
	},
}

// UpdateDutState mocks the UpdateDutState api from UFS
func (ic *FleetClient) UpdateDutState(ctx context.Context, in *ufsapi.UpdateDutStateRequest, opts ...grpc.CallOption) (*lab.DutState, error) {
	if in.GetDutMeta().GetChromeosDeviceId() == "test-machine-dut-2" || in.GetDutMeta().GetHostname() == "test-dut-2" {
		MockDutStateForDUT2 = in.GetDutState()

		MockMachineForDUT2.SerialNumber = in.GetDutMeta().GetSerialNumber()
		MockMachineForDUT2.GetChromeosMachine().Hwid = in.GetDutMeta().GetHwID()
		MockMachineForDUT2.GetChromeosMachine().Sku = in.GetDutMeta().GetDeviceSku()

		MockDUT2.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().SmartUsbhub = in.GetLabMeta().GetSmartUsbhub()
		MockDUT2.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetServo().ServoType = in.GetLabMeta().GetServoType()
		MockDUT2.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetServo().ServoTopology = in.GetLabMeta().GetServoTopology()

		return MockDutStateForDUT2, nil
	}
	return nil, errors.New("No Machine/MachineLSE found")
}

// GetMachineLSE mocks the GetMachineLSE api from UFS.
func (ic *FleetClient) GetMachineLSE(ctx context.Context, in *ufsapi.GetMachineLSERequest, opts ...grpc.CallOption) (*ufspb.MachineLSE, error) {
	if in.GetName() == ufsutil.AddPrefix(ufsutil.MachineLSECollection, "test-dut") {
		mockDUTCopy := proto.Clone(mockDUT).(*ufspb.MachineLSE)
		mockDUTCopy.Name = ufsutil.AddPrefix(ufsutil.MachineLSECollection, mockDUTCopy.Name)
		return mockDUTCopy, nil
	}
	if in.GetName() == ufsutil.AddPrefix(ufsutil.MachineLSECollection, "test-dut-2") {
		mockDUT2Copy := proto.Clone(MockDUT2).(*ufspb.MachineLSE)
		mockDUT2Copy.Name = ufsutil.AddPrefix(ufsutil.MachineLSECollection, mockDUT2Copy.Name)
		return mockDUT2Copy, nil
	}
	if in.GetName() == ufsutil.AddPrefix(ufsutil.MachineLSECollection, "test-labstation") {
		mockLabstationCopy := proto.Clone(mockLabstation).(*ufspb.MachineLSE)
		mockLabstationCopy.Name = ufsutil.AddPrefix(ufsutil.MachineLSECollection, mockLabstationCopy.Name)
		return mockLabstationCopy, nil
	}
	return nil, errors.New("No MachineLSE found")
}

// GetMachine mocks the GetMachine api from UFS.
func (ic *FleetClient) GetMachine(ctx context.Context, in *ufsapi.GetMachineRequest, opts ...grpc.CallOption) (*ufspb.Machine, error) {
	if in.GetName() == ufsutil.AddPrefix(ufsutil.MachineCollection, "test-machine-dut") {
		mockMachineForDUTCopy := proto.Clone(mockMachineForDUT).(*ufspb.Machine)
		mockMachineForDUTCopy.Name = ufsutil.AddPrefix(ufsutil.MachineCollection, mockMachineForDUTCopy.Name)
		return mockMachineForDUTCopy, nil
	}
	if in.GetName() == ufsutil.AddPrefix(ufsutil.MachineCollection, "test-machine-dut-2") {
		mockMachineForDUT2Copy := proto.Clone(MockMachineForDUT2).(*ufspb.Machine)
		mockMachineForDUT2Copy.Name = ufsutil.AddPrefix(ufsutil.MachineCollection, mockMachineForDUT2Copy.Name)
		return mockMachineForDUT2Copy, nil
	}
	if in.GetName() == ufsutil.AddPrefix(ufsutil.MachineCollection, "test-machine-labstation") {
		mockMachineForLabStationCopy := proto.Clone(mockMachineForLabStation).(*ufspb.Machine)
		mockMachineForLabStationCopy.Name = ufsutil.AddPrefix(ufsutil.MachineCollection, mockMachineForLabStationCopy.Name)
		return mockMachineForLabStationCopy, nil
	}
	return nil, errors.New("No Machine found")
}

// GetDutState mocks the GetDutState api from UFS.
func (ic *FleetClient) GetDutState(ctx context.Context, in *ufsapi.GetDutStateRequest, opts ...grpc.CallOption) (*lab.DutState, error) {
	if in.GetChromeosDeviceId() == "test-machine-dut" || in.GetHostname() == "test-dut" {
		return mockDutStateForDUT, nil
	}
	if in.GetChromeosDeviceId() == "test-machine-dut-2" || in.GetHostname() == "test-dut-2" {
		return MockDutStateForDUT2, nil
	}
	if in.GetChromeosDeviceId() == "test-machine-labstation" || in.GetHostname() == "test-labstation" {
		return mockDutStateForLabstation, nil
	}
	return nil, errors.New("No DutState found")
}

// ListMachines mocks the ListMachines api from UFS.
func (ic *FleetClient) ListMachines(ctx context.Context, in *ufsapi.ListMachinesRequest, opts ...grpc.CallOption) (*ufsapi.ListMachinesResponse, error) {
	mockMachineForDUTCopy := proto.Clone(mockMachineForDUT).(*ufspb.Machine)
	mockMachineForDUTCopy.Name = ufsutil.AddPrefix(ufsutil.MachineCollection, mockMachineForDUTCopy.Name)
	mockMachineForLabStationCopy := proto.Clone(mockMachineForLabStation).(*ufspb.Machine)
	mockMachineForLabStationCopy.Name = ufsutil.AddPrefix(ufsutil.MachineCollection, mockMachineForLabStationCopy.Name)
	if in.GetFilter() == "" {
		return &ufsapi.ListMachinesResponse{
			Machines:      []*ufspb.Machine{mockMachineForDUTCopy, mockMachineForLabStationCopy},
			NextPageToken: "",
		}, nil
	}
	if in.GetFilter() == "model=testdutmodel" {
		return &ufsapi.ListMachinesResponse{
			Machines:      []*ufspb.Machine{mockMachineForDUTCopy},
			NextPageToken: "",
		}, nil
	}
	if in.GetFilter() == "model=testlabstationmodel" {
		return &ufsapi.ListMachinesResponse{
			Machines:      []*ufspb.Machine{mockMachineForLabStationCopy},
			NextPageToken: "",
		}, nil
	}
	return &ufsapi.ListMachinesResponse{
		Machines:      nil,
		NextPageToken: "",
	}, nil
}

// ListMachineLSEs mocks the ListMachineLSEs api from UFS.
func (ic *FleetClient) ListMachineLSEs(ctx context.Context, in *ufsapi.ListMachineLSEsRequest, opts ...grpc.CallOption) (*ufsapi.ListMachineLSEsResponse, error) {
	mockDUTCopy := proto.Clone(mockDUT).(*ufspb.MachineLSE)
	mockDUTCopy.Name = ufsutil.AddPrefix(ufsutil.MachineLSECollection, mockDUTCopy.Name)
	mockDUT2Copy := proto.Clone(MockDUT2).(*ufspb.MachineLSE)
	mockDUT2Copy.Name = ufsutil.AddPrefix(ufsutil.MachineLSECollection, mockDUT2Copy.Name)
	mockLabstationCopy := proto.Clone(mockLabstation).(*ufspb.MachineLSE)
	mockLabstationCopy.Name = ufsutil.AddPrefix(ufsutil.MachineLSECollection, mockLabstationCopy.Name)
	if in.GetFilter() == "" {
		return &ufsapi.ListMachineLSEsResponse{
			MachineLSEs:   []*ufspb.MachineLSE{mockDUTCopy, mockLabstationCopy},
			NextPageToken: "",
		}, nil
	}
	if in.GetFilter() == "machine=test-machine-dut" {
		return &ufsapi.ListMachineLSEsResponse{
			MachineLSEs:   []*ufspb.MachineLSE{mockDUTCopy},
			NextPageToken: "",
		}, nil
	}
	if in.GetFilter() == "machine=test-machine-dut-2" {
		return &ufsapi.ListMachineLSEsResponse{
			MachineLSEs:   []*ufspb.MachineLSE{mockDUT2Copy},
			NextPageToken: "",
		}, nil
	}
	if in.GetFilter() == "machine=test-machine-labstation" {
		return &ufsapi.ListMachineLSEsResponse{
			MachineLSEs:   []*ufspb.MachineLSE{mockLabstationCopy},
			NextPageToken: "",
		}, nil
	}
	return &ufsapi.ListMachineLSEsResponse{
		MachineLSEs:   nil,
		NextPageToken: "",
	}, nil
}

// ListDutStates mocks the ListDutStates api from UFS.
func (ic *FleetClient) ListDutStates(ctx context.Context, in *ufsapi.ListDutStatesRequest, opts ...grpc.CallOption) (*ufsapi.ListDutStatesResponse, error) {
	return &ufsapi.ListDutStatesResponse{
		DutStates:     []*lab.DutState{mockDutStateForDUT, mockDutStateForLabstation},
		NextPageToken: "",
	}, nil
}

// GetMockDUT mocks dut machinelse
func GetMockDUT() *ufspb.MachineLSE {
	return mockDUT
}

// GetMockLabstation mocks labstation machinelse
func GetMockLabstation() *ufspb.MachineLSE {
	return mockLabstation
}

// GetMockMachineForDUT mocks machine for dut
func GetMockMachineForDUT() *ufspb.Machine {
	return mockMachineForDUT
}

// GetMockMachineForLabstation mocks machine for labstation
func GetMockMachineForLabstation() *ufspb.Machine {
	return mockMachineForLabStation
}

// GetMockDutStateForDUT mocks DutState for dut
func GetMockDutStateForDUT() *lab.DutState {
	return mockDutStateForDUT
}

// GetMockDutStateForLabstation mocks DutState for labstation
func GetMockDutStateForLabstation() *lab.DutState {
	return mockDutStateForLabstation
}
