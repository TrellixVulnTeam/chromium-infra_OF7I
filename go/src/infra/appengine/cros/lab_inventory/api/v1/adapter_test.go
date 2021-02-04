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
	. "go.chromium.org/luci/common/testing/assertions"

	"infra/libs/skylab/inventory"
)

var servoInV2 = lab.Servo{
	ServoHostname: "test_servo",
	ServoPort:     int32(9999),
	ServoSerial:   "test_servo_serial",
	ServoType:     "v3",
	ServoSetup:    lab.ServoSetupType_SERVO_SETUP_DUAL_V4,
	ServoTopology: &lab.ServoTopology{
		Main: &lab.ServoTopologyItem{
			Type:         "servo_v4",
			SysfsProduct: "Servo V4",
			Serial:       "C1903145591",
			UsbHubPort:   "6.4.1",
		},
		Children: []*lab.ServoTopologyItem{
			{
				Type:         "ccd_cr50",
				SysfsProduct: "Cr50",
				Serial:       "0681D03A-92DCCD64",
				UsbHubPort:   "6.4.2",
			},
			{
				Type:         "c2d2",
				SysfsProduct: "C2D2",
				Serial:       "0681D03A-YYYYYYYY",
				UsbHubPort:   "6.4.3",
			},
		},
	},
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
					ChameleonPeripherals: []lab.ChameleonType{
						lab.ChameleonType_CHAMELEON_TYPE_BT_BLE_HID,
						lab.ChameleonType_CHAMELEON_TYPE_BT_PEER,
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
					AudioBox:   true,
					AudioCable: true,
					Atrus:      true,
				},
				Wifi: &lab.Wifi{
					Wificell:    true,
					AntennaConn: lab.Wifi_CONN_CONDUCTIVE,
					Router:      lab.Wifi_ROUTER_802_11AX,
				},
				Touch: &lab.Touch{
					Mimo: true,
				},
				Carrier:   "att",
				Camerabox: true,
				CameraboxInfo: &lab.Camerabox{
					Facing: lab.Camerabox_FACING_BACK,
					Light:  lab.Camerabox_LIGHT_LED,
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
			Licenses: []*lab.License{
				{
					Type:       lab.LicenseType_LICENSE_TYPE_WINDOWS_10_PRO,
					Identifier: "my-windows-identifier-A001",
				},
				{
					Type:       lab.LicenseType_LICENSE_TYPE_MS_OFFICE_STANDARD,
					Identifier: "my-office-identifier-B002",
				},
			},
		},
	},
}

var servoLabstationInV2 = lab.ChromeOSDevice{
	Id:              &lab.ChromeOSDeviceID{Value: "test_labstation2"},
	SerialNumber:    "labstation2_serial",
	ManufacturingId: &manufacturing.ConfigID{Value: "labstation2_hwid"},
	DeviceConfigId: &device.ConfigId{
		PlatformId: &device.PlatformId{Value: "guado"},
		ModelId:    &device.ModelId{Value: "test_model"},
		VariantId:  &device.VariantId{Value: ""},
	},
	Device: &lab.ChromeOSDevice_Labstation{
		Labstation: &lab.Labstation{
			Hostname: "test_servo",
			Servos:   []*lab.Servo{&servoInV2},
			Rpm: &lab.RPM{
				PowerunitName:   "test_power_unit_name",
				PowerunitOutlet: "test_power_unit_outlet3",
			},
			Pools: []string{"labstation_main"},
		},
	},
}

