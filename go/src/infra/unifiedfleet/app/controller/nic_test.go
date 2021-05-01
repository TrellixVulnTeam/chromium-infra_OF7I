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
	"infra/unifiedfleet/app/util"
)

func mockNic(id string) *ufspb.Nic {
	return &ufspb.Nic{
		Name: id,
	}
}

func TestCreateNic(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	machine1 := &ufspb.Machine{
		Name: "machine-1",
		Device: &ufspb.Machine_ChromeBrowserMachine{
			ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
		},
	}
	registration.CreateMachine(ctx, machine1)
	Convey("CreateNics", t, func() {
		Convey("Create new nic with non existing machine", func() {
			nic1 := &ufspb.Nic{
				Name:    "nic-1",
				Machine: "machine-5",
			}
			resp, err := CreateNic(ctx, nic1)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "nics/nic-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})

		Convey("Create nic - duplicated switch ports", func() {
			nic := &ufspb.Nic{
				Name: "nic-create-1",
				SwitchInterface: &ufspb.SwitchInterface{
					Switch:   "nic-create-switch-1",
					PortName: "25",
				},
			}
			_, err := registration.CreateNic(ctx, nic)
			So(err, ShouldBeNil)
			switch1 := &ufspb.Switch{
				Name: "nic-create-switch-1",
			}
			_, err = registration.CreateSwitch(ctx, switch1)
			So(err, ShouldBeNil)

			nic2 := &ufspb.Nic{
				Name:    "nic-create-2",
				Machine: "machine-1",
				SwitchInterface: &ufspb.SwitchInterface{
					Switch:   "nic-create-switch-1",
					PortName: "25",
				},
			}
			_, err = CreateNic(ctx, nic2)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "switch port 25 of nic-create-switch-1 is already occupied")
		})

		Convey("Create new nic with existing machine with nics", func() {
			machine := &ufspb.Machine{
				Name: "machine-10",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
				},
			}
			_, err := registration.CreateMachine(ctx, machine)
			So(err, ShouldBeNil)

			nic := &ufspb.Nic{
				Name:    "nic-20",
				Machine: "machine-10",
			}
			resp, err := CreateNic(ctx, nic)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, nic)

			mresp, err := GetMachine(ctx, "machine-10")
			So(err, ShouldBeNil)
			So(mresp.GetChromeBrowserMachine().GetNicObjects()[0].GetName(), ShouldResemble, "nic-20")

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "nics/nic-20")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetEventLabel(), ShouldEqual, "nic")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "machines/machine-10")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "nics/nic-20")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			So(msgs[0].Delete, ShouldBeFalse)
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "machines/machine-10")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 0)
		})

		Convey("Create new nic with existing machine without nics", func() {
			machine := &ufspb.Machine{
				Name: "machine-15",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
				},
				Realm: util.BrowserLabAdminRealm,
			}
			_, err := registration.CreateMachine(ctx, machine)
			So(err, ShouldBeNil)

			nic := &ufspb.Nic{
				Name:    "nic-25",
				Machine: "machine-15",
			}
			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.RegistrationsCreate, util.BrowserLabAdminRealm)
			resp, err := CreateNic(ctx, nic)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, nic)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "nics/nic-25")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetEventLabel(), ShouldEqual, "nic")
		})

		Convey("Create new nic with non existing switch", func() {
			nic1 := &ufspb.Nic{
				Name:    "nic-1",
				Machine: "machine-1",
				SwitchInterface: &ufspb.SwitchInterface{
					Switch: "switch-1",
				},
			}
			resp, err := CreateNic(ctx, nic1)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "There is no Switch with SwitchID switch-1")

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "nics/nic-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "nics/nic-1")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 0)
		})

		Convey("Create new nic with existing switch", func() {
			switch2 := &ufspb.Switch{
				Name: "switch-2",
			}
			_, err := registration.CreateSwitch(ctx, switch2)
			So(err, ShouldBeNil)

			nic2 := &ufspb.Nic{
				Name:    "nic-2",
				Machine: "machine-1",
				SwitchInterface: &ufspb.SwitchInterface{
					Switch: "switch-2",
				},
			}
			resp, err := CreateNic(ctx, nic2)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, nic2)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "nics/nic-2")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetEventLabel(), ShouldEqual, "nic")
		})

		Convey("Create new nic - permission denied: same realm and no create permission", func() {
			machine := &ufspb.Machine{
				Name: "machine-16",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
				},
				Realm: util.BrowserLabAdminRealm,
			}
			_, err := registration.CreateMachine(ctx, machine)
			So(err, ShouldBeNil)

			nic := &ufspb.Nic{
				Name:    "nic-26",
				Machine: "machine-16",
			}
			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.RegistrationsGet, util.BrowserLabAdminRealm)
			_, err = CreateNic(ctx, nic)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, PermissionDenied)
		})

		Convey("Create new nic - permission denied: different realm", func() {
			machine := &ufspb.Machine{
				Name: "machine-17",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
				},
				Realm: util.BrowserLabAdminRealm,
			}
			_, err := registration.CreateMachine(ctx, machine)
			So(err, ShouldBeNil)

			nic := &ufspb.Nic{
				Name:    "nic-27",
				Machine: "machine-17",
			}
			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.RegistrationsCreate, util.AtlLabAdminRealm)
			_, err = CreateNic(ctx, nic)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, PermissionDenied)
		})
	})
}

