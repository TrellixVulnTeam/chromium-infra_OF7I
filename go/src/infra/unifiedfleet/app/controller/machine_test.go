// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
	"go.chromium.org/luci/server/auth"
	"go.chromium.org/luci/server/auth/authtest"
	"google.golang.org/genproto/protobuf/field_mask"

	ufspb "infra/unifiedfleet/api/v1/models"
	"infra/unifiedfleet/app/model/configuration"
	. "infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/model/history"
	"infra/unifiedfleet/app/model/inventory"
	"infra/unifiedfleet/app/model/registration"
	"infra/unifiedfleet/app/model/state"
	"infra/unifiedfleet/app/util"
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

		Convey("Register browser machine - duplicated kvm interface", func() {
			_, err := registration.CreateKVM(ctx, &ufspb.KVM{
				Name: "kvm-browser-duplicate",
			})
			So(err, ShouldBeNil)
			machine := &ufspb.Machine{
				Name: "machine-browser-duplicate1",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{
						KvmInterface: &ufspb.KVMInterface{
							Kvm:      "kvm-browser-duplicate",
							PortName: "A1",
						},
					},
				},
			}
			m, err := MachineRegistration(ctx, machine)
			So(err, ShouldBeNil)
			So(m, ShouldResembleProto, machine)

			machine2 := &ufspb.Machine{
				Name: "machine-browser-duplicate2",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{
						KvmInterface: &ufspb.KVMInterface{
							Kvm:      "kvm-browser-duplicate",
							PortName: "A1",
						},
					},
				},
			}
			_, err = MachineRegistration(ctx, machine2)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "kvm port A1 of kvm-browser-duplicate is already occupied")
		})

		Convey("Register browser machine happy path", func() {
			_, err := registration.CreateKVM(ctx, &ufspb.KVM{
				Name: "kvm-browser-3",
			})
			So(err, ShouldBeNil)
			nics := []*ufspb.Nic{{
				Name: "nic-browser-3",
			}}
			drac := &ufspb.Drac{
				Name: "drac-browser-3",
			}
			machine := &ufspb.Machine{
				Name:         "machine-browser-3",
				SerialNumber: "machine-browser-3-serial-number",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{
						NicObjects: nics,
						DracObject: drac,
						KvmInterface: &ufspb.KVMInterface{
							Kvm:      "kvm-browser-3",
							PortName: "A1",
						},
					},
				},
				Realm: util.BrowserLabAdminRealm,
			}
			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.RegistrationsCreate, util.BrowserLabAdminRealm)
			m, err := MachineRegistration(ctx, machine)
			So(err, ShouldBeNil)
			So(m, ShouldResembleProto, machine)
			s, err := state.GetStateRecord(ctx, "machines/machine-browser-3")
			So(err, ShouldBeNil)
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_REGISTERED)
			dr, err := inventory.GetMachineLSEDeployment(ctx, m.GetSerialNumber())
			So(err, ShouldBeNil)
			So(dr.GetHostname(), ShouldEqual, util.GetHostnameWithNoHostPrefix(m.GetSerialNumber()))
			So(dr.GetDeploymentIdentifier(), ShouldBeEmpty)
			So(dr.GetConfigsToPush(), ShouldBeNil)

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
			asset := &ufspb.Asset{
				Name: "machine-os-3",
				Location: &ufspb.Location{
					Zone: ufspb.Zone_ZONE_CHROMEOS6,
					Rack: "chromeos6-test",
				},
			}
			r := mockRack("chromeos6-test", "2", ufspb.Zone_ZONE_CHROMEOS6)
			_, err := registration.CreateRack(ctx, r)
			So(err, ShouldBeNil)
			_, err = AssetRegistration(ctx, asset)
			So(err, ShouldBeNil)
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

		Convey("Register machine - permission denied: same realm and no create permission", func() {
			machine := &ufspb.Machine{
				Name: "machine-os-4",
				Device: &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: &ufspb.ChromeOSMachine{},
				},
				Realm: util.AtlLabAdminRealm,
			}
			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.RegistrationsGet, util.AtlLabAdminRealm)
			_, err := MachineRegistration(ctx, machine)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, PermissionDenied)
		})

		Convey("Register machine - permission denied: different realm", func() {
			machine := &ufspb.Machine{
				Name: "machine-os-5",
				Device: &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: &ufspb.ChromeOSMachine{},
				},
				Realm: util.AtlLabAdminRealm,
			}
			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.RegistrationsCreate, util.BrowserLabAdminRealm)
			_, err := MachineRegistration(ctx, machine)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, PermissionDenied)
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
			So(err.Error(), ShouldContainSubstring, NotFound)
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
					Zone: ufspb.Zone_ZONE_CHROMEOS3,
				},
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
				},
				Realm: util.BrowserLabAdminRealm,
			}
			_, err := registration.CreateMachine(ctx, machine)
			registration.CreateNic(ctx, &ufspb.Nic{
				Name:    "nic-update-zone",
				Zone:    ufspb.Zone_ZONE_CHROMEOS3.String(),
				Machine: "machine-2",
			})
			registration.CreateDrac(ctx, &ufspb.Drac{
				Name:    "drac-update-zone",
				Zone:    ufspb.Zone_ZONE_CHROMEOS3.String(),
				Machine: "machine-2",
			})
			inventory.CreateMachineLSE(ctx, &ufspb.MachineLSE{
				Name: "lse-update-zone",
				Zone: ufspb.Zone_ZONE_CHROMEOS3.String(),
				Lse: &ufspb.MachineLSE_ChromeBrowserMachineLse{
					ChromeBrowserMachineLse: &ufspb.ChromeBrowserMachineLSE{
						VmCapacity: 12,
					},
				},
				Machines: []string{"machine-2"},
			})
			inventory.BatchUpdateVMs(ctx, []*ufspb.VM{
				{
					Name:         "vm-update-zone",
					Zone:         ufspb.Zone_ZONE_CHROMEOS3.String(),
					MachineLseId: "lse-update-zone",
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
					Zone: ufspb.Zone_ZONE_CHROMEOS2,
				},
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{
						ChromePlatform: "chromePlatform-2",
					},
				},
				Realm: util.BrowserLabAdminRealm,
			}
			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.RegistrationsUpdate, util.BrowserLabAdminRealm)
			resp, err := UpdateMachine(ctx, machine, nil)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, machine)
			lse, err := inventory.GetMachineLSE(ctx, "lse-update-zone")
			So(err, ShouldBeNil)
			So(lse.GetZone(), ShouldEqual, ufspb.Zone_ZONE_CHROMEOS2.String())
			nic, err := registration.GetNic(ctx, "nic-update-zone")
			So(err, ShouldBeNil)
			So(nic.GetZone(), ShouldEqual, ufspb.Zone_ZONE_CHROMEOS2.String())
			drac, err := registration.GetDrac(ctx, "drac-update-zone")
			So(err, ShouldBeNil)
			So(drac.GetZone(), ShouldEqual, ufspb.Zone_ZONE_CHROMEOS2.String())
			vm, err := inventory.GetVM(ctx, "vm-update-zone")
			So(err, ShouldBeNil)
			So(vm.GetZone(), ShouldEqual, ufspb.Zone_ZONE_CHROMEOS2.String())
		})

		Convey("Update machine kvm", func() {
			machine := &ufspb.Machine{
				Name: "machine-update-kvm",
				Location: &ufspb.Location{
					Zone: ufspb.Zone_ZONE_CHROMEOS3,
				},
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
				},
			}
			_, err := registration.CreateMachine(ctx, machine)

			_, err = registration.CreateKVM(ctx, &ufspb.KVM{
				Name: "kvm-update",
			})
			So(err, ShouldBeNil)

			machine = &ufspb.Machine{
				Name: "machine-update-kvm",
				Location: &ufspb.Location{
					Zone: ufspb.Zone_ZONE_CHROMEOS2,
				},
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{
						KvmInterface: &ufspb.KVMInterface{
							Kvm:      "kvm-update",
							PortName: "A1",
						},
					},
				},
			}
			resp, err := UpdateMachine(ctx, machine, nil)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, machine)
		})

		Convey("Update machine serial number", func() {
			machine := &ufspb.Machine{
				Name:         "machine-full-update-serial",
				SerialNumber: "old-serial-full-update",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
				},
			}
			_, err := registration.CreateMachine(ctx, machine)
			So(err, ShouldBeNil)
			_, err = inventory.UpdateMachineLSEDeployments(ctx, []*ufspb.MachineLSEDeployment{util.GetEmtpyDeploymentRecord(machine.GetSerialNumber())})
			So(err, ShouldBeNil)

			machine1 := &ufspb.Machine{
				Name:         "machine-full-update-serial",
				SerialNumber: "serial-full-update",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
				},
			}
			resp, err := UpdateMachine(ctx, machine1, nil)
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp.GetSerialNumber(), ShouldResemble, "serial-full-update")
			dr, err := inventory.GetMachineLSEDeployment(ctx, "serial-full-update")
			So(err, ShouldBeNil)
			So(dr.GetHostname(), ShouldEqual, util.GetHostnameWithNoHostPrefix("serial-full-update"))
			_, err = inventory.GetMachineLSEDeployment(ctx, "old-serial-full-update")
			So(err, ShouldNotBeNil)
		})

		Convey("Partial Update machine", func() {
			machine := &ufspb.Machine{
				Name: "machine-3",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{
						ChromePlatform: "chromePlatform-3",
						KvmInterface: &ufspb.KVMInterface{
							Kvm:      "kvm-3",
							PortName: "A1",
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
			_, err = registration.CreateKVM(ctx, &ufspb.KVM{
				Name: "kvm-4",
			})
			So(err, ShouldBeNil)

			machine1 := &ufspb.Machine{
				Name: "machine-3",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{
						ChromePlatform: "chromePlatform-4",
						KvmInterface: &ufspb.KVMInterface{
							Kvm:      "kvm-4",
							PortName: "A2",
						},
					},
				},
			}
			resp, err := UpdateMachine(ctx, machine1, &field_mask.FieldMask{Paths: []string{"platform", "kvm"}})
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp.GetChromeBrowserMachine().GetChromePlatform(), ShouldResemble, "chromePlatform-4")
			So(resp.GetChromeBrowserMachine().GetKvmInterface().GetKvm(), ShouldResemble, "kvm-4")
			So(resp.GetChromeBrowserMachine().GetKvmInterface().GetPortName(), ShouldResemble, "A1")
		})

		Convey("Partial Update machine - update serial number", func() {
			machine := &ufspb.Machine{
				Name: "machine-update-serial",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
				},
			}
			_, err := registration.CreateMachine(ctx, machine)
			So(err, ShouldBeNil)
			machine1 := &ufspb.Machine{
				Name:         "machine-update-serial",
				SerialNumber: "serial-update",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
				},
			}
			resp, err := UpdateMachine(ctx, machine1, &field_mask.FieldMask{Paths: []string{"serialNumber"}})
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp.GetSerialNumber(), ShouldResemble, "serial-update")
			dr, err := inventory.GetMachineLSEDeployment(ctx, "serial-update")
			So(err, ShouldBeNil)
			So(dr.GetHostname(), ShouldEqual, util.GetHostnameWithNoHostPrefix("serial-update"))
		})

		Convey("Partial Update machine - duplicated kvm", func() {
			_, err := registration.CreateKVM(ctx, &ufspb.KVM{
				Name: "kvm-update-duplicate1",
			})
			So(err, ShouldBeNil)
			machine := &ufspb.Machine{
				Name: "machine-update-duplicate1",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{
						KvmInterface: &ufspb.KVMInterface{
							Kvm:      "kvm-update-duplicate1",
							PortName: "A1",
						},
					},
				},
			}
			_, err = registration.CreateMachine(ctx, machine)
			So(err, ShouldBeNil)
			machine2 := &ufspb.Machine{
				Name: "machine-update-duplicate2",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{
						KvmInterface: &ufspb.KVMInterface{
							Kvm:      "kvm-update-duplicate2",
							PortName: "A1",
						},
					},
				},
			}
			_, err = registration.CreateMachine(ctx, machine2)
			So(err, ShouldBeNil)

			machine1 := &ufspb.Machine{
				Name: "machine-update-duplicate2",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{
						KvmInterface: &ufspb.KVMInterface{
							Kvm:      "kvm-update-duplicate1",
							PortName: "A1",
						},
					},
				},
			}
			_, err = UpdateMachine(ctx, machine1, &field_mask.FieldMask{Paths: []string{"kvm"}})
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "kvm port A1 of kvm-update-duplicate1 is already occupied")
		})

		Convey("Update machine - permission denied: same realm and no update permission", func() {
			machine := &ufspb.Machine{
				Name: "machine-os-4",
				Device: &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: &ufspb.ChromeOSMachine{},
				},
				Realm: util.AtlLabAdminRealm,
			}
			_, err := registration.CreateMachine(ctx, machine)
			So(err, ShouldBeNil)

			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.RegistrationsCreate, util.AtlLabAdminRealm)
			_, err = UpdateMachine(ctx, machine, nil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, PermissionDenied)
		})

		Convey("Update machine - permission denied: different realm", func() {
			machine := &ufspb.Machine{
				Name: "machine-os-5",
				Device: &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: &ufspb.ChromeOSMachine{},
				},
				Realm: util.AtlLabAdminRealm,
			}
			_, err := registration.CreateMachine(ctx, machine)
			So(err, ShouldBeNil)

			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.RegistrationsUpdate, util.BrowserLabAdminRealm)
			_, err = UpdateMachine(ctx, machine, nil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, PermissionDenied)
		})

		Convey("Update machine(realm name) - different realm with permission success", func() {
			machine := &ufspb.Machine{
				Name: "machine-os-6",
				Device: &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: &ufspb.ChromeOSMachine{},
				},
				Realm: util.AtlLabAdminRealm,
			}
			_, err := registration.CreateMachine(ctx, machine)
			So(err, ShouldBeNil)

			machine.Realm = util.BrowserLabAdminRealm
			ctx := auth.WithState(ctx, &authtest.FakeState{
				Identity: "user:user@example.com",
				FakeDB: authtest.NewFakeDB(
					authtest.MockMembership("user:user@example.com", "user"),
					authtest.MockPermission("user:user@example.com", util.AtlLabAdminRealm, util.RegistrationsUpdate),
					authtest.MockPermission("user:user@example.com", util.BrowserLabAdminRealm, util.RegistrationsUpdate),
				),
			})
			resp, err := UpdateMachine(ctx, machine, nil)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, machine)
		})

		Convey("Update machine(realm name) - permission denied: different realm without permission", func() {
			machine := &ufspb.Machine{
				Name: "machine-os-7",
				Device: &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: &ufspb.ChromeOSMachine{},
				},
				Realm: util.AtlLabAdminRealm,
			}
			_, err := registration.CreateMachine(ctx, machine)
			So(err, ShouldBeNil)

			machine.Realm = util.BrowserLabAdminRealm
			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.RegistrationsUpdate, util.AtlLabAdminRealm)
			_, err = UpdateMachine(ctx, machine, nil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, PermissionDenied)
		})
	})
}

