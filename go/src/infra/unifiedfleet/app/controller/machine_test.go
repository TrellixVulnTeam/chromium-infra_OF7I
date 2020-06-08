// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"testing"

	proto "infra/unifiedfleet/api/v1/proto"
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
		Convey("Create new machine with non existing ChromePlatform", func() {
			machine1 := &proto.Machine{
				Name: "machine-1",
				Device: &proto.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &proto.ChromeBrowserMachine{
						ChromePlatform: "chromePlatform-1",
					},
				},
			}
			resp, err := CreateMachine(ctx, machine1)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, CannotCreate)
		})

		Convey("Create new machine with non existing resources", func() {
			machine3 := &proto.Machine{
				Name: "machine-3",
				Device: &proto.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &proto.ChromeBrowserMachine{
						ChromePlatform: "chromePlatform-3",
						Nics:           []string{"nic-3"},
						Drac:           "drac-3",
						KvmInterface: &proto.KVMInterface{
							Kvm: "kvm-3",
						},
						RpmInterface: &proto.RPMInterface{
							Rpm: "rpm-3",
						},
					},
				},
			}
			resp, err := CreateMachine(ctx, machine3)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Cannot create")
		})

		Convey("Create new machine with existing resources", func() {
			chromePlatform2 := &proto.ChromePlatform{
				Name: "chromePlatform-2",
			}
			presp, err := configuration.CreateChromePlatform(ctx, chromePlatform2)
			So(err, ShouldBeNil)
			So(presp, ShouldResembleProto, chromePlatform2)

			nic2 := &proto.Nic{
				Name: "nic-2",
				SwitchInterface: &proto.SwitchInterface{
					Switch: "switch-2",
				},
			}
			nresp, err := registration.CreateNic(ctx, nic2)
			So(err, ShouldBeNil)
			So(nresp, ShouldResembleProto, nic2)

			drac2 := &proto.Drac{
				Name: "drac-2",
			}
			dresp, err := registration.CreateDrac(ctx, drac2)
			So(err, ShouldBeNil)
			So(dresp, ShouldResembleProto, drac2)

			switch2 := &proto.Switch{
				Name: "switch-2",
			}
			sresp, err := registration.CreateSwitch(ctx, switch2)
			So(err, ShouldBeNil)
			So(sresp, ShouldResembleProto, switch2)

			kvm2 := &proto.KVM{
				Name: "kvm-2",
			}
			kresp, err := registration.CreateKVM(ctx, kvm2)
			So(err, ShouldBeNil)
			So(kresp, ShouldResembleProto, kvm2)

			rpm2 := &proto.RPM{
				Name: "rpm-2",
			}
			rresp, err := registration.CreateRPM(ctx, rpm2)
			So(err, ShouldBeNil)
			So(rresp, ShouldResembleProto, rpm2)

			machine2 := &proto.Machine{
				Name: "machine-2",
				Device: &proto.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &proto.ChromeBrowserMachine{
						ChromePlatform: "chromePlatform-2",
						Nics:           []string{"nic-2"},
						Drac:           "drac-2",
						KvmInterface: &proto.KVMInterface{
							Kvm: "kvm-2",
						},
						RpmInterface: &proto.RPMInterface{
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

func TestDeleteMachine(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("DeleteMachine", t, func() {
		Convey("Delete machine by existing ID with machineLSE reference", func() {
			machine1 := &proto.Machine{
				Name: "machine-3",
			}
			resp, cerr := registration.CreateMachine(ctx, machine1)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, machine1)

			machineLSE1 := &proto.MachineLSE{
				Name:     "machinelse-1",
				Machines: []string{"machine-3"},
			}
			mresp, merr := inventory.CreateMachineLSE(ctx, machineLSE1)
			So(merr, ShouldBeNil)
			So(mresp, ShouldResembleProto, machineLSE1)

			err := DeleteMachine(ctx, "machine-3")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, CannotDelete)

			resp, cerr = registration.GetMachine(ctx, "machine-3")
			So(resp, ShouldNotBeNil)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, machine1)
		})
		Convey("Delete machine by existing ID without references", func() {
			machine2 := &proto.Machine{
				Name: "machine-4",
			}
			resp, cerr := registration.CreateMachine(ctx, machine2)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, machine2)
			err := DeleteMachine(ctx, "machine-4")
			So(err, ShouldBeNil)
			res, err := registration.GetMachine(ctx, "machine-4")
			So(res, ShouldBeNil)
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
			oldMachine1 := &proto.Machine{
				Name: "machine-4",
			}
			resp, cerr := CreateMachine(ctx, oldMachine1)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, oldMachine1)

			machineLSE1 := &proto.MachineLSE{
				Name:     "machinelse-1",
				Machines: []string{"machine-0", "machine-50", "machine-4", "machine-7"},
			}
			mresp, merr := inventory.CreateMachineLSE(ctx, machineLSE1)
			So(merr, ShouldBeNil)
			So(mresp, ShouldResembleProto, machineLSE1)

			newMachine2 := &proto.Machine{
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
			existingMachine1 := &proto.Machine{
				Name: "machine-105",
			}
			resp, cerr := CreateMachine(ctx, existingMachine1)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, existingMachine1)

			oldMachine1 := &proto.Machine{
				Name: "machine-5",
			}
			resp, cerr = CreateMachine(ctx, oldMachine1)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, oldMachine1)

			newMachine2 := &proto.Machine{
				Name: "machine-105",
			}
			rresp, rerr := ReplaceMachine(ctx, oldMachine1, newMachine2)
			So(rerr, ShouldNotBeNil)
			So(rresp, ShouldBeNil)
			So(rerr.Error(), ShouldContainSubstring, AlreadyExists)
		})
	})
}
