// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"

	ufspb "infra/unifiedfleet/api/v1/proto"
	"infra/unifiedfleet/app/model/history"
	"infra/unifiedfleet/app/model/registration"
	"infra/unifiedfleet/app/model/state"
)

func TestMachineRegistration(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("TestMachineRegistration", t, func() {
		Convey("Register machine with already existing machine, nic and drac", func() {
			machine := &ufspb.Machine{
				Name: "machine-1",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
				},
			}
			_, err := registration.CreateMachine(ctx, machine)
			So(err, ShouldBeNil)

			nic := &ufspb.Nic{
				Name: "nic-1",
			}
			_, err = registration.CreateNic(ctx, nic)
			So(err, ShouldBeNil)

			drac := &ufspb.Drac{
				Name: "drac-1",
			}
			_, err = registration.CreateDrac(ctx, drac)
			So(err, ShouldBeNil)

			_, _, _, err = MachineRegistration(ctx, machine, []*ufspb.Nic{nic}, drac)
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
			_, _, _, err := MachineRegistration(ctx, machine, nil, nil)
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
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
				},
			}
			nic := &ufspb.Nic{
				Name: "nic-2",
				SwitchInterface: &ufspb.SwitchInterface{
					Switch: "switch-1",
				},
			}
			_, _, _, err := MachineRegistration(ctx, machine, []*ufspb.Nic{nic}, nil)
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
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
				},
			}
			drac := &ufspb.Drac{
				Name: "drac-2",
				SwitchInterface: &ufspb.SwitchInterface{
					Switch: "switch-1",
				},
			}
			_, _, _, err := MachineRegistration(ctx, machine, nil, drac)
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
			machine := &ufspb.Machine{
				Name: "machine-browser-3",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
				},
			}
			nic := &ufspb.Nic{
				Name: "nic-browser-3",
			}
			drac := &ufspb.Drac{
				Name: "drac-browser-3",
			}
			m, n, d, err := MachineRegistration(ctx, machine, []*ufspb.Nic{nic}, drac)
			nic.Name = "machine-browser-3" + "nic-browser-3"
			So(err, ShouldBeNil)
			So(m, ShouldResembleProto, machine)
			So(n, ShouldResembleProto, []*ufspb.Nic{nic})
			So(d, ShouldResembleProto, drac)
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

		Convey("Register OS machine with nics - error", func() {
			machine := &ufspb.Machine{
				Name: "machine-os-3",
				Device: &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: &ufspb.ChromeOSMachine{},
				},
			}
			nic := &ufspb.Nic{
				Name: "nic-3",
			}
			_, _, _, err := MachineRegistration(ctx, machine, []*ufspb.Nic{nic}, nil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Machine machine-os-3 is not a browser machine. "+
				"Nics can only be added to a browser machine.")

			// No changes are recorded as the registration fails
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "machines/machine-3")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "nics/nic-3")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})

		Convey("Register OS machine happy path", func() {
			machine := &ufspb.Machine{
				Name: "machine-os-3",
				Device: &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: &ufspb.ChromeOSMachine{},
				},
			}
			m, _, _, err := MachineRegistration(ctx, machine, nil, nil)
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