func TestUpdateDutMeta(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("UpdateDutLab for an OS machine", t, func() {
		Convey("Update a non-OS machine", func() {
			_, err := registration.CreateMachine(ctx, &ufspb.Machine{
				Name: "machine-dutmeta-1",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
				},
			})
			So(err, ShouldBeNil)

			err = UpdateDutMeta(ctx, &ufspb.DutMeta{
				ChromeosDeviceId: "machine-dutmeta-1",
				Hostname:         "machinelse-labmeta-1",
				SerialNumber:     "fake-serial",
			})
			// Update is skipped without error
			So(err, ShouldBeNil)
		})

		Convey("Update a OS machine - happy path", func() {
			machine := &ufspb.Machine{
				Name: "machine-dutmeta-2",
				Device: &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: &ufspb.ChromeOSMachine{},
				},
			}
			req, err := registration.CreateMachine(ctx, machine)
			So(err, ShouldBeNil)
			So(req.GetSerialNumber(), ShouldBeEmpty)
			So(req.GetChromeosMachine().GetHwid(), ShouldBeEmpty)
			So(req.GetChromeosMachine().GetSku(), ShouldBeEmpty)

			err = UpdateDutMeta(ctx, &ufspb.DutMeta{
				ChromeosDeviceId: "machine-dutmeta-2",
				Hostname:         "machinelse-dutmeta-2",
				SerialNumber:     "fake-serial",
				HwID:             "fake-hwid",
				DeviceSku:        "fake-devicesku",
			})
			So(err, ShouldBeNil)
			req, err = registration.GetMachine(ctx, "machine-dutmeta-2")
			So(err, ShouldBeNil)
			So(req.GetSerialNumber(), ShouldEqual, "fake-serial")
			So(req.GetChromeosMachine().GetHwid(), ShouldEqual, "fake-hwid")
			So(req.GetChromeosMachine().GetSku(), ShouldEqual, "fake-devicesku")
		})
	})
}

