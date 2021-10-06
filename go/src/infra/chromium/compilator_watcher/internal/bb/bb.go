// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package bb provides a buildbucket Client with helper methods for interacting
// with builds.
package bb

import (
	"context"

	"go.chromium.org/luci/auth"
	buildbucket_pb "go.chromium.org/luci/buildbucket/proto"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/grpc/prpc"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type buildsClient interface {
	GetBuild(ctx context.Context, in *buildbucket_pb.GetBuildRequest, opts ...grpc.CallOption) (*buildbucket_pb.Build, error)
}

// Client provides helper methods to interact with buildbucket builds.
type Client struct {
	client buildsClient
}

// NewClient returns a new client to interact with buildbucket builds from the given builder.
func NewClient(ctx context.Context) (*Client, error) {
	if buildResponses := ctx.Value(FakeBuildsContextKey); buildResponses != nil {
		return &Client{client: &fakeBuildClient{
			responses: buildResponses.([]FakeGetBuildResponse),
			callCount: 0}}, nil
	}
	httpClient, err := auth.NewAuthenticator(ctx, auth.SilentLogin, auth.Options{}).Client()
	if err != nil {
		return nil, err
	}

	options := *prpc.DefaultOptions()
	options.UserAgent = "compilator_watcher"

	pClient := &prpc.Client{
		C: httpClient,
		// TODO: Change Host to be passed in as a command line argument if a non-prod buildbucket host is needed
		Host:    "cr-buildbucket.appspot.com",
		Options: &options,
	}

	return &Client{
		client: buildbucket_pb.NewBuildsPRPCClient(pClient),
	}, nil
}

// getBuildFields is the list of buildbucket fields that are needed.
// See go/buildbucket-proto for the list of all fields.
var getBuildFields = []string{
	"id",
	// Build details are parsed from the build's properties.
	"output.properties",
	// Build status is used to determine whether the build is complete.
	"status",
	"steps",
	"summary_markdown",
	"start_time",
	"end_time",
}

// GetBuild gets a buildbucket build by ID.
func (c *Client) GetBuild(ctx context.Context, ID int64) (*buildbucket_pb.Build, error) {
	req := &buildbucket_pb.GetBuildRequest{
		Id:     ID,
		Fields: &fieldmaskpb.FieldMask{Paths: getBuildFields},
	}
	build, err := c.client.GetBuild(ctx, req)
	if err != nil {
		return nil, errors.Annotate(err, "get build").Err()
	}
	return build, nil
}

const FakeBuildsContextKey = "Key for fake builds"

type FakeGetBuildResponse struct {
	// Build is the build message that will be returned as the GetBuild response
	//
	// Mutually exclusive with Err
	Build *buildbucket_pb.Build
	// error to return instead of a Build
	//
	// Mutually exclusive with Err
	Err error
}

type fakeBuildClient struct {
	responses []FakeGetBuildResponse
	callCount int
}

func (c *fakeBuildClient) GetBuild(ctx context.Context, in *buildbucket_pb.GetBuildRequest, opts ...grpc.CallOption) (*buildbucket_pb.Build, error) {
	to_return := c.responses[c.callCount]
	c.callCount += 1
	if to_return.Err != nil {
		return nil, to_return.Err
	}
	return to_return.Build, nil
}
