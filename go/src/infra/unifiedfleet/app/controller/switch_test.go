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
	. "infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/model/history"
	"infra/unifiedfleet/app/model/inventory"
	"infra/unifiedfleet/app/model/registration"
	"infra/unifiedfleet/app/model/state"
	"infra/unifiedfleet/app/util"
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
			rack := &ufspb.Rack{
				Name: "rack-11",
				Rack: &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{},
				},
			}
			_, err := registration.CreateRack(ctx, rack)
			So(err, ShouldBeNil)

			switch1 := &ufspb.Switch{
				Name: "switch-1",
				Rack: "rack-11",
			}
			_, err = registration.CreateSwitch(ctx, switch1)

			resp, err := CreateSwitch(ctx, switch1)
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
				Rack: "rack-5",
			}
			resp, err := CreateSwitch(ctx, switch2)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)

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
				Realm: util.BrowserLabAdminRealm,
			}
			_, err := registration.CreateRack(ctx, rack)
			So(err, ShouldBeNil)

			switch1 := &ufspb.Switch{
				Name: "switch-25",
				Rack: "rack-15",
			}
			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.RegistrationsCreate, util.BrowserLabAdminRealm)
			resp, err := CreateSwitch(ctx, switch1)
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

		Convey("Create new switch - Permission denied: same realm and no create permission", func() {
			rack := &ufspb.Rack{
				Name: "rack-20",
				Rack: &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{},
				},
				Realm: util.BrowserLabAdminRealm,
			}
			_, err := registration.CreateRack(ctx, rack)
			So(err, ShouldBeNil)

			switch1 := &ufspb.Switch{
				Name: "switch-20",
				Rack: "rack-20",
			}
			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.RegistrationsGet, util.BrowserLabAdminRealm)
			_, err = CreateSwitch(ctx, switch1)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, PermissionDenied)
		})

		Convey("Create new switch - Permission denied: different realm", func() {
			rack := &ufspb.Rack{
				Name: "rack-21",
				Rack: &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{},
				},
				Realm: util.BrowserLabAdminRealm,
			}
			_, err := registration.CreateRack(ctx, rack)
			So(err, ShouldBeNil)

			switch1 := &ufspb.Switch{
				Name: "switch-21",
				Rack: "rack-21",
			}
			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.RegistrationsCreate, util.AtlLabAdminRealm)
			_, err = CreateSwitch(ctx, switch1)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, PermissionDenied)
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
				Rack: "rack-1",
			}
			resp, err := UpdateSwitch(ctx, switch1, nil)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "switches/switch-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})

		Convey("Update switch with new rack(same realm) - pass", func() {
			rack3 := &ufspb.Rack{
				Name: "rack-3",
				Rack: &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{},
				},
				Realm: util.BrowserLabAdminRealm,
			}
			_, err := registration.CreateRack(ctx, rack3)
			So(err, ShouldBeNil)

			rack4 := &ufspb.Rack{
				Name: "rack-4",
				Rack: &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{},
				},
				Realm: util.BrowserLabAdminRealm,
			}
			_, err = registration.CreateRack(ctx, rack4)
			So(err, ShouldBeNil)

			switch3 := &ufspb.Switch{
				Name: "switch-3",
				Rack: "rack-3",
			}
			_, err = registration.CreateSwitch(ctx, switch3)
			So(err, ShouldBeNil)

			switch3.Rack = "rack-4"
			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.RegistrationsUpdate, util.BrowserLabAdminRealm)
			resp, err := UpdateSwitch(ctx, switch3, nil)
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

		Convey("Update switch with same rack(same realm) - pass", func() {
			rack := &ufspb.Rack{
				Name: "rack-5",
				Rack: &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{},
				},
				Realm: util.BrowserLabAdminRealm,
			}
			_, err := registration.CreateRack(ctx, rack)
			So(err, ShouldBeNil)

			switch1 := &ufspb.Switch{
				Name: "switch-5",
				Rack: "rack-5",
			}
			_, err = registration.CreateSwitch(ctx, switch1)
			So(err, ShouldBeNil)

			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.RegistrationsUpdate, util.BrowserLabAdminRealm)
			resp, err := UpdateSwitch(ctx, switch1, nil)
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
			rack1 := &ufspb.Rack{
				Name: "rack-6",
			}
			_, err := registration.CreateRack(ctx, rack1)
			So(err, ShouldBeNil)

			switch1 := &ufspb.Switch{
				Name: "switch-6",
				Rack: "rack-6",
			}
			_, err = registration.CreateSwitch(ctx, switch1)
			So(err, ShouldBeNil)

			switch1.Rack = "rack-61"
			resp, err := UpdateSwitch(ctx, switch1, nil)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "There is no Rack with RackID rack-61 in the system")

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "switches/switch-6")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})

		Convey("Partial Update switch", func() {
			rack := &ufspb.Rack{
				Name: "rack-7",
				Rack: &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{},
				},
				Realm: util.BrowserLabAdminRealm,
			}
			_, err := registration.CreateRack(ctx, rack)
			So(err, ShouldBeNil)

			s := &ufspb.Switch{
				Name:         "switch-7",
				Rack:         "rack-7",
				CapacityPort: 10,
				Description:  "Hello Switch",
			}
			_, err = registration.CreateSwitch(ctx, s)
			So(err, ShouldBeNil)

			switch1 := &ufspb.Switch{
				Name:         "switch-7",
				CapacityPort: 44,
			}
			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.RegistrationsUpdate, util.BrowserLabAdminRealm)
			resp, err := UpdateSwitch(ctx, switch1, &field_mask.FieldMask{Paths: []string{"capacity"}})
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp.GetDescription(), ShouldResemble, "Hello Switch")
			So(resp.GetCapacityPort(), ShouldEqual, 44)
		})

		Convey("Update switch - Permission denied: same realm and no update permission", func() {
			rack := &ufspb.Rack{
				Name: "rack-51",
				Rack: &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{},
				},
				Realm: util.BrowserLabAdminRealm,
			}
			_, err := registration.CreateRack(ctx, rack)
			So(err, ShouldBeNil)

			switch1 := &ufspb.Switch{
				Name: "switch-51",
				Rack: "rack-51",
			}
			_, err = registration.CreateSwitch(ctx, switch1)
			So(err, ShouldBeNil)

			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.RegistrationsGet, util.BrowserLabAdminRealm)
			_, err = UpdateSwitch(ctx, switch1, nil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, PermissionDenied)
		})

		Convey("Update switch - Permission denied: different realm", func() {
			rack := &ufspb.Rack{
				Name: "rack-52",
				Rack: &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{},
				},
				Realm: util.BrowserLabAdminRealm,
			}
			_, err := registration.CreateRack(ctx, rack)
			So(err, ShouldBeNil)

			switch1 := &ufspb.Switch{
				Name: "switch-52",
				Rack: "rack-52",
			}
			_, err = registration.CreateSwitch(ctx, switch1)
			So(err, ShouldBeNil)

			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.RegistrationsUpdate, util.AtlLabAdminRealm)
			_, err = UpdateSwitch(ctx, switch1, nil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, PermissionDenied)
		})

		Convey("Update switch with new rack(different realm with no permission)- fail", func() {
			rack3 := &ufspb.Rack{
				Name: "rack-53",
				Rack: &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{},
				},
				Realm: util.BrowserLabAdminRealm,
			}
			_, err := registration.CreateRack(ctx, rack3)
			So(err, ShouldBeNil)

			rack4 := &ufspb.Rack{
				Name: "rack-54",
				Rack: &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{},
				},
				Realm: util.AtlLabAdminRealm,
			}
			_, err = registration.CreateRack(ctx, rack4)
			So(err, ShouldBeNil)

			switch3 := &ufspb.Switch{
				Name: "switch-53",
				Rack: "rack-53",
			}
			_, err = registration.CreateSwitch(ctx, switch3)
			So(err, ShouldBeNil)

			switch3.Rack = "rack-54"
			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.RegistrationsUpdate, util.BrowserLabAdminRealm)
			_, err = UpdateSwitch(ctx, switch3, nil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, PermissionDenied)
		})

		Convey("Update switch with new rack(different realm with permission)- pass", func() {
			rack3 := &ufspb.Rack{
				Name: "rack-55",
				Rack: &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{},
				},
				Realm: util.BrowserLabAdminRealm,
			}
			_, err := registration.CreateRack(ctx, rack3)
			So(err, ShouldBeNil)

			rack4 := &ufspb.Rack{
				Name: "rack-56",
				Rack: &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{},
				},
				Realm: util.AtlLabAdminRealm,
			}
			_, err = registration.CreateRack(ctx, rack4)
			So(err, ShouldBeNil)

			switch3 := &ufspb.Switch{
				Name: "switch-55",
				Rack: "rack-55",
			}
			_, err = registration.CreateSwitch(ctx, switch3)
			So(err, ShouldBeNil)

			switch3.Rack = "rack-56"
			ctx := auth.WithState(ctx, &authtest.FakeState{
				Identity: "user:user@example.com",
				FakeDB: authtest.NewFakeDB(
					authtest.MockMembership("user:user@example.com", "user"),
					authtest.MockPermission("user:user@example.com", util.AtlLabAdminRealm, util.RegistrationsUpdate),
					authtest.MockPermission("user:user@example.com", util.BrowserLabAdminRealm, util.RegistrationsUpdate),
				),
			})
			resp, err := UpdateSwitch(ctx, switch3, nil)
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp, ShouldResembleProto, switch3)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "switches/switch-55")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetEventLabel(), ShouldEqual, "switch.rack")
			So(changes[0].GetOldValue(), ShouldEqual, "rack-55")
			So(changes[0].GetNewValue(), ShouldEqual, "rack-56")
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "switches/switch-55")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			So(msgs[0].Delete, ShouldBeFalse)
		})

		Convey("Partial Update switch with new rack(same realm) - pass", func() {
			rack := &ufspb.Rack{
				Name: "rack-57",
				Rack: &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{},
				},
				Realm: util.BrowserLabAdminRealm,
			}
			_, err := registration.CreateRack(ctx, rack)
			So(err, ShouldBeNil)

			s := &ufspb.Switch{
				Name: "switch-57",
				Rack: "rack-57",
			}
			_, err = registration.CreateSwitch(ctx, s)
			So(err, ShouldBeNil)

			rack = &ufspb.Rack{
				Name: "rack-58",
				Rack: &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{},
				},
				Realm: util.BrowserLabAdminRealm,
			}
			_, err = registration.CreateRack(ctx, rack)
			So(err, ShouldBeNil)

			s.Rack = "rack-58"
			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.RegistrationsUpdate, util.BrowserLabAdminRealm)
			resp, err := UpdateSwitch(ctx, s, &field_mask.FieldMask{Paths: []string{"rack"}})
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp.GetRack(), ShouldResemble, "rack-58")
		})

		Convey("Partial Update switch with new rack(different realm with permission) - pass", func() {
			rack := &ufspb.Rack{
				Name: "rack-59",
				Rack: &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{},
				},
				Realm: util.BrowserLabAdminRealm,
			}
			_, err := registration.CreateRack(ctx, rack)
			So(err, ShouldBeNil)

			s := &ufspb.Switch{
				Name: "switch-59",
				Rack: "rack-59",
			}
			_, err = registration.CreateSwitch(ctx, s)
			So(err, ShouldBeNil)

			rack = &ufspb.Rack{
				Name: "rack-60",
				Rack: &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{},
				},
				Realm: util.AtlLabAdminRealm,
			}
			_, err = registration.CreateRack(ctx, rack)
			So(err, ShouldBeNil)

			s.Rack = "rack-60"
			ctx := auth.WithState(ctx, &authtest.FakeState{
				Identity: "user:user@example.com",
				FakeDB: authtest.NewFakeDB(
					authtest.MockMembership("user:user@example.com", "user"),
					authtest.MockPermission("user:user@example.com", util.AtlLabAdminRealm, util.RegistrationsUpdate),
					authtest.MockPermission("user:user@example.com", util.BrowserLabAdminRealm, util.RegistrationsUpdate),
				),
			})
			resp, err := UpdateSwitch(ctx, s, &field_mask.FieldMask{Paths: []string{"rack"}})
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp.GetRack(), ShouldResemble, "rack-60")
		})

		Convey("Partial Update switch with new rack(different realm without permission) - fail", func() {
			rack := &ufspb.Rack{
				Name: "rack-61",
				Rack: &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{},
				},
				Realm: util.BrowserLabAdminRealm,
			}
			_, err := registration.CreateRack(ctx, rack)
			So(err, ShouldBeNil)

			s := &ufspb.Switch{
				Name: "switch-61",
				Rack: "rack-61",
			}
			_, err = registration.CreateSwitch(ctx, s)
			So(err, ShouldBeNil)

			rack = &ufspb.Rack{
				Name: "rack-62",
				Rack: &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{},
				},
				Realm: util.AtlLabAdminRealm,
			}
			_, err = registration.CreateRack(ctx, rack)
			So(err, ShouldBeNil)

			s.Rack = "rack-62"
			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.RegistrationsUpdate, util.BrowserLabAdminRealm)
			_, err = UpdateSwitch(ctx, s, &field_mask.FieldMask{Paths: []string{"rack"}})
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, PermissionDenied)
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
			So(err.Error(), ShouldContainSubstring, NotFound)

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
			switch1.Rack = "rack-5"
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
				Name: "rack-52",
				Rack: &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{},
				},
				Realm: util.BrowserLabAdminRealm,
			}
			_, err := registration.CreateRack(ctx, rack)
			So(err, ShouldBeNil)

			switch2 := mockSwitch("switch-2")
			switch2.Rack = "rack-52"
			_, err = registration.CreateSwitch(ctx, switch2)
			So(err, ShouldBeNil)
			_, err = state.BatchUpdateStates(ctx, []*ufspb.StateRecord{
				{
					ResourceName: "switches/switch-2",
					State:        ufspb.State_STATE_SERVING,
				},
			})
			So(err, ShouldBeNil)

			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.RegistrationsDelete, util.BrowserLabAdminRealm)
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

		Convey("Delete switch - Permission denied: same realm and no delete permission", func() {
			rack := &ufspb.Rack{
				Name: "rack-53",
				Rack: &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{},
				},
				Realm: util.BrowserLabAdminRealm,
			}
			_, err := registration.CreateRack(ctx, rack)
			So(err, ShouldBeNil)

			switch2 := mockSwitch("switch-53")
			switch2.Rack = "rack-53"
			_, err = registration.CreateSwitch(ctx, switch2)
			So(err, ShouldBeNil)

			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.RegistrationsGet, util.BrowserLabAdminRealm)
			err = DeleteSwitch(ctx, "switch-53")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, PermissionDenied)
		})

		Convey("Delete switch - Permission denied: different realm", func() {
			rack := &ufspb.Rack{
				Name: "rack-54",
				Rack: &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{},
				},
				Realm: util.BrowserLabAdminRealm,
			}
			_, err := registration.CreateRack(ctx, rack)
			So(err, ShouldBeNil)

			switch2 := mockSwitch("switch-54")
			switch2.Rack = "rack-54"
			_, err = registration.CreateSwitch(ctx, switch2)
			So(err, ShouldBeNil)

			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.RegistrationsDelete, util.AtlLabAdminRealm)
			err = DeleteSwitch(ctx, "switch-54")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, PermissionDenied)
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

