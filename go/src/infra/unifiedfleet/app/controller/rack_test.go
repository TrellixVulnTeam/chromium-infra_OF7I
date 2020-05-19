// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"testing"

	proto "infra/unifiedfleet/api/v1/proto"
	. "infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/model/inventory"
	"infra/unifiedfleet/app/model/registration"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func TestCreateRack(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("CreateRacks", t, func() {
		Convey("Create new rack with non existing switch", func() {
			rack1 := &proto.Rack{
				Name: "rack-1",
				Rack: &proto.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &proto.ChromeBrowserRack{
						Switches: []string{"switch-1"},
					},
				},
			}
			resp, err := CreateRack(ctx, rack1)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, CannotCreate)
		})

		Convey("Create new rack with non existing resources", func() {
			rack3 := &proto.Rack{
				Name: "rack-3",
				Rack: &proto.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &proto.ChromeBrowserRack{
						Switches: []string{"switch-3"},
						Kvms:     []string{"kvm-3"},
						Rpms:     []string{"rpm-3"},
					},
				},
			}
			resp, err := CreateRack(ctx, rack3)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, CannotCreate)
		})

		Convey("Create new rack with existing resources", func() {
			switch2 := &proto.Switch{
				Name: "switch-2",
			}
			sresp, err := registration.CreateSwitch(ctx, switch2)
			So(err, ShouldBeNil)
			So(sresp, ShouldResembleProto, switch2)

			kvm2 := &proto.KVM{
				Name: "kvm-2",
			}
			kresp, err := registration.CreateKVM(ctx, kvm2)
			So(err, ShouldBeNil)
			So(kresp, ShouldResembleProto, kvm2)

			rpm2 := &proto.RPM{
				Name: "rpm-2",
			}
			rresp, err := registration.CreateRPM(ctx, rpm2)
			So(err, ShouldBeNil)
			So(rresp, ShouldResembleProto, rpm2)

			rack2 := &proto.Rack{
				Name: "rack-2",
				Rack: &proto.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &proto.ChromeBrowserRack{
						Switches: []string{"switch-2"},
						Kvms:     []string{"kvm-2"},
						Rpms:     []string{"rpm-2"},
					},
				},
			}
			resp, err := CreateRack(ctx, rack2)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, rack2)
		})
	})
}

func TestDeleteRack(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("DeleteRack", t, func() {
		Convey("Delete rack by existing ID with rackLSE reference", func() {
			rack1 := &proto.Rack{
				Name: "rack-3",
			}
			resp, cerr := registration.CreateRack(ctx, rack1)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, rack1)

			rackLSE1 := &proto.RackLSE{
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
		})
		Convey("Delete rack by existing ID without references", func() {
			rack2 := &proto.Rack{
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
		})
	})
}

func TestReplaceRack(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("ReplaceRacks", t, func() {
		Convey("Repalce an old Rack with new rack with RackLSE reference", func() {
			oldRack1 := &proto.Rack{
				Name: "rack-4",
			}
			resp, cerr := CreateRack(ctx, oldRack1)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, oldRack1)

			rackLSE1 := &proto.RackLSE{
				Name:  "racklse-1",
				Racks: []string{"rack-0", "rack-50", "rack-4", "rack-7"},
			}
			mresp, merr := inventory.CreateRackLSE(ctx, rackLSE1)
			So(merr, ShouldBeNil)
			So(mresp, ShouldResembleProto, rackLSE1)

			newRack2 := &proto.Rack{
				Name: "rack-100",
			}
			rresp, rerr := ReplaceRack(ctx, oldRack1, newRack2)
			So(rerr, ShouldBeNil)
			So(rresp, ShouldResembleProto, newRack2)

			mresp, merr = inventory.GetRackLSE(ctx, "racklse-1")
			So(merr, ShouldBeNil)
			So(mresp.GetRacks(), ShouldResemble, []string{"rack-0", "rack-50", "rack-100", "rack-7"})
		})

		Convey("Repalce an old Rack with already existing rack", func() {
			existingRack1 := &proto.Rack{
				Name: "rack-105",
			}
			resp, cerr := CreateRack(ctx, existingRack1)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, existingRack1)

			oldRack1 := &proto.Rack{
				Name: "rack-5",
			}
			resp, cerr = CreateRack(ctx, oldRack1)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, oldRack1)

			newRack2 := &proto.Rack{
				Name: "rack-105",
			}
			rresp, rerr := ReplaceRack(ctx, oldRack1, newRack2)
			So(rerr, ShouldNotBeNil)
			So(rresp, ShouldBeNil)
			So(rerr.Error(), ShouldContainSubstring, AlreadyExists)
		})
	})
}
