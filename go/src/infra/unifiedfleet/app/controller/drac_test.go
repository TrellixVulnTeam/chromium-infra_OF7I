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
	"infra/unifiedfleet/app/model/registration"
)

func mockDrac(id string) *ufspb.Drac {
	return &ufspb.Drac{
		Name: id,
	}
}

func TestCreateDrac(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	machine1 := &ufspb.Machine{
		Name: "machine-1",
		Device: &ufspb.Machine_ChromeBrowserMachine{
			ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
		},
	}
	registration.CreateMachine(ctx, machine1)
	Convey("CreateDrac", t, func() {
		Convey("Create new drac with non existing machine", func() {
			drac1 := &ufspb.Drac{
				Name:    "drac-1",
				Machine: "machine-5",
			}
			resp, err := CreateDrac(ctx, drac1)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "There is no Machine with MachineID machine-5 in the system.")

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "dracs/drac-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})

		Convey("Create drac - duplicated switch ports", func() {
			drac := &ufspb.Drac{
				Name: "drac-create-1",
				SwitchInterface: &ufspb.SwitchInterface{
					Switch:   "drac-create-switch-1",
					PortName: "25",
				},
			}
			_, err := registration.CreateDrac(ctx, drac)
			So(err, ShouldBeNil)
			switch1 := &ufspb.Switch{
				Name: "drac-create-switch-1",
			}
			_, err = registration.CreateSwitch(ctx, switch1)
			So(err, ShouldBeNil)

			drac2 := &ufspb.Drac{
				Name:    "drac-create-2",
				Machine: "machine-1",
				SwitchInterface: &ufspb.SwitchInterface{
					Switch:   "drac-create-switch-1",
					PortName: "25",
				},
			}
			_, err = CreateDrac(ctx, drac2)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "switch port 25 of drac-create-switch-1 is already occupied")
		})

		Convey("Create new drac with existing machine with drac", func() {
			machine := &ufspb.Machine{
				Name: "machine-10",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
				},
			}
			_, err := registration.CreateMachine(ctx, machine)
			So(err, ShouldBeNil)

			_, err = registration.CreateDrac(ctx, &ufspb.Drac{
				Name:    "drac-5",
				Machine: "machine-10",
			})
			So(err, ShouldBeNil)

			drac := &ufspb.Drac{
				Name:    "drac-20",
				Machine: "machine-10",
			}
			_, err = CreateDrac(ctx, drac)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "There is already a drac drac-5 associated with machine machine-10")

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "dracs/drac-20")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})

		Convey("Create new drac with existing machine without drac", func() {
			machine := &ufspb.Machine{
				Name: "machine-15",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
				},
			}
			_, err := registration.CreateMachine(ctx, machine)
			So(err, ShouldBeNil)

			drac := &ufspb.Drac{
				Name:    "drac-25",
				Machine: "machine-15",
			}
			resp, err := CreateDrac(ctx, drac)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, drac)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "dracs/drac-25")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetEventLabel(), ShouldEqual, "drac")
		})

		Convey("Create new drac with non existing switch", func() {
			drac1 := &ufspb.Drac{
				Name:    "drac-1",
				Machine: "machine-1",
				SwitchInterface: &ufspb.SwitchInterface{
					Switch: "switch-1",
				},
			}
			resp, err := CreateDrac(ctx, drac1)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "There is no Switch with SwitchID switch-1")

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "dracs/drac-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})

		Convey("Create new drac with existing switch", func() {
			switch2 := &ufspb.Switch{
				Name: "switch-2",
			}
			_, err := registration.CreateSwitch(ctx, switch2)
			So(err, ShouldBeNil)

			drac2 := &ufspb.Drac{
				Name:    "drac-2",
				Machine: "machine-1",
				SwitchInterface: &ufspb.SwitchInterface{
					Switch: "switch-2",
				},
			}
			resp, err := CreateDrac(ctx, drac2)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, drac2)
		})
	})
}

