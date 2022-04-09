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
	"google.golang.org/grpc/codes"

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

func mockAttachedDeviceMachineLSE(name string) *ufspb.MachineLSE {
	return &ufspb.MachineLSE{
		Name:     name,
		Hostname: name,
		Lse: &ufspb.MachineLSE_AttachedDeviceLse{
			AttachedDeviceLse: &ufspb.AttachedDeviceLSE{
				OsVersion: &ufspb.OSVersion{
					Value: "test",
				},
			},
		},
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
			configuration.CreateVlan(ctx, vlan)
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

			// Verify the change events
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/machinelse-3")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetEventLabel(), ShouldEqual, "machine_lse")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "machineLSEDeployments/machine-3-serial")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetEventLabel(), ShouldEqual, "machine_lse_deployment")
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "machineLSEDeployments/machine-3-serial")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			So(msgs[0].Delete, ShouldBeFalse)
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

			// Verify the change events
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "machineLSEDeployments/machine-dr-3-serial")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, "no-host-yet-machine-dr-3-serial")
			So(changes[0].GetNewValue(), ShouldEqual, "machinelse-dr-3")
			So(changes[0].GetEventLabel(), ShouldEqual, "machine_lse_deployment.hostname")
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "machineLSEDeployments/machine-dr-3-serial")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			So(msgs[0].Delete, ShouldBeFalse)
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

	inventory.GetMachineLSE(ctx, "BlueLabstation-10")
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

		Convey("Update machineLSE DUT with Pool Names", func() {
			dutMachinelse3 := mockDutMachineLSE("DUTMachineLSE-22")
			dutMachinelse3.Machines = []string{"machine-22"}
			dutMachinelse3.GetChromeosMachineLse().GetDeviceLse().GetDut().Pools = []string{"pool1", "pool2"}
			dutMachinelse3.ResourceState = ufspb.State_STATE_SERVING
			resp, err := UpdateMachineLSE(ctx, dutMachinelse3, nil)
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp, ShouldResembleProto, dutMachinelse3)
		})

		Convey("Update machineLSE DUT with Invalid Pool Names", func() {
			dutMachinelse3 := mockDutMachineLSE("DUTMachineLSE-22")
			dutMachinelse3.Machines = []string{"machine-22"}
			dutMachinelse3.GetChromeosMachineLse().GetDeviceLse().GetDut().Pools = []string{"\"pool1", "pool2"}
			dutMachinelse3.ResourceState = ufspb.State_STATE_SERVING
			resp, err := UpdateMachineLSE(ctx, dutMachinelse3, nil)
			So(err, ShouldNotBeNil)
			So(resp, ShouldBeNil)
			So(err.Error(), ShouldContainSubstring, "Invalid Pool Name")
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

		Convey("Update machineLSE Labstation with Pool Names", func() {
			machine := &ufspb.Machine{
				Name: "machine-12",
			}
			_, err := registration.CreateMachine(ctx, machine)
			So(err, ShouldBeNil)

			labstationMachinelse1 := mockLabstationMachineLSE("RedLabstation-12")
			labstationMachinelse1.Machines = []string{"machine-12"}
			inventory.CreateMachineLSE(ctx, labstationMachinelse1)

			labstationMachinelse2 := mockLabstationMachineLSE("RedLabstation-12")
			labstationMachinelse2.Machines = []string{"machine-12"}
			labstationMachinelse2.GetChromeosMachineLse().GetDeviceLse().GetLabstation().Pools = []string{"pool1", "pool2"}
			resp, err := UpdateMachineLSE(ctx, labstationMachinelse2, nil)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, labstationMachinelse2)
		})

		Convey("Update machineLSE Labstation with invalid Pool Names", func() {
			machine := &ufspb.Machine{
				Name: "machine-13",
			}
			_, err := registration.CreateMachine(ctx, machine)
			So(err, ShouldBeNil)

			labstationMachinelse1 := mockLabstationMachineLSE("RedLabstation-13")
			labstationMachinelse1.Machines = []string{"machine-13"}
			inventory.CreateMachineLSE(ctx, labstationMachinelse1)

			labstationMachinelse2 := mockLabstationMachineLSE("RedLabstation-13")
			labstationMachinelse2.Machines = []string{"machine-13"}
			labstationMachinelse2.GetChromeosMachineLse().GetDeviceLse().GetLabstation().Pools = []string{"-pool1", "pool2"}
			resp, err := UpdateMachineLSE(ctx, labstationMachinelse2, nil)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Invalid Pool Name")
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
			configuration.CreateVlan(ctx, vlan)
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
			configuration.CreateVlan(ctx, vlan)
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
			configuration.CreateVlan(ctx, vlan)
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

		Convey("Update machineLSE nic for host with existing dhcp record", func() {
			nic0, err := registration.CreateNic(ctx, &ufspb.Nic{
				Name:       "eth0-nic-user",
				Machine:    "machine-update-host-nic-user",
				MacAddress: "eth0-nic-macaddress",
			})
			So(err, ShouldBeNil)
			_, err = registration.CreateNic(ctx, &ufspb.Nic{
				Name:       "eth1-nic-user",
				Machine:    "machine-update-host-nic-user",
				MacAddress: "eth1-nic-macaddress",
			})
			So(err, ShouldBeNil)
			machine1 := &ufspb.Machine{
				Name: "machine-update-host-nic-user",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
				},
			}
			machineLSE1 := &ufspb.MachineLSE{
				Name:     "machinelse-update-host-nic-user",
				Hostname: "machinelse-update-host-nic-user",
				Machines: []string{"machine-update-host-nic-user"},
				Lse: &ufspb.MachineLSE_ChromeBrowserMachineLse{
					ChromeBrowserMachineLse: &ufspb.ChromeBrowserMachineLSE{},
				},
				Nic: "eth0-nic-user",
			}
			_, err = registration.CreateMachine(ctx, machine1)
			So(err, ShouldBeNil)
			_, err = inventory.CreateMachineLSE(ctx, machineLSE1)
			So(err, ShouldBeNil)
			dhcp := &ufspb.DHCPConfig{
				Hostname:   machineLSE1.GetName(),
				Ip:         "fake_ip",
				Vlan:       "fake_vlan",
				MacAddress: nic0.GetMacAddress(),
			}
			_, err = configuration.BatchUpdateDHCPs(ctx, []*ufspb.DHCPConfig{dhcp})
			So(err, ShouldBeNil)
			ip := &ufspb.IP{
				Id:       "test_ip_id",
				Ipv4:     12345,
				Ipv4Str:  dhcp.GetIp(),
				Vlan:     dhcp.GetVlan(),
				Occupied: true,
			}
			_, err = configuration.BatchUpdateIPs(ctx, []*ufspb.IP{ip})
			So(err, ShouldBeNil)

			_, err = UpdateMachineLSEHost(ctx, machineLSE1.Name, &ufsAPI.NetworkOption{
				Nic: "eth1-nic-user",
			})
			So(err, ShouldBeNil)

			// Verify nic change & dhcp change
			dhcp, err = configuration.GetDHCPConfig(ctx, "machinelse-update-host-nic-user")
			So(err, ShouldBeNil)
			So(dhcp.GetMacAddress(), ShouldEqual, "eth1-nic-macaddress")
			lse, _ := inventory.GetMachineLSE(ctx, "machinelse-update-host-nic-user")
			So(lse.GetNic(), ShouldEqual, "eth1-nic-user")
			ips, err := configuration.QueryIPByPropertyName(ctx, map[string]string{"ipv4_str": "fake_ip"})
			So(err, ShouldBeNil)
			So(ips, ShouldHaveLength, 1)
			So(ips[0].GetOccupied(), ShouldBeTrue)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/machinelse-update-host-nic-user")
			So(err, ShouldBeNil)
			// nic is changed
			So(changes[0].GetEventLabel(), ShouldEqual, "machine_lse.nic")
			So(changes[0].GetOldValue(), ShouldEqual, "eth0-nic-user")
			So(changes[0].GetNewValue(), ShouldEqual, "eth1-nic-user")
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

		Convey("Partial Update attached device host", func() {
			machine := &ufspb.Machine{
				Name: "adm-1",
				Device: &ufspb.Machine_AttachedDevice{
					AttachedDevice: &ufspb.AttachedDevice{
						DeviceType:   ufspb.AttachedDeviceType_ATTACHED_DEVICE_TYPE_APPLE_PHONE,
						Manufacturer: "test-man",
						BuildTarget:  "test-target",
						Model:        "test-model",
					},
				},
			}
			_, err := registration.CreateMachine(ctx, machine)
			So(err, ShouldBeNil)

			lse := &ufspb.MachineLSE{
				Name:     "adh-lse-1",
				Machines: []string{"adm-1"},
				Lse: &ufspb.MachineLSE_AttachedDeviceLse{
					AttachedDeviceLse: &ufspb.AttachedDeviceLSE{
						OsVersion: &ufspb.OSVersion{
							Value: "test-os",
						},
						AssociatedHostname: "adm-1",
						AssociatedHostPort: "test-port-1",
					},
				},
				Schedulable: false,
			}
			_, err = inventory.CreateMachineLSE(ctx, lse)
			So(err, ShouldBeNil)

			lse1 := &ufspb.MachineLSE{
				Name:     "adh-lse-1",
				Machines: []string{"adm-1"},
				Lse: &ufspb.MachineLSE_AttachedDeviceLse{
					AttachedDeviceLse: &ufspb.AttachedDeviceLSE{
						OsVersion: &ufspb.OSVersion{
							Value: "test-os-2",
						},
						AssociatedHostname: "adm-1",
						AssociatedHostPort: "test-port-2",
					},
				},
				Schedulable: true,
			}
			resp, err := UpdateMachineLSE(ctx, lse1, &field_mask.FieldMask{Paths: []string{
				"osVersion",
				"assocHostname",
				"assocHostPort",
				"schedulable",
			}})
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp.GetMachines(), ShouldResemble, []string{"adm-1"})
			So(resp.GetAttachedDeviceLse().GetOsVersion().GetValue(), ShouldEqual, "test-os-2")
			So(resp.GetAttachedDeviceLse().GetAssociatedHostname(), ShouldEqual, "adm-1")
			So(resp.GetAttachedDeviceLse().GetAssociatedHostPort(), ShouldEqual, "test-port-2")
			So(resp.GetSchedulable(), ShouldBeTrue)
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
			So(peri.Servo.GetServoComponent(), ShouldResemble, []string{"fake-type"})
		})

		Convey("Update a OS machine lse - empty servo topology", func() {
			machineLSE2 := mockDutMachineLSE("machinelse-labmeta-3")
			machineLSE2.GetChromeosMachineLse().GetDeviceLse().GetDut().Peripherals = &chromeosLab.Peripherals{
				Servo: &chromeosLab.Servo{},
			}
			req, err := inventory.CreateMachineLSE(ctx, machineLSE2)
			So(err, ShouldBeNil)
			So(req.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetSmartUsbhub(), ShouldBeFalse)

			topology := &chromeosLab.ServoTopology{}
			err = UpdateLabMeta(ctx, &ufspb.LabMeta{
				ChromeosDeviceId: "machine-labmeta1",
				Hostname:         "machinelse-labmeta-3",
				SmartUsbhub:      true,
				ServoType:        "fake-type",
				ServoTopology:    topology,
			})
			So(err, ShouldBeNil)
			req, err = inventory.GetMachineLSE(ctx, "machinelse-labmeta-3")
			So(err, ShouldBeNil)
			peri := req.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals()
			So(peri.GetSmartUsbhub(), ShouldBeTrue)
			So(peri.Servo.GetServoType(), ShouldEqual, "fake-type")
			So(peri.Servo.GetServoTopology(), ShouldResembleProto, topology)
			So(peri.Servo.GetServoComponent(), ShouldResemble, []string{"fake-type"})
		})

		Convey("Update a OS machine lse - with two servo componments", func() {
			machineLSE := mockDutMachineLSE("machinelse-labmeta-4")
			machineLSE.GetChromeosMachineLse().GetDeviceLse().GetDut().Peripherals = &chromeosLab.Peripherals{
				Servo: &chromeosLab.Servo{},
			}
			req, err := inventory.CreateMachineLSE(ctx, machineLSE)
			So(err, ShouldBeNil)
			So(req.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetSmartUsbhub(), ShouldBeFalse)

			topology := &chromeosLab.ServoTopology{}
			err = UpdateLabMeta(ctx, &ufspb.LabMeta{
				ChromeosDeviceId: "machine-labmeta1",
				Hostname:         "machinelse-labmeta-4",
				SmartUsbhub:      true,
				ServoType:        "fake-type_with_foo",
				ServoTopology:    topology,
			})
			So(err, ShouldBeNil)
			req, err = inventory.GetMachineLSE(ctx, "machinelse-labmeta-4")
			So(err, ShouldBeNil)
			peri := req.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals()
			So(peri.GetSmartUsbhub(), ShouldBeTrue)
			So(peri.Servo.GetServoType(), ShouldEqual, "fake-type_with_foo")
			So(peri.Servo.GetServoTopology(), ShouldResembleProto, topology)
			So(peri.Servo.GetServoComponent(), ShouldResemble, []string{"fake-type", "foo"})
		})

		Convey("Update a OS machine lse - with three servo componments", func() {
			machineLSE := mockDutMachineLSE("machinelse-labmeta-5")
			machineLSE.GetChromeosMachineLse().GetDeviceLse().GetDut().Peripherals = &chromeosLab.Peripherals{
				Servo: &chromeosLab.Servo{},
			}
			req, err := inventory.CreateMachineLSE(ctx, machineLSE)
			So(err, ShouldBeNil)
			So(req.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetSmartUsbhub(), ShouldBeFalse)

			topology := &chromeosLab.ServoTopology{}
			err = UpdateLabMeta(ctx, &ufspb.LabMeta{
				ChromeosDeviceId: "machine-labmeta1",
				Hostname:         "machinelse-labmeta-5",
				SmartUsbhub:      true,
				ServoType:        "fake-type_with_foo_and_bar",
				ServoTopology:    topology,
			})
			So(err, ShouldBeNil)
			req, err = inventory.GetMachineLSE(ctx, "machinelse-labmeta-5")
			So(err, ShouldBeNil)
			peri := req.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals()
			So(peri.GetSmartUsbhub(), ShouldBeTrue)
			So(peri.Servo.GetServoType(), ShouldEqual, "fake-type_with_foo_and_bar")
			So(peri.Servo.GetServoTopology(), ShouldResembleProto, topology)
			So(peri.Servo.GetServoComponent(), ShouldResemble, []string{"fake-type", "foo", "bar"})
		})

		Convey("Update a OS machine lse - with no servo_type", func() {
			machineLSE := mockDutMachineLSE("machinelse-labmeta-6")
			machineLSE.GetChromeosMachineLse().GetDeviceLse().GetDut().Peripherals = &chromeosLab.Peripherals{
				Servo: &chromeosLab.Servo{},
			}
			req, err := inventory.CreateMachineLSE(ctx, machineLSE)
			So(err, ShouldBeNil)
			So(req.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetSmartUsbhub(), ShouldBeFalse)

			topology := &chromeosLab.ServoTopology{}
			err = UpdateLabMeta(ctx, &ufspb.LabMeta{
				ChromeosDeviceId: "machine-labmeta1",
				Hostname:         "machinelse-labmeta-6",
				SmartUsbhub:      true,
				ServoTopology:    topology,
			})
			So(err, ShouldBeNil)
			req, err = inventory.GetMachineLSE(ctx, "machinelse-labmeta-6")
			So(err, ShouldBeNil)
			peri := req.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals()
			So(peri.GetSmartUsbhub(), ShouldBeTrue)
			So(peri.Servo.GetServoType(), ShouldEqual, "")
			So(peri.Servo.GetServoTopology(), ShouldResembleProto, topology)
			So(len(peri.Servo.GetServoComponent()), ShouldEqual, 0)
		})
	})
}

