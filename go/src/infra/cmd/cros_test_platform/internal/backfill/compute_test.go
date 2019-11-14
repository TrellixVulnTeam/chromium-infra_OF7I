// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package backfill

import (
	"testing"

	"github.com/kylelemons/godebug/pretty"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/steps"
)

func TestPlanComputationForFailedRun(t *testing.T) {
	cases := []struct {
		Tag         string
		Enumeration *steps.EnumerationResponse
		Execution   *steps.ExecuteResponse
		Want        *test_platform.Request
	}{
		{
			Tag:  "no enumeration or execution",
			Want: nil,
		},
		{
			Tag:         "one enumeration with zero value for task state",
			Enumeration: enumerationsWithDisplayNames("some_test"),
			Execution: executeResponseWithSingleAttempt(
				test_platform.TaskState_LIFE_CYCLE_UNSPECIFIED,
				test_platform.TaskState_VERDICT_UNSPECIFIED,
			),
			Want: requestWithEnumeration(cannedParams(), enumerationsWithDisplayNames("some_test")),
		},
		{
			Tag:         "one enumeration with successful execution",
			Enumeration: enumerationsWithDisplayNames("some_test"),
			Execution: executeResponseWithSingleAttempt(
				test_platform.TaskState_LIFE_CYCLE_COMPLETED,
				test_platform.TaskState_VERDICT_PASSED,
			),
			Want: nil,
		},
		{
			Tag:         "one enumeration with successful execution",
			Enumeration: enumerationsWithDisplayNames("some_test"),
			Execution: executeResponseWithSingleAttempt(
				test_platform.TaskState_LIFE_CYCLE_COMPLETED,
				test_platform.TaskState_VERDICT_PASSED_ON_RETRY,
			),
			Want: nil,
		},
		{
			Tag:         "one enumeration without verdict",
			Enumeration: enumerationsWithDisplayNames("some_test"),
			Execution: executeResponseWithSingleAttempt(
				test_platform.TaskState_LIFE_CYCLE_COMPLETED,
				test_platform.TaskState_VERDICT_NO_VERDICT,
			),
			// No verdict implies there was no test case that could be run on
			// the chosen DUT. The test did correctly get scheduled from test
			// platform perspective and succeeded without any verdict.
			Want: nil,
		},
		{
			Tag:         "one enumeration with failed execution",
			Enumeration: enumerationsWithDisplayNames("some_test"),
			Execution: executeResponseWithSingleAttempt(
				test_platform.TaskState_LIFE_CYCLE_COMPLETED,
				test_platform.TaskState_VERDICT_FAILED,
			),
			Want: requestWithEnumeration(cannedParams(), enumerationsWithDisplayNames("some_test")),
		},
		{
			Tag:         "one enumeration with lifecycle pending",
			Enumeration: enumerationsWithDisplayNames("some_test"),
			Execution: executeResponseWithSingleAttempt(
				test_platform.TaskState_LIFE_CYCLE_PENDING,
				test_platform.TaskState_VERDICT_PASSED,
			),
			Want: requestWithEnumeration(cannedParams(), enumerationsWithDisplayNames("some_test")),
		},
		{
			Tag:         "one enumeration with aborted task",
			Enumeration: enumerationsWithDisplayNames("some_test"),
			Execution: executeResponseWithSingleAttempt(
				test_platform.TaskState_LIFE_CYCLE_ABORTED,
				test_platform.TaskState_VERDICT_UNSPECIFIED,
			),
			Want: requestWithEnumeration(cannedParams(), enumerationsWithDisplayNames("some_test")),
		},
		{
			Tag:         "one enumeration with rejected task",
			Enumeration: enumerationsWithDisplayNames("some_test"),
			Execution: executeResponseWithSingleAttempt(
				test_platform.TaskState_LIFE_CYCLE_REJECTED,
				test_platform.TaskState_VERDICT_UNSPECIFIED,
			),
			Want: requestWithEnumeration(cannedParams(), enumerationsWithDisplayNames("some_test")),
		},
		{
			Tag:         "one enumeration with zero attempts",
			Enumeration: enumerationsWithDisplayNames("some_test"),
			Execution:   executeResponseWithMultipleCompletedAttempts(),
			Want:        requestWithEnumeration(cannedParams(), enumerationsWithDisplayNames("some_test")),
		},
		{
			Tag:         "one enumeration with multiple failing attempts",
			Enumeration: enumerationsWithDisplayNames("some_test"),
			Execution: executeResponseWithMultipleCompletedAttempts(
				test_platform.TaskState_VERDICT_FAILED,
				test_platform.TaskState_VERDICT_FAILED,
			),
			Want: requestWithEnumeration(cannedParams(), enumerationsWithDisplayNames("some_test")),
		},
		{
			Tag:         "one enumeration with success on eventual attempt",
			Enumeration: enumerationsWithDisplayNames("some_test"),
			Execution: executeResponseWithMultipleCompletedAttempts(
				test_platform.TaskState_VERDICT_FAILED,
				test_platform.TaskState_VERDICT_PASSED,
			),
			Want: nil,
		},
		{
			Tag:         "two enumerations, one successful one failing",
			Enumeration: enumerationsWithDisplayNames("failed_test", "passed_test"),
			Execution: executeResponseWithMultipleSingleAttemptEnumerations(
				test_platform.TaskState_VERDICT_FAILED,
				test_platform.TaskState_VERDICT_PASSED,
			),
			Want: requestWithEnumeration(cannedParams(), enumerationsWithDisplayNames("failed_test")),
		},
	}

	for _, c := range cases {
		t.Run(c.Tag, func(t *testing.T) {
			br, err := Compute(&steps.ComputeBackfillRequest{
				Request:     &test_platform.Request{Params: cannedParams()},
				Enumeration: c.Enumeration,
				Execution:   c.Execution,
			})
			if err != nil {
				t.Fatalf("Error in Compute(): %s", err)
			}
			if diff := pretty.Compare(c.Want, br.GetRequest()); diff != "" {
				t.Errorf("Incorrect backfill request, -want +got: %s", diff)
			}
		})
	}
}

