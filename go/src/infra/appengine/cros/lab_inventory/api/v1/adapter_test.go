// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file

package api

import (
	"testing"

	"github.com/golang/protobuf/proto"
	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/chromiumos/infra/proto/go/device"
	"go.chromium.org/chromiumos/infra/proto/go/lab"
	"go.chromium.org/chromiumos/infra/proto/go/manufacturing"
	"infra/libs/skylab/inventory"
)

var servoInV2 = lab.Servo{
	ServoHostname: "test_servo",
	ServoPort:     int32(9999),
	ServoSerial:   "test_servo_serial",
	ServoType:     "v3",
}

var devInV2 = lab.ChromeOSDevice{
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
	Device: &lab.ChromeOSDevice_Dut{
		Dut: &lab.DeviceUnderTest{
			Hostname: "test_host",
			Pools:    []string{"DUT_POOL_QUOTA", "hotrod"},
			Peripherals: &lab.Peripherals{
				Servo: &servoInV2,
				Chameleon: &lab.Chameleon{
					ChameleonPeripherals: []lab.ChameleonType{lab.ChameleonType_CHAMELEON_TYPE_BT_BLE_HID, lab.ChameleonType_CHAMELEON_TYPE_BT_PEER},
					AudioBoard:           true,
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
				Wifi: &lab.Wifi{
					Wificell:    true,
					AntennaConn: lab.Wifi_CONN_CONDUCTIVE,
				},
				Touch: &lab.Touch{
					Mimo: true,
				},
				Carrier:   "att",
				Camerabox: true,
				Chaos:     true,
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
			},
		},
	},
}

var servoLabstationInV2 = lab.ChromeOSDevice{
	Device: &lab.ChromeOSDevice_Labstation{
		Labstation: &lab.Labstation{
			Hostname: "test_servo",
			Servos:   []*lab.Servo{&servoInV2},
		},
	},
}

var devInV2State = lab.DutState{
	Id: &lab.ChromeOSDeviceID{
		Value: "test_dut",
	},
	Servo:               lab.PeripheralState_WORKING,
	Chameleon:           lab.PeripheralState_WORKING,
	AudioLoopbackDongle: lab.PeripheralState_NOT_CONNECTED,
}

var labstationInV2 = lab.ChromeOSDevice{
	Id: &lab.ChromeOSDeviceID{
		Value: "test_labstation",
	},
	SerialNumber: "labstation_serial",
	ManufacturingId: &manufacturing.ConfigID{
		Value: "labstation_hwid",
	},
	DeviceConfigId: &device.ConfigId{
		PlatformId: &device.PlatformId{
			Value: "guado",
		},
		ModelId: &device.ModelId{
			Value: "test_model",
		},
		VariantId: &device.VariantId{
			Value: "",
		},
	},
	Device: &lab.ChromeOSDevice_Labstation{
		Labstation: &lab.Labstation{
			Hostname: "test_labstation_host",
			Pools:    []string{"labstation_main"},
			Servos:   []*lab.Servo{},
			Rpm: &lab.RPM{
				PowerunitName:   "test_power_unit_name",
				PowerunitOutlet: "test_power_unit_outlet2",
			},
		},
	},
}

var data = ExtendedDeviceData{
	LabConfig: &devInV2,
	DeviceConfig: &device.Config{
		Id: &device.ConfigId{
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
		FormFactor: device.Config_FORM_FACTOR_CHROMEBASE,
		GpuFamily:  "test_gpu",
		Graphics:   device.Config_GRAPHICS_GLE,
		HardwareFeatures: []device.Config_HardwareFeature{
			device.Config_HARDWARE_FEATURE_BLUETOOTH,
			device.Config_HARDWARE_FEATURE_DETACHABLE_KEYBOARD,
			device.Config_HARDWARE_FEATURE_FINGERPRINT,
			device.Config_HARDWARE_FEATURE_TOUCHSCREEN,
		},
		Power:   device.Config_POWER_SUPPLY_AC_ONLY,
		Storage: device.Config_STORAGE_SSD,
		VideoAccelerationSupports: []device.Config_VideoAcceleration{
			device.Config_VIDEO_ACCELERATION_ENC_H264,
			device.Config_VIDEO_ACCELERATION_ENC_VP8,
			device.Config_VIDEO_ACCELERATION_ENC_VP9,
		},
		Cpu: device.Config_ARM64,
	},
	HwidData: &HwidData{
		Sku:     "test_sku",
		Variant: "test_variant",
	},
	ManufacturingConfig: &manufacturing.Config{
		ManufacturingId: &manufacturing.ConfigID{
			Value: "test_hwid",
		},
		DevicePhase: manufacturing.Config_PHASE_DVT,
		Cr50Phase:   manufacturing.Config_CR50_PHASE_PVT,
		Cr50KeyEnv:  manufacturing.Config_CR50_KEYENV_PROD,
	},
}

const dutTextProto = `
common {
	attributes {
		key: "HWID",
		value: "test_hwid",
	}
	attributes {
		key: "powerunit_hostname",
		value: "test_power_unit_name",
	}
	attributes {
		key: "powerunit_outlet",
		value: "test_power_unit_outlet",
	}
	attributes {
		key: "serial_number"
		value: "test_serial"
	}
	attributes {
		key: "servo_host"
		value: "test_servo"
	}
	attributes {
		key: "servo_port"
		value: "9999"
	}
	attributes {
		key: "servo_serial",
		value: "test_servo_serial",
	}
	attributes {
		key: "servo_type",
		value: "v3",
	}
	hostname: "test_host"
	id: "test_dut"
	serial_number: "test_serial"
	labels {
		arc: true
		board: "coral"
		brand: ""
		capabilities {
			atrus: true
			bluetooth: true
			carrier: CARRIER_ATT
			detachablebase: true
			fingerprint: true
			gpu_family: "test_gpu"
			graphics: "gles"
			power: "AC_only"
			storage: "ssd"
			touchscreen: true
			video_acceleration: VIDEO_ACCELERATION_ENC_H264
			video_acceleration: VIDEO_ACCELERATION_ENC_VP8
			video_acceleration: VIDEO_ACCELERATION_ENC_VP9
		}
		cr50_phase: CR50_PHASE_PVT
		cts_abi: CTS_ABI_ARM
		cts_cpu: CTS_CPU_ARM
		cr50_ro_keyid: "prod"
		hwid_sku: "test_sku"
		model: "test_model"
		os_type: OS_TYPE_CROS
		sku: "test_variant"
		peripherals {
			audio_board: true
			audio_box: true
			chameleon: true
			chameleon_type: CHAMELEON_TYPE_BT_BLE_HID
			chameleon_type: CHAMELEON_TYPE_BT_PEER
			conductive: true
			huddly: true
			mimo: true
			ptzpro2: true
			camerabox: true
			wificell: true
		}
		phase: PHASE_DVT
		platform: "coral"
		test_coverage_hints {
			chaos_dut: true
			hangout_app: true
			meet_app: true
			test_audiojack: true
			test_hdmiaudio: true
			test_usbaudio: true
			test_usbprinting: true
		}
		variant: "test_variant"
		critical_pools: DUT_POOL_QUOTA
		self_serve_pools: "hotrod"
	}
} 
`

const labstationTextProto = `
common {
	attributes {
		key: "HWID",
		value: "labstation_hwid",
	}
	attributes {
		key: "powerunit_hostname",
		value: "test_power_unit_name",
	}
	attributes {
		key: "powerunit_outlet",
		value: "test_power_unit_outlet2",
	}
	attributes {
		key: "serial_number"
		value: "labstation_serial"
	}
	attributes {
		key: "servo_host"
		value: "servo2"
	}
	attributes {
		key: "servo_port"
		value: "9999"
	}
	attributes {
		key: "servo_serial",
		value: "serial2",
	}
	attributes {
		key: "servo_type",
		value: "v4",
	}
	id: "test_labstation"
	hostname: "test_labstation_host"
	serial_number: "labstation_serial"
	labels {
		os_type: OS_TYPE_LABSTATION
		self_serve_pools: "labstation_main"
		model: "test_model"
		board: "guado"
	}
}
`

func TestAdaptToV1DutSpec(t *testing.T) {
	t.Parallel()

	Convey("Verify V2 => V1", t, func() {
		var d1 inventory.DeviceUnderTest
		err := proto.UnmarshalText(dutTextProto, &d1)
		So(err, ShouldBeNil)
		s1, err := inventory.WriteLabToString(&inventory.Lab{
			Duts: []*inventory.DeviceUnderTest{&d1},
		})
		So(err, ShouldBeNil)

		d, err := AdaptToV1DutSpec(&data)
		So(err, ShouldBeNil)
		s, err := inventory.WriteLabToString(&inventory.Lab{
			Duts: []*inventory.DeviceUnderTest{d},
		})
		So(err, ShouldBeNil)
		So(s1, ShouldEqual, s)
		So(proto.Equal(&d1, d), ShouldBeTrue)
	})
}

func TestImportFromV1DutSpecs(t *testing.T) {
	t.Parallel()

	Convey("Verify V1 => V2", t, func() {
		var d1 inventory.DeviceUnderTest
		err := proto.UnmarshalText(dutTextProto, &d1)
		// Set servo state for testing dutState creation.
		d1.GetCommon().GetLabels().GetPeripherals().Servo = &trueValue
		So(err, ShouldBeNil)
		var l1 inventory.DeviceUnderTest
		err = proto.UnmarshalText(labstationTextProto, &l1)
		So(err, ShouldBeNil)

		devices, labstations, states, err := ImportFromV1DutSpecs([]*inventory.CommonDeviceSpecs{d1.GetCommon(), l1.GetCommon()})
		// Verify devices
		So(len(devices), ShouldEqual, 1)
		So(proto.Equal(devices[0], &devInV2), ShouldBeTrue)

		// Verify labstations
		So(len(labstations), ShouldEqual, 2)
		So(proto.Equal(getDeviceByHostname(labstations, "test_servo"), &servoLabstationInV2), ShouldBeTrue)
		So(proto.Equal(getDeviceByHostname(labstations, "test_labstation_host"), &labstationInV2), ShouldBeTrue)

		// Verify dut states.
		So(len(states), ShouldEqual, 1)
		So(proto.Equal(states[0], &devInV2State), ShouldBeTrue)
	})
}

func getDeviceByHostname(devices []*lab.ChromeOSDevice, hostname string) *lab.ChromeOSDevice {
	for _, d := range devices {
		if d.GetDut() != nil && d.GetDut().GetHostname() == hostname {
			return d
		}
		if d.GetLabstation() != nil && d.GetLabstation().GetHostname() == hostname {
			return d
		}
	}
	return nil
}