func TestUpdateRecoveryLabData(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("UpdateRecoveryLabData for an OS machine lse", t, func() {
		Convey("Update a non-OS machine lse", func() {
			machineLSE1 := &ufspb.MachineLSE{
				Name:     "machinelse-labdata-1",
				Hostname: "machinelse-labdata-1",
				Machines: []string{"machine-labdata1"},
				Lse: &ufspb.MachineLSE_ChromeBrowserMachineLse{
					ChromeBrowserMachineLse: &ufspb.ChromeBrowserMachineLSE{},
				},
			}
			_, err := registration.CreateMachine(ctx, &ufspb.Machine{
				Name: "machine-labdata1",
			})
			So(err, ShouldBeNil)
			_, err = inventory.CreateMachineLSE(ctx, machineLSE1)
			So(err, ShouldBeNil)

			err = updateRecoveryLabData(ctx, "machinelse-labdata-1", ufspb.State_STATE_UNSPECIFIED,
				&ufsAPI.UpdateDeviceRecoveryDataRequest_LabData{
					SmartUsbhub: true,
				})
			// Update is skipped without error
			So(err, ShouldBeNil)
		})
		Convey("Update a OS machine lse - happy path", func() {
			machineLSE1 := mockDutMachineLSE("machinelse-labdata-2")
			machineLSE1.GetChromeosMachineLse().GetDeviceLse().GetDut().Peripherals = &chromeosLab.Peripherals{
				Servo: &chromeosLab.Servo{},
				Wifi: &chromeosLab.Wifi{
					WifiRouters: []*chromeosLab.WifiRouter{
						{
							Hostname: "machine-labdata-2-pcap",
							State:    chromeosLab.PeripheralState_BROKEN,
						},
						{
							Hostname: "machine-labdata-2-router",
							State:    chromeosLab.PeripheralState_BROKEN,
						},
					},
				},
			}
			req, err := inventory.CreateMachineLSE(ctx, machineLSE1)
			So(err, ShouldBeNil)
			So(req.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetSmartUsbhub(), ShouldBeFalse)
			So(req.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetWifi().GetWifiRouters()[0].GetState(), ShouldEqual, chromeosLab.PeripheralState_BROKEN)
			So(req.GetResourceState(), ShouldEqual, ufspb.State_STATE_UNSPECIFIED)
			topology := &chromeosLab.ServoTopology{
				Main: &chromeosLab.ServoTopologyItem{
					Type: "v3",
				},
			}
			err = updateRecoveryLabData(ctx, "machinelse-labdata-2", ufspb.State_STATE_REPAIR_FAILED, &ufsAPI.UpdateDeviceRecoveryDataRequest_LabData{
				SmartUsbhub:   true,
				ServoType:     "fake-type",
				ServoTopology: topology,
				WifiRouters: []*ufsAPI.UpdateDeviceRecoveryDataRequest_WifiRouter{
					{
						Hostname: "machine-labdata-2-router",
						State:    chromeosLab.PeripheralState_WORKING,
					},
					{
						Hostname: "machine-labdata-2-pcap",
						State:    chromeosLab.PeripheralState_WORKING,
					},
				},
			})
			So(err, ShouldBeNil)
			req, err = inventory.GetMachineLSE(ctx, "machinelse-labdata-2")
			So(err, ShouldBeNil)
			So(req.GetResourceState(), ShouldEqual, ufspb.State_STATE_REPAIR_FAILED)
			peri := req.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals()
			So(peri.GetSmartUsbhub(), ShouldBeTrue)
			So(peri.Servo.GetServoType(), ShouldEqual, "fake-type")
			So(peri.Servo.GetServoTopology(), ShouldResembleProto, topology)
			So(peri.Servo.GetServoComponent(), ShouldResemble, []string{"fake-type"})
			So(peri.GetWifi().GetWifiRouters()[0].GetHostname(), ShouldEqual, "machine-labdata-2-pcap")
			So(peri.GetWifi().GetWifiRouters()[0].GetState(), ShouldEqual, chromeosLab.PeripheralState_WORKING)
			So(peri.GetWifi().GetWifiRouters()[1].GetHostname(), ShouldEqual, "machine-labdata-2-router")
			So(peri.GetWifi().GetWifiRouters()[1].GetState(), ShouldEqual, chromeosLab.PeripheralState_WORKING)
		})
		Convey("Update a OS machine lse - empty servo topology, add multiple wifi routers", func() {
			machineLSE2 := mockDutMachineLSE("machinelse-labdata-3")
			machineLSE2.GetChromeosMachineLse().GetDeviceLse().GetDut().Peripherals = &chromeosLab.Peripherals{
				Servo: &chromeosLab.Servo{},
				Wifi: &chromeosLab.Wifi{
					WifiRouters: []*chromeosLab.WifiRouter{
						{
							Hostname: "machine-labdata-3-router",
							State:    chromeosLab.PeripheralState_BROKEN,
						},
						{
							Hostname: "machine-labdata-3-pcap",
							State:    chromeosLab.PeripheralState_BROKEN,
						},
					},
				},
			}
			req, err := inventory.CreateMachineLSE(ctx, machineLSE2)
			So(err, ShouldBeNil)
			So(req.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetSmartUsbhub(), ShouldBeFalse)

			topology := &chromeosLab.ServoTopology{}
			err = updateRecoveryLabData(ctx, "machinelse-labdata-3", ufspb.State_STATE_READY, &ufsAPI.UpdateDeviceRecoveryDataRequest_LabData{
				SmartUsbhub:   true,
				ServoType:     "fake-type",
				ServoTopology: topology,
				WifiRouters: []*ufsAPI.UpdateDeviceRecoveryDataRequest_WifiRouter{
					{
						Hostname: "machine-labdata-3-router",
						State:    chromeosLab.PeripheralState_WORKING,
					},
					{
						Hostname: "machine-labdata-3-pcap",
						State:    chromeosLab.PeripheralState_WORKING,
					},
				},
			})
			So(err, ShouldBeNil)
			req, err = inventory.GetMachineLSE(ctx, "machinelse-labdata-3")
			So(err, ShouldBeNil)
			peri := req.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals()
			So(peri.GetSmartUsbhub(), ShouldBeTrue)
			So(peri.Servo.GetServoType(), ShouldEqual, "fake-type")
			So(peri.Servo.GetServoTopology(), ShouldResembleProto, topology)
			So(peri.Servo.GetServoComponent(), ShouldResemble, []string{"fake-type"})
			So(peri.GetWifi().GetWifiRouters()[0].GetHostname(), ShouldEqual, "machine-labdata-3-router")
			So(peri.GetWifi().GetWifiRouters()[0].GetState(), ShouldEqual, chromeosLab.PeripheralState_WORKING)
			So(peri.GetWifi().GetWifiRouters()[1].GetHostname(), ShouldEqual, "machine-labdata-3-pcap")
			So(peri.GetWifi().GetWifiRouters()[1].GetState(), ShouldEqual, chromeosLab.PeripheralState_WORKING)
		})

		Convey("Update a OS machine lse - update resource state", func() {
			machineLSE2 := mockDutMachineLSE("machinelse-labdata-3a")
			machineLSE2.GetChromeosMachineLse().GetDeviceLse().GetDut().Peripherals = &chromeosLab.Peripherals{
				Servo: &chromeosLab.Servo{},
				Wifi: &chromeosLab.Wifi{
					WifiRouters: []*chromeosLab.WifiRouter{
						{
							Hostname: "machine-labdata-3-router",
							State:    chromeosLab.PeripheralState_BROKEN,
						},
						{
							Hostname: "machine-labdata-3-pcap",
							State:    chromeosLab.PeripheralState_BROKEN,
						},
					},
				},
			}
			req, err := inventory.CreateMachineLSE(ctx, machineLSE2)
			So(err, ShouldBeNil)
			So(req.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetSmartUsbhub(), ShouldBeFalse)

			err = updateRecoveryLabData(ctx, "machinelse-labdata-3a", ufspb.State_STATE_READY, &ufsAPI.UpdateDeviceRecoveryDataRequest_LabData{})
			So(err, ShouldBeNil)
			req, err = inventory.GetMachineLSE(ctx, "machinelse-labdata-3a")
			So(err, ShouldBeNil)
			So(req.GetResourceState(), ShouldEqual, ufspb.State_STATE_READY)
		})

		Convey("Update a OS machine lse - with two servo componments", func() {
			machineLSE := mockDutMachineLSE("machinelse-labdata-4")
			machineLSE.GetChromeosMachineLse().GetDeviceLse().GetDut().Peripherals = &chromeosLab.Peripherals{
				Servo: &chromeosLab.Servo{},
			}
			req, err := inventory.CreateMachineLSE(ctx, machineLSE)
			So(err, ShouldBeNil)
			So(req.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetSmartUsbhub(), ShouldBeFalse)

			topology := &chromeosLab.ServoTopology{}
			err = updateRecoveryLabData(ctx, "machinelse-labdata-4", ufspb.State_STATE_UNSPECIFIED, &ufsAPI.UpdateDeviceRecoveryDataRequest_LabData{
				SmartUsbhub:   true,
				ServoType:     "fake-type_with_foo",
				ServoTopology: topology,
			})
			So(err, ShouldBeNil)
			req, err = inventory.GetMachineLSE(ctx, "machinelse-labdata-4")
			So(err, ShouldBeNil)
			peri := req.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals()
			So(peri.GetSmartUsbhub(), ShouldBeTrue)
			So(peri.Servo.GetServoType(), ShouldEqual, "fake-type_with_foo")
			So(peri.Servo.GetServoTopology(), ShouldResembleProto, topology)
			So(peri.Servo.GetServoComponent(), ShouldResemble, []string{"fake-type", "foo"})
		})

		Convey("Update a OS machine lse - with three servo componments", func() {
			machineLSE := mockDutMachineLSE("machinelse-labdata-5")
			machineLSE.GetChromeosMachineLse().GetDeviceLse().GetDut().Peripherals = &chromeosLab.Peripherals{
				Servo: &chromeosLab.Servo{},
			}
			req, err := inventory.CreateMachineLSE(ctx, machineLSE)
			So(err, ShouldBeNil)
			So(req.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetSmartUsbhub(), ShouldBeFalse)

			topology := &chromeosLab.ServoTopology{}
			err = updateRecoveryLabData(ctx, "machinelse-labdata-5", ufspb.State_STATE_UNSPECIFIED, &ufsAPI.UpdateDeviceRecoveryDataRequest_LabData{
				SmartUsbhub:   true,
				ServoType:     "fake-type_with_foo_and_bar",
				ServoTopology: topology,
			})
			So(err, ShouldBeNil)
			req, err = inventory.GetMachineLSE(ctx, "machinelse-labdata-5")
			So(err, ShouldBeNil)
			peri := req.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals()
			So(peri.GetSmartUsbhub(), ShouldBeTrue)
			So(peri.Servo.GetServoType(), ShouldEqual, "fake-type_with_foo_and_bar")
			So(peri.Servo.GetServoTopology(), ShouldResembleProto, topology)
			So(peri.Servo.GetServoComponent(), ShouldResemble, []string{"fake-type", "foo", "bar"})
		})

		Convey("Update a OS machine lse - with no servo_type", func() {
			machineLSE := mockDutMachineLSE("machinelse-labdata-6")
			machineLSE.GetChromeosMachineLse().GetDeviceLse().GetDut().Peripherals = &chromeosLab.Peripherals{
				Servo: &chromeosLab.Servo{},
			}
			req, err := inventory.CreateMachineLSE(ctx, machineLSE)
			So(err, ShouldBeNil)
			So(req.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetSmartUsbhub(), ShouldBeFalse)

			topology := &chromeosLab.ServoTopology{}
			err = updateRecoveryLabData(ctx, "machinelse-labdata-6", ufspb.State_STATE_UNSPECIFIED, &ufsAPI.UpdateDeviceRecoveryDataRequest_LabData{
				SmartUsbhub:   true,
				ServoTopology: topology,
			})
			So(err, ShouldBeNil)
			req, err = inventory.GetMachineLSE(ctx, "machinelse-labdata-6")
			So(err, ShouldBeNil)
			peri := req.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals()
			So(peri.GetSmartUsbhub(), ShouldBeTrue)
			So(peri.Servo.GetServoType(), ShouldEqual, "")
			So(peri.Servo.GetServoTopology(), ShouldResembleProto, topology)
			So(len(peri.Servo.GetServoComponent()), ShouldEqual, 0)
		})

		Convey("Update a OS machine lse - wifi router state update, ResourceState update", func() {
			machineLSE2 := mockDutMachineLSE("machinelse-labdata-7")
			machineLSE2.GetChromeosMachineLse().GetDeviceLse().GetDut().Peripherals = &chromeosLab.Peripherals{
				Servo: &chromeosLab.Servo{},
				Wifi: &chromeosLab.Wifi{
					WifiRouters: []*chromeosLab.WifiRouter{
						{
							Hostname: "machine-labdata-7-pcap",
							State:    chromeosLab.PeripheralState_BROKEN,
						},
						{
							Hostname: "machine-labdata-7-router",
							State:    chromeosLab.PeripheralState_BROKEN,
						},
					},
				},
			}
			req, err := inventory.CreateMachineLSE(ctx, machineLSE2)
			So(err, ShouldBeNil)
			So(req.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetSmartUsbhub(), ShouldBeFalse)
			So(req.GetResourceState(), ShouldEqual, ufspb.State_STATE_UNSPECIFIED)
			topology := &chromeosLab.ServoTopology{}
			err = updateRecoveryLabData(ctx, "machinelse-labdata-7", ufspb.State_STATE_READY, &ufsAPI.UpdateDeviceRecoveryDataRequest_LabData{
				SmartUsbhub:   true,
				ServoType:     "fake-type",
				ServoTopology: topology,
				WifiRouters: []*ufsAPI.UpdateDeviceRecoveryDataRequest_WifiRouter{
					{
						Hostname: "machine-labdata-7-router",
						State:    chromeosLab.PeripheralState_WORKING,
					},
					{
						Hostname: "machine-labdata-7-pcap",
						State:    chromeosLab.PeripheralState_WORKING,
					},
				},
			})
			So(err, ShouldBeNil)
			req, err = inventory.GetMachineLSE(ctx, "machinelse-labdata-7")
			So(err, ShouldBeNil)
			So(req.GetResourceState(), ShouldEqual, ufspb.State_STATE_READY)

			peri := req.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals()
			So(peri.GetSmartUsbhub(), ShouldBeTrue)
			So(peri.Servo.GetServoType(), ShouldEqual, "fake-type")
			So(peri.Servo.GetServoTopology(), ShouldResembleProto, topology)
			So(peri.Servo.GetServoComponent(), ShouldResemble, []string{"fake-type"})
			So(peri.GetWifi().GetWifiRouters()[0].GetHostname(), ShouldEqual, "machine-labdata-7-pcap")
			So(peri.GetWifi().GetWifiRouters()[0].GetState(), ShouldEqual, chromeosLab.PeripheralState_WORKING)
			So(peri.GetWifi().GetWifiRouters()[1].GetHostname(), ShouldEqual, "machine-labdata-7-router")
			So(peri.GetWifi().GetWifiRouters()[1].GetState(), ShouldEqual, chromeosLab.PeripheralState_WORKING)

			err = updateRecoveryLabData(ctx, "machinelse-labdata-7", ufspb.State_STATE_READY, &ufsAPI.UpdateDeviceRecoveryDataRequest_LabData{
				SmartUsbhub:   true,
				ServoType:     "fake-type",
				ServoTopology: topology,
				WifiRouters: []*ufsAPI.UpdateDeviceRecoveryDataRequest_WifiRouter{
					{
						Hostname: "machine-labdata-7-router",
						State:    chromeosLab.PeripheralState_WORKING,
					},
					{
						Hostname: "machine-labdata-99-pcap",
						State:    chromeosLab.PeripheralState_WORKING,
					},
				},
			})
			So(err, ShouldBeNil)
			req, err = inventory.GetMachineLSE(ctx, "machinelse-labdata-7")
			So(err, ShouldBeNil)
			peri = req.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals()
			So(peri.GetSmartUsbhub(), ShouldBeTrue)
			So(peri.Servo.GetServoType(), ShouldEqual, "fake-type")
			So(peri.Servo.GetServoTopology(), ShouldResembleProto, topology)
			So(peri.Servo.GetServoComponent(), ShouldResemble, []string{"fake-type"})
			So(peri.GetWifi().GetWifiRouters()[0].GetHostname(), ShouldEqual, "machine-labdata-7-router")
			So(peri.GetWifi().GetWifiRouters()[0].GetState(), ShouldEqual, chromeosLab.PeripheralState_WORKING)
			So(peri.GetWifi().GetWifiRouters()[1].GetHostname(), ShouldEqual, "machine-labdata-99-pcap")
			So(peri.GetWifi().GetWifiRouters()[1].GetState(), ShouldEqual, chromeosLab.PeripheralState_WORKING)
		})

		Convey("Update a OS machine lse - labstation with no labdata", func() {
			machineLSE := mockLabstationMachineLSE("machinelse-labdata-8")
			req, err := inventory.CreateMachineLSE(ctx, machineLSE)
			So(err, ShouldBeNil)
			So(req.GetChromeosMachineLse().GetDeviceLse().GetDut(), ShouldBeNil)
			err = updateRecoveryLabData(ctx, "machinelse-labdata-8", ufspb.State_STATE_SERVING, &ufsAPI.UpdateDeviceRecoveryDataRequest_LabData{})
			So(err, ShouldBeNil)
			req, err = inventory.GetMachineLSE(ctx, "machinelse-labdata-8")
			So(err, ShouldBeNil)
			So(req.GetResourceState(), ShouldEqual, ufspb.State_STATE_SERVING)
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

			changes, err = history.QueryChangesByPropertyName(ctx, "name", "machineLSEDeployments/machine-delete-1-serial")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 2)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetEventLabel(), ShouldEqual, "machine_lse_deployment")
			So(changes[1].GetOldValue(), ShouldEqual, LifeCycleRetire)
			So(changes[1].GetNewValue(), ShouldEqual, LifeCycleRetire)
			So(changes[1].GetEventLabel(), ShouldEqual, "machine_lse_deployment")
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "machineLSEDeployments/machine-delete-1-serial")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 2)
			So(msgs[0].Delete, ShouldBeFalse)
			So(msgs[1].Delete, ShouldBeTrue)
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
			_, err = inventory.GetMachineLSEDeployment(ctx, "machine-delete-2-serial")
			So(err, ShouldBeNil)

			err = DeleteMachineLSE(ctx, "machinelse-delete-2")
			So(err, ShouldBeNil)
			_, err = inventory.GetMachineLSEDeployment(ctx, "machine-delete-2-serial")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)

			// Verify change events
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "machineLSEDeployments/machine-delete-2-serial")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 2)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetEventLabel(), ShouldEqual, "machine_lse_deployment")
			So(changes[1].GetOldValue(), ShouldEqual, LifeCycleRetire)
			So(changes[1].GetNewValue(), ShouldEqual, LifeCycleRetire)
			So(changes[1].GetEventLabel(), ShouldEqual, "machine_lse_deployment")
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "machineLSEDeployments/machine-delete-2-serial")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 2)
			So(msgs[0].Delete, ShouldBeFalse)
			So(msgs[1].Delete, ShouldBeTrue)
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

		Convey("Delete machineLSE DUT associated with a SchedulingUnit", func() {
			machine := &ufspb.Machine{
				Name: "m-1",
			}
			_, err := registration.CreateMachine(ctx, machine)
			So(err, ShouldBeNil)

			dutMachinelse := mockDutMachineLSE("dut-1")
			dutMachinelse.Machines = []string{"m-1"}
			_, err = inventory.CreateMachineLSE(ctx, dutMachinelse)
			So(err, ShouldBeNil)

			_, err = inventory.CreateSchedulingUnit(ctx, &ufspb.SchedulingUnit{
				Name:        "su-1",
				MachineLSEs: []string{"dut-1"},
			})
			So(err, ShouldBeNil)

			err = DeleteMachineLSE(ctx, "dut-1")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "DUT is associated with SchedulingUnit.")
		})

		Convey("Delete machineLSE DUT with servod on docker", func() {
			machine := &ufspb.Machine{
				Name: "machine-8",
			}
			_, err := registration.CreateMachine(ctx, machine)
			So(err, ShouldBeNil)

			servo := &chromeosLab.Servo{
				ServoHostname:       "LocalServodHost",
				ServoPort:           9996,
				ServoSerial:         "Servo-serial",
				DockerContainerName: "docker-2",
			}

			peripherals := &chromeosLab.Peripherals{
				Servo: servo,
			}
			dutMachinelse := mockDutMachineLSE("DUTMachineLse-98")
			dutMachinelse.Machines = []string{"machine-8"}
			dutMachinelse.GetChromeosMachineLse().GetDeviceLse().GetDut().Peripherals = peripherals
			_, err = inventory.CreateMachineLSE(ctx, dutMachinelse)
			So(err, ShouldBeNil)
			// Delete should not throw any error wven though the labstation doesn't exist
			err = DeleteMachineLSE(ctx, "DUTMachineLse-98")
			So(err, ShouldBeNil)
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

func TestRenameMachineLSE(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("RenameMachineLSE", t, func() {
		Convey("Rename non-existent machineLSE", func() {
			_, err := RenameMachineLSE(ctx, "existingMLSE", "non-existentMLSE")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Rename to existing machineLSE", func() {
			_, err := registration.CreateMachine(ctx, &ufspb.Machine{
				Name:  "machine-15",
				Realm: util.AtlLabAdminRealm,
			})
			So(err, ShouldBeNil)

			_, err = registration.CreateMachine(ctx, &ufspb.Machine{
				Name:  "machine-16",
				Realm: util.AtlLabAdminRealm,
			})
			So(err, ShouldBeNil)

			_, err = inventory.CreateMachineLSE(ctx, &ufspb.MachineLSE{
				Name:     "machinelse-15",
				Machines: []string{"machine-15"},
				Hostname: "machinelse-15",
				Lse: &ufspb.MachineLSE_ChromeosMachineLse{
					ChromeosMachineLse: &ufspb.ChromeOSMachineLSE{
						ChromeosLse: &ufspb.ChromeOSMachineLSE_DeviceLse{
							DeviceLse: &ufspb.ChromeOSDeviceLSE{
								Device: &ufspb.ChromeOSDeviceLSE_Dut{
									Dut: &chromeosLab.DeviceUnderTest{
										Hostname: "machinelse-15",
									},
								},
							},
						},
					},
				},
			})
			So(err, ShouldBeNil)

			_, err = inventory.CreateMachineLSE(ctx, &ufspb.MachineLSE{
				Name:     "machinelse-16",
				Machines: []string{"machine-16"},
				Hostname: "machinelse-16",
				Lse: &ufspb.MachineLSE_ChromeosMachineLse{
					ChromeosMachineLse: &ufspb.ChromeOSMachineLSE{
						ChromeosLse: &ufspb.ChromeOSMachineLSE_DeviceLse{
							DeviceLse: &ufspb.ChromeOSDeviceLSE{
								Device: &ufspb.ChromeOSDeviceLSE_Dut{
									Dut: &chromeosLab.DeviceUnderTest{
										Hostname: "machinelse-16",
									},
								},
							},
						},
					},
				},
			})
			So(err, ShouldBeNil)
			// Should not be possible to assign machinelse-15 to machinelse-16
			_, err = RenameMachineLSE(ctx, "machinelse-15", "machinelse-16")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, codes.FailedPrecondition.String())
		})
		Convey("Rename non os machineLSE", func() {
			_, err := registration.CreateMachine(ctx, &ufspb.Machine{
				Name:  "machine-21",
				Realm: util.AtlLabAdminRealm,
			})
			So(err, ShouldBeNil)

			_, err = inventory.CreateMachineLSE(ctx, &ufspb.MachineLSE{
				Name:     "machinelse-21",
				Machines: []string{"machine-21"},
				Hostname: "machinelse-21",
				// Not a os machine as there is not ChromeosMachineLSE
			})
			So(err, ShouldBeNil)

			ctx := auth.WithState(ctx, &authtest.FakeState{
				Identity: "user:user@example.com",
				FakeDB: authtest.NewFakeDB(
					authtest.MockMembership("user:user@example.com", "user"),
					authtest.MockPermission("user:user@example.com", util.AtlLabAdminRealm, util.InventoriesDelete),
					authtest.MockPermission("user:user@example.com", util.AtlLabAdminRealm, util.InventoriesCreate),
				),
			})

			_, err = RenameMachineLSE(ctx, "machinelse-21", "machinelse-22")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, codes.Unimplemented.String())
		})
		Convey("Rename machineLSE with out create permission", func() {
			_, err := registration.CreateMachine(ctx, &ufspb.Machine{
				Name:  "machine-17",
				Realm: util.AtlLabAdminRealm,
			})
			So(err, ShouldBeNil)

			_, err = inventory.CreateMachineLSE(ctx, &ufspb.MachineLSE{
				Name:     "machinelse-17",
				Machines: []string{"machine-17"},
				Hostname: "machinelse-17",
				Lse: &ufspb.MachineLSE_ChromeosMachineLse{
					ChromeosMachineLse: &ufspb.ChromeOSMachineLSE{
						ChromeosLse: &ufspb.ChromeOSMachineLSE_DeviceLse{
							DeviceLse: &ufspb.ChromeOSDeviceLSE{
								Device: &ufspb.ChromeOSDeviceLSE_Dut{
									Dut: &chromeosLab.DeviceUnderTest{
										Hostname: "machinelse-17",
									},
								},
							},
						},
					},
				},
			})
			So(err, ShouldBeNil)

			ctx := auth.WithState(ctx, &authtest.FakeState{
				Identity: "user:user@example.com",
				FakeDB: authtest.NewFakeDB(
					authtest.MockMembership("user:user@example.com", "user"),
					authtest.MockPermission("user:user@example.com", util.AtlLabAdminRealm, util.InventoriesUpdate),
					authtest.MockPermission("user:user@example.com", util.AtlLabAdminRealm, util.InventoriesDelete),
					// Missing create permission
				),
			})

			_, err = RenameMachineLSE(ctx, "machinelse-17", "machinelse-18")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, codes.PermissionDenied.String())
		})
		Convey("Rename machineLSE with out delete permission", func() {
			_, err := registration.CreateMachine(ctx, &ufspb.Machine{
				Name:  "machine-18",
				Realm: util.AtlLabAdminRealm,
			})
			So(err, ShouldBeNil)

			_, err = inventory.CreateMachineLSE(ctx, &ufspb.MachineLSE{
				Name:     "machinelse-18",
				Machines: []string{"machine-18"},
				Hostname: "machinelse-18",
				Lse: &ufspb.MachineLSE_ChromeosMachineLse{
					ChromeosMachineLse: &ufspb.ChromeOSMachineLSE{
						ChromeosLse: &ufspb.ChromeOSMachineLSE_DeviceLse{
							DeviceLse: &ufspb.ChromeOSDeviceLSE{
								Device: &ufspb.ChromeOSDeviceLSE_Dut{
									Dut: &chromeosLab.DeviceUnderTest{
										Hostname: "machinelse-18",
									},
								},
							},
						},
					},
				},
			})
			So(err, ShouldBeNil)

			ctx := auth.WithState(ctx, &authtest.FakeState{
				Identity: "user:user@example.com",
				FakeDB: authtest.NewFakeDB(
					authtest.MockMembership("user:user@example.com", "user"),
					authtest.MockPermission("user:user@example.com", util.AtlLabAdminRealm, util.InventoriesUpdate),
					authtest.MockPermission("user:user@example.com", util.AtlLabAdminRealm, util.InventoriesCreate),
				),
			})

			_, err = RenameMachineLSE(ctx, "machinelse-18", "machinelse-19")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, codes.PermissionDenied.String())
		})
		Convey("Rename machineLSE happy path", func() {
			_, err := registration.CreateMachine(ctx, &ufspb.Machine{
				Name:  "machine-19",
				Realm: util.AtlLabAdminRealm,
			})
			So(err, ShouldBeNil)

			_, err = inventory.CreateMachineLSE(ctx, &ufspb.MachineLSE{
				Name:     "machinelse-19",
				Machines: []string{"machine-19"},
				Hostname: "machinelse-19",
				Lse: &ufspb.MachineLSE_ChromeosMachineLse{
					ChromeosMachineLse: &ufspb.ChromeOSMachineLSE{
						ChromeosLse: &ufspb.ChromeOSMachineLSE_DeviceLse{
							DeviceLse: &ufspb.ChromeOSDeviceLSE{
								Device: &ufspb.ChromeOSDeviceLSE_Dut{
									Dut: &chromeosLab.DeviceUnderTest{
										Hostname: "machinelse-19",
									},
								},
							},
						},
					},
				},
			})
			So(err, ShouldBeNil)

			ctx := auth.WithState(ctx, &authtest.FakeState{
				Identity: "user:user@example.com",
				FakeDB: authtest.NewFakeDB(
					authtest.MockMembership("user:user@example.com", "user"),
					authtest.MockPermission("user:user@example.com", util.AtlLabAdminRealm, util.InventoriesUpdate),
					authtest.MockPermission("user:user@example.com", util.AtlLabAdminRealm, util.InventoriesCreate),
					authtest.MockPermission("user:user@example.com", util.AtlLabAdminRealm, util.InventoriesDelete),
				),
			})

			_, err = RenameMachineLSE(ctx, "machinelse-19", "machinelse-20")
			So(err, ShouldBeNil)
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/machinelse-19")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/machinelse-20")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/machinelse-19")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 2)
			// Verify all changes recorded by the history.
			So(changes[1].OldValue, ShouldEqual, "machinelse-19")
			So(changes[1].NewValue, ShouldEqual, "machinelse-20")
			So(changes[0].OldValue, ShouldEqual, "RENAME")
			So(changes[0].NewValue, ShouldEqual, "RENAME")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "hosts/machinelse-20")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 2)
			// Verify all changes recorded by the history.
			So(changes[1].OldValue, ShouldEqual, "machinelse-19")
			So(changes[1].NewValue, ShouldEqual, "machinelse-20")
			So(changes[0].OldValue, ShouldEqual, "RENAME")
			So(changes[0].NewValue, ShouldEqual, "RENAME")
		})
	})
}