func TestUpdateNic(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("UpdateNics", t, func() {
		Convey("Update nic with non-existing nic", func() {
			machine1 := &ufspb.Machine{
				Name: "machine-1",
			}
			registration.CreateMachine(ctx, machine1)

			nic := &ufspb.Nic{
				Name:    "nic-1",
				Machine: "machine-1",
			}
			resp, err := UpdateNic(ctx, nic, nil)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "nics/nic-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})

		Convey("Update nic with non existing switch", func() {
			machine1 := &ufspb.Machine{
				Name: "machine-2",
			}
			registration.CreateMachine(ctx, machine1)

			nic := &ufspb.Nic{
				Name:    "nic-2",
				Machine: "machine-2",
			}
			_, err := registration.CreateNic(ctx, nic)
			So(err, ShouldBeNil)

			nic2 := &ufspb.Nic{
				Name:    "nic-2",
				Machine: "machine-2",
				SwitchInterface: &ufspb.SwitchInterface{
					Switch: "switch-1",
				},
			}
			resp, err := UpdateNic(ctx, nic2, nil)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "here is no Switch with SwitchID switch-1")

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "nics/nic-2")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})

		Convey("Update nic with new machine(same realm) - success", func() {
			machine3 := &ufspb.Machine{
				Name: "machine-3",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
				},
				Realm: util.BrowserLabAdminRealm,
			}
			_, err := registration.CreateMachine(ctx, machine3)
			So(err, ShouldBeNil)

			machine4 := &ufspb.Machine{
				Name: "machine-4",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
				},
				Realm: util.BrowserLabAdminRealm,
			}
			_, err = registration.CreateMachine(ctx, machine4)
			So(err, ShouldBeNil)

			nic := &ufspb.Nic{
				Name:    "nic-3",
				Machine: "machine-3",
			}
			_, err = registration.CreateNic(ctx, nic)
			So(err, ShouldBeNil)

			nic = &ufspb.Nic{
				Name:    "nic-3",
				Machine: "machine-4",
			}
			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.RegistrationsUpdate, util.BrowserLabAdminRealm)
			resp, err := UpdateNic(ctx, nic, nil)
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp, ShouldResembleProto, nic)

			// Verify the changes
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "nics/nic-3")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
		})

		Convey("Update nic with non existing machine", func() {
			machine1 := &ufspb.Machine{
				Name: "machine-5",
			}
			registration.CreateMachine(ctx, machine1)

			nic := &ufspb.Nic{
				Name:    "nic-6",
				Machine: "machine-5",
			}
			_, err := registration.CreateNic(ctx, nic)
			So(err, ShouldBeNil)

			nic = &ufspb.Nic{
				Name:    "nic-6",
				Machine: "machine-6",
			}
			resp, err := UpdateNic(ctx, nic, nil)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "There is no Machine with MachineID machine-6 in the system")

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "nics/nic-6")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})

		Convey("Partial Update nic", func() {
			machine1 := &ufspb.Machine{
				Name: "machine-7.1",
			}
			registration.CreateMachine(ctx, machine1)

			nic := &ufspb.Nic{
				Name:    "nic-7",
				Machine: "machine-7.1",
				SwitchInterface: &ufspb.SwitchInterface{
					Switch:   "switch-7",
					PortName: "25",
				},
			}
			_, err := registration.CreateNic(ctx, nic)
			So(err, ShouldBeNil)

			nic1 := &ufspb.Nic{
				Name:       "nic-7",
				MacAddress: "efgh",
				SwitchInterface: &ufspb.SwitchInterface{
					PortName: "75",
				},
			}
			resp, err := UpdateNic(ctx, nic1, &field_mask.FieldMask{Paths: []string{"portName", "macAddress"}})
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp.GetMachine(), ShouldResemble, "machine-7.1")
			So(resp.GetSwitchInterface().GetSwitch(), ShouldResemble, "switch-7")
			So(resp.GetSwitchInterface().GetPortName(), ShouldEqual, "75")
			So(resp.GetMacAddress(), ShouldResemble, "efgh")
		})

		Convey("Partial Update nic mac address and machine(same realm) - succeed", func() {
			machine1 := &ufspb.Machine{
				Name: "machine-8",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
				},
				Realm: util.BrowserLabAdminRealm,
			}
			registration.CreateMachine(ctx, machine1)

			nic := &ufspb.Nic{
				Name:       "nic-8",
				Machine:    "machine-8",
				MacAddress: "abcd",
				SwitchInterface: &ufspb.SwitchInterface{
					Switch:   "switch-8",
					PortName: "25",
				},
			}
			_, err := registration.CreateNic(ctx, nic)
			So(err, ShouldBeNil)

			lse, err := inventory.CreateMachineLSE(ctx, &ufspb.MachineLSE{
				Name:     "lse-partial-update-mac",
				Hostname: "lse-partial-update-mac",
				Machines: []string{"machine-8"},
				Nic:      "nic-8",
			})
			_, err = inventory.CreateMachineLSE(ctx, &ufspb.MachineLSE{
				Name:     "lse-partial-update-mac2",
				Hostname: "lse-partial-update-mac2",
				Machines: []string{"machine-8-8"},
			})
			dhcp := &ufspb.DHCPConfig{
				Hostname:   lse.GetName(),
				Ip:         "fake_ip",
				Vlan:       "fake_vlan",
				MacAddress: nic.GetMacAddress(),
			}
			_, err = configuration.BatchUpdateDHCPs(ctx, []*ufspb.DHCPConfig{dhcp})
			So(err, ShouldBeNil)

			machine1 = &ufspb.Machine{
				Name: "machine-8-8",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
				},
				Realm: util.BrowserLabAdminRealm,
			}
			registration.CreateMachine(ctx, machine1)

			nic1 := &ufspb.Nic{
				Name:       "nic-8",
				Machine:    "machine-8-8",
				MacAddress: "nic-8-address",
			}
			ctx := initializeMockAuthDB(ctx, "user:user@example.com", util.BrowserLabAdminRealm, util.RegistrationsUpdate, util.InventoriesUpdate)
			nic, err = UpdateNic(ctx, nic1, &field_mask.FieldMask{Paths: []string{"macAddress", "machine"}})
			So(err, ShouldBeNil)
			So(nic.GetMacAddress(), ShouldEqual, "nic-8-address")
			// new machine's corresponding host has new dhcp record, new nic
			lse, err = inventory.GetMachineLSE(ctx, "lse-partial-update-mac2")
			So(err, ShouldBeNil)
			So(lse.GetNic(), ShouldEqual, "nic-8")
			dhcp, err = configuration.GetDHCPConfig(ctx, "lse-partial-update-mac2")
			So(err, ShouldBeNil)
			So(dhcp.GetMacAddress(), ShouldEqual, "nic-8-address")
			// old machine's corresponding host has empty dhcp record, empty nic
			dhcp, err = configuration.GetDHCPConfig(ctx, "lse-partial-update-mac")
			So(err, ShouldNotBeNil)
			So(dhcp, ShouldBeNil)
			lse, err = inventory.GetMachineLSE(ctx, "lse-partial-update-mac")
			So(err, ShouldBeNil)
			So(lse.GetNic(), ShouldBeEmpty)

			// verify change events
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/lse-partial-update-mac")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetEventLabel(), ShouldEqual, "machine_lse.nic")
			So(changes[0].GetOldValue(), ShouldEqual, "nic-8")
			So(changes[0].GetNewValue(), ShouldEqual, "")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "hosts/lse-partial-update-mac2")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetEventLabel(), ShouldEqual, "machine_lse.nic")
			So(changes[0].GetOldValue(), ShouldEqual, "")
			So(changes[0].GetNewValue(), ShouldEqual, "nic-8")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "dhcps/lse-partial-update-mac")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetEventLabel(), ShouldEqual, "dhcp_config.ip")
			So(changes[0].GetOldValue(), ShouldEqual, "fake_ip")
			So(changes[0].GetNewValue(), ShouldEqual, "")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "dhcps/lse-partial-update-mac2")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetEventLabel(), ShouldEqual, "dhcp_config.ip")
			So(changes[0].GetOldValue(), ShouldEqual, "")
			So(changes[0].GetNewValue(), ShouldEqual, "fake_ip")
		})

		Convey("Partial Update nic mac address - duplicated mac address", func() {
			machine1 := &ufspb.Machine{
				Name: "machine-8.1",
			}
			registration.CreateMachine(ctx, machine1)

			nic := &ufspb.Nic{
				Name:       "nic-8.1",
				Machine:    "machine-8.1",
				MacAddress: "nic-8.1-address",
				SwitchInterface: &ufspb.SwitchInterface{
					Switch:   "switch-nic-8.1",
					PortName: "25",
				},
			}
			_, err := registration.CreateNic(ctx, nic)
			nic2 := &ufspb.Nic{
				Name:       "nic-8.2",
				MacAddress: "nic-8.2-address",
				SwitchInterface: &ufspb.SwitchInterface{
					Switch:   "switch-nic-8.1",
					PortName: "26",
				},
			}
			_, err = registration.CreateNic(ctx, nic2)
			So(err, ShouldBeNil)

			nic1 := &ufspb.Nic{
				Name:       "nic-8.1",
				MacAddress: "nic-8.2-address",
			}
			_, err = UpdateNic(ctx, nic1, &field_mask.FieldMask{Paths: []string{"macAddress"}})
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "mac_address nic-8.2-address is already occupied")
		})

		Convey("Partial Update nic mac address - no update at all", func() {
			machine1 := &ufspb.Machine{
				Name: "machine-9",
			}
			registration.CreateMachine(ctx, machine1)

			nic := &ufspb.Nic{
				Name:       "nic-9",
				Machine:    "machine-9",
				MacAddress: "nic-9-address",
				SwitchInterface: &ufspb.SwitchInterface{
					Switch:   "nic-switch-9",
					PortName: "25",
				},
			}
			_, err := registration.CreateNic(ctx, nic)
			So(err, ShouldBeNil)

			nic1 := &ufspb.Nic{
				Name:       "nic-9",
				MacAddress: "nic-9-address",
			}
			_, err = UpdateNic(ctx, nic1, &field_mask.FieldMask{Paths: []string{"macAddress"}})
			So(err, ShouldBeNil)
		})

		Convey("Update nic mac address - duplicated mac address", func() {
			machine1 := &ufspb.Machine{
				Name: "machine-10",
			}
			registration.CreateMachine(ctx, machine1)
			nic := &ufspb.Nic{
				Name:       "nic-full-update",
				Machine:    "machine-10",
				MacAddress: "nic-full-update-address",
				SwitchInterface: &ufspb.SwitchInterface{
					Switch:   "switch-nic-full",
					PortName: "25",
				},
			}
			_, err := registration.CreateNic(ctx, nic)

			machine2 := &ufspb.Machine{
				Name: "machine-11",
			}
			registration.CreateMachine(ctx, machine2)
			nic2 := &ufspb.Nic{
				Name:       "nic-full-update2",
				Machine:    "machine-11",
				MacAddress: "nic-full-update-address2",
				SwitchInterface: &ufspb.SwitchInterface{
					Switch:   "switch-nic-full",
					PortName: "26",
				},
			}
			_, err = registration.CreateNic(ctx, nic2)
			So(err, ShouldBeNil)

			nic1 := &ufspb.Nic{
				Name:       "nic-full-update",
				MacAddress: "nic-full-update-address2",
			}
			_, err = UpdateNic(ctx, nic1, nil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "mac_address nic-full-update-address2 is already occupied")
		})

		Convey("Update nic mac address - happy path", func() {
			machine := &ufspb.Machine{
				Name: "machine-7",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
				},
			}
			_, err := registration.CreateMachine(ctx, machine)
			So(err, ShouldBeNil)

			nic := &ufspb.Nic{
				Name:       "nic-10",
				Machine:    "machine-7",
				MacAddress: "nic-10-address-old",
			}
			_, err = registration.CreateNic(ctx, nic)
			So(err, ShouldBeNil)

			lse, err := inventory.CreateMachineLSE(ctx, &ufspb.MachineLSE{
				Name:     "lse-update-mac",
				Hostname: "lse-update-mac",
				Machines: []string{"machine-7"},
				Nic:      "nic-10",
			})
			So(err, ShouldBeNil)
			dhcp := &ufspb.DHCPConfig{
				Hostname:   lse.GetName(),
				Ip:         "fake_ip",
				Vlan:       "fake_vlan",
				MacAddress: nic.GetMacAddress(),
			}
			_, err = configuration.BatchUpdateDHCPs(ctx, []*ufspb.DHCPConfig{dhcp})
			So(err, ShouldBeNil)

			nic1 := &ufspb.Nic{
				Name:       "nic-10",
				Machine:    "machine-7",
				MacAddress: "nic-10-address",
			}
			res, _ := UpdateNic(ctx, nic1, nil)
			So(res, ShouldNotBeNil)
			So(res, ShouldResembleProto, nic1)

			// new machine's corresponding host has new dhcp record
			dhcp, err = configuration.GetDHCPConfig(ctx, "lse-update-mac")
			So(err, ShouldBeNil)
			So(dhcp.GetMacAddress(), ShouldEqual, "nic-10-address")
			// no change event for lse as nic name is not changed
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/lse-update-mac")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
			// verify dhcp change events
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "dhcps/lse-update-mac")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetEventLabel(), ShouldEqual, "dhcp_config.mac_address")
			So(changes[0].GetOldValue(), ShouldEqual, "nic-10-address-old")
			So(changes[0].GetNewValue(), ShouldEqual, "nic-10-address")
		})

		Convey("Update nic - permission denied: same realm and no update permission", func() {
			machine1 := &ufspb.Machine{
				Name:  "machine-12",
				Realm: util.AtlLabAdminRealm,
			}
			registration.CreateMachine(ctx, machine1)

			nic := &ufspb.Nic{
				Name:    "nic-12",
				Machine: "machine-12",
			}
			_, err := registration.CreateNic(ctx, nic)
			So(err, ShouldBeNil)

			nic2 := &ufspb.Nic{
				Name:       "nic-12",
				Machine:    "machine-12",
				MacAddress: "nic-12-address",
			}
			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.RegistrationsGet, util.AtlLabAdminRealm)
			_, err = UpdateNic(ctx, nic2, nil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, PermissionDenied)
		})

		Convey("Update nic - permission denied: different realm", func() {
			machine1 := &ufspb.Machine{
				Name:  "machine-13",
				Realm: util.AtlLabAdminRealm,
			}
			registration.CreateMachine(ctx, machine1)

			nic := &ufspb.Nic{
				Name:    "nic-13",
				Machine: "machine-13",
			}
			_, err := registration.CreateNic(ctx, nic)
			So(err, ShouldBeNil)

			nic2 := &ufspb.Nic{
				Name:       "nic-13",
				Machine:    "machine-13",
				MacAddress: "nic-13-address",
			}
			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.RegistrationsUpdate, util.BrowserLabAdminRealm)
			_, err = UpdateNic(ctx, nic2, nil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, PermissionDenied)
		})

		Convey("Update nic with new machine(different realm with no permission) - Permission denied", func() {
			machine3 := &ufspb.Machine{
				Name: "machine-14",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
				},
				Realm: util.BrowserLabAdminRealm,
			}
			_, err := registration.CreateMachine(ctx, machine3)
			So(err, ShouldBeNil)

			machine4 := &ufspb.Machine{
				Name: "machine-15",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
				},
				Realm: util.AtlLabAdminRealm,
			}
			_, err = registration.CreateMachine(ctx, machine4)
			So(err, ShouldBeNil)

			nic := &ufspb.Nic{
				Name:    "nic-14",
				Machine: "machine-14",
			}
			_, err = registration.CreateNic(ctx, nic)
			So(err, ShouldBeNil)

			nic = &ufspb.Nic{
				Name:    "nic-14",
				Machine: "machine-15",
			}
			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.RegistrationsUpdate, util.BrowserLabAdminRealm)
			_, err = UpdateNic(ctx, nic, nil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, PermissionDenied)
		})

		Convey("Partial Update nic with new machine(different realm with no permission) - Permission denied", func() {
			machine1 := &ufspb.Machine{
				Name: "machine-16",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
				},
				Realm: util.BrowserLabAdminRealm,
			}
			registration.CreateMachine(ctx, machine1)

			nic := &ufspb.Nic{
				Name:    "nic-16",
				Machine: "machine-16",
			}
			_, err := registration.CreateNic(ctx, nic)
			So(err, ShouldBeNil)

			machine2 := &ufspb.Machine{
				Name: "machine-17",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
				},
				Realm: util.AtlLabAdminRealm,
			}
			registration.CreateMachine(ctx, machine2)

			nic1 := &ufspb.Nic{
				Name:    "nic-16",
				Machine: "machine-17",
			}
			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.RegistrationsUpdate, util.BrowserLabAdminRealm)
			_, err = UpdateNic(ctx, nic1, &field_mask.FieldMask{Paths: []string{"machine"}})
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, PermissionDenied)
		})

		Convey("Update nic with new machine(different realm with permission) - Pass", func() {
			machine3 := &ufspb.Machine{
				Name: "machine-18",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
				},
				Realm: util.BrowserLabAdminRealm,
			}
			_, err := registration.CreateMachine(ctx, machine3)
			So(err, ShouldBeNil)

			machine4 := &ufspb.Machine{
				Name: "machine-19",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
				},
				Realm: util.AtlLabAdminRealm,
			}
			_, err = registration.CreateMachine(ctx, machine4)
			So(err, ShouldBeNil)

			nic := &ufspb.Nic{
				Name:    "nic-18",
				Machine: "machine-18",
			}
			_, err = registration.CreateNic(ctx, nic)
			So(err, ShouldBeNil)

			nic.Machine = "machine-19"
			ctx := auth.WithState(ctx, &authtest.FakeState{
				Identity: "user:user@example.com",
				FakeDB: authtest.NewFakeDB(
					authtest.MockMembership("user:user@example.com", "user"),
					authtest.MockPermission("user:user@example.com", util.AtlLabAdminRealm, util.RegistrationsUpdate),
					authtest.MockPermission("user:user@example.com", util.BrowserLabAdminRealm, util.RegistrationsUpdate),
				),
			})
			resp, err := UpdateNic(ctx, nic, nil)
			So(resp, ShouldNotBeNil)
			So(resp, ShouldResembleProto, nic)
		})

		Convey("Partial Update nic with new machine(different realm with permission) - Pass", func() {
			machine1 := &ufspb.Machine{
				Name: "machine-20",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
				},
				Realm: util.BrowserLabAdminRealm,
			}
			registration.CreateMachine(ctx, machine1)

			nic := &ufspb.Nic{
				Name:    "nic-20",
				Machine: "machine-20",
			}
			_, err := registration.CreateNic(ctx, nic)
			So(err, ShouldBeNil)

			machine2 := &ufspb.Machine{
				Name: "machine-21",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
				},
				Realm: util.AtlLabAdminRealm,
			}
			registration.CreateMachine(ctx, machine2)

			nic.Machine = "machine-21"
			ctx := auth.WithState(ctx, &authtest.FakeState{
				Identity: "user:user@example.com",
				FakeDB: authtest.NewFakeDB(
					authtest.MockMembership("user:user@example.com", "user"),
					authtest.MockPermission("user:user@example.com", util.AtlLabAdminRealm, util.RegistrationsUpdate),
					authtest.MockPermission("user:user@example.com", util.BrowserLabAdminRealm, util.RegistrationsUpdate),
				),
			})
			resp, err := UpdateNic(ctx, nic, &field_mask.FieldMask{Paths: []string{"machine"}})
			So(resp, ShouldNotBeNil)
			So(resp.GetMachine(), ShouldEqual, "machine-21")
		})
	})
}

