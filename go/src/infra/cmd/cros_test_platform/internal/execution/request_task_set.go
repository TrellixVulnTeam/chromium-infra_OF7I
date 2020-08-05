// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package execution

import (
	"context"
	"fmt"
	"infra/cmd/cros_test_platform/internal/execution/args"
	"infra/cmd/cros_test_platform/internal/execution/skylab"
	"time"

	"go.chromium.org/chromiumos/infra/proto/go/test_platform"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/config"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/steps"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
)

// invocationID is a unique identifier used to refer to the input invocations.
//
// invocationID is required because there is no natural unique name for
// invocations.
type invocationID string

// RequestTaskSet encapsulates the running state of the set of tasks for one
// cros_test_platform request.
type RequestTaskSet struct {
	// Unique names for invocations preserving the order of incoming arguments.
	// This is used to preserve the order in the response.
	invocationIDs []invocationID

	argsGenerators map[invocationID]*args.Generator
	testTaskSets   map[invocationID]*testTaskSet
	activeTasks    map[invocationID]*skylab.Task
	retryCounter   retryCounter

	launched bool
}

// TaskSetConfig is a wrapper for the parameters common to the testTaskSets.
type TaskSetConfig struct {
	ParentTaskID string
	RequestUID   string
	Deadline     time.Time
}

// NewRequestTaskSet creates a new RequestTaskSet.
func NewRequestTaskSet(tests []*steps.EnumerationResponse_AutotestInvocation, params *test_platform.Request_Params, workerConfig *config.Config_SkylabWorker, tc *TaskSetConfig) (*RequestTaskSet, error) {
	invocationIDs := make([]invocationID, len(tests))
	testTaskSets := make(map[invocationID]*testTaskSet)
	argsGenerators := make(map[invocationID]*args.Generator)
	tm := make(map[invocationID]*steps.EnumerationResponse_AutotestInvocation)
	for i, test := range tests {
		iid := newInvocationID(i, test)
		invocationIDs[i] = iid
		argsGenerators[iid] = args.NewGenerator(test, params, workerConfig, tc.ParentTaskID, tc.RequestUID, tc.Deadline)
		testTaskSets[iid] = newTestTaskSet(test.GetTest().GetName())
		tm[iid] = test
	}
	return &RequestTaskSet{
		argsGenerators: argsGenerators,
		invocationIDs:  invocationIDs,
		testTaskSets:   testTaskSets,
		activeTasks:    make(map[invocationID]*skylab.Task),
		retryCounter:   newRetryCounter(params, tm),
	}, nil
}

func newInvocationID(i int, test *steps.EnumerationResponse_AutotestInvocation) invocationID {
	return invocationID(fmt.Sprintf("%d_%s", i, test.GetTest().GetName()))
}

// completed returns true if all tasks for this request have completed.
func (r *RequestTaskSet) completed() bool {
	return r.launched && len(r.activeTasks) == 0
}

// LaunchTasks launches initial tasks for all the tests in this request.
func (r *RequestTaskSet) LaunchTasks(ctx context.Context, c skylab.Client) error {
	r.launched = true
	for _, iid := range r.invocationIDs {
		ts := r.getTestTaskSet(iid)
		ag := r.getArgsGenerator(iid)

		if rejected, err := skylab.ValidateDependencies(ctx, c, ag); err != nil {
			if !skylab.InvalidDependencies.In(err) {
				return err
			}
			logging.Warningf(ctx, "Dependency validation failed for %s: %s.", ts.Name, err)
			ts.MarkNotRunnable(rejected)
			continue
		}

		task, err := skylab.NewTask(ctx, c, ag)
		if err != nil {
			return err
		}
		ts.NotifyTask(task)
		r.activeTasks[iid] = task
	}
	return nil
}

func (r *RequestTaskSet) getTestTaskSet(iid invocationID) *testTaskSet {
	ts, ok := r.testTaskSets[iid]
	if !ok {
		panic(fmt.Sprintf("No test task set for invocation %s", iid))
	}
	return ts
}

func (r *RequestTaskSet) getArgsGenerator(iid invocationID) *args.Generator {
	ag, ok := r.argsGenerators[iid]
	if !ok {
		panic(fmt.Sprintf("No args.Generator for invocation %s", iid))
	}
	return ag
}

