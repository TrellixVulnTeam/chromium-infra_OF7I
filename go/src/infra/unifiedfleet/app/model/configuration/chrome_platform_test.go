// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package configuration

import (
	"fmt"
	fleet "infra/unifiedfleet/api/v1/proto"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/appengine/gaetesting"
	. "go.chromium.org/luci/common/testing/assertions"
	. "infra/unifiedfleet/app/model/datastore"
)

func mockChromePlatform(id, desc string) *fleet.ChromePlatform {
	return &fleet.ChromePlatform{
		Name:        id,
		Description: desc,
	}
}

func TestCreateChromePlatform(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	datastore.GetTestable(ctx).Consistent(true)
	chromePlatform1 := mockChromePlatform("ChromePlatform-1", "Camera")
	chromePlatform2 := mockChromePlatform("", "Sensor")
	Convey("CreateChromePlatform", t, func() {
		Convey("Create new chromePlatform", func() {
			resp, err := CreateChromePlatform(ctx, chromePlatform1)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, chromePlatform1)
		})
		Convey("Create existing chromePlatform", func() {
			resp, err := CreateChromePlatform(ctx, chromePlatform1)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, AlreadyExists)
		})
		Convey("Create chromePlatform - invalid ID", func() {
			resp, err := CreateChromePlatform(ctx, chromePlatform2)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}

func TestUpdateChromePlatform(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	chromePlatform1 := mockChromePlatform("ChromePlatform-1", "Camera")
	chromePlatform2 := mockChromePlatform("ChromePlatform-1", "Printer")
	chromePlatform3 := mockChromePlatform("ChromePlatform-3", "Sensor")
	chromePlatform4 := mockChromePlatform("", "Scanner")
	Convey("UpdateChromePlatform", t, func() {
		Convey("Update existing chromePlatform", func() {
			resp, err := CreateChromePlatform(ctx, chromePlatform1)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, chromePlatform1)

			resp, err = UpdateChromePlatform(ctx, chromePlatform2)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, chromePlatform2)
		})
		Convey("Update non-existing chromePlatform", func() {
			resp, err := UpdateChromePlatform(ctx, chromePlatform3)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Update chromePlatform - invalid ID", func() {
			resp, err := UpdateChromePlatform(ctx, chromePlatform4)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}

func TestGetChromePlatform(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	chromePlatform1 := mockChromePlatform("ChromePlatform-1", "Camera")
	Convey("GetChromePlatform", t, func() {
		Convey("Get chromePlatform by existing ID", func() {
			resp, err := CreateChromePlatform(ctx, chromePlatform1)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, chromePlatform1)
			resp, err = GetChromePlatform(ctx, "ChromePlatform-1")
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, chromePlatform1)
		})
		Convey("Get chromePlatform by non-existing ID", func() {
			resp, err := GetChromePlatform(ctx, "chromePlatform-2")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Get chromePlatform - invalid ID", func() {
			resp, err := GetChromePlatform(ctx, "")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}

func TestListChromePlatforms(t *testing.T) {
	t.Parallel()
	Convey("ListChromePlatforms", t, func() {
		ctx := gaetesting.TestingContextWithAppID("go-test")
		datastore.GetTestable(ctx).Consistent(true)
		chromePlatforms := make([]*fleet.ChromePlatform, 0, 4)
		for i := 0; i < 4; i++ {
			chromePlatform1 := mockChromePlatform(fmt.Sprintf("chromePlatform-%d", i), "Camera")
			resp, err := CreateChromePlatform(ctx, chromePlatform1)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, chromePlatform1)
			chromePlatforms = append(chromePlatforms, resp)
		}
		Convey("List chromePlatforms - page_token invalid", func() {
			resp, nextPageToken, err := ListChromePlatforms(ctx, 5, "abc")
			So(resp, ShouldBeNil)
			So(nextPageToken, ShouldBeEmpty)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InvalidPageToken)
		})

		Convey("List chromePlatforms - Full listing with no pagination", func() {
			resp, nextPageToken, err := ListChromePlatforms(ctx, 4, "")
			So(resp, ShouldNotBeNil)
			So(nextPageToken, ShouldNotBeEmpty)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, chromePlatforms)
		})

		Convey("List chromePlatforms - listing with pagination", func() {
			resp, nextPageToken, err := ListChromePlatforms(ctx, 3, "")
			So(resp, ShouldNotBeNil)
			So(nextPageToken, ShouldNotBeEmpty)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, chromePlatforms[:3])

			resp, _, err = ListChromePlatforms(ctx, 2, nextPageToken)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, chromePlatforms[3:])
		})
	})
}

func TestInsertChromePlatforms(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	datastore.GetTestable(ctx).Consistent(true)

	Convey("Add chrome platform to datastore", t, func() {
		Convey("Add 2 new platforms", func() {
			toAdd := []*fleet.ChromePlatform{
				mockChromePlatform("platform1", "Camera"),
				mockChromePlatform("platform2", "Camera"),
			}
			dsResp, err := InsertChromePlatforms(ctx, toAdd)
			So(err, ShouldBeNil)
			So(dsResp.Passed(), ShouldHaveLength, len(toAdd))
			So(dsResp.Failed(), ShouldHaveLength, 0)
			want := []string{
				toAdd[0].GetName(),
				toAdd[1].GetName(),
			}
			gets, err := GetAllChromePlatforms(ctx)
			So(err, ShouldBeNil)
			So(gets, ShouldHaveLength, 2)
			passed := gets.Passed()
			got := []string{
				passed[0].Data.(*fleet.ChromePlatform).Name,
				passed[1].Data.(*fleet.ChromePlatform).Name,
			}
			So(got, ShouldResemble, want)
		})

		Convey("Add existing platforms", func() {
			toAdd := []*fleet.ChromePlatform{
				mockChromePlatform("platform1", "Camera"),
				mockChromePlatform("platform2", "Camera"),
			}
			_, err := InsertChromePlatforms(ctx, toAdd)
			So(err, ShouldBeNil)

			toAddDuplicated := []*fleet.ChromePlatform{
				mockChromePlatform("platform1", "Camera"),
				mockChromePlatform("platform3", "Camera"),
			}
			dsResp, err := InsertChromePlatforms(ctx, toAddDuplicated)
			So(err, ShouldBeNil)
			So(dsResp.Passed(), ShouldHaveLength, 1)
			So(dsResp.Failed(), ShouldHaveLength, 1)
			So(dsResp.Passed()[0].Data.(*fleet.ChromePlatform).Name, ShouldEqual, "platform3")
			So(dsResp.Failed()[0].Data.(*fleet.ChromePlatform).Name, ShouldEqual, "platform1")

			want := []string{"platform1", "platform2", "platform3"}
			gets, err := GetAllChromePlatforms(ctx)
			So(err, ShouldBeNil)
			So(gets, ShouldHaveLength, 3)
			passed := gets.Passed()
			got := []string{
				passed[0].Data.(*fleet.ChromePlatform).Name,
				passed[1].Data.(*fleet.ChromePlatform).Name,
				passed[2].Data.(*fleet.ChromePlatform).Name,
			}
			So(got, ShouldResemble, want)
		})
	})
}