func TestGetAttachedDeviceData(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	ctx = external.WithTestingContext(ctx)
	ctx = useTestingCfg(ctx)

	machine := &ufspb.Machine{
		Name: "machine-1",
		Device: &ufspb.Machine_AttachedDevice{
			AttachedDevice: &ufspb.AttachedDevice{
				Manufacturer: "Apple",
				DeviceType:   ufspb.AttachedDeviceType_ATTACHED_DEVICE_TYPE_APPLE_PHONE,
				BuildTarget:  "test",
				Model:        "test",
			},
		},
	}
	registration.CreateMachine(ctx, machine)

	// var dummylse *ufspb.MachineLSE
	admlse := mockAttachedDeviceMachineLSE("lse-1")
	admlse.Machines = []string{"machine-1"}
	inventory.CreateMachineLSE(ctx, admlse)

	dutState := mockDutState("machine-1", "lse-1")
	UpdateDutState(ctx, dutState)

	Convey("TestGetAttachedDeviceData", t, func() {
		Convey("GetAttachedDeviceData - happy path", func() {
			resp, err := GetAttachedDeviceData(ctx, admlse)
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp.GetLabConfig(), ShouldResembleProto, admlse)
			So(resp.GetMachine(), ShouldResembleProto, machine)
			So(resp.GetDutState(), ShouldResembleProto, dutState)
		})

		Convey("GetAttachedDeviceData - machine not found by hostname", func() {
			admlse2 := mockAttachedDeviceMachineLSE("lse-2")
			admlse2.Machines = []string{"machine-2"}
			inventory.CreateMachineLSE(ctx, admlse2)

			resp, err := GetAttachedDeviceData(ctx, admlse2)
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp.GetLabConfig(), ShouldResembleProto, admlse2)
			So(resp.GetMachine(), ShouldBeNil)
			So(resp.GetDutState(), ShouldBeNil)
		})

		Convey("GetAttachedDeviceData - machine not found by id", func() {
			admlse3 := mockAttachedDeviceMachineLSE("lse-3")
			admlse3.Machines = []string{"machine-3"}
			inventory.CreateMachineLSE(ctx, admlse3)

			resp, err := GetAttachedDeviceData(ctx, admlse3)
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp.GetLabConfig(), ShouldResembleProto, admlse3)
			So(resp.GetMachine(), ShouldBeNil)
			So(resp.GetDutState(), ShouldBeNil)
		})

		Convey("GetAttachedDeviceData - no machine specified", func() {
			admlse4 := mockAttachedDeviceMachineLSE("lse-4")
			admlse4.Machines = []string{}
			inventory.CreateMachineLSE(ctx, admlse4)

			resp, err := GetAttachedDeviceData(ctx, admlse4)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "host does not have machines registered to it")
		})

		Convey("GetAttachedDeviceData - nil machinelse", func() {
			resp, err := GetAttachedDeviceData(ctx, nil)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "host cannot be empty")
		})
	})
}

