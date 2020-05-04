// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
	proto "infra/unifiedfleet/api/v1/proto"
	api "infra/unifiedfleet/api/v1/rpc"
	. "infra/unifiedfleet/app/constants"
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

func TestCreateMachine(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	chromeOSMachine1 := mockChromeOSMachine("Chromeos-asset-1", "chromeoslab", "samus")
	chromeOSMachine2 := mockChromeOSMachine("", "chromeoslab", "samus")
	chromeOSMachine3 := mockChromeOSMachine("a.b)7&", "chromeoslab", "samus")
	chromeOSMachine4 := mockChromeOSMachine("", "chromeoslab", "samus")
	Convey("CreateMachines", t, func() {
		Convey("Create new machine", func() {
			req := &api.CreateMachineRequest{
				Machine: chromeOSMachine1,
			}
			resp, err := tf.Fleet.CreateMachine(tf.C, req)
			So(err, ShouldBeNil)
			assertMachineEqual(resp, chromeOSMachine1)
		})
		Convey("Create new machine with machine_id", func() {
			req := &api.CreateMachineRequest{
				Machine:   chromeOSMachine4,
				MachineId: "Chrome_Asset-55",
			}
			resp, err := tf.Fleet.CreateMachine(tf.C, req)
			So(err, ShouldBeNil)
			assertMachineEqual(resp, chromeOSMachine4)
		})
		Convey("Create existing machines", func() {
			req := &api.CreateMachineRequest{
				Machine: chromeOSMachine1,
			}
			resp, err := tf.Fleet.CreateMachine(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, alreadyExists)
		})
		Convey("Create new machine - Invalid input nil", func() {
			req := &api.CreateMachineRequest{
				Machine: nil,
			}
			resp, err := tf.Fleet.CreateMachine(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, nilEntity)
		})
		Convey("Create new machine - Invalid input empty name", func() {
			req := &api.CreateMachineRequest{
				Machine: chromeOSMachine2,
			}
			resp, err := tf.Fleet.CreateMachine(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, emptyID)
		})
		Convey("Create new machine - Invalid input invalid characters", func() {
			req := &api.CreateMachineRequest{
				Machine: chromeOSMachine3,
			}
			resp, err := tf.Fleet.CreateMachine(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, invalidCharacters)
		})
	})
}

func TestUpdateMachine(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	chromeOSMachine1 := mockChromeOSMachine("chromeos-asset-1", "chromeoslab", "samus")
	chromeOSMachine2 := mockChromeOSMachine("chromeos-asset-1", "chromeoslab", "veyron")
	chromeBrowserMachine1 := mockChromeBrowserMachine("chrome-asset-1", "chromelab", "machine-1")
	chromeOSMachine3 := mockChromeOSMachine("", "chromeoslab", "samus")
	chromeOSMachine4 := mockChromeOSMachine("a.b)7&", "chromeoslab", "samus")
	Convey("UpdateMachines", t, func() {
		Convey("Update existing machines", func() {
			req := &api.CreateMachineRequest{
				Machine: chromeOSMachine1,
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
			So(err.Error(), ShouldContainSubstring, notFound)
		})
		Convey("Update machine - Invalid input nil", func() {
			req := &api.UpdateMachineRequest{
				Machine: nil,
			}
			resp, err := tf.Fleet.UpdateMachine(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, nilEntity)
		})
		Convey("Update machine - Invalid input empty name", func() {
			req := &api.UpdateMachineRequest{
				Machine: chromeOSMachine3,
			}
			resp, err := tf.Fleet.UpdateMachine(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, emptyID)
		})
		Convey("Update machine - Invalid input invalid characters", func() {
			req := &api.UpdateMachineRequest{
				Machine: chromeOSMachine4,
			}
			resp, err := tf.Fleet.UpdateMachine(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, invalidCharacters)
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
			Machine: chromeOSMachine1,
		}
		resp, err := tf.Fleet.CreateMachine(tf.C, req)
		So(err, ShouldBeNil)
		assertMachineEqual(resp, chromeOSMachine1)
		Convey("Get machine by existing ID", func() {
			req := &api.GetMachineRequest{
				Name: "chromeos-asset-1",
			}
			resp, err := tf.Fleet.GetMachine(tf.C, req)
			So(err, ShouldBeNil)
			assertMachineEqual(resp, chromeOSMachine1)
		})
		Convey("Get machine by non-existing ID", func() {
			req := &api.GetMachineRequest{
				Name: "chrome-asset-1",
			}
			resp, err := tf.Fleet.GetMachine(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, notFound)
		})
		Convey("Get machine - Invalid input empty name", func() {
			req := &api.GetMachineRequest{
				Name: "",
			}
			resp, err := tf.Fleet.GetMachine(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, emptyID)
		})
		Convey("Get machine - Invalid input invalid characters", func() {
			req := &api.GetMachineRequest{
				Name: "a.b)7&",
			}
			resp, err := tf.Fleet.GetMachine(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, invalidCharacters)
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
			So(err.Error(), ShouldContainSubstring, InvalidPageSize)
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
			Machine: chromeOSMachine1,
		}
		resp, err := tf.Fleet.CreateMachine(tf.C, req)
		So(err, ShouldBeNil)
		assertMachineEqual(resp, chromeOSMachine1)
		Convey("Delete machine by existing ID", func() {
			req := &api.DeleteMachineRequest{
				Name: "chromeos-asset-1",
			}
			_, err := tf.Fleet.DeleteMachine(tf.C, req)
			So(err, ShouldBeNil)
			greq := &api.GetMachineRequest{
				Name: "chromeos-asset-1",
			}
			res, err := tf.Fleet.GetMachine(tf.C, greq)
			So(res, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, notFound)
		})
		Convey("Delete machine by non-existing ID", func() {
			req := &api.DeleteMachineRequest{
				Name: "chrome-asset-1",
			}
			_, err := tf.Fleet.DeleteMachine(tf.C, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, notFound)
		})
		Convey("Delete machine - Invalid input empty name", func() {
			req := &api.DeleteMachineRequest{
				Name: "",
			}
			resp, err := tf.Fleet.DeleteMachine(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, emptyID)
		})
		Convey("Delete machine - Invalid input invalid characters", func() {
			req := &api.DeleteMachineRequest{
				Name: "a.b)7&",
			}
			resp, err := tf.Fleet.DeleteMachine(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, invalidCharacters)
		})
	})
}
