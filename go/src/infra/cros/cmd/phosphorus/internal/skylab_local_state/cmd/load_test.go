// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmd

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"infra/libs/skylab/inventory"
	ufspb "infra/unifiedfleet/api/v1/models"
	ufsapi "infra/unifiedfleet/api/v1/rpc"

	. "github.com/smartystreets/goconvey/convey"
	labapi "go.chromium.org/chromiumos/config/go/test/lab/api"
	"go.chromium.org/chromiumos/infra/proto/go/lab_platform"
	. "go.chromium.org/luci/common/testing/assertions"
	"google.golang.org/protobuf/types/known/timestamppb"

	"go.chromium.org/chromiumos/infra/proto/go/test_platform/skylab_local_state"
)

func TestConvertAttachedDeviceDutTopologyToHostInfoForAndroid(t *testing.T) {
	Convey("When attached device DUT topology is converted to host info the result is correct.", t, func() {
		associatedHostname := "dummy_associated_hostname"
		board := "dummy_board"
		hostname := "dummy_hostname"
		model := "dummy_model"
		serialNumber := "1234567890"
		input := labapi.DutTopology{
			Id: &labapi.DutTopology_Id{
				Value: "dummy_dut_topology_id",
			},
			Duts: []*labapi.Dut{
				{
					Id: &labapi.Dut_Id{
						Value: "dummy_dut_id",
					},
					DutType: &labapi.Dut_Android_{
						Android: &labapi.Dut_Android{
							AssociatedHostname: &labapi.IpEndpoint{
								Address: associatedHostname,
							},
							Name:         hostname,
							SerialNumber: serialNumber,
							DutModel: &labapi.DutModel{
								BuildTarget: board,
								ModelName:   model,
							},
						},
					},
				},
			},
		}

		got, err := convertDutTopologyToHostInfo(&input)

		So(got, ShouldNotBeNil)
		So(err, ShouldBeNil)

		want := &skylab_local_state.AutotestHostInfo{
			Attributes: map[string]string{},
			Labels: []string{
				"associated_hostname:" + associatedHostname,
				"board:" + board,
				"model:" + model,
				"name:" + hostname,
				"serial_number:" + serialNumber,
				"os:android",
			},
			SerializerVersion: 1,
		}

		sort.Strings(got.Labels)
		sort.Strings(want.Labels)

		So(want, ShouldResembleProto, got)
	})
}

func TestConvertAttachedDeviceDutTopologyToHostInfoForChromeOS(t *testing.T) {
	Convey("When attached device DUT topology is converted to host info the result is correct.", t, func() {
		board := "dummy_board"
		model := "dummy_model"
		servo_address := "dummy_servo_ip_address"
		servo_port := 12345
		servo_serial := "ASDF12345"
		input := labapi.DutTopology{
			Id: &labapi.DutTopology_Id{
				Value: "dummy_dut_topology_id",
			},
			Duts: []*labapi.Dut{
				{
					Id: &labapi.Dut_Id{
						Value: "dummy_dut_id",
					},
					DutType: &labapi.Dut_Chromeos{
						Chromeos: &labapi.Dut_ChromeOS{
							DutModel: &labapi.DutModel{
								BuildTarget: board,
								ModelName:   model,
							},
							Servo: &labapi.Servo{
								ServodAddress: &labapi.IpEndpoint{
									Address: servo_address,
									Port:    int32(servo_port),
								},
								Serial: servo_serial,
							},
							Chameleon: &labapi.Chameleon{
								Peripherals: []labapi.Chameleon_Peripheral{
									labapi.Chameleon_PREIPHERAL_UNSPECIFIED,
								},
								AudioBoard: true,
							},
							Audio: &labapi.Audio{
								Atrus: true,
							},
							Wifi: &labapi.Wifi{
								Environment: labapi.Wifi_ENVIRONMENT_UNSPECIFIED,
							},
							Touch: &labapi.Touch{
								Mimo: true,
							},
							Camerabox: &labapi.Camerabox{
								Facing: labapi.Camerabox_FACING_UNSPECIFIED,
							},
						},
					},
				},
			},
		}

		got, err := convertDutTopologyToHostInfo(&input)

		So(got, ShouldNotBeNil)
		So(err, ShouldBeNil)

		want := &skylab_local_state.AutotestHostInfo{
			Attributes: map[string]string{
				"servo_host":   servo_address,
				"servo_port":   fmt.Sprintf("%v", servo_port),
				"servo_serial": servo_serial,
			},
			Labels: []string{
				"board:" + board,
				"model:" + model,
				"chameleon",
				"chameleon:" + strings.ToLower(labapi.Chameleon_Peripheral_name[int32(labapi.Chameleon_PREIPHERAL_UNSPECIFIED)]),
				"audio_board",
				"atrus",
				"mimo",
				"camerabox_facing:" + strings.ToLower(labapi.Camerabox_Facing_name[int32(labapi.Camerabox_FACING_UNSPECIFIED)]),
			},
			SerializerVersion: 1,
		}

		sort.Strings(got.Labels)
		sort.Strings(want.Labels)

		So(want, ShouldResembleProto, got)
	})
}

func TestConvertDutTopologyWithMultipleDutsToHostInfo(t *testing.T) {
	Convey("When DUT topology contains multiple DUTs, conversion to host info fails.", t, func() {
		input := labapi.DutTopology{
			Id: &labapi.DutTopology_Id{
				Value: "dummy_dut_topology_id",
			},
			Duts: []*labapi.Dut{
				{
					Id: &labapi.Dut_Id{
						Value: "dummy_dut_id_1",
					},
				},
				{
					Id: &labapi.Dut_Id{
						Value: "dummy_dut_id_2",
					},
				},
			},
		}

		got, err := convertDutTopologyToHostInfo(&input)

		So(got, ShouldBeNil)
		So(err, ShouldNotBeNil)
	})
}

