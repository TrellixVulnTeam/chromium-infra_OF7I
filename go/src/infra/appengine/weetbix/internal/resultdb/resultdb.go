// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package resultdb contains logic of interacting with resultdb.
package resultdb

import (
	"context"
	"net/http"

	"google.golang.org/protobuf/proto"

	"go.chromium.org/luci/grpc/prpc"
	rdbpb "go.chromium.org/luci/resultdb/proto/v1"
	"go.chromium.org/luci/server/auth"
)

// mockResultDBClientKey is the context key indicates using mocked resultb client in tests.
var mockResultDBClientKey = "used in tests only for setting the mock resultdb client"

func newResultDBClient(ctx context.Context, host string) (rdbpb.ResultDBClient, error) {
	if mockClient, ok := ctx.Value(&mockResultDBClientKey).(*rdbpb.MockResultDBClient); ok {
		return mockClient, nil
	}

	t, err := auth.GetRPCTransport(ctx, auth.AsSelf)
	if err != nil {
		return nil, err
	}
	return rdbpb.NewResultDBPRPCClient(
		&prpc.Client{
			C:       &http.Client{Transport: t},
			Host:    host,
			Options: prpc.DefaultOptions(),
		}), nil
}

// Client is the client to communicate with ResultDB.
// It wraps a rdbpb.ResultDBClient.
type Client struct {
	client rdbpb.ResultDBClient
}

// NewClient creates a client to communicate with ResultDB.
func NewClient(ctx context.Context, host string) (*Client, error) {
	client, err := newResultDBClient(ctx, host)
	if err != nil {
		return nil, err
	}

	return &Client{
		client: client,
	}, nil
}

// QueryTestVariants queries test variants and advances the page automatically.
//
// f is called once per page of test variants.
func (c *Client) QueryTestVariants(ctx context.Context, req *rdbpb.QueryTestVariantsRequest, f func([]*rdbpb.TestVariant) error, maxPages int) error {
	// Copy the request to avoid aliasing issues when we update the page token.
	req = proto.Clone(req).(*rdbpb.QueryTestVariantsRequest)

	for page := 0; page < maxPages; page++ {
		rsp, err := c.client.QueryTestVariants(ctx, req)
		if err != nil {
			return err
		}

		if err = f(rsp.TestVariants); err != nil {
			return err
		}

		req.PageToken = rsp.GetNextPageToken()
		if req.PageToken == "" {
			// No more test variants.
			break
		}
	}

	return nil
}

// GetInvocation retrieves the invocation.
func (c *Client) GetInvocation(ctx context.Context, invName string) (*rdbpb.Invocation, error) {
	inv, err := c.client.GetInvocation(ctx, &rdbpb.GetInvocationRequest{
		Name: invName,
	})
	if err != nil {
		return nil, err
	}
	return inv, nil
}

// BatchGetTestVariants retrieves the requested test variants.
func (c *Client) BatchGetTestVariants(ctx context.Context, req *rdbpb.BatchGetTestVariantsRequest) ([]*rdbpb.TestVariant, error) {
	rsp, err := c.client.BatchGetTestVariants(ctx, req)
	if err != nil {
		return nil, err
	}
	return rsp.GetTestVariants(), nil
}