func cannedParams() *test_platform.Request_Params {
	return &test_platform.Request_Params{
		HardwareAttributes: &test_platform.Request_Params_HardwareAttributes{
			Model: "some-model",
		},
	}
}

func enumerationsWithDisplayNames(names ...string) *steps.EnumerationResponse {
	resp := &steps.EnumerationResponse{
		AutotestInvocations: make([]*steps.EnumerationResponse_AutotestInvocation, len(names)),
	}
	for i := range names {
		resp.AutotestInvocations[i] = &steps.EnumerationResponse_AutotestInvocation{
			DisplayName: names[i],
		}
	}
	return resp
}

func executeResponseWithSingleAttempt(ls test_platform.TaskState_LifeCycle, v test_platform.TaskState_Verdict) *steps.ExecuteResponse {
	return &steps.ExecuteResponse{
		ConsolidatedResults: []*steps.ExecuteResponse_ConsolidatedResult{
			{
				Attempts: []*steps.ExecuteResponse_TaskResult{
					{
						State: &test_platform.TaskState{
							LifeCycle: ls,
							Verdict:   v,
						},
					},
				},
			},
		},
	}
}

func executeResponseWithMultipleCompletedAttempts(vs ...test_platform.TaskState_Verdict) *steps.ExecuteResponse {
	var as []*steps.ExecuteResponse_TaskResult
	for _, v := range vs {
		as = append(as, &steps.ExecuteResponse_TaskResult{
			State: &test_platform.TaskState{
				LifeCycle: test_platform.TaskState_LIFE_CYCLE_COMPLETED,
				Verdict:   v,
			},
		})
	}
	return &steps.ExecuteResponse{
		ConsolidatedResults: []*steps.ExecuteResponse_ConsolidatedResult{
			{
				Attempts: as,
			},
		},
	}
}

func executeResponseWithMultipleSingleAttemptEnumerations(vs ...test_platform.TaskState_Verdict) *steps.ExecuteResponse {
	var cs []*steps.ExecuteResponse_ConsolidatedResult
	for _, v := range vs {
		cs = append(cs, &steps.ExecuteResponse_ConsolidatedResult{
			Attempts: []*steps.ExecuteResponse_TaskResult{
				{
					State: &test_platform.TaskState{
						LifeCycle: test_platform.TaskState_LIFE_CYCLE_COMPLETED,
						Verdict:   v,
					},
				},
			},
		})
	}
	return &steps.ExecuteResponse{
		ConsolidatedResults: cs,
	}
}

func requestWithEnumeration(p *test_platform.Request_Params, e *steps.EnumerationResponse) *test_platform.Request {
	return &test_platform.Request{
		Params:   p,
		TestPlan: testPlan(e),
	}
}

func testPlan(e *steps.EnumerationResponse) *test_platform.Request_TestPlan {
	ret := &test_platform.Request_TestPlan{
		Enumeration: &test_platform.Request_Enumeration{
			AutotestInvocations: make([]*test_platform.Request_Enumeration_AutotestInvocation, len(e.AutotestInvocations)),
		},
	}
	for i := range e.AutotestInvocations {
		ret.Enumeration.AutotestInvocations[i] = &test_platform.Request_Enumeration_AutotestInvocation{
			DisplayName: e.AutotestInvocations[i].DisplayName,
		}
	}
	return ret
}

func TestMismatchedEnumerationAndExecutionReturnsError(t *testing.T) {
	cases := []struct {
		Tag         string
		Enumeration *steps.EnumerationResponse
		Execution   *steps.ExecuteResponse
	}{
		{
			Tag:         "fewer enumerations than executions",
			Enumeration: enumerationsWithDisplayNames("some_test"),
			Execution: executeResponseWithMultipleSingleAttemptEnumerations(
				test_platform.TaskState_VERDICT_FAILED,
				test_platform.TaskState_VERDICT_PASSED,
			),
		},
		{
			Tag:         "fewer executions than enumerations",
			Enumeration: enumerationsWithDisplayNames("some_test", "some_other_test"),
			Execution:   executeResponseWithMultipleSingleAttemptEnumerations(),
		},
	}
	for _, c := range cases {
		t.Run(c.Tag, func(t *testing.T) {
			_, err := Compute(&steps.ComputeBackfillRequest{
				Request:     &test_platform.Request{Params: cannedParams()},
				Enumeration: c.Enumeration,
				Execution:   c.Execution,
			})
			if err == nil {
				t.Errorf("No error in Compute()")
			}
		})
	}
}
