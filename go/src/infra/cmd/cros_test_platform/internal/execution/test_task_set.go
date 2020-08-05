// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package execution

import (
	"infra/cmd/cros_test_platform/internal/execution/skylab"

	"go.chromium.org/chromiumos/infra/proto/go/test_platform"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/steps"
)

// testTaskSet encapsulates the running state of the set of tasks for one test.
type testTaskSet struct {
	Name             string
	runnable         bool
	rejectedTaskDims map[string]string
	tasks            []*skylab.Task
}

func newTestTaskSet(name string) *testTaskSet {
	return &testTaskSet{runnable: true, Name: name}
}

// NotifyTask notifies the test task set of a new task for the test.
func (t *testTaskSet) NotifyTask(task *skylab.Task) {
	t.tasks = append(t.tasks, task)
}

// MarkNotRunnable marks this test run as being unable to run.
//
// In particular, this means that this test run is Completed().
func (t *testTaskSet) MarkNotRunnable(rejectedTaskDims map[string]string) {
	t.runnable = false
	t.rejectedTaskDims = rejectedTaskDims
}

func (t *testTaskSet) TaskResult() []*steps.ExecuteResponse_TaskResult {
	if !t.runnable {
		return []*steps.ExecuteResponse_TaskResult{
			{
				Name: t.Name,
				State: &test_platform.TaskState{
					LifeCycle: test_platform.TaskState_LIFE_CYCLE_REJECTED,
					Verdict:   test_platform.TaskState_VERDICT_UNSPECIFIED,
				},
				RejectedTaskDimensions: t.rejectedTaskDims,
			},
		}
	}

	ret := make([]*steps.ExecuteResponse_TaskResult, len(t.tasks))
	for i, a := range t.tasks {
		ret[i] = a.Result()
		ret[i].Attempt = int32(i)
	}
	return ret
}

func (t *testTaskSet) Verdict() test_platform.TaskState_Verdict {
	if !t.runnable {
		return test_platform.TaskState_VERDICT_UNSPECIFIED
	}
	failedEarlierTask := false
	for _, a := range t.tasks {
		switch a.Result().GetState().GetVerdict() {
		case test_platform.TaskState_VERDICT_NO_VERDICT:
			return test_platform.TaskState_VERDICT_NO_VERDICT
		case test_platform.TaskState_VERDICT_PASSED:
			if failedEarlierTask {
				return test_platform.TaskState_VERDICT_PASSED_ON_RETRY
			}
			return test_platform.TaskState_VERDICT_PASSED
		case test_platform.TaskState_VERDICT_FAILED,
			test_platform.TaskState_VERDICT_UNSPECIFIED:
			failedEarlierTask = true
		default:
			return test_platform.TaskState_VERDICT_FAILED
		}
	}
	return test_platform.TaskState_VERDICT_FAILED
}

func (t *testTaskSet) getLatestTask() *skylab.Task {
	if len(t.tasks) == 0 {
		return nil
	}
	return t.tasks[len(t.tasks)-1]
}