func TestDeleteNic(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("DeleteNic", t, func() {
		Convey("Delete nic error by non-existing ID", func() {
			err := DeleteNic(ctx, "nic-10")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "nics/nic-10")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})

		Convey("Delete nic successfully by existing ID", func() {
			nic := mockNic("nic-1")
			machine1 := &ufspb.Machine{
				Name:  "machine-1",
				Realm: util.AtlLabAdminRealm,
			}
			registration.CreateMachine(ctx, machine1)
			nic.Machine = "machine-1"
			resp, err := registration.CreateNic(ctx, nic)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, nic)

			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.RegistrationsDelete, util.AtlLabAdminRealm)
			err = DeleteNic(ctx, "nic-1")
			So(err, ShouldBeNil)

			resp, err = registration.GetNic(ctx, "nic-1")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "nics/nic-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRetire)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRetire)
			So(changes[0].GetEventLabel(), ShouldEqual, "nic")
		})

		Convey("Delete nic error as it's used by a host", func() {
			nic := mockNic("nic-ip")
			nic.Machine = "machine-ip"
			_, err := registration.CreateNic(ctx, nic)
			So(err, ShouldBeNil)

			chromeBrowserMachine1 := &ufspb.Machine{
				Name: "machine-ip",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
				},
			}
			_, err = registration.CreateMachine(ctx, chromeBrowserMachine1)
			So(err, ShouldBeNil)

			_, err = inventory.CreateMachineLSE(ctx, &ufspb.MachineLSE{
				Name:     "lse-ip",
				Hostname: "lse-ip",
				Machines: []string{"machine-ip"},
				Nic:      "nic-ip",
			})

			err = DeleteNic(ctx, "nic-ip")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "nic nic-ip is used by host lse-ip")

			resp, err := registration.GetNic(ctx, "nic-ip")
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, nic)
		})

		Convey("Delete nic - permission denied: same realm and no delete permission", func() {
			nic := mockNic("nic-3")
			machine1 := &ufspb.Machine{
				Name:  "machine-3",
				Realm: util.AtlLabAdminRealm,
			}
			registration.CreateMachine(ctx, machine1)
			nic.Machine = "machine-3"
			resp, err := registration.CreateNic(ctx, nic)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, nic)

			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.RegistrationsGet, util.AtlLabAdminRealm)
			err = DeleteNic(ctx, "nic-3")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, PermissionDenied)
		})

		Convey("Delete nic - permission denied: different realm", func() {
			nic := mockNic("nic-4")
			machine1 := &ufspb.Machine{
				Name:  "machine-4",
				Realm: util.AtlLabAdminRealm,
			}
			registration.CreateMachine(ctx, machine1)
			nic.Machine = "machine-4"
			resp, err := registration.CreateNic(ctx, nic)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, nic)

			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.RegistrationsDelete, util.BrowserLabAdminRealm)
			err = DeleteNic(ctx, "nic-4")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, PermissionDenied)
		})
	})
}

