// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package resultdb contains logic of interacting with resultdb.
package resultdb

import (
	"context"
	"net/http"

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

// QueryTestVariants queries test variants with any unexpected results.
//
// f is called once per page of test variants.
func (c *Client) QueryTestVariants(ctx context.Context, invName string, f func([]*rdbpb.TestVariant) error) error {
	pageToken := ""

	for {
		rsp, err := c.client.QueryTestVariants(ctx, &rdbpb.QueryTestVariantsRequest{
			Invocations: []string{invName},
			Predicate: &rdbpb.TestVariantPredicate{
				Status: rdbpb.TestVariantStatus_UNEXPECTED_MASK,
			},
			PageSize:  1000, // Maximum page size.
			PageToken: pageToken,
		})
		if err != nil {
			return err
		}

		if err = f(rsp.TestVariants); err != nil {
			return err
		}

		pageToken = rsp.GetNextPageToken()
		if pageToken == "" {
			// No more test variants with unexpected results.
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
