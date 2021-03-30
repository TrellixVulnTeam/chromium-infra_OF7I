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
	chromeosLab "infra/unifiedfleet/api/v1/models/chromeos/lab"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	"infra/unifiedfleet/app/external"
	"infra/unifiedfleet/app/model/configuration"
	. "infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/model/history"
	"infra/unifiedfleet/app/model/inventory"
	"infra/unifiedfleet/app/model/registration"
	"infra/unifiedfleet/app/model/state"
	"infra/unifiedfleet/app/util"
)

func mockDutMachineLSE(name string) *ufspb.MachineLSE {
	dut := &chromeosLab.DeviceUnderTest{
		Hostname: name,
	}
	device := &ufspb.ChromeOSDeviceLSE_Dut{
		Dut: dut,
	}
	deviceLse := &ufspb.ChromeOSDeviceLSE{
		Device: device,
	}
	chromeosLse := &ufspb.ChromeOSMachineLSE_DeviceLse{
		DeviceLse: deviceLse,
	}
	chromeOSMachineLse := &ufspb.ChromeOSMachineLSE{
		ChromeosLse: chromeosLse,
	}
	lse := &ufspb.MachineLSE_ChromeosMachineLse{
		ChromeosMachineLse: chromeOSMachineLse,
	}
	return &ufspb.MachineLSE{
		Name:     name,
		Hostname: name,
		Lse:      lse,
	}
}

func mockLabstationMachineLSE(name string) *ufspb.MachineLSE {
	labstation := &chromeosLab.Labstation{
		Hostname: name,
	}
	device := &ufspb.ChromeOSDeviceLSE_Labstation{
		Labstation: labstation,
	}
	deviceLse := &ufspb.ChromeOSDeviceLSE{
		Device: device,
	}
	chromeosLse := &ufspb.ChromeOSMachineLSE_DeviceLse{
		DeviceLse: deviceLse,
	}
	chromeOSMachineLse := &ufspb.ChromeOSMachineLSE{
		ChromeosLse: chromeosLse,
	}
	lse := &ufspb.MachineLSE_ChromeosMachineLse{
		ChromeosMachineLse: chromeOSMachineLse,
	}
	return &ufspb.MachineLSE{
		Name:     name,
		Hostname: name,
		Lse:      lse,
	}
}

func TestCreateMachineLSE(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("CreateMachineLSE", t, func() {
		Convey("Create new machineLSE with already existing machinelse", func() {
			machine1 := &ufspb.Machine{
				Name: "machine-10",
			}
			_, merr := registration.CreateMachine(ctx, machine1)
			So(merr, ShouldBeNil)

			_, err := inventory.CreateMachineLSE(ctx, &ufspb.MachineLSE{
				Name: "machinelse-1",
			})
			So(err, ShouldBeNil)

			machineLSE1 := &ufspb.MachineLSE{
				Hostname: "machinelse-1",
				Machines: []string{"machine-10"},
			}
			resp, err := CreateMachineLSE(ctx, machineLSE1, nil)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "MachineLSE machinelse-1 already exists in the system.")

			// No changes are recorded as the creation fails
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/machinelse-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})

		Convey("Create new machineLSE with non existing machines", func() {
			machineLSE1 := &ufspb.MachineLSE{
				Hostname: "machinelse-2",
				Machines: []string{"machine-1", "machine-2"},
			}
			resp, err := CreateMachineLSE(ctx, machineLSE1, nil)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)

			// No changes are recorded as the creation fails
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/machinelse-2")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})

		Convey("Create new machineLSE with existing machines, specify ip with wrong vlan name", func() {
			_, err := registration.CreateNic(ctx, &ufspb.Nic{
				Name:    "eth1",
				Machine: "machine-wrong-nic",
			})
			So(err, ShouldBeNil)
			machine1 := &ufspb.Machine{
				Name: "machine-wrong-nic",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
				},
			}
			_, err = registration.CreateMachine(ctx, machine1)
			So(err, ShouldBeNil)

			machineLSE2 := &ufspb.MachineLSE{
				Hostname: "machinelse-wrong-nic",
				Machines: []string{"machine-wrong-nic"},
			}
			resp, err := CreateMachineLSE(ctx, machineLSE2, &ufsAPI.NetworkOption{
				Vlan: "vlan-1",
				Nic:  "eth1",
			})
			So(err, ShouldNotBeNil)
			So(resp, ShouldBeNil)
			So(err.Error(), ShouldContainSubstring, "There is no Vlan with VlanID vlan-1")

			// No changes are recorded as the creation fails
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/machinelse-wrong-nic")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})

		Convey("Create new machineLSE with existing machines and specify ip", func() {
			_, err := registration.CreateNic(ctx, &ufspb.Nic{
				Name:    "eth0",
				Machine: "machine-with-ip",
			})
			So(err, ShouldBeNil)
			machine1 := &ufspb.Machine{
				Name: "machine-with-ip",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
				},
			}
			_, err = registration.CreateMachine(ctx, machine1)
			So(err, ShouldBeNil)
			vlan := &ufspb.Vlan{
				Name:        "vlan-1",
				VlanAddress: "192.168.40.0/22",
			}
			_, err = configuration.CreateVlan(ctx, vlan)
			ips, _, _, _, err := util.ParseVlan(vlan.GetName(), vlan.GetVlanAddress(), "", "")
			So(err, ShouldBeNil)
			// Only import the first 20 as one single transaction cannot import all.
			_, err = configuration.ImportIPs(ctx, ips[0:20])
			So(err, ShouldBeNil)

			vm1 := &ufspb.VM{
				Name:       "vm1-ip",
				MacAddress: "old_mac_address",
			}
			machineLSE2 := &ufspb.MachineLSE{
				Name:     "machinelse-with-ip",
				Hostname: "machinelse-with-ip",
				Lse: &ufspb.MachineLSE_ChromeBrowserMachineLse{
					ChromeBrowserMachineLse: &ufspb.ChromeBrowserMachineLSE{
						Vms: []*ufspb.VM{vm1},
					},
				},
				Machines: []string{"machine-with-ip"},
			}
			resp, err := CreateMachineLSE(ctx, machineLSE2, &ufsAPI.NetworkOption{
				Vlan: "vlan-1",
				Nic:  "eth0",
			})
			So(err, ShouldBeNil)
			machineLSE2.Nic = "eth0"
			So(resp, ShouldResembleProto, machineLSE2)
			ip, err := configuration.QueryIPByPropertyName(ctx, map[string]string{"ipv4_str": "192.168.40.11"})
			So(err, ShouldBeNil)
			So(ip, ShouldHaveLength, 1)
			So(ip[0].GetOccupied(), ShouldBeTrue)
			dhcp, err := configuration.GetDHCPConfig(ctx, "machinelse-with-ip")
			So(err, ShouldBeNil)
			So(dhcp.GetIp(), ShouldEqual, "192.168.40.11")
			s, err := state.GetStateRecord(ctx, "hosts/machinelse-with-ip")
			So(err, ShouldBeNil)
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_DEPLOYING)
			s, err = state.GetStateRecord(ctx, "machines/machine-with-ip")
			So(err, ShouldBeNil)
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_SERVING)
			vm, err := inventory.GetVM(ctx, "vm1-ip")
			So(err, ShouldBeNil)
			So(vm, ShouldResembleProto, vm1)

			// verify changes
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/machinelse-with-ip")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetEventLabel(), ShouldEqual, "machine_lse")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "vms/vm1-ip")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetEventLabel(), ShouldEqual, "vm")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "states/vms/vm1-ip")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetEventLabel(), ShouldEqual, "state_record.state")
			So(changes[0].GetOldValue(), ShouldEqual, ufspb.State_STATE_UNSPECIFIED.String())
			So(changes[0].GetNewValue(), ShouldEqual, ufspb.State_STATE_REGISTERED.String())
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "states/hosts/machinelse-with-ip")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetEventLabel(), ShouldEqual, "state_record.state")
			So(changes[0].GetOldValue(), ShouldEqual, ufspb.State_STATE_UNSPECIFIED.String())
			So(changes[0].GetNewValue(), ShouldEqual, ufspb.State_STATE_DEPLOYING.String())
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "states/machines/machine-with-ip")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetEventLabel(), ShouldEqual, "state_record.state")
			So(changes[0].GetOldValue(), ShouldEqual, ufspb.State_STATE_UNSPECIFIED.String())
			So(changes[0].GetNewValue(), ShouldEqual, ufspb.State_STATE_SERVING.String())
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "dhcps/machinelse-with-ip")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetEventLabel(), ShouldEqual, "dhcp_config.ip")
			So(changes[0].GetOldValue(), ShouldEqual, "")
			So(changes[0].GetNewValue(), ShouldEqual, dhcp.GetIp())
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/machinelse-with-ip")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			So(msgs[0].Delete, ShouldBeFalse)
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "vms/vm1-ip")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			So(msgs[0].Delete, ShouldBeFalse)
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "states/vms/vm1-ip")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			So(msgs[0].Delete, ShouldBeFalse)
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "states/hosts/machinelse-with-ip")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			So(msgs[0].Delete, ShouldBeFalse)
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "states/machines/machine-with-ip")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			So(msgs[0].Delete, ShouldBeFalse)
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "dhcps/machinelse-with-ip")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			So(msgs[0].Delete, ShouldBeFalse)
		})

		Convey("Create new machineLSE with existing machines", func() {
			machine1 := &ufspb.Machine{
				Name:         "machine-3",
				SerialNumber: "machine-3-serial",
			}
			_, err := registration.CreateMachine(ctx, machine1)
			So(err, ShouldBeNil)

			machineLSE2 := &ufspb.MachineLSE{
				Hostname: "machinelse-3",
				Machines: []string{"machine-3"},
			}
			resp, err := CreateMachineLSE(ctx, machineLSE2, nil)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, machineLSE2)
			s, err := state.GetStateRecord(ctx, "hosts/machinelse-3")
			So(err, ShouldBeNil)
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_REGISTERED)
			s, err = state.GetStateRecord(ctx, "machines/machine-3")
			So(err, ShouldBeNil)
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_SERVING)
			dr, err := inventory.GetMachineLSEDeployment(ctx, "machine-3-serial")
			So(err, ShouldBeNil)
			So(dr.GetHostname(), ShouldEqual, "machinelse-3")

			// No changes are recorded as the creation fails
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/machinelse-3")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetEventLabel(), ShouldEqual, "machine_lse")
		})

		Convey("Create new machineLSE with existing machines & existing deployment record", func() {
			machine1 := &ufspb.Machine{
				Name:         "machine-dr-3",
				SerialNumber: "machine-dr-3-serial",
			}
			_, err := registration.CreateMachine(ctx, machine1)
			So(err, ShouldBeNil)

			machineLSE2 := &ufspb.MachineLSE{
				Hostname: "machinelse-dr-3",
				Machines: []string{"machine-dr-3"},
			}

			_, err = inventory.UpdateMachineLSEDeployments(ctx, []*ufspb.MachineLSEDeployment{util.FormatDeploymentRecord("", machine1.GetSerialNumber())})
			So(err, ShouldBeNil)
			dr, err := inventory.GetMachineLSEDeployment(ctx, "machine-dr-3-serial")
			So(err, ShouldBeNil)
			So(dr.GetHostname(), ShouldEqual, "no-host-yet-machine-dr-3-serial")

			_, err = CreateMachineLSE(ctx, machineLSE2, nil)
			So(err, ShouldBeNil)
			dr, err = inventory.GetMachineLSEDeployment(ctx, "machine-dr-3-serial")
			So(err, ShouldBeNil)
			So(dr.GetHostname(), ShouldEqual, "machinelse-dr-3")
		})

		Convey("Create new machineLSE with a machine already attached to a different machinelse - error", func() {
			machine1 := &ufspb.Machine{
				Name: "machine-4",
			}
			_, err := registration.CreateMachine(ctx, machine1)
			So(err, ShouldBeNil)

			_, err = inventory.CreateMachineLSE(ctx, &ufspb.MachineLSE{
				Name:     "machinelse-4",
				Machines: []string{"machine-4"},
			})
			So(err, ShouldBeNil)

			machineLSE := &ufspb.MachineLSE{
				Hostname: "machinelse-5",
				Machines: []string{"machine-4"},
			}
			resp, err := CreateMachineLSE(ctx, machineLSE, nil)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Hosts referring the machine machine-4:\nmachinelse-4")

			// No changes are recorded as the creation fails
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/machinelse-5")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "machines/machine-4")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})
	})
}

