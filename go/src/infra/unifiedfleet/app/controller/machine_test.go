// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"

	ufspb "infra/unifiedfleet/api/v1/proto"
	"infra/unifiedfleet/app/model/configuration"
	. "infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/model/history"
	"infra/unifiedfleet/app/model/inventory"
	"infra/unifiedfleet/app/model/registration"
)

func TestCreateMachine(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("CreateMachines", t, func() {
		Convey("Create already existing machine", func() {
			machine := &ufspb.Machine{
				Name: "machine-10",
			}
			_, err := registration.CreateMachine(ctx, machine)
			So(err, ShouldBeNil)

			_, err = CreateMachine(ctx, machine)
			So(err.Error(), ShouldContainSubstring, "Machine machine-10 already exists in the system.")
		})

		Convey("Create new machine with non existing ChromePlatform", func() {
			machine1 := &ufspb.Machine{
				Name: "machine-1",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{
						ChromePlatform: "chromePlatform-1",
					},
				},
			}
			_, err := CreateMachine(ctx, machine1)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, CannotCreate)

			// No changes are recorded as the creation fails
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "machines/machine-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})

		Convey("Create new machine with non existing resources", func() {
			machine3 := &ufspb.Machine{
				Name: "machine-3",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{
						ChromePlatform: "chromePlatform-3",
						Nics:           []string{"nic-3"},
						Drac:           "drac-3",
						KvmInterface: &ufspb.KVMInterface{
							Kvm: "kvm-3",
						},
						RpmInterface: &ufspb.RPMInterface{
							Rpm: "rpm-3",
						},
					},
				},
			}
			_, err := CreateMachine(ctx, machine3)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Cannot create")

			// No changes are recorded as the creation fails
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "machines/machine-3")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})

		Convey("Create new machine with existing resources", func() {
			chromePlatform2 := &ufspb.ChromePlatform{
				Name: "chromePlatform-2",
			}
			_, err := configuration.CreateChromePlatform(ctx, chromePlatform2)
			So(err, ShouldBeNil)

			switch2 := &ufspb.Switch{
				Name: "switch-2",
			}
			_, err = registration.CreateSwitch(ctx, switch2)
			So(err, ShouldBeNil)

			kvm2 := &ufspb.KVM{
				Name: "kvm-2",
			}
			_, err = registration.CreateKVM(ctx, kvm2)
			So(err, ShouldBeNil)

			rpm2 := &ufspb.RPM{
				Name: "rpm-2",
			}
			_, err = registration.CreateRPM(ctx, rpm2)
			So(err, ShouldBeNil)

			machine2 := &ufspb.Machine{
				Name: "machine-2",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{
						ChromePlatform: "chromePlatform-2",
						KvmInterface: &ufspb.KVMInterface{
							Kvm: "kvm-2",
						},
						RpmInterface: &ufspb.RPMInterface{
							Rpm: "rpm-2",
						},
					},
				},
			}
			resp, err := CreateMachine(ctx, machine2)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, machine2)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "machines/machine-2")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetName(), ShouldEqual, "machines/machine-2")
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetEventLabel(), ShouldEqual, "machine")
		})
	})
}

func TestUpdateMachine(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("UpdateMachines", t, func() {
		Convey("Update a non-existing machine", func() {
			machine := &ufspb.Machine{
				Name: "machine-10",
			}
			_, err := UpdateMachine(ctx, machine)
			So(err.Error(), ShouldContainSubstring, "There is no Machine with MachineID machine-10 in the system.")
		})

		Convey("Update new machine with non existing resource", func() {
			machine := &ufspb.Machine{
				Name: "machine-1",
			}
			_, err := registration.CreateMachine(ctx, machine)

			machine = &ufspb.Machine{
				Name: "machine-1",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{
						ChromePlatform: "chromePlatform-1",
					},
				},
			}
			_, err = UpdateMachine(ctx, machine)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Cannot update machine machine-1")
		})

		Convey("Update machine with existing resources", func() {
			machine := &ufspb.Machine{
				Name: "machine-2",
			}
			_, err := registration.CreateMachine(ctx, machine)

			chromePlatform2 := &ufspb.ChromePlatform{
				Name: "chromePlatform-2",
			}
			_, err = configuration.CreateChromePlatform(ctx, chromePlatform2)
			So(err, ShouldBeNil)

			machine = &ufspb.Machine{
				Name: "machine-2",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{
						ChromePlatform: "chromePlatform-2",
					},
				},
			}
			resp, err := UpdateMachine(ctx, machine)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, machine)
		})
	})
}

