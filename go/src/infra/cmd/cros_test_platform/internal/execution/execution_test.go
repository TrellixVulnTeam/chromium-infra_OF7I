// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package execution_test

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/duration"
	. "github.com/smartystreets/goconvey/convey"

	"go.chromium.org/chromiumos/infra/proto/go/chromite/api"
	build_api "go.chromium.org/chromiumos/infra/proto/go/chromite/api"
	"go.chromium.org/chromiumos/infra/proto/go/chromiumos"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/config"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/skylab_test_runner"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/steps"
	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/clock/testclock"
	"go.chromium.org/luci/common/data/stringset"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/logging/memlogger"

	"infra/cmd/cros_test_platform/internal/execution"
	"infra/cmd/cros_test_platform/internal/execution/skylab"
	"infra/libs/skylab/inventory"
	"infra/libs/skylab/request"
)

var (
	noDeadline        time.Time
	fakeTaskSetConfig = &execution.TaskSetConfig{
		"foo-parent-task-id",
		"TestPlanRuns/12345678/foo",
		noDeadline,
	}
)

type fakeSkylab struct {
	autotestResultGenerator autotestResultGenerator
	botExists               bool
	callback                func()
	launchCalls             []*request.Args
	nextError               error
	nextLifeCycle           test_platform.TaskState_LifeCycle
	numResultsCalls         int
	url                     string
}

func newFakeSkylab() *fakeSkylab {
	return &fakeSkylab{
		autotestResultGenerator: autotestResultAlwaysPass,
		botExists:               true,
		callback:                func() {},
		nextLifeCycle:           test_platform.TaskState_LIFE_CYCLE_COMPLETED,
	}
}

// setError causes this fake to start returning the given error on all
// future API calls.
func (s *fakeSkylab) setError(err error) {
	s.nextError = err
}

// setCallback causes this fake to call the given callback function, immediately
// prior to the return of every future API call.
func (s *fakeSkylab) setCallback(fn func()) {
	s.callback = fn
}

func (s *fakeSkylab) setURL(url string) {
	s.url = url
}

func (s *fakeSkylab) setLifeCycle(lc test_platform.TaskState_LifeCycle) {
	s.nextLifeCycle = lc
}

func (s *fakeSkylab) setAutotestResultGenerator(f autotestResultGenerator) {
	s.autotestResultGenerator = f
}

func (s *fakeSkylab) ValidateArgs(context.Context, *request.Args) (bool, error) {
	return s.botExists, nil
}

func (s *fakeSkylab) LaunchTask(_ context.Context, req *request.Args) (skylab.TaskReference, error) {
	defer s.callback()
	if s.nextError != nil {
		return skylab.TaskReference(""), s.nextError
	}
	s.launchCalls = append(s.launchCalls, req)
	return skylab.TaskReference(""), nil
}

func (s *fakeSkylab) FetchResults(context.Context, skylab.TaskReference) (*skylab.FetchResultsResponse, error) {
	s.numResultsCalls += 1
	if s.nextError != nil {
		return nil, s.nextError
	}
	return &skylab.FetchResultsResponse{
		Result: &skylab_test_runner.Result{
			Harness: &skylab_test_runner.Result_AutotestResult{
				AutotestResult: s.autotestResultGenerator(),
			},
		},
		LifeCycle: s.nextLifeCycle,
	}, nil
}

func (s *fakeSkylab) SwarmingTaskID(skylab.TaskReference) string {
	return ""
}

func (s *fakeSkylab) URL(skylab.TaskReference) string {
	return s.url
}

func invocation(name string, args string, e build_api.AutotestTest_ExecutionEnvironment) *steps.EnumerationResponse_AutotestInvocation {
	return &steps.EnumerationResponse_AutotestInvocation{
		Test:     &build_api.AutotestTest{Name: name, ExecutionEnvironment: e},
		TestArgs: args,
	}
}

func clientTestInvocation(name string, args string) *steps.EnumerationResponse_AutotestInvocation {
	return &steps.EnumerationResponse_AutotestInvocation{
		Test: &build_api.AutotestTest{
			Name:                 name,
			ExecutionEnvironment: build_api.AutotestTest_EXECUTION_ENVIRONMENT_CLIENT,
		},
		TestArgs: args,
	}
}

func serverTestInvocation(name string, args string) *steps.EnumerationResponse_AutotestInvocation {
	return &steps.EnumerationResponse_AutotestInvocation{
		Test: &build_api.AutotestTest{
			Name:                 name,
			ExecutionEnvironment: build_api.AutotestTest_EXECUTION_ENVIRONMENT_SERVER,
		},
		TestArgs: args,
	}
}