func TestCreateMachineLSELabstation(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	ctx = external.WithTestingContext(ctx)
	machine := &ufspb.Machine{
		Name: "machine-4",
		Device: &ufspb.Machine_ChromeosMachine{
			ChromeosMachine: &ufspb.ChromeOSMachine{
				BuildTarget: "test",
				Model:       "test",
			},
		},
	}
	registration.CreateMachine(ctx, machine)
	Convey("CreateMachineLSE for a Labstation", t, func() {
		Convey("Create machineLSE Labstation with Servo Info", func() {
			labstationMachinelse1 := mockLabstationMachineLSE("RedLabstation-0")
			labstationMachinelse1.Machines = []string{"machine-4"}
			servo := &chromeosLab.Servo{
				ServoHostname: "RedLabstation-0",
				ServoPort:     22,
			}
			labstationMachinelse1.GetChromeosMachineLse().GetDeviceLse().GetLabstation().Servos = []*chromeosLab.Servo{servo}
			resp, err := CreateMachineLSE(ctx, labstationMachinelse1, nil)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Servos are not allowed to be added")
			_, err = state.GetStateRecord(ctx, "hosts/RedLabstation-0")
			So(err.Error(), ShouldContainSubstring, NotFound)

			// No changes are recorded as the creation fails
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/RedLabstation-0")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})
		Convey("Create machineLSE Labstation without Servo Info", func() {
			labstationMachinelse1 := mockLabstationMachineLSE("RedLabstation-1")
			labstationMachinelse1.Machines = []string{"machine-4"}
			resp, err := CreateMachineLSE(ctx, labstationMachinelse1, nil)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, labstationMachinelse1)
			s, err := state.GetStateRecord(ctx, "hosts/RedLabstation-1")
			So(err, ShouldBeNil)
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_REGISTERED)
			s, err = state.GetStateRecord(ctx, "machines/machine-4")
			So(err, ShouldBeNil)
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_SERVING)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/RedLabstation-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetEventLabel(), ShouldEqual, "machine_lse")
		})
	})
}

func TestUpdateMachineLSEDUT(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	machine10 := &ufspb.Machine{
		Name: "machine-10",
	}
	registration.CreateMachine(ctx, machine10)

	machine21 := &ufspb.Machine{
		Name: "machine-21",
	}
	registration.CreateMachine(ctx, machine21)

	machine22 := &ufspb.Machine{
		Name: "machine-22",
	}
	registration.CreateMachine(ctx, machine22)

	servo1 := &chromeosLab.Servo{
		ServoHostname: "BlueLabstation-10",
		ServoPort:     9921,
		ServoSerial:   "Serial-1",
	}
	servo2 := &chromeosLab.Servo{
		ServoHostname: "BlueLabstation-10",
		ServoPort:     9922,
		ServoSerial:   "Serial-2",
	}

	labstationMachinelse := mockLabstationMachineLSE("BlueLabstation-10")
	labstationMachinelse.Machines = []string{"machine-10"}
	labstationMachinelse.GetChromeosMachineLse().GetDeviceLse().GetLabstation().Servos = []*chromeosLab.Servo{servo1, servo2}
	inventory.CreateMachineLSE(ctx, labstationMachinelse)

	peripherals1 := &chromeosLab.Peripherals{
		Servo: servo1,
	}
	dutMachinelse1 := mockDutMachineLSE("DUTMachineLSE-21")
	dutMachinelse1.Machines = []string{"machine-21"}
	dutMachinelse1.GetChromeosMachineLse().GetDeviceLse().GetDut().Peripherals = peripherals1
	inventory.CreateMachineLSE(ctx, dutMachinelse1)

	peripherals2 := &chromeosLab.Peripherals{
		Servo: servo2,
	}
	dutMachinelse2 := mockDutMachineLSE("DUTMachineLSE-22")
	dutMachinelse2.Machines = []string{"machine-22"}
	dutMachinelse2.GetChromeosMachineLse().GetDeviceLse().GetDut().Peripherals = peripherals2
	inventory.CreateMachineLSE(ctx, dutMachinelse2)

	labstationMachinelse, _ = inventory.GetMachineLSE(ctx, "BlueLabstation-10")
	Convey("UpdateMachineLSE for a DUT", t, func() {
		Convey("Update non-existing machineLSE DUT", func() {
			dutMachinelse := mockDutMachineLSE("DUTMachineLSE-23")
			resp, err := UpdateMachineLSE(ctx, dutMachinelse, nil)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})

		Convey("Update machineLSE DUT with same ServoPort and same ServoHostname", func() {
			servo3 := &chromeosLab.Servo{
				ServoHostname: "BlueLabstation-10",
				ServoPort:     9921,
				ServoSerial:   "Serial-1",
			}
			peripherals3 := &chromeosLab.Peripherals{
				Servo: servo3,
			}
			dutMachinelse3 := mockDutMachineLSE("DUTMachineLSE-21")
			dutMachinelse3.Machines = []string{"machine-21"}
			dutMachinelse3.GetChromeosMachineLse().GetDeviceLse().GetDut().Peripherals = peripherals3
			resp, err := UpdateMachineLSE(ctx, dutMachinelse3, nil)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, dutMachinelse3)

			resp, _ = inventory.GetMachineLSE(ctx, "BlueLabstation-10")
			So(resp.GetChromeosMachineLse().GetDeviceLse().GetLabstation().Servos, ShouldHaveLength, 2)
			// Labstation host is not updated
			_, err = state.GetStateRecord(ctx, "hosts/BlueLabstation-10")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})

		Convey("Update machineLSE DUT with different ServerPort and same ServoHostname", func() {
			servo3 := &chromeosLab.Servo{
				ServoHostname: "BlueLabstation-10",
				ServoPort:     9958,
				ServoSerial:   "Serial-2",
			}
			peripherals3 := &chromeosLab.Peripherals{
				Servo: servo3,
			}
			dutMachinelse3 := mockDutMachineLSE("DUTMachineLSE-22")
			dutMachinelse3.Machines = []string{"machine-22"}
			dutMachinelse3.GetChromeosMachineLse().GetDeviceLse().GetDut().Peripherals = peripherals3
			dutMachinelse3.ResourceState = ufspb.State_STATE_SERVING
			resp, err := UpdateMachineLSE(ctx, dutMachinelse3, nil)
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp, ShouldResembleProto, dutMachinelse3)

			dummyServos := []*chromeosLab.Servo{servo1, servo3}

			resp, _ = inventory.GetMachineLSE(ctx, "BlueLabstation-10")
			So(resp.GetChromeosMachineLse().GetDeviceLse().GetLabstation().Servos, ShouldResembleProto, dummyServos)

			// Labstation host is not updated
			_, err = state.GetStateRecord(ctx, "hosts/BlueLabstation-10")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})

		Convey("Update machineLSE DUT with different ServoHostname", func() {
			servo1 := &chromeosLab.Servo{
				ServoHostname: "BlueLabstation-17",
				ServoPort:     9917,
				ServoSerial:   "Serial-17",
			}
			labstationMachinelse1 := mockLabstationMachineLSE("BlueLabstation-17")
			labstationMachinelse1.GetChromeosMachineLse().GetDeviceLse().GetLabstation().Servos = []*chromeosLab.Servo{servo1}
			_, err := inventory.CreateMachineLSE(ctx, labstationMachinelse1)
			So(err, ShouldBeNil)

			servo2 := &chromeosLab.Servo{
				ServoHostname: "BlueLabstation-18",
				ServoPort:     9918,
				ServoSerial:   "Serial-18",
			}
			labstationMachinelse2 := mockLabstationMachineLSE("BlueLabstation-18")
			labstationMachinelse2.GetChromeosMachineLse().GetDeviceLse().GetLabstation().Servos = []*chromeosLab.Servo{}
			_, err = inventory.CreateMachineLSE(ctx, labstationMachinelse2)
			So(err, ShouldBeNil)

			_, err = registration.CreateMachine(ctx, &ufspb.Machine{
				Name: "machine-17",
			})
			So(err, ShouldBeNil)
			peripherals1 := &chromeosLab.Peripherals{
				Servo: servo1,
			}
			dutMachinelse1 := mockDutMachineLSE("DUTMachineLSE-17")
			dutMachinelse1.Machines = []string{"machine-17"}
			dutMachinelse1.GetChromeosMachineLse().GetDeviceLse().GetDut().Peripherals = peripherals1
			_, err = inventory.CreateMachineLSE(ctx, dutMachinelse1)
			So(err, ShouldBeNil)

			peripherals2 := &chromeosLab.Peripherals{
				Servo: servo2,
			}
			dutMachinelse2 := mockDutMachineLSE("DUTMachineLSE-17")
			dutMachinelse2.Machines = []string{"machine-17"}
			dutMachinelse2.GetChromeosMachineLse().GetDeviceLse().GetDut().Peripherals = peripherals2
			resp, err := UpdateMachineLSE(ctx, dutMachinelse2, nil)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, dutMachinelse2)

			resp, _ = inventory.GetMachineLSE(ctx, "BlueLabstation-17")
			So(resp, ShouldNotBeNil)
			servos := resp.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetServos()
			So(servos, ShouldBeEmpty)

			resp, _ = inventory.GetMachineLSE(ctx, "BlueLabstation-18")
			So(resp, ShouldNotBeNil)
			servos = resp.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetServos()
			So(servo2, ShouldResembleProto, servos[0])

			// Labstation host is not updated
			_, err = state.GetStateRecord(ctx, "hosts/BlueLabstation-17")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
			_, err = state.GetStateRecord(ctx, "hosts/BlueLabstation-18")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
	})
}