func TestDeleteMachine(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("DeleteMachine", t, func() {
		Convey("Delete machine by existing ID with machineLSE reference", func() {
			machine1 := &ufspb.Machine{
				Name:  "machine-3",
				Realm: util.AtlLabAdminRealm,
			}
			_, err := registration.CreateMachine(ctx, machine1)
			So(err, ShouldBeNil)

			machineLSE1 := &ufspb.MachineLSE{
				Name:     "machinelse-1",
				Machines: []string{"machine-3"},
			}
			_, err = inventory.CreateMachineLSE(ctx, machineLSE1)
			So(err, ShouldBeNil)

			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.RegistrationsDelete, util.AtlLabAdminRealm)
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

			_, err = configuration.BatchUpdateDHCPs(ctx, []*ufspb.DHCPConfig{
				{
					Hostname: "drac-5",
					Ip:       "1.2.3.5",
				},
			})
			So(err, ShouldBeNil)
			_, err = configuration.BatchUpdateIPs(ctx, []*ufspb.IP{
				{
					Id:       "ip3",
					Occupied: true,
					Ipv4Str:  "1.2.3.5",
					Vlan:     "fake_vlan",
					Ipv4:     uint32(100),
				},
			})
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
			_, err = configuration.GetDHCPConfig(ctx, "drac-5")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
			resIPs, err := configuration.QueryIPByPropertyName(ctx, map[string]string{"ipv4_str": "1.2.3.5"})
			So(err, ShouldBeNil)
			So(resIPs, ShouldHaveLength, 1)
			So(resIPs[0].Occupied, ShouldBeFalse)

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

		Convey("Delete machine - Permission denied: same realm with no delete permission", func() {
			machine2 := &ufspb.Machine{
				Name:  "machine-6",
				Realm: util.AtlLabAdminRealm,
			}
			_, err := registration.CreateMachine(ctx, machine2)
			So(err, ShouldBeNil)

			// same realm different permission
			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.RegistrationsCreate, util.AtlLabAdminRealm)
			err = DeleteMachine(ctx, "machine-6")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, PermissionDenied)
		})

		Convey("Delete machine - Permission denied: different realm", func() {
			machine2 := &ufspb.Machine{
				Name:  "machine-7",
				Realm: util.AtlLabAdminRealm,
			}
			_, err := registration.CreateMachine(ctx, machine2)
			So(err, ShouldBeNil)

			// different realm
			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.RegistrationsDelete, util.BrowserLabAdminRealm)
			err = DeleteMachine(ctx, "machine-7")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, PermissionDenied)
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

func TestRenameMachine(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("RenameMachine", t, func() {
		Convey("Rename a Machine with new machine name", func() {
			nic := &ufspb.Nic{
				Name:    "machine-10:nic-10",
				Machine: "machine-10",
			}
			_, err := registration.CreateNic(ctx, nic)
			So(err, ShouldBeNil)
			drac := &ufspb.Drac{
				Name:    "drac-10",
				Machine: "machine-10",
			}
			_, err = registration.CreateDrac(ctx, drac)
			So(err, ShouldBeNil)
			host := &ufspb.MachineLSE{
				Name:     "machinelse-10",
				Machines: []string{"machine-10"},
				Nic:      "machine-10:nic-10",
			}
			_, err = inventory.CreateMachineLSE(ctx, host)
			So(err, ShouldBeNil)
			machine := &ufspb.Machine{
				Name: "machine-10",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
				},
				Realm: util.BrowserLabAdminRealm,
			}
			_, err = registration.CreateMachine(ctx, machine)
			So(err, ShouldBeNil)

			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.RegistrationsUpdate, util.BrowserLabAdminRealm)
			res, err := RenameMachine(ctx, "machine-10", "machine-202")
			So(err, ShouldBeNil)
			So(res.Name, ShouldEqual, "machine-202")
			So(res.GetChromeBrowserMachine().GetDisplayName(), ShouldEqual, "machine-202")

			_, err = registration.GetMachine(ctx, "machine-10")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
			_, err = registration.GetNic(ctx, "machine-10:nic-10")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
			nic, err = registration.GetNic(ctx, "machine-202:nic-10")
			So(err, ShouldBeNil)
			So(nic.GetMachine(), ShouldEqual, "machine-202")
			drac, err = registration.GetDrac(ctx, "drac-10")
			So(err, ShouldBeNil)
			So(drac.GetMachine(), ShouldEqual, "machine-202")
			host, err = inventory.GetMachineLSE(ctx, "machinelse-10")
			So(err, ShouldBeNil)
			So(host.GetMachines(), ShouldResemble, []string{"machine-202"})
			So(host.GetNic(), ShouldResemble, "machine-202:nic-10")

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "machines/machine-10")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 2)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRename)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRename)
			So(changes[0].GetEventLabel(), ShouldEqual, "machine")
			So(changes[1].GetOldValue(), ShouldEqual, "machine-10")
			So(changes[1].GetNewValue(), ShouldEqual, "machine-202")
			So(changes[1].GetEventLabel(), ShouldEqual, "machine.name")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "machines/machine-202")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 2)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRename)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRename)
			So(changes[0].GetEventLabel(), ShouldEqual, "machine")
			So(changes[1].GetOldValue(), ShouldEqual, "machine-10")
			So(changes[1].GetNewValue(), ShouldEqual, "machine-202")
			So(changes[1].GetEventLabel(), ShouldEqual, "machine.name")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "nics/machine-10:nic-10")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 3)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRename)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRename)
			So(changes[0].GetEventLabel(), ShouldEqual, "nic")
			So(changes[1].GetOldValue(), ShouldEqual, "machine-10:nic-10")
			So(changes[1].GetNewValue(), ShouldEqual, "machine-202:nic-10")
			So(changes[1].GetEventLabel(), ShouldEqual, "nic.name")
			So(changes[2].GetOldValue(), ShouldEqual, "machine-10")
			So(changes[2].GetNewValue(), ShouldEqual, "machine-202")
			So(changes[2].GetEventLabel(), ShouldEqual, "nic.machine")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "nics/machine-202:nic-10")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 3)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRename)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRename)
			So(changes[0].GetEventLabel(), ShouldEqual, "nic")
			So(changes[1].GetOldValue(), ShouldEqual, "machine-10:nic-10")
			So(changes[1].GetNewValue(), ShouldEqual, "machine-202:nic-10")
			So(changes[1].GetEventLabel(), ShouldEqual, "nic.name")
			So(changes[2].GetOldValue(), ShouldEqual, "machine-10")
			So(changes[2].GetNewValue(), ShouldEqual, "machine-202")
			So(changes[2].GetEventLabel(), ShouldEqual, "nic.machine")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "dracs/drac-10")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, "machine-10")
			So(changes[0].GetNewValue(), ShouldEqual, "machine-202")
			So(changes[0].GetEventLabel(), ShouldEqual, "drac.machine")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "hosts/machinelse-10")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 2)
			So(changes[0].GetOldValue(), ShouldEqual, "[machine-10]")
			So(changes[0].GetNewValue(), ShouldEqual, "[machine-202]")
			So(changes[0].GetEventLabel(), ShouldEqual, "machine_lse.machines")
			So(changes[1].GetOldValue(), ShouldEqual, "machine-10:nic-10")
			So(changes[1].GetNewValue(), ShouldEqual, "machine-202:nic-10")
			So(changes[1].GetEventLabel(), ShouldEqual, "machine_lse.nic")
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "machines/machine-10")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			So(msgs[0].Delete, ShouldBeTrue)
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "machines/machine-202")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			So(msgs[0].Delete, ShouldBeFalse)
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "nics/machine-10:nic-10")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			So(msgs[0].Delete, ShouldBeTrue)
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "nics/machine-202:nic-10")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			So(msgs[0].Delete, ShouldBeFalse)
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "dracs/drac-10")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			So(msgs[0].Delete, ShouldBeFalse)
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/machinelse-10")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			So(msgs[0].Delete, ShouldBeFalse)
		})
		Convey("Rename a non-existing Machine", func() {
			_, err := RenameMachine(ctx, "machine-11", "machine-211")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Rename a Machine to an already existing machine name", func() {
			machine := &ufspb.Machine{
				Name: "machine-12",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
				},
			}
			_, err := registration.CreateMachine(ctx, machine)
			So(err, ShouldBeNil)

			machine = &ufspb.Machine{
				Name: "machine-212",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
				},
			}
			_, err = registration.CreateMachine(ctx, machine)
			So(err, ShouldBeNil)

			_, err = RenameMachine(ctx, "machine-12", "machine-212")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Machine machine-212 already exists in the system")
		})
		Convey("Rename a Machine - permission denied: same realm and no update permission", func() {
			machine := &ufspb.Machine{
				Name: "machine-13",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
				},
				Realm: util.BrowserLabAdminRealm,
			}
			_, err := registration.CreateMachine(ctx, machine)
			So(err, ShouldBeNil)

			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.RegistrationsGet, util.BrowserLabAdminRealm)
			_, err = RenameMachine(ctx, "machine-13", "machine-313")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, PermissionDenied)
		})
		Convey("Rename a Machine - permission denied: different realm", func() {
			machine := &ufspb.Machine{
				Name: "machine-14",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
				},
				Realm: util.BrowserLabAdminRealm,
			}
			_, err := registration.CreateMachine(ctx, machine)
			So(err, ShouldBeNil)

			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.RegistrationsUpdate, util.AtlLabAdminRealm)
			_, err = RenameMachine(ctx, "machine-13", "machine-313")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, PermissionDenied)
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
			_, _, err := ListMachines(ctx, 5, "", "invalid=mx-1", false, false)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Invalid field name invalid")
		})

		Convey("List Machines - filter chromeplatform - happy path", func() {
			resp, _, _ := ListMachines(ctx, 5, "", "platform=cp-12", false, false)
			So(resp, ShouldNotBeNil)
			So(resp, ShouldResembleProto, machinesWithChromeplatform)
		})

		Convey("ListMachines - Full listing - happy path", func() {
			resp, _, _ := ListMachines(ctx, 5, "", "", false, false)
			So(resp, ShouldNotBeNil)
			So(resp, ShouldResembleProto, machines)
		})
	})
}

