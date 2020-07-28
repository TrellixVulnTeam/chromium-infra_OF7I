// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package registration

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

func mockRPM(id string) *ufspb.RPM {
	return &ufspb.RPM{
		Name: id,
	}
}

func TestCreateRPM(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	datastore.GetTestable(ctx).Consistent(true)
	RPM1 := mockRPM("RPM-1")
	RPM2 := mockRPM("")
	Convey("CreateRPM", t, func() {
		Convey("Create new RPM", func() {
			resp, err := CreateRPM(ctx, RPM1)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, RPM1)
		})
		Convey("Create existing RPM", func() {
			resp, err := CreateRPM(ctx, RPM1)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, AlreadyExists)
		})
		Convey("Create RPM - invalid ID", func() {
			resp, err := CreateRPM(ctx, RPM2)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}

func TestUpdateRPM(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	RPM1 := mockRPM("RPM-1")
	RPM2 := mockRPM("RPM-1")
	RPM3 := mockRPM("RPM-3")
	RPM4 := mockRPM("")
	Convey("UpdateRPM", t, func() {
		Convey("Update existing RPM", func() {
			resp, err := CreateRPM(ctx, RPM1)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, RPM1)

			resp, err = UpdateRPM(ctx, RPM2)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, RPM2)
		})
		Convey("Update non-existing RPM", func() {
			resp, err := UpdateRPM(ctx, RPM3)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Update RPM - invalid ID", func() {
			resp, err := UpdateRPM(ctx, RPM4)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}

func TestGetRPM(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	RPM1 := mockRPM("RPM-1")
	Convey("GetRPM", t, func() {
		Convey("Get RPM by existing ID", func() {
			resp, err := CreateRPM(ctx, RPM1)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, RPM1)
			resp, err = GetRPM(ctx, "RPM-1")
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, RPM1)
		})
		Convey("Get RPM by non-existing ID", func() {
			resp, err := GetRPM(ctx, "RPM-2")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Get RPM - invalid ID", func() {
			resp, err := GetRPM(ctx, "")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}

func TestListRPMs(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	datastore.GetTestable(ctx).Consistent(true)
	RPMs := make([]*ufspb.RPM, 0, 4)
	for i := 0; i < 4; i++ {
		RPM1 := mockRPM(fmt.Sprintf("RPM-%d", i))
		resp, _ := CreateRPM(ctx, RPM1)
		RPMs = append(RPMs, resp)
	}
	Convey("ListRPMs", t, func() {
		Convey("List RPMs - page_token invalid", func() {
			resp, nextPageToken, err := ListRPMs(ctx, 5, "abc", nil, false)
			So(resp, ShouldBeNil)
			So(nextPageToken, ShouldBeEmpty)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InvalidPageToken)
		})

		Convey("List RPMs - Full listing with no pagination", func() {
			resp, nextPageToken, err := ListRPMs(ctx, 4, "", nil, false)
			So(resp, ShouldNotBeNil)
			So(nextPageToken, ShouldNotBeEmpty)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, RPMs)
		})

		Convey("List RPMs - listing with pagination", func() {
			resp, nextPageToken, err := ListRPMs(ctx, 3, "", nil, false)
			So(resp, ShouldNotBeNil)
			So(nextPageToken, ShouldNotBeEmpty)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, RPMs[:3])

			resp, _, err = ListRPMs(ctx, 2, nextPageToken, nil, false)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, RPMs[3:])
		})
	})
}

func TestDeleteRPM(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	datastore.GetTestable(ctx).Consistent(true)
	RPM4 := mockRPM("RPM-4")
	Convey("DeleteRPM", t, func() {
		Convey("Delete RPM successfully by existing ID", func() {
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
		Convey("Delete RPM by non-existing ID", func() {
			err := DeleteRPM(ctx, "RPM-6")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Delete RPM - invalid ID", func() {
			err := DeleteRPM(ctx, "")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}