func TestUpdateMachineLSELabstation(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("UpdateMachineLSE for a Labstation", t, func() {
		Convey("Update machineLSE Labstation with Servo Info", func() {
			machine := &ufspb.Machine{
				Name: "machine-10",
			}
			_, err := registration.CreateMachine(ctx, machine)
			So(err, ShouldBeNil)

			labstationMachinelse1 := mockLabstationMachineLSE("RedLabstation-10")
			labstationMachinelse1.Machines = []string{"machine-10"}
			inventory.CreateMachineLSE(ctx, labstationMachinelse1)

			labstationMachinelse2 := mockLabstationMachineLSE("RedLabstation-10")
			servo := &chromeosLab.Servo{
				ServoHostname: "RedLabstation-10",
				ServoPort:     22,
			}
			labstationMachinelse2.GetChromeosMachineLse().GetDeviceLse().GetLabstation().Servos = []*chromeosLab.Servo{servo}
			resp, err := UpdateMachineLSE(ctx, labstationMachinelse2, nil)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Servos are not allowed to be updated")
			_, err = state.GetStateRecord(ctx, "hosts/RedLabstation-10")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)

			// No changes are recorded as the updating fails
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/RedLabstation-10")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})

		Convey("Update machineLSE Labstation without Servo Info", func() {
			machine := &ufspb.Machine{
				Name: "machine-4x",
			}
			_, err := registration.CreateMachine(ctx, machine)
			So(err, ShouldBeNil)

			labstationMachinelse1 := mockLabstationMachineLSE("RedLabstation-11")
			labstationMachinelse1.Machines = []string{"machine-4x"}
			inventory.CreateMachineLSE(ctx, labstationMachinelse1)

			labstationMachinelse2 := mockLabstationMachineLSE("RedLabstation-11")
			labstationMachinelse2.Machines = []string{"machine-4x"}
			resp, err := UpdateMachineLSE(ctx, labstationMachinelse2, nil)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, labstationMachinelse2)

			/* TODO(eshwarn): Add back when implement labstation state
			s, err := state.GetStateRecord(ctx, "hosts/RedLabstation-11")
			So(err, ShouldBeNil)
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_DEPLOYED_PRE_SERVING)
			*/

			// No changes happened in this update
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/RedLabstation-11")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})
	})
}

func TestUpdateMachineLSE(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("UpdateMachineLSE for a browser machine", t, func() {
		Convey("Update machineLSE with setting state", func() {
			machineLSE1 := &ufspb.MachineLSE{
				Name:     "machinelse-update1",
				Hostname: "machinelse-update1",
				Machines: []string{"machine-update1"},
				Lse: &ufspb.MachineLSE_ChromeBrowserMachineLse{
					ChromeBrowserMachineLse: &ufspb.ChromeBrowserMachineLSE{},
				},
			}
			_, err := registration.CreateMachine(ctx, &ufspb.Machine{
				Name: "machine-update1",
			})
			So(err, ShouldBeNil)
			_, err = inventory.CreateMachineLSE(ctx, machineLSE1)
			So(err, ShouldBeNil)

			_, err = inventory.BatchUpdateVMs(ctx, []*ufspb.VM{
				{
					Name:         "vm1",
					MacAddress:   "old_mac_address",
					MachineLseId: "machinelse-update1",
				},
				{
					Name:         "vm-update",
					MacAddress:   "old_mac_address",
					MachineLseId: "machinelse-update1",
				},
			})
			So(err, ShouldBeNil)
			machineLSE1.GetChromeBrowserMachineLse().OsVersion = &ufspb.OSVersion{
				Value: "new_os",
			}
			machineLSE1.ResourceState = ufspb.State_STATE_DEPLOYED_TESTING
			m, err := UpdateMachineLSE(ctx, machineLSE1, nil)
			So(err, ShouldBeNil)
			So(m.GetChromeBrowserMachineLse().GetVms(), ShouldHaveLength, 2)
			// State remains unchanged as vm1 is not updated
			s, err := state.GetStateRecord(ctx, "hosts/machinelse-update1")
			So(err, ShouldBeNil)
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_DEPLOYED_TESTING)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/machinelse-update1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetEventLabel(), ShouldEqual, "machine_lse.chrome_browser_machine_lse.os_version")
			So(changes[0].OldValue, ShouldEqual, "<nil>")
			So(changes[0].NewValue, ShouldEqual, "value:\"new_os\"")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "states/hosts/machinelse-update1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetEventLabel(), ShouldEqual, "state_record.state")
			So(changes[0].GetOldValue(), ShouldEqual, ufspb.State_STATE_UNSPECIFIED.String())
			So(changes[0].GetNewValue(), ShouldEqual, ufspb.State_STATE_DEPLOYED_TESTING.String())
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "states/machines/machine-update1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "dhcps/machinelse-update1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/machinelse-update1")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "states/hosts/machinelse-update1")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "states/machines/machine-update1")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 0)
		})

		Convey("Update machineLSE by setting ip by vlan for host", func() {
			_, err := registration.CreateNic(ctx, &ufspb.Nic{
				Name:    "eth0",
				Machine: "machine-update-host",
			})
			So(err, ShouldBeNil)
			machine1 := &ufspb.Machine{
				Name: "machine-update-host",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
				},
			}
			machineLSE1 := &ufspb.MachineLSE{
				Name:     "machinelse-update-host",
				Hostname: "machinelse-update-host",
				Machines: []string{"machine-update-host"},
				Lse: &ufspb.MachineLSE_ChromeBrowserMachineLse{
					ChromeBrowserMachineLse: &ufspb.ChromeBrowserMachineLSE{
						Vms: []*ufspb.VM{
							{
								Name:       "vm1",
								MacAddress: "old_mac_address",
							},
						},
					},
				},
			}
			_, err = registration.CreateMachine(ctx, machine1)
			So(err, ShouldBeNil)
			_, err = inventory.CreateMachineLSE(ctx, machineLSE1)
			So(err, ShouldBeNil)
			vlan := &ufspb.Vlan{
				Name:        "vlan-1",
				VlanAddress: "192.168.40.0/22",
			}
			_, err = configuration.CreateVlan(ctx, vlan)
			ips, _, startFreeIP, _, err := util.ParseVlan(vlan.GetName(), vlan.GetVlanAddress(), vlan.GetFreeStartIpv4Str(), vlan.GetFreeEndIpv4Str())
			var assignedIP *ufspb.IP
			for _, ip := range ips {
				if ip.GetIpv4Str() == startFreeIP {
					assignedIP = ip
				}
			}
			So(err, ShouldBeNil)
			// Only import the first 20 as one single transaction cannot import all.
			_, err = configuration.ImportIPs(ctx, ips[0:20])
			So(err, ShouldBeNil)

			_, err = UpdateMachineLSEHost(ctx, machineLSE1.Name, &ufsAPI.NetworkOption{
				Vlan: "vlan-1",
				Nic:  "eth0",
			})
			So(err, ShouldBeNil)
			dhcp, err := configuration.GetDHCPConfig(ctx, "machinelse-update-host")
			So(err, ShouldBeNil)
			ip, err := configuration.QueryIPByPropertyName(ctx, map[string]string{"ipv4_str": dhcp.GetIp()})
			So(err, ShouldBeNil)
			So(ip, ShouldHaveLength, 1)
			So(ip[0].GetOccupied(), ShouldBeTrue)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/machinelse-update-host")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 3)
			// nic & vlan & ip info are changed
			So(changes[0].GetEventLabel(), ShouldEqual, "machine_lse.nic")
			So(changes[0].GetOldValue(), ShouldEqual, "")
			So(changes[0].GetNewValue(), ShouldEqual, "eth0")
			So(changes[1].GetEventLabel(), ShouldEqual, "machine_lse.vlan")
			So(changes[1].GetOldValue(), ShouldEqual, "")
			So(changes[1].GetNewValue(), ShouldEqual, "vlan-1")
			So(changes[2].GetEventLabel(), ShouldEqual, "machine_lse.ip")
			So(changes[2].GetOldValue(), ShouldEqual, "")
			So(changes[2].GetNewValue(), ShouldEqual, "192.168.40.11")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "states/hosts/machinelse-update-host")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetEventLabel(), ShouldEqual, "state_record.state")
			So(changes[0].GetOldValue(), ShouldEqual, ufspb.State_STATE_UNSPECIFIED.String())
			So(changes[0].GetNewValue(), ShouldEqual, ufspb.State_STATE_DEPLOYING.String())
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "dhcps/machinelse-update-host")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetEventLabel(), ShouldEqual, "dhcp_config.ip")
			So(changes[0].GetOldValue(), ShouldEqual, "")
			So(changes[0].GetNewValue(), ShouldEqual, dhcp.GetIp())
			changes, err = history.QueryChangesByPropertyName(ctx, "name", fmt.Sprintf("ips/%s", assignedIP.GetId()))
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetEventLabel(), ShouldEqual, "ip.occupied")
			So(changes[0].GetOldValue(), ShouldEqual, "false")
			So(changes[0].GetNewValue(), ShouldEqual, "true")
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/machinelse-update-host")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "states/hosts/machinelse-update-host")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "dhcps/machinelse-update-host")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
		})

		Convey("Update machineLSE by deleting ip for host", func() {
			_, err := registration.CreateNic(ctx, &ufspb.Nic{
				Name:    "eth0-delete",
				Machine: "machine-update-host-delete-ip",
			})
			So(err, ShouldBeNil)
			machine1 := &ufspb.Machine{
				Name: "machine-update-host-delete-ip",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
				},
			}
			machineLSE1 := &ufspb.MachineLSE{
				Name:     "machinelse-update-host-delete-ip",
				Hostname: "machinelse-update-host-delete-ip",
				Machines: []string{"machine-update-host-delete-ip"},
				Lse: &ufspb.MachineLSE_ChromeBrowserMachineLse{
					ChromeBrowserMachineLse: &ufspb.ChromeBrowserMachineLSE{},
				},
			}
			_, err = registration.CreateMachine(ctx, machine1)
			So(err, ShouldBeNil)
			_, err = inventory.CreateMachineLSE(ctx, machineLSE1)
			So(err, ShouldBeNil)
			vlan := &ufspb.Vlan{
				Name:        "vlan-1",
				VlanAddress: "192.168.40.0/22",
			}
			_, err = configuration.CreateVlan(ctx, vlan)
			ips, _, _, _, err := util.ParseVlan(vlan.GetName(), vlan.GetVlanAddress(), vlan.GetFreeStartIpv4Str(), vlan.GetFreeEndIpv4Str())
			So(err, ShouldBeNil)
			// Only import the first 20 as one single transaction cannot import all.
			_, err = configuration.ImportIPs(ctx, ips[0:20])
			So(err, ShouldBeNil)
			_, err = UpdateMachineLSEHost(ctx, machineLSE1.Name, &ufsAPI.NetworkOption{
				Vlan: "vlan-1",
				Ip:   "192.168.40.12",
				Nic:  "eth0-delete",
			})
			So(err, ShouldBeNil)
			oldIPs, err := configuration.QueryIPByPropertyName(ctx, map[string]string{"ipv4_str": "192.168.40.12"})
			So(err, ShouldBeNil)
			So(oldIPs, ShouldHaveLength, 1)
			So(oldIPs[0].GetOccupied(), ShouldBeTrue)

			err = DeleteMachineLSEHost(ctx, machineLSE1.Name)
			So(err, ShouldBeNil)
			_, err = configuration.GetDHCPConfig(ctx, "machinelse-update-host-delete-ip")
			// Not found error
			So(err, ShouldNotBeNil)
			ip2, err := configuration.QueryIPByPropertyName(ctx, map[string]string{"ipv4_str": "192.168.40.12"})
			So(err, ShouldBeNil)
			So(ip2, ShouldHaveLength, 1)
			So(ip2[0].GetOccupied(), ShouldBeFalse)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/machinelse-update-host-delete-ip")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 6)
			// nic & vlan & ip info are changed
			So(changes[0].GetEventLabel(), ShouldEqual, "machine_lse.nic")
			So(changes[0].GetOldValue(), ShouldEqual, "")
			So(changes[0].GetNewValue(), ShouldEqual, "eth0-delete")
			So(changes[1].GetEventLabel(), ShouldEqual, "machine_lse.vlan")
			So(changes[1].GetOldValue(), ShouldEqual, "")
			So(changes[1].GetNewValue(), ShouldEqual, "vlan-1")
			So(changes[2].GetEventLabel(), ShouldEqual, "machine_lse.ip")
			So(changes[2].GetOldValue(), ShouldEqual, "")
			So(changes[2].GetNewValue(), ShouldEqual, "192.168.40.12")
			// From deleting host
			So(changes[3].GetEventLabel(), ShouldEqual, "machine_lse.nic")
			So(changes[3].GetOldValue(), ShouldEqual, "eth0-delete")
			So(changes[3].GetNewValue(), ShouldEqual, "")
			So(changes[4].GetEventLabel(), ShouldEqual, "machine_lse.vlan")
			So(changes[4].GetOldValue(), ShouldEqual, "vlan-1")
			So(changes[4].GetNewValue(), ShouldEqual, "")
			So(changes[5].GetEventLabel(), ShouldEqual, "machine_lse.ip")
			So(changes[5].GetOldValue(), ShouldEqual, "192.168.40.12")
			So(changes[5].GetNewValue(), ShouldEqual, "")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "states/hosts/machinelse-update-host-delete-ip")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 2)
			So(changes[0].GetEventLabel(), ShouldEqual, "state_record.state")
			So(changes[0].GetOldValue(), ShouldEqual, ufspb.State_STATE_UNSPECIFIED.String())
			So(changes[0].GetNewValue(), ShouldEqual, ufspb.State_STATE_DEPLOYING.String())
			So(changes[1].GetEventLabel(), ShouldEqual, "state_record.state")
			So(changes[1].GetOldValue(), ShouldEqual, ufspb.State_STATE_DEPLOYING.String())
			So(changes[1].GetNewValue(), ShouldEqual, ufspb.State_STATE_REGISTERED.String())
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "dhcps/machinelse-update-host-delete-ip")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 2)
			So(changes[0].GetEventLabel(), ShouldEqual, "dhcp_config.ip")
			So(changes[0].GetOldValue(), ShouldEqual, "")
			So(changes[0].GetNewValue(), ShouldEqual, "192.168.40.12")
			So(changes[1].GetEventLabel(), ShouldEqual, "dhcp_config.ip")
			So(changes[1].GetOldValue(), ShouldEqual, "192.168.40.12")
			So(changes[1].GetNewValue(), ShouldEqual, "")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", fmt.Sprintf("ips/%s", oldIPs[0].GetId()))
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 2)
			So(changes[0].GetEventLabel(), ShouldEqual, "ip.occupied")
			So(changes[0].GetOldValue(), ShouldEqual, "false")
			So(changes[0].GetNewValue(), ShouldEqual, "true")
			So(changes[1].GetEventLabel(), ShouldEqual, "ip.occupied")
			So(changes[1].GetOldValue(), ShouldEqual, "true")
			So(changes[1].GetNewValue(), ShouldEqual, "false")
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/machinelse-update-host-delete-ip")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 2)
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "states/hosts/machinelse-update-host-delete-ip")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 2)
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "dhcps/machinelse-update-host-delete-ip")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 2)
			So(msgs[1].Delete, ShouldBeTrue)
		})

		Convey("Update machineLSE by setting ip by user for host", func() {
			_, err := registration.CreateNic(ctx, &ufspb.Nic{
				Name:    "eth0-user",
				Machine: "machine-update-host-user",
			})
			So(err, ShouldBeNil)
			machine1 := &ufspb.Machine{
				Name: "machine-update-host-user",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
				},
			}
			machineLSE1 := &ufspb.MachineLSE{
				Name:     "machinelse-update-host-user",
				Hostname: "machinelse-update-host-user",
				Machines: []string{"machine-update-host-user"},
				Lse: &ufspb.MachineLSE_ChromeBrowserMachineLse{
					ChromeBrowserMachineLse: &ufspb.ChromeBrowserMachineLSE{
						Vms: []*ufspb.VM{
							{
								Name:       "vm1",
								MacAddress: "old_mac_address",
							},
						},
					},
				},
			}
			_, err = registration.CreateMachine(ctx, machine1)
			So(err, ShouldBeNil)
			_, err = inventory.CreateMachineLSE(ctx, machineLSE1)
			So(err, ShouldBeNil)
			vlan := &ufspb.Vlan{
				Name:        "vlan-1",
				VlanAddress: "192.168.40.0/22",
			}
			_, err = configuration.CreateVlan(ctx, vlan)
			ips, _, _, _, err := util.ParseVlan(vlan.GetName(), vlan.GetVlanAddress(), vlan.GetFreeStartIpv4Str(), vlan.GetFreeEndIpv4Str())
			So(err, ShouldBeNil)
			// Only import the first 20 as one single transaction cannot import all.
			_, err = configuration.ImportIPs(ctx, ips[0:20])
			So(err, ShouldBeNil)

			_, err = UpdateMachineLSEHost(ctx, machineLSE1.Name, &ufsAPI.NetworkOption{
				Vlan: "vlan-1",
				Ip:   "192.168.40.19",
				Nic:  "eth0-user",
			})
			So(err, ShouldBeNil)
			dhcp, err := configuration.GetDHCPConfig(ctx, "machinelse-update-host-user")
			So(err, ShouldBeNil)
			So(dhcp.GetIp(), ShouldEqual, "192.168.40.19")
			ips, err = configuration.QueryIPByPropertyName(ctx, map[string]string{"ipv4_str": "192.168.40.19"})
			So(err, ShouldBeNil)
			So(ips, ShouldHaveLength, 1)
			So(ips[0].GetOccupied(), ShouldBeTrue)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/machinelse-update-host-user")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 3)
			// nic & vlan & ip info are changed
			So(changes[0].GetEventLabel(), ShouldEqual, "machine_lse.nic")
			So(changes[0].GetOldValue(), ShouldEqual, "")
			So(changes[0].GetNewValue(), ShouldEqual, "eth0-user")
			So(changes[1].GetEventLabel(), ShouldEqual, "machine_lse.vlan")
			So(changes[1].GetOldValue(), ShouldEqual, "")
			So(changes[1].GetNewValue(), ShouldEqual, "vlan-1")
			So(changes[2].GetEventLabel(), ShouldEqual, "machine_lse.ip")
			So(changes[2].GetOldValue(), ShouldEqual, "")
			So(changes[2].GetNewValue(), ShouldEqual, "192.168.40.19")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "states/hosts/machinelse-update-host-user")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetEventLabel(), ShouldEqual, "state_record.state")
			So(changes[0].GetOldValue(), ShouldEqual, "STATE_UNSPECIFIED")
			So(changes[0].GetNewValue(), ShouldEqual, "STATE_DEPLOYING")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "dhcps/machinelse-update-host-user")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetEventLabel(), ShouldEqual, "dhcp_config.ip")
			So(changes[0].GetOldValue(), ShouldEqual, "")
			So(changes[0].GetNewValue(), ShouldEqual, dhcp.GetIp())
			changes, err = history.QueryChangesByPropertyName(ctx, "name", fmt.Sprintf("ips/%s", ips[0].GetId()))
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetEventLabel(), ShouldEqual, "ip.occupied")
			So(changes[0].GetOldValue(), ShouldEqual, "false")
			So(changes[0].GetNewValue(), ShouldEqual, "true")
		})

		Convey("Update machineLSE Labstation without Servo Info", func() {
			machine := &ufspb.Machine{
				Name: "machine-4x",
			}
			_, err := registration.CreateMachine(ctx, machine)
			So(err, ShouldBeNil)

			labstationMachinelse1 := mockLabstationMachineLSE("RedLabstation-11")
			labstationMachinelse1.Machines = []string{"machine-4x"}
			_, err = inventory.CreateMachineLSE(ctx, labstationMachinelse1)
			So(err, ShouldBeNil)

			labstationMachinelse2 := mockLabstationMachineLSE("RedLabstation-11")
			labstationMachinelse2.Machines = []string{"machine-4x"}
			resp, err := UpdateMachineLSE(ctx, labstationMachinelse2, nil)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, labstationMachinelse2)

			/* TODO(eshwarn): Add back when implement labstation state
			s, err := state.GetStateRecord(ctx, "hosts/RedLabstation-11")
			So(err, ShouldBeNil)
			// Sets it back to needs_deloy
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_DEPLOYED_PRE_SERVING)
			*/

			// No changes happened in this update
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/RedLabstation-11")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})

		Convey("Update a machineLSE with a machine already attached to a different machinelse - error", func() {
			machine := &ufspb.Machine{
				Name: "machine-4",
			}
			_, err := registration.CreateMachine(ctx, machine)
			So(err, ShouldBeNil)

			_, err = inventory.CreateMachineLSE(ctx, &ufspb.MachineLSE{
				Name:     "machinelse-4",
				Machines: []string{"machine-4"},
			})
			So(err, ShouldBeNil)

			machine = &ufspb.Machine{
				Name: "machine-5",
			}
			_, err = registration.CreateMachine(ctx, machine)
			So(err, ShouldBeNil)

			_, err = inventory.CreateMachineLSE(ctx, &ufspb.MachineLSE{
				Name:     "machinelse-5",
				Machines: []string{"machine-5"},
			})
			So(err, ShouldBeNil)

			machineLSE := &ufspb.MachineLSE{
				Name:     "machinelse-4",
				Hostname: "machinelse-4",
				Machines: []string{"machine-5"},
			}
			resp, err := UpdateMachineLSE(ctx, machineLSE, nil)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "there is another host machinelse-5 which is referring this machine machine-5")

			_, err = state.GetStateRecord(ctx, "hosts/machinelse-4")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})

		Convey("Partial update machinelse", func() {
			machine := &ufspb.Machine{
				Name: "machine-7",
			}
			_, err := registration.CreateMachine(ctx, machine)
			So(err, ShouldBeNil)

			lse := &ufspb.MachineLSE{
				Name:     "lse-7",
				Tags:     []string{"tag-1"},
				Machines: []string{"machine-7"},
				Lse: &ufspb.MachineLSE_ChromeBrowserMachineLse{
					ChromeBrowserMachineLse: &ufspb.ChromeBrowserMachineLSE{},
				},
			}
			_, err = inventory.CreateMachineLSE(ctx, lse)
			So(err, ShouldBeNil)

			lse1 := &ufspb.MachineLSE{
				Name: "lse-7",
				Tags: []string{"tag-2"},
			}
			resp, err := UpdateMachineLSE(ctx, lse1, &field_mask.FieldMask{Paths: []string{"tags"}})
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp.GetTags(), ShouldResemble, []string{"tag-1", "tag-2"})
			So(resp.GetMachines(), ShouldResemble, []string{"machine-7"})
		})
	})
}

