// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package registration

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/appengine/gaetesting"
	. "go.chromium.org/luci/common/testing/assertions"
	proto "infra/unifiedfleet/api/v1/proto"
	. "infra/unifiedfleet/app/model/datastore"
)

func mockNic(id string) *proto.Nic {
	return &proto.Nic{
		Name: id,
	}
}

func TestCreateNic(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	datastore.GetTestable(ctx).Consistent(true)
	nic1 := mockNic("Nic-1")
	nic2 := mockNic("")
	Convey("CreateNic", t, func() {
		Convey("Create new nic", func() {
			resp, err := CreateNic(ctx, nic1)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, nic1)
		})
		Convey("Create existing nic", func() {
			resp, err := CreateNic(ctx, nic1)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, AlreadyExists)
		})
		Convey("Create nic - invalid ID", func() {
			resp, err := CreateNic(ctx, nic2)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}

func TestUpdateNic(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	nic1 := mockNic("Nic-1")
	nic2 := mockNic("Nic-1")
	nic3 := mockNic("Nic-3")
	nic4 := mockNic("")
	Convey("UpdateNic", t, func() {
		Convey("Update existing nic", func() {
			resp, err := CreateNic(ctx, nic1)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, nic1)

			resp, err = UpdateNic(ctx, nic2)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, nic2)
		})
		Convey("Update non-existing nic", func() {
			resp, err := UpdateNic(ctx, nic3)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Update nic - invalid ID", func() {
			resp, err := UpdateNic(ctx, nic4)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}
