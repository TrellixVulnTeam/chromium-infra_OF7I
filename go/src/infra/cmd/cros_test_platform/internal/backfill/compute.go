// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package backfill

import (
	"github.com/golang/protobuf/proto"

	"go.chromium.org/chromiumos/infra/proto/go/test_platform"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/steps"
	"go.chromium.org/luci/common/errors"
)

// Compute computes the backfill response for a single request and execution.
func Compute(r *steps.ComputeBackfillRequest) (*steps.ComputeBackfillResponse, error) {
	tp, err := computePlan(r.GetEnumeration(), r.GetExecution())
	if err != nil {
		return nil, err
	}
	if tp == nil {
		return &steps.ComputeBackfillResponse{}, nil
	}
	return &steps.ComputeBackfillResponse{
		Request: requestWithTestPlan(r.GetRequest(), tp),
	}, nil
}

func requestWithTestPlan(r *test_platform.Request, tp *test_platform.Request_TestPlan) *test_platform.Request {
	br := proto.Clone(r).(*test_platform.Request)
	br.TestPlan = tp
	return br
}

func computePlan(enumeration *steps.EnumerationResponse, execution *steps.ExecuteResponse) (*test_platform.Request_TestPlan, error) {
	if len(enumeration.GetAutotestInvocations()) != len(execution.GetConsolidatedResults()) {
		return nil, errors.Reason("unequal test enumeration and result counts: %d vs %d",
			len(enumeration.GetAutotestInvocations()), len(execution.GetConsolidatedResults())).Err()
	}

	var rinvs []*test_platform.Request_Enumeration_AutotestInvocation
	for i := range enumeration.GetAutotestInvocations() {
		if needsBackfill(execution.ConsolidatedResults[i]) {
			rinvs = append(rinvs, requestInvocationFromEnumeration(enumeration.AutotestInvocations[i]))
		}
	}

	if len(rinvs) == 0 {
		return nil, nil
	}
	return &test_platform.Request_TestPlan{
		Enumeration: &test_platform.Request_Enumeration{
			AutotestInvocations: rinvs,
		},
	}, nil
}

func needsBackfill(r *steps.ExecuteResponse_ConsolidatedResult) bool {
	for _, a := range r.Attempts {
		if isSuccessfulAttempt(a) {
			return false
		}
	}
	return true
}

func isSuccessfulAttempt(a *steps.ExecuteResponse_TaskResult) bool {
	if a.State.LifeCycle != test_platform.TaskState_LIFE_CYCLE_COMPLETED {
		return false
	}
	switch a.State.Verdict {
	case test_platform.TaskState_VERDICT_PASSED,
		test_platform.TaskState_VERDICT_PASSED_ON_RETRY,
		test_platform.TaskState_VERDICT_NO_VERDICT:
		return true
	default:
		return false
	}
}

func requestInvocationFromEnumeration(inv *steps.EnumerationResponse_AutotestInvocation) *test_platform.Request_Enumeration_AutotestInvocation {
	return &test_platform.Request_Enumeration_AutotestInvocation{
		Test:          inv.GetTest(),
		TestArgs:      inv.GetTestArgs(),
		DisplayName:   inv.GetDisplayName(),
		ResultKeyvals: inv.GetResultKeyvals(),
	}
}
