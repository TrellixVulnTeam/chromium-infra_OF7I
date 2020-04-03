// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package registration

import (
	fleet "infra/libs/fleet/protos/go"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/appengine/gaetesting"
)

func mockChromeOSMachine(id, lab, board string) *fleet.Machine {
	return &fleet.Machine{
		Id: &fleet.MachineID{
			Value: id,
		},
		Location: &fleet.Location{
			Lab: lab,
		},
		Device: &fleet.Machine_ChromeosMachine{
			ChromeosMachine: &fleet.ChromeOSMachine{
				ReferenceBoard: board,
			},
		},
	}
}

func mockChromeMachine(id, lab, name string) *fleet.Machine {
	return &fleet.Machine{
		Id: &fleet.MachineID{
			Value: id,
		},
		Location: &fleet.Location{
			Lab: lab,
		},
		Device: &fleet.Machine_ChromeMachine{
			ChromeMachine: &fleet.ChromeMachine{
				Name: name,
			},
		},
	}
}

func assertMachineEqual(a, b *fleet.Machine) {
	So(a.GetId().GetValue(), ShouldEqual, b.GetId().GetValue())
	So(a.GetLocation().GetLab(), ShouldEqual, b.GetLocation().GetLab())
	So(a.GetChromeMachine().GetName(), ShouldEqual,
		b.GetChromeMachine().GetName())
	So(a.GetChromeosMachine().GetReferenceBoard(), ShouldEqual,
		b.GetChromeosMachine().GetReferenceBoard())
}

func TestCreateMachines(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	datastore.GetTestable(ctx).Consistent(true)
	chromeOSMachine1 := mockChromeOSMachine("chromeos-asset-1", "chromeoslab", "samus")
	chromeMachine1 := mockChromeMachine("chrome-asset-1", "chromelab", "machine-1")
	Convey("CreateMachines", t, func() {
		Convey("Create 2 new machines", func() {
			req := []*fleet.Machine{chromeOSMachine1, chromeMachine1}
			resp, err := CreateMachines(ctx, req)
			So(err, ShouldBeNil)
			So(resp.Passed(), ShouldHaveLength, 2)
			So(resp.Failed(), ShouldHaveLength, 0)
			assertMachineEqual(resp.Passed()[0].Data.(*fleet.Machine), chromeOSMachine1)
			assertMachineEqual(resp.Passed()[1].Data.(*fleet.Machine), chromeMachine1)
		})
		Convey("Create existing machines", func() {
			chromeMachine2 := mockChromeMachine("chrome-asset-2", "chromelab", "machine-2")
			req := []*fleet.Machine{chromeOSMachine1, chromeMachine2}
			resp, err := CreateMachines(ctx, req)
			So(err, ShouldBeNil)
			So(resp.Passed(), ShouldHaveLength, 1)
			So(resp.Failed(), ShouldHaveLength, 1)
			assertMachineEqual(resp.Failed()[0].Data.(*fleet.Machine), chromeOSMachine1)
			assertMachineEqual(resp.Passed()[0].Data.(*fleet.Machine), chromeMachine2)
		})
	})
}

func TestUpdateMachines(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	chromeOSMachine1 := mockChromeOSMachine("chromeos-asset-1", "chromeoslab", "samus")
	chromeOSMachine2 := mockChromeOSMachine("chromeos-asset-1", "chromeoslab", "veyron")
	chromeOSMachine3 := mockChromeOSMachine("chromeos-asset-1", "chromeoslab", "minnie")
	chromeMachine1 := mockChromeMachine("chrome-asset-1", "chromelab", "machine-1")
	Convey("UpdateMachines", t, func() {
		Convey("Update existing machines", func() {
			req := []*fleet.Machine{chromeOSMachine1}
			resp, err := CreateMachines(ctx, req)
			So(err, ShouldBeNil)
			So(resp.Failed(), ShouldHaveLength, 0)
			So(resp.Passed(), ShouldHaveLength, 1)

			req = []*fleet.Machine{chromeOSMachine2}
			resp, err = UpdateMachines(ctx, req)
			So(err, ShouldBeNil)
			So(resp.Passed(), ShouldHaveLength, 1)
			So(resp.Failed(), ShouldHaveLength, 0)
			assertMachineEqual(resp.Passed()[0].Data.(*fleet.Machine), chromeOSMachine2)
		})
		Convey("Update non-existing machines", func() {
			req := []*fleet.Machine{chromeMachine1, chromeOSMachine3}
			resp, err := UpdateMachines(ctx, req)
			So(err, ShouldBeNil)
			So(resp.Passed(), ShouldHaveLength, 1)
			So(resp.Failed(), ShouldHaveLength, 1)
			assertMachineEqual(resp.Passed()[0].Data.(*fleet.Machine), chromeOSMachine3)
			assertMachineEqual(resp.Failed()[0].Data.(*fleet.Machine), chromeMachine1)
		})
	})
}