func TestUpdateDrac(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("UpdateDrac", t, func() {
		Convey("Update drac with non-existing drac", func() {
			machine1 := &ufspb.Machine{
				Name: "machine-1",
			}
			registration.CreateMachine(ctx, machine1)
			drac := &ufspb.Drac{
				Name:    "drac-1",
				Machine: "machine-1",
			}
			resp, err := UpdateDrac(ctx, drac, nil)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "There is no Drac with DracID drac-1 in the system.")

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "dracs/drac-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})

		Convey("Update drac with non existing switch", func() {
			drac := &ufspb.Drac{
				Name: "drac-2",
			}
			_, err := registration.CreateDrac(ctx, drac)
			So(err, ShouldBeNil)

			drac2 := &ufspb.Drac{
				Name:    "drac-2",
				Machine: "machine-1",
				SwitchInterface: &ufspb.SwitchInterface{
					Switch: "switch-1",
				},
			}
			resp, err := UpdateDrac(ctx, drac2, nil)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "There is no Switch with SwitchID switch-1")

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "dracs/drac-2")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})

		Convey("Update drac with new machine", func() {
			machine3 := &ufspb.Machine{
				Name: "machine-3",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
				},
			}
			_, err := registration.CreateMachine(ctx, machine3)
			So(err, ShouldBeNil)

			machine4 := &ufspb.Machine{
				Name: "machine-4",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
				},
			}
			_, err = registration.CreateMachine(ctx, machine4)
			So(err, ShouldBeNil)

			drac := &ufspb.Drac{
				Name:    "drac-3",
				Machine: "machine-3",
			}
			_, err = registration.CreateDrac(ctx, drac)
			So(err, ShouldBeNil)

			_, err = registration.CreateDrac(ctx, &ufspb.Drac{
				Name:    "drac-4",
				Machine: "machine-4",
			})
			So(err, ShouldBeNil)

			drac.Machine = "machine-4"
			_, err = UpdateDrac(ctx, drac, nil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "There is already a drac drac-4 associated with machine machine-4")

			// Verify the changes - update fails
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "dracs/drac-3")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})

		Convey("Update drac with same machine", func() {
			machine := &ufspb.Machine{
				Name: "machine-5",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
				},
			}
			_, err := registration.CreateMachine(ctx, machine)
			So(err, ShouldBeNil)

			drac := &ufspb.Drac{
				Name:    "drac-5",
				Machine: "machine-5",
			}
			_, err = registration.CreateDrac(ctx, drac)
			So(err, ShouldBeNil)

			drac = &ufspb.Drac{
				Name:       "drac-5",
				MacAddress: "ab:cd:ef",
				Machine:    "machine-5",
			}
			resp, err := UpdateDrac(ctx, drac, nil)
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp, ShouldResembleProto, drac)

			// Verify the changes
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "dracs/drac-5")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, "")
			So(changes[0].GetNewValue(), ShouldEqual, "ab:cd:ef")
			So(changes[0].GetEventLabel(), ShouldEqual, "drac.mac_address")
		})

		Convey("Update drac with non existing machine", func() {
			drac := &ufspb.Drac{
				Name: "drac-6",
			}
			_, err := registration.CreateDrac(ctx, drac)
			So(err, ShouldBeNil)

			drac = &ufspb.Drac{
				Name:    "drac-6",
				Machine: "machine-6",
			}
			resp, err := UpdateDrac(ctx, drac, nil)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "There is no Machine with MachineID machine-6 in the system.")

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "dracs/drac-6")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})

		Convey("Partial Update drac", func() {
			drac := &ufspb.Drac{
				Name: "drac-7",
				SwitchInterface: &ufspb.SwitchInterface{
					Switch:   "switch-7",
					PortName: "25",
				},
			}
			_, err := registration.CreateDrac(ctx, drac)
			So(err, ShouldBeNil)

			drac1 := &ufspb.Drac{
				Name:       "drac-7",
				MacAddress: "drac-7-macaddress",
				SwitchInterface: &ufspb.SwitchInterface{
					PortName: "75",
				},
			}
			resp, err := UpdateDrac(ctx, drac1, &field_mask.FieldMask{Paths: []string{"portName", "macAddress"}})
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp.GetSwitchInterface().GetSwitch(), ShouldResemble, "switch-7")
			So(resp.GetMacAddress(), ShouldResemble, "drac-7-macaddress")
			So(resp.GetSwitchInterface().GetPortName(), ShouldEqual, "75")
		})

		Convey("Partial Update drac mac address - succeed", func() {
			drac := &ufspb.Drac{
				Name:       "drac-8",
				MacAddress: "drac-8-address",
				SwitchInterface: &ufspb.SwitchInterface{
					Switch:   "switch-8",
					PortName: "25",
				},
			}
			_, err := registration.CreateDrac(ctx, drac)
			So(err, ShouldBeNil)

			drac1 := &ufspb.Drac{
				Name:       "drac-8",
				MacAddress: "drac-8-address",
			}
			drac, err = UpdateDrac(ctx, drac1, &field_mask.FieldMask{Paths: []string{"macAddress"}})
			So(err, ShouldBeNil)
			So(drac.GetMacAddress(), ShouldEqual, "drac-8-address")
		})

		Convey("Partial Update drac mac address - duplicated mac address", func() {
			drac := &ufspb.Drac{
				Name:       "drac-8.1",
				MacAddress: "drac-8.1-address",
				SwitchInterface: &ufspb.SwitchInterface{
					Switch:   "switch-8.1",
					PortName: "25",
				},
			}
			_, err := registration.CreateDrac(ctx, drac)
			drac2 := &ufspb.Drac{
				Name:       "drac-8.2",
				MacAddress: "drac-8.2-address",
				SwitchInterface: &ufspb.SwitchInterface{
					Switch:   "switch-8.1",
					PortName: "26",
				},
			}
			_, err = registration.CreateDrac(ctx, drac2)
			So(err, ShouldBeNil)

			drac1 := &ufspb.Drac{
				Name:       "drac-8.1",
				MacAddress: "drac-8.2-address",
			}
			_, err = UpdateDrac(ctx, drac1, &field_mask.FieldMask{Paths: []string{"macAddress"}})
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "mac_address drac-8.2-address is already occupied")
		})

		Convey("Partial Update drac mac address - no update at all", func() {
			drac := &ufspb.Drac{
				Name:       "drac-9",
				MacAddress: "drac-9-address",
				SwitchInterface: &ufspb.SwitchInterface{
					Switch:   "switch-9",
					PortName: "25",
				},
			}
			_, err := registration.CreateDrac(ctx, drac)
			So(err, ShouldBeNil)

			drac1 := &ufspb.Drac{
				Name:       "drac-9",
				MacAddress: "drac-9-address",
			}
			_, err = UpdateDrac(ctx, drac1, &field_mask.FieldMask{Paths: []string{"macAddress"}})
			So(err, ShouldBeNil)
		})

		Convey("Fully Update drac mac address - duplicated mac address", func() {
			drac := &ufspb.Drac{
				Name:       "drac-full",
				MacAddress: "drac-full-address",
				SwitchInterface: &ufspb.SwitchInterface{
					Switch:   "switch-drac-full",
					PortName: "25",
				},
			}
			_, err := registration.CreateDrac(ctx, drac)
			drac2 := &ufspb.Drac{
				Name:       "drac-full-2",
				MacAddress: "drac-full-address-2",
				SwitchInterface: &ufspb.SwitchInterface{
					Switch:   "switch-drac-full",
					PortName: "26",
				},
			}
			_, err = registration.CreateDrac(ctx, drac2)
			So(err, ShouldBeNil)

			drac1 := &ufspb.Drac{
				Name:       "drac-full",
				MacAddress: "drac-full-address-2",
			}
			_, err = UpdateDrac(ctx, drac1, nil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "mac_address drac-full-address-2 is already occupied")
		})
	})
}

