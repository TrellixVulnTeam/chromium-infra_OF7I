// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package retry provides a way to determine when a task should be retried.
package retry

import (
	"context"
	"fmt"
	"math"

	"infra/cmd/cros_test_platform/internal/execution/types"

	"go.chromium.org/chromiumos/infra/proto/go/test_platform"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/steps"
	"go.chromium.org/luci/common/logging"
)

// IsNeeded determines if a task result indicates that the invocation needs to
// be retried.
//
// Panics on unknown verdicts.
func IsNeeded(result *steps.ExecuteResponse_TaskResult) bool {
	switch v := result.GetState().GetVerdict(); v {
	case test_platform.TaskState_VERDICT_UNSPECIFIED,
		test_platform.TaskState_VERDICT_FAILED:
		return true
	case test_platform.TaskState_VERDICT_NO_VERDICT,
		test_platform.TaskState_VERDICT_PASSED,
		test_platform.TaskState_VERDICT_PASSED_ON_RETRY:
		return false
	default:
		panic(fmt.Sprintf("shouldRetry: unknown verdict %s", v.String()))
	}
}

// NewCounter initializes a new Counter.
func NewCounter(params *test_platform.Request_Params, iids map[types.InvocationID]*steps.EnumerationResponse_AutotestInvocation) Counter {
	rc := Counter{
		globalMaxRetries: inferGlobalMaxRetries(params),
		testRetryCounter: make(map[types.InvocationID]*invocationCounter),
	}
	for name, inv := range iids {
		rc.testRetryCounter[name] = &invocationCounter{
			Max: int(inferTestMaxRetries(inv)),
		}
	}
	return rc
}

// Counter tracks retries across tests and exposes a method to decide if a
// particular test can be retried any further.
type Counter struct {
	globalMaxRetries int32
	retries          int32
	testRetryCounter map[types.InvocationID]*invocationCounter
}

// NotifyRetry notifies retryCounter of a retry attempt for a test.
//
// NotifyRetry panics for an unknown types.InvocationID.
func (c *Counter) NotifyRetry(iid types.InvocationID) {
	c.retries++
	c.getTestRetryCounter(iid).Count++
}

// CanRetry determines if a retry is allowed for a test based on the count of
// retries so far.
//
// CanRetry panics for an unknown types.InvocationID.
func (c *Counter) CanRetry(ctx context.Context, iid types.InvocationID) bool {
	tc := c.getTestRetryCounter(iid)

	if tc.Remaining() <= 0 {
		logging.Infof(ctx, "Not retrying %s. Hit the test retry limit.", iid)
		return false
	}
	if c.globalRetriesRemaining() <= 0 {
		logging.Infof(ctx, "Not retrying %s. Hit the task set retry limit.", iid)
		return false
	}
	return true
}

func (c *Counter) getTestRetryCounter(iid types.InvocationID) *invocationCounter {
	tc, ok := c.testRetryCounter[iid]
	if !ok {
		panic(fmt.Sprintf("unknown test %s", iid))
	}
	return tc
}

func (c *Counter) globalRetriesRemaining() int32 {
	return c.globalMaxRetries - c.retries
}

// invocationCounter tracks the number of retries of a single invocation.
type invocationCounter struct {
	Max   int
	Count int
}

func (t *invocationCounter) Remaining() int {
	r := t.Max - t.Count
	if r > 0 {
		return r
	}
	return 0
}

func inferGlobalMaxRetries(params *test_platform.Request_Params) int32 {
	if !params.GetRetry().GetAllow() {
		return 0
	}
	return maxInt32IfZero(params.GetRetry().GetMax())
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
