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
				Dut: &lab.DeviceUnderTest{
					Hostname: "dut1",
				},
			},
		}
		labstation1 := lab.ChromeOSDevice{
			Id: &lab.ChromeOSDeviceID{},
			Device: &lab.ChromeOSDevice_Labstation{
				Labstation: &lab.Labstation{
					Hostname: "labstation1",
				},
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
				Devices: []*lab.ChromeOSDevice{&dut1, &labstation1},
			}
			err := req.Validate()
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Hostname is missing")
		})
		Convey("Duplicated hostnames", func() {
			dut1.GetDut().Hostname = "dut1"
			labstation1.GetLabstation().Hostname = "dut1"
			req := AddCrosDevicesRequest{
				Devices: []*lab.ChromeOSDevice{&dut1, &labstation1},
			}
			err := req.Validate()
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Duplicated hostname found: dut1")
		})
		Convey("Duplicated ID", func() {
			dut1.GetDut().Hostname = "dut1"
			dut1.Id.Value = "ID"
			labstation1.GetLabstation().Hostname = "labstation1"
			labstation1.Id.Value = "ID"
			req := AddCrosDevicesRequest{
				Devices: []*lab.ChromeOSDevice{&dut1, &labstation1},
			}
			err := req.Validate()
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Duplicated id found: ID")
		})
		Convey("Happy path", func() {
			dut1.GetDut().Hostname = "dut1"
			dut1.Id.Value = ""
			labstation1.GetLabstation().Hostname = "labstation1"
			labstation1.Id.Value = ""
			req := AddCrosDevicesRequest{
				Devices: []*lab.ChromeOSDevice{&dut1, &labstation1},
			}
			err := req.Validate()
			So(err, ShouldBeNil)
		})
	})
}
