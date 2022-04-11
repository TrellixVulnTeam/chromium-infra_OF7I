// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package resultdb

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"

	rdbpb "go.chromium.org/luci/resultdb/proto/v1"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func TestResultDB(t *testing.T) {
	t.Parallel()
	Convey(`resultdb`, t, func() {
		ctl := gomock.NewController(t)
		defer ctl.Finish()
		mc := NewMockedClient(context.Background(), ctl)
		rc, err := NewClient(mc.Ctx, "rdbhost")
		So(err, ShouldBeNil)

		inv := "invocations/build-87654321"
		Convey(`QueryTestVariants`, func() {
			req := &rdbpb.QueryTestVariantsRequest{
				Invocations: []string{inv},
				PageSize:    1000,
				Predicate: &rdbpb.TestVariantPredicate{
					Status: rdbpb.TestVariantStatus_UNEXPECTED_MASK,
				},
			}

			res := &rdbpb.QueryTestVariantsResponse{
				TestVariants: []*rdbpb.TestVariant{
					{
						TestId:      "ninja://test1",
						VariantHash: "hash1",
						Status:      rdbpb.TestVariantStatus_UNEXPECTED,
					},
					{
						TestId:      "ninja://test2",
						VariantHash: "hash2",
						Status:      rdbpb.TestVariantStatus_FLAKY,
					},
				},
			}
			mc.QueryTestVariants(req, res)

			maxPages := 1
			var tvs []*rdbpb.TestVariant
			err := rc.QueryTestVariants(mc.Ctx, req, func(res []*rdbpb.TestVariant) error {
				tvs = append(tvs, res...)
				return nil
			}, maxPages)
			So(err, ShouldBeNil)
			So(len(tvs), ShouldEqual, 2)
		})

		Convey(`GetInvocation`, func() {
			realm := "realm"
			req := &rdbpb.GetInvocationRequest{
				Name: inv,
			}
			res := &rdbpb.Invocation{
				Name:  inv,
				Realm: realm,
			}
			mc.GetInvocation(req, res)

			invProto, err := rc.GetInvocation(mc.Ctx, inv)
			So(err, ShouldBeNil)
			So(invProto, ShouldResembleProto, res)
		})

		Convey(`BatchGetTestVariants`, func() {
			req := &rdbpb.BatchGetTestVariantsRequest{
				Invocation: inv,
				TestVariants: []*rdbpb.BatchGetTestVariantsRequest_TestVariantIdentifier{
					{
						TestId:      "ninja://test1",
						VariantHash: "hash1",
					},
					{
						TestId:      "ninja://test2",
						VariantHash: "hash2",
					},
				},
			}

			res := &rdbpb.BatchGetTestVariantsResponse{
				TestVariants: []*rdbpb.TestVariant{
					{
						TestId:      "ninja://test1",
						VariantHash: "hash1",
						Status:      rdbpb.TestVariantStatus_UNEXPECTED,
					},
					{
						TestId:      "ninja://test2",
						VariantHash: "hash2",
						Status:      rdbpb.TestVariantStatus_FLAKY,
					},
				},
			}
			mc.BatchGetTestVariants(req, res)
			tvs, err := rc.BatchGetTestVariants(mc.Ctx, req)
			So(err, ShouldBeNil)
			So(len(tvs), ShouldEqual, 2)
		})
	})
}
