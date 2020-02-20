// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package swarming defines an interface for interacting with swarming.
package swarming

import (
	"context"

	"go.chromium.org/chromiumos/infra/proto/go/test_platform"
	swarming_api "go.chromium.org/luci/common/api/swarming/swarming/v1"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/swarming/proto/jsonrpc"

	"infra/libs/skylab/swarming"
)

// Client defines an interface used to interact with a swarming service.
type Client interface {
	GetTaskURL(taskID string) string
	BotExists(context.Context, []*swarming_api.SwarmingRpcsStringPair) (bool, error)
	CreateTask(context.Context, *swarming_api.SwarmingRpcsNewTaskRequest) (*swarming_api.SwarmingRpcsTaskRequestMetadata, error)
	GetResults(ctx context.Context, IDs []string) ([]*swarming_api.SwarmingRpcsTaskResult, error)
}

// swarming.Client is the reference implementation of the Client interface.
var _ Client = &swarming.Client{}

var taskStateToLifeCycle = map[jsonrpc.TaskState]test_platform.TaskState_LifeCycle{
	jsonrpc.TaskState_BOT_DIED:    test_platform.TaskState_LIFE_CYCLE_ABORTED,
	jsonrpc.TaskState_CANCELED:    test_platform.TaskState_LIFE_CYCLE_CANCELLED,
	jsonrpc.TaskState_COMPLETED:   test_platform.TaskState_LIFE_CYCLE_COMPLETED,
	jsonrpc.TaskState_EXPIRED:     test_platform.TaskState_LIFE_CYCLE_CANCELLED,
	jsonrpc.TaskState_KILLED:      test_platform.TaskState_LIFE_CYCLE_ABORTED,
	jsonrpc.TaskState_NO_RESOURCE: test_platform.TaskState_LIFE_CYCLE_REJECTED,
	jsonrpc.TaskState_PENDING:     test_platform.TaskState_LIFE_CYCLE_PENDING,
	jsonrpc.TaskState_RUNNING:     test_platform.TaskState_LIFE_CYCLE_RUNNING,
	jsonrpc.TaskState_TIMED_OUT:   test_platform.TaskState_LIFE_CYCLE_ABORTED,
}

// AsLifeCycle converts the string swarming task state into enum representation.
func AsLifeCycle(state string) (test_platform.TaskState_LifeCycle, error) {
	val, ok := jsonrpc.TaskState_value[state]
	if !ok {
		return test_platform.TaskState_LIFE_CYCLE_UNSPECIFIED, errors.Reason("invalid task state %s", state).Err()
	}
	return taskStateToLifeCycle[jsonrpc.TaskState(val)], nil
}
