// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package caching

import (
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
