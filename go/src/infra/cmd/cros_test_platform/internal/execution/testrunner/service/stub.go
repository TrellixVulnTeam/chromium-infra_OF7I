// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package service

import (
	"context"
	"infra/libs/skylab/request"

	"go.chromium.org/chromiumos/infra/proto/go/test_platform"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/skylab_test_runner"
)

// StubClient implements a noop Client with "reasonable" default behavior for
// all methods.
type StubClient struct{}

// Ensure we implement the promised interface.
var _ Client = StubClient{}

// ValidateArgs implements Client interface.
func (c StubClient) ValidateArgs(context.Context, *request.Args) (bool, map[string]string, error) {
	return true, nil, nil
}

// LaunchTask implements Client interface.
func (c StubClient) LaunchTask(context.Context, *request.Args) (TaskReference, error) {
	return "stub-task-reference", nil
}

// FetchResults implements Client interface.
func (c StubClient) FetchResults(context.Context, TaskReference) (*FetchResultsResponse, error) {
	return &FetchResultsResponse{
		Result:    &skylab_test_runner.Result{},
		LifeCycle: test_platform.TaskState_LIFE_CYCLE_COMPLETED,
	}, nil
}

// SwarmingTaskID implements Client interface.
func (c StubClient) SwarmingTaskID(TaskReference) string {
	return "stub-swarming-task-id"
}

// URL implements Client interface.
func (c StubClient) URL(TaskReference) string {
	return "stub-url"
}
