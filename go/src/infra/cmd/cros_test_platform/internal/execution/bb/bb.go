// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package bb implements a skylab.Client using calls to BuildBucket.
package bb

import (
	"context"
	"infra/cmd/cros_test_platform/internal/execution/skylab"
	"infra/libs/skylab/request"

	"go.chromium.org/chromiumos/infra/proto/go/test_platform/config"
)

type bbSkylabClient struct {
}

// NewSkylabClient creates a new skylab.Client.
func NewSkylabClient(ctx context.Context, cfg *config.Config) (skylab.Client, error) {
	return &bbSkylabClient{}, nil
}

// ValidateArgs stub.
func (c *bbSkylabClient) ValidateArgs(ctx context.Context, args *request.Args) (bool, error) {
	panic("Not yet implemented.")
}

// LaunchTask stub.
func (c *bbSkylabClient) LaunchTask(ctx context.Context, args *request.Args) (skylab.TaskReference, error) {
	panic("Not yet implemented.")
}

// FetchResults stub.
func (c *bbSkylabClient) FetchResults(ctx context.Context, t skylab.TaskReference) (*skylab.FetchResultsResponse, error) {
	panic("Not yet implemented.")
}

// URL stub.
func (c *bbSkylabClient) URL(t skylab.TaskReference) string {
	panic("Not yet implemented.")
}

// SwarmingTaskID stub.
func (c *bbSkylabClient) SwarmingTaskID(t skylab.TaskReference) string {
	panic("Not yet implemented.")
}
