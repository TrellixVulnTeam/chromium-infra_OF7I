// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package cv contains logic of interacting with CV(Luci Test Verifier).
package cv

import (
	"context"
	"net/http"

	cvv0 "go.chromium.org/luci/cv/api/v0"
	"go.chromium.org/luci/grpc/prpc"
	"go.chromium.org/luci/server/auth"
)

// fakeCVClientKey is the context key indicates using fake CV client in tests.
var fakeCVClientKey = "used in tests only for setting the fake CV client"

func newRunsClient(ctx context.Context, host string) (cvv0.RunsClient, error) {
	if fc, ok := ctx.Value(&fakeCVClientKey).(*FakeClient); ok {
		return fc, nil
	}

	t, err := auth.GetRPCTransport(ctx, auth.AsSelf)
	if err != nil {
		return nil, err
	}
	return cvv0.NewRunsClient(
		&prpc.Client{
			C:       &http.Client{Transport: t},
			Host:    host,
			Options: prpc.DefaultOptions(),
		}), nil
}

// Client is the client to communicate with CV.
// It wraps a cvv0.RunsClient.
type Client struct {
	client cvv0.RunsClient
}

// NewClient creates a client to communicate with CV.
func NewClient(ctx context.Context, host string) (*Client, error) {
	client, err := newRunsClient(ctx, host)
	if err != nil {
		return nil, err
	}

	return &Client{
		client: client,
	}, nil
}

// GetRun returns cvv0.Run for the requested CV run.
func (c *Client) GetRun(ctx context.Context, req *cvv0.GetRunRequest) (*cvv0.Run, error) {
	return c.client.GetRun(ctx, req)
}
