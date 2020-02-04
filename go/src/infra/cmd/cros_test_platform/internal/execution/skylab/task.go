// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package skylab

import (
	"context"
	"infra/cmd/cros_test_platform/internal/execution/swarming"
	"infra/libs/skylab/request"

	"go.chromium.org/chromiumos/infra/proto/go/test_platform"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/skylab_test_runner"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/steps"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/swarming/proto/jsonrpc"
)

type task struct {
	args  request.Args
	ID    string
	url   string
	state jsonrpc.TaskState
	// Note: If we ever begin supporting other harnesses's result formats
	// then this field will change to a *skylab_test_runner.Result.
	// For now, the autotest-specific variant is more convenient.
	autotestResult *skylab_test_runner.Result_Autotest
}

func (t *task) Name() string {
	return t.args.Cmd.TaskName
}

func (t *task) Launch(ctx context.Context, clients Clients) error {
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
func (t *task) Completed() bool {
	return t.autotestResult != nil
}

func (t *task) Verdict() test_platform.TaskState_Verdict {
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
func (t *task) FetchResults(ctx context.Context, clients Clients) error {
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

var liftTestCaseRunnerVerdict = map[skylab_test_runner.Result_Autotest_TestCase_Verdict]test_platform.TaskState_Verdict{
	skylab_test_runner.Result_Autotest_TestCase_VERDICT_PASS: test_platform.TaskState_VERDICT_PASSED,
	skylab_test_runner.Result_Autotest_TestCase_VERDICT_FAIL: test_platform.TaskState_VERDICT_FAILED,
}

func (t *task) TestCases() []*steps.ExecuteResponse_TaskResult_TestCaseResult {
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

func (t *task) URL() string {
	return t.url
}
