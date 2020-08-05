// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package execution

import (
	"context"
	"infra/cmd/cros_test_platform/internal/execution/args"
	"infra/cmd/cros_test_platform/internal/execution/skylab"

	"go.chromium.org/chromiumos/infra/proto/go/test_platform"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/config"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/steps"
	"go.chromium.org/luci/common/logging"
)

// testTaskSet encapsulates the running state of the set of tasks for one test.
type testTaskSet struct {
	argsGenerator    *args.Generator
	Name             string
	runnable         bool
	rejectedTaskDims map[string]string
	tasks            []*skylab.Task
}

func newTestTaskSet(invocation *steps.EnumerationResponse_AutotestInvocation, params *test_platform.Request_Params, workerConfig *config.Config_SkylabWorker, tc *TaskSetConfig) (*testTaskSet, error) {
	t := testTaskSet{runnable: true, Name: invocation.GetTest().GetName()}
	t.argsGenerator = args.NewGenerator(invocation, params, workerConfig, tc.ParentTaskID, tc.RequestUID, tc.Deadline)
	return &t, nil
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
	t.NotifyTask(a)
	return nil
}

// NotifyTask notifies the test task set of a new task for the test.
func (t *testTaskSet) NotifyTask(task *skylab.Task) {
	t.tasks = append(t.tasks, task)
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
