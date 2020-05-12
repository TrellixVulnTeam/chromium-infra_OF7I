// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package inventory

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/appengine/gaetesting"
	. "go.chromium.org/luci/common/testing/assertions"
	proto "infra/unifiedfleet/api/v1/proto"
	. "infra/unifiedfleet/app/model/datastore"
)

func mockRackLSE(id string) *proto.RackLSE {
	return &proto.RackLSE{
		Name: id,
	}
}

func TestCreateRackLSE(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	datastore.GetTestable(ctx).Consistent(true)
	rackLSE1 := mockRackLSE("rackLSE-1")
	rackLSE2 := mockRackLSE("")
	Convey("CreateRackLSE", t, func() {
		Convey("Create new rackLSE", func() {
			resp, err := CreateRackLSE(ctx, rackLSE1)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, rackLSE1)
		})
		Convey("Create existing rackLSE", func() {
			resp, err := CreateRackLSE(ctx, rackLSE1)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, AlreadyExists)
		})
		Convey("Create rackLSE - invalid ID", func() {
			resp, err := CreateRackLSE(ctx, rackLSE2)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}

func TestUpdateRackLSE(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	rackLSE1 := mockRackLSE("rackLSE-1")
	rackLSE2 := mockRackLSE("rackLSE-1")
	rackLSE3 := mockRackLSE("rackLSE-3")
	rackLSE4 := mockRackLSE("")
	Convey("UpdateRackLSE", t, func() {
		Convey("Update existing rackLSE", func() {
			resp, err := CreateRackLSE(ctx, rackLSE1)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, rackLSE1)

			resp, err = UpdateRackLSE(ctx, rackLSE2)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, rackLSE2)
		})
		Convey("Update non-existing rackLSE", func() {
			resp, err := UpdateRackLSE(ctx, rackLSE3)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Update rackLSE - invalid ID", func() {
			resp, err := UpdateRackLSE(ctx, rackLSE4)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}
