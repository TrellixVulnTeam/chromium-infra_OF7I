// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"

	ufspb "infra/unifiedfleet/api/v1/models"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	"infra/unifiedfleet/app/util"
)

func mockCachingService(name string) *ufspb.CachingService {
	return &ufspb.CachingService{
		Name: util.AddPrefix(util.CachingServiceCollection, name),
	}
}

func TestCreateCachingService(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	Convey("CreateCachingService", t, func() {
		Convey("Create new CachingService with cachingServiceId - happy path", func() {
			cs := mockCachingService("")
			cs.PrimaryNode = "127.0.0.2"
			cs.SecondaryNode = "127.0.0.3"
			req := &ufsAPI.CreateCachingServiceRequest{
				CachingService:   cs,
				CachingServiceId: "127.0.0.1",
			}
			resp, err := tf.Fleet.CreateCachingService(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, cs)
		})

		Convey("Create new CachingService with nil entity", func() {
			req := &ufsAPI.CreateCachingServiceRequest{
				CachingService:   nil,
				CachingServiceId: "128.0.0.1",
			}
			_, err := tf.Fleet.CreateCachingService(tf.C, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.NilEntity)
		})

		Convey("Create new CachingService without cachingServiceId", func() {
			cs := mockCachingService("")
			req := &ufsAPI.CreateCachingServiceRequest{
				CachingService: cs,
			}
			_, err := tf.Fleet.CreateCachingService(tf.C, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.EmptyID)
		})

		Convey("Create new CachingService with invalid ipv4", func() {
			cs := mockCachingService("")
			req := &ufsAPI.CreateCachingServiceRequest{
				CachingService:   cs,
				CachingServiceId: "127.5.6.5666",
			}
			_, err := tf.Fleet.CreateCachingService(tf.C, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, fmt.Sprintf(ufsAPI.IPV4Format, "name"))
		})

		Convey("Create new CachingService with invalid primary node", func() {
			cs := mockCachingService("")
			cs.PrimaryNode = "127.0.0.856"
			req := &ufsAPI.CreateCachingServiceRequest{
				CachingService:   cs,
				CachingServiceId: "132.0.0.1",
			}
			_, err := tf.Fleet.CreateCachingService(tf.C, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, fmt.Sprintf(ufsAPI.IPV4Format, "primaryNode"))
		})

		Convey("Create new CachingService with invalid secondary node", func() {
			cs := mockCachingService("")
			cs.SecondaryNode = "127.0.0.856"
			req := &ufsAPI.CreateCachingServiceRequest{
				CachingService:   cs,
				CachingServiceId: "133.0.0.1",
			}
			_, err := tf.Fleet.CreateCachingService(tf.C, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, fmt.Sprintf(ufsAPI.IPV4Format, "secondaryNode"))
		})
	})
}
