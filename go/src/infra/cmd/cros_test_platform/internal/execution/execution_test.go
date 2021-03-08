// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package execution_test contains blackbox tests for the execution package.
//
// Tests are split across multiple files, grouping together logically related
// tests. `execution_test.go` contains unclassified tests and common helpers.
package execution_test

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/duration"
	. "github.com/smartystreets/goconvey/convey"

	buildapi "go.chromium.org/chromiumos/infra/proto/go/chromite/api"
	"go.chromium.org/chromiumos/infra/proto/go/chromiumos"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/config"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/skylab_test_runner"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/steps"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/steps/execute"
	bbpb "go.chromium.org/luci/buildbucket/proto"
	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/clock/testclock"
	"go.chromium.org/luci/common/data/stringset"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/logging/memlogger"
	"go.chromium.org/luci/luciexe/exe"

	"infra/cmd/cros_test_platform/internal/execution"
	trservice "infra/cmd/cros_test_platform/internal/execution/testrunner/service"
	"infra/libs/skylab/request"
)

func TestLaunchAndWaitTest(t *testing.T) {
	Convey("Given two enumerated test", t, func() {
		Convey("when running a skylab execution", func() {
			trClient := &trservice.CallCountingClientWrapper{
				Client: trservice.StubClient{},
			}
			resps, err := runWithDefaults(
				context.Background(),
				trClient,
				[]*steps.EnumerationResponse_AutotestInvocation{
					clientTestInvocation("", ""),
					clientTestInvocation("", ""),
				},
			)
			So(err, ShouldBeNil)
			resp := extractSingleResponse(resps)

			Convey("then results for all tests are reflected.", func() {
				So(resp.TaskResults, ShouldHaveLength, 2)
				for _, tr := range resp.TaskResults {
					So(tr.State.LifeCycle, ShouldEqual, test_platform.TaskState_LIFE_CYCLE_COMPLETED)
				}
			})
			Convey("then the expected number of external test_runner calls are made.", func() {
				So(trClient.CallCounts.LaunchTask, ShouldEqual, 2)
				So(trClient.CallCounts.FetchResults, ShouldEqual, 2)
			})
		})
	})
}

// Note: the purpose of this test is the test the behavior when a parsed
// autotest result is not available from a task, because the task didn't run
// far enough to output one.
//
// For detailed tests on the handling of autotest test results, see results_test.go.
func TestTaskStates(t *testing.T) {
	Convey("Given a single test", t, func() {
		cases := []struct {
			description   string
			lifeCycle     test_platform.TaskState_LifeCycle
			expectVerdict test_platform.TaskState_Verdict
		}{
			{
				description:   "that was never scheduled",
				lifeCycle:     test_platform.TaskState_LIFE_CYCLE_CANCELLED,
				expectVerdict: test_platform.TaskState_VERDICT_FAILED,
			},
			{
				description:   "that was killed",
				lifeCycle:     test_platform.TaskState_LIFE_CYCLE_ABORTED,
				expectVerdict: test_platform.TaskState_VERDICT_FAILED,
			},
			{
				description:   "that completed",
				lifeCycle:     test_platform.TaskState_LIFE_CYCLE_COMPLETED,
				expectVerdict: test_platform.TaskState_VERDICT_FAILED,
			},
		}
		for _, c := range cases {
			Convey(c.description, func() {
				resps, err := runWithDefaults(
					context.Background(),
					trservice.NewStubClientWithCannedIncompleteTasks(c.lifeCycle),
					[]*steps.EnumerationResponse_AutotestInvocation{
						clientTestInvocation("", ""),
					},
				)
				So(err, ShouldBeNil)
				resp := extractSingleResponse(resps)

				Convey("then the task state is correct.", func() {
					So(resp.TaskResults, ShouldHaveLength, 1)
					So(resp.TaskResults[0].State.LifeCycle, ShouldEqual, c.lifeCycle)
					So(resp.TaskResults[0].State.Verdict, ShouldResemble, c.expectVerdict)
				})
			})
		}
	})
}

func TestLaunchTaskError(t *testing.T) {
	Convey("Error in creating test_runner builds is surfaced correctly", t, func() {
		_, err := runWithDefaults(
			context.Background(),
			errorProneLaunchTaskClient{},
			[]*steps.EnumerationResponse_AutotestInvocation{clientTestInvocation("", "")},
		)
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldContainSubstring, "new task")
		So(err.Error(), ShouldContainSubstring, "simulated error from fake client")
	})
}

