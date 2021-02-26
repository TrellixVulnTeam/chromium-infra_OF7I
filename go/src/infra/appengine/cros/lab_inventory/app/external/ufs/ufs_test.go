// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package ufs

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/proto"
	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/luci/appengine/gaetesting"
	. "go.chromium.org/luci/common/testing/assertions"

	"go.chromium.org/chromiumos/infra/proto/go/lab"
	api "infra/appengine/cros/lab_inventory/api/v1"
	"infra/appengine/cros/lab_inventory/app/config"
	"infra/appengine/cros/lab_inventory/app/external"
	"infra/appengine/cros/lab_inventory/app/frontend/fake"
	ufspb "infra/unifiedfleet/api/v1/models"
	ufschromeoslab "infra/unifiedfleet/api/v1/models/chromeos/lab"
)

type testFixture struct {
	T *testing.T
	C context.Context
}

func newTestFixtureWithContext(ctx context.Context, t *testing.T) (testFixture, func()) {
	tf := testFixture{T: t, C: ctx}
	mc := gomock.NewController(t)
	validate := func() {
		mc.Finish()
	}
	return tf, validate
}

func testingContext() context.Context {
	c := gaetesting.TestingContextWithAppID("dev~infra-lab-inventory")
	c = config.Use(c, &config.Config{
		Readers: &config.LuciAuthGroup{
			Value: "fake_group",
		},
	})
	return c
}

