// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"testing"

	proto "infra/unifiedfleet/api/v1/proto"
	api "infra/unifiedfleet/api/v1/rpc"
	"infra/unifiedfleet/app/model/configuration"
	. "infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/model/inventory"
	"infra/unifiedfleet/app/model/registration"
	"infra/unifiedfleet/app/util"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
	code "google.golang.org/genproto/googleapis/rpc/code"
	"google.golang.org/grpc/status"
)

func mockChromeOSMachine(id, lab, board string) *proto.Machine {
	return &proto.Machine{
		Name: util.AddPrefix(util.MachineCollection, id),
		Device: &proto.Machine_ChromeosMachine{
			ChromeosMachine: &proto.ChromeOSMachine{
				ReferenceBoard: board,
			},
		},
	}
}

func mockChromeBrowserMachine(id, lab, name string) *proto.Machine {
	return &proto.Machine{
		Name: util.AddPrefix(util.MachineCollection, id),
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
		Name:       util.AddPrefix(util.RackCollection, id),
		CapacityRu: rackCapactiy,
	}
}

func mockNic(id string) *proto.Nic {
	return &proto.Nic{
		Name:       util.AddPrefix(util.NicCollection, id),
		MacAddress: "12:ab",
		SwitchInterface: &proto.SwitchInterface{
			Switch: "test-switch",
			Port:   1,
		},
	}
}

func mockKVM(id string) *proto.KVM {
	return &proto.KVM{
		Name: util.AddPrefix(util.KVMCollection, id),
	}
}

func mockRPM(id string) *proto.RPM {
	return &proto.RPM{
		Name: util.AddPrefix(util.RPMCollection, id),
	}
}

func mockDrac(id string) *proto.Drac {
	return &proto.Drac{
		Name:       util.AddPrefix(util.DracCollection, id),
		MacAddress: "12:ab",
		SwitchInterface: &proto.SwitchInterface{
			Switch: "test-switch",
			Port:   1,
		},
	}
}

func mockSwitch(id string) *proto.Switch {
	return &proto.Switch{
		Name: util.AddPrefix(util.SwitchCollection, id),
	}
}

