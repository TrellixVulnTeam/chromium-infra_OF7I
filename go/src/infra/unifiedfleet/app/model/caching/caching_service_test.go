// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package caching

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/luci/appengine/gaetesting"
	. "go.chromium.org/luci/common/testing/assertions"
	"go.chromium.org/luci/gae/service/datastore"

	ufspb "infra/unifiedfleet/api/v1/models"
	. "infra/unifiedfleet/app/model/datastore"
)

func mockCachingService(name string) *ufspb.CachingService {
	return &ufspb.CachingService{
		Name: name,
	}
}

func TestCreateCachingService(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	datastore.GetTestable(ctx).Consistent(true)
	Convey("CreateCachingService", t, func() {
		Convey("Create new CachingService", func() {
			cs := mockCachingService("127.0.0.1")
			resp, err := CreateCachingService(ctx, cs)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, cs)
		})
		Convey("Create existing CachingService", func() {
			cs1 := mockCachingService("128.0.0.1")
			CreateCachingService(ctx, cs1)

			resp, err := CreateCachingService(ctx, cs1)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, AlreadyExists)
		})
	})
}

func TestBatchCreateCachingServices(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	datastore.GetTestable(ctx).Consistent(true)
	Convey("BatchUpdateCachingServices", t, func() {
		Convey("Create new CachingService", func() {
			cs := mockCachingService("128.0.0.1")
			resp, err := BatchUpdateCachingServices(ctx, []*ufspb.CachingService{cs})
			So(err, ShouldBeNil)
			So(resp[0], ShouldResembleProto, cs)
		})
	})
}

func TestGetCachingService(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	cs1 := mockCachingService("cs-1")
	Convey("GetCachingService", t, func() {
		Convey("Get CachingService by existing name/ID", func() {
			resp, err := CreateCachingService(ctx, cs1)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, cs1)
			resp, err = GetCachingService(ctx, "cs-1")
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, cs1)
		})
		Convey("Get CachingService by non-existing name/ID", func() {
			resp, err := GetCachingService(ctx, "cs-2")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Get CachingService - invalid name/ID", func() {
			resp, err := GetCachingService(ctx, "")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}

func TestDeleteCachingService(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	datastore.GetTestable(ctx).Consistent(true)
	cs1 := mockCachingService("cs-1")
	CreateCachingService(ctx, cs1)
	Convey("DeleteCachingService", t, func() {
		Convey("Delete CachingService successfully by existing ID", func() {
			err := DeleteCachingService(ctx, "cs-1")
			So(err, ShouldBeNil)

			resp, err := GetCachingService(ctx, "cs-1")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Delete CachingService by non-existing ID", func() {
			err := DeleteCachingService(ctx, "cs-5")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Delete CachingService - invalid ID", func() {
			err := DeleteCachingService(ctx, "")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}

func TestListCachingServices(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	datastore.GetTestable(ctx).Consistent(true)
	cachingServices := make([]*ufspb.CachingService, 0, 4)
	for i := 0; i < 4; i++ {
		cs := mockCachingService(fmt.Sprintf("cs-%d", i))
		resp, _ := CreateCachingService(ctx, cs)
		cachingServices = append(cachingServices, resp)
	}
	Convey("ListCachingServices", t, func() {
		Convey("List CachingServices - page_token invalid", func() {
			resp, nextPageToken, err := ListCachingServices(ctx, 5, "abc", nil, false)
			So(resp, ShouldBeNil)
			So(nextPageToken, ShouldBeEmpty)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InvalidPageToken)
		})

		Convey("List CachingServices - Full listing with no pagination", func() {
			resp, nextPageToken, err := ListCachingServices(ctx, 4, "", nil, false)
			So(resp, ShouldNotBeNil)
			So(nextPageToken, ShouldNotBeEmpty)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, cachingServices)
		})

		Convey("List CachingServices - listing with pagination", func() {
			resp, nextPageToken, err := ListCachingServices(ctx, 3, "", nil, false)
			So(resp, ShouldNotBeNil)
			So(nextPageToken, ShouldNotBeEmpty)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, cachingServices[:3])

			resp, _, err = ListCachingServices(ctx, 2, nextPageToken, nil, false)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, cachingServices[3:])
		})
	})
}
