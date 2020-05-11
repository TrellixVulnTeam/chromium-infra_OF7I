// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"fmt"
	"testing"

	proto "infra/unifiedfleet/api/v1/proto"
	api "infra/unifiedfleet/api/v1/rpc"
	. "infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/model/registration"
	"infra/unifiedfleet/app/util"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
	code "google.golang.org/genproto/googleapis/rpc/code"
	"google.golang.org/grpc/status"
)

func mockChromeOSMachine(id, lab, board string) *proto.Machine {
	return &proto.Machine{
		Name: util.AddPrefix(machineCollection, id),
		Device: &proto.Machine_ChromeosMachine{
			ChromeosMachine: &proto.ChromeOSMachine{
				ReferenceBoard: board,
			},
		},
	}
}

func mockChromeBrowserMachine(id, lab, name string) *proto.Machine {
	return &proto.Machine{
		Name: util.AddPrefix(machineCollection, id),
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

func mockRack(id string, rackCapactiy int32) *proto.Rack {
	return &proto.Rack{
		Name:       util.AddPrefix(rackCollection, id),
		CapacityRu: rackCapactiy,
	}
}

func TestCreateMachine(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	chromeOSMachine1 := mockChromeOSMachine("", "chromeoslab1", "samus1")
	chromeOSMachine2 := mockChromeOSMachine("", "chromeoslab2", "samus2")
	chromeOSMachine3 := mockChromeOSMachine("", "chromeoslab3", "samus3")
	chromeOSMachine4 := mockChromeOSMachine("", "chromeoslab1", "samus1")
	Convey("CreateMachines", t, func() {
		Convey("Create new machine with machine_id", func() {
			req := &api.CreateMachineRequest{
				Machine:   chromeOSMachine1,
				MachineId: "Chromeos-asset-1",
			}
			resp, err := tf.Fleet.CreateMachine(tf.C, req)
			So(err, ShouldBeNil)
			assertMachineEqual(resp, chromeOSMachine1)
		})

		Convey("Create existing machines", func() {
			req := &api.CreateMachineRequest{
				Machine:   chromeOSMachine4,
				MachineId: "Chromeos-asset-1",
			}
			resp, err := tf.Fleet.CreateMachine(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, AlreadyExists)
		})

		Convey("Create new machine - Invalid input nil", func() {
			req := &api.CreateMachineRequest{
				Machine: nil,
			}
			resp, err := tf.Fleet.CreateMachine(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.NilEntity)
		})

		Convey("Create new machine - Invalid input empty ID", func() {
			req := &api.CreateMachineRequest{
				Machine:   chromeOSMachine2,
				MachineId: "",
			}
			resp, err := tf.Fleet.CreateMachine(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.EmptyID)
		})

		Convey("Create new machine - Invalid input invalid characters", func() {
			req := &api.CreateMachineRequest{
				Machine:   chromeOSMachine3,
				MachineId: "a.b)7&",
			}
			resp, err := tf.Fleet.CreateMachine(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.InvalidCharacters)
		})
	})
}

func TestUpdateMachine(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	chromeOSMachine1 := mockChromeOSMachine("", "chromeoslab1", "samus1")
	chromeOSMachine2 := mockChromeOSMachine("chromeos-asset-1", "chromeoslab", "veyron")
	chromeBrowserMachine1 := mockChromeBrowserMachine("chrome-asset-1", "chromelab", "machine-1")
	chromeOSMachine3 := mockChromeOSMachine("", "chromeoslab", "samus")
	chromeOSMachine4 := mockChromeOSMachine("a.b)7&", "chromeoslab", "samus")
	Convey("UpdateMachines", t, func() {
		Convey("Update existing machines", func() {
			req := &api.CreateMachineRequest{
				Machine:   chromeOSMachine1,
				MachineId: "chromeos-asset-1",
			}
			resp, err := tf.Fleet.CreateMachine(tf.C, req)
			So(err, ShouldBeNil)
			assertMachineEqual(resp, chromeOSMachine1)
			ureq := &api.UpdateMachineRequest{
				Machine: chromeOSMachine2,
			}
			resp, err = tf.Fleet.UpdateMachine(tf.C, ureq)
			So(err, ShouldBeNil)
			assertMachineEqual(resp, chromeOSMachine2)
		})

		Convey("Update non-existing machines", func() {
			ureq := &api.UpdateMachineRequest{
				Machine: chromeBrowserMachine1,
			}
			resp, err := tf.Fleet.UpdateMachine(tf.C, ureq)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})

		Convey("Update machine - Invalid input nil", func() {
			req := &api.UpdateMachineRequest{
				Machine: nil,
			}
			resp, err := tf.Fleet.UpdateMachine(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.NilEntity)
		})

		Convey("Update machine - Invalid input empty name", func() {
			chromeOSMachine3.Name = ""
			req := &api.UpdateMachineRequest{
				Machine: chromeOSMachine3,
			}
			resp, err := tf.Fleet.UpdateMachine(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.EmptyName)
		})

		Convey("Update machine - Invalid input invalid characters", func() {
			req := &api.UpdateMachineRequest{
				Machine: chromeOSMachine4,
			}
			resp, err := tf.Fleet.UpdateMachine(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.InvalidCharacters)
		})
	})
}

func TestGetMachine(t *testing.T) {
	t.Parallel()
	Convey("GetMachine", t, func() {
		ctx := testingContext()
		tf, validate := newTestFixtureWithContext(ctx, t)
		defer validate()
		chromeOSMachine1 := mockChromeOSMachine("chromeos-asset-1", "chromeoslab", "samus")
		req := &api.CreateMachineRequest{
			Machine:   chromeOSMachine1,
			MachineId: "chromeos-asset-1",
		}
		resp, err := tf.Fleet.CreateMachine(tf.C, req)
		So(err, ShouldBeNil)
		assertMachineEqual(resp, chromeOSMachine1)
		Convey("Get machine by existing ID", func() {
			req := &api.GetMachineRequest{
				Name: util.AddPrefix(machineCollection, "chromeos-asset-1"),
			}
			resp, err := tf.Fleet.GetMachine(tf.C, req)
			So(err, ShouldBeNil)
			assertMachineEqual(resp, chromeOSMachine1)
		})
		Convey("Get machine by non-existing ID", func() {
			req := &api.GetMachineRequest{
				Name: util.AddPrefix(machineCollection, "chrome-asset-1"),
			}
			resp, err := tf.Fleet.GetMachine(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Get machine - Invalid input empty name", func() {
			req := &api.GetMachineRequest{
				Name: "",
			}
			resp, err := tf.Fleet.GetMachine(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.EmptyName)
		})
		Convey("Get machine - Invalid input invalid characters", func() {
			req := &api.GetMachineRequest{
				Name: util.AddPrefix(machineCollection, "a.b)7&"),
			}
			resp, err := tf.Fleet.GetMachine(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.InvalidCharacters)
		})
	})
}

func TestListMachines(t *testing.T) {
	t.Parallel()
	Convey("ListMachines", t, func() {
		ctx := testingContext()
		tf, validate := newTestFixtureWithContext(ctx, t)
		defer validate()
		machines := make([]*proto.Machine, 0, 4)
		for i := 0; i < 4; i++ {
			chromeOSMachine1 := mockChromeOSMachine("", "chromeoslab", "samus")
			req := &api.CreateMachineRequest{
				Machine:   chromeOSMachine1,
				MachineId: fmt.Sprintf("chromeos-asset-%d", i),
			}
			resp, err := tf.Fleet.CreateMachine(tf.C, req)
			So(err, ShouldBeNil)
			assertMachineEqual(resp, chromeOSMachine1)
			machines = append(machines, resp)
		}

		Convey("ListMachines - page_size negative", func() {
			req := &api.ListMachinesRequest{
				PageSize: -5,
			}
			resp, err := tf.Fleet.ListMachines(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.InvalidPageSize)
		})

		Convey("ListMachines - page_token invalid", func() {
			req := &api.ListMachinesRequest{
				PageSize:  5,
				PageToken: "abc",
			}
			resp, err := tf.Fleet.ListMachines(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InvalidPageToken)
		})

		Convey("ListMachines - Full listing Max PageSize", func() {
			req := &api.ListMachinesRequest{
				PageSize: 2000,
			}
			resp, err := tf.Fleet.ListMachines(tf.C, req)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp.Machines, ShouldResembleProto, machines)
		})

		Convey("ListMachines - Full listing with no pagination", func() {
			req := &api.ListMachinesRequest{
				PageSize: 0,
			}
			resp, err := tf.Fleet.ListMachines(tf.C, req)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp.Machines, ShouldResembleProto, machines)
		})

		Convey("ListMachines - listing with pagination", func() {
			req := &api.ListMachinesRequest{
				PageSize: 3,
			}
			resp, err := tf.Fleet.ListMachines(tf.C, req)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp.Machines, ShouldResembleProto, machines[:3])

			req = &api.ListMachinesRequest{
				PageSize:  3,
				PageToken: resp.NextPageToken,
			}
			resp, err = tf.Fleet.ListMachines(tf.C, req)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp.Machines, ShouldResembleProto, machines[3:])
		})
	})
}

