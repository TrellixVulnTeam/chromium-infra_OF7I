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

func mockMachineLSEPrototype(id string) *fleet.MachineLSEPrototype {
	return &fleet.MachineLSEPrototype{
		Name: id,
	}
}

func TestCreateMachineLSEPrototype(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	datastore.GetTestable(ctx).Consistent(true)
	machineLSEPrototype1 := mockMachineLSEPrototype("MachineLSEPrototype-1")
	machineLSEPrototype2 := mockMachineLSEPrototype("")
	Convey("CreateMachineLSEPrototype", t, func() {
		Convey("Create new machineLSEPrototype", func() {
			resp, err := CreateMachineLSEPrototype(ctx, machineLSEPrototype1)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, machineLSEPrototype1)
		})
		Convey("Create existing machineLSEPrototype", func() {
			resp, err := CreateMachineLSEPrototype(ctx, machineLSEPrototype1)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, AlreadyExists)
		})
		Convey("Create machineLSEPrototype - invalid ID", func() {
			resp, err := CreateMachineLSEPrototype(ctx, machineLSEPrototype2)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}

func TestUpdateMachineLSEPrototype(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	machineLSEPrototype1 := mockMachineLSEPrototype("MachineLSEPrototype-1")
	machineLSEPrototype2 := mockMachineLSEPrototype("MachineLSEPrototype-1")
	machineLSEPrototype3 := mockMachineLSEPrototype("MachineLSEPrototype-3")
	machineLSEPrototype4 := mockMachineLSEPrototype("")
	Convey("UpdateMachineLSEPrototype", t, func() {
		Convey("Update existing machineLSEPrototype", func() {
			resp, err := CreateMachineLSEPrototype(ctx, machineLSEPrototype1)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, machineLSEPrototype1)

			resp, err = UpdateMachineLSEPrototype(ctx, machineLSEPrototype2)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, machineLSEPrototype2)
		})
		Convey("Update non-existing machineLSEPrototype", func() {
			resp, err := UpdateMachineLSEPrototype(ctx, machineLSEPrototype3)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Update machineLSEPrototype - invalid ID", func() {
			resp, err := UpdateMachineLSEPrototype(ctx, machineLSEPrototype4)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}

func TestGetMachineLSEPrototype(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	machineLSEPrototype1 := mockMachineLSEPrototype("MachineLSEPrototype-1")
	Convey("GetMachineLSEPrototype", t, func() {
		Convey("Get machineLSEPrototype by existing ID", func() {
			resp, err := CreateMachineLSEPrototype(ctx, machineLSEPrototype1)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, machineLSEPrototype1)
			resp, err = GetMachineLSEPrototype(ctx, "MachineLSEPrototype-1")
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, machineLSEPrototype1)
		})
		Convey("Get machineLSEPrototype by non-existing ID", func() {
			resp, err := GetMachineLSEPrototype(ctx, "machineLSEPrototype-2")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Get machineLSEPrototype - invalid ID", func() {
			resp, err := GetMachineLSEPrototype(ctx, "")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}

func TestListMachineLSEPrototypes(t *testing.T) {
	t.Parallel()
	Convey("ListMachineLSEPrototypes", t, func() {
		ctx := gaetesting.TestingContextWithAppID("go-test")
		datastore.GetTestable(ctx).Consistent(true)
		machineLSEPrototypes := make([]*fleet.MachineLSEPrototype, 0, 4)
		for i := 0; i < 4; i++ {
			var machineLSEPrototype1 *fleet.MachineLSEPrototype
			if i == 0 {
				machineLSEPrototype1 = mockMachineLSEPrototype(fmt.Sprintf("browser-lab:machineLSEPrototype-%d", i))
			} else {
				machineLSEPrototype1 = mockMachineLSEPrototype(fmt.Sprintf("machineLSEPrototype-%d", i))
			}
			resp, err := CreateMachineLSEPrototype(ctx, machineLSEPrototype1)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, machineLSEPrototype1)
			machineLSEPrototypes = append(machineLSEPrototypes, resp)
		}
		Convey("List machineLSEPrototypes - page_token invalid", func() {
			resp, nextPageToken, err := ListMachineLSEPrototypes(ctx, 5, "abc", "")
			So(resp, ShouldBeNil)
			So(nextPageToken, ShouldBeEmpty)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InvalidPageToken)
		})
		Convey("List machineLSEPrototypes - Full listing with no pagination", func() {
			resp, nextPageToken, err := ListMachineLSEPrototypes(ctx, 4, "", "")
			So(resp, ShouldNotBeNil)
			So(nextPageToken, ShouldNotBeEmpty)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, machineLSEPrototypes)
		})
		Convey("List machineLSEPrototypes - listing with pagination", func() {
			resp, nextPageToken, err := ListMachineLSEPrototypes(ctx, 3, "", "")
			So(resp, ShouldNotBeNil)
			So(nextPageToken, ShouldNotBeEmpty)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, machineLSEPrototypes[:3])

			resp, _, err = ListMachineLSEPrototypes(ctx, 2, nextPageToken, "")
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, machineLSEPrototypes[3:])
		})
		Convey("List machineLSEPrototypes - filter only browser lab prototypes", func() {
			resp, _, err := ListMachineLSEPrototypes(ctx, 10, "", "lab:browser")
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, machineLSEPrototypes[:1])
		})
	})
}

func TestDeleteMachineLSEPrototype(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	datastore.GetTestable(ctx).Consistent(true)
	machineLSEPrototype2 := mockMachineLSEPrototype("machineLSEPrototype-2")
	Convey("DeleteMachineLSEPrototype", t, func() {
		Convey("Delete machineLSEPrototype successfully by existing ID", func() {
			resp, cerr := CreateMachineLSEPrototype(ctx, machineLSEPrototype2)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, machineLSEPrototype2)

			err := DeleteMachineLSEPrototype(ctx, "machineLSEPrototype-2")
			So(err, ShouldBeNil)

			resp, cerr = GetMachineLSEPrototype(ctx, "machineLSEPrototype-2")
			So(resp, ShouldBeNil)
			So(cerr, ShouldNotBeNil)
			So(cerr.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Delete machineLSEPrototype by non-existing ID", func() {
			err := DeleteMachineLSEPrototype(ctx, "machineLSEPrototype-2")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Delete machineLSEPrototype - invalid ID", func() {
			err := DeleteMachineLSEPrototype(ctx, "")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}
