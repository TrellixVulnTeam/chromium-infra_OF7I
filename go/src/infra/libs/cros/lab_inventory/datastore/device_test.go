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
			So(dsResp.Failed()[0].Device.GetId().GetValue(), ShouldEqual, "ID_FAIL")
		})
		Convey("Add device with existing ID", func() {
			devsToAdd := []*lab.ChromeOSDevice{
				mockDut("dut3", "ID_PASS", "labstation1"),
			}

			dsResp, _ := AddDevices(ctx, devsToAdd)
			So(dsResp.Passed(), ShouldHaveLength, 0)
			So(dsResp.Failed(), ShouldHaveLength, 1)

			So(dsResp.Failed()[0].Device.GetId().GetValue(), ShouldEqual, "ID_PASS")
		})
	})
}
func TestRemoveDevices(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	Convey("Remove devices from datastore", t, func() {
		Convey("Remove non-existing devices by Ids and hostnames", func() {
			resp := DeleteDevicesByIds(ctx, []string{"1234", "abcd"})
			So(resp.Passed(), ShouldHaveLength, 2)
			So(resp.Failed(), ShouldHaveLength, 0)

			resp = DeleteDevicesByHostnames(ctx, []string{"dut1", "labstation1"})
			So(resp.Passed(), ShouldHaveLength, 2)
			So(resp.Failed(), ShouldHaveLength, 0)
		})
		Convey("Happy path", func() {

			devsToAdd := []*lab.ChromeOSDevice{
				mockDut("dut1", "", "labstation1"),
				mockDut("dut2", "", "labstation1"),
				mockLabstation("labstation1", "ASSET_ID_123"),
			}
			_, err := AddDevices(ctx, devsToAdd)
			So(err, ShouldBeNil)

			datastore.GetTestable(ctx).Consistent(true)
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