// CheckTasksAndRetry checks the status of currently running tasks for this
// request and retries failed tasks when allowed.
//
// Returns whether all tasks are complete (so future calls to this function are
// unnecessary)
func (r *RequestTaskSet) CheckTasksAndRetry(ctx context.Context, c skylab.Client) (bool, error) {
	completedTests := make([]invocationID, len(r.activeTasks))
	newTasks := make(map[invocationID]*skylab.Task)
	for iid, task := range r.activeTasks {
		rerr := task.Refresh(ctx, c)
		tr := task.Result()
		if rerr != nil {
			return false, errors.Annotate(rerr, "tick for task %s", tr.LogUrl).Err()
		}
		if !task.Completed() {
			continue
		}

		ts := r.getTestTaskSet(iid)
		logging.Infof(ctx, "Task %s (%s) completed with verdict %s", tr.LogUrl, ts.Name, tr.GetState().GetVerdict())

		// At this point, we've determined that latestTask finished, and we've
		// updated the testTaskSet with its result. We can remove it from our
		// attention set... as long as we don't have to retry.
		shouldRetry := needsRetry(task.Result()) && r.retryCounter.CanRetry(ctx, iid)
		if !shouldRetry {
			completedTests = append(completedTests, iid)
			continue
		}

		logging.Infof(ctx, "Retrying %s", ts.Name)
		nt, err := task.Retry(ctx, c)
		if err != nil {
			return false, errors.Annotate(err, "tick for task %s: retry test", tr.LogUrl).Err()
		}
		newTasks[iid] = nt
		ts.NotifyTask(nt)
		r.retryCounter.NotifyRetry(iid)
	}

	for _, iid := range completedTests {
		delete(r.activeTasks, iid)
	}
	for iid, task := range newTasks {
		r.activeTasks[iid] = task
	}
	return r.completed(), nil
}

// Response returns the current response for this test.
func (r *RequestTaskSet) Response() *steps.ExecuteResponse {
	resp := &steps.ExecuteResponse{
		TaskResults:         r.taskResults(),
		ConsolidatedResults: r.results(),
		State: &test_platform.TaskState{
			Verdict:   r.verdict(),
			LifeCycle: r.lifecycle(),
		},
	}
	return resp
}

func (r *RequestTaskSet) lifecycle() test_platform.TaskState_LifeCycle {
	aborted := false
	running := false
	for _, ts := range r.testTaskSets {
		if ts.LifeCycle() == test_platform.TaskState_LIFE_CYCLE_ABORTED {
			aborted = true
		}
		if ts.LifeCycle() == test_platform.TaskState_LIFE_CYCLE_RUNNING {
			running = true
		}
	}

	// Order matters here. We return the "worst" lifecycle found.
	if aborted {
		return test_platform.TaskState_LIFE_CYCLE_ABORTED
	}
	if running {
		return test_platform.TaskState_LIFE_CYCLE_RUNNING
	}
	return test_platform.TaskState_LIFE_CYCLE_COMPLETED
}

func (r *RequestTaskSet) verdict() test_platform.TaskState_Verdict {
	v := test_platform.TaskState_VERDICT_PASSED
	for _, t := range r.testTaskSets {
		if !successfulVerdict(t.Verdict()) {
			v = test_platform.TaskState_VERDICT_FAILED
			break
		}
	}
	return v
}

func successfulVerdict(v test_platform.TaskState_Verdict) bool {
	switch v {
	case test_platform.TaskState_VERDICT_PASSED,
		test_platform.TaskState_VERDICT_PASSED_ON_RETRY,
		test_platform.TaskState_VERDICT_NO_VERDICT:
		return true
	default:
		return false
	}
}

func (r *RequestTaskSet) results() []*steps.ExecuteResponse_ConsolidatedResult {
	rs := make([]*steps.ExecuteResponse_ConsolidatedResult, len(r.invocationIDs))
	for i, iid := range r.invocationIDs {
		rs[i] = &steps.ExecuteResponse_ConsolidatedResult{
			Attempts: r.getTestTaskSet(iid).TaskResult(),
		}
	}
	return rs
}

func (r *RequestTaskSet) taskResults() []*steps.ExecuteResponse_TaskResult {
	results := r.results()
	var trs []*steps.ExecuteResponse_TaskResult
	for _, result := range results {
		trs = append(trs, result.Attempts...)
	}
	return trs
}
