// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package response

import (
	"infra/cmd/cros_test_platform/internal/execution/testrunner"

	"go.chromium.org/chromiumos/infra/proto/go/test_platform"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/steps"
)

// Invocation accumulates the response for a single invocation
type Invocation struct {
	Name             string
	runnable         bool
	rejectedTaskDims map[string]string
	tasks            []*testrunner.Build
}

// NewInvocation returns a new Invocation with the given name.
func NewInvocation(name string) *Invocation {
	return &Invocation{runnable: true, Name: name}
}

// NotifyTask notifies the test task set of a new task for the test.
func (t *Invocation) NotifyTask(task *testrunner.Build) {
	t.tasks = append(t.tasks, task)
}

// MarkNotRunnable marks this invocation as being unable to run.
//
// When the invocation is not runnable because of unsatisfiable dependencies,
// the rejected dimensions should be supplied as the rejectedTaskDims argument.
func (t *Invocation) MarkNotRunnable(rejectedTaskDims map[string]string) {
	t.runnable = false
	t.rejectedTaskDims = rejectedTaskDims
}

// Verdict returns the test_platform TaskState Verdict for this invocation.
func (t *Invocation) Verdict() test_platform.TaskState_Verdict {
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

// LifeCycle returns the test_platform TaskState LifeCycle for this invocation.
func (t *Invocation) LifeCycle() test_platform.TaskState_LifeCycle {
	as := t.taskResult()
	// A test result can have 0 attempts only if the test hasn't run *yet*.
	// This is not possible in practice because the first attempt for each test
	// is created at the beginning.
	if len(as) == 0 {
		return test_platform.TaskState_LIFE_CYCLE_ABORTED
	}
	a := as[len(as)-1]
	if int(a.GetState().LifeCycle)&int(test_platform.TaskState_LIFE_CYCLE_MASK_FINAL) == 0 {
		return test_platform.TaskState_LIFE_CYCLE_RUNNING
	}
	return test_platform.TaskState_LIFE_CYCLE_COMPLETED
}

func (t *Invocation) taskResult() []*steps.ExecuteResponse_TaskResult {
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

func (t *Invocation) getLatestTask() *testrunner.Build {
	if len(t.tasks) == 0 {
		return nil
	}
	return t.tasks[len(t.tasks)-1]
}
