// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package execution

import (
	"context"
	"fmt"
	"math"

	"go.chromium.org/chromiumos/infra/proto/go/test_platform"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/steps"
	"go.chromium.org/luci/common/logging"
)

// needsRetry determines if a task result indicates that the invocation needs to
// be retried.
//
// Panics on unknown verdicts.
func needsRetry(result *steps.ExecuteResponse_TaskResult) bool {
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

// newRetryCounter initializes a new retryCounter.
func newRetryCounter(params *test_platform.Request_Params, iids map[invocationID]*steps.EnumerationResponse_AutotestInvocation) retryCounter {
	rc := retryCounter{
		globalMaxRetries: inferGlobalMaxRetries(params),
		testRetryCounter: make(map[invocationID]*testRetryCounter),
	}
	for name, inv := range iids {
		rc.testRetryCounter[name] = &testRetryCounter{
			Max: int(inferTestMaxRetries(inv)),
		}
	}
	return rc
}

// retryCounter tracks retries across tests and exposes a method to decide if a
// particular test can be retried any further.
type retryCounter struct {
	globalMaxRetries int32
	retries          int32
	testRetryCounter map[invocationID]*testRetryCounter
}

// NotifyRetry notifies retryCounter of a retry attempt for a test.
//
// NotifyRetry panics for an unknown invocationID.
func (c *retryCounter) NotifyRetry(iid invocationID) {
	c.retries++
	c.getTestRetryCounter(iid).Count++
}

// CanRetry determines if a retry is allowed for a test based on the count of
// retries so far.
//
// CanRetry panics for an unknown invocationID.
func (c *retryCounter) CanRetry(ctx context.Context, iid invocationID) bool {
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

func (c *retryCounter) getTestRetryCounter(iid invocationID) *testRetryCounter {
	tc, ok := c.testRetryCounter[iid]
	if !ok {
		panic(fmt.Sprintf("unknown test %s", iid))
	}
	return tc
}

func (c *retryCounter) globalRetriesRemaining() int32 {
	return c.globalMaxRetries - c.retries
}

// testRetryCounter tracks the number of retries of a single test.
type testRetryCounter struct {
	Max   int
	Count int
}

func (t *testRetryCounter) Remaining() int {
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