func addAutotestDependency(inv *steps.EnumerationResponse_AutotestInvocation, dep string) {
	inv.Test.Dependencies = append(inv.Test.Dependencies, &api.AutotestTaskDependency{Label: dep})
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

func basicConfig() *config.Config_SkylabWorker {
	return &config.Config_SkylabWorker{
		LuciProject: "foo-luci-project",
		LogDogHost:  "foo-logdog-host",
	}
}

func TestLaunchForNonExistentBot(t *testing.T) {
	Convey("Given one test invocation but non existent bots", t, func() {
		ctx := context.Background()

		skylab := newFakeSkylab()
		skylab.botExists = false

		invs := []*steps.EnumerationResponse_AutotestInvocation{
			clientTestInvocation("", ""),
		}

		Convey("when running a skylab execution", func() {
			ts, err := execution.NewRequestTaskSet(invs, basicParams(), basicConfig(), fakeTaskSetConfig)
			So(err, ShouldBeNil)
			run := execution.NewRunnerWithRequestTaskSets(ts)
			err = run.LaunchAndWait(ctx, skylab)
			So(err, ShouldBeNil)

			resp := getSingleResponse(run)
			So(resp, ShouldNotBeNil)

			Convey("then task result is complete with unspecified verdict.", func() {
				So(resp.TaskResults, ShouldHaveLength, 1)
				tr := resp.TaskResults[0]
				So(tr.State.LifeCycle, ShouldEqual, test_platform.TaskState_LIFE_CYCLE_REJECTED)
				So(tr.State.Verdict, ShouldEqual, test_platform.TaskState_VERDICT_UNSPECIFIED)

			})
			Convey("and overall result is complete with failed verdict.", func() {
				So(resp.State.LifeCycle, ShouldEqual, test_platform.TaskState_LIFE_CYCLE_COMPLETED)
				So(resp.State.Verdict, ShouldEqual, test_platform.TaskState_VERDICT_FAILED)
			})
			Convey("and no skylab tasks are created.", func() {
				So(skylab.launchCalls, ShouldHaveLength, 0)
				So(skylab.numResultsCalls, ShouldEqual, 0)
			})
		})
	})
}

func TestLaunchAndWaitTest(t *testing.T) {
	Convey("Given two enumerated test", t, func() {
		ctx := context.Background()
		skylab := newFakeSkylab()

		var invs []*steps.EnumerationResponse_AutotestInvocation
		invs = append(invs, clientTestInvocation("", ""), clientTestInvocation("", ""))

		Convey("when running a skylab execution", func() {
			ts, err := execution.NewRequestTaskSet(invs, basicParams(), basicConfig(), fakeTaskSetConfig)
			So(err, ShouldBeNil)
			run := execution.NewRunnerWithRequestTaskSets(ts)

			err = run.LaunchAndWait(ctx, skylab)
			So(err, ShouldBeNil)

			resp := getSingleResponse(run)
			So(resp, ShouldNotBeNil)

			Convey("then results for all tests are reflected.", func() {
				So(resp.TaskResults, ShouldHaveLength, 2)
				for _, tr := range resp.TaskResults {
					So(tr.State.LifeCycle, ShouldEqual, test_platform.TaskState_LIFE_CYCLE_COMPLETED)
				}
			})
			Convey("then the expected number of external Skylab calls are made.", func() {
				So(skylab.launchCalls, ShouldHaveLength, 2)
				So(skylab.numResultsCalls, ShouldEqual, 2)
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
		ctx := context.Background()

		var invs []*steps.EnumerationResponse_AutotestInvocation
		invs = append(invs, clientTestInvocation("", ""))

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
				ts, err := execution.NewRequestTaskSet(invs, basicParams(), basicConfig(), fakeTaskSetConfig)
				So(err, ShouldBeNil)
				run := execution.NewRunnerWithRequestTaskSets(ts)

				skylab := newFakeSkylab()
				skylab.setLifeCycle(c.lifeCycle)
				skylab.setAutotestResultGenerator(autotestResultAlwaysEmpty)
				err = run.LaunchAndWait(ctx, skylab)
				So(err, ShouldBeNil)

				Convey("then the task state is correct.", func() {
					resp := getSingleResponse(run)
					So(resp.TaskResults, ShouldHaveLength, 1)
					So(resp.TaskResults[0].State.LifeCycle, ShouldEqual, c.lifeCycle)
					So(resp.TaskResults[0].State.Verdict, ShouldResemble, c.expectVerdict)
				})
			})
		}
	})
}

func TestServiceError(t *testing.T) {
	Convey("Given a single enumerated test", t, func() {
		ctx := context.Background()
		skylab := newFakeSkylab()

		invs := []*steps.EnumerationResponse_AutotestInvocation{clientTestInvocation("", "")}
		ts, err := execution.NewRequestTaskSet(invs, basicParams(), basicConfig(), fakeTaskSetConfig)
		So(err, ShouldBeNil)
		run := execution.NewRunnerWithRequestTaskSets(ts)

		Convey("when the skylab service immediately returns errors, that error is surfaced as a launch error.", func() {
			skylab.setError(fmt.Errorf("foo error"))
			err = run.LaunchAndWait(ctx, skylab)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "launch attempt")
			So(err.Error(), ShouldContainSubstring, "foo error")
		})

		Convey("when the skylab service starts returning errors after the initial launch calls, that errors is surfaced as a wait error.", func() {
			skylab.setCallback(func() {
				skylab.setError(fmt.Errorf("foo error"))
			})
			err = run.LaunchAndWait(ctx, skylab)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "tick for task")
			So(err.Error(), ShouldContainSubstring, "foo error")
		})
	})
}

func TestTaskURL(t *testing.T) {
	Convey("Given a single enumerated test running to completion, its task URL is propagated correctly.", t, func() {
		ctx := context.Background()
		skylab := newFakeSkylab()
		skylab.setURL("foo-url")

		invs := []*steps.EnumerationResponse_AutotestInvocation{clientTestInvocation("", "")}
		ts, err := execution.NewRequestTaskSet(invs, basicParams(), basicConfig(), fakeTaskSetConfig)
		So(err, ShouldBeNil)
		run := execution.NewRunnerWithRequestTaskSets(ts)

		err = run.LaunchAndWait(ctx, skylab)
		So(err, ShouldBeNil)

		resp := getSingleResponse(run)
		So(resp.TaskResults, ShouldHaveLength, 1)
		So(resp.TaskResults[0].TaskUrl, ShouldEqual, "foo-url")
	})
}

