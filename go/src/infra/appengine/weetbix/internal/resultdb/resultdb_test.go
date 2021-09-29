// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package resultdb

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"google.golang.org/grpc"

	rdbpb "go.chromium.org/luci/resultdb/proto/v1"

	. "github.com/smartystreets/goconvey/convey"
)

func TestQueryTestVariants(t *testing.T) {
	t.Parallel()

	Convey("QueryTestVariants", t, func() {
		ctl := gomock.NewController(t)
		defer ctl.Finish()
		mc := NewMockedClient(context.Background(), ctl)

		inv := "invocations/build-87654321"
		req := &rdbpb.QueryTestVariantsRequest{
			Invocations: []string{inv},
			PageSize:    1000,
		}

		resF := func(ctx context.Context, in *rdbpb.QueryTestVariantsRequest, opt grpc.CallOption) (*rdbpb.QueryTestVariantsResponse, error) {
			if in.GetPageToken() == "" {
				return &rdbpb.QueryTestVariantsResponse{
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
					NextPageToken: expectedTestVariantsPageToken,
				}, nil
			}
			return &rdbpb.QueryTestVariantsResponse{
				TestVariants: []*rdbpb.TestVariant{
					{
						TestId:      "ninja://test3",
						VariantHash: "hash3",
						Status:      rdbpb.TestVariantStatus_EXPECTED,
					},
				},
				NextPageToken: "",
			}, nil
		}
		mc.QueryTestVariants(req, resF)

		rc, err := NewClient(mc.Ctx, "rdbhost")
		So(err, ShouldBeNil)
		tvs, err := rc.QueryTestVariants(mc.Ctx, inv)
		So(err, ShouldBeNil)
		So(len(tvs), ShouldEqual, 2)
	})
}