func TestUpdateBluetoothPeerStates(t *testing.T) {
	// Nil success
	if err := updateBluetoothPeerStates(nil, nil); err != nil {
		t.Errorf("updateBluetoothPeerStates = %v; want success on nil btps", err)
	}

	// Extra BTPs that are not present.
	var (
		p    chromeosLab.Peripherals
		btps = []*ufsAPI.UpdateDeviceRecoveryDataRequest_BluetoothPeer{
			{
				Hostname: "h1",
				State:    chromeosLab.PeripheralState_WORKING,
			},
		}
	)
	if err := updateBluetoothPeerStates(&p, btps); err == nil {
		t.Error("updateBluetoothPeerStates succeeded, want unknown BTP error")
	}

	// Successful update on one BTP
	p.BluetoothPeers = []*chromeosLab.BluetoothPeer{
		{
			Device: &chromeosLab.BluetoothPeer_RaspberryPi{
				RaspberryPi: &chromeosLab.RaspberryPi{
					Hostname: "h1",
					State:    chromeosLab.PeripheralState_BROKEN,
				},
			},
		},
		{
			Device: &chromeosLab.BluetoothPeer_RaspberryPi{
				RaspberryPi: &chromeosLab.RaspberryPi{
					Hostname: "h2",
					State:    chromeosLab.PeripheralState_BROKEN,
				},
			},
		},
	}
	if err := updateBluetoothPeerStates(&p, btps); err != nil {
		t.Errorf("updateBluetoothPeerStates = %v, want success", err)
	}
	wants := map[string]chromeosLab.PeripheralState{
		"h1": chromeosLab.PeripheralState_WORKING,
		"h2": chromeosLab.PeripheralState_BROKEN,
	}
	for _, btp := range p.GetBluetoothPeers() {
		rpi := btp.GetRaspberryPi()
		if want := wants[rpi.GetHostname()]; rpi.GetState() != want {
			t.Errorf("rpi(%s).State = %s; want %s", rpi.GetHostname(), rpi.GetState(), want)
		}
	}
}