func TestConvertChromeOsDeviceInfoToHostInfo(t *testing.T) {
	Convey("When DUT device info is converted to host info the result is correct.", t, func() {
		board := "dummy_board"
		sku := "dummy_sku"
		osType := inventory.SchedulableLabels_OS_TYPE_CROS
		dut := inventory.DeviceUnderTest{
			Common: &inventory.CommonDeviceSpecs{
				Attributes: []*inventory.KeyValue{
					keyValue("sku", "dummy_sku"),
					keyValue("dummy_key", "dummy_value"),
				},
				Labels: &inventory.SchedulableLabels{
					Board:    &board,
					OsType:   &osType,
					Platform: &board,
					Sku:      &sku,
				},
			},
		}
		input := ufsapi.GetDeviceDataResponse{
			ResourceType: ufsapi.GetDeviceDataResponse_RESOURCE_TYPE_CHROMEOS_DEVICE,
			Resource: &ufsapi.GetDeviceDataResponse_ChromeOsDeviceData{
				ChromeOsDeviceData: &ufspb.ChromeOSDeviceData{
					DutV1: &dut,
				},
			},
		}

		got := hostInfoFromDeviceInfo(&input)

		So(got, ShouldNotBeNil)

		want := &skylab_local_state.AutotestHostInfo{
			Attributes: map[string]string{
				"dummy_key": "dummy_value",
				"sku":       "dummy_sku",
			},
			Labels: []string{
				"board:dummy_board",
				"conductive:False",
				"device-sku:dummy_sku",
				"os:cros",
				"platform:dummy_board",
			},
			SerializerVersion: 1,
		}

		sort.Strings(got.Labels)
		sort.Strings(want.Labels)

		So(want, ShouldResembleProto, got)
	})
}

func TestConvertAttachedDeviceInfoToHostInfo(t *testing.T) {
	Convey("When attached device info is converted to host info the result is correct.", t, func() {
		associatedHostname := "dummy_associated_hostname"
		board := "dummy_board"
		hostname := "dummy_hostname"
		model := "dummy_model"
		serialNumber := "1234567890"
		attachedDevice := ufsapi.AttachedDeviceData{
			LabConfig: &ufspb.MachineLSE{
				Name:                "dummy_name",
				MachineLsePrototype: "dummy_machine_lse_prototype",
				Hostname:            hostname,
				Lse: &ufspb.MachineLSE_AttachedDeviceLse{
					AttachedDeviceLse: &ufspb.AttachedDeviceLSE{
						OsVersion: &ufspb.OSVersion{
							Value:       "dummy_value",
							Description: "dummy_description",
							Image:       "dummy_image",
						},
						AssociatedHostname: associatedHostname,
					},
				},
				UpdateTime: &timestamppb.Timestamp{
					Seconds: 0,
					Nanos:   0,
				},
				Schedulable: true,
			},
			Machine: &ufspb.Machine{
				SerialNumber: serialNumber,
				Device: &ufspb.Machine_AttachedDevice{
					AttachedDevice: &ufspb.AttachedDevice{
						Model:        model,
						BuildTarget:  board,
						Manufacturer: "dummy_manufacturer",
						DeviceType:   ufspb.AttachedDeviceType_ATTACHED_DEVICE_TYPE_ANDROID_PHONE,
					},
				},
			},
		}
		input := ufsapi.GetDeviceDataResponse{
			ResourceType: ufsapi.GetDeviceDataResponse_RESOURCE_TYPE_ATTACHED_DEVICE,
			Resource: &ufsapi.GetDeviceDataResponse_AttachedDeviceData{
				AttachedDeviceData: &attachedDevice,
			},
		}

		got := hostInfoFromDeviceInfo(&input)

		So(got, ShouldNotBeNil)

		want := &skylab_local_state.AutotestHostInfo{
			Attributes: map[string]string{},
			Labels: []string{
				"associated_hostname:" + associatedHostname,
				"board:" + board,
				"model:" + model,
				"name:" + hostname,
				"serial_number:" + serialNumber,
			},
			SerializerVersion: 1,
		}

		sort.Strings(got.Labels)
		sort.Strings(want.Labels)

		So(want, ShouldResembleProto, got)
	})
}

func TestAddBotStateToHostInfo(t *testing.T) {
	Convey("When host info is updated from bot info the resulting labels and attributes are correct.", t, func() {
		hostInfo := &skylab_local_state.AutotestHostInfo{
			Attributes: map[string]string{
				"attribute1": "value1",
			},
			Labels: []string{
				"label2:value2",
			},
			SerializerVersion: 1,
		}

		s := &lab_platform.DutState{
			ProvisionableAttributes: map[string]string{
				"attribute3": "value3",
			},
			ProvisionableLabels: map[string]string{
				"label4": "value4",
			},
		}

		addDeviceStateToHostInfo(hostInfo, s)

		// There's no guarantee on the order.
		sort.Strings(hostInfo.Labels)

		want := &skylab_local_state.AutotestHostInfo{
			Attributes: map[string]string{
				"attribute1": "value1",
				"attribute3": "value3",
			},
			Labels: []string{
				"label2:value2",
				"label4:value4",
			},
			SerializerVersion: 1,
		}

		So(want, ShouldResembleProto, hostInfo)
	})
}

func keyValue(key string, value string) *inventory.KeyValue {
	return &inventory.KeyValue{
		Key:   &key,
		Value: &value,
	}
}
