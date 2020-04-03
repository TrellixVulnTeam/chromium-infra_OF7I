// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"infra/appengine/unified-fleet/api/v1"
	fleet "infra/libs/fleet/protos/go"
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

func getMachineNames(machines []*fleet.Machine) []string {
	names := make([]string, len(machines))
	for i, p := range machines {
		names[i] = p.GetId().GetValue()
	}
	return names
}

func TestCreateMachines(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	chromeOSMachine1 := mockChromeOSMachine("chromeos-asset-1", "chromeoslab", "samus")
	chromeMachine1 := mockChromeMachine("chrome-asset-1", "chromelab", "machine-1")
	Convey("CreateMachines", t, func() {
		Convey("Create new machines", func() {
			req := &api.MachineList{
				Machine: []*fleet.Machine{chromeOSMachine1, chromeMachine1},
			}
			resp, err := tf.Registration.CreateMachines(tf.C, req)
			So(err, ShouldBeNil)
			So(resp.GetPassed(), ShouldHaveLength, 2)
			So(resp.GetFailed(), ShouldHaveLength, 0)
			assertMachineEqual(resp.GetPassed()[0].Machine, chromeOSMachine1)
			assertMachineEqual(resp.GetPassed()[1].Machine, chromeMachine1)
		})
		Convey("Create existing machines", func() {
			chromeMachine2 := mockChromeMachine("chrome-asset-2", "chromelab", "machine-2")
			req := &api.MachineList{
				Machine: []*fleet.Machine{chromeOSMachine1, chromeMachine2},
			}
			resp, err := tf.Registration.CreateMachines(tf.C, req)
			So(err, ShouldBeNil)
			So(resp.GetPassed(), ShouldHaveLength, 1)
			So(resp.GetFailed(), ShouldHaveLength, 1)
			assertMachineEqual(resp.GetFailed()[0].Machine, chromeOSMachine1)
			assertMachineEqual(resp.GetPassed()[0].Machine, chromeMachine2)
		})
	})
}

func TestUpdateMachines(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	chromeOSMachine1 := mockChromeOSMachine("chromeos-asset-1", "chromeoslab", "samus")
	chromeOSMachine2 := mockChromeOSMachine("chromeos-asset-1", "chromeoslab", "veyron")
	chromeOSMachine3 := mockChromeOSMachine("chromeos-asset-1", "chromeoslab", "minnie")
	chromeMachine1 := mockChromeMachine("chrome-asset-1", "chromelab", "machine-1")
	Convey("UpdateMachines", t, func() {
		Convey("Update existing machines", func() {
			req := &api.MachineList{
				Machine: []*fleet.Machine{chromeOSMachine1},
			}
			resp, err := tf.Registration.CreateMachines(tf.C, req)
			So(err, ShouldBeNil)
			So(resp.GetFailed(), ShouldHaveLength, 0)
			So(resp.GetPassed(), ShouldHaveLength, 1)

			req = &api.MachineList{
				Machine: []*fleet.Machine{chromeOSMachine2},
			}
			resp, err = tf.Registration.UpdateMachines(tf.C, req)
			So(err, ShouldBeNil)
			So(resp.GetPassed(), ShouldHaveLength, 1)
			So(resp.GetFailed(), ShouldHaveLength, 0)
			assertMachineEqual(resp.GetPassed()[0].Machine, chromeOSMachine2)
		})
		Convey("Update non-existing machines", func() {
			req := &api.MachineList{
				Machine: []*fleet.Machine{chromeMachine1, chromeOSMachine3},
			}
			resp, err := tf.Registration.UpdateMachines(tf.C, req)
			So(err, ShouldBeNil)
			So(resp.GetPassed(), ShouldHaveLength, 1)
			So(resp.GetFailed(), ShouldHaveLength, 1)
			assertMachineEqual(resp.GetPassed()[0].Machine, chromeOSMachine3)
			assertMachineEqual(resp.GetFailed()[0].Machine, chromeMachine1)
		})
	})
}

func TestGetMachines(t *testing.T) {
	t.Parallel()
	Convey("GetMachines", t, func() {
		ctx := testingContext()
		tf, validate := newTestFixtureWithContext(ctx, t)
		defer validate()
		chromeOSMachine1 := mockChromeOSMachine("chromeos-asset-1", "chromeoslab", "samus")
		chromeMachine1 := mockChromeMachine("chrome-asset-1", "chromelab", "machine-1")
		chromeMachine2 := mockChromeMachine("chrome-asset-2", "chromelab", "machine-2")
		req := &api.MachineList{
			Machine: []*fleet.Machine{chromeOSMachine1, chromeMachine1},
		}
		resp, err := tf.Registration.CreateMachines(tf.C, req)
		So(err, ShouldBeNil)
		So(resp.GetPassed(), ShouldHaveLength, 2)
		So(resp.GetFailed(), ShouldHaveLength, 0)
		assertMachineEqual(resp.GetPassed()[0].Machine, chromeOSMachine1)
		assertMachineEqual(resp.GetPassed()[1].Machine, chromeMachine1)
		Convey("Get machines by existing ID", func() {
			req := &api.EntityIDList{
				Id: []string{
					chromeOSMachine1.GetId().GetValue(),
					chromeMachine1.GetId().GetValue(),
				},
			}
			resp, err := tf.Registration.GetMachines(tf.C, req)
			So(err, ShouldBeNil)
			So(resp.GetPassed(), ShouldHaveLength, 2)
			So(resp.GetFailed(), ShouldHaveLength, 0)
			assertMachineEqual(resp.GetPassed()[0].Machine, chromeOSMachine1)
			assertMachineEqual(resp.GetPassed()[1].Machine, chromeMachine1)
		})
		Convey("Get machines by non-existing ID", func() {
			req := &api.EntityIDList{
				Id: []string{
					chromeMachine2.GetId().GetValue(),
				},
			}
			_, err := tf.Registration.GetMachines(tf.C, req)
			So(err, ShouldNotBeNil)
		})
	})
}

