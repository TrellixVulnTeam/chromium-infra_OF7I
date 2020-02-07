// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package attempt contains the logic for running individual test tasks.
package attempt

import (
	"bytes"
	"context"
	"fmt"
	"infra/cmd/cros_test_platform/internal/execution/isolate"
	"infra/cmd/cros_test_platform/internal/execution/swarming"
	"infra/libs/skylab/request"

	"github.com/golang/protobuf/jsonpb"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/common"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/skylab_test_runner"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/steps"
	swarming_api "go.chromium.org/luci/common/api/swarming/swarming/v1"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/isolated"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/swarming/proto/jsonrpc"
)

// Clients bundles local interfaces to various remote services used by Runner.
type Clients struct {
	Swarming      swarming.Client
	IsolateGetter isolate.GetterFactory
}

// Task represents an individual test task.
type Task struct {
	args  request.Args
	ID    string
	url   string
	state jsonrpc.TaskState
	// Note: If we ever begin supporting other harnesses's result formats
	// then this field will change to a *skylab_test_runner.Result.
	// For now, the autotest-specific variant is more convenient.
	autotestResult *skylab_test_runner.Result_Autotest
}

// NewTask initializes a Task object.
func NewTask(args request.Args) *Task {
	return &Task{args: args}
}

// Name is the task name as it is displayed in the UI.
func (t *Task) Name() string {
	return t.args.Cmd.TaskName
}

// Launch sends an RPC request to start the task.
func (t *Task) Launch(ctx context.Context, clients Clients) error {
	req, err := t.args.SwarmingNewTaskRequest()
	if err != nil {
		return errors.Annotate(err, "launch attempt for %s", t.Name()).Err()
	}
	resp, err := clients.Swarming.CreateTask(ctx, req)
	if err != nil {
		return errors.Annotate(err, "launch attempt for %s", t.Name()).Err()
	}
	t.ID = resp.TaskId
	t.url = clients.Swarming.GetTaskURL(t.ID)
	logging.Infof(ctx, "Launched attempt for %s as task %s", t.Name(), t.url)
	return nil
}

// Completed returns whether the current task is complete.
func (t *Task) Completed() bool {
	return t.autotestResult != nil
}

// Verdict aggregates the information about test cases contained in a task into
// a single verdict.
func (t *Task) Verdict() test_platform.TaskState_Verdict {
	if !t.Completed() {
		return test_platform.TaskState_VERDICT_UNSPECIFIED
	}
	if t.autotestResult == nil {
		return test_platform.TaskState_VERDICT_UNSPECIFIED
	}
	if t.autotestResult.Incomplete {
		return test_platform.TaskState_VERDICT_FAILED
	}

	// By default (if no test cases ran), then there is no verdict.
	verdict := test_platform.TaskState_VERDICT_NO_VERDICT
	for _, c := range t.autotestResult.GetTestCases() {
		switch c.Verdict {
		case skylab_test_runner.Result_Autotest_TestCase_VERDICT_FAIL:
			// Any case failing means the flat verdict is a failure.
			return test_platform.TaskState_VERDICT_FAILED
		case skylab_test_runner.Result_Autotest_TestCase_VERDICT_PASS:
			// Otherwise, at least 1 passing verdict means a pass.
			verdict = test_platform.TaskState_VERDICT_PASSED
		case skylab_test_runner.Result_Autotest_TestCase_VERDICT_UNDEFINED:
			// Undefined verdicts do not affect flat verdict.
		}
	}
	return verdict
}

// FetchResults fetches the latest swarming and isolate state of the given task,
// and updates the task accordingly.
func (t *Task) FetchResults(ctx context.Context, clients Clients) error {
	results, err := clients.Swarming.GetResults(ctx, []string{t.ID})
	if err != nil {
		return errors.Annotate(err, "fetch results").Err()
	}
	result, err := unpackResult(results, t.ID)
	if err != nil {
		return errors.Annotate(err, "fetch results").Err()
	}
	state, err := swarming.AsTaskState(result.State)
	if err != nil {
		return errors.Annotate(err, "fetch results").Err()
	}
	t.state = state

	switch {
	// Task ran to completion.
	case swarming.CompletedTaskStates[state]:
		r, err := getAutotestResult(ctx, result, clients.IsolateGetter)
		if err != nil {
			logging.Debugf(ctx, "failed to fetch autotest results for task %s due to error '%s', treating its results as incomplete (failure)", t.ID, err.Error())
			r = &skylab_test_runner.Result_Autotest{Incomplete: true}
		}
		t.autotestResult = r
	// Task no longer running, but didn't run to completion.
	case !swarming.UnfinishedTaskStates[state]:
		t.autotestResult = &skylab_test_runner.Result_Autotest{Incomplete: true}
	// Task is still running.
	default:
	}
	return nil
}

func unpackResult(results []*swarming_api.SwarmingRpcsTaskResult, taskID string) (*swarming_api.SwarmingRpcsTaskResult, error) {
	if len(results) != 1 {
		return nil, errors.Reason("expected 1 result for task id %s, got %d", taskID, len(results)).Err()
	}

	result := results[0]
	if result.TaskId != taskID {
		return nil, errors.Reason("expected result for task id %s, got %s", taskID, result.TaskId).Err()
	}

	return result, nil
}

