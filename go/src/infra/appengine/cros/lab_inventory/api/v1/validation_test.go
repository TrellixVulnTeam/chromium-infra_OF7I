// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package api

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/chromiumos/infra/proto/go/lab"
)

func TestAddDevicesValidation(t *testing.T) {
	t.Parallel()

	Convey("Add devices to storage backend", t, func() {
		dut1 := lab.ChromeOSDevice{
			Id: &lab.ChromeOSDeviceID{},
			Device: &lab.ChromeOSDevice_Dut{
				Dut: &lab.DeviceUnderTest{Hostname: "dut1"},
			},
		}
		labstation1 := lab.ChromeOSDevice{
			Id: &lab.ChromeOSDeviceID{},
			Device: &lab.ChromeOSDevice_Labstation{
				Labstation: &lab.Labstation{Hostname: "labstation1"},
			},
		}
		labstation2 := lab.ChromeOSDevice{
			Id: &lab.ChromeOSDeviceID{},
			Device: &lab.ChromeOSDevice_Labstation{
				Labstation: &lab.Labstation{Hostname: "labstation2"},
			},
		}
		Convey("Empty request", func() {
			req := &AddCrosDevicesRequest{}
			err := req.Validate()
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "no devices to add")
		})
		Convey("Zero device", func() {
			req := AddCrosDevicesRequest{
				Devices: []*lab.ChromeOSDevice{},
			}
			err := req.Validate()
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "no devices to add")
		})
		Convey("Missing hostname", func() {
			labstation1.GetLabstation().Hostname = ""
			req := AddCrosDevicesRequest{
				Devices: []*lab.ChromeOSDevice{&labstation1, &labstation2},
			}
			err := req.Validate()
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Hostname is missing")
		})
		Convey("Duplicated hostnames", func() {
			labstation2.GetLabstation().Hostname = "labstation1"
			req := AddCrosDevicesRequest{
				Devices: []*lab.ChromeOSDevice{&labstation1, &labstation2},
			}
			err := req.Validate()
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Duplicated hostname found: labstation1")
		})
		Convey("Duplicated ID", func() {
			labstation1.Id.Value = "ID"
			labstation2.Id.Value = "ID"
			req := AddCrosDevicesRequest{
				Devices: []*lab.ChromeOSDevice{&labstation1, &labstation2},
			}
			err := req.Validate()
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Duplicated id found: ID")
		})
		Convey("Mix DUT and labstation", func() {
			req := AddCrosDevicesRequest{
				Devices: []*lab.ChromeOSDevice{&dut1, &labstation2},
			}
			err := req.Validate()
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "mix")
		})
		Convey("Happy path", func() {
			dut1.GetDut().Hostname = "dut1"
			dut1.Id.Value = ""
			labstation1.GetLabstation().Hostname = "labstation1"
			labstation1.Id.Value = ""
			req := AddCrosDevicesRequest{
				Devices: []*lab.ChromeOSDevice{&labstation1, &labstation2},
			}
			err := req.Validate()
			So(err, ShouldBeNil)
		})
	})
}

func TestDeleteDevicesValidation(t *testing.T) {
	t.Parallel()

	hostname := DeviceID{
		Id: &DeviceID_Hostname{
			Hostname: "the_hostname",
		},
	}
	id := DeviceID{
		Id: &DeviceID_ChromeosDeviceId{
			ChromeosDeviceId: "UUID:123",
		},
	}
	Convey("Delete devices from storage backend", t, func() {
		Convey("Empty request", func() {
			req := &DeleteCrosDevicesRequest{}
			err := req.Validate()
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "no devices to remove")
		})
		Convey("Zero device", func() {
			req := DeleteCrosDevicesRequest{
				Ids: []*DeviceID{},
			}
			err := req.Validate()
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "no devices to remove")
		})
		Convey("Duplicated hostname", func() {
			req := DeleteCrosDevicesRequest{
				Ids: []*DeviceID{&hostname, &hostname},
			}
			err := req.Validate()
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Duplicated hostname found: the_hostname")
		})
		Convey("Duplicated device id", func() {
			req := DeleteCrosDevicesRequest{
				Ids: []*DeviceID{&id, &id},
			}
			err := req.Validate()
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Duplicated id found: UUID:123")
		})
		Convey("Happy path", func() {
			req := DeleteCrosDevicesRequest{
				Ids: []*DeviceID{&hostname, &id},
			}
			err := req.Validate()
			So(err, ShouldBeNil)
		})
	})
}

func TestGetDevicesValidation(t *testing.T) {
	t.Parallel()

	hostname := DeviceID{
		Id: &DeviceID_Hostname{
			Hostname: "the_hostname",
		},
	}
	id := DeviceID{
		Id: &DeviceID_ChromeosDeviceId{
			ChromeosDeviceId: "UUID:123",
		},
	}
	Convey("Delete devices from storage backend", t, func() {
		Convey("Empty request", func() {
			req := GetCrosDevicesRequest{}
			err := req.Validate()
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "must specify device ID(s)")
		})
		Convey("Happy path", func() {
			req := GetCrosDevicesRequest{
				Ids: []*DeviceID{&hostname, &id},
			}
			err := req.Validate()
			So(err, ShouldBeNil)
		})
	})
}