func TestUpdateLabMeta(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("UpdateLabMeta for an OS machine lse", t, func() {
		Convey("Update a non-OS machine lse", func() {
			machineLSE1 := &ufspb.MachineLSE{
				Name:     "machinelse-labmeta-1",
				Hostname: "machinelse-labmeta-1",
				Machines: []string{"machine-labmeta1"},
				Lse: &ufspb.MachineLSE_ChromeBrowserMachineLse{
					ChromeBrowserMachineLse: &ufspb.ChromeBrowserMachineLSE{},
				},
			}
			_, err := registration.CreateMachine(ctx, &ufspb.Machine{
				Name: "machine-labmeta1",
			})
			So(err, ShouldBeNil)
			_, err = inventory.CreateMachineLSE(ctx, machineLSE1)
			So(err, ShouldBeNil)

			err = UpdateLabMeta(ctx, &ufspb.LabMeta{
				ChromeosDeviceId: "machine-labmeta1",
				Hostname:         "machinelse-labmeta-1",
				SmartUsbhub:      true,
			})
			// Update is skipped without error
			So(err, ShouldBeNil)
		})

		Convey("Update a OS machine lse - happy path", func() {
			machineLSE1 := mockDutMachineLSE("machinelse-labmeta-2")
			machineLSE1.GetChromeosMachineLse().GetDeviceLse().GetDut().Peripherals = &chromeosLab.Peripherals{
				Servo: &chromeosLab.Servo{},
			}
			req, err := inventory.CreateMachineLSE(ctx, machineLSE1)
			So(err, ShouldBeNil)
			So(req.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetSmartUsbhub(), ShouldBeFalse)

			topology := &chromeosLab.ServoTopology{
				Main: &chromeosLab.ServoTopologyItem{
					Type: "v3",
				},
			}
			err = UpdateLabMeta(ctx, &ufspb.LabMeta{
				ChromeosDeviceId: "machine-labmeta1",
				Hostname:         "machinelse-labmeta-2",
				SmartUsbhub:      true,
				ServoType:        "fake-type",
				ServoTopology:    topology,
			})
			So(err, ShouldBeNil)
			req, err = inventory.GetMachineLSE(ctx, "machinelse-labmeta-2")
			So(err, ShouldBeNil)
			peri := req.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals()
			So(peri.GetSmartUsbhub(), ShouldBeTrue)
			So(peri.Servo.GetServoType(), ShouldEqual, "fake-type")
			So(peri.Servo.GetServoTopology(), ShouldResembleProto, topology)
		})
	})
}

