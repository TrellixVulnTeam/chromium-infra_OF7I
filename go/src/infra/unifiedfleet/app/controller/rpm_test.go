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
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	"infra/unifiedfleet/app/model/configuration"
	. "infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/model/history"
	"infra/unifiedfleet/app/model/inventory"
	"infra/unifiedfleet/app/model/registration"
	"infra/unifiedfleet/app/model/state"
	"infra/unifiedfleet/app/util"
)

func mockRPM(id string) *ufspb.RPM {
	return &ufspb.RPM{
		Name: id,
	}
}

func TestCreateRPM(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	rack1 := &ufspb.Rack{
		Name: "rack-1",
		Rack: &ufspb.Rack_ChromeBrowserRack{
			ChromeBrowserRack: &ufspb.ChromeBrowserRack{},
		},
	}
	registration.CreateRack(ctx, rack1)
	Convey("CreateRPM", t, func() {
		Convey("Create new rpm with already existing rpm - error", func() {
			rack1 := &ufspb.Rack{
				Name: "rack-5",
				Rack: &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{},
				},
			}
			registration.CreateRack(ctx, rack1)

			rpm1 := &ufspb.RPM{
				Name: "rpm-1",
				Rack: "rack-5",
			}
			_, err := registration.CreateRPM(ctx, rpm1)

			resp, err := CreateRPM(ctx, rpm1)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "RPM rpm-1 already exists in the system")

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "rpms/rpm-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})

		Convey("Create RPM - duplicated mac address", func() {
			rpm := &ufspb.RPM{
				Name:       "rpm-2-mac",
				MacAddress: "rpm-2-address",
			}
			_, err := registration.CreateRPM(ctx, rpm)
			rpm2 := &ufspb.RPM{
				Name:       "rpm-2-mac2",
				MacAddress: "rpm-2-address",
				Rack:       "rack-1",
			}
			_, err = CreateRPM(ctx, rpm2)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "mac_address rpm-2-address is already occupied")
		})

		Convey("Create new rpm with existing rack", func() {
			rack := &ufspb.Rack{
				Name: "rack-10",
				Rack: &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{},
				},
			}
			_, err := registration.CreateRack(ctx, rack)
			So(err, ShouldBeNil)

			rpm1 := &ufspb.RPM{
				Name: "rpm-20",
				Rack: "rack-10",
			}
			resp, err := CreateRPM(ctx, rpm1)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, rpm1)

			s, err := state.GetStateRecord(ctx, "rpms/rpm-20")
			So(err, ShouldBeNil)
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_REGISTERED)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "rpms/rpm-20")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetEventLabel(), ShouldEqual, "rpm")
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "rpms/rpm-20")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			So(msgs[0].Delete, ShouldBeFalse)
		})

		Convey("Create new rpm - Permission denied: same realm and no create permission", func() {
			rack := &ufspb.Rack{
				Name: "rack-20",
				Rack: &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{},
				},
				Realm: util.BrowserLabAdminRealm,
			}
			_, err := registration.CreateRack(ctx, rack)
			So(err, ShouldBeNil)

			rpm1 := &ufspb.RPM{
				Name: "rpm-20",
				Rack: "rack-20",
			}
			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.RegistrationsGet, util.BrowserLabAdminRealm)
			_, err = CreateRPM(ctx, rpm1)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, PermissionDenied)
		})

		Convey("Create new rpm - Permission denied: different realm", func() {
			rack := &ufspb.Rack{
				Name: "rack-21",
				Rack: &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{},
				},
				Realm: util.BrowserLabAdminRealm,
			}
			_, err := registration.CreateRack(ctx, rack)
			So(err, ShouldBeNil)

			rpm1 := &ufspb.RPM{
				Name: "rpm-21",
				Rack: "rack-21",
			}
			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.RegistrationsCreate, util.AtlLabAdminRealm)
			_, err = CreateRPM(ctx, rpm1)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, PermissionDenied)
		})
	})
}