func TestIncompleteWait(t *testing.T) {
	Convey("Given a run that is cancelled while running, response reflects cancellation.", t, func() {
		ctx, cancel := context.WithCancel(context.Background())

		skylab := newFakeSkylab()
		skylab.setLifeCycle(test_platform.TaskState_LIFE_CYCLE_RUNNING)

		invs := []*steps.EnumerationResponse_AutotestInvocation{clientTestInvocation("", "")}
		ts, err := execution.NewRequestTaskSet(invs, basicParams(), basicConfig(), fakeTaskSetConfig)
		So(err, ShouldBeNil)
		run := execution.NewRunnerWithRequestTaskSets(ts)

		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			err = run.LaunchAndWait(ctx, skylab)
			wg.Done()
		}()

		cancel()
		wg.Wait()
		So(err, ShouldBeNil)

		resp := getSingleResponse(run)
		So(resp, ShouldNotBeNil)
		So(resp.TaskResults, ShouldHaveLength, 1)
		So(resp.TaskResults[0].State.LifeCycle, ShouldEqual, test_platform.TaskState_LIFE_CYCLE_RUNNING)
	})
}

func TestRequestArguments(t *testing.T) {
	Convey("Given a server test with autotest labels", t, func() {
		ctx := context.Background()
		skylab := newFakeSkylab()

		inv := serverTestInvocation("name1", "foo-arg1 foo-arg2")
		addAutotestDependency(inv, "cr50:pvt")
		addAutotestDependency(inv, "cleanup-reboot")
		inv.DisplayName = "given_name"
		invs := []*steps.EnumerationResponse_AutotestInvocation{inv}

		ts, err := execution.NewRequestTaskSet(invs, basicParams(), basicConfig(), fakeTaskSetConfig)
		So(err, ShouldBeNil)
		run := execution.NewRunnerWithRequestTaskSets(ts)

		err = run.LaunchAndWait(ctx, skylab)
		So(err, ShouldBeNil)

		Convey("the launched task request should have correct parameters.", func() {
			So(skylab.launchCalls, ShouldHaveLength, 1)
			launchArgs := skylab.launchCalls[0]

			So(launchArgs.SwarmingTags, ShouldContain, "luci_project:foo-luci-project")
			So(launchArgs.SwarmingTags, ShouldContain, "foo-tag1")
			So(launchArgs.SwarmingTags, ShouldContain, "foo-tag2")
			So(launchArgs.ParentTaskID, ShouldEqual, fakeTaskSetConfig.ParentTaskID)
			So(launchArgs.ParentRequestUID, ShouldEqual, fakeTaskSetConfig.RequestUID)

			So(launchArgs.Priority, ShouldEqual, 79)

			prefix := "log_location:"
			var logdogURL string
			matchingTags := 0
			for _, tag := range launchArgs.SwarmingTags {
				if strings.HasPrefix(tag, prefix) {
					matchingTags++
					So(tag, ShouldEndWith, "+/annotations")

					logdogURL = strings.TrimPrefix(tag, "log_location:")
				}
			}
			So(matchingTags, ShouldEqual, 1)
			So(logdogURL, ShouldStartWith, "logdog://foo-logdog-host/foo-luci-project/skylab/")
			So(logdogURL, ShouldEndWith, "/+/annotations")

			So(launchArgs.Cmd.TaskName, ShouldEqual, "name1")
			So(launchArgs.Cmd.ClientTest, ShouldBeFalse)

			// Logdog annotation url argument should match the associated tag's url.
			So(launchArgs.Cmd.LogDogAnnotationURL, ShouldEqual, logdogURL)

			So(launchArgs.Cmd.TestArgs, ShouldEqual, "foo-arg1 foo-arg2")

			So(launchArgs.Cmd.Keyvals["k1"], ShouldEqual, "v1")
			So(launchArgs.Cmd.Keyvals["parent_job_id"], ShouldEqual, "foo-parent-task-id")
			So(launchArgs.Cmd.Keyvals["label"], ShouldEqual, "given_name")

			So(launchArgs.ProvisionableDimensions, ShouldHaveLength, 3)
			So(launchArgs.ProvisionableDimensions, ShouldContain, "provisionable-cros-version:foo-build")
			So(launchArgs.ProvisionableDimensions, ShouldContain, "provisionable-fwro-version:foo-ro-firmware")
			So(launchArgs.ProvisionableDimensions, ShouldContain, "provisionable-fwrw-version:foo-rw-firmware")

			So(launchArgs.ProvisionableDimensionExpiration, ShouldEqual, time.Minute)

			So(launchArgs.SchedulableLabels.GetCr50Phase(), ShouldEqual, inventory.SchedulableLabels_CR50_PHASE_PVT)
			So(launchArgs.SchedulableLabels.GetModel(), ShouldEqual, "foo-model")
			So(launchArgs.SchedulableLabels.GetBoard(), ShouldEqual, "foo-board")
			So(launchArgs.SchedulableLabels.GetCriticalPools(), ShouldHaveLength, 1)
			So(launchArgs.SchedulableLabels.GetCriticalPools()[0], ShouldEqual, inventory.SchedulableLabels_DUT_POOL_CQ)

			So(launchArgs.Dimensions, ShouldHaveLength, 1)
			So(launchArgs.Dimensions, ShouldContain, "freeform-key:freeform-value")
		})
	})
}

var keyvalsPattern = regexp.MustCompile(`\-keyvals\s*\{([\w\s":,-/]+)\}`)

func extractKeyvalsArgument(cmd string) string {
	ms := keyvalsPattern.FindAllStringSubmatch(cmd, -1)
	So(ms, ShouldHaveLength, 1)
	m := ms[0]
	// Guaranteed by the constant regexp definition.
	if len(m) != 2 {
		panic(fmt.Sprintf("Match %s of regexp %s has length %d, want 2", m, keyvalsPattern, len(m)))
	}
	return m[1]
}

type autotestResultGenerator func() *skylab_test_runner.Result_Autotest

func autotestResultAlwaysEmpty() *skylab_test_runner.Result_Autotest {
	return nil
}

