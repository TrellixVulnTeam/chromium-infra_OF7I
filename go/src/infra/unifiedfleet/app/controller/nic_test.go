// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"

	ufspb "infra/unifiedfleet/api/v1/proto"
	. "infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/model/history"
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
				Name: "nic-1",
			}
			resp, err := CreateNic(ctx, nic1, "machine-5")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "There is no Machine with MachineID machine-5 in the system.")

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "nics/nic-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})

		Convey("Create new nic with existing machine with nics", func() {
			machine := &ufspb.Machine{
				Name: "machine-10",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{
						Nics: []string{"nic-5"},
					},
				},
			}
			_, err := registration.CreateMachine(ctx, machine)
			So(err, ShouldBeNil)

			nic := &ufspb.Nic{
				Name: "nic-20",
			}
			resp, err := CreateNic(ctx, nic, "machine-10")
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, nic)

			mresp, err := registration.GetMachine(ctx, "machine-10")
			So(err, ShouldBeNil)
			So(mresp.GetChromeBrowserMachine().GetNics(), ShouldResemble, []string{"nic-5", "nic-20"})

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "nics/nic-20")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetEventLabel(), ShouldEqual, "nic")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "machines/machine-10")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, "[nic-5]")
			So(changes[0].GetNewValue(), ShouldEqual, "[nic-5 nic-20]")
			So(changes[0].GetEventLabel(), ShouldEqual, "machine.chrome_browser_machine.nics")
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
				Name: "nic-25",
			}
			resp, err := CreateNic(ctx, nic, "machine-15")
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, nic)

			mresp, err := registration.GetMachine(ctx, "machine-15")
			So(err, ShouldBeNil)
			So(mresp.GetChromeBrowserMachine().GetNics(), ShouldResemble, []string{"nic-25"})

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "nics/nic-25")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetEventLabel(), ShouldEqual, "nic")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "machines/machine-15")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, "[]")
			So(changes[0].GetNewValue(), ShouldEqual, "[nic-25]")
			So(changes[0].GetEventLabel(), ShouldEqual, "machine.chrome_browser_machine.nics")
		})

		Convey("Create new nic with non existing switch", func() {
			nic1 := &ufspb.Nic{
				Name: "nic-1",
				SwitchInterface: &ufspb.SwitchInterface{
					Switch: "switch-1",
				},
			}
			resp, err := CreateNic(ctx, nic1, "machine-1")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "There is no Switch with SwitchID switch-1")

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "nics/nic-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})

		Convey("Create new nic with existing switch", func() {
			switch2 := &ufspb.Switch{
				Name: "switch-2",
			}
			_, err := registration.CreateSwitch(ctx, switch2)
			So(err, ShouldBeNil)

			nic2 := &ufspb.Nic{
				Name: "nic-2",
				SwitchInterface: &ufspb.SwitchInterface{
					Switch: "switch-2",
				},
			}
			resp, err := CreateNic(ctx, nic2, "machine-1")
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
				Name: "nic-1",
			}
			resp, err := UpdateNic(ctx, nic, "machine-1")
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
				Name: "nic-2",
				SwitchInterface: &ufspb.SwitchInterface{
					Switch: "switch-1",
				},
			}
			resp, err := UpdateNic(ctx, nic2, "machine-1")
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
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{
						Nics: []string{"nic-3"},
					},
				},
			}
			_, err := registration.CreateMachine(ctx, machine3)
			So(err, ShouldBeNil)

			machine4 := &ufspb.Machine{
				Name: "machine-4",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{
						Nics: []string{"nic-4"},
					},
				},
			}
			_, err = registration.CreateMachine(ctx, machine4)
			So(err, ShouldBeNil)

			nic := &ufspb.Nic{
				Name: "nic-3",
			}
			_, err = registration.CreateNic(ctx, nic)
			So(err, ShouldBeNil)

			nic = &ufspb.Nic{
				Name: "nic-3",
			}
			resp, err := UpdateNic(ctx, nic, "machine-4")
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp, ShouldResembleProto, nic)

			mresp, err := registration.GetMachine(ctx, "machine-3")
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(mresp.GetChromeBrowserMachine().GetNics(), ShouldBeNil)

			mresp, err = registration.GetMachine(ctx, "machine-4")
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(mresp.GetChromeBrowserMachine().GetNics(), ShouldResemble, []string{"nic-4", "nic-3"})

			// Verify the changes
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "nics/nic-3")
			So(err, ShouldBeNil)
			// No change for the updated nic
			So(changes, ShouldHaveLength, 0)
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "machines/machine-3")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, "[nic-3]")
			So(changes[0].GetNewValue(), ShouldEqual, "[]")
			So(changes[0].GetEventLabel(), ShouldEqual, "machine.chrome_browser_machine.nics")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "machines/machine-4")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, "[nic-4]")
			So(changes[0].GetNewValue(), ShouldEqual, "[nic-4 nic-3]")
			So(changes[0].GetEventLabel(), ShouldEqual, "machine.chrome_browser_machine.nics")
		})

		Convey("Update nic with same machine", func() {
			machine := &ufspb.Machine{
				Name: "machine-5",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{
						Nics: []string{"nic-5"},
					},
				},
			}
			_, err := registration.CreateMachine(ctx, machine)
			So(err, ShouldBeNil)

			nic := &ufspb.Nic{
				Name: "nic-5",
			}
			_, err = registration.CreateNic(ctx, nic)
			So(err, ShouldBeNil)

			nic = &ufspb.Nic{
				Name:       "nic-5",
				MacAddress: "ab:cd:ef",
			}
			resp, err := UpdateNic(ctx, nic, "machine-5")
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp, ShouldResembleProto, nic)

			// Verify the changes
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "nics/nic-5")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, "")
			So(changes[0].GetNewValue(), ShouldEqual, "ab:cd:ef")
			So(changes[0].GetEventLabel(), ShouldEqual, "nic.mac_address")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "machines/machine-5")
			So(err, ShouldBeNil)
			// No changes in machine.nics
			So(changes, ShouldHaveLength, 0)
		})

		Convey("Update nic with non existing machine", func() {
			nic := &ufspb.Nic{
				Name: "nic-6",
			}
			_, err := registration.CreateNic(ctx, nic)
			So(err, ShouldBeNil)

			nic = &ufspb.Nic{
				Name: "nic-6",
			}
			resp, err := UpdateNic(ctx, nic, "machine-6")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "There is no Machine with MachineID machine-6 in the system.")

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "nics/nic-6")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
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

		Convey("Delete nic successfully by existing ID with machine reference", func() {
			nic := mockNic("nic-1")
			resp, err := registration.CreateNic(ctx, nic)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, nic)

			chromeBrowserMachine1 := &ufspb.Machine{
				Name: "machine-1",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{
						Nics: []string{"nic-1"},
					},
				},
			}
			_, err = registration.CreateMachine(ctx, chromeBrowserMachine1)
			So(err, ShouldBeNil)

			err = DeleteNic(ctx, "nic-1")
			So(err, ShouldBeNil)

			resp, err = registration.GetNic(ctx, "nic-1")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)

			mresp, err := registration.GetMachine(ctx, "machine-1")
			So(mresp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(mresp.GetChromeBrowserMachine().GetNics(), ShouldBeNil)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "nics/nic-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRetire)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRetire)
			So(changes[0].GetEventLabel(), ShouldEqual, "nic")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "machines/machine-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, "[nic-1]")
			So(changes[0].GetNewValue(), ShouldEqual, "[]")
			So(changes[0].GetEventLabel(), ShouldEqual, "machine.chrome_browser_machine.nics")
		})

		Convey("Delete nic successfully by existing ID without references", func() {
			nic := mockNic("nic-2")
			_, err := registration.CreateNic(ctx, nic)
			So(err, ShouldBeNil)

			err = DeleteNic(ctx, "nic-2")
			So(err, ShouldBeNil)

			resp, err := registration.GetNic(ctx, "nic-2")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "nics/nic-2")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRetire)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRetire)
			So(changes[0].GetEventLabel(), ShouldEqual, "nic")
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