func TestUpdateRPM(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("UpdateRPM", t, func() {
		Convey("Update rpm with non-existing rpm", func() {
			rack1 := &ufspb.Rack{
				Name: "rack-1",
			}
			_, err := registration.CreateRack(ctx, rack1)
			So(err, ShouldBeNil)

			rpm1 := &ufspb.RPM{
				Name: "rpm-1",
				Rack: "rack-1",
			}
			resp, err := UpdateRPM(ctx, rpm1, nil)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "rpms/rpm-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})

		Convey("Update rpm with new rack(same realm) - pass", func() {
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

			rpm3 := &ufspb.RPM{
				Name: "rpm-3",
				Rack: "rack-3",
			}
			_, err = registration.CreateRPM(ctx, rpm3)
			So(err, ShouldBeNil)

			rpm3.Rack = "rack-4"
			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.RegistrationsUpdate, util.BrowserLabAdminRealm)
			resp, err := UpdateRPM(ctx, rpm3, nil)
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp, ShouldResembleProto, rpm3)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "rpms/rpm-3")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetEventLabel(), ShouldEqual, "rpm.rack")
			So(changes[0].GetOldValue(), ShouldEqual, "rack-3")
			So(changes[0].GetNewValue(), ShouldEqual, "rack-4")
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "rpms/rpm-3")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			So(msgs[0].Delete, ShouldBeFalse)
		})

		Convey("Update rpm with same rack(same realm) - pass", func() {
			rack := &ufspb.Rack{
				Name: "rack-5",
				Rack: &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{},
				},
				Realm: util.BrowserLabAdminRealm,
			}
			_, err := registration.CreateRack(ctx, rack)
			So(err, ShouldBeNil)

			rpm1 := &ufspb.RPM{
				Name:       "rpm-5",
				Rack:       "rack-5",
				MacAddress: "rpm-10-address",
			}
			_, err = registration.CreateRPM(ctx, rpm1)
			So(err, ShouldBeNil)

			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.RegistrationsUpdate, util.BrowserLabAdminRealm)
			resp, err := UpdateRPM(ctx, rpm1, nil)
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp, ShouldResembleProto, rpm1)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "rpms/rpm-5")
			So(err, ShouldBeNil)
			// Nothing is changed for rpm-5
			So(changes, ShouldHaveLength, 0)
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "rpms/rpm-5")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			So(msgs[0].Delete, ShouldBeFalse)
		})

		Convey("Update rpm with non existing rack", func() {
			rack1 := &ufspb.Rack{
				Name: "rack-6.1",
			}
			_, err := registration.CreateRack(ctx, rack1)
			So(err, ShouldBeNil)

			rpm1 := &ufspb.RPM{
				Name: "rpm-6",
				Rack: "rack-6.1",
			}
			_, err = registration.CreateRPM(ctx, rpm1)
			So(err, ShouldBeNil)

			rpm1.Rack = "rack-6"
			resp, err := UpdateRPM(ctx, rpm1, nil)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "There is no Rack with RackID rack-6 in the system")

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "rpms/rpm-6")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})

		Convey("Partial Update rpm mac address - duplicated mac address", func() {
			rack1 := &ufspb.Rack{
				Name: "rack-8",
			}
			_, err := registration.CreateRack(ctx, rack1)
			So(err, ShouldBeNil)

			rpm := &ufspb.RPM{
				Name:       "rpm-8",
				Rack:       "rack-8",
				MacAddress: "rpm-8-address",
			}
			_, err = registration.CreateRPM(ctx, rpm)
			So(err, ShouldBeNil)

			rpm2 := &ufspb.RPM{
				Name:       "rpm-8.2",
				Rack:       "rack-8",
				MacAddress: "rpm-8.2-address",
			}
			_, err = registration.CreateRPM(ctx, rpm2)
			So(err, ShouldBeNil)

			rpm1 := &ufspb.RPM{
				Name:       "rpm-8",
				MacAddress: "rpm-8.2-address",
			}
			_, err = UpdateRPM(ctx, rpm1, &field_mask.FieldMask{Paths: []string{"macAddress"}})
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "mac_address rpm-8.2-address is already occupied")
		})

		Convey("Update rpm mac address - duplicated mac address", func() {
			rack1 := &ufspb.Rack{
				Name: "rack-9",
			}
			_, err := registration.CreateRack(ctx, rack1)
			So(err, ShouldBeNil)

			rpm := &ufspb.RPM{
				Name:       "rpm-9",
				Rack:       "rack-9",
				MacAddress: "rpm-9-address",
			}
			_, err = registration.CreateRPM(ctx, rpm)
			So(err, ShouldBeNil)
			rpm2 := &ufspb.RPM{
				Name:       "rpm-9.2",
				Rack:       "rack-9",
				MacAddress: "rpm-9.2-address",
			}
			_, err = registration.CreateRPM(ctx, rpm2)
			So(err, ShouldBeNil)

			rpm1 := &ufspb.RPM{
				Name:       "rpm-9",
				MacAddress: "rpm-9.2-address",
			}
			_, err = UpdateRPM(ctx, rpm1, nil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "mac_address rpm-9.2-address is already occupied")
		})

		Convey("Update rpm - Permission denied: same realm and no update permission", func() {
			rack := &ufspb.Rack{
				Name: "rack-51",
				Rack: &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{},
				},
				Realm: util.BrowserLabAdminRealm,
			}
			_, err := registration.CreateRack(ctx, rack)
			So(err, ShouldBeNil)

			rpm1 := &ufspb.RPM{
				Name: "rpm-51",
				Rack: "rack-51",
			}
			_, err = registration.CreateRPM(ctx, rpm1)
			So(err, ShouldBeNil)

			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.RegistrationsGet, util.BrowserLabAdminRealm)
			_, err = UpdateRPM(ctx, rpm1, nil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, PermissionDenied)
		})

		Convey("Update rpm - Permission denied: different realm", func() {
			rack := &ufspb.Rack{
				Name: "rack-52",
				Rack: &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{},
				},
				Realm: util.BrowserLabAdminRealm,
			}
			_, err := registration.CreateRack(ctx, rack)
			So(err, ShouldBeNil)

			rpm1 := &ufspb.RPM{
				Name: "rpm-52",
				Rack: "rack-52",
			}
			_, err = registration.CreateRPM(ctx, rpm1)
			So(err, ShouldBeNil)

			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.RegistrationsUpdate, util.AtlLabAdminRealm)
			_, err = UpdateRPM(ctx, rpm1, nil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, PermissionDenied)
		})

		Convey("Update rpm with new rack(different realm with no permission)- fail", func() {
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

			rpm3 := &ufspb.RPM{
				Name: "rpm-53",
				Rack: "rack-53",
			}
			_, err = registration.CreateRPM(ctx, rpm3)
			So(err, ShouldBeNil)

			rpm3.Rack = "rack-54"
			ctx := initializeFakeAuthDB(ctx, "user:user@example.com", util.RegistrationsUpdate, util.BrowserLabAdminRealm)
			_, err = UpdateRPM(ctx, rpm3, nil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, PermissionDenied)
		})

		Convey("Update rpm with new rack(different realm with permission)- pass", func() {
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

			rpm3 := &ufspb.RPM{
				Name: "rpm-55",
				Rack: "rack-55",
			}
			_, err = registration.CreateRPM(ctx, rpm3)
			So(err, ShouldBeNil)

			rpm3.Rack = "rack-56"
			ctx := auth.WithState(ctx, &authtest.FakeState{
				Identity: "user:user@example.com",
				FakeDB: authtest.NewFakeDB(
					authtest.MockMembership("user:user@example.com", "user"),
					authtest.MockPermission("user:user@example.com", util.AtlLabAdminRealm, util.RegistrationsUpdate),
					authtest.MockPermission("user:user@example.com", util.BrowserLabAdminRealm, util.RegistrationsUpdate),
				),
			})
			resp, err := UpdateRPM(ctx, rpm3, nil)
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp, ShouldResembleProto, rpm3)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "rpms/rpm-55")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetEventLabel(), ShouldEqual, "rpm.rack")
			So(changes[0].GetOldValue(), ShouldEqual, "rack-55")
			So(changes[0].GetNewValue(), ShouldEqual, "rack-56")
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "rpms/rpm-55")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			So(msgs[0].Delete, ShouldBeFalse)
		})

		Convey("Partial Update rpm with new rack(same realm) - pass", func() {
			rack := &ufspb.Rack{
				Name: "rack-57",
				Rack: &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{},
				},
				Realm: util.BrowserLabAdminRealm,
			}
			_, err := registration.CreateRack(ctx, rack)
			So(err, ShouldBeNil)

			s := &ufspb.RPM{
				Name: "rpm-57",
				Rack: "rack-57",
			}
			_, err = registration.CreateRPM(ctx, s)
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
			resp, err := UpdateRPM(ctx, s, &field_mask.FieldMask{Paths: []string{"rack"}})
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp.GetRack(), ShouldResemble, "rack-58")
		})

		Convey("Partial Update rpm with new rack(different realm with permission) - pass", func() {
			rack := &ufspb.Rack{
				Name: "rack-59",
				Rack: &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{},
				},
				Realm: util.BrowserLabAdminRealm,
			}
			_, err := registration.CreateRack(ctx, rack)
			So(err, ShouldBeNil)

			s := &ufspb.RPM{
				Name: "rpm-59",
				Rack: "rack-59",
			}
			_, err = registration.CreateRPM(ctx, s)
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
			resp, err := UpdateRPM(ctx, s, &field_mask.FieldMask{Paths: []string{"rack"}})
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp.GetRack(), ShouldResemble, "rack-60")
		})

		Convey("Partial Update rpm with new rack(different realm without permission) - fail", func() {
			rack := &ufspb.Rack{
				Name: "rack-61",
				Rack: &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{},
				},
				Realm: util.BrowserLabAdminRealm,
			}
			_, err := registration.CreateRack(ctx, rack)
			So(err, ShouldBeNil)

			s := &ufspb.RPM{
				Name: "rpm-61",
				Rack: "rack-61",
			}
			_, err = registration.CreateRPM(ctx, s)
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
			_, err = UpdateRPM(ctx, s, &field_mask.FieldMask{Paths: []string{"rack"}})
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, PermissionDenied)
		})

	})
}

