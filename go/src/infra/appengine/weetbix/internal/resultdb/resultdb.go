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

// expectedTestVariantsPageToken is the next page token for expected test variants.
// See https://source.chromium.org/chromium/_/chromium/infra/luci/luci-go/+/bcfdf0380e026668674c1f1cae919e1b2ace8ed3:resultdb/internal/testvariants/query.go;l=353;drc=2a22b94e783feb1c02109d8b6f7cf93a4a4b69f6
const expectedTestVariantsPageToken = "CghFWFBFQ1RFRAoACgA="

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
			C:    &http.Client{Transport: t},
			Host: host,
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
func (c *Client) QueryTestVariants(ctx context.Context, invName string) (tvs []*rdbpb.TestVariant, err error) {
	pageToken := ""

	for {
		rsp, err := c.client.QueryTestVariants(ctx, &rdbpb.QueryTestVariantsRequest{
			Invocations: []string{invName},
			PageSize:    1000, // Maximum page size.
			PageToken:   pageToken,
		}, nil)
		if err != nil {
			return tvs, err
		}

		tvs = append(tvs, rsp.TestVariants...)
		pageToken = rsp.GetNextPageToken()
		// QueryTestVariants always returns expected test variants in a new page.
		//
		// We only care about test variants with any unexpected result, so we can stop
		// if the next page is for expected test variants.
		// TODO(crbug.com/1249596): Update the request to query test variants with
		// any unexpected results only.
		if pageToken == expectedTestVariantsPageToken || pageToken == "" {
			// No more test variants with unexpected results.
			break
		}
	}

	return tvs, nil
}

// GetInvocation retrieves the invocation.
func (c *Client) GetInvocation(ctx context.Context, invName string) (*rdbpb.Invocation, error) {
	inv, err := c.client.GetInvocation(ctx, &rdbpb.GetInvocationRequest{
		Name: invName,
	}, nil)
	if err != nil {
		return nil, err
	}
	return inv, nil
}

// RealmFromInvocation retrieves the realm of the invocation.
func (c *Client) RealmFromInvocation(ctx context.Context, invName string) (string, error) {
	inv, err := c.GetInvocation(ctx, invName)
	if err != nil {
		return "", err
	}
	return inv.GetRealm(), nil
}
