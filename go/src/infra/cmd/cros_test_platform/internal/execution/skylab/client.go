// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package skylab

import (
	"context"

	"go.chromium.org/chromiumos/infra/proto/go/test_platform"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/skylab_test_runner"

	"infra/libs/skylab/request"
)

// TaskReference is an implementation-independent way to identify Skylab tasks.
type TaskReference interface {
	FetchResults(context.Context) (*FetchResultsResponse, error)
	SwarmingTaskID() string
	URL() string
}

// FetchResultsResponse is an implementation-independent container for
// information about running and finished tasks.
type FetchResultsResponse struct {
	Result    *skylab_test_runner.Result
	LifeCycle test_platform.TaskState_LifeCycle
}

// Client defines an interface used to interact with the Skylab
// infrastructure.
type Client interface {
	ValidateArgs(context.Context, *request.Args) (bool, error)
	LaunchTask(context.Context, *request.Args) (TaskReference, error)
}
