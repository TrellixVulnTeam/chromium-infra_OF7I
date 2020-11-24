// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package client

import (
	"context"
	"net/http"

	buildbucketpb "go.chromium.org/luci/buildbucket/proto"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/grpc/prpc"
	"go.chromium.org/luci/server/auth"
	"google.golang.org/grpc"
)

// BBBuildersClient is for testing purpose.
type BBBuildersClient interface {
	ListBuilders(c context.Context, in *buildbucketpb.ListBuildersRequest, opts ...grpc.CallOption) (*buildbucketpb.ListBuildersResponse, error)
}

// BuildersClient returns a buildbucketpb.BuildersClient.
func BuildersClient(c context.Context) (buildbucketpb.BuildersClient, error) {
	t, err := auth.GetRPCTransport(c, auth.AsSelf)
	if err != nil {
		return nil, err
	}
	return buildbucketpb.NewBuildersPRPCClient(&prpc.Client{
		C:    &http.Client{Transport: t},
		Host: "cr-buildbucket.appspot.com",
	}), nil
}

// ListBuildersByBucket queries BuildBucket for a list of builders for a bucket.
func ListBuildersByBucket(c context.Context, client BBBuildersClient, project string, bucket string) ([]*buildbucketpb.BuilderItem, error) {
	logging.Infof(c, "Querying for bucket (%s/%s)", project, bucket)
	// TODO (crbug.com/1103410): Handle pagination
	req := &buildbucketpb.ListBuildersRequest{
		Project:  project,
		Bucket:   bucket,
		PageSize: 1000,
	}
	res, err := client.ListBuilders(c, req)
	if err != nil {
		return nil, err
	}
	logging.Infof(c, "Got %d builders", len(res.Builders))
	return res.Builders, nil
}
