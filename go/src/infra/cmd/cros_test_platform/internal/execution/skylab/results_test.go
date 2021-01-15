// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package skylab

import (
	"context"
	"errors"
	"infra/libs/skylab/request"
	"infra/libs/skylab/worker"
	"sort"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"

	"go.chromium.org/chromiumos/infra/proto/go/test_platform"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/common"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/skylab_test_runner"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/steps"

	trservice "infra/cmd/cros_test_platform/internal/execution/testrunner/service"
)

type fakeArgsGenerator struct {
	cannedArgs request.Args
}

func (g *fakeArgsGenerator) GenerateArgs(ctx context.Context) (request.Args, error) {
	return g.cannedArgs, nil
}

func (g *fakeArgsGenerator) CheckConsistency() error {
	return nil
}

type fakeClient struct{}

func (c *fakeClient) ValidateArgs(context.Context, *request.Args) (bool, map[string]string, error) {
	return true, nil, nil
}
func (c *fakeClient) LaunchTask(context.Context, *request.Args) (trservice.TaskReference, error) {
	return "fake-task-reference", nil
}
func (c *fakeClient) FetchResults(context.Context, trservice.TaskReference) (*trservice.FetchResultsResponse, error) {
	return nil, errors.New("not implemented in fake")
}
func (c *fakeClient) SwarmingTaskID(trservice.TaskReference) string {
	return ""
}
func (c *fakeClient) URL(trservice.TaskReference) string {
	return ""
}

func TestResultBeforeRefresh(t *testing.T) {
	Convey("Give a single task that has not be Refresh()ed", t, func() {
		t, err := NewTask(
			context.Background(),
			&fakeClient{},
			&fakeArgsGenerator{
				cannedArgs: request.Args{
					Cmd: worker.Command{
						TaskName: "foo-task",
					},
				},
			},
		)
		So(err, ShouldBeNil)
		Convey("Result() returns known values and reasonable defaults", func() {
			r := t.Result()
			So(r.Name, ShouldEqual, "foo-task")
			So(r.State, ShouldNotBeNil)
			So(r.State.LifeCycle, ShouldEqual, test_platform.TaskState_LIFE_CYCLE_PENDING)
			So(r.State.Verdict, ShouldEqual, test_platform.TaskState_VERDICT_UNSPECIFIED)
			So(r.LogUrl, ShouldBeEmpty)
			So(r.LogData, ShouldBeNil)
			So(r.TestCases, ShouldHaveLength, 0)
		})
	})
}

// Test that autotest results for a single completed task map correctly.
func TestSingleAutotestTaskResults(t *testing.T) {
	Convey("Given a single task's autotest results", t, func() {
		cases := []struct {
			description   string
			result        *skylab_test_runner.Result_Autotest
			expectVerdict test_platform.TaskState_Verdict
		}{
			// 0 autotest test cases.
			{
				description:   "with no test cases",
				result:        &skylab_test_runner.Result_Autotest{},
				expectVerdict: test_platform.TaskState_VERDICT_NO_VERDICT,
			},

			// 1 autotest test case.
			{
				description: "with 1 passing test case",
				result: &skylab_test_runner.Result_Autotest{
					TestCases: []*skylab_test_runner.Result_Autotest_TestCase{
						{Verdict: skylab_test_runner.Result_Autotest_TestCase_VERDICT_PASS},
					},
				},
				expectVerdict: test_platform.TaskState_VERDICT_PASSED,
			},
			{
				description: "with 1 failing test case",
				result: &skylab_test_runner.Result_Autotest{
					TestCases: []*skylab_test_runner.Result_Autotest_TestCase{
						{Verdict: skylab_test_runner.Result_Autotest_TestCase_VERDICT_FAIL},
					},
				},
				expectVerdict: test_platform.TaskState_VERDICT_FAILED,
			},
			{
				description: "with 1 undefined-verdict test case",
				result: &skylab_test_runner.Result_Autotest{
					TestCases: []*skylab_test_runner.Result_Autotest_TestCase{
						{Verdict: skylab_test_runner.Result_Autotest_TestCase_VERDICT_UNDEFINED},
					},
				},
				expectVerdict: test_platform.TaskState_VERDICT_NO_VERDICT,
			},
			{
				description: "with 1 not-available-verdict test case",
				result: &skylab_test_runner.Result_Autotest{
					TestCases: []*skylab_test_runner.Result_Autotest_TestCase{
						{Verdict: skylab_test_runner.Result_Autotest_TestCase_VERDICT_NO_VERDICT},
					},
				},
				expectVerdict: test_platform.TaskState_VERDICT_NO_VERDICT,
			},

			// multiple autotest test cases.
			{
				description: "with 2 passing test cases",
				result: &skylab_test_runner.Result_Autotest{
					TestCases: []*skylab_test_runner.Result_Autotest_TestCase{
						{Verdict: skylab_test_runner.Result_Autotest_TestCase_VERDICT_PASS},
						{Verdict: skylab_test_runner.Result_Autotest_TestCase_VERDICT_PASS},
					},
				},
				expectVerdict: test_platform.TaskState_VERDICT_PASSED,
			},
			{
				description: "with 1 passing and 1 not-applicable test case",
				result: &skylab_test_runner.Result_Autotest{
					TestCases: []*skylab_test_runner.Result_Autotest_TestCase{
						{Verdict: skylab_test_runner.Result_Autotest_TestCase_VERDICT_PASS},
						{Verdict: skylab_test_runner.Result_Autotest_TestCase_VERDICT_NO_VERDICT},
					},
				},
				expectVerdict: test_platform.TaskState_VERDICT_PASSED,
			},
			{
				description: "with 1 passing and 1 undefined-verdict test case",
				result: &skylab_test_runner.Result_Autotest{
					TestCases: []*skylab_test_runner.Result_Autotest_TestCase{
						{Verdict: skylab_test_runner.Result_Autotest_TestCase_VERDICT_PASS},
						{Verdict: skylab_test_runner.Result_Autotest_TestCase_VERDICT_UNDEFINED},
					},
				},
				expectVerdict: test_platform.TaskState_VERDICT_PASSED,
			},
			{
				description: "with 1 passing and 1 failing test case",
				result: &skylab_test_runner.Result_Autotest{
					TestCases: []*skylab_test_runner.Result_Autotest_TestCase{
						{Verdict: skylab_test_runner.Result_Autotest_TestCase_VERDICT_PASS},
						{Verdict: skylab_test_runner.Result_Autotest_TestCase_VERDICT_FAIL},
					},
				},
				expectVerdict: test_platform.TaskState_VERDICT_FAILED,
			},

			// task with incomplete test cases
			{
				description: "with 1 passing test case, but incomplete results",
				result: &skylab_test_runner.Result_Autotest{
					Incomplete: true,
					TestCases: []*skylab_test_runner.Result_Autotest_TestCase{
						{Verdict: skylab_test_runner.Result_Autotest_TestCase_VERDICT_PASS},
					},
				},
				expectVerdict: test_platform.TaskState_VERDICT_FAILED,
			},

			// task with no results
			{
				description:   "with no autotest results",
				expectVerdict: test_platform.TaskState_VERDICT_UNSPECIFIED,
			},
		}
		for _, c := range cases {
			Convey(c.description, func() {
				Convey("then task results are correctly converted to verdict.", func() {
					result := callTaskResult(c.result)
					So(result, ShouldNotBeNil)
					So(result.State.LifeCycle, ShouldEqual, test_platform.TaskState_LIFE_CYCLE_COMPLETED)
					So(result.State.Verdict, ShouldEqual, c.expectVerdict)
					So(result.LogData, ShouldNotBeNil)
					if result.LogData != nil {
						So(result.LogData.GsUrl, ShouldEqual, "gs://some-url")
					}
					So(result.LogUrl, ShouldEqual, "https://stainless.corp.google.com/browse/some-url")
				})
			})
		}
	})
}

