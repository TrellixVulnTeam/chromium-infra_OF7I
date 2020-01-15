// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package datastore

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/chromiumos/infra/proto/go/lab"
	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/appengine/gaetesting"
)

func mockServo(servoHost string) *lab.Servo {
	return &lab.Servo{
		ServoHostname: servoHost,
		ServoPort:     8888,
		ServoSerial:   "SERVO1",
		ServoType:     "v3",
	}
}
func mockDut(hostname, id, servoHost string) *lab.ChromeOSDevice {
	return &lab.ChromeOSDevice{
		Id: &lab.ChromeOSDeviceID{
			Value: id,
		},
		Device: &lab.ChromeOSDevice_Dut{
			Dut: &lab.DeviceUnderTest{
				Hostname: hostname,
				Peripherals: &lab.Peripherals{
					Servo: mockServo(servoHost),
				},
			},
		},
	}
}

func mockLabstation(hostname, id string) *lab.ChromeOSDevice {
	return &lab.ChromeOSDevice{
		Id: &lab.ChromeOSDeviceID{
			Value: id,
		},
		Device: &lab.ChromeOSDevice_Labstation{
			Labstation: &lab.Labstation{
				Hostname: hostname,
				Servos:   []*lab.Servo{mockServo(hostname)},
			},
		},
	}
}

func TestAddDevices(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	Convey("Add device to datastore", t, func() {
		Convey("Add 2 new devices", func() {
			devsToAdd := []*lab.ChromeOSDevice{
				mockDut("dut1", "", "labstation1"),
				mockLabstation("labstation1", "ASSET_ID_123"),
			}
			dsResp, err := AddDevices(ctx, devsToAdd)
			if err != nil {
				t.Fatal(err)
			}
			So(dsResp.Passed(), ShouldHaveLength, len(devsToAdd))
			So(dsResp.Failed(), ShouldHaveLength, 0)
			want := []string{
				devsToAdd[0].GetDut().GetHostname(),
				devsToAdd[1].GetLabstation().GetHostname(),
			}
			got := make([]string, len(devsToAdd))
			for i, result := range dsResp.Passed() {
				d := DeviceEntity{ID: result.Entity.ID, Parent: fakeAcestorKey(ctx)}
				if err := datastore.Get(ctx, &d); err != nil {
					t.Errorf("cannot get device by id %s: %v", result.Entity.ID, err)
				}
				got[i] = d.Hostname
			}
			So(got, ShouldResemble, want)
			So(devsToAdd[0].GetId().GetValue(), ShouldNotEqual, "")
			So(devsToAdd[1].GetId().GetValue(), ShouldEqual, "ASSET_ID_123")

		})
		Convey("Add device with existing hostname", func() {
			devsToAdd := []*lab.ChromeOSDevice{
				mockDut("dut1", "ID_FAIL", "labstation1"),
				mockDut("dut2", "ID_PASS", "labstation1"),
			}

			dsResp, _ := AddDevices(ctx, devsToAdd)
			So(dsResp.Passed(), ShouldHaveLength, 1)
			So(dsResp.Failed(), ShouldHaveLength, 1)

			So(dsResp.Passed()[0].Entity.ID, ShouldEqual, "ID_PASS")
			So(dsResp.Failed()[0].Data.(*lab.ChromeOSDevice).GetId().GetValue(), ShouldEqual, "ID_FAIL")
		})
		Convey("Add device with existing ID", func() {
			devsToAdd := []*lab.ChromeOSDevice{
				mockDut("dut3", "ID_PASS", "labstation1"),
			}

			dsResp, _ := AddDevices(ctx, devsToAdd)
			So(dsResp.Passed(), ShouldHaveLength, 0)
			So(dsResp.Failed(), ShouldHaveLength, 1)

			So(dsResp.Failed()[0].Data.(*lab.ChromeOSDevice).GetId().GetValue(), ShouldEqual, "ID_PASS")
		})
	})
}
func TestRemoveDevices(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	Convey("Remove devices from datastore", t, func() {
		devsToAdd := []*lab.ChromeOSDevice{
			mockDut("dut1", "", "labstation1"),
			mockDut("dut2", "UUID:02", "labstation1"),
			mockLabstation("labstation1", "ASSET_ID_123"),
		}
		_, err := AddDevices(ctx, devsToAdd)
		So(err, ShouldBeNil)

		datastore.GetTestable(ctx).Consistent(true)

		Convey("Remove non-existing devices by Ids and hostnames", func() {
			resp := DeleteDevicesByIds(ctx, []string{"1234", "abcd"})
			So(resp.Passed(), ShouldHaveLength, 2)
			So(resp.Failed(), ShouldHaveLength, 0)

			resp = DeleteDevicesByHostnames(ctx, []string{"dutX", "labstationX"})
			So(resp.Passed(), ShouldHaveLength, 2)
			So(resp.Failed(), ShouldHaveLength, 0)

			// There are still 3 device entities unchanged.
			var devs []*DeviceEntity
			err = datastore.GetAll(ctx, datastore.NewQuery(DeviceKind), &devs)
			So(err, ShouldBeNil)
			So(devs, ShouldHaveLength, 3)
		})

		Convey("Happy path", func() {
			var devs []*DeviceEntity
			err = datastore.GetAll(ctx, datastore.NewQuery(DeviceKind), &devs)
			So(err, ShouldBeNil)
			So(devs, ShouldHaveLength, 3)

			resp := DeleteDevicesByIds(ctx, []string{"ASSET_ID_123"})
			So(resp.Passed(), ShouldHaveLength, 1)
			So(resp.Failed(), ShouldHaveLength, 0)
			devs = nil
			err = datastore.GetAll(ctx, datastore.NewQuery(DeviceKind), &devs)
			So(err, ShouldBeNil)
			So(devs, ShouldHaveLength, 2)

			resp = DeleteDevicesByHostnames(ctx, []string{"dut1", "dut2"})
			So(resp.Passed(), ShouldHaveLength, 2)
			So(resp.Failed(), ShouldHaveLength, 0)
			devs = nil
			err = datastore.GetAll(ctx, datastore.NewQuery(DeviceKind), &devs)
			So(err, ShouldBeNil)
			So(devs, ShouldHaveLength, 0)
		})
	})
}