type errorProneLaunchTaskClient struct {
	trservice.StubClient
}

// LaunchTask implements Client interface.
func (c errorProneLaunchTaskClient) LaunchTask(ctx context.Context, args *request.Args) (trservice.TaskReference, error) {
	return "", errors.Reason("simulated error from fake client").Err()
}

func TestFetchResultsError(t *testing.T) {
	Convey("Error in fetching test_runner results is surfaced correctly", t, func() {
		_, err := runWithDefaults(
			context.Background(),
			errorProneFetchResultsClient{},
			[]*steps.EnumerationResponse_AutotestInvocation{clientTestInvocation("", "")},
		)
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldContainSubstring, "tick for task")
		So(err.Error(), ShouldContainSubstring, "simulated error from fake client")
	})
}

type errorProneFetchResultsClient struct {
	trservice.StubClient
}

// FetchResults implements Client interface.
func (c errorProneFetchResultsClient) FetchResults(context.Context, trservice.TaskReference) (*trservice.FetchResultsResponse, error) {
	return nil, errors.Reason("simulated error from fake client").Err()
}

func TestTaskURL(t *testing.T) {
	Convey("Given a single enumerated test running to completion, its task URL is propagated correctly.", t, func() {
		resps, err := runWithDefaults(
			context.Background(),
			stubTestRunnerClientWithCannedURL{
				Client:    trservice.NewStubClientWithSuccessfulTasks(),
				CannedURL: "foo-url",
			},
			[]*steps.EnumerationResponse_AutotestInvocation{
				clientTestInvocation("", ""),
			},
		)
		So(err, ShouldBeNil)
		resp := extractSingleResponse(resps)
		So(resp.TaskResults, ShouldHaveLength, 1)
		So(resp.TaskResults[0].TaskUrl, ShouldEqual, "foo-url")
	})
}

func TestIncompleteWait(t *testing.T) {
	Convey("Given a run that is cancelled while running, response reflects cancellation.", t, func() {
		ctx, cancel := context.WithCancel(context.Background())
		var gresps map[string]*steps.ExecuteResponse
		var gerr error
		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			gresps, gerr = runWithDefaults(
				ctx,
				trservice.NewStubClientWithCannedIncompleteTasks(test_platform.TaskState_LIFE_CYCLE_RUNNING),
				[]*steps.EnumerationResponse_AutotestInvocation{
					clientTestInvocation("", ""),
				},
			)
			wg.Done()
		}()

		cancel()
		wg.Wait()
		So(gerr, ShouldBeNil)

		resp := extractSingleResponse(gresps)
		So(resp, ShouldNotBeNil)
		So(resp.TaskResults, ShouldHaveLength, 1)
		So(resp.TaskResults[0].State.LifeCycle, ShouldEqual, test_platform.TaskState_LIFE_CYCLE_RUNNING)
	})
}

func TestEnumerationResponseWithRetries(t *testing.T) {
	Convey("Given a request with retry enabled", t, func() {
		ctx := setFakeTimeWithImmediateTimeout(context.Background())
		params := basicParams()
		params.Retry = &test_platform.Request_Params_Retry{
			Allow: true,
		}
		Convey("and two tests that always fail and retry limit", func() {
			invs := invocationsWithServerTests("name1", "name2")
			for _, inv := range invs {
				inv.Test.AllowRetries = true
				inv.Test.MaxRetries = 2
			}
			Convey("for skylab execution", func() {
				resps, err := runWithParams(
					ctx,
					trservice.NewStubClientWithFailedTasks(),
					params,
					invs,
				)
				So(err, ShouldBeNil)
				resp := extractSingleResponse(resps)
				Convey("response should contain two enumerated results", func() {
					So(resp.ConsolidatedResults, ShouldHaveLength, 2)
				})

				for i, er := range resp.ConsolidatedResults {
					Convey(fmt.Sprintf("%dst enumerated result should contain 3 attempts of a single test", i), func() {
						as := er.GetAttempts()
						n := as[0].Name
						for _, a := range as {
							So(a.Name, ShouldEqual, n)
						}
					})
				}
				Convey("both tests' results should be enumerated", func() {
					names := make([]string, 2)
					for i := range resp.ConsolidatedResults {
						names[i] = resp.ConsolidatedResults[i].Attempts[0].Name
					}
					sort.Strings(names)
					So(names, ShouldResemble, []string{"name1", "name2"})
				})
			})
		})
	})
}

