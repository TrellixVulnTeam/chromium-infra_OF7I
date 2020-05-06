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

func mockRack(id string, rackCapactiy int32) *proto.Rack {
	return &proto.Rack{
		Name:       id,
		CapacityRu: rackCapactiy,
	}
}

func TestCreateRack(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	datastore.GetTestable(ctx).Consistent(true)
	rack1 := mockRack("Rack-1", 5)
	rack2 := mockRack("", 10)
	Convey("CreateRack", t, func() {
		Convey("Create new rack", func() {
			resp, err := CreateRack(ctx, rack1)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, rack1)
		})
		Convey("Create existing rack", func() {
			resp, err := CreateRack(ctx, rack1)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, AlreadyExists)
		})
		Convey("Create rack - invalid ID", func() {
			resp, err := CreateRack(ctx, rack2)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}

func TestUpdateRack(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	rack1 := mockRack("Rack-1", 5)
	rack2 := mockRack("Rack-1", 10)
	rack3 := mockRack("Rack-3", 15)
	rack4 := mockRack("", 20)
	Convey("UpdateRack", t, func() {
		Convey("Update existing rack", func() {
			resp, err := CreateRack(ctx, rack1)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, rack1)

			resp, err = UpdateRack(ctx, rack2)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, rack2)
		})
		Convey("Update non-existing rack", func() {
			resp, err := UpdateRack(ctx, rack3)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Update rack - invalid ID", func() {
			resp, err := UpdateRack(ctx, rack4)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}

func TestGetRack(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	rack1 := mockRack("Rack-1", 5)
	Convey("GetRack", t, func() {
		Convey("Get rack by existing ID", func() {
			resp, err := CreateRack(ctx, rack1)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, rack1)
			resp, err = GetRack(ctx, "Rack-1")
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, rack1)
		})
		Convey("Get rack by non-existing ID", func() {
			resp, err := GetRack(ctx, "rack-2")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Get rack - invalid ID", func() {
			resp, err := GetRack(ctx, "")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}

func TestDeleteRack(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	rack1 := mockRack("rack-1", 5)
	Convey("DeleteRack", t, func() {
		Convey("Delete rack by existing ID", func() {
			resp, cerr := CreateRack(ctx, rack1)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, rack1)
			err := DeleteRack(ctx, "rack-1")
			So(err, ShouldBeNil)
			res, err := GetRack(ctx, "rack-1")
			So(res, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Delete rack by non-existing ID", func() {
			err := DeleteRack(ctx, "rack-2")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Delete rack - invalid ID", func() {
			err := DeleteRack(ctx, "")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}