func TestBatchGetSwitches(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("BatchGetSwitches", t, func() {
		Convey("Batch get switches - happy path", func() {
			entities := make([]*ufspb.Switch, 4)
			for i := 0; i < 4; i++ {
				entities[i] = &ufspb.Switch{
					Name: fmt.Sprintf("switch-batchGet-%d", i),
				}
			}
			_, err := registration.BatchUpdateSwitches(ctx, entities)
			So(err, ShouldBeNil)
			resp, err := registration.BatchGetSwitches(ctx, []string{"switch-batchGet-0", "switch-batchGet-1", "switch-batchGet-2", "switch-batchGet-3"})
			So(err, ShouldBeNil)
			So(resp, ShouldHaveLength, 4)
			So(resp, ShouldResembleProto, entities)
		})
		Convey("Batch get switches  - missing id", func() {
			resp, err := registration.BatchGetSwitches(ctx, []string{"switch-batchGet-non-existing"})
			So(err, ShouldNotBeNil)
			So(resp, ShouldBeNil)
			So(err.Error(), ShouldContainSubstring, "switch-batchGet-non-existing")
		})
		Convey("Batch get switches  - empty input", func() {
			resp, err := registration.BatchGetSwitches(ctx, nil)
			So(err, ShouldBeNil)
			So(resp, ShouldHaveLength, 0)

			input := make([]string, 0)
			resp, err = registration.BatchGetSwitches(ctx, input)
			So(err, ShouldBeNil)
			So(resp, ShouldHaveLength, 0)
		})
	})
}

