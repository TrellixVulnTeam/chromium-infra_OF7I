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

	"go.chromium.org/chromiumos/infra/proto/go/lab"
	"infra/appengine/cros/lab_inventory/app/config"
	"infra/appengine/cros/lab_inventory/app/external"
	"infra/appengine/cros/lab_inventory/app/frontend/fake"
	ufspb "infra/unifiedfleet/api/v1/models"
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