func TestRetries(t *testing.T) {
	Convey("Given a test with", t, func() {
		ctx := context.Background()
		ctx, ts := testclock.UseTime(ctx, time.Now())
		// Setup testclock to immediately advance whenever timer is set; this
		// avoids slowdown due to timer inside of LaunchAndWait.
		ts.SetTimerCallback(func(d time.Duration, t clock.Timer) {
			ts.Add(2 * d)
		})
		params := basicParams()

		cases := []struct {
			name           string
			invocations    []*steps.EnumerationResponse_AutotestInvocation
			trClient       trservice.Client
			retryParams    *test_platform.Request_Params_Retry
			testAllowRetry bool
			testMaxRetry   int32

			// Total number of expected tasks is this +1
			expectedRetryCount          int
			expectedSummaryVerdict      test_platform.TaskState_Verdict
			expectedLogShouldContain    string
			expectedLogShouldNotContain string
		}{
			{
				name:        "1 test; no retry configuration in test or request params",
				invocations: invocationsWithServerTests("name1"),
				trClient:    trservice.NewStubClientWithFailedTasks(),

				expectedRetryCount:       0,
				expectedSummaryVerdict:   test_platform.TaskState_VERDICT_FAILED,
				expectedLogShouldContain: "Hit the test retry limit",
			},
			{
				name:        "1 passing test; retries allowed",
				invocations: invocationsWithServerTests("name1"),
				retryParams: &test_platform.Request_Params_Retry{
					Allow: true,
				},
				testAllowRetry: true,
				testMaxRetry:   1,
				trClient:       trservice.NewStubClientWithSuccessfulTasks(),

				expectedRetryCount:          0,
				expectedSummaryVerdict:      test_platform.TaskState_VERDICT_PASSED,
				expectedLogShouldNotContain: "retry",
			},
			{
				name:        "1 failing test; retries disabled globally",
				invocations: invocationsWithServerTests("name1"),
				retryParams: &test_platform.Request_Params_Retry{
					Allow: false,
				},
				testAllowRetry: true,
				testMaxRetry:   1,
				trClient:       trservice.NewStubClientWithFailedTasks(),

				expectedRetryCount:          0,
				expectedSummaryVerdict:      test_platform.TaskState_VERDICT_FAILED,
				expectedLogShouldContain:    "Hit the task set retry limit",
				expectedLogShouldNotContain: "Hit the test retry limit",
			},
			{
				name:        "1 failing test; retries allowed globally and for test",
				invocations: invocationsWithServerTests("name1"),
				retryParams: &test_platform.Request_Params_Retry{
					Allow: true,
				},
				testAllowRetry: true,
				testMaxRetry:   1,
				trClient:       trservice.NewStubClientWithFailedTasks(),

				expectedRetryCount:     1,
				expectedSummaryVerdict: test_platform.TaskState_VERDICT_FAILED,
			},
			{
				name:        "1 failing test; retries allowed globally, disabled for test",
				invocations: invocationsWithServerTests("name1"),
				retryParams: &test_platform.Request_Params_Retry{
					Allow: true,
				},
				testAllowRetry: false,
				trClient:       trservice.NewStubClientWithFailedTasks(),

				expectedRetryCount:     0,
				expectedSummaryVerdict: test_platform.TaskState_VERDICT_FAILED,
			},
			{
				name:        "1 failing test; retries allowed globally with test maximum",
				invocations: invocationsWithServerTests("name1"),
				retryParams: &test_platform.Request_Params_Retry{
					Allow: true,
				},
				testAllowRetry: true,
				testMaxRetry:   10,
				trClient:       trservice.NewStubClientWithFailedTasks(),

				expectedRetryCount:     10,
				expectedSummaryVerdict: test_platform.TaskState_VERDICT_FAILED,
			},
			{
				name:        "1 failing test; retries allowed globally with global maximum",
				invocations: invocationsWithServerTests("name1"),
				retryParams: &test_platform.Request_Params_Retry{
					Allow: true,
					Max:   5,
				},
				testAllowRetry: true,
				trClient:       trservice.NewStubClientWithFailedTasks(),

				expectedRetryCount:     5,
				expectedSummaryVerdict: test_platform.TaskState_VERDICT_FAILED,
			},
			{
				name:        "1 failing test; retries allowed globally with global maximum smaller than test maxium",
				invocations: invocationsWithServerTests("name1"),
				retryParams: &test_platform.Request_Params_Retry{
					Allow: true,
					Max:   5,
				},
				testAllowRetry: true,
				testMaxRetry:   7,
				trClient:       trservice.NewStubClientWithFailedTasks(),

				expectedRetryCount:          5,
				expectedSummaryVerdict:      test_platform.TaskState_VERDICT_FAILED,
				expectedLogShouldContain:    "Hit the task set retry limit",
				expectedLogShouldNotContain: "Hit the test retry limit",
			},
			{
				name:        "1 failing test; retries allowed globally with test maximum smaller than global maximum",
				invocations: invocationsWithServerTests("name1"),
				retryParams: &test_platform.Request_Params_Retry{
					Allow: true,
					Max:   7,
				},
				testAllowRetry: true,
				testMaxRetry:   5,
				trClient:       trservice.NewStubClientWithFailedTasks(),

				expectedRetryCount:          5,
				expectedSummaryVerdict:      test_platform.TaskState_VERDICT_FAILED,
				expectedLogShouldContain:    "Hit the test retry limit",
				expectedLogShouldNotContain: "Hit the task set retry limit",
			},
			{
				name:        "2 failing tests; retries allowed globally with global maximum",
				invocations: invocationsWithServerTests("name1", "name2"),
				retryParams: &test_platform.Request_Params_Retry{
					Allow: true,
					Max:   5,
				},
				testAllowRetry: true,
				trClient:       trservice.NewStubClientWithFailedTasks(),

				expectedRetryCount:     5,
				expectedSummaryVerdict: test_platform.TaskState_VERDICT_FAILED,
			},

			{
				name:        "1 test that fails then passes; retries allowed",
				invocations: invocationsWithServerTests("name1"),
				retryParams: &test_platform.Request_Params_Retry{
					Allow: true,
				},
				testAllowRetry: true,
				trClient: &trservice.StubClientWithCannedResults{
					CannedResponses: []trservice.FetchResultsResponse{
						{
							LifeCycle: test_platform.TaskState_LIFE_CYCLE_COMPLETED,
							Result: &skylab_test_runner.Result{
								Harness: &skylab_test_runner.Result_AutotestResult{
									AutotestResult: &skylab_test_runner.Result_Autotest{
										Incomplete: false,
										TestCases: []*skylab_test_runner.Result_Autotest_TestCase{
											{
												Name:    "foo",
												Verdict: skylab_test_runner.Result_Autotest_TestCase_VERDICT_FAIL,
											},
										},
									},
								},
							},
						},
						{
							LifeCycle: test_platform.TaskState_LIFE_CYCLE_COMPLETED,
							Result: &skylab_test_runner.Result{
								Harness: &skylab_test_runner.Result_AutotestResult{
									AutotestResult: &skylab_test_runner.Result_Autotest{
										Incomplete: false,
										TestCases: []*skylab_test_runner.Result_Autotest_TestCase{
											{
												Name:    "foo",
												Verdict: skylab_test_runner.Result_Autotest_TestCase_VERDICT_PASS,
											},
										},
									},
								},
							},
						},
					},
				},

				expectedRetryCount: 1,
				// TODO(crbug.com/1005609) Indicate in *some way* that a test
				// passed only on retry.
				expectedSummaryVerdict:      test_platform.TaskState_VERDICT_PASSED,
				expectedLogShouldContain:    "Retrying name1",
				expectedLogShouldNotContain: "retry limit",
			},
		}
		for _, c := range cases {
			Convey(c.name, func() {
				params.Retry = c.retryParams
				for _, inv := range c.invocations {
					inv.Test.AllowRetries = c.testAllowRetry
					inv.Test.MaxRetries = c.testMaxRetry
				}
				var ml memlogger.MemLogger
				ctx = logging.SetFactory(ctx, func(context.Context) logging.Logger { return &ml })
				trClient := &trservice.CallCountingClientWrapper{
					Client: c.trClient,
				}
				resps, err := runWithParams(ctx, trClient, params, c.invocations)
				So(err, ShouldBeNil)
				resp := extractSingleResponse(resps)

				Convey("then the launched task count should be correct.", func() {
					// Each test is tried at least once.
					attemptCount := len(c.invocations) + c.expectedRetryCount
					So(resp.TaskResults, ShouldHaveLength, attemptCount)
				})
				Convey("then task (name, attempt) should be unique.", func() {
					s := make(stringset.Set)
					for _, res := range resp.TaskResults {
						s.Add(fmt.Sprintf("%s__%d", res.Name, res.Attempt))
					}
					So(s, ShouldHaveLength, len(resp.TaskResults))
				})

				Convey("then the build verdict should be correct.", func() {
					So(resp.State.Verdict, ShouldEqual, c.expectedSummaryVerdict)
				})
				Convey("then the log output should match the retry.", func() {
					if len(c.expectedLogShouldContain) > 0 {
						So(loggerInfo(ml), ShouldContainSubstring, c.expectedLogShouldContain)
					}
					if len(c.expectedLogShouldNotContain) > 0 {
						So(loggerInfo(ml), ShouldNotContainSubstring, c.expectedLogShouldNotContain)
					}
				})
			})
		}
	})
}

