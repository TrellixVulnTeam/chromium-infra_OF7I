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

func mockSwitch(id string) *ufspb.Switch {
	return &ufspb.Switch{
		Name: id,
	}
}

func TestCreateSwitch(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	rack1 := &ufspb.Rack{
		Name: "rack-1",
		Rack: &ufspb.Rack_ChromeBrowserRack{
			ChromeBrowserRack: &ufspb.ChromeBrowserRack{},
		},
	}
	registration.CreateRack(ctx, rack1)
	Convey("CreateSwitch", t, func() {
		Convey("Create new switch with already existing switch - error", func() {
			switch1 := &ufspb.Switch{
				Name: "switch-1",
			}
			_, err := registration.CreateSwitch(ctx, switch1)

			resp, err := CreateSwitch(ctx, switch1, "rack-5")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Switch switch-1 already exists in the system")

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "switches/switch-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})

		Convey("Create new switch with non existing rack", func() {
			switch2 := &ufspb.Switch{
				Name: "switch-2",
			}
			resp, err := CreateSwitch(ctx, switch2, "rack-5")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "There is no Rack with RackID rack-5 in the system.")

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "switches/switch-2")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})

		Convey("Create new switch with existing rack with switches", func() {
			rack := &ufspb.Rack{
				Name: "rack-10",
				Rack: &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{
						Switches: []string{"switch-5"},
					},
				},
			}
			_, err := registration.CreateRack(ctx, rack)
			So(err, ShouldBeNil)

			switch1 := &ufspb.Switch{
				Name: "switch-20",
			}
			resp, err := CreateSwitch(ctx, switch1, "rack-10")
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, switch1)

			mresp, err := registration.GetRack(ctx, "rack-10")
			So(err, ShouldBeNil)
			So(mresp.GetChromeBrowserRack().GetSwitches(), ShouldResemble, []string{"switch-5", "switch-20"})

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "switches/switch-20")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetEventLabel(), ShouldEqual, "switch")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "racks/rack-10")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, "[switch-5]")
			So(changes[0].GetNewValue(), ShouldEqual, "[switch-5 switch-20]")
			So(changes[0].GetEventLabel(), ShouldEqual, "rack.chrome_browser_rack.switches")
		})

		Convey("Create new switch with existing rack without switches", func() {
			rack := &ufspb.Rack{
				Name: "rack-15",
				Rack: &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{},
				},
			}
			_, err := registration.CreateRack(ctx, rack)
			So(err, ShouldBeNil)

			switch1 := &ufspb.Switch{
				Name: "switch-25",
			}
			resp, err := CreateSwitch(ctx, switch1, "rack-15")
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, switch1)

			mresp, err := registration.GetRack(ctx, "rack-15")
			So(err, ShouldBeNil)
			So(mresp.GetChromeBrowserRack().GetSwitches(), ShouldResemble, []string{"switch-25"})

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "switches/switch-25")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetEventLabel(), ShouldEqual, "switch")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "racks/rack-15")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, "[]")
			So(changes[0].GetNewValue(), ShouldEqual, "[switch-25]")
			So(changes[0].GetEventLabel(), ShouldEqual, "rack.chrome_browser_rack.switches")
		})
	})
}

