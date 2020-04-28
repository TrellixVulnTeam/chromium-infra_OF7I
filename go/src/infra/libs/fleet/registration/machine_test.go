// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package registration

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/appengine/gaetesting"
	proto "infra/appengine/unified-fleet/api/v1/proto"
)

func mockChromeOSMachine(id, lab, board string) *proto.Machine {
	return &proto.Machine{
		Name: id,
		Device: &proto.Machine_ChromeosMachine{
			ChromeosMachine: &proto.ChromeOSMachine{
				ReferenceBoard: board,
			},
		},
	}
}

func mockChromeBrowserMachine(id, lab, name string) *proto.Machine {
	return &proto.Machine{
		Name: id,
		Device: &proto.Machine_ChromeBrowserMachine{
			ChromeBrowserMachine: &proto.ChromeBrowserMachine{
				Description: name,
			},
		},
	}
}

func assertMachineEqual(a *proto.Machine, b *proto.Machine) {
	So(a.GetName(), ShouldEqual, b.GetName())
	So(a.GetChromeBrowserMachine().GetDescription(), ShouldEqual,
		b.GetChromeBrowserMachine().GetDescription())
	So(a.GetChromeosMachine().GetReferenceBoard(), ShouldEqual,
		b.GetChromeosMachine().GetReferenceBoard())
}

func getMachineNames(machines []*proto.Machine) []string {
	names := make([]string, len(machines))
	for i, p := range machines {
		names[i] = p.GetName()
	}
	return names
}

func TestCreateMachine(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	datastore.GetTestable(ctx).Consistent(true)
	chromeOSMachine1 := mockChromeOSMachine("chromeos-asset-1", "chromeoslab", "samus")
	chromeOSMachine2 := mockChromeOSMachine("", "chromeoslab", "samus")
	Convey("CreateMachine", t, func() {
		Convey("Create new machine", func() {
			resp, err := CreateMachine(ctx, chromeOSMachine1)
			So(err, ShouldBeNil)
			assertMachineEqual(resp, chromeOSMachine1)
		})
		Convey("Create existing machine", func() {
			resp, err := CreateMachine(ctx, chromeOSMachine1)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "already exists")
		})
		Convey("Create machine - invalid ID", func() {
			resp, err := CreateMachine(ctx, chromeOSMachine2)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Internal Server error")
		})
	})
}

func TestUpdateMachine(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	chromeOSMachine1 := mockChromeOSMachine("chromeos-asset-1", "chromeoslab", "samus")
	chromeOSMachine2 := mockChromeOSMachine("chromeos-asset-1", "chromeoslab", "veyron")
	chromeBrowserMachine1 := mockChromeBrowserMachine("chrome-asset-1", "chromelab", "machine-1")
	chromeOSMachine3 := mockChromeOSMachine("", "chromeoslab", "samus")
	Convey("UpdateMachine", t, func() {
		Convey("Update existing machine", func() {
			resp, err := CreateMachine(ctx, chromeOSMachine1)
			So(err, ShouldBeNil)
			assertMachineEqual(resp, chromeOSMachine1)

			resp, err = UpdateMachine(ctx, chromeOSMachine2)
			So(err, ShouldBeNil)
			assertMachineEqual(resp, chromeOSMachine2)
		})
		Convey("Update non-existing machine", func() {
			resp, err := UpdateMachine(ctx, chromeBrowserMachine1)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Not found")
		})
		Convey("Update machine - invalid ID", func() {
			resp, err := UpdateMachine(ctx, chromeOSMachine3)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Internal Server error")
		})
	})
}

func TestGetMachine(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	chromeOSMachine1 := mockChromeOSMachine("chromeos-asset-3", "chromeoslab", "samus")
	Convey("GetMachine", t, func() {
		Convey("Get machine by existing ID", func() {
			resp, err := CreateMachine(ctx, chromeOSMachine1)
			So(err, ShouldBeNil)
			assertMachineEqual(resp, chromeOSMachine1)
			resp, err = GetMachine(ctx, "chromeos-asset-3")
			So(err, ShouldBeNil)
			assertMachineEqual(resp, chromeOSMachine1)
		})
		Convey("Get machine by non-existing ID", func() {
			resp, err := GetMachine(ctx, "chrome-asset-1")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Not found")
		})
		Convey("Get machine - invalid ID", func() {
			resp, err := GetMachine(ctx, "")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Internal Server error")
		})
	})
}

func TestDeleteMachine(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	chromeOSMachine2 := mockChromeOSMachine("chromeos-asset-2", "chromeoslab", "samus")
	Convey("DeleteMachine", t, func() {
		Convey("Delete machine by existing ID", func() {
			resp, cerr := CreateMachine(ctx, chromeOSMachine2)
			So(cerr, ShouldBeNil)
			assertMachineEqual(resp, chromeOSMachine2)
			err := DeleteMachine(ctx, "chromeos-asset-2")
			So(err, ShouldBeNil)
			res, err := GetMachine(ctx, "chromeos-asset-2")
			So(res, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Not found")
		})
		Convey("Delete machine by non-existing ID", func() {
			err := DeleteMachine(ctx, "chrome-asset-1")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Not found")
		})
		Convey("Delete machine - invalid ID", func() {
			err := DeleteMachine(ctx, "")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Internal Server error")
		})
	})
}

/*
func TestGetAllMachines(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	Convey("GetAllMachines", t, func() {
		Convey("Get empty machines", func() {
			resp, err := GetAllMachines(ctx)
			So(err, ShouldBeNil)
			So(resp.Passed(), ShouldHaveLength, 0)
			So(resp.Failed(), ShouldHaveLength, 0)
		})
		Convey("Get all the machines", func() {
			chromeOSMachine1 := mockChromeOSMachine("chromeos-asset-1", "chromeoslab", "samus")
			chromeMachine1 := mockChromeMachine("chrome-asset-1", "chromelab", "machine-1")
			input := []*fleet.Machine{chromeMachine1, chromeOSMachine1}
			resp, err := CreateMachines(ctx, input)
			So(err, ShouldBeNil)
			So(resp.Passed(), ShouldHaveLength, 2)
			So(resp.Failed(), ShouldHaveLength, 0)
			assertMachineEqual(resp.Passed()[0].Data.(*fleet.Machine), chromeMachine1)
			assertMachineEqual(resp.Passed()[1].Data.(*fleet.Machine), chromeOSMachine1)

			resp, err = GetAllMachines(ctx)
			So(err, ShouldBeNil)
			So(resp.Passed(), ShouldHaveLength, 2)
			So(resp.Failed(), ShouldHaveLength, 0)
			output := []*fleet.Machine{
				resp.Passed()[0].Data.(*fleet.Machine),
				resp.Passed()[1].Data.(*fleet.Machine),
			}
			wants := getMachineNames(input)
			gets := getMachineNames(output)
			So(wants, ShouldResemble, gets)
		})
	})
}
*/
