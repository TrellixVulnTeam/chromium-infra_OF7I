// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
	code "google.golang.org/genproto/googleapis/rpc/code"
	"google.golang.org/grpc/status"

	ufspb "infra/unifiedfleet/api/v1/proto"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	"infra/unifiedfleet/app/model/configuration"
	. "infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/model/inventory"
	"infra/unifiedfleet/app/model/registration"
	"infra/unifiedfleet/app/util"
)

func mockChromeOSMachine(id, lab, board string) *ufspb.Machine {
	return &ufspb.Machine{
		Name: util.AddPrefix(util.MachineCollection, id),
		Device: &ufspb.Machine_ChromeosMachine{
			ChromeosMachine: &ufspb.ChromeOSMachine{
				ReferenceBoard: board,
			},
		},
	}
}

func mockChromeBrowserMachine(id, lab, name string) *ufspb.Machine {
	return &ufspb.Machine{
		Name: util.AddPrefix(util.MachineCollection, id),
		Device: &ufspb.Machine_ChromeBrowserMachine{
			ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{
				Description: name,
			},
		},
	}
}

func assertMachineEqual(a *ufspb.Machine, b *ufspb.Machine) {
	So(a.GetName(), ShouldEqual, b.GetName())
	So(a.GetChromeBrowserMachine().GetDescription(), ShouldEqual,
		b.GetChromeBrowserMachine().GetDescription())
	So(a.GetChromeosMachine().GetReferenceBoard(), ShouldEqual,
		b.GetChromeosMachine().GetReferenceBoard())
}

func mockRack(id string, rackCapactiy int32) *ufspb.Rack {
	return &ufspb.Rack{
		Name:       util.AddPrefix(util.RackCollection, id),
		CapacityRu: rackCapactiy,
	}
}

func mockNic(id string) *ufspb.Nic {
	return &ufspb.Nic{
		Name:       util.AddPrefix(util.NicCollection, id),
		MacAddress: "12:ab",
		SwitchInterface: &ufspb.SwitchInterface{
			Switch:   "test-switch",
			PortName: "1",
		},
	}
}

func mockKVM(id string) *ufspb.KVM {
	return &ufspb.KVM{
		Name: util.AddPrefix(util.KVMCollection, id),
	}
}

func mockRPM(id string) *ufspb.RPM {
	return &ufspb.RPM{
		Name: util.AddPrefix(util.RPMCollection, id),
	}
}

func mockDrac(id string) *ufspb.Drac {
	return &ufspb.Drac{
		Name:       util.AddPrefix(util.DracCollection, id),
		MacAddress: "12:ab",
		SwitchInterface: &ufspb.SwitchInterface{
			Switch:   "test-switch",
			PortName: "1",
		},
	}
}

func mockSwitch(id string) *ufspb.Switch {
	return &ufspb.Switch{
		Name: util.AddPrefix(util.SwitchCollection, id),
	}
}