func TestResponseVerdict(t *testing.T) {
	Convey("Given a client test", t, func() {
		ctx := context.Background()
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		// Setup testclock to immediately advance whenever timer is set; this
		// avoids slowdown due to timer inside of LaunchAndWait.
		ctx, ts := testclock.UseTime(ctx, time.Now())
		ts.SetTimerCallback(func(d time.Duration, t clock.Timer) {
			ts.Add(2 * d)
		})

		Convey("when the test passed, response verdict is correct.", func() {
			resps, err := runWithDefaults(
				ctx,
				trservice.NewStubClientWithSuccessfulTasks(),
				[]*steps.EnumerationResponse_AutotestInvocation{
					serverTestInvocation("name1", ""),
				},
			)
			So(err, ShouldBeNil)
			resp := extractSingleResponse(resps)
			So(resp.State.LifeCycle, ShouldEqual, test_platform.TaskState_LIFE_CYCLE_COMPLETED)
			So(resp.State.Verdict, ShouldEqual, test_platform.TaskState_VERDICT_PASSED)
		})

		Convey("when the test failed, response verdict is correct.", func() {
			resps, err := runWithDefaults(
				ctx,
				trservice.NewStubClientWithFailedTasks(),
				[]*steps.EnumerationResponse_AutotestInvocation{
					serverTestInvocation("name1", ""),
				},
			)
			So(err, ShouldBeNil)
			resp := extractSingleResponse(resps)
			So(resp.State.LifeCycle, ShouldEqual, test_platform.TaskState_LIFE_CYCLE_COMPLETED)
			So(resp.State.Verdict, ShouldEqual, test_platform.TaskState_VERDICT_FAILED)
		})

		SkipConvey("when execution is aborted (e.g., timeout), response verdict is correct.", func() {
			wg := sync.WaitGroup{}
			wg.Add(1)
			var resps map[string]*steps.ExecuteResponse
			var err error
			go func() {
				resps, err = runWithDefaults(
					ctx,
					trservice.NewStubClientWithCannedIncompleteTasks(test_platform.TaskState_LIFE_CYCLE_RUNNING),
					[]*steps.EnumerationResponse_AutotestInvocation{
						serverTestInvocation("name1", ""),
					},
				)
				wg.Done()
			}()

			cancel()
			wg.Wait()
			So(err, ShouldBeNil)
			resp := extractSingleResponse(resps)
			So(resp.State.LifeCycle, ShouldEqual, test_platform.TaskState_LIFE_CYCLE_ABORTED)
			So(resp.State.Verdict, ShouldEqual, test_platform.TaskState_VERDICT_FAILED)
		})
	})
}