func TestListNics(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	nicsWithSwitch := make([]*ufspb.Nic, 0, 2)
	nics := make([]*ufspb.Nic, 0, 4)
	for i := 0; i < 4; i++ {
		nic := mockNic(fmt.Sprintf("nic-%d", i))
		if i%2 == 0 {
			nic.SwitchInterface = &ufspb.SwitchInterface{Switch: "switch-12"}
		}
		resp, _ := registration.CreateNic(ctx, nic)
		if i%2 == 0 {
			nicsWithSwitch = append(nicsWithSwitch, resp)
		}
		nics = append(nics, resp)
	}
	Convey("ListNics", t, func() {
		Convey("List Nics - filter invalid - error", func() {
			_, _, err := ListNics(ctx, 5, "", "invalid=mx-1", false)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Invalid field name invalid")
		})

		Convey("List Nics - filter switch - happy path", func() {
			resp, _, _ := ListNics(ctx, 5, "", "switch=switch-12", false)
			So(resp, ShouldNotBeNil)
			So(resp, ShouldResembleProto, nicsWithSwitch)
		})

		Convey("ListNics - Full listing - happy path", func() {
			resp, _, _ := ListNics(ctx, 5, "", "", false)
			So(resp, ShouldNotBeNil)
			So(resp, ShouldResembleProto, nics)
		})
	})
}