func TestDeleteMachineLSE(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("DeleteMachineLSE", t, func() {
		Convey("Delete machineLSE - happy path", func() {
			machineLSE1 := &ufspb.MachineLSE{
				Name:     "machinelse-delete-1",
				Hostname: "machinelse-delete-1",
				Machines: []string{"machine-delete-1"},
				Lse: &ufspb.MachineLSE_ChromeBrowserMachineLse{
					ChromeBrowserMachineLse: &ufspb.ChromeBrowserMachineLSE{
						Vms: []*ufspb.VM{
							{
								Name:       "vm-delete-1",
								MacAddress: "old_mac_address",
							},
						},
					},
				},
			}
			_, err := registration.CreateMachine(ctx, &ufspb.Machine{
				Name:         "machine-delete-1",
				SerialNumber: "machine-delete-1-serial",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
				},
			})
			So(err, ShouldBeNil)
			_, err = CreateMachineLSE(ctx, machineLSE1, nil)
			So(err, ShouldBeNil)

			err = DeleteMachineLSE(ctx, "machinelse-delete-1")
			So(err, ShouldBeNil)
			_, err = inventory.GetMachineLSEDeployment(ctx, "machine-delete-1-serial")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)

			// verify changes
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/machinelse-delete-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 2)
			So(changes[1].GetOldValue(), ShouldEqual, LifeCycleRetire)
			So(changes[1].GetNewValue(), ShouldEqual, LifeCycleRetire)
			So(changes[1].GetEventLabel(), ShouldEqual, "machine_lse")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "vms/vm-delete-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 2)
			So(changes[1].GetOldValue(), ShouldEqual, LifeCycleRetire)
			So(changes[1].GetNewValue(), ShouldEqual, LifeCycleRetire)
			So(changes[1].GetEventLabel(), ShouldEqual, "vm")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "states/vms/vm-delete-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 2)
			So(changes[1].GetEventLabel(), ShouldEqual, "state_record.state")
			So(changes[1].GetOldValue(), ShouldEqual, ufspb.State_STATE_REGISTERED.String())
			So(changes[1].GetNewValue(), ShouldEqual, ufspb.State_STATE_UNSPECIFIED.String())
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "states/hosts/machinelse-delete-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 2)
			So(changes[1].GetEventLabel(), ShouldEqual, "state_record.state")
			So(changes[1].GetOldValue(), ShouldEqual, ufspb.State_STATE_REGISTERED.String())
			So(changes[1].GetNewValue(), ShouldEqual, ufspb.State_STATE_UNSPECIFIED.String())
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "states/machines/machine-delete-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 2)
			So(changes[1].GetEventLabel(), ShouldEqual, "state_record.state")
			So(changes[1].GetOldValue(), ShouldEqual, ufspb.State_STATE_SERVING.String())
			So(changes[1].GetNewValue(), ShouldEqual, ufspb.State_STATE_REGISTERED.String())
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "dhcps/machinelse-delete-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/machinelse-delete-1")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 2)
			So(msgs[0].Delete, ShouldBeFalse)
			So(msgs[1].Delete, ShouldBeTrue)
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "vms/vm-delete-1")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 2)
			So(msgs[0].Delete, ShouldBeFalse)
			So(msgs[1].Delete, ShouldBeTrue)
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "states/vms/vm-delete-1")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 2)
			So(msgs[0].Delete, ShouldBeFalse)
			So(msgs[1].Delete, ShouldBeTrue)
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "states/hosts/machinelse-delete-1")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 2)
			So(msgs[0].Delete, ShouldBeFalse)
			So(msgs[1].Delete, ShouldBeTrue)
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "states/machines/machine-delete-1")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 2)
			So(msgs[0].Delete, ShouldBeFalse)
			So(msgs[1].Delete, ShouldBeFalse)
		})

		Convey("Delete machineLSE with existing deployment record - happy path", func() {
			machineLSE1 := &ufspb.MachineLSE{
				Name:     "machinelse-delete-2",
				Hostname: "machinelse-delete-2",
				Machines: []string{"machine-delete-2"},
				Lse: &ufspb.MachineLSE_ChromeBrowserMachineLse{
					ChromeBrowserMachineLse: &ufspb.ChromeBrowserMachineLSE{},
				},
			}
			_, err := registration.CreateMachine(ctx, &ufspb.Machine{
				Name:         "machine-delete-2",
				SerialNumber: "machine-delete-2-serial",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
				},
			})
			So(err, ShouldBeNil)
			_, err = CreateMachineLSE(ctx, machineLSE1, nil)
			So(err, ShouldBeNil)
			_, err = inventory.UpdateMachineLSEDeployments(ctx, []*ufspb.MachineLSEDeployment{util.FormatDeploymentRecord("machinelse-delete-2", "machine-delete-2-serial")})
			So(err, ShouldBeNil)
			_, err = inventory.GetMachineLSEDeployment(ctx, "machine-delete-2-serial")
			So(err, ShouldBeNil)

			err = DeleteMachineLSE(ctx, "machinelse-delete-2")
			So(err, ShouldBeNil)
			_, err = inventory.GetMachineLSEDeployment(ctx, "machine-delete-2-serial")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
	})
}