func TestAutotestTestCases(t *testing.T) {
	Convey("Given a single task's autotest results", t, func() {
		cases := []struct {
			description     string
			result          *skylab_test_runner.Result_Autotest
			expectTestCases []*steps.ExecuteResponse_TaskResult_TestCaseResult
		}{
			{
				description: "with no autotest results",
			},
			{
				description: "with no test cases",
				result:      &skylab_test_runner.Result_Autotest{},
			},
			{
				description: "with multiple test cases",
				result: &skylab_test_runner.Result_Autotest{
					TestCases: []*skylab_test_runner.Result_Autotest_TestCase{
						{
							Name:    "foo-pass",
							Verdict: skylab_test_runner.Result_Autotest_TestCase_VERDICT_PASS,
						},
						{
							Name:    "foo-fail",
							Verdict: skylab_test_runner.Result_Autotest_TestCase_VERDICT_FAIL,
						},
						{
							Name: "foo-undefined",
						},
					},
				},
				expectTestCases: []*steps.ExecuteResponse_TaskResult_TestCaseResult{
					{
						Name:    "foo-pass",
						Verdict: test_platform.TaskState_VERDICT_PASSED,
					},
					{
						Name:    "foo-fail",
						Verdict: test_platform.TaskState_VERDICT_FAILED,
					},
					{
						Name: "foo-undefined",
					},
				},
			},
			{
				description: "with a test case that has an informational string",
				result: &skylab_test_runner.Result_Autotest{
					TestCases: []*skylab_test_runner.Result_Autotest_TestCase{
						{
							Name:                 "foo-fail",
							Verdict:              skylab_test_runner.Result_Autotest_TestCase_VERDICT_FAIL,
							HumanReadableSummary: "Something horrible happened.",
						},
					},
				},
				expectTestCases: []*steps.ExecuteResponse_TaskResult_TestCaseResult{
					{
						Name:                 "foo-fail",
						Verdict:              test_platform.TaskState_VERDICT_FAILED,
						HumanReadableSummary: "Something horrible happened.",
					},
				},
			},
		}
		for _, c := range cases {
			Convey(c.description, func() {
				Convey("then test cases are reported correctly.", func() {
					result := callTaskResult(c.result)
					sort.SliceStable(result.TestCases, func(i, j int) bool {
						return result.TestCases[i].Name < result.TestCases[j].Name
					})
					sort.SliceStable(c.expectTestCases, func(i, j int) bool {
						return c.expectTestCases[i].Name < c.expectTestCases[j].Name
					})
					So(result.TestCases, ShouldResembleProto, c.expectTestCases)
				})
			})
		}
	})
}

func callTaskResult(autotestResult *skylab_test_runner.Result_Autotest) *steps.ExecuteResponse_TaskResult {
	t := &Task{
		result: &skylab_test_runner.Result{
			Harness: &skylab_test_runner.Result_AutotestResult{
				AutotestResult: autotestResult,
			},
			LogData: &common.TaskLogData{
				GsUrl:        "gs://some-url",
				StainlessUrl: "https://stainless.corp.google.com/browse/some-url",
			},
		},
		lifeCycle:      test_platform.TaskState_LIFE_CYCLE_COMPLETED,
		swarmingTaskID: "foo-task-ID",
	}
	return t.Result()
}