func TestUpdateCrosDevicesSetupValidation(t *testing.T) {
	t.Parallel()
	dut1 := lab.ChromeOSDevice{
		Id: &lab.ChromeOSDeviceID{Value: "UUID:01"},
		Device: &lab.ChromeOSDevice_Dut{
			Dut: &lab.DeviceUnderTest{Hostname: "dut1"},
		},
	}
	Convey("Update Chrome OS devices setup", t, func() {
		Convey("empty request", func() {
			req := &UpdateCrosDevicesSetupRequest{}
			err := req.Validate()
			So(err, ShouldNotBeNil)
		})

		Convey("zero devices", func() {
			req := &UpdateCrosDevicesSetupRequest{Devices: nil}
			err := req.Validate()
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "no devices to update")
		})

		Convey("Request has two identical entries", func() {
			req := &UpdateCrosDevicesSetupRequest{Devices: []*lab.ChromeOSDevice{&dut1, &dut1}}
			err := req.Validate()
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Duplicated id found")
		})
		Convey("Happy path", func() {
			req := &UpdateCrosDevicesSetupRequest{Devices: []*lab.ChromeOSDevice{&dut1}}
			err := req.Validate()
			So(err, ShouldBeNil)
		})
	})
}

func TestUpdateDutStatusValidation(t *testing.T) {
	t.Parallel()
	state1 := lab.DutState{
		Id: &lab.ChromeOSDeviceID{Value: "UUID:01"},
	}
	dutMeta1 := DutMeta{
		ChromeosDeviceId: "UUID:11",
	}
	Convey("Update DUT status", t, func() {
		Convey("empty request", func() {
			req := &UpdateDutsStatusRequest{}
			err := req.Validate()
			So(err, ShouldNotBeNil)
		})

		Convey("zero devices", func() {
			req := &UpdateDutsStatusRequest{States: nil}
			err := req.Validate()
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "no devices to update")
		})

		Convey("Request has two identical entries", func() {
			req := &UpdateDutsStatusRequest{States: []*lab.DutState{&state1, &state1}}
			err := req.Validate()
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Duplicated id found")
		})
		Convey("Request has two identical dut metas", func() {
			req := &UpdateDutsStatusRequest{
				States:   []*lab.DutState{&state1},
				DutMetas: []*DutMeta{&dutMeta1, &dutMeta1},
			}
			err := req.Validate()
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Duplicated id found in meta")
		})
		Convey("Request has unmatched meta and state", func() {
			req := &UpdateDutsStatusRequest{
				States:   []*lab.DutState{&state1},
				DutMetas: []*DutMeta{&dutMeta1},
			}
			err := req.Validate()
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Cannot update meta without valid dut states")
		})
		Convey("Happy path", func() {
			req := &UpdateDutsStatusRequest{States: []*lab.DutState{&state1}}
			err := req.Validate()
			So(err, ShouldBeNil)
		})
	})
}

func TestBatchUpdateDevicesValidation(t *testing.T) {
	t.Parallel()
	Convey("Batch update devcies", t, func() {
		Convey("empty request", func() {
			req := &BatchUpdateDevicesRequest{}
			err := req.Validate()
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "no devices")
		})
		Convey("duplicated hostname", func() {
			req := &BatchUpdateDevicesRequest{
				DeviceProperties: []*DeviceProperty{
					{Hostname: "host", Pool: "pool"},
					{Hostname: "host"},
				},
			}
			err := req.Validate()
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Duplicated hostname")
		})
		Convey("no property", func() {
			req := &BatchUpdateDevicesRequest{DeviceProperties: []*DeviceProperty{{Hostname: "host"}}}
			err := req.Validate()
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "nothing to update")
		})
		Convey("no hostname", func() {
			req := &BatchUpdateDevicesRequest{DeviceProperties: []*DeviceProperty{{Pool: "pool"}}}
			err := req.Validate()
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Hostname is missing")
		})
		Convey("happy path 1", func() {
			req := &BatchUpdateDevicesRequest{
				DeviceProperties: []*DeviceProperty{{Hostname: "host", Pool: "pool"}},
			}
			err := req.Validate()
			So(err, ShouldBeNil)
		})
		Convey("happy path 2", func() {
			req := &BatchUpdateDevicesRequest{
				DeviceProperties: []*DeviceProperty{{
					Hostname: "host",
					Rpm:      &DeviceProperty_Rpm{PowerunitName: "rpm host"},
				}},
			}
			err := req.Validate()
			So(err, ShouldBeNil)
		})
	})
}