func TestDeleteMachineLSEDUT(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("DeleteMachineLSE for a DUT", t, func() {
		Convey("Delete machineLSE DUT with Servo Info", func() {
			machine := &ufspb.Machine{
				Name: "machine-1",
			}
			_, err := registration.CreateMachine(ctx, machine)
			So(err, ShouldBeNil)

			servo := &chromeosLab.Servo{
				ServoHostname: "RedLabstation-92",
				ServoPort:     92,
				ServoSerial:   "RedSerial-92",
			}

			labstationMachinelse := mockLabstationMachineLSE("RedLabstation-92")
			labstationMachinelse.GetChromeosMachineLse().GetDeviceLse().GetLabstation().Servos = []*chromeosLab.Servo{servo}
			_, err = inventory.CreateMachineLSE(ctx, labstationMachinelse)
			So(err, ShouldBeNil)

			peripherals := &chromeosLab.Peripherals{
				Servo: servo,
			}
			dutMachinelse := mockDutMachineLSE("DUTMachineLse-92")
			dutMachinelse.Machines = []string{"machine-1"}
			dutMachinelse.GetChromeosMachineLse().GetDeviceLse().GetDut().Peripherals = peripherals
			_, err = inventory.CreateMachineLSE(ctx, dutMachinelse)
			So(err, ShouldBeNil)
			_, err = state.UpdateStateRecord(ctx, &ufspb.StateRecord{
				State:        ufspb.State_STATE_SERVING,
				ResourceName: "hosts/DUTMachineLse-92",
			})
			So(err, ShouldBeNil)

			err = DeleteMachineLSE(ctx, "DUTMachineLse-92")
			So(err, ShouldBeNil)

			_, err = state.GetStateRecord(ctx, "hosts/DUTMachineLse-92")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)

			resp, _ := inventory.GetMachineLSE(ctx, "RedLabstation-92")
			So(resp.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetServos(), ShouldBeEmpty)

			// verify changes
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/RedLabstation-92")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetEventLabel(), ShouldEqual, "machine_lse.chromeos_machine_lse.labstation.servos")
			So(changes[0].GetOldValue(), ShouldContainSubstring, servo.ServoHostname)
			So(changes[0].GetNewValue(), ShouldEqual, "[]")
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/RedLabstation-92")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
		})

		Convey("Delete machineLSE DUT with non existing Servo Info", func() {
			// Attempt to delete a misconfigured DUT with servo host that doesn't exist. Controller should log an error
			// as this means that somewhere DUT/Labstation was misconfigured. But delete should not fail
			machine := &ufspb.Machine{
				Name: "machine-2",
			}
			_, err := registration.CreateMachine(ctx, machine)
			So(err, ShouldBeNil)

			servo := &chromeosLab.Servo{
				ServoHostname: "RedLabstation-93",
				ServoPort:     9996,
				ServoSerial:   "RedSerial-93",
			}

			peripherals := &chromeosLab.Peripherals{
				Servo: servo,
			}
			dutMachinelse := mockDutMachineLSE("DUTMachineLse-93")
			dutMachinelse.Machines = []string{"machine-2"}
			dutMachinelse.GetChromeosMachineLse().GetDeviceLse().GetDut().Peripherals = peripherals
			_, err = inventory.CreateMachineLSE(ctx, dutMachinelse)
			So(err, ShouldBeNil)
			_, err = state.UpdateStateRecord(ctx, &ufspb.StateRecord{
				State:        ufspb.State_STATE_SERVING,
				ResourceName: "hosts/DUTMachineLse-93",
			})
			So(err, ShouldBeNil)

			err = DeleteMachineLSE(ctx, "DUTMachineLse-93")
			So(err, ShouldBeNil)

			_, err = state.GetStateRecord(ctx, "hosts/DUTMachineLse-93")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)

			// Ensure that the labstation doesn't exist
			_, err = inventory.GetMachineLSE(ctx, "RedLabstation-93")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)

			// verify that no changes were recorded for labstation
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/RedLabstation-93")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/RedLabstation-93")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 0)
		})
	})
}