func TestMachineRegistration(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	Convey("Machine Registration", t, func() {
		Convey("Register machine - happy path", func() {
			s := mockSwitch("")
			s.Name = "test-switch"
			_, err := registration.CreateSwitch(ctx, s)
			So(err, ShouldBeNil)

			nic := mockNic("")
			nic.Name = "nic-X"
			nics := []*ufspb.Nic{nic}
			drac := mockDrac("")
			drac.Name = "drac-X"
			machine := &ufspb.Machine{
				Name: "machine-X",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{
						NicObjects: nics,
						DracObject: drac,
					},
				},
			}
			req := &ufsAPI.MachineRegistrationRequest{
				Machine: machine,
			}
			resp, _ := tf.Fleet.MachineRegistration(tf.C, req)
			So(resp, ShouldNotBeNil)
			So(resp, ShouldResembleProto, machine)
		})

		Convey("Register machine with nil machine", func() {
			req := &ufsAPI.MachineRegistrationRequest{}
			_, err := tf.Fleet.MachineRegistration(tf.C, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Machine "+ufsAPI.NilEntity)
		})

		Convey("Register machine - Invalid input empty machine name", func() {
			req := &ufsAPI.MachineRegistrationRequest{
				Machine: &ufspb.Machine{
					Name: "",
				},
			}
			_, err := tf.Fleet.MachineRegistration(tf.C, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Machine "+ufsAPI.EmptyName)
		})

		Convey("Create new machine - Invalid input invalid characters in machine name", func() {
			req := &ufsAPI.MachineRegistrationRequest{
				Machine: &ufspb.Machine{
					Name: "a.b)7&",
				},
			}
			_, err := tf.Fleet.MachineRegistration(tf.C, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Machine "+ufsAPI.InvalidCharacters)
		})

		Convey("Register machine - Invalid input empty nic name", func() {
			req := &ufsAPI.MachineRegistrationRequest{
				Machine: &ufspb.Machine{
					Name: "machine-1",
					Device: &ufspb.Machine_ChromeBrowserMachine{
						ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{
							NicObjects: []*ufspb.Nic{{
								Name: "",
							}},
						},
					},
				},
			}
			_, err := tf.Fleet.MachineRegistration(tf.C, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Nic "+ufsAPI.EmptyName)
		})

		Convey("Register machine - Invalid input invalid characters in nic name", func() {
			req := &ufsAPI.MachineRegistrationRequest{
				Machine: &ufspb.Machine{
					Name: "machine-1",
					Device: &ufspb.Machine_ChromeBrowserMachine{
						ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{
							NicObjects: []*ufspb.Nic{{
								Name: "a.b)7&",
							}},
						},
					},
				},
			}
			_, err := tf.Fleet.MachineRegistration(tf.C, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Nic a.b)7& has invalid characters in the name."+ufsAPI.InvalidCharacters)
		})

		Convey("Register machine - Invalid input empty drac name", func() {
			req := &ufsAPI.MachineRegistrationRequest{
				Machine: &ufspb.Machine{
					Name: "machine-1",
					Device: &ufspb.Machine_ChromeBrowserMachine{
						ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{
							DracObject: &ufspb.Drac{
								Name: "",
							},
						},
					},
				},
			}
			_, err := tf.Fleet.MachineRegistration(tf.C, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Drac "+ufsAPI.EmptyName)
		})

		Convey("Register machine - Invalid input invalid characters in drac name", func() {
			req := &ufsAPI.MachineRegistrationRequest{
				Machine: &ufspb.Machine{
					Name: "machine-1",
					Device: &ufspb.Machine_ChromeBrowserMachine{
						ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{
							DracObject: &ufspb.Drac{
								Name: "a.b)7&",
							},
						},
					},
				},
			}
			_, err := tf.Fleet.MachineRegistration(tf.C, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Drac "+ufsAPI.InvalidCharacters)
		})
	})
}

func TestRackRegistration(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	Convey("Rack Registration", t, func() {
		Convey("Register Rack - happy path", func() {
			switches := []*ufspb.Switch{{
				Name: "switch-X",
			}}
			kvms := []*ufspb.KVM{{
				Name: "kvm-X",
			}}
			rack := &ufspb.Rack{
				Name: "rack-X",
				Rack: &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{
						SwitchObjects: switches,
						KvmObjects:    kvms,
					},
				},
			}
			req := &ufsAPI.RackRegistrationRequest{
				Rack: rack,
			}
			resp, _ := tf.Fleet.RackRegistration(tf.C, req)
			So(resp, ShouldNotBeNil)
			So(resp, ShouldResembleProto, rack)
		})

		Convey("Register rack with nil rack", func() {
			req := &ufsAPI.RackRegistrationRequest{}
			_, err := tf.Fleet.RackRegistration(tf.C, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Rack "+ufsAPI.NilEntity)
		})

		Convey("Register rack - Invalid input empty rack name", func() {
			req := &ufsAPI.RackRegistrationRequest{
				Rack: &ufspb.Rack{
					Name: "",
				},
			}
			_, err := tf.Fleet.RackRegistration(tf.C, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Rack "+ufsAPI.EmptyName)
		})

		Convey("Create new rack - Invalid input invalid characters in rack name", func() {
			req := &ufsAPI.RackRegistrationRequest{
				Rack: &ufspb.Rack{
					Name: "a.b)7&",
				},
			}
			_, err := tf.Fleet.RackRegistration(tf.C, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Rack "+ufsAPI.InvalidCharacters)
		})

		Convey("Register rack - Invalid input empty switch name", func() {
			req := &ufsAPI.RackRegistrationRequest{
				Rack: &ufspb.Rack{
					Name: "rack-1",
					Rack: &ufspb.Rack_ChromeBrowserRack{
						ChromeBrowserRack: &ufspb.ChromeBrowserRack{
							SwitchObjects: []*ufspb.Switch{{
								Name: "",
							}},
						},
					},
				},
			}
			_, err := tf.Fleet.RackRegistration(tf.C, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Switch "+ufsAPI.EmptyName)
		})

		Convey("Register rack - Invalid input invalid characters in switch name", func() {
			req := &ufsAPI.RackRegistrationRequest{
				Rack: &ufspb.Rack{
					Name: "rack-1",
					Rack: &ufspb.Rack_ChromeBrowserRack{
						ChromeBrowserRack: &ufspb.ChromeBrowserRack{
							SwitchObjects: []*ufspb.Switch{{
								Name: "a.b)7&",
							}},
						},
					},
				},
			}
			_, err := tf.Fleet.RackRegistration(tf.C, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Switch a.b)7& has invalid characters in the name."+ufsAPI.InvalidCharacters)
		})

		Convey("Register rack - Invalid input empty kvm name", func() {
			req := &ufsAPI.RackRegistrationRequest{
				Rack: &ufspb.Rack{
					Name: "rack-1",
					Rack: &ufspb.Rack_ChromeBrowserRack{
						ChromeBrowserRack: &ufspb.ChromeBrowserRack{
							KvmObjects: []*ufspb.KVM{{
								Name: "",
							}},
						},
					},
				},
			}
			_, err := tf.Fleet.RackRegistration(tf.C, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "KVM "+ufsAPI.EmptyName)
		})

		Convey("Register rack - Invalid input invalid characters in kvm name", func() {
			req := &ufsAPI.RackRegistrationRequest{
				Rack: &ufspb.Rack{
					Name: "rack-1",
					Rack: &ufspb.Rack_ChromeBrowserRack{
						ChromeBrowserRack: &ufspb.ChromeBrowserRack{
							KvmObjects: []*ufspb.KVM{{
								Name: "a.b)7&",
							}},
						},
					},
				},
			}
			_, err := tf.Fleet.RackRegistration(tf.C, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "KVM a.b)7& has invalid characters in the name."+ufsAPI.InvalidCharacters)
		})
	})
}

func TestUpdateMachine(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	chromeOSMachine2 := mockChromeOSMachine("chromeos-asset-1", "chromeoslab", "veyron")
	chromeOSMachine3 := mockChromeOSMachine("", "chromeoslab", "samus")
	chromeOSMachine4 := mockChromeOSMachine("a.b)7&", "chromeoslab", "samus")
	Convey("UpdateMachines", t, func() {
		Convey("Update existing machines - happy path", func() {
			_, err := registration.CreateMachine(tf.C, &ufspb.Machine{
				Name: "chromeos-asset-1",
			})
			So(err, ShouldBeNil)

			ureq := &ufsAPI.UpdateMachineRequest{
				Machine: chromeOSMachine2,
			}
			resp, err := tf.Fleet.UpdateMachine(tf.C, ureq)
			So(err, ShouldBeNil)
			assertMachineEqual(resp, chromeOSMachine2)
		})

		Convey("Update machine - Invalid input nil", func() {
			req := &ufsAPI.UpdateMachineRequest{
				Machine: nil,
			}
			resp, err := tf.Fleet.UpdateMachine(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.NilEntity)
		})

		Convey("Update machine - Invalid input empty name", func() {
			chromeOSMachine3.Name = ""
			req := &ufsAPI.UpdateMachineRequest{
				Machine: chromeOSMachine3,
			}
			resp, err := tf.Fleet.UpdateMachine(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.EmptyName)
		})

		Convey("Update machine - Invalid input invalid characters", func() {
			req := &ufsAPI.UpdateMachineRequest{
				Machine: chromeOSMachine4,
			}
			resp, err := tf.Fleet.UpdateMachine(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidCharacters)
		})
	})
}

func TestGetMachine(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	chromeOSMachine1, _ := registration.CreateMachine(tf.C, &ufspb.Machine{
		Name: "chromeos-asset-1",
	})
	Convey("GetMachine", t, func() {
		Convey("Get machine by existing ID", func() {
			req := &ufsAPI.GetMachineRequest{
				Name: util.AddPrefix(util.MachineCollection, "chromeos-asset-1"),
			}
			resp, err := tf.Fleet.GetMachine(tf.C, req)
			So(err, ShouldBeNil)
			resp.Name = util.RemovePrefix(resp.Name)
			So(resp, ShouldResembleProto, chromeOSMachine1)
		})

		Convey("Get machine - Invalid input empty name", func() {
			req := &ufsAPI.GetMachineRequest{
				Name: "",
			}
			resp, err := tf.Fleet.GetMachine(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.EmptyName)
		})

		Convey("Get machine - Invalid input invalid characters", func() {
			req := &ufsAPI.GetMachineRequest{
				Name: util.AddPrefix(util.MachineCollection, "a.b)7&"),
			}
			resp, err := tf.Fleet.GetMachine(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidCharacters)
		})
	})
}

func TestListMachines(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	machines := make([]*ufspb.Machine, 0, 4)
	for i := 0; i < 4; i++ {
		chromeOSMachine1 := mockChromeOSMachine("", "chromeoslab", "samus")
		chromeOSMachine1.Name = fmt.Sprintf("chromeos-asset-%d", i)
		resp, _ := registration.CreateMachine(tf.C, chromeOSMachine1)
		chromeOSMachine1.Name = util.AddPrefix(util.MachineCollection, chromeOSMachine1.Name)
		machines = append(machines, resp)
	}
	Convey("ListMachines", t, func() {
		Convey("ListMachines - page_size negative - error", func() {
			req := &ufsAPI.ListMachinesRequest{
				PageSize: -5,
			}
			resp, err := tf.Fleet.ListMachines(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidPageSize)
		})

		Convey("ListMachines - Full listing - happy path", func() {
			req := &ufsAPI.ListMachinesRequest{}
			resp, err := tf.Fleet.ListMachines(tf.C, req)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp.Machines, ShouldResembleProto, machines)
		})

		Convey("ListMachines - filter format invalid format OR - error", func() {
			req := &ufsAPI.ListMachinesRequest{
				Filter: "machine=mac-1 | rpm=rpm-2",
			}
			_, err := tf.Fleet.ListMachines(tf.C, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidFilterFormat)
		})

		Convey("ListMachines - filter format valid AND", func() {
			req := &ufsAPI.ListMachinesRequest{
				Filter: "kvm=kvm-1 & rpm=rpm-1",
			}
			resp, err := tf.Fleet.ListMachines(tf.C, req)
			So(err, ShouldBeNil)
			So(resp.Machines, ShouldBeNil)
		})

		Convey("ListMachines - filter format valid", func() {
			req := &ufsAPI.ListMachinesRequest{
				Filter: "rack=rack-1",
			}
			resp, err := tf.Fleet.ListMachines(tf.C, req)
			So(err, ShouldBeNil)
			So(resp.Machines, ShouldBeNil)
		})
	})
}

func TestDeleteMachine(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	registration.CreateMachine(tf.C, &ufspb.Machine{
		Name: "chromeos-asset-1",
	})
	Convey("DeleteMachine", t, func() {
		Convey("Delete machine by existing ID", func() {
			req := &ufsAPI.DeleteMachineRequest{
				Name: util.AddPrefix(util.MachineCollection, "chromeos-asset-1"),
			}
			_, err := tf.Fleet.DeleteMachine(tf.C, req)
			So(err, ShouldBeNil)

			_, err = registration.GetMachine(tf.C, "chromeos-asset-1")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})

		Convey("Delete machine - Invalid input empty name", func() {
			req := &ufsAPI.DeleteMachineRequest{
				Name: "",
			}
			resp, err := tf.Fleet.DeleteMachine(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.EmptyName)
		})

		Convey("Delete machine - Invalid input invalid characters", func() {
			req := &ufsAPI.DeleteMachineRequest{
				Name: util.AddPrefix(util.MachineCollection, "a.b)7&"),
			}
			resp, err := tf.Fleet.DeleteMachine(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidCharacters)
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
			req := &ufsAPI.ImportMachinesRequest{
				Source: &ufsAPI.ImportMachinesRequest_MachineDbSource{
					MachineDbSource: &ufsAPI.MachineDBSource{
						Host: "fake_host",
					},
				},
			}
			res, err := tf.Fleet.ImportMachines(ctx, req)
			So(err, ShouldBeNil)
			So(res.Code, ShouldEqual, code.Code_OK)
			machines, _, err := registration.ListMachines(ctx, 100, "", nil, false)
			So(err, ShouldBeNil)
			So(machines, ShouldHaveLength, 3)
			So(ufsAPI.ParseResources(machines, "Name"), ShouldResemble, []string{"machine1", "machine2", "machine3"})
			for _, m := range machines {
				So(m.GetRealm(), ShouldEqual, util.BrowserLabAdminRealm)
				bm := m.GetChromeBrowserMachine()
				switch m.GetName() {
				case "machine1":
					So(bm.GetChromePlatform(), ShouldEqual, "fake_platform")
				case "machine2":
					So(bm.GetChromePlatform(), ShouldEqual, "fake_platform")
				case "machine3":
					So(bm.GetChromePlatform(), ShouldEqual, "fake_platform2")
				}
			}
		})
		Convey("import browser machines with empty machineDB host", func() {
			req := &ufsAPI.ImportMachinesRequest{
				Source: &ufsAPI.ImportMachinesRequest_MachineDbSource{
					MachineDbSource: &ufsAPI.MachineDBSource{
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
			req := &ufsAPI.ImportMachinesRequest{
				Source: &ufsAPI.ImportMachinesRequest_MachineDbSource{},
			}
			_, err := tf.Fleet.ImportMachines(ctx, req)
			So(err, ShouldNotBeNil)
			s, ok := status.FromError(err)
			So(ok, ShouldBeTrue)
			So(s.Code(), ShouldEqual, code.Code_INVALID_ARGUMENT)
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
			req := &ufsAPI.UpdateRackRequest{
				Rack: nil,
			}
			resp, err := tf.Fleet.UpdateRack(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.NilEntity)
		})

		Convey("Update rack - Invalid input empty name", func() {
			rack3.Name = ""
			req := &ufsAPI.UpdateRackRequest{
				Rack: rack3,
			}
			resp, err := tf.Fleet.UpdateRack(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.EmptyName)
		})

		Convey("Update rack - Invalid input invalid characters", func() {
			req := &ufsAPI.UpdateRackRequest{
				Rack: rack4,
			}
			resp, err := tf.Fleet.UpdateRack(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidCharacters)
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
			rack1 := &ufspb.Rack{
				Name: "rack-1",
			}
			_, err := registration.CreateRack(tf.C, rack1)
			So(err, ShouldBeNil)
			rack1.Name = util.AddPrefix(util.RackCollection, "rack-1")

			req := &ufsAPI.GetRackRequest{
				Name: util.AddPrefix(util.RackCollection, "rack-1"),
			}
			resp, err := tf.Fleet.GetRack(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, rack1)
		})
		Convey("Get rack by non-existing ID", func() {
			req := &ufsAPI.GetRackRequest{
				Name: util.AddPrefix(util.RackCollection, "rack-2"),
			}
			resp, err := tf.Fleet.GetRack(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Get rack - Invalid input empty name", func() {
			req := &ufsAPI.GetRackRequest{
				Name: "",
			}
			resp, err := tf.Fleet.GetRack(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.EmptyName)
		})
		Convey("Get rack - Invalid input invalid characters", func() {
			req := &ufsAPI.GetRackRequest{
				Name: util.AddPrefix(util.RackCollection, "a.b)7&"),
			}
			resp, err := tf.Fleet.GetRack(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidCharacters)
		})
	})
}

func TestListRacks(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	racks := make([]*ufspb.Rack, 0, 4)
	for i := 0; i < 4; i++ {
		rack1 := &ufspb.Rack{
			Name: fmt.Sprintf("rack-%d", i),
		}
		resp, _ := registration.CreateRack(tf.C, rack1)
		rack1.Name = util.AddPrefix(util.RackCollection, rack1.Name)
		racks = append(racks, resp)
	}
	Convey("ListRacks", t, func() {
		Convey("ListRacks - page_size negative - error", func() {
			req := &ufsAPI.ListRacksRequest{
				PageSize: -5,
			}
			resp, err := tf.Fleet.ListRacks(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidPageSize)
		})

		Convey("ListRacks - Full listing - happy path", func() {
			req := &ufsAPI.ListRacksRequest{}
			resp, err := tf.Fleet.ListRacks(tf.C, req)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp.Racks, ShouldResembleProto, racks)
		})

		Convey("ListRacks - filter format valid", func() {
			req := &ufsAPI.ListRacksRequest{
				Filter: "tag=tag-1",
			}
			resp, err := tf.Fleet.ListRacks(tf.C, req)
			So(err, ShouldBeNil)
			So(resp.Racks, ShouldBeNil)
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
			_, err := registration.CreateRack(tf.C, &ufspb.Rack{
				Name: "rack-1",
			})
			So(err, ShouldBeNil)

			req := &ufsAPI.DeleteRackRequest{
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
			req := &ufsAPI.DeleteRackRequest{
				Name: util.AddPrefix(util.RackCollection, "rack-2"),
			}
			_, err := tf.Fleet.DeleteRack(tf.C, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Delete rack - Invalid input empty name", func() {
			req := &ufsAPI.DeleteRackRequest{
				Name: "",
			}
			resp, err := tf.Fleet.DeleteRack(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.EmptyName)
		})
		Convey("Delete rack - Invalid input invalid characters", func() {
			req := &ufsAPI.DeleteRackRequest{
				Name: util.AddPrefix(util.RackCollection, "a.b)7&"),
			}
			resp, err := tf.Fleet.DeleteRack(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidCharacters)
		})
	})
}

func TestCreateNic(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	machine1 := &ufspb.Machine{
		Name: "machine-1",
		Device: &ufspb.Machine_ChromeBrowserMachine{
			ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
		},
	}
	registration.CreateMachine(tf.C, machine1)
	registration.CreateSwitch(tf.C, &ufspb.Switch{
		Name:         "test-switch",
		CapacityPort: 100,
	})
	Convey("CreateNic", t, func() {
		Convey("Create new nic with nic_id", func() {
			nic := mockNic("")
			req := &ufsAPI.CreateNicRequest{
				Nic:     nic,
				NicId:   "nic-1",
				Machine: "machine-1",
			}
			resp, err := tf.Fleet.CreateNic(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, nic)
		})

		Convey("Create new nic - Invalid input nil", func() {
			req := &ufsAPI.CreateNicRequest{
				Nic: nil,
			}
			resp, err := tf.Fleet.CreateNic(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.NilEntity)
		})

		Convey("Create new nic - Invalid input empty ID", func() {
			nic := mockNic("")
			req := &ufsAPI.CreateNicRequest{
				Nic:     nic,
				NicId:   "",
				Machine: "machine-1",
			}
			resp, err := tf.Fleet.CreateNic(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.EmptyID)
		})

		Convey("Create new nic - Invalid input invalid characters", func() {
			nic := mockNic("")
			req := &ufsAPI.CreateNicRequest{
				Nic:     nic,
				NicId:   "a.b)7&",
				Machine: "machine-1",
			}
			resp, err := tf.Fleet.CreateNic(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidCharacters)
		})

		Convey("Create new nic - Invalid input empty machine", func() {
			nic := mockNic("")
			req := &ufsAPI.CreateNicRequest{
				Nic:   nic,
				NicId: "nic-5",
			}
			resp, err := tf.Fleet.CreateNic(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.EmptyMachineName)
		})
	})
}

func TestUpdateNic(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	machine1 := &ufspb.Machine{
		Name: "machine-1",
		Device: &ufspb.Machine_ChromeBrowserMachine{
			ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
		},
	}
	registration.CreateMachine(tf.C, machine1)
	Convey("UpdateNic", t, func() {
		Convey("Update existing nic", func() {
			nic1 := &ufspb.Nic{
				Name:    "nic-1",
				Machine: "machine-1",
			}
			_, err := registration.CreateNic(tf.C, nic1)
			So(err, ShouldBeNil)

			nic2 := mockNic("nic-1")
			nic2.SwitchInterface = nil
			ureq := &ufsAPI.UpdateNicRequest{
				Nic:     nic2,
				Machine: "machine-1",
			}
			resp, err := tf.Fleet.UpdateNic(tf.C, ureq)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, nic2)
		})

		Convey("Update nic - Invalid input nil", func() {
			req := &ufsAPI.UpdateNicRequest{
				Nic: nil,
			}
			resp, err := tf.Fleet.UpdateNic(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.NilEntity)
		})

		Convey("Update nic - Invalid input empty name", func() {
			nic := mockNic("")
			nic.Name = ""
			req := &ufsAPI.UpdateNicRequest{
				Nic:     nic,
				Machine: "machine-1",
			}
			resp, err := tf.Fleet.UpdateNic(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.EmptyName)
		})

		Convey("Update nic - Invalid input invalid characters", func() {
			nic := mockNic("a.b)7&")
			req := &ufsAPI.UpdateNicRequest{
				Nic:     nic,
				Machine: "machine-1",
			}
			resp, err := tf.Fleet.UpdateNic(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidCharacters)
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
			nic1 := &ufspb.Nic{
				Name: "nic-1",
			}
			registration.CreateNic(tf.C, nic1)
			nic1.Name = util.AddPrefix(util.NicCollection, "nic-1")

			req := &ufsAPI.GetNicRequest{
				Name: util.AddPrefix(util.NicCollection, "nic-1"),
			}
			resp, err := tf.Fleet.GetNic(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, nic1)
		})

		Convey("Get nic - Invalid input empty name", func() {
			req := &ufsAPI.GetNicRequest{
				Name: "",
			}
			resp, err := tf.Fleet.GetNic(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.EmptyName)
		})

		Convey("Get nic - Invalid input invalid characters", func() {
			req := &ufsAPI.GetNicRequest{
				Name: util.AddPrefix(util.NicCollection, "a.b)7&"),
			}
			resp, err := tf.Fleet.GetNic(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidCharacters)
		})
	})
}

func TestListNics(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	nics := make([]*ufspb.Nic, 0, 4)
	for i := 0; i < 4; i++ {
		nic := &ufspb.Nic{
			Name: fmt.Sprintf("nic-%d", i),
		}
		resp, _ := registration.CreateNic(tf.C, nic)
		nic.Name = util.AddPrefix(util.NicCollection, nic.Name)
		nics = append(nics, resp)
	}
	Convey("ListNics", t, func() {
		Convey("ListNics - page_size negative - error", func() {
			req := &ufsAPI.ListNicsRequest{
				PageSize: -5,
			}
			resp, err := tf.Fleet.ListNics(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidPageSize)
		})

		Convey("ListNics - Full listing - happy path", func() {
			req := &ufsAPI.ListNicsRequest{}
			resp, err := tf.Fleet.ListNics(tf.C, req)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp.Nics, ShouldResembleProto, nics)
		})

		Convey("ListNics - filter format invalid format OR - error", func() {
			req := &ufsAPI.ListNicsRequest{
				Filter: "nic=mac-1 | rpm=rpm-2",
			}
			_, err := tf.Fleet.ListNics(tf.C, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidFilterFormat)
		})

		Convey("ListNics - filter format valid", func() {
			req := &ufsAPI.ListNicsRequest{
				Filter: "switch=switch-1",
			}
			resp, err := tf.Fleet.ListNics(tf.C, req)
			So(err, ShouldBeNil)
			So(resp.Nics, ShouldBeNil)
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
			req := &ufsAPI.DeleteNicRequest{
				Name: "",
			}
			resp, err := tf.Fleet.DeleteNic(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.EmptyName)
		})

		Convey("Delete nic - Invalid input invalid characters", func() {
			req := &ufsAPI.DeleteNicRequest{
				Name: util.AddPrefix(util.NicCollection, "a.b)7&"),
			}
			resp, err := tf.Fleet.DeleteNic(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidCharacters)
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
			req := &ufsAPI.ImportNicsRequest{
				Source: &ufsAPI.ImportNicsRequest_MachineDbSource{
					MachineDbSource: &ufsAPI.MachineDBSource{
						Host: "fake_host",
					},
				},
			}
			res, err := tf.Fleet.ImportNics(ctx, req)
			So(err, ShouldBeNil)
			So(res.Code, ShouldEqual, code.Code_OK)
			nics, _, err := registration.ListNics(ctx, 100, "", nil, false)
			So(err, ShouldBeNil)
			So(ufsAPI.ParseResources(nics, "Name"), ShouldResemble, []string{"machine1:eth0", "machine1:eth1", "machine2:eth0", "machine3:eth0"})
			switches := make([]*ufspb.SwitchInterface, len(nics))
			for i, nic := range nics {
				switches[i] = nic.GetSwitchInterface()
			}
			So(switches, ShouldResembleProto, []*ufspb.SwitchInterface{
				{
					Switch:   "eq017.atl97",
					PortName: "2",
				},
				{
					Switch:   "eq017.atl97",
					PortName: "3",
				},
				{
					Switch:   "eq017.atl97",
					PortName: "4",
				},
				{
					Switch:   "eq041.atl97",
					PortName: "1",
				},
			})
			dracs, _, err := registration.ListDracs(ctx, 100, "", nil, false)
			So(err, ShouldBeNil)
			So(ufsAPI.ParseResources(dracs, "Name"), ShouldResemble, []string{"drac-hostname"})
			So(ufsAPI.ParseResources(dracs, "DisplayName"), ShouldResemble, []string{"machine1:drac"})
			dhcps, _, err := configuration.ListDHCPConfigs(ctx, 100, "", nil, false)
			So(err, ShouldBeNil)
			So(ufsAPI.ParseResources(dhcps, "Ip"), ShouldResemble, []string{"ip1.1"})
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
			req := &ufsAPI.ImportDatacentersRequest{
				Source: &ufsAPI.ImportDatacentersRequest_ConfigSource{
					ConfigSource: &ufsAPI.ConfigSource{
						ConfigServiceName: "fake-service",
						FileName:          "fakeDatacenter.cfg",
					},
				},
			}
			res, err := tf.Fleet.ImportDatacenters(ctx, req)
			So(err, ShouldBeNil)
			So(res.Code, ShouldEqual, code.Code_OK)
			dhcps, _, err := configuration.ListDHCPConfigs(ctx, 100, "", nil, false)
			So(err, ShouldBeNil)
			So(dhcps, ShouldHaveLength, 3)
			racks, _, err := registration.ListRacks(ctx, 100, "", nil, false)
			So(err, ShouldBeNil)
			So(racks, ShouldHaveLength, 2)
			So(ufsAPI.ParseResources(racks, "Name"), ShouldResemble, []string{"cr20", "cr22"})
			kvms, _, err := registration.ListKVMs(ctx, 100, "", nil, false)
			So(err, ShouldBeNil)
			So(kvms, ShouldHaveLength, 3)
			So(ufsAPI.ParseResources(kvms, "Name"), ShouldResemble, []string{"cr20-kvm1", "cr22-kvm1", "cr22-kvm2"})
			So(ufsAPI.ParseResources(kvms, "ChromePlatform"), ShouldResemble, []string{"Raritan_DKX3", "Raritan_DKX3", "Raritan_DKX3"})
			switches, _, err := registration.ListSwitches(ctx, 100, "", nil, false)
			So(err, ShouldBeNil)
			So(switches, ShouldHaveLength, 4)
			So(ufsAPI.ParseResources(switches, "Name"), ShouldResemble, []string{"eq017.atl97", "eq041.atl97", "eq050.atl97", "eq113.atl97"})
			rackLSEs, _, err := inventory.ListRackLSEs(ctx, 100, "", nil, false)
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
	rack1 := &ufspb.Rack{
		Name: "rack-1",
		Rack: &ufspb.Rack_ChromeBrowserRack{
			ChromeBrowserRack: &ufspb.ChromeBrowserRack{},
		},
	}
	registration.CreateRack(tf.C, rack1)
	Convey("CreateKVM", t, func() {
		Convey("Create new KVM with KVM_id", func() {
			KVM1 := mockKVM("")
			req := &ufsAPI.CreateKVMRequest{
				KVM:   KVM1,
				KVMId: "KVM-1",
				Rack:  "rack-1",
			}
			resp, err := tf.Fleet.CreateKVM(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, KVM1)
		})

		Convey("Create new KVM - Invalid input nil", func() {
			req := &ufsAPI.CreateKVMRequest{
				KVM:  nil,
				Rack: "rack-1",
			}
			resp, err := tf.Fleet.CreateKVM(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.NilEntity)
		})

		Convey("Create new KVM - Invalid input empty ID", func() {
			req := &ufsAPI.CreateKVMRequest{
				KVM:   mockKVM(""),
				KVMId: "",
				Rack:  "rack-1",
			}
			resp, err := tf.Fleet.CreateKVM(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.EmptyID)
		})

		Convey("Create new KVM - Invalid input invalid characters", func() {
			req := &ufsAPI.CreateKVMRequest{
				KVM:   mockKVM(""),
				KVMId: "a.b)7&",
				Rack:  "rack-1",
			}
			resp, err := tf.Fleet.CreateKVM(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidCharacters)
		})

		Convey("Create new kvm - Invalid input empty rack", func() {
			req := &ufsAPI.CreateKVMRequest{
				KVM:   mockKVM("x"),
				KVMId: "kvm-5",
			}
			resp, err := tf.Fleet.CreateKVM(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.EmptyRackName)
		})
	})
}

func TestUpdateKVM(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	rack1 := &ufspb.Rack{
		Name: "rack-1",
		Rack: &ufspb.Rack_ChromeBrowserRack{
			ChromeBrowserRack: &ufspb.ChromeBrowserRack{},
		},
	}
	registration.CreateRack(tf.C, rack1)
	Convey("UpdateKVM", t, func() {
		Convey("Update existing KVM", func() {
			KVM1 := &ufspb.KVM{
				Name: "KVM-1",
			}
			resp, err := registration.CreateKVM(tf.C, KVM1)
			So(err, ShouldBeNil)

			KVM2 := mockKVM("KVM-1")
			ureq := &ufsAPI.UpdateKVMRequest{
				KVM:  KVM2,
				Rack: "rack-1",
			}
			resp, err = tf.Fleet.UpdateKVM(tf.C, ureq)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, KVM2)
		})

		Convey("Update KVM - Invalid input nil", func() {
			req := &ufsAPI.UpdateKVMRequest{
				KVM:  nil,
				Rack: "rack-1",
			}
			resp, err := tf.Fleet.UpdateKVM(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.NilEntity)
		})

		Convey("Update KVM - Invalid input empty name", func() {
			KVM3 := mockKVM("KVM-3")
			KVM3.Name = ""
			req := &ufsAPI.UpdateKVMRequest{
				KVM:  KVM3,
				Rack: "rack-1",
			}
			resp, err := tf.Fleet.UpdateKVM(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.EmptyName)
		})

		Convey("Update KVM - Invalid input invalid characters", func() {
			req := &ufsAPI.UpdateKVMRequest{
				KVM: mockKVM("a.b)7&"),
			}
			resp, err := tf.Fleet.UpdateKVM(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidCharacters)
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
			KVM1 := &ufspb.KVM{
				Name: "KVM-1",
			}
			_, err := registration.CreateKVM(tf.C, KVM1)
			So(err, ShouldBeNil)
			KVM1.Name = util.AddPrefix(util.KVMCollection, "KVM-1")

			req := &ufsAPI.GetKVMRequest{
				Name: util.AddPrefix(util.KVMCollection, "KVM-1"),
			}
			resp, err := tf.Fleet.GetKVM(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, KVM1)
		})
		Convey("Get KVM by non-existing ID", func() {
			req := &ufsAPI.GetKVMRequest{
				Name: util.AddPrefix(util.KVMCollection, "KVM-2"),
			}
			resp, err := tf.Fleet.GetKVM(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Get KVM - Invalid input empty name", func() {
			req := &ufsAPI.GetKVMRequest{
				Name: "",
			}
			resp, err := tf.Fleet.GetKVM(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.EmptyName)
		})
		Convey("Get KVM - Invalid input invalid characters", func() {
			req := &ufsAPI.GetKVMRequest{
				Name: util.AddPrefix(util.KVMCollection, "a.b)7&"),
			}
			resp, err := tf.Fleet.GetKVM(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidCharacters)
		})
	})
}

func TestListKVMs(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	KVMs := make([]*ufspb.KVM, 0, 4)
	for i := 0; i < 4; i++ {
		kvm := &ufspb.KVM{
			Name: fmt.Sprintf("kvm-%d", i),
		}
		resp, _ := registration.CreateKVM(tf.C, kvm)
		kvm.Name = util.AddPrefix(util.KVMCollection, kvm.Name)
		KVMs = append(KVMs, resp)
	}
	Convey("ListKVMs", t, func() {
		Convey("ListKVMs - page_size negative - error", func() {
			req := &ufsAPI.ListKVMsRequest{
				PageSize: -5,
			}
			resp, err := tf.Fleet.ListKVMs(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidPageSize)
		})

		Convey("ListKVMs - Full listing - happy path", func() {
			req := &ufsAPI.ListKVMsRequest{}
			resp, err := tf.Fleet.ListKVMs(tf.C, req)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp.KVMs, ShouldResembleProto, KVMs)
		})

		Convey("ListKVMs - filter format invalid format OR - error", func() {
			req := &ufsAPI.ListKVMsRequest{
				Filter: "platform=chromeplatform-1 | rpm=rpm-2",
			}
			_, err := tf.Fleet.ListKVMs(tf.C, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidFilterFormat)
		})

		Convey("ListKVMs - filter format valid", func() {
			req := &ufsAPI.ListKVMsRequest{
				Filter: "platform=chromeplatform-1",
			}
			resp, err := tf.Fleet.ListKVMs(tf.C, req)
			So(err, ShouldBeNil)
			So(resp.KVMs, ShouldBeNil)
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
			KVM2 := &ufspb.KVM{
				Name: "KVM-2",
			}
			_, err := registration.CreateKVM(tf.C, KVM2)
			So(err, ShouldBeNil)

			dreq := &ufsAPI.DeleteKVMRequest{
				Name: util.AddPrefix(util.KVMCollection, "KVM-2"),
			}
			_, err = tf.Fleet.DeleteKVM(tf.C, dreq)
			So(err, ShouldBeNil)

			_, err = registration.GetKVM(tf.C, "KVM-2")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})

		Convey("Delete KVM - Invalid input empty name", func() {
			req := &ufsAPI.DeleteKVMRequest{
				Name: "",
			}
			resp, err := tf.Fleet.DeleteKVM(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.EmptyName)
		})

		Convey("Delete KVM - Invalid input invalid characters", func() {
			req := &ufsAPI.DeleteKVMRequest{
				Name: util.AddPrefix(util.KVMCollection, "a.b)7&"),
			}
			resp, err := tf.Fleet.DeleteKVM(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidCharacters)
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
			req := &ufsAPI.CreateRPMRequest{
				RPM:   RPM1,
				RPMId: "RPM-1",
			}
			resp, err := tf.Fleet.CreateRPM(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, RPM1)
		})

		Convey("Create existing RPM", func() {
			req := &ufsAPI.CreateRPMRequest{
				RPM:   RPM3,
				RPMId: "RPM-1",
			}
			resp, err := tf.Fleet.CreateRPM(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, AlreadyExists)
		})

		Convey("Create new RPM - Invalid input nil", func() {
			req := &ufsAPI.CreateRPMRequest{
				RPM: nil,
			}
			resp, err := tf.Fleet.CreateRPM(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.NilEntity)
		})

		Convey("Create new RPM - Invalid input empty ID", func() {
			req := &ufsAPI.CreateRPMRequest{
				RPM:   RPM2,
				RPMId: "",
			}
			resp, err := tf.Fleet.CreateRPM(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.EmptyID)
		})

		Convey("Create new RPM - Invalid input invalid characters", func() {
			req := &ufsAPI.CreateRPMRequest{
				RPM:   RPM2,
				RPMId: "a.b)7&",
			}
			resp, err := tf.Fleet.CreateRPM(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidCharacters)
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
			req := &ufsAPI.CreateRPMRequest{
				RPM:   RPM1,
				RPMId: "RPM-1",
			}
			resp, err := tf.Fleet.CreateRPM(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, RPM1)
			ureq := &ufsAPI.UpdateRPMRequest{
				RPM: RPM2,
			}
			resp, err = tf.Fleet.UpdateRPM(tf.C, ureq)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, RPM2)
		})

		Convey("Update non-existing RPM", func() {
			ureq := &ufsAPI.UpdateRPMRequest{
				RPM: RPM3,
			}
			resp, err := tf.Fleet.UpdateRPM(tf.C, ureq)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})

		Convey("Update RPM - Invalid input nil", func() {
			req := &ufsAPI.UpdateRPMRequest{
				RPM: nil,
			}
			resp, err := tf.Fleet.UpdateRPM(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.NilEntity)
		})

		Convey("Update RPM - Invalid input empty name", func() {
			RPM3.Name = ""
			req := &ufsAPI.UpdateRPMRequest{
				RPM: RPM3,
			}
			resp, err := tf.Fleet.UpdateRPM(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.EmptyName)
		})

		Convey("Update RPM - Invalid input invalid characters", func() {
			req := &ufsAPI.UpdateRPMRequest{
				RPM: RPM4,
			}
			resp, err := tf.Fleet.UpdateRPM(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidCharacters)
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
		req := &ufsAPI.CreateRPMRequest{
			RPM:   RPM1,
			RPMId: "RPM-1",
		}
		resp, err := tf.Fleet.CreateRPM(tf.C, req)
		So(err, ShouldBeNil)
		So(resp, ShouldResembleProto, RPM1)
		Convey("Get RPM by existing ID", func() {
			req := &ufsAPI.GetRPMRequest{
				Name: util.AddPrefix(util.RPMCollection, "RPM-1"),
			}
			resp, err := tf.Fleet.GetRPM(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, RPM1)
		})
		Convey("Get RPM by non-existing ID", func() {
			req := &ufsAPI.GetRPMRequest{
				Name: util.AddPrefix(util.RPMCollection, "RPM-2"),
			}
			resp, err := tf.Fleet.GetRPM(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Get RPM - Invalid input empty name", func() {
			req := &ufsAPI.GetRPMRequest{
				Name: "",
			}
			resp, err := tf.Fleet.GetRPM(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.EmptyName)
		})
		Convey("Get RPM - Invalid input invalid characters", func() {
			req := &ufsAPI.GetRPMRequest{
				Name: util.AddPrefix(util.RPMCollection, "a.b)7&"),
			}
			resp, err := tf.Fleet.GetRPM(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidCharacters)
		})
	})
}

func TestListRPMs(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	RPMs := make([]*ufspb.RPM, 0, 4)
	for i := 0; i < 4; i++ {
		rpm := &ufspb.RPM{
			Name: fmt.Sprintf("rpm-%d", i),
		}
		resp, _ := registration.CreateRPM(tf.C, rpm)
		resp.Name = util.AddPrefix(util.RPMCollection, resp.Name)
		RPMs = append(RPMs, resp)
	}
	Convey("ListRPMs", t, func() {
		Convey("ListRPMs - page_size negative - error", func() {
			req := &ufsAPI.ListRPMsRequest{
				PageSize: -5,
			}
			resp, err := tf.Fleet.ListRPMs(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidPageSize)
		})

		Convey("ListRPMs - Full listing - happy path", func() {
			req := &ufsAPI.ListRPMsRequest{
				PageSize: 2000,
			}
			resp, err := tf.Fleet.ListRPMs(tf.C, req)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp.RPMs, ShouldResembleProto, RPMs)
		})

		Convey("ListRPMs - filter format invalid format OR - error", func() {
			req := &ufsAPI.ListRPMsRequest{
				Filter: "platform=chromeplatform-1 | rpm=rpm-2",
			}
			_, err := tf.Fleet.ListRPMs(tf.C, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidFilterFormat)
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
			req := &ufsAPI.CreateRPMRequest{
				RPM:   RPM2,
				RPMId: "RPM-2",
			}
			resp, err := tf.Fleet.CreateRPM(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, RPM2)

			dreq := &ufsAPI.DeleteRPMRequest{
				Name: util.AddPrefix(util.RPMCollection, "RPM-2"),
			}
			_, err = tf.Fleet.DeleteRPM(tf.C, dreq)
			So(err, ShouldBeNil)

			greq := &ufsAPI.GetRPMRequest{
				Name: util.AddPrefix(util.RPMCollection, "RPM-2"),
			}
			res, err := tf.Fleet.GetRPM(tf.C, greq)
			So(res, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})

		Convey("Delete RPM by non-existing ID", func() {
			req := &ufsAPI.DeleteRPMRequest{
				Name: util.AddPrefix(util.RPMCollection, "RPM-2"),
			}
			_, err := tf.Fleet.DeleteRPM(tf.C, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})

		Convey("Delete RPM - Invalid input empty name", func() {
			req := &ufsAPI.DeleteRPMRequest{
				Name: "",
			}
			resp, err := tf.Fleet.DeleteRPM(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.EmptyName)
		})

		Convey("Delete RPM - Invalid input invalid characters", func() {
			req := &ufsAPI.DeleteRPMRequest{
				Name: util.AddPrefix(util.RPMCollection, "a.b)7&"),
			}
			resp, err := tf.Fleet.DeleteRPM(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidCharacters)
		})
	})
}

func TestCreateDrac(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	machine1 := &ufspb.Machine{
		Name: "machine-1",
		Device: &ufspb.Machine_ChromeBrowserMachine{
			ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
		},
	}
	registration.CreateMachine(tf.C, machine1)
	registration.CreateSwitch(tf.C, &ufspb.Switch{
		Name:         "test-switch",
		CapacityPort: 100,
	})
	Convey("CreateDrac", t, func() {
		Convey("Create new drac with drac_id", func() {
			drac := mockDrac("")
			req := &ufsAPI.CreateDracRequest{
				Drac:    drac,
				DracId:  "Drac-1",
				Machine: "machine-1",
			}
			resp, err := tf.Fleet.CreateDrac(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, drac)
		})

		Convey("Create new drac - Invalid input nil", func() {
			req := &ufsAPI.CreateDracRequest{
				Drac: nil,
			}
			resp, err := tf.Fleet.CreateDrac(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.NilEntity)
		})

		Convey("Create new drac - Invalid input empty ID", func() {
			drac := mockDrac("")
			req := &ufsAPI.CreateDracRequest{
				Drac:    drac,
				DracId:  "",
				Machine: "machine-1",
			}
			resp, err := tf.Fleet.CreateDrac(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.EmptyID)
		})

		Convey("Create new drac - Invalid input invalid characters", func() {
			drac := mockDrac("")
			req := &ufsAPI.CreateDracRequest{
				Drac:    drac,
				DracId:  "a.b)7&",
				Machine: "machine-1",
			}
			resp, err := tf.Fleet.CreateDrac(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidCharacters)
		})

		Convey("Create new drac - Invalid input empty machine", func() {
			drac := mockDrac("")
			req := &ufsAPI.CreateDracRequest{
				Drac:   drac,
				DracId: "drac-5",
			}
			resp, err := tf.Fleet.CreateDrac(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.EmptyMachineName)
		})
	})
}

func TestUpdateDrac(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	machine1 := &ufspb.Machine{
		Name: "machine-1",
		Device: &ufspb.Machine_ChromeBrowserMachine{
			ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
		},
	}
	registration.CreateMachine(tf.C, machine1)
	Convey("UpdateDrac", t, func() {
		Convey("Update existing drac", func() {
			drac1 := &ufspb.Drac{
				Name:    "drac-1",
				Machine: "machine-1",
			}
			resp, err := registration.CreateDrac(tf.C, drac1)
			So(err, ShouldBeNil)

			drac2 := mockDrac("drac-1")
			drac2.SwitchInterface = nil
			ureq := &ufsAPI.UpdateDracRequest{
				Drac:    drac2,
				Machine: "machine-1",
			}
			resp, err = tf.Fleet.UpdateDrac(tf.C, ureq)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, drac2)
		})

		Convey("Update drac - Invalid input nil", func() {
			req := &ufsAPI.UpdateDracRequest{
				Drac: nil,
			}
			resp, err := tf.Fleet.UpdateDrac(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.NilEntity)
		})

		Convey("Update drac - Invalid input empty name", func() {
			drac := mockDrac("")
			drac.Name = ""
			req := &ufsAPI.UpdateDracRequest{
				Drac:    drac,
				Machine: "machine-1",
			}
			resp, err := tf.Fleet.UpdateDrac(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.EmptyName)
		})

		Convey("Update drac - Invalid input invalid characters", func() {
			drac := mockDrac("a.b)7&")
			req := &ufsAPI.UpdateDracRequest{
				Drac:    drac,
				Machine: "machine-1",
			}
			resp, err := tf.Fleet.UpdateDrac(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidCharacters)
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
			drac1 := &ufspb.Drac{
				Name: "drac-1",
			}
			registration.CreateDrac(tf.C, drac1)
			drac1.Name = util.AddPrefix(util.DracCollection, "drac-1")

			req := &ufsAPI.GetDracRequest{
				Name: util.AddPrefix(util.DracCollection, "drac-1"),
			}
			resp, err := tf.Fleet.GetDrac(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, drac1)
		})

		Convey("Get drac - Invalid input empty name", func() {
			req := &ufsAPI.GetDracRequest{
				Name: "",
			}
			resp, err := tf.Fleet.GetDrac(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.EmptyName)
		})

		Convey("Get drac - Invalid input invalid characters", func() {
			req := &ufsAPI.GetDracRequest{
				Name: util.AddPrefix(util.DracCollection, "a.b)7&"),
			}
			resp, err := tf.Fleet.GetDrac(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidCharacters)
		})
	})
}

func TestListDracs(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	dracs := make([]*ufspb.Drac, 0, 4)
	for i := 0; i < 4; i++ {
		drac := &ufspb.Drac{
			Name: fmt.Sprintf("drac-%d", i),
		}
		resp, _ := registration.CreateDrac(tf.C, drac)
		drac.Name = util.AddPrefix(util.DracCollection, drac.Name)
		dracs = append(dracs, resp)
	}
	Convey("ListDracs", t, func() {
		Convey("ListDracs - page_size negative - error", func() {
			req := &ufsAPI.ListDracsRequest{
				PageSize: -5,
			}
			resp, err := tf.Fleet.ListDracs(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidPageSize)
		})

		Convey("ListDracs - Full listing - happy path", func() {
			req := &ufsAPI.ListDracsRequest{}
			resp, err := tf.Fleet.ListDracs(tf.C, req)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp.Dracs, ShouldResembleProto, dracs)
		})

		Convey("ListDracs - filter format invalid format OR - error", func() {
			req := &ufsAPI.ListDracsRequest{
				Filter: "drac=mac-1 | rpm=rpm-2",
			}
			_, err := tf.Fleet.ListDracs(tf.C, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidFilterFormat)
		})

		Convey("ListDracs - filter format valid", func() {
			req := &ufsAPI.ListDracsRequest{
				Filter: "switch=switch-1",
			}
			resp, err := tf.Fleet.ListDracs(tf.C, req)
			So(err, ShouldBeNil)
			So(resp.Dracs, ShouldBeNil)
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
			req := &ufsAPI.DeleteDracRequest{
				Name: "",
			}
			resp, err := tf.Fleet.DeleteDrac(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.EmptyName)
		})

		Convey("Delete drac - Invalid input invalid characters", func() {
			req := &ufsAPI.DeleteDracRequest{
				Name: util.AddPrefix(util.DracCollection, "a.b)7&"),
			}
			resp, err := tf.Fleet.DeleteDrac(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidCharacters)
		})
	})
}

func TestCreateSwitch(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	rack1 := &ufspb.Rack{
		Name: "rack-1",
		Rack: &ufspb.Rack_ChromeBrowserRack{
			ChromeBrowserRack: &ufspb.ChromeBrowserRack{},
		},
	}
	registration.CreateRack(tf.C, rack1)
	Convey("CreateSwitch", t, func() {
		Convey("Create new switch with switch_id", func() {
			switch1 := mockSwitch("")
			req := &ufsAPI.CreateSwitchRequest{
				Switch:   switch1,
				SwitchId: "Switch-1",
				Rack:     "rack-1",
			}
			resp, err := tf.Fleet.CreateSwitch(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, switch1)
		})

		Convey("Create new switch - Invalid input nil", func() {
			req := &ufsAPI.CreateSwitchRequest{
				Switch: nil,
				Rack:   "rack-1",
			}
			resp, err := tf.Fleet.CreateSwitch(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.NilEntity)
		})

		Convey("Create new switch - Invalid input empty ID", func() {
			req := &ufsAPI.CreateSwitchRequest{
				Switch:   mockSwitch(""),
				SwitchId: "",
				Rack:     "rack-1",
			}
			resp, err := tf.Fleet.CreateSwitch(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.EmptyID)
		})

		Convey("Create new switch - Invalid input invalid characters", func() {
			req := &ufsAPI.CreateSwitchRequest{
				Switch:   mockSwitch(""),
				SwitchId: "a.b)7&",
				Rack:     "rack-1",
			}
			resp, err := tf.Fleet.CreateSwitch(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidCharacters)
		})

		Convey("Create new switch - Invalid input empty rack", func() {
			req := &ufsAPI.CreateSwitchRequest{
				Switch:   mockSwitch("x"),
				SwitchId: "switch-5",
			}
			resp, err := tf.Fleet.CreateSwitch(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.EmptyRackName)
		})
	})
}

func TestUpdateSwitch(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	rack1 := &ufspb.Rack{
		Name: "rack-1",
		Rack: &ufspb.Rack_ChromeBrowserRack{
			ChromeBrowserRack: &ufspb.ChromeBrowserRack{},
		},
	}
	registration.CreateRack(tf.C, rack1)
	Convey("UpdateSwitch", t, func() {
		Convey("Update existing switch", func() {
			switch1 := &ufspb.Switch{
				Name: "switch-1",
			}
			resp, err := registration.CreateSwitch(tf.C, switch1)
			So(err, ShouldBeNil)

			switch2 := mockSwitch("switch-1")
			ureq := &ufsAPI.UpdateSwitchRequest{
				Switch: switch2,
				Rack:   "rack-1",
			}
			resp, err = tf.Fleet.UpdateSwitch(tf.C, ureq)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, switch2)
		})

		Convey("Update switch - Invalid input nil", func() {
			req := &ufsAPI.UpdateSwitchRequest{
				Switch: nil,
				Rack:   "rack-1",
			}
			resp, err := tf.Fleet.UpdateSwitch(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.NilEntity)
		})

		Convey("Update switch - Invalid input empty name", func() {
			switch1 := mockSwitch("")
			switch1.Name = ""
			req := &ufsAPI.UpdateSwitchRequest{
				Switch: switch1,
				Rack:   "rack-1",
			}
			resp, err := tf.Fleet.UpdateSwitch(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.EmptyName)
		})

		Convey("Update switch - Invalid input invalid characters", func() {
			req := &ufsAPI.UpdateSwitchRequest{
				Switch: mockSwitch("a.b)7&"),
			}
			resp, err := tf.Fleet.UpdateSwitch(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidCharacters)
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
			switch1 := &ufspb.Switch{
				Name: "switch-1",
			}
			_, err := registration.CreateSwitch(tf.C, switch1)
			So(err, ShouldBeNil)
			switch1.Name = util.AddPrefix(util.SwitchCollection, "switch-1")

			req := &ufsAPI.GetSwitchRequest{
				Name: util.AddPrefix(util.SwitchCollection, "switch-1"),
			}
			resp, err := tf.Fleet.GetSwitch(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, switch1)
		})
		Convey("Get switch by non-existing ID", func() {
			req := &ufsAPI.GetSwitchRequest{
				Name: util.AddPrefix(util.SwitchCollection, "switch-2"),
			}
			resp, err := tf.Fleet.GetSwitch(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Get switch - Invalid input empty name", func() {
			req := &ufsAPI.GetSwitchRequest{
				Name: "",
			}
			resp, err := tf.Fleet.GetSwitch(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.EmptyName)
		})
		Convey("Get switch - Invalid input invalid characters", func() {
			req := &ufsAPI.GetSwitchRequest{
				Name: util.AddPrefix(util.SwitchCollection, "a.b)7&"),
			}
			resp, err := tf.Fleet.GetSwitch(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidCharacters)
		})
	})
}

func TestListSwitches(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	switches := make([]*ufspb.Switch, 0, 4)
	for i := 0; i < 4; i++ {
		s := &ufspb.Switch{
			Name: fmt.Sprintf("switch-%d", i),
		}
		resp, _ := registration.CreateSwitch(tf.C, s)
		s.Name = util.AddPrefix(util.SwitchCollection, s.Name)
		switches = append(switches, resp)
	}
	Convey("ListSwitches", t, func() {
		Convey("ListSwitches - page_size negative - error", func() {
			req := &ufsAPI.ListSwitchesRequest{
				PageSize: -5,
			}
			resp, err := tf.Fleet.ListSwitches(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidPageSize)
		})

		Convey("ListSwitches - Full listing - happy path", func() {
			req := &ufsAPI.ListSwitchesRequest{}
			resp, err := tf.Fleet.ListSwitches(tf.C, req)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp.Switches, ShouldResembleProto, switches)
		})

		Convey("ListSwitches - filter format invalid format OR - error", func() {
			req := &ufsAPI.ListSwitchesRequest{
				Filter: "platform=chromeplatform-1 | rpm=rpm-2",
			}
			_, err := tf.Fleet.ListSwitches(tf.C, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidFilterFormat)
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
			switch2 := &ufspb.Switch{
				Name: "switch-2",
			}
			_, err := registration.CreateSwitch(tf.C, switch2)
			So(err, ShouldBeNil)

			dreq := &ufsAPI.DeleteSwitchRequest{
				Name: util.AddPrefix(util.SwitchCollection, "switch-2"),
			}
			_, err = tf.Fleet.DeleteSwitch(tf.C, dreq)
			So(err, ShouldBeNil)

			_, err = registration.GetSwitch(tf.C, "switch-2")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})

		Convey("Delete switch - Invalid input empty name", func() {
			req := &ufsAPI.DeleteSwitchRequest{
				Name: "",
			}
			resp, err := tf.Fleet.DeleteSwitch(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.EmptyName)
		})

		Convey("Delete switch - Invalid input invalid characters", func() {
			req := &ufsAPI.DeleteSwitchRequest{
				Name: util.AddPrefix(util.SwitchCollection, "a.b)7&"),
			}
			resp, err := tf.Fleet.DeleteSwitch(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidCharacters)
		})
	})
}
