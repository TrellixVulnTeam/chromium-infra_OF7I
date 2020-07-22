// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"testing"

	ufspb "infra/unifiedfleet/api/v1/proto"
	. "infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/model/inventory"
	"infra/unifiedfleet/app/model/registration"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
	"infra/unifiedfleet/app/model/history"
)

func TestCreateRack(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("CreateRack", t, func() {
		Convey("Create new rack with output only fields", func() {
			rack := &ufspb.Rack{
				Name: "rack-1",
				Rack: &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{
						Switches: []string{"switch-3"},
						Kvms:     []string{"kvm-3"},
						Rpms:     []string{"rpm-3"},
					},
				},
			}
			resp, err := CreateRack(ctx, rack)
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp.GetChromeBrowserRack().GetKvms(), ShouldBeNil)
			So(resp.GetChromeBrowserRack().GetRpms(), ShouldBeNil)
			So(resp.GetChromeBrowserRack().GetSwitches(), ShouldBeNil)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "racks/rack-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetEventLabel(), ShouldEqual, "rack")
		})

		Convey("Create new rack with nil browser/chromeos rack", func() {
			rack := &ufspb.Rack{
				Name: "rack-2",
				Location: &ufspb.Location{
					Lab: ufspb.Lab_LAB_CHROME_ATLANTA,
				},
			}
			resp, err := CreateRack(ctx, rack)
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp.GetChromeBrowserRack(), ShouldNotBeNil)

			resp, _ = registration.GetRack(ctx, "rack-2")
			So(resp.GetChromeBrowserRack(), ShouldNotBeNil)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "racks/rack-2")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetEventLabel(), ShouldEqual, "rack")
		})

		Convey("Create new rack with nil browser/chromeos rack and no lcoation info", func() {
			rack := &ufspb.Rack{
				Name: "rack-3",
			}
			_, err := CreateRack(ctx, rack)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "lab information in the location object cannot be empty/unspecified for a rack")

			// No changes are recorded as the creation fails
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "racks/rack-3")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})
	})
}

func TestUpdateRack(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("UpdateRack", t, func() {
		Convey("Update non-existing rack - error", func() {
			rack := &ufspb.Rack{
				Name: "rack-1",
			}
			_, err := UpdateRack(ctx, rack)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "There is no Rack with RackID rack-1 in the system.")
		})

		Convey("Update existing rack with output only fields", func() {
			rack := &ufspb.Rack{
				Name: "rack-2",
				Rack: &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{
						Switches: []string{"switch-2"},
						Kvms:     []string{"kvm-2"},
						Rpms:     []string{"rpm-2"},
					},
				},
			}
			_, err := registration.CreateRack(ctx, rack)
			So(err, ShouldBeNil)

			rack = &ufspb.Rack{
				Name: "rack-2",
				Rack: &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{
						Switches: []string{"switch-5"},
						Kvms:     []string{"kvm-5"},
						Rpms:     []string{"rpm-5"},
					},
				},
			}
			resp, err := UpdateRack(ctx, rack)
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp, ShouldResembleProto, rack)
			So(resp.GetChromeBrowserRack().GetKvms(), ShouldResemble, []string{"kvm-2"})
			So(resp.GetChromeBrowserRack().GetRpms(), ShouldResemble, []string{"rpm-2"})
			So(resp.GetChromeBrowserRack().GetSwitches(), ShouldResemble, []string{"switch-2"})
		})

		Convey("Update existing rack with nil rack/browser machine", func() {
			rack := &ufspb.Rack{
				Name: "rack-3",
				Rack: &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{
						Switches: []string{"switch-3"},
						Kvms:     []string{"kvm-3"},
						Rpms:     []string{"rpm-3"},
					},
				},
			}
			_, err := registration.CreateRack(ctx, rack)
			So(err, ShouldBeNil)

			rack = &ufspb.Rack{
				Name: "rack-3",
			}
			resp, err := UpdateRack(ctx, rack)
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp.GetChromeBrowserRack().GetKvms(), ShouldResemble, []string{"kvm-3"})
			So(resp.GetChromeBrowserRack().GetRpms(), ShouldResemble, []string{"rpm-3"})
			So(resp.GetChromeBrowserRack().GetSwitches(), ShouldResemble, []string{"switch-3"})
		})
	})
}