func TestDeleteMachineLSELabstation(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("DeleteMachineLSE for a Labstation", t, func() {
		Convey("Delete machineLSE Labstation with Servo Info", func() {
			machine := &ufspb.Machine{
				Name: "machine-90",
			}
			_, err := registration.CreateMachine(ctx, machine)
			So(err, ShouldBeNil)

			servo := &chromeosLab.Servo{
				ServoHostname: "RedLabstation-90",
				ServoPort:     90,
			}
			labstationMachinelse := mockLabstationMachineLSE("RedLabstation-90")
			labstationMachinelse.Machines = []string{"machine-90"}
			labstationMachinelse.GetChromeosMachineLse().GetDeviceLse().GetLabstation().Servos = []*chromeosLab.Servo{servo}
			_, err = inventory.CreateMachineLSE(ctx, labstationMachinelse)
			So(err, ShouldBeNil)

			peripherals := &chromeosLab.Peripherals{
				Servo: servo,
			}
			dutMachinelse := mockDutMachineLSE("DUTMachineLSE-90")
			dutMachinelse.GetChromeosMachineLse().GetDeviceLse().GetDut().Peripherals = peripherals
			_, err = inventory.CreateMachineLSE(ctx, dutMachinelse)
			So(err, ShouldBeNil)
			_, err = state.UpdateStateRecord(ctx, &ufspb.StateRecord{
				State:        ufspb.State_STATE_SERVING,
				ResourceName: "hosts/RedLabstation-90",
			})
			So(err, ShouldBeNil)

			err = DeleteMachineLSE(ctx, "RedLabstation-90")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "cannot be deleted")
			s, err := state.GetStateRecord(ctx, "hosts/RedLabstation-90")
			So(err, ShouldBeNil)
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_SERVING)

			// No changes are recorded as the deletion fails
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/RedLabstation-90")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})
		Convey("Delete machineLSE Labstation without Servo Info", func() {
			machine := &ufspb.Machine{
				Name: "machine-100",
			}
			_, err := registration.CreateMachine(ctx, machine)
			So(err, ShouldBeNil)

			labstationMachinelse := mockLabstationMachineLSE("RedLabstation-100")
			labstationMachinelse.Machines = []string{"machine-100"}
			inventory.CreateMachineLSE(ctx, labstationMachinelse)
			_, err = state.UpdateStateRecord(ctx, &ufspb.StateRecord{
				State:        ufspb.State_STATE_SERVING,
				ResourceName: "hosts/RedLabstation-100",
			})
			So(err, ShouldBeNil)

			err = DeleteMachineLSE(ctx, "RedLabstation-100")
			So(err, ShouldBeNil)
			_, err = state.GetStateRecord(ctx, "hosts/RedLabstation-100")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/RedLabstation-100")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRetire)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRetire)
			So(changes[0].GetEventLabel(), ShouldEqual, "machine_lse")
			// Both states for old & new are unspecified.
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "states/hosts/RedLabstation-100")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetEventLabel(), ShouldEqual, "state_record.state")
			So(changes[0].GetOldValue(), ShouldEqual, ufspb.State_STATE_SERVING.String())
			So(changes[0].GetNewValue(), ShouldEqual, ufspb.State_STATE_UNSPECIFIED.String())
		})
	})
}

func TestListMachineLSEs(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	machineLSEsWithSwitch := make([]*ufspb.MachineLSE, 0, 2)
	machineLSEs := make([]*ufspb.MachineLSE, 0, 4)
	for i := 0; i < 4; i++ {
		machineLSE := mockDutMachineLSE(fmt.Sprintf("machineLSE-%d", i))
		if i%2 == 0 {
			machineLSE.GetChromeosMachineLse().GetDeviceLse().NetworkDeviceInterface = &ufspb.SwitchInterface{Switch: "switch-1"}
		}
		resp, _ := inventory.CreateMachineLSE(ctx, machineLSE)
		if i%2 == 0 {
			machineLSEsWithSwitch = append(machineLSEsWithSwitch, resp)
		}
		machineLSEs = append(machineLSEs, resp)
	}
	Convey("ListMachineLSEs", t, func() {
		Convey("List MachineLSEs - filter invalid - error", func() {
			_, _, err := ListMachineLSEs(ctx, 5, "", "invalid=mx-1", false, false)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Invalid field name invalid")
		})

		Convey("List MachineLSEs - filter switch - happy path with filter", func() {
			resp, _, _ := ListMachineLSEs(ctx, 5, "", "switch=switch-1", false, false)
			So(resp, ShouldNotBeNil)
			So(resp, ShouldResembleProto, machineLSEsWithSwitch)
		})

		Convey("ListMachineLSEs - Full listing - happy path", func() {
			resp, _, _ := ListMachineLSEs(ctx, 5, "", "", false, false)
			So(resp, ShouldNotBeNil)
			So(resp, ShouldResembleProto, machineLSEs)
		})
		Convey("List machineLSEs - list machine lses with free slots", func() {
			for i := 0; i < 8; i++ {
				machineLSE1 := &ufspb.MachineLSE{
					Name: fmt.Sprintf("machineLSE-free-%d", i),
				}
				machineLSE1.Manufacturer = "apple"
				var vmCapacity int32
				if i > 4 && i <= 6 {
					vmCapacity = int32(i - 4)
				}
				machineLSE1.Lse = &ufspb.MachineLSE_ChromeBrowserMachineLse{
					ChromeBrowserMachineLse: &ufspb.ChromeBrowserMachineLSE{
						VmCapacity: vmCapacity,
					},
				}
				inventory.CreateMachineLSE(ctx, machineLSE1)
			}
			fields := make([]interface{}, 1)
			fields[0] = "apple"
			resp, nextPageToken, err := ListMachineLSEs(ctx, 4, "", "man=apple & free=true", false, false)
			So(resp, ShouldNotBeNil)
			So(nextPageToken, ShouldBeEmpty)
			So(err, ShouldBeNil)
			So(resp, ShouldHaveLength, 2)
			for _, r := range resp {
				So(r.GetName(), ShouldBeIn, []string{"machineLSE-free-5", "machineLSE-free-6"})
			}
		})
	})
}

func TestBatchGetMachineLSEs(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("BatchGetMachineLSEs", t, func() {
		Convey("Batch get machine lses - happy path", func() {
			lses := make([]*ufspb.MachineLSE, 4)
			for i := 0; i < 4; i++ {
				lse := &ufspb.MachineLSE{
					Name: fmt.Sprintf("lse-batchGet-%d", i),
				}
				lses[i] = lse
			}
			_, err := inventory.BatchUpdateMachineLSEs(ctx, lses)
			So(err, ShouldBeNil)
			resp, err := inventory.BatchGetMachineLSEs(ctx, []string{"lse-batchGet-0", "lse-batchGet-1", "lse-batchGet-2", "lse-batchGet-3"})
			So(err, ShouldBeNil)
			So(resp, ShouldHaveLength, 4)
			So(resp, ShouldResembleProto, lses)
		})
		Convey("Batch get machine lses - missing id", func() {
			resp, err := inventory.BatchGetMachineLSEs(ctx, []string{"lse-batchGet-non-existing"})
			So(err, ShouldNotBeNil)
			So(resp, ShouldBeNil)
			So(err.Error(), ShouldContainSubstring, "lse-batchGet-non-existing")
		})
		Convey("Batch get machine lses - empty input", func() {
			resp, err := inventory.BatchGetMachineLSEs(ctx, nil)
			So(err, ShouldBeNil)
			So(resp, ShouldHaveLength, 0)

			input := make([]string, 0)
			resp, err = inventory.BatchGetMachineLSEs(ctx, input)
			So(err, ShouldBeNil)
			So(resp, ShouldHaveLength, 0)
		})
	})
}