func TestDeleteRPM(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	RPM1 := mockRPM("RPM-1")
	RPM3 := mockRPM("RPM-3")
	RPM4 := mockRPM("RPM-4")
	Convey("DeleteRPM", t, func() {
		Convey("Delete RPM by existing ID with machine reference", func() {
			resp, cerr := registration.CreateRPM(ctx, RPM1)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, RPM1)

			chromeBrowserMachine1 := &ufspb.Machine{
				Name: "machine-1",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{
						RpmInterface: &ufspb.RPMInterface{
							Rpm: "RPM-1",
						},
					},
				},
			}
			mresp, merr := registration.CreateMachine(ctx, chromeBrowserMachine1)
			So(merr, ShouldBeNil)
			So(mresp, ShouldResembleProto, chromeBrowserMachine1)

			err := DeleteRPM(ctx, "RPM-1")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, CannotDelete)

			resp, cerr = GetRPM(ctx, "RPM-1")
			So(resp, ShouldNotBeNil)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, RPM1)
		})

		Convey("Delete RPM by existing ID with racklse reference", func() {
			resp, cerr := registration.CreateRPM(ctx, RPM3)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, RPM3)

			chromeOSRackLSE1 := &ufspb.RackLSE{
				Name: "racklse-1",
				Lse: &ufspb.RackLSE_ChromeosRackLse{
					ChromeosRackLse: &ufspb.ChromeOSRackLSE{
						Rpms: []string{"RPM-3", "RPM-5"},
					},
				},
			}
			mresp, merr := inventory.CreateRackLSE(ctx, chromeOSRackLSE1)
			So(merr, ShouldBeNil)
			So(mresp, ShouldResembleProto, chromeOSRackLSE1)

			err := DeleteRPM(ctx, "RPM-3")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, CannotDelete)

			resp, cerr = GetRPM(ctx, "RPM-3")
			So(resp, ShouldNotBeNil)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, RPM3)
		})
		Convey("Delete RPM successfully by existing ID without references", func() {
			resp, cerr := registration.CreateRPM(ctx, RPM4)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, RPM4)

			err := DeleteRPM(ctx, "RPM-4")
			So(err, ShouldBeNil)

			resp, cerr = GetRPM(ctx, "RPM-4")
			So(resp, ShouldBeNil)
			So(cerr, ShouldNotBeNil)
			So(cerr.Error(), ShouldContainSubstring, NotFound)
		})
	})
}

