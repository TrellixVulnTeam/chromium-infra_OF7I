// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package swarming

import (
	"go.chromium.org/chromiumos/infra/proto/go/test_platform"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/swarming/proto/jsonrpc"
)

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

// asLifeCycle converts the string swarming task state into enum representation.
func asLifeCycle(state string) (test_platform.TaskState_LifeCycle, error) {
	val, ok := jsonrpc.TaskState_value[state]
	if !ok {
		return test_platform.TaskState_LIFE_CYCLE_UNSPECIFIED, errors.Reason("invalid task state %s", state).Err()
	}
	return taskStateToLifeCycle[jsonrpc.TaskState(val)], nil
}

// Tasks with these life cycles contain test results.
// E.g. this excludes killed tasks as they have no chance to produce results.
var lifeCyclesWithResults = map[test_platform.TaskState_LifeCycle]bool{
	test_platform.TaskState_LIFE_CYCLE_COMPLETED: true,
}
