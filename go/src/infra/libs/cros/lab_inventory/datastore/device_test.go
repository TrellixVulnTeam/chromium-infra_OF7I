// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package datastore

import (
	"context"
	"testing"

	"github.com/golang/protobuf/proto"
	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/chromiumos/infra/proto/go/device"
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

func getEntityByID(ctx context.Context, t *testing.T, id DeviceEntityID) *DeviceEntity {
	d := DeviceEntity{ID: id, Parent: fakeAcestorKey(ctx)}
	if err := datastore.Get(ctx, &d); err != nil {
		t.Errorf("cannot get device by id %s: %v", id, err)
	}
	return &d
}

func getLabConfigByID(ctx context.Context, t *testing.T, id DeviceEntityID) *lab.ChromeOSDevice {
	d := getEntityByID(ctx, t, id)
	var labConfig lab.ChromeOSDevice
	if err := proto.Unmarshal(d.LabConfig, &labConfig); err != nil {
		t.Errorf("cannot unmarshal labconfig of %s: %v", id, err)
	}
	return &labConfig
}

func getLabConfigByHostname(ctx context.Context, t *testing.T, hostname string) *lab.ChromeOSDevice {
	q := datastore.NewQuery(DeviceKind).Ancestor(fakeAcestorKey(ctx)).Eq("Hostname", hostname)
	var devices []DeviceEntity
	if err := datastore.GetAll(ctx, q, &devices); err != nil {
		t.Errorf("cannot query from the datastore for %s: %s", hostname, err)
	}
	So(devices, ShouldHaveLength, 1)

	d := devices[0]
	var labConfig lab.ChromeOSDevice
	if err := proto.Unmarshal(d.LabConfig, &labConfig); err != nil {
		t.Errorf("cannot unmarshal labconfig of %s: %v", d.ID, err)
	}
	return &labConfig
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
			dsResp, err := AddDevices(ctx, devsToAdd, false)
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
				d := getEntityByID(ctx, t, result.Entity.ID)
				got[i] = d.Hostname
			}
			So(got, ShouldResemble, want)
			So(devsToAdd[0].GetId().GetValue(), ShouldNotEqual, "")
			So(devsToAdd[1].GetId().GetValue(), ShouldEqual, "ASSET_ID_123")

		})
		Convey("Add 2 duts and 1 labstation with servo port auto assigned", func() {
			dut1 := mockDut("dut1-1", "id-1", "labstation10")
			dut1.GetDut().GetPeripherals().GetServo().ServoPort = 0
			dut1.GetDut().GetPeripherals().GetServo().ServoSerial = "SN1"
			dut2 := mockDut("dut1-2", "id-2", "labstation10")
			dut2.GetDut().GetPeripherals().GetServo().ServoPort = 0
			dut2.GetDut().GetPeripherals().GetServo().ServoSerial = "SN2"

			labstation := &lab.ChromeOSDevice{
				Device: &lab.ChromeOSDevice_Labstation{
					Labstation: &lab.Labstation{
						Hostname: "labstation10",
						Servos: []*lab.Servo{
							{ServoPort: 9999, ServoSerial: "SN9999"},
							{ServoPort: 9998, ServoSerial: "SN9998"},
							{ServoPort: 9996, ServoSerial: "SN9996"},
						},
					},
				},
			}
			devsToAdd := []*lab.ChromeOSDevice{dut1, labstation, dut2}
			dsResp, err := AddDevices(ctx, devsToAdd, true)
			if err != nil {
				t.Fatal(err)
			}
			So(dsResp.Passed(), ShouldHaveLength, len(devsToAdd))
			So(dsResp.Failed(), ShouldHaveLength, 0)

			// The servo ports are assigned.
			{
				ports := []int{}
				labConfig := getLabConfigByID(ctx, t, DeviceEntityID("id-1"))
				ports = append(ports, int(labConfig.GetDut().GetPeripherals().GetServo().GetServoPort()))
				labConfig = getLabConfigByID(ctx, t, DeviceEntityID("id-2"))
				ports = append(ports, int(labConfig.GetDut().GetPeripherals().GetServo().GetServoPort()))
				So(ports, ShouldResemble, []int{9997, 9995})
			}

			// There's a labstation saved to datastore and has two servos
			// attached.
			{
				labConfig := getLabConfigByHostname(ctx, t, "labstation10")
				So(labConfig.GetLabstation().GetServos(), ShouldHaveLength, 5)
				ports := []int{}
				for _, s := range labConfig.GetLabstation().GetServos() {
					ports = append(ports, int(s.GetServoPort()))
				}
				So(ports, ShouldResemble, []int{9999, 9998, 9997, 9996, 9995})
			}
		})

		Convey("Add device with existing hostname", func() {
			devsToAdd := []*lab.ChromeOSDevice{
				mockDut("dut1", "ID_FAIL", "labstation1"),
				mockDut("dut2", "ID_PASS", "labstation1"),
			}

			dsResp, _ := AddDevices(ctx, devsToAdd, false)
			So(dsResp.Passed(), ShouldHaveLength, 1)
			So(dsResp.Failed(), ShouldHaveLength, 1)

			So(dsResp.Passed()[0].Entity.ID, ShouldEqual, "ID_PASS")
			So(dsResp.Failed()[0].Data.(*lab.ChromeOSDevice).GetId().GetValue(), ShouldEqual, "ID_FAIL")
		})
		Convey("Add device with existing ID", func() {
			devsToAdd := []*lab.ChromeOSDevice{
				mockDut("dut3", "ID_PASS", "labstation1"),
			}

			dsResp, _ := AddDevices(ctx, devsToAdd, false)
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
		_, err := AddDevices(ctx, devsToAdd, false)
		So(err, ShouldBeNil)

		datastore.GetTestable(ctx).Consistent(true)

		Convey("Remove non-existing devices by Ids and hostnames", func() {
			resp := DeleteDevicesByIds(ctx, []string{"1234", "abcd"})
			So(resp.Passed(), ShouldHaveLength, 2)
			So(resp.Failed(), ShouldHaveLength, 0)

			resp = DeleteDevicesByHostnames(ctx, []string{"dutX", "labstationX"})
			So(resp.Passed(), ShouldHaveLength, 0)
			So(resp.Failed(), ShouldHaveLength, 2)

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
		Convey("Get by models", func() {
			result, err := GetDevicesByModels(ctx, []string{"model1", "model2"})
			So(err, ShouldBeNil)
			So(result.Passed(), ShouldBeEmpty)
			So(result.Failed(), ShouldBeEmpty)
		})
	})
	Convey("Get devices from a non-empty datastore", t, func() {
		dut1 := mockDut("dut1", "", "labstation1")
		labstation1 := mockLabstation("labstation1", "ASSET_ID_123")
		dut1.DeviceConfigId = &device.ConfigId{
			ModelId: &device.ModelId{Value: "model1"},
		}
		labstation1.DeviceConfigId = &device.ConfigId{
			ModelId: &device.ModelId{Value: "model2"},
		}
		devsToAdd := []*lab.ChromeOSDevice{dut1, labstation1}
		_, err := AddDevices(ctx, devsToAdd, false)
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
		Convey("Get by models", func() {
			result, err := GetDevicesByModels(ctx, []string{"model3", "model1"})
			So(err, ShouldBeNil)
			So(result.Passed(), ShouldHaveLength, 1)
			So(result.Passed()[0].Entity.Hostname, ShouldEqual, "dut1")
			So(result.Failed(), ShouldHaveLength, 0)
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
			mockLabstation("labstation2", "UUID:03"),
		}
		_, err := AddDevices(ctx, devsToAdd, false)
		So(err, ShouldBeNil)
		labConfig := getLabConfigByID(ctx, t, DeviceEntityID("UUID:01"))
		So(int(labConfig.GetDut().GetPeripherals().GetServo().GetServoPort()), ShouldEqual, 8888)

		datastore.GetTestable(ctx).Consistent(true)
		Convey("Update non-existing devices", func() {
			dut1 := mockDut("dut1", "UUID:01", "labstation2")
			dut1.GetDut().GetPeripherals().GetServo().ServoPort = 0
			result, err := UpdateDeviceSetup(ctx, []*lab.ChromeOSDevice{
				mockDut("dut1", "UUID:ghost", "labstation1"),
				dut1,
			}, true)
			if err != nil {
				t.Fatal(err)
			}
			passed := result.Passed()
			So(passed, ShouldHaveLength, 1)
			So(passed[0].Entity.ID, ShouldEqual, "UUID:01")
			labConfig := getLabConfigByID(ctx, t, DeviceEntityID("UUID:01"))
			So(int(labConfig.GetDut().GetPeripherals().GetServo().GetServoPort()), ShouldEqual, 9999)

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
			mockLabstation("labstation1", ""),
		}
		_, err := AddDevices(ctx, devsToAdd, false)
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

func TestUpdateLabMeta(t *testing.T) {
	t.Parallel()

	Convey("Update devices setup in datastore", t, func() {
		ctx := gaetesting.TestingContextWithAppID("go-test")

		devsToAdd := []*lab.ChromeOSDevice{
			mockDut("dut1", "UUID:01", "labstation1"),
			mockLabstation("labstation1", ""),
		}
		originalServo := devsToAdd[0].GetDut().GetPeripherals().GetServo()
		So(devsToAdd[0].GetDut().GetPeripherals().GetServo().ServoType, ShouldEqual, "v3")
		_, err := AddDevices(ctx, devsToAdd, false)
		So(err, ShouldBeNil)

		datastore.GetTestable(ctx).Consistent(true)
		Convey("Update only ServoType in meta", func() {
			meta := map[string]LabMeta{
				"UUID:01": {
					ServoType: "servo_v4_with_ccd_cr50",
				},
				"UUID:ghost": {},
			}
			result, err := UpdateLabMeta(ctx, meta)
			if err != nil {
				t.Fatal(err)
			}
			passed := result.Passed()
			So(passed, ShouldHaveLength, 1)
			So(passed[0].Entity.ID, ShouldEqual, "UUID:01")
			var p lab.ChromeOSDevice
			passed[0].Entity.GetCrosDeviceProto(&p)
			So(p.GetDut().GetPeripherals().GetServo().ServoType, ShouldEqual, "servo_v4_with_ccd_cr50")

			//validates only the single field was change from original
			newServoPr := proto.MarshalTextString(p.GetDut().GetPeripherals().GetServo())
			originalServoPr := proto.MarshalTextString(originalServo)
			So(newServoPr, ShouldNotEqual, originalServoPr)
			originalServo.ServoType = "servo_v4_with_ccd_cr50"
			originalServoPr = proto.MarshalTextString(originalServo)
			So(newServoPr, ShouldEqual, originalServoPr)

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
		_, err := AddDevices(ctx, devsToAdd, false)
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

func TestBatchUpdateDevices(t *testing.T) {
	t.Parallel()

	Convey("Batch update devices", t, func() {
		ctx := gaetesting.TestingContextWithAppID("go-test")

		devsToAdd := []*lab.ChromeOSDevice{
			mockDut("dut1", "UUID:01", "labstation1"),
			mockLabstation("labstation1", "UUID:02"),
		}
		_, err := AddDevices(ctx, devsToAdd, false)
		So(err, ShouldBeNil)

		datastore.GetTestable(ctx).Consistent(true)
		Convey("update non-existing devices", func() {
			err := BatchUpdateDevices(ctx, []*DeviceProperty{
				{Hostname: "dut2"},
			})
			So(err, ShouldBeNil)
		})
		Convey("update pools", func() {
			err := BatchUpdateDevices(ctx, []*DeviceProperty{
				{Hostname: "dut1", Pool: "pool1"},
			})
			So(err, ShouldBeNil)
			labConfig := getLabConfigByHostname(ctx, t, "dut1")
			So(labConfig.GetDut().GetPools(), ShouldResemble, []string{"pool1"})
		})
		Convey("update multiple devices", func() {
			err := BatchUpdateDevices(ctx, []*DeviceProperty{
				{Hostname: "dut1", Pool: "pool2"},
				{Hostname: "labstation1", PowerunitName: "powerunit", Pool: "pool3"},
			})
			So(err, ShouldBeNil)
			labConfig := getLabConfigByHostname(ctx, t, "dut1")
			So(labConfig.GetDut().GetPools(), ShouldResemble, []string{"pool2"})

			labConfig = getLabConfigByHostname(ctx, t, "labstation1")
			So(labConfig.GetLabstation().GetRpm().GetPowerunitName(), ShouldEqual, "powerunit")
			So(labConfig.GetLabstation().GetPools(), ShouldResemble, []string{"pool3"})
		})
	})
}