func TestListRPMs(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	rpms := make([]*ufspb.RPM, 0, 2)
	for i := 0; i < 4; i++ {
		rpm := mockRPM(fmt.Sprintf("rpm-%d", i))
		resp, _ := registration.CreateRPM(ctx, rpm)
		rpms = append(rpms, resp)
	}
	Convey("ListRPMs", t, func() {
		Convey("List RPMs - filter invalid - error", func() {
			_, _, err := ListRPMs(ctx, 5, "", "invalid=mx-1", false)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Invalid field name invalid")
		})

		Convey("ListRPMs - Full listing - happy path", func() {
			resp, _, _ := ListRPMs(ctx, 5, "", "", false)
			So(resp, ShouldNotBeNil)
			So(resp, ShouldResembleProto, rpms)
		})
	})
}

func TestBatchGetRPMs(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("BatchGetRPMs", t, func() {
		Convey("Batch get rpms - happy path", func() {
			entities := make([]*ufspb.RPM, 4)
			for i := 0; i < 4; i++ {
				entities[i] = &ufspb.RPM{
					Name: fmt.Sprintf("rpm-batchGet-%d", i),
				}
			}
			_, err := registration.BatchUpdateRPMs(ctx, entities)
			So(err, ShouldBeNil)
			resp, err := registration.BatchGetRPMs(ctx, []string{"rpm-batchGet-0", "rpm-batchGet-1", "rpm-batchGet-2", "rpm-batchGet-3"})
			So(err, ShouldBeNil)
			So(resp, ShouldHaveLength, 4)
			So(resp, ShouldResembleProto, entities)
		})
		Convey("Batch get rpms  - missing id", func() {
			resp, err := registration.BatchGetRPMs(ctx, []string{"rpm-batchGet-non-existing"})
			So(err, ShouldNotBeNil)
			So(resp, ShouldBeNil)
			So(err.Error(), ShouldContainSubstring, "rpm-batchGet-non-existing")
		})
		Convey("Batch get rpms  - empty input", func() {
			resp, err := registration.BatchGetRPMs(ctx, nil)
			So(err, ShouldBeNil)
			So(resp, ShouldHaveLength, 0)

			input := make([]string, 0)
			resp, err = registration.BatchGetRPMs(ctx, input)
			So(err, ShouldBeNil)
			So(resp, ShouldHaveLength, 0)
		})
	})
}