func TestDeleteMachine(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("DeleteMachine", t, func() {
		Convey("Delete machine by existing ID with machineLSE reference", func() {
			machine1 := &ufspb.Machine{
				Name: "machine-3",
			}
			_, err := registration.CreateMachine(ctx, machine1)
			So(err, ShouldBeNil)

			machineLSE1 := &ufspb.MachineLSE{
				Name:     "machinelse-1",
				Machines: []string{"machine-3"},
			}
			_, err = inventory.CreateMachineLSE(ctx, machineLSE1)
			So(err, ShouldBeNil)

			err = DeleteMachine(ctx, "machine-3")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, CannotDelete)

			resp, _ := registration.GetMachine(ctx, "machine-3")
			So(resp, ShouldNotBeNil)
			So(resp, ShouldResembleProto, machine1)

			// No changes are recorded as the deletion fails
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "machines/machine-3")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})

		Convey("Delete machine by existing ID without references", func() {
			machine2 := &ufspb.Machine{
				Name: "machine-4",
			}
			_, err := registration.CreateMachine(ctx, machine2)
			So(err, ShouldBeNil)

			err = DeleteMachine(ctx, "machine-4")
			So(err, ShouldBeNil)

			_, err = registration.GetMachine(ctx, "machine-4")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})

		Convey("Delete machine with nics and drac - happy path", func() {
			nic := &ufspb.Nic{
				Name: "nic-5",
			}
			_, err := registration.CreateNic(ctx, nic)
			So(err, ShouldBeNil)

			drac := &ufspb.Drac{
				Name: "drac-5",
			}
			_, err = registration.CreateDrac(ctx, drac)
			So(err, ShouldBeNil)

			machine := &ufspb.Machine{
				Name: "machine-5",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{
						Nics: []string{"nic-5"},
						Drac: "drac-5",
					},
				},
			}
			_, err = registration.CreateMachine(ctx, machine)
			So(err, ShouldBeNil)

			err = DeleteMachine(ctx, "machine-5")
			So(err, ShouldBeNil)

			_, err = registration.GetMachine(ctx, "machine-4")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)

			_, err = registration.GetNic(ctx, "nic-5")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)

			_, err = registration.GetDrac(ctx, "drac-5")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "machines/machine-4")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetName(), ShouldEqual, "machines/machine-4")
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRetire)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRetire)
			So(changes[0].GetEventLabel(), ShouldEqual, "machine")
		})
	})
}

func TestReplaceMachine(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("ReplaceMachines", t, func() {
		Convey("Repalce an old Machine with new machine with MachineLSE reference", func() {
			oldMachine1 := &ufspb.Machine{
				Name: "machine-4",
			}
			_, cerr := registration.CreateMachine(ctx, oldMachine1)
			So(cerr, ShouldBeNil)

			machineLSE1 := &ufspb.MachineLSE{
				Name:     "machinelse-1",
				Machines: []string{"machine-0", "machine-50", "machine-4", "machine-7"},
			}
			mresp, merr := inventory.CreateMachineLSE(ctx, machineLSE1)
			So(merr, ShouldBeNil)
			So(mresp, ShouldResembleProto, machineLSE1)

			newMachine2 := &ufspb.Machine{
				Name: "machine-100",
			}
			rresp, rerr := ReplaceMachine(ctx, oldMachine1, newMachine2)
			So(rerr, ShouldBeNil)
			So(rresp, ShouldResembleProto, newMachine2)

			mresp, merr = inventory.GetMachineLSE(ctx, "machinelse-1")
			So(merr, ShouldBeNil)
			So(mresp.GetMachines(), ShouldResemble, []string{"machine-0", "machine-50", "machine-100", "machine-7"})

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "machines/machine-4")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRetire)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRetire)
			So(changes[0].GetEventLabel(), ShouldEqual, "machine")

			changes, err = history.QueryChangesByPropertyName(ctx, "name", "machines/machine-100")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetEventLabel(), ShouldEqual, "machine")
		})

		Convey("Repalce an old Machine with already existing machine", func() {
			existingMachine1 := &ufspb.Machine{
				Name: "machine-105",
			}
			_, cerr := registration.CreateMachine(ctx, existingMachine1)
			So(cerr, ShouldBeNil)

			oldMachine1 := &ufspb.Machine{
				Name: "machine-5",
			}
			_, cerr = registration.CreateMachine(ctx, oldMachine1)
			So(cerr, ShouldBeNil)

			newMachine2 := &ufspb.Machine{
				Name: "machine-105",
			}
			rresp, rerr := ReplaceMachine(ctx, oldMachine1, newMachine2)
			So(rerr, ShouldNotBeNil)
			So(rresp, ShouldBeNil)
			So(rerr.Error(), ShouldContainSubstring, AlreadyExists)

			// No change are recorded as the replacement fails
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "machines/machine-5")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "machines/machine-105")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})
	})
}

func TestListMachines(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	machinesWithNic := make([]*ufspb.Machine, 0, 2)
	machinesWithNicAndDrac := make([]*ufspb.Machine, 0, 1)
	machines := make([]*ufspb.Machine, 0, 4)
	for i := 0; i < 4; i++ {
		machine := &ufspb.Machine{
			Name: fmt.Sprintf("machinefilter-%d", i),
			Device: &ufspb.Machine_ChromeBrowserMachine{
				ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
			},
		}
		if i == 0 {
			machine.GetChromeBrowserMachine().Drac = "drac-12"
		}
		if i%2 == 0 {
			machine.GetChromeBrowserMachine().Nics = []string{"nic-12"}
		}
		resp, _ := registration.CreateMachine(ctx, machine)
		if i == 0 {
			machinesWithNicAndDrac = append(machinesWithNicAndDrac, resp)
		}
		if i%2 == 0 {
			machinesWithNic = append(machinesWithNic, resp)
		}
		machines = append(machines, resp)
	}
	Convey("ListMachines", t, func() {
		Convey("List Machines - filter invalid - error", func() {
			_, _, err := ListMachines(ctx, 5, "", "invalid=mx-1", false)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Invalid field name invalid")
		})

		Convey("List Machines - filter nic - happy path", func() {
			resp, _, _ := ListMachines(ctx, 5, "", "nic=nic-12", false)
			So(resp, ShouldNotBeNil)
			So(resp, ShouldResembleProto, machinesWithNic)
		})

		Convey("List Machines - filter nic AND drac - happy path", func() {
			resp, _, _ := ListMachines(ctx, 5, "", "nic=nic-12 & drac=drac-12", false)
			So(resp, ShouldNotBeNil)
			So(resp, ShouldResembleProto, machinesWithNicAndDrac)
		})

		Convey("ListMachines - Full listing - happy path", func() {
			resp, _, _ := ListMachines(ctx, 5, "", "", false)
			So(resp, ShouldNotBeNil)
			So(resp, ShouldResembleProto, machines)
		})
	})
}
