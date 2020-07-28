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

func mockChromePlatform(id, desc string) *ufspb.ChromePlatform {
	return &ufspb.ChromePlatform{
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
	ctx := gaetesting.TestingContextWithAppID("go-test")
	datastore.GetTestable(ctx).Consistent(true)
	chromePlatforms := make([]*ufspb.ChromePlatform, 0, 4)
	for i := 0; i < 4; i++ {
		chromePlatform1 := mockChromePlatform(fmt.Sprintf("chromePlatform-%d", i), "Camera")
		resp, _ := CreateChromePlatform(ctx, chromePlatform1)
		chromePlatforms = append(chromePlatforms, resp)
	}
	Convey("ListChromePlatforms", t, func() {
		Convey("List chromePlatforms - page_token invalid", func() {
			resp, nextPageToken, err := ListChromePlatforms(ctx, 5, "abc", nil, false)
			So(resp, ShouldBeNil)
			So(nextPageToken, ShouldBeEmpty)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InvalidPageToken)
		})

		Convey("List chromePlatforms - Full listing with no pagination", func() {
			resp, nextPageToken, err := ListChromePlatforms(ctx, 4, "", nil, false)
			So(resp, ShouldNotBeNil)
			So(nextPageToken, ShouldNotBeEmpty)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, chromePlatforms)
		})

		Convey("List chromePlatforms - listing with pagination", func() {
			resp, nextPageToken, err := ListChromePlatforms(ctx, 3, "", nil, false)
			So(resp, ShouldNotBeNil)
			So(nextPageToken, ShouldNotBeEmpty)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, chromePlatforms[:3])

			resp, _, err = ListChromePlatforms(ctx, 2, nextPageToken, nil, false)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, chromePlatforms[3:])
		})
	})
}

func TestDeleteChromePlatform(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	datastore.GetTestable(ctx).Consistent(true)
	chromePlatform2 := mockChromePlatform("chromePlatform-2", "Camera")
	Convey("DeleteChromePlatform", t, func() {
		Convey("Delete chromePlatform successfully by existing ID", func() {
			resp, cerr := CreateChromePlatform(ctx, chromePlatform2)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, chromePlatform2)

			err := DeleteChromePlatform(ctx, "chromePlatform-2")
			So(err, ShouldBeNil)

			resp, cerr = GetChromePlatform(ctx, "chromePlatform-2")
			So(resp, ShouldBeNil)
			So(cerr, ShouldNotBeNil)
			So(cerr.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Delete chromePlatform by non-existing ID", func() {
			err := DeleteChromePlatform(ctx, "chromePlatform-2")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Delete chromePlatform - invalid ID", func() {
			err := DeleteChromePlatform(ctx, "")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}

func TestImportChromePlatforms(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	datastore.GetTestable(ctx).Consistent(true)

	Convey("Add chrome platform to datastore", t, func() {
		Convey("Add 2 new platforms", func() {
			toAdd := []*ufspb.ChromePlatform{
				mockChromePlatform("platform1", "Camera"),
				mockChromePlatform("platform2", "Camera"),
			}
			dsResp, err := ImportChromePlatforms(ctx, toAdd)
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
				passed[0].Data.(*ufspb.ChromePlatform).Name,
				passed[1].Data.(*ufspb.ChromePlatform).Name,
			}
			So(got, ShouldResemble, want)
		})

		Convey("Add existing platforms", func() {
			toAdd := []*ufspb.ChromePlatform{
				mockChromePlatform("platform1", "Camera"),
				mockChromePlatform("platform2", "Camera"),
			}
			_, err := ImportChromePlatforms(ctx, toAdd)
			So(err, ShouldBeNil)

			toAddDuplicated := []*ufspb.ChromePlatform{
				mockChromePlatform("platform1", "Camera2"),
				mockChromePlatform("platform3", "Camera"),
			}
			dsResp, err := ImportChromePlatforms(ctx, toAddDuplicated)
			So(err, ShouldBeNil)
			So(dsResp.Passed(), ShouldHaveLength, 2)

			got, err := GetChromePlatform(ctx, "platform1")
			So(err, ShouldBeNil)
			So(got.GetDescription(), ShouldEqual, "Camera2")
		})
	})
}
