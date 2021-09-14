// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package buildbucket contains logic of interacting with Buildbucket.
package buildbucket

import (
	"context"
	"net/http"

	"google.golang.org/genproto/protobuf/field_mask"

	bbpb "go.chromium.org/luci/buildbucket/proto"
	"go.chromium.org/luci/grpc/prpc"
	"go.chromium.org/luci/server/auth"
)

var mockedBBClientKey = "used in tests only for setting the mock buildbucket client"

func newBuildsClient(ctx context.Context, host string) (bbpb.BuildsClient, error) {
	if mockClient, ok := ctx.Value(&mockedBBClientKey).(*bbpb.MockBuildsClient); ok {
		return mockClient, nil
	}

	t, err := auth.GetRPCTransport(ctx, auth.AsSelf)
	if err != nil {
		return nil, err
	}
	return bbpb.NewBuildsPRPCClient(
		&prpc.Client{
			C:    &http.Client{Transport: t},
			Host: host,
		}), nil
}

// Client is the client to communicate with Buildbucket.
// It wraps a bbpb.BuildsClient.
type Client struct {
	client bbpb.BuildsClient
}

// NewClient creates a client to communicate with Buildbucket.
func NewClient(ctx context.Context, host string) (*Client, error) {
	client, err := newBuildsClient(ctx, host)
	if err != nil {
		return nil, err
	}

	return &Client{
		client: client,
	}, nil
}

// GetResultDBInfo returns bbpb.Build which contains information about the resultdb
// invocation.
func (c *Client) GetResultDBInfo(ctx context.Context, bId int64) (*bbpb.Build, error) {
	return c.client.GetBuild(ctx, &bbpb.GetBuildRequest{
		Id: bId,
		Fields: &field_mask.FieldMask{
			Paths: []string{"infra.resultdb"},
		},
	}, nil)
}