func TestRealmPermissionForMachineLSE(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("TestRealmPermissionForMachineLSE", t, func() {
		Convey("CreateMachineLSE with permission - pass", func() {
			_, err := registration.CreateMachine(ctx, &ufspb.Machine{
				Name:  "machine-1",
				Realm: util.BrowserLabAdminRealm,
			})
			So(err, ShouldBeNil)

			mlse := &ufspb.MachineLSE{
				Name:     "machinelse-1",
				Machines: []string{"machine-1"},
				Hostname: "machinelse-1",
			}

			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.InventoriesCreate, util.BrowserLabAdminRealm)
			resp, err := CreateMachineLSE(ctx, mlse, nil)
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp, ShouldResembleProto, mlse)
		})

		Convey("CreateMachineLSE without permission - fail", func() {
			_, err := registration.CreateMachine(ctx, &ufspb.Machine{
				Name:  "machine-2",
				Realm: util.BrowserLabAdminRealm,
			})
			So(err, ShouldBeNil)

			mlse := &ufspb.MachineLSE{
				Name:     "machinelse-2",
				Machines: []string{"machine-2"},
			}

			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.InventoriesUpdate, util.AtlLabAdminRealm)
			_, err = CreateMachineLSE(ctx, mlse, nil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, PermissionDenied)
		})

		Convey("DeleteMachineLSE with permission - pass", func() {
			_, err := registration.CreateMachine(ctx, &ufspb.Machine{
				Name:  "machine-3",
				Realm: util.BrowserLabAdminRealm,
			})
			So(err, ShouldBeNil)

			_, err = inventory.CreateMachineLSE(ctx, &ufspb.MachineLSE{
				Name:     "machinelse-3",
				Machines: []string{"machine-3"},
				Hostname: "machinelse-3",
			})
			So(err, ShouldBeNil)

			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.InventoriesDelete, util.BrowserLabAdminRealm)
			err = DeleteMachineLSE(ctx, "machinelse-3")
			So(err, ShouldBeNil)
		})

		Convey("DeleteMachineLSE without permission - fail", func() {
			_, err := registration.CreateMachine(ctx, &ufspb.Machine{
				Name:  "machine-4",
				Realm: util.BrowserLabAdminRealm,
			})
			So(err, ShouldBeNil)

			_, err = inventory.CreateMachineLSE(ctx, &ufspb.MachineLSE{
				Name:     "machinelse-4",
				Machines: []string{"machine-4"},
				Hostname: "machinelse-4",
			})
			So(err, ShouldBeNil)

			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.InventoriesDelete, util.AtlLabAdminRealm)
			err = DeleteMachineLSE(ctx, "machinelse-4")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, PermissionDenied)
		})

		Convey("UpdateMachineLSE with permission - pass", func() {
			_, err := registration.CreateMachine(ctx, &ufspb.Machine{
				Name:  "machine-5",
				Realm: util.BrowserLabAdminRealm,
			})
			So(err, ShouldBeNil)

			mlse, err := inventory.CreateMachineLSE(ctx, &ufspb.MachineLSE{
				Name:     "machinelse-5",
				Machines: []string{"machine-5"},
				Hostname: "machinelse-5",
			})
			So(err, ShouldBeNil)

			mlse.Tags = []string{"Dell"}
			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.InventoriesUpdate, util.BrowserLabAdminRealm)
			resp, err := UpdateMachineLSE(ctx, mlse, nil)
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp.Tags, ShouldResemble, []string{"Dell"})
		})

		Convey("UpdateMachineLSE without permission - fail", func() {
			_, err := registration.CreateMachine(ctx, &ufspb.Machine{
				Name:  "machine-6",
				Realm: util.BrowserLabAdminRealm,
			})
			So(err, ShouldBeNil)

			mlse, err := inventory.CreateMachineLSE(ctx, &ufspb.MachineLSE{
				Name:     "machinelse-6",
				Machines: []string{"machine-6"},
				Hostname: "machinelse-6",
			})
			So(err, ShouldBeNil)

			mlse.Tags = []string{"Dell"}
			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.InventoriesUpdate, util.AtlLabAdminRealm)
			_, err = UpdateMachineLSE(ctx, mlse, nil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, PermissionDenied)
		})

		Convey("UpdateMachineLSE(new machine and same realm) with permission - pass", func() {
			_, err := registration.CreateMachine(ctx, &ufspb.Machine{
				Name:  "machine-7",
				Realm: util.BrowserLabAdminRealm,
			})
			So(err, ShouldBeNil)

			mlse, err := inventory.CreateMachineLSE(ctx, &ufspb.MachineLSE{
				Name:     "machinelse-7",
				Machines: []string{"machine-7"},
				Hostname: "machinelse-7",
			})
			So(err, ShouldBeNil)

			_, err = registration.CreateMachine(ctx, &ufspb.Machine{
				Name:  "machine-7.1",
				Realm: util.BrowserLabAdminRealm,
			})
			So(err, ShouldBeNil)

			mlse.Machines = []string{"machine-7.1"}
			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.InventoriesUpdate, util.BrowserLabAdminRealm)
			resp, err := UpdateMachineLSE(ctx, mlse, nil)
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp.Machines, ShouldResemble, []string{"machine-7.1"})
		})

		Convey("UpdateMachineLSE(new machine and different realm) without permission - fail", func() {
			_, err := registration.CreateMachine(ctx, &ufspb.Machine{
				Name:  "machine-8",
				Realm: util.BrowserLabAdminRealm,
			})
			So(err, ShouldBeNil)

			mlse, err := inventory.CreateMachineLSE(ctx, &ufspb.MachineLSE{
				Name:     "machinelse-8",
				Machines: []string{"machine-8"},
				Hostname: "machinelse-8",
			})
			So(err, ShouldBeNil)

			_, err = registration.CreateMachine(ctx, &ufspb.Machine{
				Name:  "machine-8.1",
				Realm: util.AtlLabAdminRealm,
			})
			So(err, ShouldBeNil)

			mlse.Machines = []string{"machine-8.1"}
			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.InventoriesUpdate, util.BrowserLabAdminRealm)
			_, err = UpdateMachineLSE(ctx, mlse, nil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, PermissionDenied)
		})

		Convey("UpdateMachineLSE(new machine and different realm) with permission - pass", func() {
			_, err := registration.CreateMachine(ctx, &ufspb.Machine{
				Name:  "machine-9",
				Realm: util.BrowserLabAdminRealm,
			})
			So(err, ShouldBeNil)

			mlse, err := inventory.CreateMachineLSE(ctx, &ufspb.MachineLSE{
				Name:     "machinelse-9",
				Machines: []string{"machine-9"},
				Hostname: "machinelse-9",
			})
			So(err, ShouldBeNil)

			_, err = registration.CreateMachine(ctx, &ufspb.Machine{
				Name:  "machine-9.1",
				Realm: util.AtlLabAdminRealm,
			})
			So(err, ShouldBeNil)

			mlse.Machines = []string{"machine-9.1"}
			ctx := auth.WithState(ctx, &authtest.FakeState{
				Identity: "user:user@example.com",
				FakeDB: authtest.NewFakeDB(
					authtest.MockMembership("user:user@example.com", "user"),
					authtest.MockPermission("user:user@example.com", util.AtlLabAdminRealm, util.InventoriesUpdate),
					authtest.MockPermission("user:user@example.com", util.BrowserLabAdminRealm, util.InventoriesUpdate),
				),
			})
			resp, err := UpdateMachineLSE(ctx, mlse, nil)
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp.Machines, ShouldResemble, []string{"machine-9.1"})
		})

		Convey("Partial UpdateMachineLSE with permission - pass", func() {
			_, err := registration.CreateMachine(ctx, &ufspb.Machine{
				Name:  "machine-10",
				Realm: util.BrowserLabAdminRealm,
			})
			So(err, ShouldBeNil)

			mlse, err := inventory.CreateMachineLSE(ctx, &ufspb.MachineLSE{
				Name:     "machinelse-10",
				Machines: []string{"machine-10"},
				Hostname: "machinelse-10",
			})
			So(err, ShouldBeNil)

			mlse.Tags = []string{"Dell"}
			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.InventoriesUpdate, util.BrowserLabAdminRealm)
			resp, err := UpdateMachineLSE(ctx, mlse, &field_mask.FieldMask{Paths: []string{"tags"}})
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp.Tags, ShouldResemble, []string{"Dell"})
		})

		Convey("Partial UpdateMachineLSE without permission - fail", func() {
			_, err := registration.CreateMachine(ctx, &ufspb.Machine{
				Name:  "machine-11",
				Realm: util.BrowserLabAdminRealm,
			})
			So(err, ShouldBeNil)

			mlse, err := inventory.CreateMachineLSE(ctx, &ufspb.MachineLSE{
				Name:     "machinelse-11",
				Machines: []string{"machine-11"},
				Hostname: "machinelse-11",
			})
			So(err, ShouldBeNil)

			mlse.Tags = []string{"Dell"}
			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.InventoriesUpdate, util.AtlLabAdminRealm)
			_, err = UpdateMachineLSE(ctx, mlse, &field_mask.FieldMask{Paths: []string{"tags"}})
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, PermissionDenied)
		})

		Convey("Partial UpdateMachineLSE(new machine and same realm) with permission - pass", func() {
			_, err := registration.CreateMachine(ctx, &ufspb.Machine{
				Name:  "machine-12",
				Realm: util.BrowserLabAdminRealm,
			})
			So(err, ShouldBeNil)

			mlse, err := inventory.CreateMachineLSE(ctx, &ufspb.MachineLSE{
				Name:     "machinelse-12",
				Machines: []string{"machine-12"},
				Hostname: "machinelse-12",
			})
			So(err, ShouldBeNil)

			_, err = registration.CreateMachine(ctx, &ufspb.Machine{
				Name:  "machine-12.1",
				Realm: util.BrowserLabAdminRealm,
			})
			So(err, ShouldBeNil)

			mlse.Machines = []string{"machine-12.1"}
			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.InventoriesUpdate, util.BrowserLabAdminRealm)
			resp, err := UpdateMachineLSE(ctx, mlse, &field_mask.FieldMask{Paths: []string{"machines"}})
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp.Machines, ShouldResemble, []string{"machine-12.1"})
		})

		Convey("Partial UpdateMachineLSE(new machine and different realm) without permission - fail", func() {
			_, err := registration.CreateMachine(ctx, &ufspb.Machine{
				Name:  "machine-13",
				Realm: util.BrowserLabAdminRealm,
			})
			So(err, ShouldBeNil)

			mlse, err := inventory.CreateMachineLSE(ctx, &ufspb.MachineLSE{
				Name:     "machinelse-13",
				Machines: []string{"machine-13"},
				Hostname: "machinelse-13",
			})
			So(err, ShouldBeNil)

			_, err = registration.CreateMachine(ctx, &ufspb.Machine{
				Name:  "machine-13.1",
				Realm: util.AtlLabAdminRealm,
			})
			So(err, ShouldBeNil)

			mlse.Machines = []string{"machine-13.1"}
			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.InventoriesUpdate, util.BrowserLabAdminRealm)
			_, err = UpdateMachineLSE(ctx, mlse, &field_mask.FieldMask{Paths: []string{"machines"}})
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, PermissionDenied)
		})

		Convey("Partial UpdateMachineLSE(new machine and different realm) with permission - pass", func() {
			_, err := registration.CreateMachine(ctx, &ufspb.Machine{
				Name:  "machine-14",
				Realm: util.BrowserLabAdminRealm,
			})
			So(err, ShouldBeNil)

			mlse, err := inventory.CreateMachineLSE(ctx, &ufspb.MachineLSE{
				Name:     "machinelse-14",
				Machines: []string{"machine-14"},
				Hostname: "machinelse-14",
			})
			So(err, ShouldBeNil)

			_, err = registration.CreateMachine(ctx, &ufspb.Machine{
				Name:  "machine-14.1",
				Realm: util.AtlLabAdminRealm,
			})
			So(err, ShouldBeNil)

			mlse.Machines = []string{"machine-14.1"}
			ctx := auth.WithState(ctx, &authtest.FakeState{
				Identity: "user:user@example.com",
				FakeDB: authtest.NewFakeDB(
					authtest.MockMembership("user:user@example.com", "user"),
					authtest.MockPermission("user:user@example.com", util.AtlLabAdminRealm, util.InventoriesUpdate),
					authtest.MockPermission("user:user@example.com", util.BrowserLabAdminRealm, util.InventoriesUpdate),
				),
			})
			resp, err := UpdateMachineLSE(ctx, mlse, &field_mask.FieldMask{Paths: []string{"machines"}})
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp.Machines, ShouldResemble, []string{"machine-14.1"})
		})

	})
}