func TestUpdateSwitch(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("UpdateSwitch", t, func() {
		Convey("Update switch with non-existing switch", func() {
			rack1 := &ufspb.Rack{
				Name: "rack-1",
			}
			_, err := registration.CreateRack(ctx, rack1)
			So(err, ShouldBeNil)

			switch1 := &ufspb.Switch{
				Name: "switch-1",
			}
			resp, err := UpdateSwitch(ctx, switch1, "rack-1")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "There is no Switch with SwitchID switch-1 in the system")

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "switches/switch-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})

		Convey("Update switch with new rack", func() {
			rack3 := &ufspb.Rack{
				Name: "rack-3",
				Rack: &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{
						Switches: []string{"switch-3"},
					},
				},
			}
			_, err := registration.CreateRack(ctx, rack3)
			So(err, ShouldBeNil)

			rack4 := &ufspb.Rack{
				Name: "rack-4",
				Rack: &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{
						Switches: []string{"switch-4"},
					},
				},
			}
			_, err = registration.CreateRack(ctx, rack4)
			So(err, ShouldBeNil)

			switch3 := &ufspb.Switch{
				Name: "switch-3",
			}
			_, err = registration.CreateSwitch(ctx, switch3)
			So(err, ShouldBeNil)

			resp, err := UpdateSwitch(ctx, switch3, "rack-4")
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp, ShouldResembleProto, switch3)

			mresp, err := registration.GetRack(ctx, "rack-3")
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(mresp.GetChromeBrowserRack().GetSwitches(), ShouldBeNil)

			mresp, err = registration.GetRack(ctx, "rack-4")
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(mresp.GetChromeBrowserRack().GetSwitches(), ShouldResemble, []string{"switch-4", "switch-3"})

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "switches/switch-3")
			So(err, ShouldBeNil)
			// Nothing is changed for switch-3
			So(changes, ShouldHaveLength, 0)
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "racks/rack-4")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, "[switch-4]")
			So(changes[0].GetNewValue(), ShouldEqual, "[switch-4 switch-3]")
			So(changes[0].GetEventLabel(), ShouldEqual, "rack.chrome_browser_rack.switches")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "racks/rack-3")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, "[switch-3]")
			So(changes[0].GetNewValue(), ShouldEqual, "[]")
			So(changes[0].GetEventLabel(), ShouldEqual, "rack.chrome_browser_rack.switches")
		})

		Convey("Update switch with same rack", func() {
			rack := &ufspb.Rack{
				Name: "rack-5",
				Rack: &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{
						Switches: []string{"switch-5"},
					},
				},
			}
			_, err := registration.CreateRack(ctx, rack)
			So(err, ShouldBeNil)

			switch1 := &ufspb.Switch{
				Name: "switch-5",
			}
			_, err = registration.CreateSwitch(ctx, switch1)
			So(err, ShouldBeNil)

			resp, err := UpdateSwitch(ctx, switch1, "rack-5")
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp, ShouldResembleProto, switch1)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "switches/switch-3")
			So(err, ShouldBeNil)
			// Nothing is changed for switch-3
			So(changes, ShouldHaveLength, 0)
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "racks/rack-5")
			So(err, ShouldBeNil)
			// Nothing is changed for rack-5
			So(changes, ShouldHaveLength, 0)
		})

		Convey("Update switch with non existing rack", func() {
			switch1 := &ufspb.Switch{
				Name: "switch-6",
			}
			_, err := registration.CreateSwitch(ctx, switch1)
			So(err, ShouldBeNil)

			resp, err := UpdateSwitch(ctx, switch1, "rack-6")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "There is no Rack with RackID rack-6 in the system.")

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "switches/switch-6")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})

	})
}

func TestDeleteSwitch(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("DeleteSwitch", t, func() {
		Convey("Delete switch by non-existing ID - error", func() {
			err := DeleteSwitch(ctx, "switch-10")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Unable to delete switch switch-10")

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "switches/switch-10")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})

		Convey("Delete switch by existing ID with nic reference", func() {
			rack := &ufspb.Rack{
				Name: "rack-5",
				Rack: &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{
						Switches: []string{"switch-1"},
					},
				},
			}
			_, err := registration.CreateRack(ctx, rack)
			So(err, ShouldBeNil)

			switch1 := mockSwitch("switch-1")
			_, err = registration.CreateSwitch(ctx, switch1)
			So(err, ShouldBeNil)

			nic := &ufspb.Nic{
				Name: "machine1-eth0",
				SwitchInterface: &ufspb.SwitchInterface{
					Switch: "switch-1",
				},
			}
			_, err = registration.CreateNic(ctx, nic)
			So(err, ShouldBeNil)

			err = DeleteSwitch(ctx, "switch-1")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Nics referring to the Switch:")

			resp, err := registration.GetSwitch(ctx, "switch-1")
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, switch1)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "switches/switch-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})

		Convey("Delete switch successfully", func() {
			rack := &ufspb.Rack{
				Name: "rack-6",
				Rack: &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{
						Switches: []string{"switch-2"},
					},
				},
			}
			_, err := registration.CreateRack(ctx, rack)
			So(err, ShouldBeNil)

			switch2 := mockSwitch("switch-2")
			_, err = registration.CreateSwitch(ctx, switch2)
			So(err, ShouldBeNil)

			err = DeleteSwitch(ctx, "switch-2")
			So(err, ShouldBeNil)

			resp, err := registration.GetSwitch(ctx, "switch-2")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)

			rresp, err := registration.GetRack(ctx, "rack-6")
			So(rresp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(rresp.GetChromeBrowserRack().GetSwitches(), ShouldBeNil)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "switches/switch-2")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRetire)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRetire)
			So(changes[0].GetEventLabel(), ShouldEqual, "switch")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "racks/rack-6")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, "[switch-2]")
			So(changes[0].GetNewValue(), ShouldEqual, "[]")
			So(changes[0].GetEventLabel(), ShouldEqual, "rack.chrome_browser_rack.switches")
		})
	})
}

func TestListSwitches(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	switches := make([]*ufspb.Switch, 0, 2)
	for i := 0; i < 4; i++ {
		Switch := mockSwitch(fmt.Sprintf("Switch-%d", i))
		resp, _ := registration.CreateSwitch(ctx, Switch)
		switches = append(switches, resp)
	}
	Convey("ListSwitches", t, func() {
		Convey("List Switches - filter invalid - error", func() {
			_, _, err := ListSwitches(ctx, 5, "", "invalid=mx-1", false)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Invalid field name invalid")
		})

		Convey("ListSwitches - Full listing - happy path", func() {
			resp, _, _ := ListSwitches(ctx, 5, "", "", false)
			So(resp, ShouldNotBeNil)
			So(resp, ShouldResembleProto, switches)
		})
	})
}