func TestDeleteDrac(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("DeleteDrac", t, func() {
		Convey("Delete drac error by non-existing ID", func() {
			err := DeleteDrac(ctx, "drac-10")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "dracs/drac-10")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})

		Convey("Delete drac successfully by existing ID without references", func() {
			drac := mockDrac("drac-2")
			_, err := registration.CreateDrac(ctx, drac)
			So(err, ShouldBeNil)

			err = DeleteDrac(ctx, "drac-2")
			So(err, ShouldBeNil)

			resp, err := registration.GetDrac(ctx, "drac-2")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "dracs/drac-2")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRetire)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRetire)
			So(changes[0].GetEventLabel(), ShouldEqual, "drac")
		})

		Convey("Delete drac successfully together with deleting ip", func() {
			drac := mockDrac("drac-ip")
			_, err := registration.CreateDrac(ctx, drac)
			So(err, ShouldBeNil)
			_, err = configuration.BatchUpdateDHCPs(ctx, []*ufspb.DHCPConfig{
				{
					Hostname: "drac-ip",
					Ip:       "1.2.3.4",
				},
			})
			So(err, ShouldBeNil)
			_, err = configuration.ImportIPs(ctx, []*ufspb.IP{
				{
					Id:       "vlan-1:123",
					Ipv4Str:  "1.2.3.4",
					Vlan:     "vlan-1",
					Occupied: true,
					Ipv4:     123,
				},
			})
			So(err, ShouldBeNil)

			err = DeleteDrac(ctx, "drac-ip")
			So(err, ShouldBeNil)
			ip, err := configuration.QueryIPByPropertyName(ctx, map[string]string{"ipv4_str": "1.2.3.4"})
			So(err, ShouldBeNil)
			So(ip, ShouldHaveLength, 1)
			So(ip[0].GetOccupied(), ShouldBeFalse)
			_, err = configuration.GetDHCPConfig(ctx, "drac-ip")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)

			resp, err := registration.GetDrac(ctx, "drac-ip")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "dracs/drac-ip")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRetire)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRetire)
			So(changes[0].GetEventLabel(), ShouldEqual, "drac")
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "dracs/drac-ip")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			So(msgs[0].Delete, ShouldBeTrue)
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "dhcps/drac-ip")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			So(msgs[0].Delete, ShouldBeTrue)
		})
	})
}

