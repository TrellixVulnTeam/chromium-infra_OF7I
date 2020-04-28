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

func TestCreateMachines(t *testing.T) {
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
			So(err.Error(), ShouldContainSubstring, "Empty Machine ID")
		})
	})
}

func TestUpdateMachines(t *testing.T) {
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
			So(err.Error(), ShouldContainSubstring, "Empty Machine ID")
		})
	})
}

/*
func TestGetMachinesByID(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	chromeOSMachine1 := mockChromeOSMachine("chromeos-asset-1", "chromeoslab", "samus")
	chromeMachine1 := mockChromeMachine("chrome-asset-1", "chromelab", "machine-1")
	chromeMachine2 := mockChromeMachine("chrome-asset-2", "chromelab", "machine-2")
	Convey("GetMachinesByID", t, func() {
		Convey("Get machines by existing ID", func() {
			req := []*fleet.Machine{chromeOSMachine1, chromeMachine1}
			resp, err := CreateMachines(ctx, req)
			So(err, ShouldBeNil)
			So(resp.Passed(), ShouldHaveLength, 2)
			So(resp.Failed(), ShouldHaveLength, 0)
			assertMachineEqual(resp.Passed()[0].Data.(*fleet.Machine), chromeOSMachine1)
			assertMachineEqual(resp.Passed()[1].Data.(*fleet.Machine), chromeMachine1)

			ids := []string{
				chromeOSMachine1.GetId().GetValue(),
				chromeMachine1.GetId().GetValue(),
			}
			resp = GetMachinesByID(ctx, ids)
			So(err, ShouldBeNil)
			So(resp.Passed(), ShouldHaveLength, 2)
			So(resp.Failed(), ShouldHaveLength, 0)
			assertMachineEqual(resp.Passed()[0].Data.(*fleet.Machine), chromeOSMachine1)
			assertMachineEqual(resp.Passed()[1].Data.(*fleet.Machine), chromeMachine1)
		})
		Convey("Get machines by non-existing ID", func() {
			ids := []string{
				chromeMachine2.GetId().GetValue(),
			}
			resp := GetMachinesByID(ctx, ids)
			So(resp.Passed(), ShouldHaveLength, 0)
			So(resp.Failed(), ShouldHaveLength, 1)
			So(resp.Failed()[0].Err.Error(), ShouldContainSubstring, "datastore: no such entity")
		})
	})
}

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

func TestDeleteMachines(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	Convey("DeleteMachines", t, func() {
		chromeOSMachine1 := mockChromeOSMachine("chromeos-asset-1", "chromeoslab", "samus")
		chromeMachine1 := mockChromeMachine("chrome-asset-1", "chromelab", "machine-1")
		chromeMachine2 := mockChromeMachine("chrome-asset-2", "chromelab", "machine-2")
		req := []*fleet.Machine{chromeOSMachine1, chromeMachine1}
		resp, err := CreateMachines(ctx, req)
		So(err, ShouldBeNil)
		So(resp.Passed(), ShouldHaveLength, 2)
		So(resp.Failed(), ShouldHaveLength, 0)
		assertMachineEqual(resp.Passed()[0].Data.(*fleet.Machine), chromeOSMachine1)
		assertMachineEqual(resp.Passed()[1].Data.(*fleet.Machine), chromeMachine1)

		Convey("Delete machines by existing ID", func() {
			req := []string{
				chromeOSMachine1.GetId().GetValue(),
				chromeMachine1.GetId().GetValue(),
			}
			resp := DeleteMachines(ctx, req)
			So(resp.Passed(), ShouldHaveLength, 2)
			So(resp.Failed(), ShouldHaveLength, 0)
			So(resp.Passed()[0].Data.(*fleet.Machine).Id.GetValue(), ShouldEqual, "chromeos-asset-1")
			So(resp.Passed()[1].Data.(*fleet.Machine).Id.GetValue(), ShouldEqual, "chrome-asset-1")

			res, err := GetAllMachines(ctx)
			So(err, ShouldBeNil)
			So(res.Passed(), ShouldHaveLength, 0)
			So(res.Failed(), ShouldHaveLength, 0)
		})
		Convey("Delete machines by non-existing ID", func() {
			req := []string{
				chromeMachine2.GetId().GetValue(),
			}
			res := DeleteMachines(ctx, req)
			So(res.Passed(), ShouldHaveLength, 0)
			So(res.Failed(), ShouldHaveLength, 1)
			So(res.Failed()[0].Err.Error(), ShouldContainSubstring, "Entity not found")
		})
	})
}
*/