func TestMachineRegistration(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	Convey("Machine Registration", t, func() {
		Convey("Register machine with nil machine", func() {
			req := &api.MachineRegistrationRequest{}
			_, err := tf.Fleet.MachineRegistration(tf.C, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Machine "+api.NilEntity)
		})

		Convey("Register machine - Invalid input empty machine name", func() {
			req := &api.MachineRegistrationRequest{
				Machine: &proto.Machine{
					Name: "",
				},
			}
			_, err := tf.Fleet.MachineRegistration(tf.C, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Machine "+api.EmptyName)
		})

		Convey("Create new machine - Invalid input invalid characters in machine name", func() {
			req := &api.MachineRegistrationRequest{
				Machine: &proto.Machine{
					Name: "a.b)7&",
				},
			}
			_, err := tf.Fleet.MachineRegistration(tf.C, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Machine "+api.InvalidCharacters)
		})

		Convey("Register machine - Invalid input empty nic name", func() {
			req := &api.MachineRegistrationRequest{
				Machine: &proto.Machine{
					Name: "machine-1",
				},
				Nics: []*proto.Nic{{
					Name: "",
				}},
			}
			_, err := tf.Fleet.MachineRegistration(tf.C, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Nic "+api.EmptyName)
		})

		Convey("Register machine - Invalid input invalid characters in nic name", func() {
			req := &api.MachineRegistrationRequest{
				Machine: &proto.Machine{
					Name: "machine-1",
				},
				Nics: []*proto.Nic{{
					Name: "a.b)7&",
				}},
			}
			_, err := tf.Fleet.MachineRegistration(tf.C, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Nic a.b)7& has invalid characters in the name."+api.InvalidCharacters)
		})

		Convey("Register machine - Invalid input empty drac name", func() {
			req := &api.MachineRegistrationRequest{
				Machine: &proto.Machine{
					Name: "machine-1",
				},
				Drac: &proto.Drac{
					Name: "",
				},
			}
			_, err := tf.Fleet.MachineRegistration(tf.C, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Drac "+api.EmptyName)
		})

		Convey("Register machine - Invalid input invalid characters in drac name", func() {
			req := &api.MachineRegistrationRequest{
				Machine: &proto.Machine{
					Name: "machine-1",
				},
				Drac: &proto.Drac{
					Name: "a.b)7&",
				},
			}
			_, err := tf.Fleet.MachineRegistration(tf.C, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Drac "+api.InvalidCharacters)
		})
	})
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
		Convey("Create new machine with machine_id - happy path", func() {
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
			So(err.Error(), ShouldContainSubstring, "Machine Chromeos-asset-1 already exists in the system.")
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
		Convey("Update existing machines - happy path", func() {
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
			So(err.Error(), ShouldContainSubstring, "There is no Machine with MachineID chrome-asset-1 in the system.")
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
				Name: util.AddPrefix(util.MachineCollection, "chromeos-asset-1"),
			}
			resp, err := tf.Fleet.GetMachine(tf.C, req)
			So(err, ShouldBeNil)
			assertMachineEqual(resp, chromeOSMachine1)
		})
		Convey("Get machine by non-existing ID", func() {
			req := &api.GetMachineRequest{
				Name: util.AddPrefix(util.MachineCollection, "chrome-asset-1"),
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
				Name: util.AddPrefix(util.MachineCollection, "a.b)7&"),
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
				Name: util.AddPrefix(util.MachineCollection, "chromeos-asset-1"),
			}
			_, err := tf.Fleet.DeleteMachine(tf.C, req)
			So(err, ShouldBeNil)
			greq := &api.GetMachineRequest{
				Name: util.AddPrefix(util.MachineCollection, "chromeos-asset-1"),
			}
			res, err := tf.Fleet.GetMachine(tf.C, greq)
			So(res, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Delete machine by non-existing ID", func() {
			req := &api.DeleteMachineRequest{
				Name: util.AddPrefix(util.MachineCollection, "chrome-asset-1"),
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
				Name: util.AddPrefix(util.MachineCollection, "a.b)7&"),
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
			machines, _, err := registration.ListMachines(ctx, 100, "")
			So(err, ShouldBeNil)
			So(machines, ShouldHaveLength, 3)
			So(api.ParseResources(machines, "Name"), ShouldResemble, []string{"machine1", "machine2", "machine3"})
			for _, m := range machines {
				So(m.GetRealm(), ShouldEqual, util.BrowserLabAdminRealm)
				bm := m.GetChromeBrowserMachine()
				switch m.GetName() {
				case "machine1":
					So(bm.GetNics(), ShouldResemble, []string{"machine1-eth0", "machine1-eth1"})
					So(bm.GetDrac(), ShouldEqual, "drac-hostname")
					So(bm.GetChromePlatform(), ShouldEqual, "fake_platform")
				case "machine2":
					So(bm.GetNics(), ShouldResemble, []string{"machine2-eth0"})
					So(bm.GetDrac(), ShouldEqual, "")
					So(bm.GetChromePlatform(), ShouldEqual, "fake_platform")
				case "machine3":
					So(bm.GetNics(), ShouldResemble, []string{"machine3-eth0"})
					So(bm.GetDrac(), ShouldEqual, "")
					So(bm.GetChromePlatform(), ShouldEqual, "fake_platform2")
				}
			}
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
	rack2 := mockRack("", 4)
	Convey("CreateRack", t, func() {
		Convey("Create new rack with rack_id - Happy path", func() {
			rack1 := mockRack("", 4)
			rack1.Location = &proto.Location{
				Lab: proto.Lab_LAB_CHROME_ATLANTA,
			}
			req := &api.CreateRackRequest{
				Rack:   rack1,
				RackId: "Rack-1",
			}
			resp, err := tf.Fleet.CreateRack(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, rack1)
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
	rack3 := mockRack("rack-3", 6)
	rack4 := mockRack("a.b)7&", 6)
	Convey("UpdateRack", t, func() {
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
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	Convey("GetRack", t, func() {
		Convey("Get rack by existing ID", func() {
			rack1 := &proto.Rack{
				Name: "rack-1",
			}
			_, err := registration.CreateRack(tf.C, rack1)
			So(err, ShouldBeNil)
			rack1.Name = util.AddPrefix(util.RackCollection, "rack-1")

			req := &api.GetRackRequest{
				Name: util.AddPrefix(util.RackCollection, "rack-1"),
			}
			resp, err := tf.Fleet.GetRack(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, rack1)
		})
		Convey("Get rack by non-existing ID", func() {
			req := &api.GetRackRequest{
				Name: util.AddPrefix(util.RackCollection, "rack-2"),
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
				Name: util.AddPrefix(util.RackCollection, "a.b)7&"),
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
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	racks := make([]*proto.Rack, 0, 4)
	for i := 0; i < 4; i++ {
		rack1 := &proto.Rack{
			Name: fmt.Sprintf("rack-%d", i),
		}
		resp, _ := registration.CreateRack(tf.C, rack1)
		rack1.Name = util.AddPrefix(util.RackCollection, rack1.Name)
		racks = append(racks, resp)
	}
	Convey("ListRacks", t, func() {
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
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	Convey("DeleteRack", t, func() {
		Convey("Delete rack by existing ID", func() {
			_, err := registration.CreateRack(tf.C, &proto.Rack{
				Name: "rack-1",
			})
			So(err, ShouldBeNil)

			req := &api.DeleteRackRequest{
				Name: util.AddPrefix(util.RackCollection, "rack-1"),
			}
			_, err = tf.Fleet.DeleteRack(tf.C, req)
			So(err, ShouldBeNil)

			res, err := registration.GetRack(tf.C, "rack-1")
			So(res, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Delete rack by non-existing ID", func() {
			req := &api.DeleteRackRequest{
				Name: util.AddPrefix(util.RackCollection, "rack-2"),
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
				Name: util.AddPrefix(util.RackCollection, "a.b)7&"),
			}
			resp, err := tf.Fleet.DeleteRack(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.InvalidCharacters)
		})
	})
}

func TestCreateNic(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	machine1 := &proto.Machine{
		Name: "machine-1",
		Device: &proto.Machine_ChromeBrowserMachine{
			ChromeBrowserMachine: &proto.ChromeBrowserMachine{},
		},
	}
	registration.CreateMachine(tf.C, machine1)
	registration.CreateSwitch(tf.C, &proto.Switch{
		Name:         "test-switch",
		CapacityPort: 100,
	})
	Convey("CreateNic", t, func() {
		Convey("Create new nic with nic_id", func() {
			nic := mockNic("")
			req := &api.CreateNicRequest{
				Nic:     nic,
				NicId:   "nic-1",
				Machine: "machine-1",
			}
			resp, err := tf.Fleet.CreateNic(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, nic)
		})

		Convey("Create existing nic", func() {
			nic := &proto.Nic{
				Name: "nic-3",
			}
			_, err := registration.CreateNic(tf.C, nic)
			So(err, ShouldBeNil)

			nic2 := mockNic("nic-3")
			req := &api.CreateNicRequest{
				Nic:     nic2,
				NicId:   "nic-3",
				Machine: "machine-1",
			}
			resp, err := tf.Fleet.CreateNic(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Nic nic-3 already exists in the system.")
		})

		Convey("Create new nic - Invalid input nil", func() {
			req := &api.CreateNicRequest{
				Nic: nil,
			}
			resp, err := tf.Fleet.CreateNic(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.NilEntity)
		})

		Convey("Create new nic - Invalid input empty ID", func() {
			nic := mockNic("")
			req := &api.CreateNicRequest{
				Nic:     nic,
				NicId:   "",
				Machine: "machine-1",
			}
			resp, err := tf.Fleet.CreateNic(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.EmptyID)
		})

		Convey("Create new nic - Invalid input invalid characters", func() {
			nic := mockNic("")
			req := &api.CreateNicRequest{
				Nic:     nic,
				NicId:   "a.b)7&",
				Machine: "machine-1",
			}
			resp, err := tf.Fleet.CreateNic(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.InvalidCharacters)
		})

		Convey("Create new nic - Invalid input empty machine", func() {
			nic := mockNic("")
			req := &api.CreateNicRequest{
				Nic:   nic,
				NicId: "nic-5",
			}
			resp, err := tf.Fleet.CreateNic(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.EmptyMachineName)
		})
	})
}

func TestUpdateNic(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	machine1 := &proto.Machine{
		Name: "machine-1",
		Device: &proto.Machine_ChromeBrowserMachine{
			ChromeBrowserMachine: &proto.ChromeBrowserMachine{
				Nics: []string{"nic-1"},
			},
		},
	}
	registration.CreateMachine(tf.C, machine1)
	Convey("UpdateNic", t, func() {
		Convey("Update existing nic", func() {
			nic1 := &proto.Nic{
				Name: "nic-1",
			}
			resp, err := registration.CreateNic(tf.C, nic1)
			So(err, ShouldBeNil)

			nic2 := mockNic("nic-1")
			nic2.SwitchInterface = nil
			ureq := &api.UpdateNicRequest{
				Nic:     nic2,
				Machine: "machine-1",
			}
			resp, err = tf.Fleet.UpdateNic(tf.C, ureq)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, nic2)
		})

		Convey("Update non-existing nic", func() {
			nic := mockNic("nic-3")
			ureq := &api.UpdateNicRequest{
				Nic:     nic,
				Machine: "machine-1",
			}
			resp, err := tf.Fleet.UpdateNic(tf.C, ureq)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "There is no Nic with NicID nic-3 in the system")
		})

		Convey("Update nic - Invalid input nil", func() {
			req := &api.UpdateNicRequest{
				Nic: nil,
			}
			resp, err := tf.Fleet.UpdateNic(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.NilEntity)
		})

		Convey("Update nic - Invalid input empty name", func() {
			nic := mockNic("")
			nic.Name = ""
			req := &api.UpdateNicRequest{
				Nic:     nic,
				Machine: "machine-1",
			}
			resp, err := tf.Fleet.UpdateNic(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.EmptyName)
		})

		Convey("Update nic - Invalid input invalid characters", func() {
			nic := mockNic("a.b)7&")
			req := &api.UpdateNicRequest{
				Nic:     nic,
				Machine: "machine-1",
			}
			resp, err := tf.Fleet.UpdateNic(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.InvalidCharacters)
		})
	})
}

func TestGetNic(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()

	Convey("GetNic", t, func() {
		Convey("Get nic by existing ID", func() {
			nic1 := &proto.Nic{
				Name: "nic-1",
			}
			registration.CreateNic(tf.C, nic1)
			nic1.Name = util.AddPrefix(util.NicCollection, "nic-1")

			req := &api.GetNicRequest{
				Name: util.AddPrefix(util.NicCollection, "nic-1"),
			}
			resp, err := tf.Fleet.GetNic(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, nic1)
		})
		Convey("Get nic by non-existing ID", func() {
			req := &api.GetNicRequest{
				Name: util.AddPrefix(util.NicCollection, "nic-2"),
			}
			resp, err := tf.Fleet.GetNic(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Get nic - Invalid input empty name", func() {
			req := &api.GetNicRequest{
				Name: "",
			}
			resp, err := tf.Fleet.GetNic(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.EmptyName)
		})
		Convey("Get nic - Invalid input invalid characters", func() {
			req := &api.GetNicRequest{
				Name: util.AddPrefix(util.NicCollection, "a.b)7&"),
			}
			resp, err := tf.Fleet.GetNic(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.InvalidCharacters)
		})
	})
}

func TestListNics(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	nics := make([]*proto.Nic, 0, 4)
	for i := 0; i < 4; i++ {
		nic := &proto.Nic{
			Name: fmt.Sprintf("nic-%d", i),
		}
		resp, _ := registration.CreateNic(tf.C, nic)
		nic.Name = util.AddPrefix(util.NicCollection, nic.Name)
		nics = append(nics, resp)
	}
	Convey("ListNics", t, func() {
		Convey("ListNics - page_size negative", func() {
			req := &api.ListNicsRequest{
				PageSize: -5,
			}
			resp, err := tf.Fleet.ListNics(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.InvalidPageSize)
		})

		Convey("ListNics - page_token invalid", func() {
			req := &api.ListNicsRequest{
				PageSize:  5,
				PageToken: "abc",
			}
			resp, err := tf.Fleet.ListNics(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InvalidPageToken)
		})

		Convey("ListNics - Full listing Max PageSize", func() {
			req := &api.ListNicsRequest{
				PageSize: 2000,
			}
			resp, err := tf.Fleet.ListNics(tf.C, req)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp.Nics, ShouldResembleProto, nics)
		})

		Convey("ListNics - Full listing with no pagination", func() {
			req := &api.ListNicsRequest{
				PageSize: 0,
			}
			resp, err := tf.Fleet.ListNics(tf.C, req)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp.Nics, ShouldResembleProto, nics)
		})

		Convey("ListNics - listing with pagination", func() {
			req := &api.ListNicsRequest{
				PageSize: 3,
			}
			resp, err := tf.Fleet.ListNics(tf.C, req)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp.Nics, ShouldResembleProto, nics[:3])

			req = &api.ListNicsRequest{
				PageSize:  3,
				PageToken: resp.NextPageToken,
			}
			resp, err = tf.Fleet.ListNics(tf.C, req)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp.Nics, ShouldResembleProto, nics[3:])
		})
	})
}

func TestDeleteNic(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	Convey("DeleteNic", t, func() {
		Convey("Delete nic - Invalid input empty name", func() {
			req := &api.DeleteNicRequest{
				Name: "",
			}
			resp, err := tf.Fleet.DeleteNic(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.EmptyName)
		})

		Convey("Delete nic - Invalid input invalid characters", func() {
			req := &api.DeleteNicRequest{
				Name: util.AddPrefix(util.NicCollection, "a.b)7&"),
			}
			resp, err := tf.Fleet.DeleteNic(tf.C, req)
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
			nics, _, err := registration.ListNics(ctx, 100, "")
			So(err, ShouldBeNil)
			So(api.ParseResources(nics, "Name"), ShouldResemble, []string{"machine1-eth0", "machine1-eth1", "machine2-eth0", "machine3-eth0"})
			switches := make([]*proto.SwitchInterface, len(nics))
			for i, nic := range nics {
				switches[i] = nic.GetSwitchInterface()
			}
			So(switches, ShouldResembleProto, []*proto.SwitchInterface{
				{
					Switch: "eq017.atl97",
					Port:   2,
				},
				{
					Switch: "eq017.atl97",
					Port:   3,
				},
				{
					Switch: "eq017.atl97",
					Port:   4,
				},
				{
					Switch: "eq041.atl97",
					Port:   1,
				},
			})
			dracs, _, err := registration.ListDracs(ctx, 100, "")
			So(err, ShouldBeNil)
			So(api.ParseResources(dracs, "Name"), ShouldResemble, []string{"drac-hostname"})
			So(api.ParseResources(dracs, "DisplayName"), ShouldResemble, []string{"machine1-drac"})
			dhcps, _, err := configuration.ListDHCPConfigs(ctx, 100, "")
			So(err, ShouldBeNil)
			So(api.ParseResources(dhcps, "Ip"), ShouldResemble, []string{"ip1.1", "ip1.2", "ip1.3", "ip2", "ip3"})
		})
		// Invalid & Empty machines DB hosts are tested in TestImportMachines
		// Skip testing here
	})
}

func TestImportDatacenters(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	Convey("Import datacenters", t, func() {
		Convey("happy path", func() {
			req := &api.ImportDatacentersRequest{
				Source: &api.ImportDatacentersRequest_ConfigSource{
					ConfigSource: &api.ConfigSource{
						ConfigServiceName: "fake-service",
						FileName:          "fakeDatacenter.cfg",
					},
				},
			}
			res, err := tf.Fleet.ImportDatacenters(ctx, req)
			So(err, ShouldBeNil)
			So(res.Code, ShouldEqual, code.Code_OK)
			dhcps, _, err := configuration.ListDHCPConfigs(ctx, 100, "")
			So(err, ShouldBeNil)
			So(dhcps, ShouldHaveLength, 3)
			racks, _, err := registration.ListRacks(ctx, 100, "")
			So(err, ShouldBeNil)
			So(racks, ShouldHaveLength, 2)
			So(api.ParseResources(racks, "Name"), ShouldResemble, []string{"cr20", "cr22"})
			for _, r := range racks {
				switch r.GetName() {
				case "cr20":
					So(r.GetChromeBrowserRack().GetKvms(), ShouldResemble, []string{"cr20-kvm1"})
					So(r.GetChromeBrowserRack().GetSwitches(), ShouldResemble, []string{"eq017.atl97"})
				}
			}
			kvms, _, err := registration.ListKVMs(ctx, 100, "")
			So(err, ShouldBeNil)
			So(kvms, ShouldHaveLength, 3)
			So(api.ParseResources(kvms, "Name"), ShouldResemble, []string{"cr20-kvm1", "cr22-kvm1", "cr22-kvm2"})
			So(api.ParseResources(kvms, "ChromePlatform"), ShouldResemble, []string{"Raritan_DKX3", "Raritan_DKX3", "Raritan_DKX3"})
			switches, _, err := registration.ListSwitches(ctx, 100, "")
			So(err, ShouldBeNil)
			So(switches, ShouldHaveLength, 4)
			So(api.ParseResources(switches, "Name"), ShouldResemble, []string{"eq017.atl97", "eq041.atl97", "eq050.atl97", "eq113.atl97"})
			rackLSEs, _, err := inventory.ListRackLSEs(ctx, 100, "")
			So(err, ShouldBeNil)
			So(rackLSEs, ShouldHaveLength, 2)
			rlse, err := inventory.QueryRackLSEByPropertyName(ctx, "rack_ids", "cr20", false)
			So(err, ShouldBeNil)
			So(rlse, ShouldHaveLength, 1)
			So(rlse[0].GetRackLsePrototype(), ShouldEqual, "browser-lab:normal")
			So(rlse[0].GetChromeBrowserRackLse().GetKvms(), ShouldResemble, []string{"cr20-kvm1"})
			So(rlse[0].GetChromeBrowserRackLse().GetSwitches(), ShouldResemble, []string{"eq017.atl97"})
		})
	})
}
func TestCreateKVM(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	rack1 := &proto.Rack{
		Name: "rack-1",
		Rack: &proto.Rack_ChromeBrowserRack{
			ChromeBrowserRack: &proto.ChromeBrowserRack{},
		},
	}
	registration.CreateRack(tf.C, rack1)
	Convey("CreateKVM", t, func() {
		Convey("Create new KVM with KVM_id", func() {
			KVM1 := mockKVM("")
			req := &api.CreateKVMRequest{
				KVM:   KVM1,
				KVMId: "KVM-1",
				Rack:  "rack-1",
			}
			resp, err := tf.Fleet.CreateKVM(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, KVM1)
		})

		Convey("Create new KVM - Invalid input nil", func() {
			req := &api.CreateKVMRequest{
				KVM:  nil,
				Rack: "rack-1",
			}
			resp, err := tf.Fleet.CreateKVM(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.NilEntity)
		})

		Convey("Create new KVM - Invalid input empty ID", func() {
			req := &api.CreateKVMRequest{
				KVM:   mockKVM(""),
				KVMId: "",
				Rack:  "rack-1",
			}
			resp, err := tf.Fleet.CreateKVM(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.EmptyID)
		})

		Convey("Create new KVM - Invalid input invalid characters", func() {
			req := &api.CreateKVMRequest{
				KVM:   mockKVM(""),
				KVMId: "a.b)7&",
				Rack:  "rack-1",
			}
			resp, err := tf.Fleet.CreateKVM(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.InvalidCharacters)
		})

		Convey("Create new kvm - Invalid input empty rack", func() {
			req := &api.CreateKVMRequest{
				KVM:   mockKVM("x"),
				KVMId: "kvm-5",
			}
			resp, err := tf.Fleet.CreateKVM(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.EmptyRackName)
		})
	})
}

func TestUpdateKVM(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	rack1 := &proto.Rack{
		Name: "rack-1",
		Rack: &proto.Rack_ChromeBrowserRack{
			ChromeBrowserRack: &proto.ChromeBrowserRack{
				Kvms: []string{"KVM-1"},
			},
		},
	}
	registration.CreateRack(tf.C, rack1)
	Convey("UpdateKVM", t, func() {
		Convey("Update existing KVM", func() {
			KVM1 := &proto.KVM{
				Name: "KVM-1",
			}
			resp, err := registration.CreateKVM(tf.C, KVM1)
			So(err, ShouldBeNil)

			KVM2 := mockKVM("KVM-1")
			ureq := &api.UpdateKVMRequest{
				KVM:  KVM2,
				Rack: "rack-1",
			}
			resp, err = tf.Fleet.UpdateKVM(tf.C, ureq)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, KVM2)
		})

		Convey("Update KVM - Invalid input nil", func() {
			req := &api.UpdateKVMRequest{
				KVM:  nil,
				Rack: "rack-1",
			}
			resp, err := tf.Fleet.UpdateKVM(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.NilEntity)
		})

		Convey("Update KVM - Invalid input empty name", func() {
			KVM3 := mockKVM("KVM-3")
			KVM3.Name = ""
			req := &api.UpdateKVMRequest{
				KVM:  KVM3,
				Rack: "rack-1",
			}
			resp, err := tf.Fleet.UpdateKVM(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.EmptyName)
		})

		Convey("Update KVM - Invalid input invalid characters", func() {
			req := &api.UpdateKVMRequest{
				KVM: mockKVM("a.b)7&"),
			}
			resp, err := tf.Fleet.UpdateKVM(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.InvalidCharacters)
		})
	})
}

func TestGetKVM(t *testing.T) {
	t.Parallel()
	Convey("GetKVM", t, func() {
		ctx := testingContext()
		tf, validate := newTestFixtureWithContext(ctx, t)
		defer validate()
		Convey("Get KVM by existing ID", func() {
			KVM1 := &proto.KVM{
				Name: "KVM-1",
			}
			_, err := registration.CreateKVM(tf.C, KVM1)
			So(err, ShouldBeNil)
			KVM1.Name = util.AddPrefix(util.KVMCollection, "KVM-1")

			req := &api.GetKVMRequest{
				Name: util.AddPrefix(util.KVMCollection, "KVM-1"),
			}
			resp, err := tf.Fleet.GetKVM(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, KVM1)
		})
		Convey("Get KVM by non-existing ID", func() {
			req := &api.GetKVMRequest{
				Name: util.AddPrefix(util.KVMCollection, "KVM-2"),
			}
			resp, err := tf.Fleet.GetKVM(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Get KVM - Invalid input empty name", func() {
			req := &api.GetKVMRequest{
				Name: "",
			}
			resp, err := tf.Fleet.GetKVM(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.EmptyName)
		})
		Convey("Get KVM - Invalid input invalid characters", func() {
			req := &api.GetKVMRequest{
				Name: util.AddPrefix(util.KVMCollection, "a.b)7&"),
			}
			resp, err := tf.Fleet.GetKVM(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.InvalidCharacters)
		})
	})
}

func TestListKVMs(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	KVMs := make([]*proto.KVM, 0, 4)
	for i := 0; i < 4; i++ {
		kvm := &proto.KVM{
			Name: fmt.Sprintf("kvm-%d", i),
		}
		resp, _ := registration.CreateKVM(tf.C, kvm)
		kvm.Name = util.AddPrefix(util.KVMCollection, kvm.Name)
		KVMs = append(KVMs, resp)
	}
	Convey("ListKVMs", t, func() {
		Convey("ListKVMs - page_size negative", func() {
			req := &api.ListKVMsRequest{
				PageSize: -5,
			}
			resp, err := tf.Fleet.ListKVMs(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.InvalidPageSize)
		})

		Convey("ListKVMs - page_token invalid", func() {
			req := &api.ListKVMsRequest{
				PageSize:  5,
				PageToken: "abc",
			}
			resp, err := tf.Fleet.ListKVMs(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InvalidPageToken)
		})

		Convey("ListKVMs - Full listing Max PageSize", func() {
			req := &api.ListKVMsRequest{
				PageSize: 2000,
			}
			resp, err := tf.Fleet.ListKVMs(tf.C, req)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp.KVMs, ShouldResembleProto, KVMs)
		})

		Convey("ListKVMs - Full listing with no pagination", func() {
			req := &api.ListKVMsRequest{
				PageSize: 0,
			}
			resp, err := tf.Fleet.ListKVMs(tf.C, req)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp.KVMs, ShouldResembleProto, KVMs)
		})

		Convey("ListKVMs - listing with pagination", func() {
			req := &api.ListKVMsRequest{
				PageSize: 3,
			}
			resp, err := tf.Fleet.ListKVMs(tf.C, req)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp.KVMs, ShouldResembleProto, KVMs[:3])

			req = &api.ListKVMsRequest{
				PageSize:  3,
				PageToken: resp.NextPageToken,
			}
			resp, err = tf.Fleet.ListKVMs(tf.C, req)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp.KVMs, ShouldResembleProto, KVMs[3:])
		})
	})
}

func TestDeleteKVM(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	Convey("DeleteKVM", t, func() {
		Convey("Delete KVM by existing ID without references", func() {
			KVM2 := &proto.KVM{
				Name: "KVM-2",
			}
			_, err := registration.CreateKVM(tf.C, KVM2)
			So(err, ShouldBeNil)

			dreq := &api.DeleteKVMRequest{
				Name: util.AddPrefix(util.KVMCollection, "KVM-2"),
			}
			_, err = tf.Fleet.DeleteKVM(tf.C, dreq)
			So(err, ShouldBeNil)

			_, err = registration.GetKVM(tf.C, "KVM-2")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})

		Convey("Delete KVM - Invalid input empty name", func() {
			req := &api.DeleteKVMRequest{
				Name: "",
			}
			resp, err := tf.Fleet.DeleteKVM(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.EmptyName)
		})

		Convey("Delete KVM - Invalid input invalid characters", func() {
			req := &api.DeleteKVMRequest{
				Name: util.AddPrefix(util.KVMCollection, "a.b)7&"),
			}
			resp, err := tf.Fleet.DeleteKVM(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.InvalidCharacters)
		})
	})
}

func TestCreateRPM(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	RPM1 := mockRPM("")
	RPM2 := mockRPM("")
	RPM3 := mockRPM("")
	Convey("CreateRPM", t, func() {
		Convey("Create new RPM with RPM_id", func() {
			req := &api.CreateRPMRequest{
				RPM:   RPM1,
				RPMId: "RPM-1",
			}
			resp, err := tf.Fleet.CreateRPM(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, RPM1)
		})

		Convey("Create existing RPM", func() {
			req := &api.CreateRPMRequest{
				RPM:   RPM3,
				RPMId: "RPM-1",
			}
			resp, err := tf.Fleet.CreateRPM(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, AlreadyExists)
		})

		Convey("Create new RPM - Invalid input nil", func() {
			req := &api.CreateRPMRequest{
				RPM: nil,
			}
			resp, err := tf.Fleet.CreateRPM(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.NilEntity)
		})

		Convey("Create new RPM - Invalid input empty ID", func() {
			req := &api.CreateRPMRequest{
				RPM:   RPM2,
				RPMId: "",
			}
			resp, err := tf.Fleet.CreateRPM(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.EmptyID)
		})

		Convey("Create new RPM - Invalid input invalid characters", func() {
			req := &api.CreateRPMRequest{
				RPM:   RPM2,
				RPMId: "a.b)7&",
			}
			resp, err := tf.Fleet.CreateRPM(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.InvalidCharacters)
		})
	})
}

func TestUpdateRPM(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	RPM1 := mockRPM("")
	RPM2 := mockRPM("RPM-1")
	RPM3 := mockRPM("RPM-3")
	RPM4 := mockRPM("a.b)7&")
	Convey("UpdateRPM", t, func() {
		Convey("Update existing RPM", func() {
			req := &api.CreateRPMRequest{
				RPM:   RPM1,
				RPMId: "RPM-1",
			}
			resp, err := tf.Fleet.CreateRPM(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, RPM1)
			ureq := &api.UpdateRPMRequest{
				RPM: RPM2,
			}
			resp, err = tf.Fleet.UpdateRPM(tf.C, ureq)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, RPM2)
		})

		Convey("Update non-existing RPM", func() {
			ureq := &api.UpdateRPMRequest{
				RPM: RPM3,
			}
			resp, err := tf.Fleet.UpdateRPM(tf.C, ureq)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})

		Convey("Update RPM - Invalid input nil", func() {
			req := &api.UpdateRPMRequest{
				RPM: nil,
			}
			resp, err := tf.Fleet.UpdateRPM(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.NilEntity)
		})

		Convey("Update RPM - Invalid input empty name", func() {
			RPM3.Name = ""
			req := &api.UpdateRPMRequest{
				RPM: RPM3,
			}
			resp, err := tf.Fleet.UpdateRPM(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.EmptyName)
		})

		Convey("Update RPM - Invalid input invalid characters", func() {
			req := &api.UpdateRPMRequest{
				RPM: RPM4,
			}
			resp, err := tf.Fleet.UpdateRPM(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.InvalidCharacters)
		})
	})
}

func TestGetRPM(t *testing.T) {
	t.Parallel()
	Convey("GetRPM", t, func() {
		ctx := testingContext()
		tf, validate := newTestFixtureWithContext(ctx, t)
		defer validate()
		RPM1 := mockRPM("RPM-1")
		req := &api.CreateRPMRequest{
			RPM:   RPM1,
			RPMId: "RPM-1",
		}
		resp, err := tf.Fleet.CreateRPM(tf.C, req)
		So(err, ShouldBeNil)
		So(resp, ShouldResembleProto, RPM1)
		Convey("Get RPM by existing ID", func() {
			req := &api.GetRPMRequest{
				Name: util.AddPrefix(util.RPMCollection, "RPM-1"),
			}
			resp, err := tf.Fleet.GetRPM(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, RPM1)
		})
		Convey("Get RPM by non-existing ID", func() {
			req := &api.GetRPMRequest{
				Name: util.AddPrefix(util.RPMCollection, "RPM-2"),
			}
			resp, err := tf.Fleet.GetRPM(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Get RPM - Invalid input empty name", func() {
			req := &api.GetRPMRequest{
				Name: "",
			}
			resp, err := tf.Fleet.GetRPM(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.EmptyName)
		})
		Convey("Get RPM - Invalid input invalid characters", func() {
			req := &api.GetRPMRequest{
				Name: util.AddPrefix(util.RPMCollection, "a.b)7&"),
			}
			resp, err := tf.Fleet.GetRPM(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.InvalidCharacters)
		})
	})
}

func TestListRPMs(t *testing.T) {
	t.Parallel()
	Convey("ListRPMs", t, func() {
		ctx := testingContext()
		tf, validate := newTestFixtureWithContext(ctx, t)
		defer validate()
		RPMs := make([]*proto.RPM, 0, 4)
		for i := 0; i < 4; i++ {
			RPM1 := mockRPM("")
			req := &api.CreateRPMRequest{
				RPM:   RPM1,
				RPMId: fmt.Sprintf("RPM-%d", i),
			}
			resp, err := tf.Fleet.CreateRPM(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, RPM1)
			RPMs = append(RPMs, resp)
		}

		Convey("ListRPMs - page_size negative", func() {
			req := &api.ListRPMsRequest{
				PageSize: -5,
			}
			resp, err := tf.Fleet.ListRPMs(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.InvalidPageSize)
		})

		Convey("ListRPMs - page_token invalid", func() {
			req := &api.ListRPMsRequest{
				PageSize:  5,
				PageToken: "abc",
			}
			resp, err := tf.Fleet.ListRPMs(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InvalidPageToken)
		})

		Convey("ListRPMs - Full listing Max PageSize", func() {
			req := &api.ListRPMsRequest{
				PageSize: 2000,
			}
			resp, err := tf.Fleet.ListRPMs(tf.C, req)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp.RPMs, ShouldResembleProto, RPMs)
		})

		Convey("ListRPMs - Full listing with no pagination", func() {
			req := &api.ListRPMsRequest{
				PageSize: 0,
			}
			resp, err := tf.Fleet.ListRPMs(tf.C, req)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp.RPMs, ShouldResembleProto, RPMs)
		})

		Convey("ListRPMs - listing with pagination", func() {
			req := &api.ListRPMsRequest{
				PageSize: 3,
			}
			resp, err := tf.Fleet.ListRPMs(tf.C, req)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp.RPMs, ShouldResembleProto, RPMs[:3])

			req = &api.ListRPMsRequest{
				PageSize:  3,
				PageToken: resp.NextPageToken,
			}
			resp, err = tf.Fleet.ListRPMs(tf.C, req)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp.RPMs, ShouldResembleProto, RPMs[3:])
		})
	})
}

func TestDeleteRPM(t *testing.T) {
	t.Parallel()
	Convey("DeleteRPM", t, func() {
		ctx := testingContext()
		tf, validate := newTestFixtureWithContext(ctx, t)
		defer validate()
		Convey("Delete RPM by existing ID", func() {
			RPM2 := mockRPM("")
			req := &api.CreateRPMRequest{
				RPM:   RPM2,
				RPMId: "RPM-2",
			}
			resp, err := tf.Fleet.CreateRPM(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, RPM2)

			dreq := &api.DeleteRPMRequest{
				Name: util.AddPrefix(util.RPMCollection, "RPM-2"),
			}
			_, err = tf.Fleet.DeleteRPM(tf.C, dreq)
			So(err, ShouldBeNil)

			greq := &api.GetRPMRequest{
				Name: util.AddPrefix(util.RPMCollection, "RPM-2"),
			}
			res, err := tf.Fleet.GetRPM(tf.C, greq)
			So(res, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})

		Convey("Delete RPM by non-existing ID", func() {
			req := &api.DeleteRPMRequest{
				Name: util.AddPrefix(util.RPMCollection, "RPM-2"),
			}
			_, err := tf.Fleet.DeleteRPM(tf.C, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})

		Convey("Delete RPM - Invalid input empty name", func() {
			req := &api.DeleteRPMRequest{
				Name: "",
			}
			resp, err := tf.Fleet.DeleteRPM(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.EmptyName)
		})

		Convey("Delete RPM - Invalid input invalid characters", func() {
			req := &api.DeleteRPMRequest{
				Name: util.AddPrefix(util.RPMCollection, "a.b)7&"),
			}
			resp, err := tf.Fleet.DeleteRPM(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.InvalidCharacters)
		})
	})
}

func TestCreateDrac(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	machine1 := &proto.Machine{
		Name: "machine-1",
		Device: &proto.Machine_ChromeBrowserMachine{
			ChromeBrowserMachine: &proto.ChromeBrowserMachine{},
		},
	}
	registration.CreateMachine(tf.C, machine1)
	registration.CreateSwitch(tf.C, &proto.Switch{
		Name:         "test-switch",
		CapacityPort: 100,
	})
	Convey("CreateDrac", t, func() {
		Convey("Create new drac with drac_id", func() {
			drac := mockDrac("")
			req := &api.CreateDracRequest{
				Drac:    drac,
				DracId:  "Drac-1",
				Machine: "machine-1",
			}
			resp, err := tf.Fleet.CreateDrac(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, drac)
		})

		Convey("Create existing drac", func() {
			drac := &proto.Drac{
				Name: "Drac-3",
			}
			_, err := registration.CreateDrac(tf.C, drac)
			So(err, ShouldBeNil)

			req := &api.CreateDracRequest{
				Drac:    mockDrac("Drac-3"),
				DracId:  "Drac-3",
				Machine: "machine-1",
			}
			resp, err := tf.Fleet.CreateDrac(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Drac Drac-3 already exists in the system.")
		})

		Convey("Create new drac - Invalid input nil", func() {
			req := &api.CreateDracRequest{
				Drac: nil,
			}
			resp, err := tf.Fleet.CreateDrac(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.NilEntity)
		})

		Convey("Create new drac - Invalid input empty ID", func() {
			drac := mockDrac("")
			req := &api.CreateDracRequest{
				Drac:    drac,
				DracId:  "",
				Machine: "machine-1",
			}
			resp, err := tf.Fleet.CreateDrac(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.EmptyID)
		})

		Convey("Create new drac - Invalid input invalid characters", func() {
			drac := mockDrac("")
			req := &api.CreateDracRequest{
				Drac:    drac,
				DracId:  "a.b)7&",
				Machine: "machine-1",
			}
			resp, err := tf.Fleet.CreateDrac(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.InvalidCharacters)
		})

		Convey("Create new drac - Invalid input empty machine", func() {
			drac := mockDrac("")
			req := &api.CreateDracRequest{
				Drac:   drac,
				DracId: "drac-5",
			}
			resp, err := tf.Fleet.CreateDrac(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.EmptyMachineName)
		})
	})
}

func TestUpdateDrac(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	machine1 := &proto.Machine{
		Name: "machine-1",
		Device: &proto.Machine_ChromeBrowserMachine{
			ChromeBrowserMachine: &proto.ChromeBrowserMachine{
				Drac: "drac-1",
			},
		},
	}
	registration.CreateMachine(tf.C, machine1)
	Convey("UpdateDrac", t, func() {
		Convey("Update existing drac", func() {
			drac1 := &proto.Drac{
				Name: "drac-1",
			}
			resp, err := registration.CreateDrac(tf.C, drac1)
			So(err, ShouldBeNil)

			drac2 := mockDrac("drac-1")
			drac2.SwitchInterface = nil
			ureq := &api.UpdateDracRequest{
				Drac:    drac2,
				Machine: "machine-1",
			}
			resp, err = tf.Fleet.UpdateDrac(tf.C, ureq)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, drac2)
		})

		Convey("Update non-existing drac", func() {
			drac := mockDrac("drac-3")
			ureq := &api.UpdateDracRequest{
				Drac:    drac,
				Machine: "machine-1",
			}
			resp, err := tf.Fleet.UpdateDrac(tf.C, ureq)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "There is no Drac with DracID drac-3 in the system")
		})

		Convey("Update drac - Invalid input nil", func() {
			req := &api.UpdateDracRequest{
				Drac: nil,
			}
			resp, err := tf.Fleet.UpdateDrac(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.NilEntity)
		})

		Convey("Update drac - Invalid input empty name", func() {
			drac := mockDrac("")
			drac.Name = ""
			req := &api.UpdateDracRequest{
				Drac:    drac,
				Machine: "machine-1",
			}
			resp, err := tf.Fleet.UpdateDrac(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.EmptyName)
		})

		Convey("Update drac - Invalid input invalid characters", func() {
			drac := mockDrac("a.b)7&")
			req := &api.UpdateDracRequest{
				Drac:    drac,
				Machine: "machine-1",
			}
			resp, err := tf.Fleet.UpdateDrac(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.InvalidCharacters)
		})
	})
}

func TestGetDrac(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()

	Convey("GetDrac", t, func() {
		Convey("Get drac by existing ID", func() {
			drac1 := &proto.Drac{
				Name: "drac-1",
			}
			registration.CreateDrac(tf.C, drac1)
			drac1.Name = util.AddPrefix(util.DracCollection, "drac-1")

			req := &api.GetDracRequest{
				Name: util.AddPrefix(util.DracCollection, "drac-1"),
			}
			resp, err := tf.Fleet.GetDrac(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, drac1)
		})
		Convey("Get drac by non-existing ID", func() {
			req := &api.GetDracRequest{
				Name: util.AddPrefix(util.DracCollection, "drac-2"),
			}
			resp, err := tf.Fleet.GetDrac(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Get drac - Invalid input empty name", func() {
			req := &api.GetDracRequest{
				Name: "",
			}
			resp, err := tf.Fleet.GetDrac(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.EmptyName)
		})
		Convey("Get drac - Invalid input invalid characters", func() {
			req := &api.GetDracRequest{
				Name: util.AddPrefix(util.DracCollection, "a.b)7&"),
			}
			resp, err := tf.Fleet.GetDrac(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.InvalidCharacters)
		})
	})
}

func TestListDracs(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	dracs := make([]*proto.Drac, 0, 4)
	for i := 0; i < 4; i++ {
		drac := &proto.Drac{
			Name: fmt.Sprintf("drac-%d", i),
		}
		resp, _ := registration.CreateDrac(tf.C, drac)
		drac.Name = util.AddPrefix(util.DracCollection, drac.Name)
		dracs = append(dracs, resp)
	}
	Convey("ListDracs", t, func() {
		Convey("ListDracs - page_size negative", func() {
			req := &api.ListDracsRequest{
				PageSize: -5,
			}
			resp, err := tf.Fleet.ListDracs(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.InvalidPageSize)
		})

		Convey("ListDracs - page_token invalid", func() {
			req := &api.ListDracsRequest{
				PageSize:  5,
				PageToken: "abc",
			}
			resp, err := tf.Fleet.ListDracs(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InvalidPageToken)
		})

		Convey("ListDracs - Full listing Max PageSize", func() {
			req := &api.ListDracsRequest{
				PageSize: 2000,
			}
			resp, err := tf.Fleet.ListDracs(tf.C, req)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp.Dracs, ShouldResembleProto, dracs)
		})

		Convey("ListDracs - Full listing with no pagination", func() {
			req := &api.ListDracsRequest{
				PageSize: 0,
			}
			resp, err := tf.Fleet.ListDracs(tf.C, req)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp.Dracs, ShouldResembleProto, dracs)
		})

		Convey("ListDracs - listing with pagination", func() {
			req := &api.ListDracsRequest{
				PageSize: 3,
			}
			resp, err := tf.Fleet.ListDracs(tf.C, req)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp.Dracs, ShouldResembleProto, dracs[:3])

			req = &api.ListDracsRequest{
				PageSize:  3,
				PageToken: resp.NextPageToken,
			}
			resp, err = tf.Fleet.ListDracs(tf.C, req)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp.Dracs, ShouldResembleProto, dracs[3:])
		})
	})
}

func TestDeleteDrac(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	Convey("DeleteDrac", t, func() {
		Convey("Delete drac - Invalid input empty name", func() {
			req := &api.DeleteDracRequest{
				Name: "",
			}
			resp, err := tf.Fleet.DeleteDrac(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.EmptyName)
		})

		Convey("Delete drac - Invalid input invalid characters", func() {
			req := &api.DeleteDracRequest{
				Name: util.AddPrefix(util.DracCollection, "a.b)7&"),
			}
			resp, err := tf.Fleet.DeleteDrac(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.InvalidCharacters)
		})
	})
}

func TestCreateSwitch(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	rack1 := &proto.Rack{
		Name: "rack-1",
		Rack: &proto.Rack_ChromeBrowserRack{
			ChromeBrowserRack: &proto.ChromeBrowserRack{},
		},
	}
	registration.CreateRack(tf.C, rack1)
	Convey("CreateSwitch", t, func() {
		Convey("Create new switch with switch_id", func() {
			switch1 := mockSwitch("")
			req := &api.CreateSwitchRequest{
				Switch:   switch1,
				SwitchId: "Switch-1",
				Rack:     "rack-1",
			}
			resp, err := tf.Fleet.CreateSwitch(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, switch1)
		})

		Convey("Create new switch - Invalid input nil", func() {
			req := &api.CreateSwitchRequest{
				Switch: nil,
				Rack:   "rack-1",
			}
			resp, err := tf.Fleet.CreateSwitch(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.NilEntity)
		})

		Convey("Create new switch - Invalid input empty ID", func() {
			req := &api.CreateSwitchRequest{
				Switch:   mockSwitch(""),
				SwitchId: "",
				Rack:     "rack-1",
			}
			resp, err := tf.Fleet.CreateSwitch(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.EmptyID)
		})

		Convey("Create new switch - Invalid input invalid characters", func() {
			req := &api.CreateSwitchRequest{
				Switch:   mockSwitch(""),
				SwitchId: "a.b)7&",
				Rack:     "rack-1",
			}
			resp, err := tf.Fleet.CreateSwitch(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.InvalidCharacters)
		})

		Convey("Create new switch - Invalid input empty rack", func() {
			req := &api.CreateSwitchRequest{
				Switch:   mockSwitch("x"),
				SwitchId: "switch-5",
			}
			resp, err := tf.Fleet.CreateSwitch(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.EmptyRackName)
		})
	})
}

func TestUpdateSwitch(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	rack1 := &proto.Rack{
		Name: "rack-1",
		Rack: &proto.Rack_ChromeBrowserRack{
			ChromeBrowserRack: &proto.ChromeBrowserRack{
				Switches: []string{"switch-1"},
			},
		},
	}
	registration.CreateRack(tf.C, rack1)
	Convey("UpdateSwitch", t, func() {
		Convey("Update existing switch", func() {
			switch1 := &proto.Switch{
				Name: "switch-1",
			}
			resp, err := registration.CreateSwitch(tf.C, switch1)
			So(err, ShouldBeNil)

			switch2 := mockSwitch("switch-1")
			ureq := &api.UpdateSwitchRequest{
				Switch: switch2,
				Rack:   "rack-1",
			}
			resp, err = tf.Fleet.UpdateSwitch(tf.C, ureq)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, switch2)
		})

		Convey("Update switch - Invalid input nil", func() {
			req := &api.UpdateSwitchRequest{
				Switch: nil,
				Rack:   "rack-1",
			}
			resp, err := tf.Fleet.UpdateSwitch(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.NilEntity)
		})

		Convey("Update switch - Invalid input empty name", func() {
			switch1 := mockSwitch("")
			switch1.Name = ""
			req := &api.UpdateSwitchRequest{
				Switch: switch1,
				Rack:   "rack-1",
			}
			resp, err := tf.Fleet.UpdateSwitch(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.EmptyName)
		})

		Convey("Update switch - Invalid input invalid characters", func() {
			req := &api.UpdateSwitchRequest{
				Switch: mockSwitch("a.b)7&"),
			}
			resp, err := tf.Fleet.UpdateSwitch(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.InvalidCharacters)
		})
	})
}

func TestGetSwitch(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	Convey("GetSwitch", t, func() {
		Convey("Get switch by existing ID", func() {
			switch1 := &proto.Switch{
				Name: "switch-1",
			}
			_, err := registration.CreateSwitch(tf.C, switch1)
			So(err, ShouldBeNil)
			switch1.Name = util.AddPrefix(util.SwitchCollection, "switch-1")

			req := &api.GetSwitchRequest{
				Name: util.AddPrefix(util.SwitchCollection, "switch-1"),
			}
			resp, err := tf.Fleet.GetSwitch(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, switch1)
		})
		Convey("Get switch by non-existing ID", func() {
			req := &api.GetSwitchRequest{
				Name: util.AddPrefix(util.SwitchCollection, "switch-2"),
			}
			resp, err := tf.Fleet.GetSwitch(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Get switch - Invalid input empty name", func() {
			req := &api.GetSwitchRequest{
				Name: "",
			}
			resp, err := tf.Fleet.GetSwitch(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.EmptyName)
		})
		Convey("Get switch - Invalid input invalid characters", func() {
			req := &api.GetSwitchRequest{
				Name: util.AddPrefix(util.SwitchCollection, "a.b)7&"),
			}
			resp, err := tf.Fleet.GetSwitch(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.InvalidCharacters)
		})
	})
}

func TestListSwitches(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	switches := make([]*proto.Switch, 0, 4)
	for i := 0; i < 4; i++ {
		s := &proto.Switch{
			Name: fmt.Sprintf("switch-%d", i),
		}
		resp, _ := registration.CreateSwitch(tf.C, s)
		s.Name = util.AddPrefix(util.SwitchCollection, s.Name)
		switches = append(switches, resp)
	}
	Convey("ListSwitches", t, func() {
		Convey("ListSwitches - page_size negative", func() {
			req := &api.ListSwitchesRequest{
				PageSize: -5,
			}
			resp, err := tf.Fleet.ListSwitches(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.InvalidPageSize)
		})

		Convey("ListSwitches - page_token invalid", func() {
			req := &api.ListSwitchesRequest{
				PageSize:  5,
				PageToken: "abc",
			}
			resp, err := tf.Fleet.ListSwitches(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InvalidPageToken)
		})

		Convey("ListSwitches - Full listing Max PageSize", func() {
			req := &api.ListSwitchesRequest{
				PageSize: 2000,
			}
			resp, err := tf.Fleet.ListSwitches(tf.C, req)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp.Switches, ShouldResembleProto, switches)
		})

		Convey("ListSwitches - Full listing with no pagination", func() {
			req := &api.ListSwitchesRequest{
				PageSize: 0,
			}
			resp, err := tf.Fleet.ListSwitches(tf.C, req)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp.Switches, ShouldResembleProto, switches)
		})

		Convey("ListSwitches - listing with pagination", func() {
			req := &api.ListSwitchesRequest{
				PageSize: 3,
			}
			resp, err := tf.Fleet.ListSwitches(tf.C, req)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp.Switches, ShouldResembleProto, switches[:3])

			req = &api.ListSwitchesRequest{
				PageSize:  3,
				PageToken: resp.NextPageToken,
			}
			resp, err = tf.Fleet.ListSwitches(tf.C, req)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp.Switches, ShouldResembleProto, switches[3:])
		})
	})
}

func TestDeleteSwitch(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	Convey("DeleteSwitch", t, func() {
		Convey("Delete switch by existing ID without references", func() {
			switch2 := &proto.Switch{
				Name: "switch-2",
			}
			_, err := registration.CreateSwitch(tf.C, switch2)
			So(err, ShouldBeNil)

			dreq := &api.DeleteSwitchRequest{
				Name: util.AddPrefix(util.SwitchCollection, "switch-2"),
			}
			_, err = tf.Fleet.DeleteSwitch(tf.C, dreq)
			So(err, ShouldBeNil)

			_, err = registration.GetSwitch(tf.C, "switch-2")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})

		Convey("Delete switch - Invalid input empty name", func() {
			req := &api.DeleteSwitchRequest{
				Name: "",
			}
			resp, err := tf.Fleet.DeleteSwitch(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.EmptyName)
		})

		Convey("Delete switch - Invalid input invalid characters", func() {
			req := &api.DeleteSwitchRequest{
				Name: util.AddPrefix(util.SwitchCollection, "a.b)7&"),
			}
			resp, err := tf.Fleet.DeleteSwitch(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.InvalidCharacters)
		})
	})
}

func getCapacity(cidr string) float64 {
	cover := strings.Split(cidr, "/")[1]
	coverN, err := strconv.Atoi(cover)
	if err != nil {
		return 0
	}
	return math.Exp2(32 - float64(coverN))
}
