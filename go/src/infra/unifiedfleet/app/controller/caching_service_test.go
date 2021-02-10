// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"

	ufspb "infra/unifiedfleet/api/v1/models"
	"infra/unifiedfleet/app/model/caching"
	"infra/unifiedfleet/app/model/history"
	"infra/unifiedfleet/app/model/state"
)

func mockCachingService(name string) *ufspb.CachingService {
	return &ufspb.CachingService{
		Name: name,
	}
}

func TestCreateCachingService(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("CreateCachingService", t, func() {
		Convey("Create new CachingService - happy path", func() {
			cs := mockCachingService("127.0.0.1")
			cs.State = ufspb.State_STATE_SERVING
			resp, err := CreateCachingService(ctx, cs)
			So(resp, ShouldNotBeNil)
			So(resp, ShouldResembleProto, cs)

			s, err := state.GetStateRecord(ctx, "cachingservices/127.0.0.1")
			So(err, ShouldBeNil)
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_SERVING)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "cachingservices/127.0.0.1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetEventLabel(), ShouldEqual, "cachingservice")

			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "cachingservices/127.0.0.1")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			So(msgs[0].Delete, ShouldBeFalse)
		})

		Convey("Create new CachingService - already existing", func() {
			cs1 := mockCachingService("128.0.0.1")
			caching.CreateCachingService(ctx, cs1)

			cs2 := mockCachingService("128.0.0.1")
			_, err := CreateCachingService(ctx, cs2)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "already exists")

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "cachingservices/128.0.0.1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)

			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "cachingservices/128.0.0.1")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 0)
		})
	})
}
