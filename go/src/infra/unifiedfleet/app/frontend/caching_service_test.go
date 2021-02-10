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
	"infra/unifiedfleet/app/model/caching"
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

func TestUpdateCachingService(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	Convey("UpdateCachingService", t, func() {
		Convey("Update existing CachingService - happy path", func() {
			caching.CreateCachingService(ctx, &ufspb.CachingService{
				Name: "127.0.0.1",
			})

			cs1 := mockCachingService("127.0.0.1")
			cs1.Port = 30000
			ureq := &ufsAPI.UpdateCachingServiceRequest{
				CachingService: cs1,
			}
			resp, err := tf.Fleet.UpdateCachingService(tf.C, ureq)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, cs1)
		})

		Convey("Update CachingService - Invalid input nil", func() {
			req := &ufsAPI.UpdateCachingServiceRequest{
				CachingService: nil,
			}
			resp, err := tf.Fleet.UpdateCachingService(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.NilEntity)
		})

		Convey("Update CachingService - Invalid input empty name", func() {
			cs := mockCachingService("")
			cs.Name = ""
			req := &ufsAPI.UpdateCachingServiceRequest{
				CachingService: cs,
			}
			resp, err := tf.Fleet.UpdateCachingService(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.EmptyName)
		})

		Convey("Update CachingService - Invalid input invalid name ipv4", func() {
			cs := mockCachingService("a.b)7&")
			req := &ufsAPI.UpdateCachingServiceRequest{
				CachingService: cs,
			}
			resp, err := tf.Fleet.UpdateCachingService(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.CachingServiceNameFormat)
		})

		Convey("Update new CachingService with invalid primary node", func() {
			cs := mockCachingService("128.0.0.1")
			cs.PrimaryNode = "128.0.0.856"
			req := &ufsAPI.UpdateCachingServiceRequest{
				CachingService: cs,
			}
			_, err := tf.Fleet.UpdateCachingService(tf.C, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, fmt.Sprintf(ufsAPI.IPV4Format, "primaryNode"))
		})

		Convey("Update new CachingService with invalid secondary node", func() {
			cs := mockCachingService("129.0.0.1")
			cs.SecondaryNode = "129.0.0.856"
			req := &ufsAPI.UpdateCachingServiceRequest{
				CachingService: cs,
			}
			_, err := tf.Fleet.UpdateCachingService(tf.C, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, fmt.Sprintf(ufsAPI.IPV4Format, "secondaryNode"))
		})

	})
}