// generateAutotestResultsFromSlice returns a autotestResultGenerator that
// sequentially returns the provided results.
//
// An attempt to generate more results than provided results in panic().
func generateAutotestResultsFromSlice(canned []*skylab_test_runner.Result_Autotest) autotestResultGenerator {
	i := 0
	f := func() *skylab_test_runner.Result_Autotest {
		if i >= len(canned) {
			panic(fmt.Sprintf("requested more results than available (%d)", len(canned)))
		}
		r := canned[i]
		i++
		return r
	}
	return f
}

func autotestResultAlwaysPass() *skylab_test_runner.Result_Autotest {
	return &skylab_test_runner.Result_Autotest{
		Incomplete: false,
		TestCases: []*skylab_test_runner.Result_Autotest_TestCase{
			{Name: "foo", Verdict: skylab_test_runner.Result_Autotest_TestCase_VERDICT_PASS},
		},
	}
}

func autotestResultAlwaysFail() *skylab_test_runner.Result_Autotest {
	return &skylab_test_runner.Result_Autotest{
		Incomplete: false,
		TestCases: []*skylab_test_runner.Result_Autotest_TestCase{
			{Name: "foo", Verdict: skylab_test_runner.Result_Autotest_TestCase_VERDICT_FAIL},
		},
	}
}

func TestInvocationKeyvals(t *testing.T) {
	Convey("Given an enumeration with a suite keyval", t, func() {
		ctx := context.Background()
		skylab := newFakeSkylab()

		invs := []*steps.EnumerationResponse_AutotestInvocation{
			{
				Test: &api.AutotestTest{
					Name:                 "someTest",
					ExecutionEnvironment: api.AutotestTest_EXECUTION_ENVIRONMENT_CLIENT,
				},
				ResultKeyvals: map[string]string{
					"suite": "someSuite",
				},
			},
		}

		Convey("and a request without keyvals", func() {
			p := basicParams()
			p.Decorations = nil
			ts, err := execution.NewRequestTaskSet(invs, p, basicConfig(), fakeTaskSetConfig)
			So(err, ShouldBeNil)
			run := execution.NewRunnerWithRequestTaskSets(ts)

			err = run.LaunchAndWait(ctx, skylab)
			So(err, ShouldBeNil)
			Convey("created command includes invocation suite keyval", func() {
				So(skylab.launchCalls, ShouldHaveLength, 1)
				launchArgs := skylab.launchCalls[0]
				flatCommand := strings.Join(launchArgs.Cmd.Args(), " ")
				keyvals := extractKeyvalsArgument(flatCommand)
				So(keyvals, ShouldContainSubstring, `"suite":"someSuite"`)
				So(keyvals, ShouldContainSubstring, `"label":"foo-build/someSuite/someTest"`)
			})
		})

		Convey("and a request with different suite keyvals", func() {
			p := basicParams()
			p.Decorations = &test_platform.Request_Params_Decorations{
				AutotestKeyvals: map[string]string{
					"suite": "someOtherSuite",
				},
			}
			ts, err := execution.NewRequestTaskSet(invs, p, basicConfig(), fakeTaskSetConfig)
			So(err, ShouldBeNil)
			run := execution.NewRunnerWithRequestTaskSets(ts)

			err = run.LaunchAndWait(ctx, skylab)
			So(err, ShouldBeNil)
			Convey("created command includes request suite keyval", func() {
				So(skylab.launchCalls, ShouldHaveLength, 1)
				launchArgs := skylab.launchCalls[0]
				flatCommand := strings.Join(launchArgs.Cmd.Args(), " ")
				keyvals := extractKeyvalsArgument(flatCommand)
				So(keyvals, ShouldContainSubstring, `"suite":"someOtherSuite"`)
				So(keyvals, ShouldContainSubstring, `"label":"foo-build/someOtherSuite/someTest"`)
			})
		})
	})
}

