// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package configuration

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/appengine/gaetesting"
	. "go.chromium.org/luci/common/testing/assertions"

	ufspb "infra/unifiedfleet/api/v1/proto"
	. "infra/unifiedfleet/app/model/datastore"
)

func mockRackLSEPrototype(id string) *ufspb.RackLSEPrototype {
	return &ufspb.RackLSEPrototype{
		Name: id,
	}
}

func TestCreateRackLSEPrototype(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	datastore.GetTestable(ctx).Consistent(true)
	rackLSEPrototype1 := mockRackLSEPrototype("RackLSEPrototype-1")
	rackLSEPrototype2 := mockRackLSEPrototype("")
	Convey("CreateRackLSEPrototype", t, func() {
		Convey("Create new rackLSEPrototype", func() {
			resp, err := CreateRackLSEPrototype(ctx, rackLSEPrototype1)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, rackLSEPrototype1)
		})
		Convey("Create existing rackLSEPrototype", func() {
			resp, err := CreateRackLSEPrototype(ctx, rackLSEPrototype1)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, AlreadyExists)
		})
		Convey("Create rackLSEPrototype - invalid ID", func() {
			resp, err := CreateRackLSEPrototype(ctx, rackLSEPrototype2)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}

func TestUpdateRackLSEPrototype(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	rackLSEPrototype1 := mockRackLSEPrototype("RackLSEPrototype-1")
	rackLSEPrototype2 := mockRackLSEPrototype("RackLSEPrototype-1")
	rackLSEPrototype3 := mockRackLSEPrototype("RackLSEPrototype-3")
	rackLSEPrototype4 := mockRackLSEPrototype("")
	Convey("UpdateRackLSEPrototype", t, func() {
		Convey("Update existing rackLSEPrototype", func() {
			resp, err := CreateRackLSEPrototype(ctx, rackLSEPrototype1)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, rackLSEPrototype1)

			resp, err = UpdateRackLSEPrototype(ctx, rackLSEPrototype2)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, rackLSEPrototype2)
		})
		Convey("Update non-existing rackLSEPrototype", func() {
			resp, err := UpdateRackLSEPrototype(ctx, rackLSEPrototype3)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Update rackLSEPrototype - invalid ID", func() {
			resp, err := UpdateRackLSEPrototype(ctx, rackLSEPrototype4)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}

func TestGetRackLSEPrototype(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	rackLSEPrototype1 := mockRackLSEPrototype("RackLSEPrototype-1")
	Convey("GetRackLSEPrototype", t, func() {
		Convey("Get rackLSEPrototype by existing ID", func() {
			resp, err := CreateRackLSEPrototype(ctx, rackLSEPrototype1)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, rackLSEPrototype1)
			resp, err = GetRackLSEPrototype(ctx, "RackLSEPrototype-1")
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, rackLSEPrototype1)
		})
		Convey("Get rackLSEPrototype by non-existing ID", func() {
			resp, err := GetRackLSEPrototype(ctx, "rackLSEPrototype-2")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Get rackLSEPrototype - invalid ID", func() {
			resp, err := GetRackLSEPrototype(ctx, "")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}

func TestListRackLSEPrototypes(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	datastore.GetTestable(ctx).Consistent(true)
	rackLSEPrototypes := make([]*ufspb.RackLSEPrototype, 0, 4)
	for i := 0; i < 4; i++ {
		rackLSEPrototype1 := mockRackLSEPrototype(fmt.Sprintf("rackLSEPrototype-%d", i))
		resp, _ := CreateRackLSEPrototype(ctx, rackLSEPrototype1)
		rackLSEPrototypes = append(rackLSEPrototypes, resp)
	}
	Convey("ListRackLSEPrototypes", t, func() {
		Convey("List rackLSEPrototypes - page_token invalid", func() {
			resp, nextPageToken, err := ListRackLSEPrototypes(ctx, 5, "abc", nil, false)
			So(resp, ShouldBeNil)
			So(nextPageToken, ShouldBeEmpty)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InvalidPageToken)
		})

		Convey("List rackLSEPrototypes - Full listing with no pagination", func() {
			resp, nextPageToken, err := ListRackLSEPrototypes(ctx, 4, "", nil, false)
			So(resp, ShouldNotBeNil)
			So(nextPageToken, ShouldNotBeEmpty)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, rackLSEPrototypes)
		})

		Convey("List rackLSEPrototypes - listing with pagination", func() {
			resp, nextPageToken, err := ListRackLSEPrototypes(ctx, 3, "", nil, false)
			So(resp, ShouldNotBeNil)
			So(nextPageToken, ShouldNotBeEmpty)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, rackLSEPrototypes[:3])

			resp, _, err = ListRackLSEPrototypes(ctx, 2, nextPageToken, nil, false)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, rackLSEPrototypes[3:])
		})
	})
}

func TestDeleteRackLSEPrototype(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	datastore.GetTestable(ctx).Consistent(true)
	rackLSEPrototype2 := mockRackLSEPrototype("rackLSEPrototype-2")
	Convey("DeleteRackLSEPrototype", t, func() {
		Convey("Delete rackLSEPrototype successfully by existing ID", func() {
			resp, cerr := CreateRackLSEPrototype(ctx, rackLSEPrototype2)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, rackLSEPrototype2)

			err := DeleteRackLSEPrototype(ctx, "rackLSEPrototype-2")
			So(err, ShouldBeNil)

			resp, cerr = GetRackLSEPrototype(ctx, "rackLSEPrototype-2")
			So(resp, ShouldBeNil)
			So(cerr, ShouldNotBeNil)
			So(cerr.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Delete rackLSEPrototype by non-existing ID", func() {
			err := DeleteRackLSEPrototype(ctx, "rackLSEPrototype-2")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Delete rackLSEPrototype - invalid ID", func() {
			err := DeleteRackLSEPrototype(ctx, "")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}