var devInV2State = lab.DutState{
	Id: &lab.ChromeOSDeviceID{
		Value: "test_dut",
	},
	Servo:                  lab.PeripheralState_BROKEN,
	Chameleon:              lab.PeripheralState_WORKING,
	AudioLoopbackDongle:    lab.PeripheralState_NOT_CONNECTED,
	WorkingBluetoothBtpeer: 3,
	Cr50Phase:              lab.DutState_CR50_PHASE_PVT,
	Cr50KeyEnv:             lab.DutState_CR50_KEYENV_PROD,
	StorageState:           lab.HardwareState_HARDWARE_NORMAL,
	ServoUsbState:          lab.HardwareState_HARDWARE_NEED_REPLACEMENT,
	RpmState:               lab.PeripheralState_WORKING,
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
	DutState:  &devInV2State,
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
	attributes {
		key: "servo_setup",
		value: "DUAL_V4",
	}
	hostname: "test_host"
	id: "test_dut"
	hwid: "test_hwid"
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
		ec_type: EC_TYPE_CHROME_OS
		hwid_sku: "test_sku"
		licenses: {
			type: 1
			identifier: "my-windows-identifier-A001"
		}
		licenses: {
			type: 2
			identifier: "my-office-identifier-B002"
		}
		model: "test_model"
		os_type: OS_TYPE_CROS
		sku: "test_variant"
		peripherals {
			audio_board: true
			audio_box: true
			audio_cable: true
			audio_loopback_dongle: false
			chameleon: true
			chameleon_type: CHAMELEON_TYPE_BT_BLE_HID
			chameleon_type: CHAMELEON_TYPE_BT_PEER
			conductive: true
			huddly: true
			mimo: true
			ptzpro2: true
			camerabox: true
			camerabox_facing: CAMERABOX_FACING_BACK
			camerabox_light: CAMERABOX_LIGHT_LED
			servo: true
			servo_topology: {
				main: {
					usb_hub_port: "6.4.1"
					serial: "C1903145591"
					type: "servo_v4"
					sysfs_product: "Servo V4"
				}
				children: {
					usb_hub_port: "6.4.2"
					serial: "0681D03A-92DCCD64"
					type: "ccd_cr50"
					sysfs_product: "Cr50"
				}
				children: {
					usb_hub_port: "6.4.3"
					serial: "0681D03A-YYYYYYYY"
					type: "c2d2"
					sysfs_product: "C2D2"
				}
			  }
			servo_state: BROKEN
			servo_type: "v3"
			rpm_state: WORKING
			smart_usbhub: true
			storage_state: HARDWARE_NORMAL,
			servo_usb_state: HARDWARE_NEED_REPLACEMENT,
			wificell: true
			router_802_11ax: true
			working_bluetooth_btpeer: 3
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
		wifi_chip: ""
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
const labstationProtoFromV2 = `
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
	id: "test_labstation"
	hostname: "test_labstation_host"
	serial_number: "labstation_serial"
	labels {
		arc: false
		os_type: OS_TYPE_LABSTATION
		self_serve_pools: "labstation_main"
		model: "test_model"
		board: "guado"
		brand: ""
		sku: ""
        capabilities {
          atrus: false
          bluetooth: false
          carrier: CARRIER_INVALID
          detachablebase: false
          fingerprint: false
          flashrom: false
          gpu_family: ""
          graphics: ""
          hotwording: false
          internal_display: false
          lucidsleep: false
          modem: ""
          power: "AC_only"
          storage: ""
          telephony: ""
          webcam: false
          touchpad: false
          touchscreen: false
        }
        cr50_phase: CR50_PHASE_INVALID
        cr50_ro_keyid: ""
        cr50_ro_version: ""
        cr50_rw_keyid: ""
        cr50_rw_version: ""
        ec_type: EC_TYPE_INVALID
        hwid_sku: ""
		peripherals {
          audio_board: false
          audio_box: false
          audio_loopback_dongle: false
          chameleon: false
          chameleon_type: CHAMELEON_TYPE_INVALID
          conductive: false
          huddly: false
          mimo: false
          servo: false
          servo_state: UNKNOWN
          smart_usbhub: false
          stylus: false
          camerabox: false
          wificell: false
          router_802_11ax: false
		}
		platform:""
        test_coverage_hints {
            chaos_dut: false
            chaos_nightly: false
            chromesign: false
            hangout_app: false
            meet_app: false
            recovery_test: false
            test_audiojack: false
            test_hdmiaudio: false
            test_usbaudio: false
            test_usbprinting: false
            usb_detect: false
            use_lid: false
        }
        wifi_chip: ""
	}
}
`

// The servo host associated with test_dut.
const labstation2TextProto = `
common {
	attributes {
		key: "HWID",
		value: "labstation2_hwid",
	}
	attributes {
		key: "powerunit_hostname",
		value: "test_power_unit_name",
	}
	attributes {
		key: "powerunit_outlet",
		value: "test_power_unit_outlet3",
	}
	attributes {
		key: "serial_number"
		value: "labstation2_serial"
	}
	id: "test_labstation2"
	hostname: "test_servo"
	serial_number: "labstation2_serial"
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
		dataCopy := proto.Clone(&data).(*ExtendedDeviceData)

		Convey("empty input", func() {
			_, err := AdaptToV1DutSpec(&ExtendedDeviceData{})
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "nil ext data to adapt")
		})
		Convey("empty hwid data", func() {
			dataCopy.HwidData = nil
			d, err := AdaptToV1DutSpec(dataCopy)
			So(err, ShouldBeNil)
			So(d.GetCommon().GetHostname(), ShouldEqual, "test_host")
		})
		Convey("empty device config", func() {
			dataCopy.DeviceConfig = nil
			d, err := AdaptToV1DutSpec(dataCopy)
			So(err, ShouldBeNil)
			So(d.GetCommon().GetHostname(), ShouldEqual, "test_host")
		})
		Convey("empty manufacturing config", func() {
			dataCopy.ManufacturingConfig = nil
			d, err := AdaptToV1DutSpec(dataCopy)
			So(err, ShouldBeNil)
			So(d.GetCommon().GetHostname(), ShouldEqual, "test_host")
		})
		Convey("may os_type", func() {
			board := "fizz-moblab"
			osType := inventory.SchedulableLabels_OS_TYPE_MOBLAB
			d := proto.Clone(&d1).(*inventory.DeviceUnderTest)
			d.GetCommon().GetLabels().Board = &board
			d.GetCommon().GetLabels().Platform = &board
			d.GetCommon().GetLabels().OsType = &osType
			d.GetCommon().GetLabels().Arc = &falseValue
			s1, err := inventory.WriteLabToString(&inventory.Lab{
				Duts: []*inventory.DeviceUnderTest{d},
			})
			So(err, ShouldBeNil)

			dataCopy.LabConfig = proto.Clone(data.LabConfig).(*lab.ChromeOSDevice)
			dataCopy.LabConfig.GetDeviceConfigId().GetPlatformId().Value = board
			d2, err := AdaptToV1DutSpec(dataCopy)
			So(err, ShouldBeNil)
			s2, err := inventory.WriteLabToString(&inventory.Lab{
				Duts: []*inventory.DeviceUnderTest{d2},
			})
			So(s1, ShouldEqual, s2)
		})
		Convey("servo_state is UNKNOWN/false by default", func() {
			dataCopy.DutState = &lab.DutState{}
			d, err := AdaptToV1DutSpec(dataCopy)
			So(err, ShouldBeNil)
			So(*d.GetCommon().GetLabels().GetPeripherals().ServoState, ShouldEqual, inventory.PeripheralState_UNKNOWN)
			So(*d.GetCommon().GetLabels().GetPeripherals().Servo, ShouldBeFalse)
		})
		Convey("servo_state is broken", func() {
			dataCopy.DutState = &lab.DutState{}
			dataCopy.DutState.Servo = lab.PeripheralState_BROKEN
			d, err := AdaptToV1DutSpec(dataCopy)
			So(err, ShouldBeNil)
			So(*d.GetCommon().GetLabels().GetPeripherals().ServoState,
				ShouldEqual,
				inventory.PeripheralState_BROKEN)
			So(*d.GetCommon().GetLabels().GetPeripherals().Servo, ShouldEqual, true)
		})
		Convey("servo_state is wrong_config", func() {
			dataCopy.DutState = &lab.DutState{}
			dataCopy.DutState.Servo = lab.PeripheralState_WRONG_CONFIG
			d, err := AdaptToV1DutSpec(dataCopy)
			So(err, ShouldBeNil)
			So(*d.GetCommon().GetLabels().GetPeripherals().ServoState,
				ShouldEqual,
				inventory.PeripheralState_WRONG_CONFIG)
			So(*d.GetCommon().GetLabels().GetPeripherals().Servo, ShouldEqual, true)
		})
		Convey("servo_state is working", func() {
			dataCopy.DutState = &lab.DutState{}
			dataCopy.DutState.Servo = lab.PeripheralState_WORKING
			d, err := AdaptToV1DutSpec(dataCopy)
			So(err, ShouldBeNil)
			So(*d.GetCommon().GetLabels().GetPeripherals().ServoState,
				ShouldEqual,
				inventory.PeripheralState_WORKING)
			So(*d.GetCommon().GetLabels().GetPeripherals().Servo, ShouldEqual, true)
		})
		Convey("servo_state is not_connected", func() {
			dataCopy.DutState = &lab.DutState{}
			dataCopy.DutState.Servo = lab.PeripheralState_NOT_CONNECTED
			d, err := AdaptToV1DutSpec(dataCopy)
			So(err, ShouldBeNil)
			So(*d.GetCommon().GetLabels().GetPeripherals().ServoState,
				ShouldEqual,
				inventory.PeripheralState_NOT_CONNECTED)
			So(*d.GetCommon().GetLabels().GetPeripherals().Servo, ShouldEqual, false)
		})
		Convey("happy path", func() {
			d, err := AdaptToV1DutSpec(&data)
			So(err, ShouldBeNil)
			s, err := inventory.WriteLabToString(&inventory.Lab{
				Duts: []*inventory.DeviceUnderTest{d},
			})
			So(err, ShouldBeNil)
			So(proto.Equal(&d1, d), ShouldBeTrue)
			So(s1, ShouldEqual, s)
		})
	})

	Convey("Verify labstation v2 => v1", t, func() {
		var labstation inventory.DeviceUnderTest
		err := proto.UnmarshalText(labstationProtoFromV2, &labstation)
		So(err, ShouldBeNil)

		extLabstaion := ExtendedDeviceData{
			LabConfig: &labstationInV2,
		}
		d, err := AdaptToV1DutSpec(&extLabstaion)
		So(err, ShouldBeNil)

		s, err := inventory.WriteLabToString(&inventory.Lab{
			Duts: []*inventory.DeviceUnderTest{d},
		})
		So(err, ShouldBeNil)
		strLabstation, err := inventory.WriteLabToString(&inventory.Lab{
			Duts: []*inventory.DeviceUnderTest{&labstation},
		})
		So(err, ShouldBeNil)
		So(s, ShouldEqual, strLabstation)
	})
}

func TestImportFromV1DutSpecs(t *testing.T) {
	t.Parallel()

	Convey("Verify V1 => V2", t, func() {
		var d1 inventory.DeviceUnderTest
		err := proto.UnmarshalText(dutTextProto, &d1)
		// Set servo state for testing dutState creation.
		inventoryServoStateBroken := inventory.PeripheralState_BROKEN
		d1.GetCommon().GetLabels().GetPeripherals().ServoState = &inventoryServoStateBroken
		So(err, ShouldBeNil)
		var l1 inventory.DeviceUnderTest
		err = proto.UnmarshalText(labstationTextProto, &l1)
		So(err, ShouldBeNil)

		var l2 inventory.DeviceUnderTest
		err = proto.UnmarshalText(labstation2TextProto, &l2)
		So(err, ShouldBeNil)

		devices, labstations, states, err := ImportFromV1DutSpecs([]*inventory.CommonDeviceSpecs{
			d1.GetCommon(),
			l1.GetCommon(),
			l2.GetCommon(),
		})
		// Verify devices
		So(len(devices), ShouldEqual, 1)
		So(devices[0], ShouldResembleProto, &devInV2)

		// Verify labstations
		So(len(labstations), ShouldEqual, 2)
		So(proto.Equal(getDeviceByHostname(labstations, "test_servo"), &servoLabstationInV2), ShouldBeTrue)
		So(proto.Equal(getDeviceByHostname(labstations, "test_labstation_host"), &labstationInV2), ShouldBeTrue)

		// Verify dut states.
		So(len(states), ShouldEqual, 1)
		So(proto.Equal(states[0], &devInV2State), ShouldBeTrue)

		Convey("set servo from servo_state", func() {
			*d1.GetCommon().GetLabels().GetPeripherals().ServoState = inventory.PeripheralState_WORKING
			d1.GetCommon().GetLabels().GetPeripherals().Servo = &falseValue
			_, _, states, err = ImportFromV1DutSpecs([]*inventory.CommonDeviceSpecs{
				d1.GetCommon(),
			})
			So(err, ShouldBeNil)
			So(states[0].GetServo(), ShouldEqual, lab.PeripheralState_WORKING)
		})
		Convey("set servo from servo when servo_state is UNKNOWN and servo is true", func() {
			*d1.GetCommon().GetLabels().GetPeripherals().ServoState = inventory.PeripheralState_UNKNOWN
			d1.GetCommon().GetLabels().GetPeripherals().Servo = &trueValue
			_, _, states, err = ImportFromV1DutSpecs([]*inventory.CommonDeviceSpecs{
				d1.GetCommon(),
			})
			So(err, ShouldBeNil)
			So(states[0].GetServo(), ShouldEqual, lab.PeripheralState_WORKING)
		})
		Convey("set servo from servo when servo_state is nil  and servo is true", func() {
			d1.GetCommon().GetLabels().GetPeripherals().ServoState = nil
			d1.GetCommon().GetLabels().GetPeripherals().Servo = &trueValue
			_, _, states, err = ImportFromV1DutSpecs([]*inventory.CommonDeviceSpecs{
				d1.GetCommon(),
			})
			So(err, ShouldBeNil)
			So(states[0].GetServo(), ShouldEqual, lab.PeripheralState_WORKING)
		})
		Convey("set servo from servo when servo_state is nil  and servo is false", func() {
			d1.GetCommon().GetLabels().GetPeripherals().ServoState = nil
			d1.GetCommon().GetLabels().GetPeripherals().Servo = &falseValue
			_, _, states, err = ImportFromV1DutSpecs([]*inventory.CommonDeviceSpecs{
				d1.GetCommon(),
			})
			So(err, ShouldBeNil)
			So(states[0].GetServo(), ShouldEqual, lab.PeripheralState_NOT_CONNECTED)
		})
		Convey("set servo from servo_state 2", func() {
			*d1.GetCommon().GetLabels().GetPeripherals().ServoState = inventory.PeripheralState_BROKEN
			d1.GetCommon().GetLabels().GetPeripherals().Servo = &falseValue
			_, _, states, err = ImportFromV1DutSpecs([]*inventory.CommonDeviceSpecs{
				d1.GetCommon(),
			})
			So(err, ShouldBeNil)
			So(states[0].GetServo(), ShouldEqual, lab.PeripheralState_BROKEN)
		})
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
