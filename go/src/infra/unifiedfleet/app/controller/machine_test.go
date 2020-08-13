// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
	"google.golang.org/genproto/protobuf/field_mask"

	ufspb "infra/unifiedfleet/api/v1/proto"
	"infra/unifiedfleet/app/model/configuration"
	. "infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/model/history"
	"infra/unifiedfleet/app/model/inventory"
	"infra/unifiedfleet/app/model/registration"
	"infra/unifiedfleet/app/model/state"
)

func TestMachineRegistration(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("TestMachineRegistration", t, func() {
		Convey("Register machine with already existing machine, nic and drac", func() {
			nic := &ufspb.Nic{
				Name:    "nic-1",
				Machine: "machine-1",
			}
			_, err := registration.CreateNic(ctx, nic)
			So(err, ShouldBeNil)

			drac := &ufspb.Drac{
				Name:    "drac-1",
				Machine: "machine-1",
			}
			_, err = registration.CreateDrac(ctx, drac)
			So(err, ShouldBeNil)

			machine := &ufspb.Machine{
				Name: "machine-1",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{
						NicObjects: []*ufspb.Nic{{
							Name:    "nic-1",
							Machine: "machine-1",
						}},
						DracObject: &ufspb.Drac{
							Name:    "drac-1",
							Machine: "machine-1",
						},
					},
				},
			}
			_, err = registration.CreateMachine(ctx, machine)
			So(err, ShouldBeNil)

			_, err = MachineRegistration(ctx, machine)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring,
				"Machine machine-1 already exists in the system.\n"+
					"Nic nic-1 already exists in the system.\n"+
					"Drac drac-1 already exists in the system.\n")

			// No changes are recorded as the registration fails
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "machines/machine-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "nics/nic-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "dracs/drac-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})

		Convey("Register machine with invalid machine(referencing non existing resources)", func() {
			machine := &ufspb.Machine{
				Name: "machine-3",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{
						ChromePlatform: "chromePlatform-3",
						KvmInterface: &ufspb.KVMInterface{
							Kvm: "kvm-3",
						},
						RpmInterface: &ufspb.RPMInterface{
							Rpm: "rpm-3",
						},
					},
				},
			}
			_, err := MachineRegistration(ctx, machine)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Cannot create machine machine-3:\n"+
				"There is no KVM with KVMID kvm-3 in the system.\n"+
				"There is no RPM with RPMID rpm-3 in the system.\n"+
				"There is no ChromePlatform with ChromePlatformID chromePlatform-3 in the system.")

			// No changes are recorded as the registration fails
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "machines/machine-3")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})

		Convey("Register browser machine with invalid nic(referencing non existing resources)", func() {
			machine := &ufspb.Machine{
				Name: "machine-3",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{
						NicObjects: []*ufspb.Nic{{
							Name: "nic-2",
							SwitchInterface: &ufspb.SwitchInterface{
								Switch: "switch-1",
							},
						}},
					},
				},
			}
			_, err := MachineRegistration(ctx, machine)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Cannot create machine machine-3:\n"+
				"There is no Switch with SwitchID switch-1 in the system.")

			// No changes are recorded as the registration fails
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "machines/machine-3")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "nics/nic-2")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})

		Convey("Register browser machine with invalid drac(referencing non existing resources)", func() {
			machine := &ufspb.Machine{
				Name: "machine-3",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{
						DracObject: &ufspb.Drac{
							Name: "drac-2",
							SwitchInterface: &ufspb.SwitchInterface{
								Switch: "switch-1",
							},
						},
					},
				},
			}
			_, err := MachineRegistration(ctx, machine)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Cannot create machine machine-3:\n"+
				"There is no Switch with SwitchID switch-1 in the system.")

			// No changes are recorded as the registration fails
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "machines/machine-3")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "dracs/drac-2")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})

		Convey("Register browser machine happy path", func() {
			nics := []*ufspb.Nic{{
				Name: "nic-browser-3",
			}}
			drac := &ufspb.Drac{
				Name: "drac-browser-3",
			}
			machine := &ufspb.Machine{
				Name: "machine-browser-3",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{
						NicObjects: nics,
						DracObject: drac,
					},
				},
			}
			m, err := MachineRegistration(ctx, machine)
			//nic.Name = "machine-browser-3" + "nic-browser-3"
			So(err, ShouldBeNil)
			So(m, ShouldResembleProto, machine)
			s, err := state.GetStateRecord(ctx, "machines/machine-browser-3")
			So(err, ShouldBeNil)
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_REGISTERED)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "machines/machine-browser-3")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetEventLabel(), ShouldEqual, "machine")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "nics/nic-browser-3")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetEventLabel(), ShouldEqual, "nic")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "dracs/drac-browser-3")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetEventLabel(), ShouldEqual, "drac")
		})

		Convey("Register OS machine happy path", func() {
			machine := &ufspb.Machine{
				Name: "machine-os-3",
				Device: &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: &ufspb.ChromeOSMachine{},
				},
			}
			m, err := MachineRegistration(ctx, machine)
			So(err, ShouldBeNil)
			So(m, ShouldNotBeNil)
			So(m, ShouldResembleProto, machine)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "machines/machine-os-3")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
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
			_, err := UpdateMachine(ctx, machine, nil)
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
			_, err = UpdateMachine(ctx, machine, nil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Cannot update machine machine-1")
		})

		Convey("Update machine with existing resources", func() {
			machine := &ufspb.Machine{
				Name: "machine-2",
				Location: &ufspb.Location{
					Lab: ufspb.Lab_LAB_CHROMEOS_ATLANTIS,
				},
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
				},
			}
			_, err := registration.CreateMachine(ctx, machine)
			registration.CreateNic(ctx, &ufspb.Nic{
				Name:    "nic-update-lab",
				Lab:     ufspb.Lab_LAB_CHROMEOS_ATLANTIS.String(),
				Machine: "machine-2",
			})
			registration.CreateDrac(ctx, &ufspb.Drac{
				Name:    "drac-update-lab",
				Lab:     ufspb.Lab_LAB_CHROMEOS_ATLANTIS.String(),
				Machine: "machine-2",
			})
			inventory.CreateMachineLSE(ctx, &ufspb.MachineLSE{
				Name: "lse-update-lab",
				Lab:  ufspb.Lab_LAB_CHROMEOS_ATLANTIS.String(),
				Lse: &ufspb.MachineLSE_ChromeBrowserMachineLse{
					ChromeBrowserMachineLse: &ufspb.ChromeBrowserMachineLSE{
						VmCapacity: 12,
					},
				},
				Machines: []string{"machine-2"},
			})
			inventory.BatchUpdateVMs(ctx, []*ufspb.VM{
				{
					Name:         "vm-update-lab",
					Lab:          ufspb.Lab_LAB_CHROMEOS_ATLANTIS.String(),
					MachineLseId: "lse-update-lab",
				},
			})

			chromePlatform2 := &ufspb.ChromePlatform{
				Name: "chromePlatform-2",
			}
			_, err = configuration.CreateChromePlatform(ctx, chromePlatform2)
			So(err, ShouldBeNil)

			machine = &ufspb.Machine{
				Name: "machine-2",
				Location: &ufspb.Location{
					Lab: ufspb.Lab_LAB_CHROMEOS_SANTIAM,
				},
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{
						ChromePlatform: "chromePlatform-2",
					},
				},
			}
			resp, err := UpdateMachine(ctx, machine, nil)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, machine)
			lse, err := inventory.GetMachineLSE(ctx, "lse-update-lab")
			So(err, ShouldBeNil)
			So(lse.GetLab(), ShouldEqual, ufspb.Lab_LAB_CHROMEOS_SANTIAM.String())
			nic, err := registration.GetNic(ctx, "nic-update-lab")
			So(err, ShouldBeNil)
			So(nic.GetLab(), ShouldEqual, ufspb.Lab_LAB_CHROMEOS_SANTIAM.String())
			drac, err := registration.GetDrac(ctx, "drac-update-lab")
			So(err, ShouldBeNil)
			So(drac.GetLab(), ShouldEqual, ufspb.Lab_LAB_CHROMEOS_SANTIAM.String())
			vm, err := inventory.GetVM(ctx, "vm-update-lab")
			So(err, ShouldBeNil)
			So(vm.GetLab(), ShouldEqual, ufspb.Lab_LAB_CHROMEOS_SANTIAM.String())
		})

		Convey("Partial Update machine", func() {
			machine := &ufspb.Machine{
				Name: "machine-3",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{
						ChromePlatform: "chromePlatform-3",
						KvmInterface: &ufspb.KVMInterface{
							Kvm: "kvm-3",
						},
					},
				},
			}
			_, err := registration.CreateMachine(ctx, machine)
			So(err, ShouldBeNil)

			chromePlatform := &ufspb.ChromePlatform{
				Name: "chromePlatform-4",
			}
			_, err = configuration.CreateChromePlatform(ctx, chromePlatform)
			So(err, ShouldBeNil)

			machine1 := &ufspb.Machine{
				Name: "machine-3",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{
						ChromePlatform: "chromePlatform-4",
					},
				},
			}
			resp, err := UpdateMachine(ctx, machine1, &field_mask.FieldMask{Paths: []string{"platform"}})
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp.GetChromeBrowserMachine().GetChromePlatform(), ShouldResemble, "chromePlatform-4")
			So(resp.GetChromeBrowserMachine().GetKvmInterface().GetKvm(), ShouldResemble, "kvm-3")
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
				Name:    "nic-5",
				Machine: "machine-5",
			}
			_, err := registration.CreateNic(ctx, nic)
			So(err, ShouldBeNil)

			drac := &ufspb.Drac{
				Name:    "drac-5",
				Machine: "machine-5",
			}
			_, err = registration.CreateDrac(ctx, drac)
			So(err, ShouldBeNil)

			machine := &ufspb.Machine{
				Name: "machine-5",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
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
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "machines/machine-4")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			So(msgs[0].Delete, ShouldBeTrue)
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
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "machines/machine-4")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			So(msgs[0].Delete, ShouldBeTrue)
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "machines/machine-100")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			So(msgs[0].Delete, ShouldBeFalse)
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
	machinesWithChromeplatform := make([]*ufspb.Machine, 0, 2)
	machines := make([]*ufspb.Machine, 0, 4)
	for i := 0; i < 4; i++ {
		machine := &ufspb.Machine{
			Name: fmt.Sprintf("machinefilter-%d", i),
			Device: &ufspb.Machine_ChromeBrowserMachine{
				ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
			},
		}
		if i%2 == 0 {
			machine.GetChromeBrowserMachine().ChromePlatform = "cp-12"
		}
		resp, _ := registration.CreateMachine(ctx, machine)
		if i%2 == 0 {
			machinesWithChromeplatform = append(machinesWithChromeplatform, resp)
		}
		machines = append(machines, resp)
	}
	Convey("ListMachines", t, func() {
		Convey("List Machines - filter invalid - error", func() {
			_, _, err := ListMachines(ctx, 5, "", "invalid=mx-1", false)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Invalid field name invalid")
		})

		Convey("List Machines - filter chromeplatform - happy path", func() {
			resp, _, _ := ListMachines(ctx, 5, "", "platform=cp-12", false)
			So(resp, ShouldNotBeNil)
			So(resp, ShouldResembleProto, machinesWithChromeplatform)
		})

		Convey("ListMachines - Full listing - happy path", func() {
			resp, _, _ := ListMachines(ctx, 5, "", "", false)
			So(resp, ShouldNotBeNil)
			So(resp, ShouldResembleProto, machines)
		})
	})
}
