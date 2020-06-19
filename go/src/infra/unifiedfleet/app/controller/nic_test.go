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

func mockNic(id string) *proto.Nic {
	return &proto.Nic{
		Name: id,
	}
}

func TestCreateNic(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("CreateNics", t, func() {
		Convey("Create new nic with non existing switch", func() {
			nic1 := &proto.Nic{
				Name: "nic-1",
				SwitchInterface: &proto.SwitchInterface{
					Switch: "switch-1",
				},
			}
			resp, err := CreateNic(ctx, nic1)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, CannotCreate)
		})

		Convey("Create new nic with existing resources", func() {
			switch2 := &proto.Switch{
				Name: "switch-2",
			}
			sresp, err := registration.CreateSwitch(ctx, switch2)
			So(err, ShouldBeNil)
			So(sresp, ShouldResembleProto, switch2)

			nic2 := &proto.Nic{
				Name: "nic-2",
				SwitchInterface: &proto.SwitchInterface{
					Switch: "switch-2",
				},
			}
			resp, err := CreateNic(ctx, nic2)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, nic2)
		})
	})
}

func TestDeleteNic(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	nic1 := mockNic("nic-1")
	nic2 := mockNic("nic-2")
	Convey("DeleteNic", t, func() {
		Convey("Delete nic by existing ID with machine reference", func() {
			resp, cerr := CreateNic(ctx, nic1)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, nic1)

			chromeBrowserMachine1 := &proto.Machine{
				Name: "machine-1",
				Device: &proto.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &proto.ChromeBrowserMachine{
						Nics: []string{"nic-1"},
					},
				},
			}
			mresp, merr := CreateMachine(ctx, chromeBrowserMachine1)
			So(merr, ShouldBeNil)
			So(mresp, ShouldResembleProto, chromeBrowserMachine1)

			err := DeleteNic(ctx, "nic-1")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, CannotDelete)

			resp, cerr = GetNic(ctx, "nic-1")
			So(resp, ShouldNotBeNil)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, nic1)
		})
		Convey("Delete nic successfully by existing ID without references", func() {
			resp, cerr := CreateNic(ctx, nic2)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, nic2)

			err := DeleteNic(ctx, "nic-2")
			So(err, ShouldBeNil)

			resp, cerr = GetNic(ctx, "nic-2")
			So(resp, ShouldBeNil)
			So(cerr, ShouldNotBeNil)
			So(cerr.Error(), ShouldContainSubstring, NotFound)
		})
	})
}
