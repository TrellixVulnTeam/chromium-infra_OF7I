// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"testing"

	proto "infra/unifiedfleet/api/v1/proto"
	. "infra/unifiedfleet/app/model/datastore"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func mockSwitch(id string) *proto.Switch {
	return &proto.Switch{
		Name: id,
	}
}

func TestDeleteSwitch(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	switch1 := mockSwitch("switch-1")
	switch2 := mockSwitch("switch-2")
	Convey("DeleteSwitch", t, func() {
		Convey("Delete switch by existing ID with machine reference", func() {
			resp, cerr := CreateSwitch(ctx, switch1)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, switch1)

			nic := &proto.Nic{
				Name: "machine1-eth0",
				SwitchInterface: &proto.SwitchInterface{
					Switch: "switch-1",
				},
			}
			mresp, merr := CreateNic(ctx, nic)
			So(merr, ShouldBeNil)
			So(mresp, ShouldResembleProto, nic)

			err := DeleteSwitch(ctx, "switch-1")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, CannotDelete)

			resp, cerr = GetSwitch(ctx, "switch-1")
			So(resp, ShouldNotBeNil)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, switch1)
		})
		Convey("Delete switch successfully by existing ID without references", func() {
			resp, cerr := CreateSwitch(ctx, switch2)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, switch2)

			err := DeleteSwitch(ctx, "switch-2")
			So(err, ShouldBeNil)

			resp, cerr = GetSwitch(ctx, "switch-2")
			So(resp, ShouldBeNil)
			So(cerr, ShouldNotBeNil)
			So(cerr.Error(), ShouldContainSubstring, NotFound)
		})
	})
}
