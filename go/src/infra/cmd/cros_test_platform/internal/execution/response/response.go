// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package response provides a way to accumulate and summarize the
// response for a request.
package response

import (
	"go.chromium.org/chromiumos/infra/proto/go/test_platform"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/steps"
)

// Summarize summarizes the response from a list of Invocations.
func Summarize(tss []*Invocation) *steps.ExecuteResponse {
	resp := &steps.ExecuteResponse{
		TaskResults:         taskResults(tss),
		ConsolidatedResults: results(tss),
		State: &test_platform.TaskState{
			Verdict:   verdict(tss),
			LifeCycle: lifecycle(tss),
		},
	}
	return resp
}

func lifecycle(tss []*Invocation) test_platform.TaskState_LifeCycle {
	aborted := false
	running := false
	for _, ts := range tss {
		if ts.lifeCycle() == test_platform.TaskState_LIFE_CYCLE_ABORTED {
			aborted = true
		}
		if ts.lifeCycle() == test_platform.TaskState_LIFE_CYCLE_RUNNING {
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

func verdict(tss []*Invocation) test_platform.TaskState_Verdict {
	v := test_platform.TaskState_VERDICT_PASSED
	for _, t := range tss {
		if !successfulVerdict(t.verdict()) {
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

func results(tss []*Invocation) []*steps.ExecuteResponse_ConsolidatedResult {
	rs := make([]*steps.ExecuteResponse_ConsolidatedResult, len(tss))
	for i, ts := range tss {
		rs[i] = &steps.ExecuteResponse_ConsolidatedResult{
			Attempts: ts.taskResult(),
		}
	}
	return rs
}

func taskResults(tss []*Invocation) []*steps.ExecuteResponse_TaskResult {
	results := results(tss)
	var trs []*steps.ExecuteResponse_TaskResult
	for _, result := range results {
		trs = append(trs, result.Attempts...)
	}
	return trs
}
