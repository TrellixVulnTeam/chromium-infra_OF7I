// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package client

import (
	"context"
	"net/http"

	buildbucketpb "go.chromium.org/luci/buildbucket/proto"
	"go.chromium.org/luci/grpc/prpc"
	"go.chromium.org/luci/server/auth"
)

// BuildersClient returns a buildbucketpb.BuildersClient.
func BuildersClient(c context.Context, host string) (buildbucketpb.BuildersClient, error) {
	t, err := auth.GetRPCTransport(c, auth.AsSelf)
	if err != nil {
		return nil, err
	}
	return buildbucketpb.NewBuildersPRPCClient(&prpc.Client{
		C:    &http.Client{Transport: t},
		Host: host,
	}), nil
}

// ListBuilders queries BuildBucket for a list of builders.
func ListBuilders(c context.Context, req *buildbucketpb.ListBuildersRequest) (*buildbucketpb.ListBuildersResponse, error) {
	client, err := BuildersClient(c, "cr-buildbucket.appspot.com")
	res, err := client.ListBuilders(c, req)
	return res, err
}

// ListBuildersByBucket queries BuildBucket for a list of builders for a bucket.
func ListBuildersByBucket(c context.Context, project string, bucket string) ([]*buildbucketpb.BuilderItem, error) {
	// TODO (crbug.com/1103410): Handle pagination
	req := &buildbucketpb.ListBuildersRequest{
		Project:  project,
		Bucket:   bucket,
		PageSize: 1000,
	}
	res, err := ListBuilders(c, req)
	if err != nil {
		return nil, err
	}
	return res.Builders, nil
}