func TestBatchGetNics(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("BatchGetNics", t, func() {
		Convey("Batch get nics - happy path", func() {
			entities := make([]*ufspb.Nic, 4)
			for i := 0; i < 4; i++ {
				entities[i] = &ufspb.Nic{
					Name: fmt.Sprintf("nic-batchGet-%d", i),
				}
			}
			_, err := registration.BatchUpdateNics(ctx, entities)
			So(err, ShouldBeNil)
			resp, err := registration.BatchGetNics(ctx, []string{"nic-batchGet-0", "nic-batchGet-1", "nic-batchGet-2", "nic-batchGet-3"})
			So(err, ShouldBeNil)
			So(resp, ShouldHaveLength, 4)
			So(resp, ShouldResembleProto, entities)
		})
		Convey("Batch get nics  - missing id", func() {
			resp, err := registration.BatchGetNics(ctx, []string{"nic-batchGet-non-existing"})
			So(err, ShouldNotBeNil)
			So(resp, ShouldBeNil)
			So(err.Error(), ShouldContainSubstring, "nic-batchGet-non-existing")
		})
		Convey("Batch get nics  - empty input", func() {
			resp, err := registration.BatchGetNics(ctx, nil)
			So(err, ShouldBeNil)
			So(resp, ShouldHaveLength, 0)

			input := make([]string, 0)
			resp, err = registration.BatchGetNics(ctx, input)
			So(err, ShouldBeNil)
			So(resp, ShouldHaveLength, 0)
		})
	})
}

