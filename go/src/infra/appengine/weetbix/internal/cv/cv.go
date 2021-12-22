// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package cv contains logic of interacting with CV (LUCI Change Verifier).
package cv

import (
	"context"
	"net/http"

	"google.golang.org/grpc"

	cvv0 "go.chromium.org/luci/cv/api/v0"
	"go.chromium.org/luci/grpc/prpc"
	"go.chromium.org/luci/server/auth"
)

// fakeCVClientKey is the context key indicates using fake CV client in tests.
var fakeCVClientKey = "used in tests only for setting the fake CV client"

func newRunsClient(ctx context.Context, host string) (Client, error) {
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

// Client defines a subset of CV API consumed by Weebtix.
type Client interface {
	GetRun(ctx context.Context, in *cvv0.GetRunRequest, opts ...grpc.CallOption) (*cvv0.Run, error)
}

// ensure Client is a subset of CV interface.
var _ Client = (cvv0.RunsClient)(nil)

// NewClient creates a client to communicate with CV.
func NewClient(ctx context.Context, host string) (Client, error) {
	client, err := newRunsClient(ctx, host)
	if err != nil {
		return nil, err
	}

	return client, nil
}