func runWithDefaults(ctx context.Context, skylab trservice.Client, invs []*steps.EnumerationResponse_AutotestInvocation) (map[string]*steps.ExecuteResponse, error) {
	return runWithParams(ctx, skylab, basicParams(), invs)
}

func runWithParams(ctx context.Context, skylab trservice.Client, params *test_platform.Request_Params, invs []*steps.EnumerationResponse_AutotestInvocation) (map[string]*steps.ExecuteResponse, error) {
	args := execution.Args{
		Build: &bbpb.Build{},
		Send:  exe.BuildSender(func() {}),
		Request: &steps.ExecuteRequests{
			TaggedRequests: map[string]*steps.ExecuteRequest{
				"12345678/foo": {
					RequestParams: params,
					Enumeration: &steps.EnumerationResponse{
						AutotestInvocations: invs,
					},
				},
			},
			Build: &execute.Build{
				Id: 42,
			},
		},
		WorkerConfig: &config.Config_SkylabWorker{
			LuciProject: "foo-luci-project",
			LogDogHost:  "foo-logdog-host",
		},
		ParentTaskID: "foo-parent-task-id",
		Deadline:     time.Now().Add(time.Hour),
	}
	return execution.Run(ctx, skylab, args)
}

func basicParams() *test_platform.Request_Params {
	return &test_platform.Request_Params{
		SoftwareAttributes: &test_platform.Request_Params_SoftwareAttributes{
			BuildTarget: &chromiumos.BuildTarget{Name: "foo-board"},
		},
		HardwareAttributes: &test_platform.Request_Params_HardwareAttributes{
			Model: "foo-model",
		},
		FreeformAttributes: &test_platform.Request_Params_FreeformAttributes{
			SwarmingDimensions: []string{"freeform-key:freeform-value"},
		},
		SoftwareDependencies: []*test_platform.Request_Params_SoftwareDependency{
			{
				Dep: &test_platform.Request_Params_SoftwareDependency_ChromeosBuild{ChromeosBuild: "foo-build"},
			},
			{
				Dep: &test_platform.Request_Params_SoftwareDependency_RoFirmwareBuild{RoFirmwareBuild: "foo-ro-firmware"},
			},
			{
				Dep: &test_platform.Request_Params_SoftwareDependency_RwFirmwareBuild{RwFirmwareBuild: "foo-rw-firmware"},
			},
		},
		Scheduling: &test_platform.Request_Params_Scheduling{
			Pool: &test_platform.Request_Params_Scheduling_ManagedPool_{
				ManagedPool: test_platform.Request_Params_Scheduling_MANAGED_POOL_CQ,
			},
			Priority: 79,
		},
		Time: &test_platform.Request_Params_Time{
			MaximumDuration: &duration.Duration{Seconds: 60},
		},
		Decorations: &test_platform.Request_Params_Decorations{
			AutotestKeyvals: map[string]string{"k1": "v1"},
			Tags:            []string{"foo-tag1", "foo-tag2"},
		},
	}
}