func TestUpdateRPMIP(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	rack := &ufspb.Rack{
		Name: "rack-1-ip",
		Rack: &ufspb.Rack_ChromeBrowserRack{
			ChromeBrowserRack: &ufspb.ChromeBrowserRack{},
		},
	}
	registration.CreateRack(ctx, rack)
	Convey("TestUpdateRPMIP", t, func() {
		Convey("Update rpm by setting ip by vlan for rpm", func() {
			rpm := &ufspb.RPM{
				Name:       "rpm-1",
				Rack:       "rack-1-ip",
				MacAddress: "old_mac_address",
			}
			_, err := registration.CreateRPM(ctx, rpm)
			So(err, ShouldBeNil)

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

			err = UpdateRPMHost(ctx, rpm, &ufsAPI.NetworkOption{
				Vlan: "vlan-1",
			})
			So(err, ShouldBeNil)
			dhcp, err := configuration.GetDHCPConfig(ctx, "rpm-1")
			So(err, ShouldBeNil)
			ip, err := configuration.QueryIPByPropertyName(ctx, map[string]string{"ipv4_str": dhcp.GetIp()})
			So(err, ShouldBeNil)
			So(ip, ShouldHaveLength, 1)
			So(ip[0].GetOccupied(), ShouldBeTrue)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "states/rpms/rpm-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetEventLabel(), ShouldEqual, "state_record.state")
			So(changes[0].GetOldValue(), ShouldEqual, ufspb.State_STATE_UNSPECIFIED.String())
			So(changes[0].GetNewValue(), ShouldEqual, ufspb.State_STATE_DEPLOYING.String())
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "dhcps/rpm-1")
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
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "states/rpms/rpm-1")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "dhcps/rpm-1")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
		})

		Convey("Update rpm by deleting ip for rpm host", func() {
			rpm := &ufspb.RPM{
				Name:       "rpm-2",
				Rack:       "rack-1-ip",
				MacAddress: "old_mac_address",
			}
			_, err := registration.CreateRPM(ctx, rpm)
			So(err, ShouldBeNil)

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

			err = UpdateRPMHost(ctx, rpm, &ufsAPI.NetworkOption{
				Vlan: "vlan-1",
				Ip:   "192.168.40.12",
			})
			So(err, ShouldBeNil)
			ip, err := configuration.QueryIPByPropertyName(ctx, map[string]string{"ipv4_str": "192.168.40.12"})
			So(err, ShouldBeNil)
			So(ip, ShouldHaveLength, 1)
			So(ip[0].GetOccupied(), ShouldBeTrue)

			err = DeleteRPMHost(ctx, rpm.Name)
			So(err, ShouldBeNil)
			_, err = configuration.GetDHCPConfig(ctx, "rpm-2")
			// Not found error
			So(err, ShouldNotBeNil)
			ip2, err := configuration.QueryIPByPropertyName(ctx, map[string]string{"ipv4_str": "192.168.40.12"})
			So(err, ShouldBeNil)
			So(ip2, ShouldHaveLength, 1)
			So(ip2[0].GetOccupied(), ShouldBeFalse)
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "states/rpms/rpm-2")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 2)
			So(changes[0].GetEventLabel(), ShouldEqual, "state_record.state")
			So(changes[0].GetOldValue(), ShouldEqual, ufspb.State_STATE_UNSPECIFIED.String())
			So(changes[0].GetNewValue(), ShouldEqual, ufspb.State_STATE_DEPLOYING.String())
			So(changes[1].GetEventLabel(), ShouldEqual, "state_record.state")
			So(changes[1].GetOldValue(), ShouldEqual, ufspb.State_STATE_DEPLOYING.String())
			So(changes[1].GetNewValue(), ShouldEqual, ufspb.State_STATE_REGISTERED.String())
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "dhcps/rpm-2")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 2)
			So(changes[0].GetEventLabel(), ShouldEqual, "dhcp_config.ip")
			So(changes[0].GetOldValue(), ShouldEqual, "")
			So(changes[0].GetNewValue(), ShouldEqual, "192.168.40.12")
			So(changes[1].GetEventLabel(), ShouldEqual, "dhcp_config.ip")
			So(changes[1].GetOldValue(), ShouldEqual, "192.168.40.12")
			So(changes[1].GetNewValue(), ShouldEqual, "")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", fmt.Sprintf("ips/%s", ip[0].GetId()))
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 2)
			So(changes[0].GetEventLabel(), ShouldEqual, "ip.occupied")
			So(changes[0].GetOldValue(), ShouldEqual, "false")
			So(changes[0].GetNewValue(), ShouldEqual, "true")
			So(changes[1].GetEventLabel(), ShouldEqual, "ip.occupied")
			So(changes[1].GetOldValue(), ShouldEqual, "true")
			So(changes[1].GetNewValue(), ShouldEqual, "false")
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "states/rpms/rpm-2")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 2)
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "dhcps/rpm-2")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 2)
			So(msgs[1].Delete, ShouldBeTrue)
		})

		Convey("Update rpm by setting ip by user for host", func() {
			rpm := &ufspb.RPM{
				Name:       "rpm-3",
				Rack:       "rack-1-ip",
				MacAddress: "old_mac_address",
			}
			_, err := registration.CreateRPM(ctx, rpm)
			So(err, ShouldBeNil)

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

			err = UpdateRPMHost(ctx, rpm, &ufsAPI.NetworkOption{
				Vlan: "vlan-1",
				Ip:   "192.168.40.19",
			})
			So(err, ShouldBeNil)
			dhcp, err := configuration.GetDHCPConfig(ctx, "rpm-3")
			So(err, ShouldBeNil)
			So(dhcp.GetIp(), ShouldEqual, "192.168.40.19")
			ip, err := configuration.QueryIPByPropertyName(ctx, map[string]string{"ipv4_str": "192.168.40.19"})
			So(err, ShouldBeNil)
			So(ip, ShouldHaveLength, 1)
			So(ip[0].GetOccupied(), ShouldBeTrue)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "states/rpms/rpm-3")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetEventLabel(), ShouldEqual, "state_record.state")
			So(changes[0].GetOldValue(), ShouldEqual, "STATE_UNSPECIFIED")
			So(changes[0].GetNewValue(), ShouldEqual, "STATE_DEPLOYING")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "dhcps/rpm-3")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetEventLabel(), ShouldEqual, "dhcp_config.ip")
			So(changes[0].GetOldValue(), ShouldEqual, "")
			So(changes[0].GetNewValue(), ShouldEqual, dhcp.GetIp())
			changes, err = history.QueryChangesByPropertyName(ctx, "name", fmt.Sprintf("ips/%s", ip[0].GetId()))
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetEventLabel(), ShouldEqual, "ip.occupied")
			So(changes[0].GetOldValue(), ShouldEqual, "false")
			So(changes[0].GetNewValue(), ShouldEqual, "true")
		})

	})
}