func TestDeleteMachine(t *testing.T) {
	t.Parallel()
	Convey("DeleteMachine", t, func() {
		ctx := testingContext()
		tf, validate := newTestFixtureWithContext(ctx, t)
		defer validate()
		chromeOSMachine1 := mockChromeOSMachine("chromeos-asset-1", "chromeoslab", "samus")
		req := &api.CreateMachineRequest{
			Machine:   chromeOSMachine1,
			MachineId: "chromeos-asset-1",
		}
		resp, err := tf.Fleet.CreateMachine(tf.C, req)
		So(err, ShouldBeNil)
		assertMachineEqual(resp, chromeOSMachine1)
		Convey("Delete machine by existing ID", func() {
			req := &api.DeleteMachineRequest{
				Name: util.AddPrefix(machineCollection, "chromeos-asset-1"),
			}
			_, err := tf.Fleet.DeleteMachine(tf.C, req)
			So(err, ShouldBeNil)
			greq := &api.GetMachineRequest{
				Name: util.AddPrefix(machineCollection, "chromeos-asset-1"),
			}
			res, err := tf.Fleet.GetMachine(tf.C, greq)
			So(res, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Delete machine by non-existing ID", func() {
			req := &api.DeleteMachineRequest{
				Name: util.AddPrefix(machineCollection, "chrome-asset-1"),
			}
			_, err := tf.Fleet.DeleteMachine(tf.C, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Delete machine - Invalid input empty name", func() {
			req := &api.DeleteMachineRequest{
				Name: "",
			}
			resp, err := tf.Fleet.DeleteMachine(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.EmptyName)
		})
		Convey("Delete machine - Invalid input invalid characters", func() {
			req := &api.DeleteMachineRequest{
				Name: util.AddPrefix(machineCollection, "a.b)7&"),
			}
			resp, err := tf.Fleet.DeleteMachine(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.InvalidCharacters)
		})
	})
}

func TestImportMachines(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	Convey("Import browser machines", t, func() {
		Convey("happy path", func() {
			req := &api.ImportMachinesRequest{
				Source: &api.ImportMachinesRequest_MachineDbSource{
					MachineDbSource: &api.MachineDBSource{
						Host: "fake_host",
					},
				},
			}
			res, err := tf.Fleet.ImportMachines(ctx, req)
			So(err, ShouldBeNil)
			So(res.Code, ShouldEqual, code.Code_OK)
			getRes, err := registration.GetAllMachines(ctx)
			So(err, ShouldBeNil)
			So(getRes, ShouldHaveLength, len(testMachines))
			gets := getMachineNames(*getRes)
			So(gets, ShouldResemble, testMachines)
		})
		Convey("import browser machines with empty machineDB host", func() {
			req := &api.ImportMachinesRequest{
				Source: &api.ImportMachinesRequest_MachineDbSource{
					MachineDbSource: &api.MachineDBSource{
						Host: "",
					},
				},
			}
			_, err := tf.Fleet.ImportMachines(ctx, req)
			So(err, ShouldNotBeNil)
			s, ok := status.FromError(err)
			So(ok, ShouldBeTrue)
			So(s.Code(), ShouldEqual, code.Code_INVALID_ARGUMENT)
		})
		Convey("import browser machines with empty machineDB source", func() {
			req := &api.ImportMachinesRequest{
				Source: &api.ImportMachinesRequest_MachineDbSource{},
			}
			_, err := tf.Fleet.ImportMachines(ctx, req)
			So(err, ShouldNotBeNil)
			s, ok := status.FromError(err)
			So(ok, ShouldBeTrue)
			So(s.Code(), ShouldEqual, code.Code_INVALID_ARGUMENT)
		})
	})
}

func TestCreateRack(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	rack1 := mockRack("", 5)
	rack2 := mockRack("", 4)
	rack3 := mockRack("", 2)
	Convey("CreateRack", t, func() {
		Convey("Create new rack with rack_id", func() {
			req := &api.CreateRackRequest{
				Rack:   rack1,
				RackId: "Rack-1",
			}
			resp, err := tf.Fleet.CreateRack(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, rack1)
		})

		Convey("Create existing rack", func() {
			req := &api.CreateRackRequest{
				Rack:   rack3,
				RackId: "Rack-1",
			}
			resp, err := tf.Fleet.CreateRack(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, AlreadyExists)
		})

		Convey("Create new rack - Invalid input nil", func() {
			req := &api.CreateRackRequest{
				Rack: nil,
			}
			resp, err := tf.Fleet.CreateRack(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.NilEntity)
		})

		Convey("Create new rack - Invalid input empty ID", func() {
			req := &api.CreateRackRequest{
				Rack:   rack2,
				RackId: "",
			}
			resp, err := tf.Fleet.CreateRack(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.EmptyID)
		})

		Convey("Create new rack - Invalid input invalid characters", func() {
			req := &api.CreateRackRequest{
				Rack:   rack2,
				RackId: "a.b)7&",
			}
			resp, err := tf.Fleet.CreateRack(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.InvalidCharacters)
		})
	})
}

func TestUpdateRack(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	rack1 := mockRack("", 5)
	rack2 := mockRack("rack-1", 10)
	rack3 := mockRack("rack-3", 6)
	rack4 := mockRack("a.b)7&", 6)
	Convey("UpdateRack", t, func() {
		Convey("Update existing rack", func() {
			req := &api.CreateRackRequest{
				Rack:   rack1,
				RackId: "rack-1",
			}
			resp, err := tf.Fleet.CreateRack(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, rack1)
			ureq := &api.UpdateRackRequest{
				Rack: rack2,
			}
			resp, err = tf.Fleet.UpdateRack(tf.C, ureq)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, rack2)
		})

		Convey("Update non-existing rack", func() {
			ureq := &api.UpdateRackRequest{
				Rack: rack3,
			}
			resp, err := tf.Fleet.UpdateRack(tf.C, ureq)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})

		Convey("Update rack - Invalid input nil", func() {
			req := &api.UpdateRackRequest{
				Rack: nil,
			}
			resp, err := tf.Fleet.UpdateRack(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.NilEntity)
		})

		Convey("Update rack - Invalid input empty name", func() {
			rack3.Name = ""
			req := &api.UpdateRackRequest{
				Rack: rack3,
			}
			resp, err := tf.Fleet.UpdateRack(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.EmptyName)
		})

		Convey("Update rack - Invalid input invalid characters", func() {
			req := &api.UpdateRackRequest{
				Rack: rack4,
			}
			resp, err := tf.Fleet.UpdateRack(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.InvalidCharacters)
		})
	})
}

func TestGetRack(t *testing.T) {
	t.Parallel()
	Convey("GetRack", t, func() {
		ctx := testingContext()
		tf, validate := newTestFixtureWithContext(ctx, t)
		defer validate()
		rack1 := mockRack("rack-1", 10)
		req := &api.CreateRackRequest{
			Rack:   rack1,
			RackId: "rack-1",
		}
		resp, err := tf.Fleet.CreateRack(tf.C, req)
		So(err, ShouldBeNil)
		So(resp, ShouldResembleProto, rack1)
		Convey("Get rack by existing ID", func() {
			req := &api.GetRackRequest{
				Name: util.AddPrefix(rackCollection, "rack-1"),
			}
			resp, err := tf.Fleet.GetRack(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, rack1)
		})
		Convey("Get rack by non-existing ID", func() {
			req := &api.GetRackRequest{
				Name: util.AddPrefix(rackCollection, "rack-2"),
			}
			resp, err := tf.Fleet.GetRack(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Get rack - Invalid input empty name", func() {
			req := &api.GetRackRequest{
				Name: "",
			}
			resp, err := tf.Fleet.GetRack(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.EmptyName)
		})
		Convey("Get rack - Invalid input invalid characters", func() {
			req := &api.GetRackRequest{
				Name: util.AddPrefix(rackCollection, "a.b)7&"),
			}
			resp, err := tf.Fleet.GetRack(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.InvalidCharacters)
		})
	})
}

func TestListRacks(t *testing.T) {
	t.Parallel()
	Convey("ListRacks", t, func() {
		ctx := testingContext()
		tf, validate := newTestFixtureWithContext(ctx, t)
		defer validate()
		racks := make([]*proto.Rack, 0, 4)
		for i := 0; i < 4; i++ {
			rack1 := mockRack("", 10)
			req := &api.CreateRackRequest{
				Rack:   rack1,
				RackId: fmt.Sprintf("rack-%d", i),
			}
			resp, err := tf.Fleet.CreateRack(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, rack1)
			racks = append(racks, resp)
		}

		Convey("ListRacks - page_size negative", func() {
			req := &api.ListRacksRequest{
				PageSize: -5,
			}
			resp, err := tf.Fleet.ListRacks(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.InvalidPageSize)
		})

		Convey("ListRacks - page_token invalid", func() {
			req := &api.ListRacksRequest{
				PageSize:  5,
				PageToken: "abc",
			}
			resp, err := tf.Fleet.ListRacks(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InvalidPageToken)
		})

		Convey("ListRacks - Full listing Max PageSize", func() {
			req := &api.ListRacksRequest{
				PageSize: 2000,
			}
			resp, err := tf.Fleet.ListRacks(tf.C, req)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp.Racks, ShouldResembleProto, racks)
		})

		Convey("ListRacks - Full listing with no pagination", func() {
			req := &api.ListRacksRequest{
				PageSize: 0,
			}
			resp, err := tf.Fleet.ListRacks(tf.C, req)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp.Racks, ShouldResembleProto, racks)
		})

		Convey("ListRacks - listing with pagination", func() {
			req := &api.ListRacksRequest{
				PageSize: 3,
			}
			resp, err := tf.Fleet.ListRacks(tf.C, req)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp.Racks, ShouldResembleProto, racks[:3])

			req = &api.ListRacksRequest{
				PageSize:  3,
				PageToken: resp.NextPageToken,
			}
			resp, err = tf.Fleet.ListRacks(tf.C, req)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp.Racks, ShouldResembleProto, racks[3:])
		})
	})
}