func invocationsWithServerTests(names ...string) []*steps.EnumerationResponse_AutotestInvocation {
	ret := make([]*steps.EnumerationResponse_AutotestInvocation, len(names))
	for i, n := range names {
		ret[i] = serverTestInvocation(n, "")
	}
	return ret
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
func TestKeyvalsAcrossTestRuns(t *testing.T) {
	Convey("Given a request with a suite keyval", t, func() {
		ctx := context.Background()
		skylab := newFakeSkylab()

		p := basicParams()
		p.Decorations = &test_platform.Request_Params_Decorations{
			AutotestKeyvals: map[string]string{
				"suite": "someSuite",
			},
		}

		Convey("and two enumerations with different test names", func() {

			invs := []*steps.EnumerationResponse_AutotestInvocation{
				{
					Test: &api.AutotestTest{
						Name:                 "firstTest",
						ExecutionEnvironment: api.AutotestTest_EXECUTION_ENVIRONMENT_CLIENT,
					},
				},
				{
					Test: &api.AutotestTest{
						Name:                 "secondTest",
						ExecutionEnvironment: api.AutotestTest_EXECUTION_ENVIRONMENT_CLIENT,
					},
				},
			}

			Convey("created commands include common suite keyval and different label keyvals", func() {
				ts, err := execution.NewRequestTaskSet(invs, p, basicConfig(), fakeTaskSetConfig)
				So(err, ShouldBeNil)
				run := execution.NewRunnerWithRequestTaskSets(ts)
				err = run.LaunchAndWait(ctx, skylab)
				So(err, ShouldBeNil)

				So(skylab.launchCalls, ShouldHaveLength, 2)
				cmd := make([]string, 2)
				for i, ls := range skylab.launchCalls {
					cmd[i] = strings.Join(ls.Cmd.Args(), " ")
				}
				kv0 := extractKeyvalsArgument(cmd[0])
				So(kv0, ShouldContainSubstring, `"suite":"someSuite"`)
				So(kv0, ShouldContainSubstring, `"label":"foo-build/someSuite/firstTest"`)
				kv1 := extractKeyvalsArgument(cmd[1])
				So(kv1, ShouldContainSubstring, `"suite":"someSuite"`)
				So(kv1, ShouldContainSubstring, `"label":"foo-build/someSuite/secondTest"`)
			})
		})
	})
}

func TestEnumerationResponseWithRetries(t *testing.T) {
	Convey("Given a request with retry enabled", t, func() {
		ctx := context.Background()
		ctx = setFakeTimeWithImmediateTimeout(ctx)
		params := basicParams()
		params.Retry = &test_platform.Request_Params_Retry{
			Allow: true,
		}
		skylab := newFakeSkylab()
		Convey("and two tests that always fail and retry limit", func() {
			skylab.setAutotestResultGenerator(autotestResultAlwaysFail)
			invs := invocationsWithServerTests("name1", "name2")
			for _, inv := range invs {
				inv.Test.AllowRetries = true
				inv.Test.MaxRetries = 2
			}
			Convey("for skylab execution", func() {
				ts, err := execution.NewRequestTaskSet(invs, params, basicConfig(), fakeTaskSetConfig)
				So(err, ShouldBeNil)
				run := execution.NewRunnerWithRequestTaskSets(ts)
				err = run.LaunchAndWait(ctx, skylab)
				So(err, ShouldBeNil)
				resp := getSingleResponse(run)
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

func setFakeTimeWithImmediateTimeout(ctx context.Context) context.Context {
	ctx, ts := testclock.UseTime(ctx, time.Now())
	// Setup testclock to immediately advance whenever timer is set; this
	// avoids slowdown due to timer inside of LaunchAndWait.
	ts.SetTimerCallback(func(d time.Duration, t clock.Timer) {
		ts.Add(2 * d)
	})
	return ctx
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
		skylab := newFakeSkylab()

		cases := []struct {
			name        string
			invocations []*steps.EnumerationResponse_AutotestInvocation
			// autotestResult will be returned by all attempts of this test.
			autotestResultGenerator autotestResultGenerator
			retryParams             *test_platform.Request_Params_Retry
			testAllowRetry          bool
			testMaxRetry            int32

			// Total number of expected tasks is this +1
			expectedRetryCount          int
			expectedSummaryVerdict      test_platform.TaskState_Verdict
			expectedLogShouldContain    string
			expectedLogShouldNotContain string
		}{
			{
				name:                    "1 test; no retry configuration in test or request params",
				invocations:             invocationsWithServerTests("name1"),
				autotestResultGenerator: autotestResultAlwaysFail,

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
				testAllowRetry:          true,
				testMaxRetry:            1,
				autotestResultGenerator: autotestResultAlwaysPass,

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
				testAllowRetry:          true,
				testMaxRetry:            1,
				autotestResultGenerator: autotestResultAlwaysFail,

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
				testAllowRetry:          true,
				testMaxRetry:            1,
				autotestResultGenerator: autotestResultAlwaysFail,

				expectedRetryCount:     1,
				expectedSummaryVerdict: test_platform.TaskState_VERDICT_FAILED,
			},
			{
				name:        "1 failing test; retries allowed globally, disabled for test",
				invocations: invocationsWithServerTests("name1"),
				retryParams: &test_platform.Request_Params_Retry{
					Allow: true,
				},
				testAllowRetry:          false,
				autotestResultGenerator: autotestResultAlwaysFail,

				expectedRetryCount:     0,
				expectedSummaryVerdict: test_platform.TaskState_VERDICT_FAILED,
			},
			{
				name:        "1 failing test; retries allowed globally with test maximum",
				invocations: invocationsWithServerTests("name1"),
				retryParams: &test_platform.Request_Params_Retry{
					Allow: true,
				},
				testAllowRetry:          true,
				testMaxRetry:            10,
				autotestResultGenerator: autotestResultAlwaysFail,

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
				testAllowRetry:          true,
				autotestResultGenerator: autotestResultAlwaysFail,

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
				testAllowRetry:          true,
				testMaxRetry:            7,
				autotestResultGenerator: autotestResultAlwaysFail,

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
				testAllowRetry:          true,
				testMaxRetry:            5,
				autotestResultGenerator: autotestResultAlwaysFail,

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
				testAllowRetry:          true,
				autotestResultGenerator: autotestResultAlwaysFail,

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
				autotestResultGenerator: generateAutotestResultsFromSlice([]*skylab_test_runner.Result_Autotest{
					{
						TestCases: []*skylab_test_runner.Result_Autotest_TestCase{
							{Name: "foo", Verdict: skylab_test_runner.Result_Autotest_TestCase_VERDICT_FAIL},
						},
					},
					{
						TestCases: []*skylab_test_runner.Result_Autotest_TestCase{
							{Name: "foo", Verdict: skylab_test_runner.Result_Autotest_TestCase_VERDICT_PASS},
						},
					},
				}),

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
				skylab.setAutotestResultGenerator(c.autotestResultGenerator)
				params.Retry = c.retryParams
				for _, inv := range c.invocations {
					inv.Test.AllowRetries = c.testAllowRetry
					inv.Test.MaxRetries = c.testMaxRetry
				}
				var ml memlogger.MemLogger
				ctx = logging.SetFactory(ctx, func(context.Context) logging.Logger { return &ml })
				ts, err := execution.NewRequestTaskSet(c.invocations, params, basicConfig(), fakeTaskSetConfig)
				So(err, ShouldBeNil)
				run := execution.NewRunnerWithRequestTaskSets(ts)
				err = run.LaunchAndWait(ctx, skylab)
				So(err, ShouldBeNil)
				resp := getSingleResponse(run)

				Convey("each attempt request should have a unique logdog url in the.", func() {
					s := map[string]bool{}
					for _, req := range skylab.launchCalls {
						url, ok := extractLogdogUrlFromCommand(req.Cmd.Args())
						So(ok, ShouldBeTrue)
						s[url] = true
					}
					So(s, ShouldHaveLength, len(skylab.launchCalls))
				})
				// TODO(crbug.com/1003874, pprabhu) This test case is in the wrong place.
				// Once the hack to manipulate logdog URL is removed, this block can also be dropped.
				Convey("the logdog url in the command and in tags should match.", func() {
					for _, req := range skylab.launchCalls {
						cmdURL, _ := extractLogdogUrlFromCommand(req.Cmd.Args())
						tagURL := extractLogdogUrlFromTags(req.SwarmingTags)
						So(cmdURL, ShouldEqual, tagURL)
					}
				})
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

func extractLogdogUrlFromCommand(command []string) (string, bool) {
	for i, s := range command[:len(command)-1] {
		if s == "-logdog-annotation-url" {
			return command[i+1], true
		}
	}
	return "", false
}

func extractLogdogUrlFromTags(tags []string) string {
	for _, s := range tags {
		if strings.HasPrefix(s, "log_location:") {
			return s[len("log_location:"):]
		}
	}
	return ""
}

func TestClientTestArg(t *testing.T) {
	Convey("Given a client test", t, func() {
		ctx := context.Background()
		skylab := newFakeSkylab()

		invs := []*steps.EnumerationResponse_AutotestInvocation{clientTestInvocation("name1", "")}

		ts, err := execution.NewRequestTaskSet(invs, basicParams(), basicConfig(), fakeTaskSetConfig)
		So(err, ShouldBeNil)
		run := execution.NewRunnerWithRequestTaskSets(ts)

		err = run.LaunchAndWait(ctx, skylab)
		So(err, ShouldBeNil)

		Convey("the launched task request should have correct parameters.", func() {
			So(skylab.launchCalls, ShouldHaveLength, 1)
			So(skylab.launchCalls[0].Cmd.ClientTest, ShouldBeTrue)
		})
	})
}

func TestQuotaSchedulerAccount(t *testing.T) {
	Convey("Given a client test and a quota account set in legacy way", t, func() {
		ctx := context.Background()
		skylab := newFakeSkylab()
		invs := []*steps.EnumerationResponse_AutotestInvocation{serverTestInvocation("name1", "")}
		params := basicParams()
		params.Scheduling = &test_platform.Request_Params_Scheduling{
			Pool: &test_platform.Request_Params_Scheduling_QuotaAccount{
				QuotaAccount: "foo-account",
			},
			// TODO(linxinan@) Drop this test once the migration is done. Use the deprecated
			// field "QuotaAccount" only when QsAccount is empty. This test ensures that
			// the legacy user will not be interrupted.
			QsAccount: "",
		}
		ts, err := execution.NewRequestTaskSet(invs, params, basicConfig(), fakeTaskSetConfig)
		So(err, ShouldBeNil)
		run := execution.NewRunnerWithRequestTaskSets(ts)

		err = run.LaunchAndWait(ctx, skylab)
		So(err, ShouldBeNil)

		Convey("the launched task request should have a tag specifying the correct quota account and run in the quota pool.", func() {
			So(skylab.launchCalls, ShouldHaveLength, 1)
			launchArgs := skylab.launchCalls[0]
			So(launchArgs.SwarmingTags, ShouldContain, "qs_account:foo-account")
			So(launchArgs.SchedulableLabels.GetSelfServePools(), ShouldHaveLength, 0)
			So(launchArgs.SchedulableLabels.GetCriticalPools(), ShouldHaveLength, 1)
			So(launchArgs.SchedulableLabels.GetCriticalPools()[0], ShouldEqual, inventory.SchedulableLabels_DUT_POOL_QUOTA)
		})
	})
}

func TestQuotaSchedulerAccountOnQSAccount(t *testing.T) {
	Convey("Given a client test and a quota account", t, func() {
		ctx := context.Background()
		skylab := newFakeSkylab()
		invs := []*steps.EnumerationResponse_AutotestInvocation{serverTestInvocation("name1", "")}
		params := basicParams()
		params.Scheduling = &test_platform.Request_Params_Scheduling{
			Pool: &test_platform.Request_Params_Scheduling_UnmanagedPool{
				UnmanagedPool: "foo-pool",
			},
			QsAccount: "foo-account",
		}
		ts, err := execution.NewRequestTaskSet(invs, params, basicConfig(), fakeTaskSetConfig)
		So(err, ShouldBeNil)
		run := execution.NewRunnerWithRequestTaskSets(ts)

		err = run.LaunchAndWait(ctx, skylab)
		So(err, ShouldBeNil)
		Convey("the launched task request should have a tag specifying the correct quota account and run in foo-pool.", func() {
			So(skylab.launchCalls, ShouldHaveLength, 1)
			launchArgs := skylab.launchCalls[0]
			So(launchArgs.SwarmingTags, ShouldContain, "qs_account:foo-account")
			So(launchArgs.SchedulableLabels.GetSelfServePools(), ShouldHaveLength, 1)
			So(launchArgs.SchedulableLabels.GetCriticalPools(), ShouldHaveLength, 0)
			So(launchArgs.SchedulableLabels.GetSelfServePools()[0], ShouldEqual, "foo-pool")
		})
	})
}

func TestReservedTagShouldNotBeSetByUsers(t *testing.T) {
	Convey("Given a client test and a fake quota account set by user", t, func() {
		ctx := context.Background()
		skylab := newFakeSkylab()
		invs := []*steps.EnumerationResponse_AutotestInvocation{serverTestInvocation("name1", "")}
		params := basicParams()
		params.Scheduling = &test_platform.Request_Params_Scheduling{
			Pool: &test_platform.Request_Params_Scheduling_ManagedPool_{
				ManagedPool: test_platform.Request_Params_Scheduling_MANAGED_POOL_QUOTA,
			},
			QsAccount: "real-account",
		}
		params.Decorations = &test_platform.Request_Params_Decorations{
			Tags: []string{"qs_account:fake-account"},
		}
		ts, err := execution.NewRequestTaskSet(invs, params, basicConfig(), fakeTaskSetConfig)
		So(err, ShouldBeNil)
		run := execution.NewRunnerWithRequestTaskSets(ts)

		err = run.LaunchAndWait(ctx, skylab)
		So(err, ShouldBeNil)

		Convey("the launched task request should have a tag specifying the correct quota account and run in the quota pool.", func() {
			So(skylab.launchCalls, ShouldHaveLength, 1)
			launchArgs := skylab.launchCalls[0]
			So(launchArgs.SwarmingTags, ShouldContain, "qs_account:real-account")
			So(launchArgs.SchedulableLabels.GetSelfServePools(), ShouldHaveLength, 0)
			So(launchArgs.SchedulableLabels.GetCriticalPools(), ShouldHaveLength, 1)
			So(launchArgs.SchedulableLabels.GetCriticalPools()[0], ShouldEqual, inventory.SchedulableLabels_DUT_POOL_QUOTA)
		})
	})
}

func TestRequestShouldNotSetBothQSAccountAndQuotaAccount(t *testing.T) {
	Convey("Given a client test with both quota account and priority set", t, func() {
		ctx := context.Background()
		skylab := newFakeSkylab()
		invs := []*steps.EnumerationResponse_AutotestInvocation{serverTestInvocation("name1", "")}
		params := basicParams()
		params.Scheduling = &test_platform.Request_Params_Scheduling{
			Pool: &test_platform.Request_Params_Scheduling_QuotaAccount{
				QuotaAccount: "foo-account",
			},
			QsAccount: "foo-account",
		}
		ts, err := execution.NewRequestTaskSet(invs, params, basicConfig(), fakeTaskSetConfig)
		So(err, ShouldBeNil)
		run := execution.NewRunnerWithRequestTaskSets(ts)
		Convey("The test should end up with a panic.", func() {
			So(func() { run.LaunchAndWait(ctx, skylab) }, ShouldPanic)
		})
	})
}

func TestRequestShouldNotSetBothQSAccountAndPriority(t *testing.T) {
	Convey("Given a client test with both quota account and priority set", t, func() {
		ctx := context.Background()
		skylab := newFakeSkylab()
		invs := []*steps.EnumerationResponse_AutotestInvocation{serverTestInvocation("name1", "")}
		params := basicParams()
		params.Scheduling = &test_platform.Request_Params_Scheduling{
			Pool: &test_platform.Request_Params_Scheduling_UnmanagedPool{
				UnmanagedPool: "foo-pool",
			},
			QsAccount: "foo-account",
			Priority:  50,
		}
		ts, err := execution.NewRequestTaskSet(invs, params, basicConfig(), fakeTaskSetConfig)
		So(err, ShouldBeNil)
		run := execution.NewRunnerWithRequestTaskSets(ts)
		Convey("The test should end up with a panic.", func() {
			So(func() { run.LaunchAndWait(ctx, skylab) }, ShouldPanic)
		})
	})
}

func TestUnmanagedPool(t *testing.T) {
	Convey("Given a client test and an unmanaged pool.", t, func() {
		ctx := context.Background()
		skylab := newFakeSkylab()
		invs := []*steps.EnumerationResponse_AutotestInvocation{serverTestInvocation("name1", "")}
		params := basicParams()
		params.Scheduling.Pool = &test_platform.Request_Params_Scheduling_UnmanagedPool{
			UnmanagedPool: "foo-pool",
		}

		ts, err := execution.NewRequestTaskSet(invs, params, basicConfig(), fakeTaskSetConfig)
		So(err, ShouldBeNil)
		run := execution.NewRunnerWithRequestTaskSets(ts)

		err = run.LaunchAndWait(ctx, skylab)
		So(err, ShouldBeNil)

		Convey("the launched task request run in the unmanaged pool.", func() {
			So(skylab.launchCalls, ShouldHaveLength, 1)
			launchArgs := skylab.launchCalls[0]
			So(launchArgs.SchedulableLabels.GetCriticalPools(), ShouldHaveLength, 0)
			So(launchArgs.SchedulableLabels.GetSelfServePools(), ShouldHaveLength, 1)
			So(launchArgs.SchedulableLabels.GetSelfServePools()[0], ShouldEqual, "foo-pool")
		})
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

		skylab := newFakeSkylab()
		invs := []*steps.EnumerationResponse_AutotestInvocation{serverTestInvocation("name1", "")}
		params := basicParams()

		taskSets, err := execution.NewRequestTaskSet(invs, params, basicConfig(), fakeTaskSetConfig)
		So(err, ShouldBeNil)
		run := execution.NewRunnerWithRequestTaskSets(taskSets)

		// TODO(crbug.com/1001746, akeshet) Fix this test.
		// This test is broken even after adding locks around testRun.attempts because it is possible that the
		// assertions at the end are run before LaunchAndWait() does anything. That is not the intent of this test.
		SkipConvey("when tests are still running, response verdict is correct.", func() {
			skylab.setLifeCycle(test_platform.TaskState_LIFE_CYCLE_RUNNING)

			wg := sync.WaitGroup{}
			defer wg.Wait()
			defer cancel()
			wg.Add(1)
			go func() {
				defer wg.Done()
				// Can't verify error returned is nil because Convey() doesn't
				// like assertions in goroutines.
				_ = run.LaunchAndWait(ctx, skylab)
			}()

			resp := getSingleResponse(run)
			So(resp.State.LifeCycle, ShouldEqual, test_platform.TaskState_LIFE_CYCLE_RUNNING)
			So(resp.State.Verdict, ShouldEqual, test_platform.TaskState_VERDICT_UNSPECIFIED)
		})

		Convey("when the test passed, response verdict is correct.", func() {
			skylab.setAutotestResultGenerator(autotestResultAlwaysPass)
			run.LaunchAndWait(ctx, skylab)
			resp := getSingleResponse(run)
			So(resp.State.LifeCycle, ShouldEqual, test_platform.TaskState_LIFE_CYCLE_COMPLETED)
			So(resp.State.Verdict, ShouldEqual, test_platform.TaskState_VERDICT_PASSED)
		})

		Convey("when the test failed, response verdict is correct.", func() {
			skylab.setAutotestResultGenerator(autotestResultAlwaysFail)
			run.LaunchAndWait(ctx, skylab)
			resp := getSingleResponse(run)
			So(resp.State.LifeCycle, ShouldEqual, test_platform.TaskState_LIFE_CYCLE_COMPLETED)
			So(resp.State.Verdict, ShouldEqual, test_platform.TaskState_VERDICT_FAILED)
		})

		Convey("when execution is aborted (e.g., timeout), response verdict is correct.", func() {
			skylab.setLifeCycle(test_platform.TaskState_LIFE_CYCLE_RUNNING)

			wg := sync.WaitGroup{}
			wg.Add(1)
			var err error
			go func() {
				err = run.LaunchAndWait(ctx, skylab)
				wg.Done()
			}()

			cancel()
			wg.Wait()
			So(err, ShouldBeNil)

			resp := getSingleResponse(run)
			So(resp.State.LifeCycle, ShouldEqual, test_platform.TaskState_LIFE_CYCLE_ABORTED)
			So(resp.State.Verdict, ShouldEqual, test_platform.TaskState_VERDICT_FAILED)
		})
	})
}

func TestIncompatibleDependencies(t *testing.T) {

	Convey("In testing context", t, func() {
		cases := []struct {
			Tag    string
			Params *test_platform.Request_Params
			Invs   []*steps.EnumerationResponse_AutotestInvocation
		}{
			{
				Tag: "incompatible build target between enumeration and request",
				Params: &test_platform.Request_Params{
					SoftwareAttributes: &test_platform.Request_Params_SoftwareAttributes{
						BuildTarget: &chromiumos.BuildTarget{Name: "requested"},
					},
					Time: &test_platform.Request_Params_Time{
						MaximumDuration: &duration.Duration{Seconds: 3600},
					},
				},
				Invs: []*steps.EnumerationResponse_AutotestInvocation{
					testInvocationWithDependency("some_test", "board:enumerated"),
				},
			},
			{
				Tag: "incompatible model between enumeration and request",
				Params: &test_platform.Request_Params{
					HardwareAttributes: &test_platform.Request_Params_HardwareAttributes{
						Model: "requested",
					},
					Time: &test_platform.Request_Params_Time{
						MaximumDuration: &duration.Duration{Seconds: 3600},
					},
				},
				Invs: []*steps.EnumerationResponse_AutotestInvocation{
					testInvocationWithDependency("some_test", "model:enumerated"),
				},
			},
			{
				Tag:    "unsupported dependencies",
				Params: basicParams(),
				Invs: []*steps.EnumerationResponse_AutotestInvocation{
					testInvocationWithDependency("some_test", "some_unsupported_dependency"),
				},
			},
		}

		ctx := context.Background()
		skylab := newFakeSkylab()
		for _, c := range cases {
			Convey(fmt.Sprintf("with %s", c.Tag), func() {
				ts, err := execution.NewRequestTaskSet(c.Invs, c.Params, basicConfig(), fakeTaskSetConfig)
				So(err, ShouldBeNil)
				run := execution.NewRunnerWithRequestTaskSets(ts)
				err = run.LaunchAndWait(ctx, skylab)
				So(err, ShouldBeNil)

				resp := getSingleResponse(run)
				So(resp, ShouldNotBeNil)

				Convey("then task result is rejected with unspecified verdict.", func() {
					So(resp.TaskResults, ShouldHaveLength, 1)
					tr := resp.TaskResults[0]
					So(tr.State.LifeCycle, ShouldEqual, test_platform.TaskState_LIFE_CYCLE_REJECTED)
					So(tr.State.Verdict, ShouldEqual, test_platform.TaskState_VERDICT_UNSPECIFIED)

				})
				Convey("and overall result is complete with failed verdict.", func() {
					So(resp.State.LifeCycle, ShouldEqual, test_platform.TaskState_LIFE_CYCLE_COMPLETED)
					So(resp.State.Verdict, ShouldEqual, test_platform.TaskState_VERDICT_FAILED)
				})
				Convey("and no skylab swarming tasks are created.", func() {
					So(skylab.launchCalls, ShouldHaveLength, 0)
					So(skylab.numResultsCalls, ShouldEqual, 0)
				})
			})
		}
	})
}

func testInvocationWithDependency(name string, deps ...string) *steps.EnumerationResponse_AutotestInvocation {
	inv := steps.EnumerationResponse_AutotestInvocation{
		Test: &build_api.AutotestTest{
			Name:                 name,
			ExecutionEnvironment: build_api.AutotestTest_EXECUTION_ENVIRONMENT_SERVER,
		},
	}
	for _, d := range deps {
		inv.Test.Dependencies = append(inv.Test.Dependencies, &build_api.AutotestTaskDependency{Label: d})
	}
	return &inv
}

func getSingleResponse(r *execution.Runner) *steps.ExecuteResponse {
	resps := make([]*steps.ExecuteResponse, 0, 1)
	for _, resp := range r.Responses() {
		resps = append(resps, resp)
	}
	So(resps, ShouldHaveLength, 1)
	return resps[0]
}