func TestGetDevices(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	Convey("Get devices from an empty datastore", t, func() {
		Convey("Get all", func() {
			devs, err := GetAllDevices(ctx)
			So(devs, ShouldBeEmpty)
			So(err, ShouldBeNil)
		})
		Convey("Get by Ids", func() {
			result := GetDevicesByIds(ctx, []string{"1234", "5678"})
			So(result.Passed(), ShouldBeEmpty)
			So(result.Failed(), ShouldHaveLength, 2)
		})
		Convey("Get by hostnames", func() {
			result := GetDevicesByHostnames(ctx, []string{"dut1", "labstation2"})
			So(result.Passed(), ShouldBeEmpty)
			So(result.Failed(), ShouldHaveLength, 2)
		})
	})
	Convey("Get devices from a non-empty datastore", t, func() {
		devsToAdd := []*lab.ChromeOSDevice{
			mockDut("dut1", "", "labstation1"),
			mockLabstation("labstation1", "ASSET_ID_123"),
		}
		_, err := AddDevices(ctx, devsToAdd)
		if err != nil {
			t.Fatal(err)
		}
		Convey("Get all", func() {
			devs, err := GetAllDevices(ctx)
			So(devs, ShouldHaveLength, 2)
			So(err, ShouldBeNil)
		})
		Convey("Get by Ids", func() {
			result := GetDevicesByIds(ctx, []string{"ASSET_ID_123", "5678"})
			So(result.Passed(), ShouldHaveLength, 1)
			So(result.Passed()[0].Entity.Hostname, ShouldEqual, "labstation1")
			So(result.Failed(), ShouldHaveLength, 1)
		})
		Convey("Get by hostnames", func() {
			result := GetDevicesByHostnames(ctx, []string{"dut1", "labstationX"})
			So(result.Passed(), ShouldHaveLength, 1)
			So(result.Passed()[0].Entity.Hostname, ShouldEqual, "dut1")
			So(result.Failed(), ShouldHaveLength, 1)
		})
	})
}

func TestUpdateDeviceSetup(t *testing.T) {
	t.Parallel()

	Convey("Update devices setup in datastore", t, func() {
		ctx := gaetesting.TestingContextWithAppID("go-test")

		devsToAdd := []*lab.ChromeOSDevice{
			mockDut("dut1", "UUID:01", "labstation1"),
			mockLabstation("labstation1", "UUID:02"),
		}
		_, err := AddDevices(ctx, devsToAdd)
		So(err, ShouldBeNil)

		datastore.GetTestable(ctx).Consistent(true)
		Convey("Update non-existing devices", func() {
			result, err := UpdateDeviceSetup(ctx, []*lab.ChromeOSDevice{
				mockDut("dut1", "UUID:ghost", ""),
				mockDut("dut1", "UUID:01", "labstation2"),
			})
			if err != nil {
				t.Fatal(err)
			}
			passed := result.Passed()
			So(passed, ShouldHaveLength, 1)
			So(passed[0].Entity.ID, ShouldEqual, "UUID:01")

			failed := result.Failed()
			So(failed, ShouldHaveLength, 1)
			So(failed[0].Entity.ID, ShouldEqual, "UUID:ghost")
		})

	})
}