func TestDeleteRack(t *testing.T) {
	t.Parallel()
	Convey("DeleteRack", t, func() {
		ctx := testingContext()
		tf, validate := newTestFixtureWithContext(ctx, t)
		defer validate()
		rack1 := mockRack("", 10)
		req := &api.CreateRackRequest{
			Rack:   rack1,
			RackId: "rack-1",
		}
		resp, err := tf.Fleet.CreateRack(tf.C, req)
		So(err, ShouldBeNil)
		So(resp, ShouldResembleProto, rack1)
		Convey("Delete rack by existing ID", func() {
			req := &api.DeleteRackRequest{
				Name: util.AddPrefix(rackCollection, "rack-1"),
			}
			_, err := tf.Fleet.DeleteRack(tf.C, req)
			So(err, ShouldBeNil)
			greq := &api.GetRackRequest{
				Name: util.AddPrefix(rackCollection, "rack-1"),
			}
			res, err := tf.Fleet.GetRack(tf.C, greq)
			So(res, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Delete rack by non-existing ID", func() {
			req := &api.DeleteRackRequest{
				Name: util.AddPrefix(rackCollection, "rack-2"),
			}
			_, err := tf.Fleet.DeleteRack(tf.C, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Delete rack - Invalid input empty name", func() {
			req := &api.DeleteRackRequest{
				Name: "",
			}
			resp, err := tf.Fleet.DeleteRack(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.EmptyName)
		})
		Convey("Delete rack - Invalid input invalid characters", func() {
			req := &api.DeleteRackRequest{
				Name: util.AddPrefix(rackCollection, "a.b)7&"),
			}
			resp, err := tf.Fleet.DeleteRack(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.InvalidCharacters)
		})
	})
}

func TestImportNics(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	Convey("Import nics", t, func() {
		Convey("happy path", func() {
			req := &api.ImportNicsRequest{
				Source: &api.ImportNicsRequest_MachineDbSource{
					MachineDbSource: &api.MachineDBSource{
						Host: "fake_host",
					},
				},
			}
			res, err := tf.Fleet.ImportNics(ctx, req)
			So(err, ShouldBeNil)
			So(res.Code, ShouldEqual, code.Code_OK)
		})
		// Invalid & Empty machine DB hosts are tested in TestImportMachines
		// Skip testing here
	})
}

func getMachineNames(res OpResults) []string {
	names := make([]string, 0)
	for _, r := range res {
		names = append(names, r.Data.(*proto.Machine).GetName())
	}
	return names
}
