// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"testing"

	ufspb "infra/unifiedfleet/api/v1/proto"
	"infra/unifiedfleet/app/model/configuration"
	. "infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/model/inventory"
	"infra/unifiedfleet/app/model/registration"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
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
			resp, cerr := CreateMachine(ctx, oldMachine1)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, oldMachine1)

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
		})

		Convey("Repalce an old Machine with already existing machine", func() {
			existingMachine1 := &ufspb.Machine{
				Name: "machine-105",
			}
			resp, cerr := CreateMachine(ctx, existingMachine1)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, existingMachine1)

			oldMachine1 := &ufspb.Machine{
				Name: "machine-5",
			}
			resp, cerr = CreateMachine(ctx, oldMachine1)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, oldMachine1)

			newMachine2 := &ufspb.Machine{
				Name: "machine-105",
			}
			rresp, rerr := ReplaceMachine(ctx, oldMachine1, newMachine2)
			So(rerr, ShouldNotBeNil)
			So(rresp, ShouldBeNil)
			So(rerr.Error(), ShouldContainSubstring, AlreadyExists)
		})
	})
}