func TestUpdateUFSDutState(t *testing.T) {
	t.Parallel()
	Convey("UpdateUFSDutState", t, func() {
		ctx := testingContext()
		ctx = external.WithTestingContext(ctx)
		tf, validate := newTestFixtureWithContext(ctx, t)
		defer validate()
		Convey("Happy path", func() {
			req := &api.UpdateDutsStatusRequest{
				States:   []*lab.DutState{mockInvV2DutState("test-machine-dut-2")},
				DutMetas: []*api.DutMeta{mockInvV2DutMeta("test-machine-dut-2")},
				LabMetas: []*api.LabMeta{mockInvV2LabMeta("test-machine-dut-2")},
			}
			passed, failed, err := UpdateUFSDutState(tf.C, req)
			So(err, ShouldBeNil)
			So(failed, ShouldBeEmpty)
			So(passed, ShouldHaveLength, 1)
			So(passed[0].GetId(), ShouldEqual, "test-machine-dut-2")
			So(passed[0].GetHostname(), ShouldEqual, "test-dut-2")

			ufsDutState := &ufschromeoslab.DutState{
				Id: &ufschromeoslab.ChromeOSDeviceID{
					Value: "test-machine-dut-2",
				},
				Servo:                  ufschromeoslab.PeripheralState_WORKING,
				StorageState:           ufschromeoslab.HardwareState_HARDWARE_NORMAL,
				WorkingBluetoothBtpeer: 1,
				Cr50Phase:              ufschromeoslab.DutState_CR50_PHASE_PVT,
				Hostname:               "test-dut-2",
			}
			ufsDut := &ufspb.MachineLSE{
				Name:     "test-dut-2",
				Hostname: "test-dut-2",
				Machines: []string{"test-machine-dut-2"},
				Lse: &ufspb.MachineLSE_ChromeosMachineLse{
					ChromeosMachineLse: &ufspb.ChromeOSMachineLSE{
						ChromeosLse: &ufspb.ChromeOSMachineLSE_DeviceLse{
							DeviceLse: &ufspb.ChromeOSDeviceLSE{
								Device: &ufspb.ChromeOSDeviceLSE_Dut{
									Dut: &ufschromeoslab.DeviceUnderTest{
										Hostname: "test-dut-2",
										Peripherals: &ufschromeoslab.Peripherals{
											Servo: &ufschromeoslab.Servo{
												ServoType: "v3",
												ServoTopology: &ufschromeoslab.ServoTopology{
													Main: &ufschromeoslab.ServoTopologyItem{
														Type: "v3",
													},
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
			}
			ufsMachine := &ufspb.Machine{
				Name:         "test-machine-dut-2",
				SerialNumber: "test-machine-dut-2-serial",
				Device: &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: &ufspb.ChromeOSMachine{
						Sku:  "testdut2variant",
						Hwid: "testdut2hwid",
					},
				},
			}
			So(fake.MockDutStateForDUT2, ShouldResembleProto, ufsDutState)
			So(fake.MockDUT2, ShouldResembleProto, ufsDut)
			So(fake.MockMachineForDUT2, ShouldResembleProto, ufsMachine)
		})

		Convey("non-existing Machine", func() {
			req := &api.UpdateDutsStatusRequest{
				States:   []*lab.DutState{mockInvV2DutState("ghost")},
				DutMetas: []*api.DutMeta{mockInvV2DutMeta("ghost")},
				LabMetas: []*api.LabMeta{mockInvV2LabMeta("ghost")},
			}
			passed, failed, err := UpdateUFSDutState(tf.C, req)
			So(err, ShouldBeNil)
			So(passed, ShouldBeEmpty)
			So(failed, ShouldHaveLength, 1)
			So(failed[0].ErrorMsg, ShouldContainSubstring, "No MachineLSE found for for ghost")
		})
	})
}

func TestGetUFSDevicesByIds(t *testing.T) {
	t.Parallel()
	Convey("GetUFSDevicesByIds", t, func() {
		ctx := testingContext()
		ctx = external.WithTestingContext(ctx)
		ufsClient, _ := GetUFSClient(ctx)
		tf, validate := newTestFixtureWithContext(ctx, t)
		defer validate()
		Convey("Happy path - 2 passed", func() {
			devices, failedDevices := GetUFSDevicesByIds(tf.C, ufsClient, []string{"test-machine-dut", "test-machine-labstation"})
			So(failedDevices, ShouldBeEmpty)
			So(devices, ShouldHaveLength, 2)
			for _, d := range devices {
				var machine *ufspb.Machine
				if d.GetDut() != nil {
					nb, err := proto.Marshal(d.GetDut())
					So(err, ShouldBeNil)
					ob, err := proto.Marshal(fake.GetMockDUT().GetChromeosMachineLse().GetDeviceLse().GetDut())
					So(err, ShouldBeNil)
					So(nb, ShouldResemble, ob)
					machine = fake.GetMockMachineForDUT()
				} else {
					nb, err := proto.Marshal(d.GetLabstation())
					So(err, ShouldBeNil)
					ob, err := proto.Marshal(fake.GetMockLabstation().GetChromeosMachineLse().GetDeviceLse().GetLabstation())
					So(err, ShouldBeNil)
					So(nb, ShouldResemble, ob)
					machine = fake.GetMockMachineForLabstation()
				}
				So(d.GetSerialNumber(), ShouldEqual, machine.GetSerialNumber())
				So(d.GetId().GetValue(), ShouldEqual, machine.GetName())
				So(d.GetDeviceConfigId().GetPlatformId().GetValue(), ShouldEqual, machine.GetChromeosMachine().GetBuildTarget())
				So(d.GetDeviceConfigId().GetModelId().GetValue(), ShouldEqual, machine.GetChromeosMachine().GetModel())
				So(d.GetDeviceConfigId().GetVariantId().GetValue(), ShouldEqual, machine.GetChromeosMachine().GetSku())
				So(d.GetManufacturingId().GetValue(), ShouldEqual, machine.GetChromeosMachine().GetHwid())
			}
		})

		Convey("Get non existing device", func() {
			devices, failedDevices := GetUFSDevicesByIds(tf.C, ufsClient, []string{"test-machine-dut", "test-machine-labstation", "ghost"})
			So(failedDevices, ShouldHaveLength, 1)
			So(devices, ShouldHaveLength, 2)
			So(failedDevices[0].ErrorMsg, ShouldContainSubstring, "No Machine found")
		})
	})
}

func TestGetUFSDevicesByHostnames(t *testing.T) {
	t.Parallel()
	Convey("GetUFSDevicesByHostnames", t, func() {
		ctx := testingContext()
		ctx = external.WithTestingContext(ctx)
		ufsClient, _ := GetUFSClient(ctx)
		tf, validate := newTestFixtureWithContext(ctx, t)
		defer validate()
		Convey("Happy path - 2 passed", func() {
			devices, failedDevices := GetUFSDevicesByHostnames(tf.C, ufsClient, []string{"test-dut", "test-labstation"})
			So(failedDevices, ShouldBeEmpty)
			So(devices, ShouldHaveLength, 2)
			for _, d := range devices {
				var machine *ufspb.Machine
				if d.GetDut() != nil {
					nb, err := proto.Marshal(d.GetDut())
					So(err, ShouldBeNil)
					ob, err := proto.Marshal(fake.GetMockDUT().GetChromeosMachineLse().GetDeviceLse().GetDut())
					So(err, ShouldBeNil)
					So(nb, ShouldResemble, ob)
					machine = fake.GetMockMachineForDUT()
				} else {
					nb, err := proto.Marshal(d.GetLabstation())
					So(err, ShouldBeNil)
					ob, err := proto.Marshal(fake.GetMockLabstation().GetChromeosMachineLse().GetDeviceLse().GetLabstation())
					So(err, ShouldBeNil)
					So(nb, ShouldResemble, ob)
					machine = fake.GetMockMachineForLabstation()
				}
				So(d.GetSerialNumber(), ShouldEqual, machine.GetSerialNumber())
				So(d.GetId().GetValue(), ShouldEqual, machine.GetName())
				So(d.GetDeviceConfigId().GetPlatformId().GetValue(), ShouldEqual, machine.GetChromeosMachine().GetBuildTarget())
				So(d.GetDeviceConfigId().GetModelId().GetValue(), ShouldEqual, machine.GetChromeosMachine().GetModel())
				So(d.GetDeviceConfigId().GetVariantId().GetValue(), ShouldEqual, machine.GetChromeosMachine().GetSku())
				So(d.GetManufacturingId().GetValue(), ShouldEqual, machine.GetChromeosMachine().GetHwid())
			}
		})

		Convey("Get non existing device", func() {
			devices, failedDevices := GetUFSDevicesByHostnames(tf.C, ufsClient, []string{"test-dut", "test-labstation", "ghost"})
			So(failedDevices, ShouldHaveLength, 1)
			So(devices, ShouldHaveLength, 2)
			So(failedDevices[0].ErrorMsg, ShouldContainSubstring, "No MachineLSE found")
		})
	})
}

func TestGetUFSDevicesByModels(t *testing.T) {
	t.Parallel()
	Convey("GetUFSDevicesByModels", t, func() {
		ctx := testingContext()
		ctx = external.WithTestingContext(ctx)
		ufsClient, _ := GetUFSClient(ctx)
		tf, validate := newTestFixtureWithContext(ctx, t)
		defer validate()
		Convey("Happy path - 2 passed", func() {
			devices, failedDevices := GetUFSDevicesByModels(tf.C, ufsClient, []string{"testdutmodel", "testlabstationmodel"})
			So(failedDevices, ShouldBeEmpty)
			So(devices, ShouldHaveLength, 2)
			for _, d := range devices {
				var machine *ufspb.Machine
				if d.GetDut() != nil {
					nb, err := proto.Marshal(d.GetDut())
					So(err, ShouldBeNil)
					ob, err := proto.Marshal(fake.GetMockDUT().GetChromeosMachineLse().GetDeviceLse().GetDut())
					So(err, ShouldBeNil)
					So(nb, ShouldResemble, ob)
					machine = fake.GetMockMachineForDUT()
				} else {
					nb, err := proto.Marshal(d.GetLabstation())
					So(err, ShouldBeNil)
					ob, err := proto.Marshal(fake.GetMockLabstation().GetChromeosMachineLse().GetDeviceLse().GetLabstation())
					So(err, ShouldBeNil)
					So(nb, ShouldResemble, ob)
					machine = fake.GetMockMachineForLabstation()
				}
				So(d.GetSerialNumber(), ShouldEqual, machine.GetSerialNumber())
				So(d.GetId().GetValue(), ShouldEqual, machine.GetName())
				So(d.GetDeviceConfigId().GetPlatformId().GetValue(), ShouldEqual, machine.GetChromeosMachine().GetBuildTarget())
				So(d.GetDeviceConfigId().GetModelId().GetValue(), ShouldEqual, machine.GetChromeosMachine().GetModel())
				So(d.GetDeviceConfigId().GetVariantId().GetValue(), ShouldEqual, machine.GetChromeosMachine().GetSku())
				So(d.GetManufacturingId().GetValue(), ShouldEqual, machine.GetChromeosMachine().GetHwid())
			}
		})

		Convey("Get non existing device", func() {
			devices, failedDevices := GetUFSDevicesByModels(tf.C, ufsClient, []string{"testdutmodel", "testlabstationmodel", "ghost"})
			So(failedDevices, ShouldBeEmpty)
			So(devices, ShouldHaveLength, 2)
		})
	})
}

func TestGetUFSDutStateForDevices(t *testing.T) {
	t.Parallel()
	dut1 := &lab.ChromeOSDevice{
		Id: &lab.ChromeOSDeviceID{Value: "test-machine-dut"},
	}
	labstation1 := &lab.ChromeOSDevice{
		Id: &lab.ChromeOSDeviceID{Value: "test-machine-labstation"},
	}
	devIDNonExisting := &lab.ChromeOSDevice{
		Id: &lab.ChromeOSDeviceID{Value: "ghost"},
	}
	Convey("GetUFSDutStateForDevices", t, func() {
		ctx := testingContext()
		ctx = external.WithTestingContext(ctx)
		ufsClient, _ := GetUFSClient(ctx)
		tf, validate := newTestFixtureWithContext(ctx, t)
		defer validate()
		Convey("Happy path", func() {
			data, failedDevices := GetUFSDutStateForDevices(tf.C, ufsClient, []*lab.ChromeOSDevice{dut1, labstation1})
			So(failedDevices, ShouldBeEmpty)
			So(data, ShouldHaveLength, 2)
			for _, d := range data {
				if d.GetDutState().GetId().GetValue() == "test-machine-dut" {
					nb, err := proto.Marshal(d.GetDutState())
					So(err, ShouldBeNil)
					fakeDutState := fake.GetMockDutStateForDUT()
					fakeDutState.Hostname = ""
					ob, err := proto.Marshal(fakeDutState)
					So(err, ShouldBeNil)
					So(nb, ShouldResemble, ob)
				} else {
					nb, err := proto.Marshal(d.GetDutState())
					So(err, ShouldBeNil)
					fakeDutState := fake.GetMockDutStateForLabstation()
					fakeDutState.Hostname = ""
					ob, err := proto.Marshal(fakeDutState)
					So(err, ShouldBeNil)
					So(nb, ShouldResemble, ob)
				}
			}
		})

		Convey("Get non existing device", func() {
			data, failedDevices := GetUFSDutStateForDevices(tf.C, ufsClient, []*lab.ChromeOSDevice{dut1, labstation1, devIDNonExisting})
			So(failedDevices, ShouldHaveLength, 1)
			So(failedDevices[0].ErrorMsg, ShouldContainSubstring, "No DutState found")
			So(data, ShouldHaveLength, 2)
		})
	})
}

func TestCopyUFSDutToInvV2Dut(t *testing.T) {
	Convey("Verify CopyUFSDutToInvV2Dut", t, func() {
		Convey("happy path", func() {
			dut := fake.GetMockDUT()
			newDUT := CopyUFSDutToInvV2Dut(dut.GetChromeosMachineLse().GetDeviceLse().GetDut())
			nb, err := proto.Marshal(newDUT)
			So(err, ShouldBeNil)
			ob, err := proto.Marshal(dut.GetChromeosMachineLse().GetDeviceLse().GetDut())
			So(err, ShouldBeNil)
			So(nb, ShouldResemble, ob)
		})
	})
}

func TestCopyUFSLabstationToInvV2Labstation(t *testing.T) {
	Convey("Verify CopyUFSLabstationToInvV2Labstation", t, func() {
		Convey("happy path", func() {
			labstation := fake.GetMockLabstation()
			newL := CopyUFSLabstationToInvV2Labstation(labstation.GetChromeosMachineLse().GetDeviceLse().GetLabstation())
			nb, err := proto.Marshal(newL)
			So(err, ShouldBeNil)
			ob, err := proto.Marshal(labstation.GetChromeosMachineLse().GetDeviceLse().GetLabstation())
			So(err, ShouldBeNil)
			So(nb, ShouldResemble, ob)
		})
	})
}

func TestCopyUFSDutStateToInvV2DutState(t *testing.T) {
	Convey("Verify CopyUFSDutStateToInvV2DutState", t, func() {
		Convey("happy path", func() {
			dutState := fake.GetMockDutStateForDUT()
			dutState.Hostname = ""
			newL := CopyUFSDutStateToInvV2DutState(dutState)
			nb, err := proto.Marshal(newL)
			So(err, ShouldBeNil)
			ob, err := proto.Marshal(dutState)
			So(err, ShouldBeNil)
			So(nb, ShouldResemble, ob)
		})
	})
}

func TestCopyInvV2DutStateToUFSDutState(t *testing.T) {
	Convey("Verify CopyInvV2DutStateToUFSDutState", t, func() {
		Convey("happy path", func() {
			invV2dutState := mockInvV2DutState("test-machine")
			ufsDutState := CopyInvV2DutStateToUFSDutState(invV2dutState, "test-hostname")
			ufsDutState.Hostname = ""
			ufsb, err := proto.Marshal(ufsDutState)
			So(err, ShouldBeNil)
			invV2b, err := proto.Marshal(invV2dutState)
			So(err, ShouldBeNil)
			So(ufsb, ShouldResemble, invV2b)
		})
	})
}

func TestCopyInvV2DutMetaToUFSDutMeta(t *testing.T) {
	Convey("Verify CopyInvV2DutMetaToUFSDutMeta", t, func() {
		Convey("happy path", func() {
			invV2dutMeta := mockInvV2DutMeta("test-machine")
			ufsDutMeta := CopyInvV2DutMetaToUFSDutMeta(invV2dutMeta, "test-hostname")
			So(ufsDutMeta.Hostname, ShouldEqual, "test-hostname")
			So(ufsDutMeta.GetChromeosDeviceId(), ShouldEqual, invV2dutMeta.GetChromeosDeviceId())
			So(ufsDutMeta.GetSerialNumber(), ShouldEqual, invV2dutMeta.GetSerialNumber())
			So(ufsDutMeta.GetHwID(), ShouldEqual, invV2dutMeta.GetHwID())
			So(ufsDutMeta.GetDeviceSku(), ShouldEqual, invV2dutMeta.GetDeviceSku())
		})
	})
}

func TestCopyInvV2LabMetaToUFSLabMeta(t *testing.T) {
	Convey("Verify CopyInvV2LabMetaToUFSLabMeta", t, func() {
		Convey("happy path", func() {
			invV2LabMeta := mockInvV2LabMeta("test-machine")
			ufsLabMeta := CopyInvV2LabMetaToUFSLabMeta(invV2LabMeta, "test-hostname")
			So(ufsLabMeta.Hostname, ShouldEqual, "test-hostname")
			So(ufsLabMeta.GetChromeosDeviceId(), ShouldEqual, invV2LabMeta.GetChromeosDeviceId())
			So(ufsLabMeta.GetServoType(), ShouldEqual, invV2LabMeta.GetServoType())
			So(ufsLabMeta.GetSmartUsbhub(), ShouldEqual, invV2LabMeta.GetSmartUsbhub())
			So(ufsLabMeta.GetServoTopology().GetMain().GetType(), ShouldEqual, invV2LabMeta.GetServoTopology().GetMain().GetType())
		})
	})
}

func mockInvV2DutState(id string) *lab.DutState {
	return &lab.DutState{
		Id: &lab.ChromeOSDeviceID{
			Value: id,
		},
		Servo:                  lab.PeripheralState_WORKING,
		StorageState:           lab.HardwareState_HARDWARE_NORMAL,
		WorkingBluetoothBtpeer: 1,
		Cr50Phase:              lab.DutState_CR50_PHASE_PVT,
	}
}

func mockInvV2DutMeta(id string) *api.DutMeta {
	return &api.DutMeta{
		ChromeosDeviceId: id,
		SerialNumber:     "test-machine-dut-2-serial",
		HwID:             "testdut2hwid",
		DeviceSku:        "testdut2variant",
	}
}

func mockInvV2LabMeta(id string) *api.LabMeta {
	return &api.LabMeta{
		ChromeosDeviceId: id,
		SmartUsbhub:      true,
		ServoType:        "v3",
		ServoTopology: &lab.ServoTopology{
			Main: &lab.ServoTopologyItem{
				Type: "v3",
			},
		},
	}
}