func TestDeleteRack(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("DeleteRack", t, func() {
		Convey("Delete rack by existing ID with rackLSE reference", func() {
			rack1 := &ufspb.Rack{
				Name: "rack-3",
			}
			resp, cerr := registration.CreateRack(ctx, rack1)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, rack1)

			rackLSE1 := &ufspb.RackLSE{
				Name:  "racklse-1",
				Racks: []string{"rack-3"},
			}
			mresp, merr := inventory.CreateRackLSE(ctx, rackLSE1)
			So(merr, ShouldBeNil)
			So(mresp, ShouldResembleProto, rackLSE1)

			err := DeleteRack(ctx, "rack-3")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, CannotDelete)

			resp, cerr = registration.GetRack(ctx, "rack-3")
			So(resp, ShouldNotBeNil)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, rack1)

			// No changes are recorded as the deletion fails
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "racks/rack-3")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})
		Convey("Delete rack by existing ID without references", func() {
			rack2 := &ufspb.Rack{
				Name: "rack-4",
			}
			resp, cerr := registration.CreateRack(ctx, rack2)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, rack2)
			err := DeleteRack(ctx, "rack-4")
			So(err, ShouldBeNil)
			res, err := registration.GetRack(ctx, "rack-4")
			So(res, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "racks/rack-4")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRetire)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRetire)
			So(changes[0].GetEventLabel(), ShouldEqual, "rack")
		})
	})
}

func TestReplaceRack(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("ReplaceRacks", t, func() {
		Convey("Repalce an old Rack with new rack with RackLSE reference", func() {
			oldRack1 := &ufspb.Rack{
				Name: "rack-4",
			}
			_, err := registration.CreateRack(ctx, oldRack1)
			So(err, ShouldBeNil)

			rackLSE1 := &ufspb.RackLSE{
				Name:  "racklse-1",
				Racks: []string{"rack-0", "rack-50", "rack-4", "rack-7"},
			}
			_, err = inventory.CreateRackLSE(ctx, rackLSE1)
			So(err, ShouldBeNil)

			newRack2 := &ufspb.Rack{
				Name: "rack-100",
			}
			resp, err := ReplaceRack(ctx, oldRack1, newRack2)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, newRack2)

			rlse, err := inventory.GetRackLSE(ctx, "racklse-1")
			So(rlse, ShouldNotBeNil)
			So(rlse.GetRacks(), ShouldResemble, []string{"rack-0", "rack-50", "rack-100", "rack-7"})

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "racks/rack-4")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRetire)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRetire)
			So(changes[0].GetEventLabel(), ShouldEqual, "rack")

			changes, err = history.QueryChangesByPropertyName(ctx, "name", "racks/rack-100")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetEventLabel(), ShouldEqual, "rack")
		})

		Convey("Repalce an old Rack with already existing rack", func() {
			existingRack1 := &ufspb.Rack{
				Name: "rack-105",
			}
			_, err := registration.CreateRack(ctx, existingRack1)
			So(err, ShouldBeNil)

			oldRack1 := &ufspb.Rack{
				Name: "rack-5",
			}
			_, err = registration.CreateRack(ctx, oldRack1)
			So(err, ShouldBeNil)

			newRack2 := &ufspb.Rack{
				Name: "rack-105",
			}
			rresp, rerr := ReplaceRack(ctx, oldRack1, newRack2)
			So(rerr, ShouldNotBeNil)
			So(rresp, ShouldBeNil)
			So(rerr.Error(), ShouldContainSubstring, AlreadyExists)

			// No change are recorded as the replacement fails
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "racks/rack-5")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "racks/rack-105")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})
	})
}
