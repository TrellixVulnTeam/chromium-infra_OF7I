// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package buildbucket contains logic of interacting with Buildbucket.
package buildbucket

import (
	"context"
	"net/http"

	bbpb "go.chromium.org/luci/buildbucket/proto"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/grpc/prpc"
	"go.chromium.org/luci/server/auth"
)

const (
	bbHost = "cr-buildbucket.appspot.com"
)

// mockedBBClientKey is the context key indicates using mocked buildbucket client in tests.
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
			C:       &http.Client{Transport: t},
			Host:    host,
			Options: prpc.DefaultOptions(),
		}), nil
}

// Client is the client to communicate with Buildbucket.
// It wraps a bbpb.BuildsClient.
type Client struct {
	Client bbpb.BuildsClient
}

// NewClient creates a client to communicate with Buildbucket.
func NewClient(ctx context.Context, host string) (*Client, error) {
	client, err := newBuildsClient(ctx, host)
	if err != nil {
		return nil, err
	}

	return &Client{
		Client: client,
	}, nil
}

// GetBuild returns bbpb.Build for the requested build.
func (c *Client) GetBuild(ctx context.Context, req *bbpb.GetBuildRequest) (*bbpb.Build, error) {
	return c.Client.GetBuild(ctx, req)
}

func GetBuild(c context.Context, bbid int64) (*bbpb.Build, error) {
	// For now, just query all fields
	q := &bbpb.GetBuildRequest{
		Id: bbid,
	}

	cl, err := NewClient(c, bbHost)
	if err != nil {
		logging.Errorf(c, "Cannot create Buildbucket client")
	}
	return cl.GetBuild(c, q)
}
