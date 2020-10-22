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

	ufspb "infra/unifiedfleet/api/v1/models"
	. "infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/model/history"
	"infra/unifiedfleet/app/model/inventory"
	"infra/unifiedfleet/app/model/registration"
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
			So(err.Error(), ShouldContainSubstring, "There is no Machine with MachineID machine-5 in the system.")

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
			}
			_, err := registration.CreateMachine(ctx, machine)
			So(err, ShouldBeNil)

			nic := &ufspb.Nic{
				Name:    "nic-25",
				Machine: "machine-15",
			}
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
			So(err.Error(), ShouldContainSubstring, "There is no Nic with NicID nic-1 in the system")

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "nics/nic-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})

		Convey("Update nic with non existing switch", func() {
			nic := &ufspb.Nic{
				Name: "nic-2",
			}
			_, err := registration.CreateNic(ctx, nic)
			So(err, ShouldBeNil)

			nic2 := &ufspb.Nic{
				Name:    "nic-2",
				Machine: "machine-1",
				SwitchInterface: &ufspb.SwitchInterface{
					Switch: "switch-1",
				},
			}
			resp, err := UpdateNic(ctx, nic2, nil)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "There is no Switch with SwitchID switch-1")

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "nics/nic-2")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})

		Convey("Update nic with new machine", func() {
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
			nic := &ufspb.Nic{
				Name: "nic-6",
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
			So(err.Error(), ShouldContainSubstring, "There is no Machine with MachineID machine-6 in the system.")

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "nics/nic-6")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})

		Convey("Partial Update nic", func() {
			nic := &ufspb.Nic{
				Name: "nic-7",
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
			So(resp.GetSwitchInterface().GetSwitch(), ShouldResemble, "switch-7")
			So(resp.GetSwitchInterface().GetPortName(), ShouldEqual, "75")
			So(resp.GetMacAddress(), ShouldResemble, "efgh")
		})

		Convey("Partial Update nic mac address - succeed", func() {
			nic := &ufspb.Nic{
				Name:       "nic-8",
				MacAddress: "abcd",
				SwitchInterface: &ufspb.SwitchInterface{
					Switch:   "switch-8",
					PortName: "25",
				},
			}
			_, err := registration.CreateNic(ctx, nic)
			So(err, ShouldBeNil)

			nic1 := &ufspb.Nic{
				Name:       "nic-8",
				MacAddress: "nic-8-address",
			}
			nic, err = UpdateNic(ctx, nic1, &field_mask.FieldMask{Paths: []string{"macAddress"}})
			So(err, ShouldBeNil)
			So(nic.GetMacAddress(), ShouldEqual, "nic-8-address")
		})

		Convey("Partial Update nic mac address - duplicated mac address", func() {
			nic := &ufspb.Nic{
				Name:       "nic-8.1",
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
			nic := &ufspb.Nic{
				Name:       "nic-9",
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
			nic := &ufspb.Nic{
				Name:       "nic-full-update",
				MacAddress: "nic-full-update-address",
				SwitchInterface: &ufspb.SwitchInterface{
					Switch:   "switch-nic-full",
					PortName: "25",
				},
			}
			_, err := registration.CreateNic(ctx, nic)
			nic2 := &ufspb.Nic{
				Name:       "nic-full-update2",
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
				Name:    "nic-10",
				Machine: "machine-7",
			}
			_, err = registration.CreateNic(ctx, nic)
			So(err, ShouldBeNil)

			nic1 := &ufspb.Nic{
				Name:       "nic-10",
				Machine:    "machine-7",
				MacAddress: "nic-10-address",
			}
			res, _ := UpdateNic(ctx, nic1, nil)
			So(res, ShouldNotBeNil)
			So(res, ShouldResembleProto, nic1)
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
			So(err.Error(), ShouldContainSubstring, "There is no Nic with NicID nic-10 in the system")

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "nics/nic-10")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})

		Convey("Delete nic successfully by existing ID", func() {
			nic := mockNic("nic-1")
			resp, err := registration.CreateNic(ctx, nic)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, nic)

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
