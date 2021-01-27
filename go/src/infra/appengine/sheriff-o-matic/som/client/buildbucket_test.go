// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package client

import (
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	buildbucketpb "go.chromium.org/luci/buildbucket/proto"
	"go.chromium.org/luci/common/logging/gologger"
	"google.golang.org/grpc"
)

type mockBuildersClient struct{}

func (mbc mockBuildersClient) ListBuilders(c context.Context, req *buildbucketpb.ListBuildersRequest, opts ...grpc.CallOption) (*buildbucketpb.ListBuildersResponse, error) {
	if req.PageToken == "" {
		return &buildbucketpb.ListBuildersResponse{
			Builders: []*buildbucketpb.BuilderItem{
				{
					Id: &buildbucketpb.BuilderID{
						Project: "chromium",
						Bucket:  "ci",
						Builder: "ci_1",
					},
				},
				{
					Id: &buildbucketpb.BuilderID{
						Project: "chromium",
						Bucket:  "ci",
						Builder: "ci_2",
					},
				},
			},
			NextPageToken: "token",
		}, nil
	}
	if req.PageToken == "token" {
		return &buildbucketpb.ListBuildersResponse{
			Builders: []*buildbucketpb.BuilderItem{
				{
					Id: &buildbucketpb.BuilderID{
						Project: "chromium",
						Bucket:  "ci",
						Builder: "ci_3",
					},
				},
			},
		}, nil
	}
	return nil, nil
}

func TestListBuildersByBucket(t *testing.T) {
	ctx := context.Background()
	ctx = gologger.StdConfig.Use(ctx)
	cl := mockBuildersClient{}

	Convey("no builder", t, func() {
		builders, err := ListBuildersByBucket(ctx, cl, "chromium", "ci")
		So(err, ShouldBeNil)
		So(len(builders), ShouldEqual, 3)
		So(builders[0].Id.Builder, ShouldEqual, "ci_1")
		So(builders[1].Id.Builder, ShouldEqual, "ci_2")
		So(builders[2].Id.Builder, ShouldEqual, "ci_3")
	})
}
