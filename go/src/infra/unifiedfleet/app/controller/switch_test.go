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
	. "infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/model/history"
	"infra/unifiedfleet/app/model/registration"
	"infra/unifiedfleet/app/model/state"
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

		Convey("Create new switch with existing rack", func() {
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

			s, err := state.GetStateRecord(ctx, "switches/switch-25")
			So(err, ShouldBeNil)
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_SERVING)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "switches/switch-25")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetEventLabel(), ShouldEqual, "switch")
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
			resp, err := UpdateSwitch(ctx, switch1, "rack-1", nil)
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
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{},
				},
			}
			_, err := registration.CreateRack(ctx, rack3)
			So(err, ShouldBeNil)

			rack4 := &ufspb.Rack{
				Name: "rack-4",
				Rack: &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{},
				},
			}
			_, err = registration.CreateRack(ctx, rack4)
			So(err, ShouldBeNil)

			switch3 := &ufspb.Switch{
				Name: "switch-3",
				Rack: "rack-3",
			}
			_, err = registration.CreateSwitch(ctx, switch3)
			So(err, ShouldBeNil)

			resp, err := UpdateSwitch(ctx, switch3, "rack-4", nil)
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp, ShouldResembleProto, switch3)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "switches/switch-3")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetEventLabel(), ShouldEqual, "switch.rack")
			So(changes[0].GetOldValue(), ShouldEqual, "rack-3")
			So(changes[0].GetNewValue(), ShouldEqual, "rack-4")
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "switches/switch-3")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			So(msgs[0].Delete, ShouldBeFalse)
		})

		Convey("Update switch with same rack", func() {
			rack := &ufspb.Rack{
				Name: "rack-5",
				Rack: &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{},
				},
			}
			_, err := registration.CreateRack(ctx, rack)
			So(err, ShouldBeNil)

			switch1 := &ufspb.Switch{
				Name: "switch-5",
				Rack: "rack-5",
			}
			_, err = registration.CreateSwitch(ctx, switch1)
			So(err, ShouldBeNil)

			resp, err := UpdateSwitch(ctx, switch1, "rack-5", nil)
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp, ShouldResembleProto, switch1)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "switches/switch-5")
			So(err, ShouldBeNil)
			// Nothing is changed for switch-5
			So(changes, ShouldHaveLength, 0)
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "switches/switch-5")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			So(msgs[0].Delete, ShouldBeFalse)
		})

		Convey("Update switch with non existing rack", func() {
			switch1 := &ufspb.Switch{
				Name: "switch-6",
			}
			_, err := registration.CreateSwitch(ctx, switch1)
			So(err, ShouldBeNil)

			resp, err := UpdateSwitch(ctx, switch1, "rack-6", nil)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "There is no Rack with RackID rack-6 in the system.")

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "switches/switch-6")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})

		Convey("Partial Update switch", func() {
			s := &ufspb.Switch{
				Name:         "switch-7",
				CapacityPort: 10,
				Description:  "Hello Switch",
			}
			_, err := registration.CreateSwitch(ctx, s)
			So(err, ShouldBeNil)

			switch1 := &ufspb.Switch{
				Name:         "switch-7",
				CapacityPort: 44,
			}
			resp, err := UpdateSwitch(ctx, switch1, "", &field_mask.FieldMask{Paths: []string{"capacity"}})
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp.GetDescription(), ShouldResemble, "Hello Switch")
			So(resp.GetCapacityPort(), ShouldEqual, 44)
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
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{},
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
			switch2 := mockSwitch("switch-2")
			_, err := registration.CreateSwitch(ctx, switch2)
			So(err, ShouldBeNil)
			_, err = state.BatchUpdateStates(ctx, []*ufspb.StateRecord{
				{
					ResourceName: "switches/switch-2",
					State:        ufspb.State_STATE_SERVING,
				},
			})
			So(err, ShouldBeNil)

			err = DeleteSwitch(ctx, "switch-2")
			So(err, ShouldBeNil)

			resp, err := registration.GetSwitch(ctx, "switch-2")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)

			_, err = state.GetStateRecord(ctx, "switches/switch-2")
			So(err.Error(), ShouldContainSubstring, NotFound)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "switches/switch-2")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRetire)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRetire)
			So(changes[0].GetEventLabel(), ShouldEqual, "switch")
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "switches/switch-2")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			So(msgs[0].Delete, ShouldBeTrue)
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