func TestUpdateDutMeta(t *testing.T) {
	t.Parallel()

	Convey("Update devices setup in datastore", t, func() {
		ctx := gaetesting.TestingContextWithAppID("go-test")

		devsToAdd := []*lab.ChromeOSDevice{
			mockDut("dut1", "UUID:01", "labstation1"),
		}
		_, err := AddDevices(ctx, devsToAdd)
		So(err, ShouldBeNil)

		datastore.GetTestable(ctx).Consistent(true)
		Convey("Update meta", func() {
			meta := map[string]DutMeta{
				"UUID:01": {
					SerialNumber: "serial2",
					HwID:         "hwid2",
				},
				"UUID:ghost": {},
			}
			result, err := UpdateDutMeta(ctx, meta)
			if err != nil {
				t.Fatal(err)
			}
			passed := result.Passed()
			So(passed, ShouldHaveLength, 1)
			So(passed[0].Entity.ID, ShouldEqual, "UUID:01")
			var p lab.ChromeOSDevice
			passed[0].Entity.GetCrosDeviceProto(&p)
			So(p.GetSerialNumber(), ShouldEqual, "serial2")
			So(p.GetManufacturingId().GetValue(), ShouldEqual, "hwid2")

			failed := result.Failed()
			So(failed, ShouldHaveLength, 1)
			So(failed[0].Entity.ID, ShouldEqual, "UUID:ghost")
		})
	})
}

func TestUpdateDutsStatus(t *testing.T) {
	t.Parallel()

	Convey("Update dut status in datastore", t, func() {
		ctx := gaetesting.TestingContextWithAppID("go-test")

		devsToAdd := []*lab.ChromeOSDevice{
			mockDut("dut1", "UUID:01", "labstation1"),
			mockLabstation("labstation1", "UUID:02"),
		}
		_, err := AddDevices(ctx, devsToAdd)
		So(err, ShouldBeNil)

		datastore.GetTestable(ctx).Consistent(true)
		Convey("Update status of non-existing dut", func() {
			state := lab.DutState{Id: &lab.ChromeOSDeviceID{Value: "ghost"}}
			result, err := UpdateDutsStatus(ctx, []*lab.DutState{
				&state,
				{Id: &lab.ChromeOSDeviceID{Value: "UUID:01"}},
			})
			if err != nil {
				t.Fatal(err)
			}
			passed := result.Passed()
			So(passed, ShouldHaveLength, 1)
			So(passed[0].Entity.ID, ShouldEqual, "UUID:01")

			failed := result.Failed()
			So(failed, ShouldHaveLength, 1)
			So(failed[0].Entity.ID, ShouldEqual, "ghost")
		})
		Convey("Update dut state", func() {
			state := lab.DutState{
				Id:    &lab.ChromeOSDeviceID{Value: "UUID:01"},
				Servo: lab.PeripheralState_NOT_CONNECTED,
			}
			result, err := UpdateDutsStatus(ctx, []*lab.DutState{&state})
			if err != nil {
				t.Fatal(err)
			}
			passed := result.Passed()
			So(passed, ShouldHaveLength, 1)
			So(passed[0].Entity.ID, ShouldEqual, "UUID:01")

			failed := result.Failed()
			So(failed, ShouldBeEmpty)

			// Read from datastore and verify the result.
			s := DeviceEntity{ID: "UUID:01", Parent: fakeAcestorKey(ctx)}
			if err := datastore.Get(ctx, &s); err != nil {
				t.Errorf("cannot get dut state by id %v: %v", s, err)
			}
			var p lab.DutState
			if err := s.GetDutStateProto(&p); err != nil {
				t.Errorf("Cannot get proto message of dut state %v: %v", s, err)
			}
			So(p.GetServo(), ShouldEqual, lab.PeripheralState_NOT_CONNECTED)
		})

		Convey("Cannot update status of a labstation", func() {
			state := lab.DutState{Id: &lab.ChromeOSDeviceID{Value: "UUID:02"}}
			result, err := UpdateDutsStatus(ctx, []*lab.DutState{&state})
			if err != nil {
				t.Fatal(err)
			}
			passed := result.Passed()
			So(passed, ShouldBeEmpty)

			failed := result.Failed()
			So(failed, ShouldHaveLength, 1)
			So(failed[0].Entity.ID, ShouldEqual, "UUID:02")
		})
	})
}
