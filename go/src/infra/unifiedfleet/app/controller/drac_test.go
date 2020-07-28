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
				Name: "drac-1",
			}
			resp, err := CreateDrac(ctx, drac1, "machine-5")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "There is no Machine with MachineID machine-5 in the system.")

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "dracs/drac-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})

		Convey("Create new drac with existing machine with drac", func() {
			machine := &ufspb.Machine{
				Name: "machine-10",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{
						Drac: "drac-5",
					},
				},
			}
			_, err := registration.CreateMachine(ctx, machine)
			So(err, ShouldBeNil)

			_, err = registration.CreateDrac(ctx, &ufspb.Drac{
				Name: "drac-5",
			})
			So(err, ShouldBeNil)

			drac := &ufspb.Drac{
				Name: "drac-20",
			}
			resp, err := CreateDrac(ctx, drac, "machine-10")
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, drac)

			mresp, err := registration.GetMachine(ctx, "machine-10")
			So(err, ShouldBeNil)
			So(mresp.GetChromeBrowserMachine().GetDrac(), ShouldResemble, "drac-20")

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "dracs/drac-20")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetEventLabel(), ShouldEqual, "drac")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "dracs/drac-5")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRetire)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRetire)
			So(changes[0].GetEventLabel(), ShouldEqual, "drac")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "machines/machine-10")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, "drac-5")
			So(changes[0].GetNewValue(), ShouldEqual, "drac-20")
			So(changes[0].GetEventLabel(), ShouldEqual, "machine.chrome_browser_machine.drac")
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
				Name: "drac-25",
			}
			resp, err := CreateDrac(ctx, drac, "machine-15")
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, drac)

			mresp, err := registration.GetMachine(ctx, "machine-15")
			So(err, ShouldBeNil)
			So(mresp.GetChromeBrowserMachine().GetDrac(), ShouldResemble, "drac-25")

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "dracs/drac-25")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetEventLabel(), ShouldEqual, "drac")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "machines/machine-15")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, "")
			So(changes[0].GetNewValue(), ShouldEqual, "drac-25")
			So(changes[0].GetEventLabel(), ShouldEqual, "machine.chrome_browser_machine.drac")
		})

		Convey("Create new drac with non existing switch", func() {
			drac1 := &ufspb.Drac{
				Name: "drac-1",
				SwitchInterface: &ufspb.SwitchInterface{
					Switch: "switch-1",
				},
			}
			resp, err := CreateDrac(ctx, drac1, "machine-1")
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
				Name: "drac-2",
				SwitchInterface: &ufspb.SwitchInterface{
					Switch: "switch-2",
				},
			}
			resp, err := CreateDrac(ctx, drac2, "machine-1")
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
				Name: "drac-1",
			}
			resp, err := UpdateDrac(ctx, drac, "machine-1")
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
				Name: "drac-2",
				SwitchInterface: &ufspb.SwitchInterface{
					Switch: "switch-1",
				},
			}
			resp, err := UpdateDrac(ctx, drac2, "machine-1")
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
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{
						Drac: "drac-3",
					},
				},
			}
			_, err := registration.CreateMachine(ctx, machine3)
			So(err, ShouldBeNil)

			machine4 := &ufspb.Machine{
				Name: "machine-4",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{
						Drac: "drac-4",
					},
				},
			}
			_, err = registration.CreateMachine(ctx, machine4)
			So(err, ShouldBeNil)

			drac := &ufspb.Drac{
				Name: "drac-3",
			}
			_, err = registration.CreateDrac(ctx, drac)
			So(err, ShouldBeNil)

			_, err = registration.CreateDrac(ctx, &ufspb.Drac{
				Name: "drac-4",
			})
			So(err, ShouldBeNil)

			resp, err := UpdateDrac(ctx, drac, "machine-4")
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp, ShouldResembleProto, drac)

			mresp, err := registration.GetMachine(ctx, "machine-3")
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(mresp.GetChromeBrowserMachine().GetDrac(), ShouldResemble, "")

			mresp, err = registration.GetMachine(ctx, "machine-4")
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(mresp.GetChromeBrowserMachine().GetDrac(), ShouldResemble, "drac-3")

			_, err = registration.GetDrac(ctx, "drac-4")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Entity not found.")

			// Verify the changes
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "dracs/drac-3")
			So(err, ShouldBeNil)
			// No change for the updated drac
			So(changes, ShouldHaveLength, 0)
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "dracs/drac-4")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRetire)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRetire)
			So(changes[0].GetEventLabel(), ShouldEqual, "drac")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "machines/machine-3")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, "drac-3")
			So(changes[0].GetNewValue(), ShouldEqual, "")
			So(changes[0].GetEventLabel(), ShouldEqual, "machine.chrome_browser_machine.drac")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "machines/machine-4")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, "drac-4")
			So(changes[0].GetNewValue(), ShouldEqual, "drac-3")
			So(changes[0].GetEventLabel(), ShouldEqual, "machine.chrome_browser_machine.drac")
		})

		Convey("Update drac with same machine", func() {
			machine := &ufspb.Machine{
				Name: "machine-5",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{
						Drac: "drac-5",
					},
				},
			}
			_, err := registration.CreateMachine(ctx, machine)
			So(err, ShouldBeNil)

			drac := &ufspb.Drac{
				Name: "drac-5",
			}
			_, err = registration.CreateDrac(ctx, drac)
			So(err, ShouldBeNil)

			drac = &ufspb.Drac{
				Name:       "drac-5",
				MacAddress: "ab:cd:ef",
			}
			resp, err := UpdateDrac(ctx, drac, "machine-5")
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
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "machines/machine-5")
			So(err, ShouldBeNil)
			// No changes in machine.drac
			So(changes, ShouldHaveLength, 0)
		})

		Convey("Update drac with non existing machine", func() {
			drac := &ufspb.Drac{
				Name: "drac-6",
			}
			_, err := registration.CreateDrac(ctx, drac)
			So(err, ShouldBeNil)

			drac = &ufspb.Drac{
				Name: "drac-6",
			}
			resp, err := UpdateDrac(ctx, drac, "machine-6")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "There is no Machine with MachineID machine-6 in the system.")

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "dracs/drac-6")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
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

		Convey("Delete drac successfully by existing ID with machine reference", func() {
			drac := mockDrac("drac-1")
			resp, err := registration.CreateDrac(ctx, drac)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, drac)

			chromeBrowserMachine1 := &ufspb.Machine{
				Name: "machine-1",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{
						Drac: "drac-1",
					},
				},
			}
			_, err = registration.CreateMachine(ctx, chromeBrowserMachine1)
			So(err, ShouldBeNil)

			err = DeleteDrac(ctx, "drac-1")
			So(err, ShouldBeNil)

			resp, err = registration.GetDrac(ctx, "drac-1")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)

			mresp, err := registration.GetMachine(ctx, "machine-1")
			So(mresp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(mresp.GetChromeBrowserMachine().GetDrac(), ShouldResemble, "")

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "dracs/drac-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRetire)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRetire)
			So(changes[0].GetEventLabel(), ShouldEqual, "drac")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "machines/machine-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, "drac-1")
			So(changes[0].GetNewValue(), ShouldEqual, "")
			So(changes[0].GetEventLabel(), ShouldEqual, "machine.chrome_browser_machine.drac")
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