func TestListDracs(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	dracsWithSwitch := make([]*ufspb.Drac, 0, 2)
	dracs := make([]*ufspb.Drac, 0, 4)
	for i := 0; i < 4; i++ {
		drac := mockDrac(fmt.Sprintf("drac-%d", i))
		if i%2 == 0 {
			drac.SwitchInterface = &ufspb.SwitchInterface{Switch: "switch-12"}
		}
		resp, _ := registration.CreateDrac(ctx, drac)
		if i%2 == 0 {
			dracsWithSwitch = append(dracsWithSwitch, resp)
		}
		dracs = append(dracs, resp)
	}
	Convey("ListDracs", t, func() {
		Convey("List Dracs - filter invalid - error", func() {
			_, _, err := ListDracs(ctx, 5, "", "invalid=mx-1", false)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Invalid field name invalid")
		})

		Convey("List Dracs - filter switch - happy path", func() {
			resp, _, _ := ListDracs(ctx, 5, "", "switch=switch-12", false)
			So(resp, ShouldNotBeNil)
			So(resp, ShouldResembleProto, dracsWithSwitch)
		})

		Convey("ListDracs - Full listing - happy path", func() {
			resp, _, _ := ListDracs(ctx, 5, "", "", false)
			So(resp, ShouldNotBeNil)
			So(resp, ShouldResembleProto, dracs)
		})
	})
}

func TestBatchGetDracs(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("BatchGetDracs", t, func() {
		Convey("Batch get dracs - happy path", func() {
			entities := make([]*ufspb.Drac, 4)
			for i := 0; i < 4; i++ {
				entities[i] = &ufspb.Drac{
					Name: fmt.Sprintf("drac-batchGet-%d", i),
				}
			}
			_, err := registration.BatchUpdateDracs(ctx, entities)
			So(err, ShouldBeNil)
			resp, err := registration.BatchGetDracs(ctx, []string{"drac-batchGet-0", "drac-batchGet-1", "drac-batchGet-2", "drac-batchGet-3"})
			So(err, ShouldBeNil)
			So(resp, ShouldHaveLength, 4)
			So(resp, ShouldResembleProto, entities)
		})
		Convey("Batch get dracs  - missing id", func() {
			resp, err := registration.BatchGetDracs(ctx, []string{"drac-batchGet-non-existing"})
			So(err, ShouldNotBeNil)
			So(resp, ShouldBeNil)
			So(err.Error(), ShouldContainSubstring, "drac-batchGet-non-existing")
		})
		Convey("Batch get dracs  - empty input", func() {
			resp, err := registration.BatchGetDracs(ctx, nil)
			So(err, ShouldBeNil)
			So(resp, ShouldHaveLength, 0)

			input := make([]string, 0)
			resp, err = registration.BatchGetDracs(ctx, input)
			So(err, ShouldBeNil)
			So(resp, ShouldHaveLength, 0)
		})
	})
}