func TestRenameNic(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	registration.CreateMachine(ctx, &ufspb.Machine{
		Name: "machine-ren-1",
		Device: &ufspb.Machine_ChromeBrowserMachine{
			ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
		},
		Realm: util.BrowserLabAdminRealm,
	})
	Convey("RenameNic", t, func() {

		Convey("Rename a Nic with new nic name", func() {
			_, err := registration.CreateNic(ctx, &ufspb.Nic{
				Name:    "machine-ren-1:nic-1",
				Machine: "machine-ren-1",
			})
			So(err, ShouldBeNil)
			_, err = inventory.CreateMachineLSE(ctx, &ufspb.MachineLSE{
				Name:     "machinelse-1",
				Machines: []string{"machine-ren-1"},
				Nic:      "machine-ren-1:nic-1",
			})
			So(err, ShouldBeNil)

			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.RegistrationsUpdate, util.BrowserLabAdminRealm)
			res, err := RenameNic(ctx, "machine-ren-1:nic-1", "machine-ren-1:nic-2")
			So(err, ShouldBeNil)
			So(res.Name, ShouldEqual, "machine-ren-1:nic-2")
			machine, err := GetMachine(ctx, "machine-ren-1")
			So(machine, ShouldNotBeNil)
			So(machine.GetChromeBrowserMachine().GetNicObjects()[0].GetName(), ShouldEqual, "machine-ren-1:nic-2")
			_, err = registration.GetNic(ctx, "machine-ren-1:nic-1")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
			nic, err := registration.GetNic(ctx, "machine-ren-1:nic-2")
			So(nic, ShouldNotBeNil)
			So(nic.GetName(), ShouldEqual, "machine-ren-1:nic-2")
			So(nic.GetMachine(), ShouldEqual, "machine-ren-1")
			host, err := inventory.GetMachineLSE(ctx, "machinelse-1")
			So(host, ShouldNotBeNil)
			So(host.GetNic(), ShouldResemble, "machine-ren-1:nic-2")

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "nics/machine-ren-1:nic-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 2)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRename)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRename)
			So(changes[0].GetEventLabel(), ShouldEqual, "nic")
			So(changes[1].GetOldValue(), ShouldEqual, "machine-ren-1:nic-1")
			So(changes[1].GetNewValue(), ShouldEqual, "machine-ren-1:nic-2")
			So(changes[1].GetEventLabel(), ShouldEqual, "nic.name")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "nics/machine-ren-1:nic-2")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 2)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRename)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRename)
			So(changes[0].GetEventLabel(), ShouldEqual, "nic")
			So(changes[1].GetOldValue(), ShouldEqual, "machine-ren-1:nic-1")
			So(changes[1].GetNewValue(), ShouldEqual, "machine-ren-1:nic-2")
			So(changes[1].GetEventLabel(), ShouldEqual, "nic.name")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "hosts/machinelse-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, "machine-ren-1:nic-1")
			So(changes[0].GetNewValue(), ShouldEqual, "machine-ren-1:nic-2")
			So(changes[0].GetEventLabel(), ShouldEqual, "machine_lse.nic")
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "nics/machine-ren-1:nic-1")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			So(msgs[0].Delete, ShouldBeTrue)
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "nics/machine-ren-1:nic-2")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			So(msgs[0].Delete, ShouldBeFalse)
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/machinelse-1")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			So(msgs[0].Delete, ShouldBeFalse)
		})
		Convey("Rename a non-existing Nic", func() {
			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.RegistrationsUpdate, util.BrowserLabAdminRealm)
			_, err := RenameNic(ctx, "machine-ren-1:nic-3", "machine-ren-1:nic-4")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Rename a Nic to an already existing nic name", func() {
			_, err := registration.CreateNic(ctx, &ufspb.Nic{
				Name:    "machine-ren-1:nic-5",
				Machine: "machine-ren-1",
			})
			So(err, ShouldBeNil)

			_, err = registration.CreateNic(ctx, &ufspb.Nic{
				Name:    "machine-ren-1:nic-6",
				Machine: "machine-ren-1",
			})
			So(err, ShouldBeNil)

			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.RegistrationsUpdate, util.BrowserLabAdminRealm)
			_, err = RenameNic(ctx, "machine-ren-1:nic-5", "machine-ren-1:nic-6")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Nic machine-ren-1:nic-6 already exists in the system")
		})
		Convey("Rename a Machine - permission denied: same realm and no update permission", func() {
			_, err := registration.CreateNic(ctx, &ufspb.Nic{
				Name:    "machine-ren-1:nic-7",
				Machine: "machine-ren-1",
			})
			So(err, ShouldBeNil)

			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.RegistrationsGet, util.BrowserLabAdminRealm)
			_, err = RenameNic(ctx, "machine-ren-1:nic-7", "machine-ren-1:nic-8")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, PermissionDenied)
		})
		Convey("Rename a Nic - permission denied: different realm", func() {
			_, err := registration.CreateNic(ctx, &ufspb.Nic{
				Name:    "machine-ren-1:nic-9",
				Machine: "machine-ren-1",
			})
			So(err, ShouldBeNil)

			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.RegistrationsUpdate, util.AtlLabAdminRealm)
			_, err = RenameNic(ctx, "machine-ren-1:nic-9", "machine-ren-1:nic-10")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, PermissionDenied)
		})
	})
}