func TestListMachines(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	Convey("ListMachines", t, func() {
		Convey("List empty machines", func() {
			req := &api.ListMachinesRequest{}
			resp, err := tf.Registration.ListMachines(tf.C, req)
			So(err, ShouldBeNil)
			So(resp.GetPassed(), ShouldHaveLength, 0)
			So(resp.GetFailed(), ShouldHaveLength, 0)
		})
		Convey("List all the machines", func() {
			chromeOSMachine1 := mockChromeOSMachine("chromeos-asset-1", "chromeoslab", "samus")
			chromeMachine1 := mockChromeMachine("chrome-asset-1", "chromelab", "machine-1")
			input := []*fleet.Machine{chromeMachine1, chromeOSMachine1}
			req := &api.MachineList{
				Machine: input,
			}
			resp, err := tf.Registration.CreateMachines(tf.C, req)
			So(err, ShouldBeNil)
			So(resp.GetPassed(), ShouldHaveLength, 2)
			So(resp.GetFailed(), ShouldHaveLength, 0)
			assertMachineEqual(resp.GetPassed()[0].Machine, chromeMachine1)
			assertMachineEqual(resp.GetPassed()[1].Machine, chromeOSMachine1)

			listReq := &api.ListMachinesRequest{}
			resp, err = tf.Registration.ListMachines(tf.C, listReq)
			So(err, ShouldBeNil)
			So(resp.GetPassed(), ShouldHaveLength, 2)
			So(resp.GetFailed(), ShouldHaveLength, 0)
			output := []*fleet.Machine{resp.GetPassed()[0].Machine, resp.GetPassed()[1].Machine}
			wants := getMachineNames(input)
			gets := getMachineNames(output)
			So(wants, ShouldResemble, gets)
		})
	})
}

func TestDeleteMachines(t *testing.T) {
	t.Parallel()
	Convey("DeleteMachines", t, func() {
		ctx := testingContext()
		tf, validate := newTestFixtureWithContext(ctx, t)
		defer validate()
		chromeOSMachine1 := mockChromeOSMachine("chromeos-asset-1", "chromeoslab", "samus")
		chromeMachine1 := mockChromeMachine("chrome-asset-1", "chromelab", "machine-1")
		chromeMachine2 := mockChromeMachine("chrome-asset-2", "chromelab", "machine-2")
		req := &api.MachineList{
			Machine: []*fleet.Machine{chromeOSMachine1, chromeMachine1},
		}
		resp, err := tf.Registration.CreateMachines(tf.C, req)
		So(err, ShouldBeNil)
		So(resp.GetPassed(), ShouldHaveLength, 2)
		So(resp.GetFailed(), ShouldHaveLength, 0)
		assertMachineEqual(resp.GetPassed()[0].Machine, chromeOSMachine1)
		assertMachineEqual(resp.GetPassed()[1].Machine, chromeMachine1)
		Convey("Delete machines by existing ID", func() {
			req := &api.EntityIDList{
				Id: []string{
					chromeOSMachine1.GetId().GetValue(),
					chromeMachine1.GetId().GetValue(),
				},
			}
			resp, err := tf.Registration.DeleteMachines(tf.C, req)
			So(err, ShouldBeNil)
			So(resp.GetPassed(), ShouldHaveLength, 2)
			So(resp.GetFailed(), ShouldHaveLength, 0)
			So(resp.GetPassed()[0].Id, ShouldEqual, "chromeos-asset-1")
			So(resp.GetPassed()[1].Id, ShouldEqual, "chrome-asset-1")

			listReq := &api.ListMachinesRequest{}
			res, err := tf.Registration.ListMachines(tf.C, listReq)
			So(err, ShouldBeNil)
			So(res.GetPassed(), ShouldHaveLength, 0)
			So(res.GetFailed(), ShouldHaveLength, 0)
		})
		Convey("Delete machines by non-existing ID", func() {
			req := &api.EntityIDList{
				Id: []string{
					chromeMachine2.GetId().GetValue(),
				},
			}
			res, err := tf.Registration.DeleteMachines(tf.C, req)
			So(err, ShouldBeNil)
			So(res.GetPassed(), ShouldHaveLength, 0)
			So(res.GetFailed(), ShouldHaveLength, 1)
			So(res.GetFailed()[0].ErrorMsg, ShouldContainSubstring, "Entity not found")
		})
	})
}
