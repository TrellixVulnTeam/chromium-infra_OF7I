// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package gitiles

import (
	"context"
	"fmt"
	"net/http"

	"go.chromium.org/luci/common/api/gitiles"
	gitilesProto "go.chromium.org/luci/common/proto/gitiles"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
)

// NewThrottlingClient creates REST Gitiles client and consumes limiter quota
// on each API call to Gitiles. If there is no quota left, it blocks until
// there is.
func NewThrottlingClient(host string, limiter *rate.Limiter) (Client, error) {
	c, err := gitiles.NewRESTClient(&http.Client{}, host, false)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", "couldn't initialize Gitiles REST client", err)
	}
	return c, err
}

type throttlingClient struct {
	limiter *rate.Limiter
	client  Client
}

// Log retrieves commit log.
func (t *throttlingClient) Log(ctx context.Context, in *gitilesProto.LogRequest, opts ...grpc.CallOption) (*gitilesProto.LogResponse, error) {
	if err := t.limiter.Wait(ctx); err != nil {
		return nil, err
	}
	return t.client.Log(ctx, in)
}

// Refs retrieves repo refs.
func (t *throttlingClient) Refs(ctx context.Context, in *gitilesProto.RefsRequest, opts ...grpc.CallOption) (*gitilesProto.RefsResponse, error) {
	if err := t.limiter.Wait(ctx); err != nil {
		return nil, err
	}
	return t.client.Refs(ctx, in)
}

// Archive retrieves archived contents of the project.
//
// An archive is a shallow bundle of the contents of a repository.
//
// DEPRECATED: Use DownloadFile to obtain plain text files.
// TODO(pprabhu): Migrate known users to DownloadFile and delete this RPC.
func (t *throttlingClient) Archive(ctx context.Context, in *gitilesProto.ArchiveRequest, opts ...grpc.CallOption) (*gitilesProto.ArchiveResponse, error) {
	if err := t.limiter.Wait(ctx); err != nil {
		return nil, err
	}
	return t.client.Archive(ctx, in)
}

// DownloadFile retrieves a file from the project.
func (t *throttlingClient) DownloadFile(ctx context.Context, in *gitilesProto.DownloadFileRequest, opts ...grpc.CallOption) (*gitilesProto.DownloadFileResponse, error) {
	if err := t.limiter.Wait(ctx); err != nil {
		return nil, err
	}
	return t.client.DownloadFile(ctx, in)
}

// Projects retrieves list of available Gitiles projects
func (t *throttlingClient) Projects(ctx context.Context, in *gitilesProto.ProjectsRequest, opts ...grpc.CallOption) (*gitilesProto.ProjectsResponse, error) {
	if err := t.limiter.Wait(ctx); err != nil {
		return nil, err
	}
	return t.client.Projects(ctx, in)
}