var liftTestCaseRunnerVerdict = map[skylab_test_runner.Result_Autotest_TestCase_Verdict]test_platform.TaskState_Verdict{
	skylab_test_runner.Result_Autotest_TestCase_VERDICT_PASS: test_platform.TaskState_VERDICT_PASSED,
	skylab_test_runner.Result_Autotest_TestCase_VERDICT_FAIL: test_platform.TaskState_VERDICT_FAILED,
}

// TestCases unpacks test cases contained in the results of a task.
func (t *Task) TestCases() []*steps.ExecuteResponse_TaskResult_TestCaseResult {
	tcs := t.autotestResult.GetTestCases()
	if len(tcs) == 0 {
		// Prefer a nil over an empty slice since it's the proto default.
		return nil
	}
	ret := make([]*steps.ExecuteResponse_TaskResult_TestCaseResult, len(tcs))
	for i, tc := range tcs {
		ret[i] = &steps.ExecuteResponse_TaskResult_TestCaseResult{
			Name:                 tc.Name,
			Verdict:              liftTestCaseRunnerVerdict[tc.Verdict],
			HumanReadableSummary: tc.HumanReadableSummary,
		}
	}
	return ret
}

// URL return the URL of the task page.
func (t *Task) URL() string {
	return t.url
}

const resultsFileName = "results.json"

func getAutotestResult(ctx context.Context, sResult *swarming_api.SwarmingRpcsTaskResult, gf isolate.GetterFactory) (*skylab_test_runner.Result_Autotest, error) {
	if sResult == nil {
		return nil, errors.Reason("get result: nil swarming result").Err()
	}

	taskID := sResult.TaskId
	outputRef := sResult.OutputsRef
	if outputRef == nil {
		logging.Debugf(ctx, "task %s has no output ref, considering it failed due to incompleteness", taskID)
		return &skylab_test_runner.Result_Autotest{Incomplete: true}, nil
	}

	getter, err := gf(ctx, outputRef.Isolatedserver)
	if err != nil {
		return nil, errors.Annotate(err, "get result").Err()
	}

	logging.Debugf(ctx, "fetching result for task %s from isolate ref %+v", taskID, outputRef)
	content, err := getter.GetFile(ctx, isolated.HexDigest(outputRef.Isolated), resultsFileName)
	if err != nil {
		return nil, errors.Annotate(err, "get result for task %s", taskID).Err()
	}

	var result skylab_test_runner.Result

	err = jsonpb.Unmarshal(bytes.NewReader(content), &result)
	if err != nil {
		return nil, errors.Annotate(err, "get result for task %s", taskID).Err()
	}

	a := result.GetAutotestResult()
	if a == nil {
		return nil, errors.Reason("get result for task %s: no autotest result; other harnesses not yet supported", taskID).Err()
	}

	return a, nil
}

var taskStateToLifeCycle = map[jsonrpc.TaskState]test_platform.TaskState_LifeCycle{
	jsonrpc.TaskState_BOT_DIED:  test_platform.TaskState_LIFE_CYCLE_ABORTED,
	jsonrpc.TaskState_CANCELED:  test_platform.TaskState_LIFE_CYCLE_CANCELLED,
	jsonrpc.TaskState_COMPLETED: test_platform.TaskState_LIFE_CYCLE_COMPLETED,
	// TODO(akeshet): This mapping is inexact. Add a lifecycle entry for this.
	jsonrpc.TaskState_EXPIRED:     test_platform.TaskState_LIFE_CYCLE_CANCELLED,
	jsonrpc.TaskState_KILLED:      test_platform.TaskState_LIFE_CYCLE_ABORTED,
	jsonrpc.TaskState_NO_RESOURCE: test_platform.TaskState_LIFE_CYCLE_REJECTED,
	jsonrpc.TaskState_PENDING:     test_platform.TaskState_LIFE_CYCLE_PENDING,
	jsonrpc.TaskState_RUNNING:     test_platform.TaskState_LIFE_CYCLE_RUNNING,
	// TODO(akeshet): This mapping is inexact. Add a lifecycle entry for this.
	jsonrpc.TaskState_TIMED_OUT: test_platform.TaskState_LIFE_CYCLE_ABORTED,
}

// Result constructs a TaskResults out of the data already contained in the
// Task object. In order to get the latest result, FetchResult needs to be
// called first.
func (t *Task) Result(attemptNum int) *steps.ExecuteResponse_TaskResult {
	logURL := fmt.Sprintf(
		"https://stainless.corp.google.com/browse/chromeos-autotest-results/swarming-%s/",
		t.ID,
	)
	gsURL := fmt.Sprintf(
		"gs://chromeos-autotest-results/swarming-%s/",
		t.ID,
	)

	return &steps.ExecuteResponse_TaskResult{
		Name: t.Name(),
		State: &test_platform.TaskState{
			LifeCycle: taskStateToLifeCycle[t.state],
			Verdict:   t.Verdict(),
		},
		TaskUrl: t.URL(),
		LogUrl:  logURL,
		LogData: &common.TaskLogData{
			GsUrl: gsURL,
		},
		Attempt:   int32(attemptNum),
		TestCases: t.TestCases(),
	}
}