func TestBatchGetMachines(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("BatchGetMachines", t, func() {
		Convey("Batch get machine - happy path", func() {
			entities := make([]*ufspb.Machine, 4)
			for i := 0; i < 4; i++ {
				entities[i] = &ufspb.Machine{
					Name: fmt.Sprintf("machine-batchGet-%d", i),
				}
			}
			_, err := registration.BatchUpdateMachines(ctx, entities)
			So(err, ShouldBeNil)
			resp, err := registration.BatchGetMachines(ctx, []string{"machine-batchGet-0", "machine-batchGet-1", "machine-batchGet-2", "machine-batchGet-3"})
			So(err, ShouldBeNil)
			So(resp, ShouldHaveLength, 4)
			So(resp, ShouldResembleProto, entities)
		})
		Convey("Batch get machines  - missing id", func() {
			resp, err := registration.BatchGetMachines(ctx, []string{"machine-batchGet-non-existing"})
			So(err, ShouldNotBeNil)
			So(resp, ShouldBeNil)
			So(err.Error(), ShouldContainSubstring, "machine-batchGet-non-existing")
		})
		Convey("Batch get machines  - empty input", func() {
			resp, err := registration.BatchGetMachines(ctx, nil)
			So(err, ShouldBeNil)
			So(resp, ShouldHaveLength, 0)

			input := make([]string, 0)
			resp, err = registration.BatchGetMachines(ctx, input)
			So(err, ShouldBeNil)
			So(resp, ShouldHaveLength, 0)
		})
	})
}