func TestRenameSwitch(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	registration.CreateRack(ctx, &ufspb.Rack{
		Name:  "rack-1",
		Realm: util.BrowserLabAdminRealm,
	})
	Convey("RenameSwitch", t, func() {
		Convey("Rename a Switch with new switch name", func() {
			_, err := registration.CreateNic(ctx, &ufspb.Nic{
				Name: "nic-1",
				SwitchInterface: &ufspb.SwitchInterface{
					Switch: "switch-1",
				},
			})
			So(err, ShouldBeNil)
			_, err = registration.CreateDrac(ctx, &ufspb.Drac{
				Name: "drac-1",
				SwitchInterface: &ufspb.SwitchInterface{
					Switch: "switch-1",
				},
			})
			So(err, ShouldBeNil)
			host := mockDutMachineLSE("machinelse-1")
			host.GetChromeosMachineLse().GetDeviceLse().NetworkDeviceInterface = &ufspb.SwitchInterface{
				Switch: "switch-1",
			}
			_, err = inventory.CreateMachineLSE(ctx, host)
			So(err, ShouldBeNil)

			switch2 := mockSwitch("switch-1")
			switch2.Rack = "rack-1"
			_, err = registration.CreateSwitch(ctx, switch2)

			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.RegistrationsUpdate, util.BrowserLabAdminRealm)
			res, err := RenameSwitch(ctx, "switch-1", "switch-2")
			So(err, ShouldBeNil)
			So(res.Name, ShouldEqual, "switch-2")

			_, err = registration.GetSwitch(ctx, "switch-1")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
			nic, err := registration.GetNic(ctx, "nic-1")
			So(nic, ShouldNotBeNil)
			So(nic.GetSwitchInterface().GetSwitch(), ShouldEqual, "switch-2")
			drac, err := registration.GetDrac(ctx, "drac-1")
			So(drac, ShouldNotBeNil)
			So(drac.GetSwitchInterface().GetSwitch(), ShouldEqual, "switch-2")
			lse, err := inventory.GetMachineLSE(ctx, "machinelse-1")
			So(lse, ShouldNotBeNil)
			So(lse.GetChromeosMachineLse().GetDeviceLse().GetNetworkDeviceInterface().GetSwitch(), ShouldEqual, "switch-2")

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "switches/switch-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 2)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRename)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRename)
			So(changes[0].GetEventLabel(), ShouldEqual, "switch")
			So(changes[1].GetOldValue(), ShouldEqual, "switch-1")
			So(changes[1].GetNewValue(), ShouldEqual, "switch-2")
			So(changes[1].GetEventLabel(), ShouldEqual, "switch.name")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "switches/switch-2")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 2)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRename)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRename)
			So(changes[0].GetEventLabel(), ShouldEqual, "switch")
			So(changes[1].GetOldValue(), ShouldEqual, "switch-1")
			So(changes[1].GetNewValue(), ShouldEqual, "switch-2")
			So(changes[1].GetEventLabel(), ShouldEqual, "switch.name")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "nics/nic-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, "switch-1")
			So(changes[0].GetNewValue(), ShouldEqual, "switch-2")
			So(changes[0].GetEventLabel(), ShouldEqual, "switch_interface.switch")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "dracs/drac-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, "switch-1")
			So(changes[0].GetNewValue(), ShouldEqual, "switch-2")
			So(changes[0].GetEventLabel(), ShouldEqual, "switch_interface.switch")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "hosts/machinelse-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, "switch-1")
			So(changes[0].GetNewValue(), ShouldEqual, "switch-2")
			So(changes[0].GetEventLabel(), ShouldEqual, "switch_interface.switch")
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "switches/switch-1")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			So(msgs[0].Delete, ShouldBeTrue)
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "switches/switch-2")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			So(msgs[0].Delete, ShouldBeFalse)
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "nics/nic-1")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			So(msgs[0].Delete, ShouldBeFalse)
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "dracs/drac-1")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			So(msgs[0].Delete, ShouldBeFalse)
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/machinelse-1")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			So(msgs[0].Delete, ShouldBeFalse)
		})
		Convey("Rename a non-existing Switch", func() {
			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.RegistrationsUpdate, util.BrowserLabAdminRealm)
			_, err := RenameSwitch(ctx, "switch-3", "switch-4")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Rename a Switch to an already existing switch name", func() {
			_, err := registration.CreateSwitch(ctx, &ufspb.Switch{
				Name: "switch-5",
				Rack: "rack-1",
			})
			So(err, ShouldBeNil)

			_, err = registration.CreateSwitch(ctx, &ufspb.Switch{
				Name: "switch-6",
				Rack: "rack-1",
			})
			So(err, ShouldBeNil)

			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.RegistrationsUpdate, util.BrowserLabAdminRealm)
			_, err = RenameSwitch(ctx, "switch-5", "switch-6")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Switch switch-6 already exists in the system")
		})
		Convey("Rename a Machine - permission denied: same realm and no update permission", func() {
			_, err := registration.CreateSwitch(ctx, &ufspb.Switch{
				Name: "switch-7",
				Rack: "rack-1",
			})
			So(err, ShouldBeNil)

			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.RegistrationsGet, util.BrowserLabAdminRealm)
			_, err = RenameSwitch(ctx, "switch-7", "switch-8")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, PermissionDenied)
		})
		Convey("Rename a Switch - permission denied: different realm", func() {
			_, err := registration.CreateSwitch(ctx, &ufspb.Switch{
				Name: "switch-9",
				Rack: "rack-1",
			})
			So(err, ShouldBeNil)

			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.RegistrationsUpdate, util.AtlLabAdminRealm)
			_, err = RenameSwitch(ctx, "switch-9", "switch-10")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, PermissionDenied)
		})
	})
}
