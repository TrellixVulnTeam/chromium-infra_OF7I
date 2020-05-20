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

func mockRPM(id string) *proto.RPM {
	return &proto.RPM{
		Name: id,
	}
}

func TestDeleteRPM(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	RPM1 := mockRPM("RPM-1")
	RPM2 := mockRPM("RPM-2")
	RPM3 := mockRPM("RPM-3")
	RPM4 := mockRPM("RPM-4")
	Convey("DeleteRPM", t, func() {
		Convey("Delete RPM by existing ID with machine reference", func() {
			resp, cerr := CreateRPM(ctx, RPM1)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, RPM1)

			chromeBrowserMachine1 := &proto.Machine{
				Name: "machine-1",
				Device: &proto.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &proto.ChromeBrowserMachine{
						RpmInterface: &proto.RPMInterface{
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
		Convey("Delete RPM by existing ID with rack reference", func() {
			resp, cerr := CreateRPM(ctx, RPM2)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, RPM2)

			chromeBrowserRack1 := &proto.Rack{
				Name: "rack-1",
				Rack: &proto.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &proto.ChromeBrowserRack{
						Rpms: []string{"RPM-2", "RPM-5"},
					},
				},
			}
			mresp, merr := registration.CreateRack(ctx, chromeBrowserRack1)
			So(merr, ShouldBeNil)
			So(mresp, ShouldResembleProto, chromeBrowserRack1)

			err := DeleteRPM(ctx, "RPM-2")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, CannotDelete)

			resp, cerr = GetRPM(ctx, "RPM-2")
			So(resp, ShouldNotBeNil)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, RPM2)
		})
		Convey("Delete RPM by existing ID with racklse reference", func() {
			resp, cerr := CreateRPM(ctx, RPM3)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, RPM3)

			chromeOSRackLSE1 := &proto.RackLSE{
				Name: "racklse-1",
				Lse: &proto.RackLSE_ChromeosRackLse{
					ChromeosRackLse: &proto.ChromeOSRackLSE{
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
			resp, cerr := CreateRPM(ctx, RPM4)
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