func invocationsWithServerTests(names ...string) []*steps.EnumerationResponse_AutotestInvocation {
	ret := make([]*steps.EnumerationResponse_AutotestInvocation, len(names))
	for i, n := range names {
		ret[i] = serverTestInvocation(n, "")
	}
	return ret
}

func serverTestInvocation(name string, args string) *steps.EnumerationResponse_AutotestInvocation {
	return &steps.EnumerationResponse_AutotestInvocation{
		Test: &buildapi.AutotestTest{
			Name:                 name,
			ExecutionEnvironment: buildapi.AutotestTest_EXECUTION_ENVIRONMENT_SERVER,
		},
		TestArgs: args,
	}
}

func clientTestInvocation(name string, args string) *steps.EnumerationResponse_AutotestInvocation {
	return &steps.EnumerationResponse_AutotestInvocation{
		Test: &buildapi.AutotestTest{
			Name:                 name,
			ExecutionEnvironment: buildapi.AutotestTest_EXECUTION_ENVIRONMENT_CLIENT,
		},
		TestArgs: args,
	}
}

func setFakeTimeWithImmediateTimeout(ctx context.Context) context.Context {
	ctx, ts := testclock.UseTime(ctx, time.Now())
	// Setup testclock to immediately advance whenever timer is set; this
	// avoids slowdown due to timer inside of LaunchAndWait.
	ts.SetTimerCallback(func(d time.Duration, t clock.Timer) {
		ts.Add(2 * d)
	})
	return ctx
}

func extractSingleResponse(resps map[string]*steps.ExecuteResponse) *steps.ExecuteResponse {
	So(resps, ShouldHaveLength, 1)
	for _, resp := range resps {
		So(resp, ShouldNotBeNil)
		return resp
	}
	panic("unreachable")
}

func loggerInfo(ml memlogger.MemLogger) string {
	out := ""
	for _, m := range ml.Messages() {
		if m.Level == logging.Info {
			out = out + m.Msg
		}
	}
	return out
}
