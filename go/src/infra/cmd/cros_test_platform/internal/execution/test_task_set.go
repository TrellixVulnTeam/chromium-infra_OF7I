// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package execution

import (
	"context"
	"infra/cmd/cros_test_platform/internal/execution/args"
	"infra/cmd/cros_test_platform/internal/execution/skylab"
	"math"

	"go.chromium.org/chromiumos/infra/proto/go/test_platform"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/config"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/steps"
	"go.chromium.org/luci/common/logging"
)

// testTaskSet encapsulates the running state of the set of tasks for one test.
type testTaskSet struct {
	argsGenerator    *args.Generator
	Name             string
	maxAttempts      int
	runnable         bool
	rejectedTaskDims map[string]string
	tasks            []*skylab.Task
}

func newTestTaskSet(invocation *steps.EnumerationResponse_AutotestInvocation, params *test_platform.Request_Params, workerConfig *config.Config_SkylabWorker, tc *TaskSetConfig) (*testTaskSet, error) {
	t := testTaskSet{runnable: true, Name: invocation.GetTest().GetName()}
	t.argsGenerator = args.NewGenerator(invocation, params, workerConfig, tc.ParentTaskID, tc.RequestUID, tc.Deadline)
	t.maxAttempts = 1 + int(inferTestMaxRetries(invocation))
	return &t, nil
}

func inferTestMaxRetries(inv *steps.EnumerationResponse_AutotestInvocation) int32 {
	if !inv.GetTest().GetAllowRetries() {
		return 0
	}
	return maxInt32IfZero(inv.GetTest().GetMaxRetries())
}

func maxInt32IfZero(v int32) int32 {
	if v == 0 {
		return int32(math.MaxInt32)
	}
	return v
}

func (t *testTaskSet) AttemptsRemaining() int {
	r := t.maxAttempts - len(t.tasks)
	if r > 0 {
		return r
	}
	return 0
}

func (t *testTaskSet) AttemptedAtLeastOnce() bool {
	return len(t.tasks) > 0
}

func (t *testTaskSet) LaunchTask(ctx context.Context, c skylab.Client) error {
	if rejected, err := skylab.ValidateDependencies(ctx, c, t.argsGenerator); err != nil {
		if !skylab.InvalidDependencies.In(err) {
			return err
		}
		logging.Warningf(ctx, "Dependency validation failed for %s: %s.", t.Name, err)
		t.markNotRunnable(rejected)
		return nil
	}

	a, err := skylab.NewTask(ctx, c, t.argsGenerator)
	if err != nil {
		return err
	}
	t.tasks = append(t.tasks, a)
	return nil
}

// markNotRunnable marks this test run as being unable to run.
//
// In particular, this means that this test run is Completed().
func (t *testTaskSet) markNotRunnable(rejectedTaskDims map[string]string) {
	t.runnable = false
	t.rejectedTaskDims = rejectedTaskDims
}

// Completed determines whether we have completed a task for this test.
func (t *testTaskSet) Completed() bool {
	if !t.runnable {
		return true
	}
	a := t.GetLatestTask()
	return a != nil && a.Completed()
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

func (t *testTaskSet) GetLatestTask() *skylab.Task {
	if len(t.tasks) == 0 {
		return nil
	}
	return t.tasks[len(t.tasks)-1]
}
