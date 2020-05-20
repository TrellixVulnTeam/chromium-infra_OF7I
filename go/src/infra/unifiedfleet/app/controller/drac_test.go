// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"testing"

	proto "infra/unifiedfleet/api/v1/proto"
	. "infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/model/registration"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func mockDrac(id string) *proto.Drac {
	return &proto.Drac{
		Name: id,
	}
}

func TestCreateDrac(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("CreateDracs", t, func() {
		Convey("Create new drac with non existing switch", func() {
			drac1 := &proto.Drac{
				Name: "drac-1",
				SwitchInterface: &proto.SwitchInterface{
					Switch: "switch-1",
				},
			}
			resp, err := CreateDrac(ctx, drac1)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, CannotCreate)
		})

		Convey("Create new drac with existing resources", func() {
			switch2 := &proto.Switch{
				Name: "switch-2",
			}
			sresp, err := registration.CreateSwitch(ctx, switch2)
			So(err, ShouldBeNil)
			So(sresp, ShouldResembleProto, switch2)

			drac2 := &proto.Drac{
				Name: "drac-2",
				SwitchInterface: &proto.SwitchInterface{
					Switch: "switch-2",
				},
			}
			resp, err := CreateDrac(ctx, drac2)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, drac2)
		})
	})
}

func TestDeleteDrac(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	drac1 := mockDrac("drac-1")
	drac2 := mockDrac("drac-2")
	Convey("DeleteDrac", t, func() {
		Convey("Delete drac by existing ID with machine reference", func() {
			resp, cerr := CreateDrac(ctx, drac1)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, drac1)

			chromeBrowserMachine1 := &proto.Machine{
				Name: "machine-1",
				Device: &proto.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &proto.ChromeBrowserMachine{
						Drac: "drac-1",
					},
				},
			}
			mresp, merr := CreateMachine(ctx, chromeBrowserMachine1)
			So(merr, ShouldBeNil)
			So(mresp, ShouldResembleProto, chromeBrowserMachine1)

			err := DeleteDrac(ctx, "drac-1")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, CannotDelete)

			resp, cerr = GetDrac(ctx, "drac-1")
			So(resp, ShouldNotBeNil)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, drac1)
		})
		Convey("Delete drac successfully by existing ID without references", func() {
			resp, cerr := CreateDrac(ctx, drac2)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, drac2)

			err := DeleteDrac(ctx, "drac-2")
			So(err, ShouldBeNil)

			resp, cerr = GetDrac(ctx, "drac-2")
			So(resp, ShouldBeNil)
			So(cerr, ShouldNotBeNil)
			So(cerr.Error(), ShouldContainSubstring, NotFound)
		})
	})
}
