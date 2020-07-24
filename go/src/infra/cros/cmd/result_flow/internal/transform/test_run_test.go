// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package transform_test

import (
	"context"
	"fmt"
	"infra/cros/cmd/result_flow/internal/transform"
	"sort"
	"testing"

	"go.chromium.org/chromiumos/infra/proto/go/test_platform/analytics"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/result_flow"
	runner "go.chromium.org/chromiumos/infra/proto/go/test_platform/skylab_test_runner"

	structpb "github.com/golang/protobuf/ptypes/struct"
	. "github.com/smartystreets/goconvey/convey"
	bbpb "go.chromium.org/luci/buildbucket/proto"
)

var (
	fakeTestRunnerBuildID int64 = 8875806983290035888
	fakeBuildbucketConfig       = &result_flow.BuildbucketConfig{
		Host:    "fakehost",
		Project: "fakeProject",
		Bucket:  "fakeBucket",
		Builder: "fakeBuilder",
	}
)

func TestRunnerOutputToTestRun(t *testing.T) {
	cases := []struct {
		description   string
		in            *bbpb.Build
		wantTestRun   *analytics.TestRun
		wantTestCases []*analytics.TestCaseResult
	}{
		{
			"Given an ongoing test runner build",
			genFakeTestRunnerBuild("foo", bbpb.Status_STARTED, runner.Result_Prejob_Step_VERDICT_UNDEFINED, false),
			genFakeTestRun("foo", "", "RUNNING", ""),
			[]*analytics.TestCaseResult{},
		},
		{
			"Given a test runner build with failed prejob",
			genFakeTestRunnerBuild("foo", bbpb.Status_FAILURE, runner.Result_Prejob_Step_VERDICT_FAIL, false),
			genFakeTestRun("foo", "FAILED", "COMPLETED", "NO_VERDICT"),
			[]*analytics.TestCaseResult{},
		},
		{
			"Given a completed test runner build",
			genFakeTestRunnerBuild("hoo", bbpb.Status_SUCCESS, runner.Result_Prejob_Step_VERDICT_PASS, false,
				genFakeAutotestTestCase("foo", runner.Result_Autotest_TestCase_VERDICT_PASS),
				genFakeAutotestTestCase("hoo", runner.Result_Autotest_TestCase_VERDICT_FAIL),
			),
			genFakeTestRun("hoo", "PASSED", "COMPLETED", "FAILED"),
			[]*analytics.TestCaseResult{
				genFakeTestCaseResult("foo", "VERDICT_PASS"),
				genFakeTestCaseResult("hoo", "VERDICT_FAIL"),
			},
		},
		{
			"Given an incomplete test runner build",
			genFakeTestRunnerBuild("hoo", bbpb.Status_SUCCESS, runner.Result_Prejob_Step_VERDICT_PASS, true,
				genFakeAutotestTestCase("foo", runner.Result_Autotest_TestCase_VERDICT_PASS),
			),
			genFakeTestRun("hoo", "PASSED", "COMPLETED", "FAILED"),
			[]*analytics.TestCaseResult{
				genFakeTestCaseResult("foo", "VERDICT_PASS"),
			},
		},
	}
	ctx := context.Background()
	for _, c := range cases {
		Convey(c.description, t, func() {
			Convey("then test runner build is correctly converted to TestRun and TestCaseResult.", func() {
				build, _ := transform.LoadTestRunnerBuild(ctx, "TestPlanRuns/fake-build-id/fake-test-run", c.in, fakeBuildbucketConfig)
				got := build.ToTestRun()
				So(got, ShouldNotBeNil)
				checkTestRunEquality(got, c.wantTestRun)
				if c.wantTestCases != nil {
					cs := build.ToTestCaseResults()
					sort.Slice(cs, func(i, j int) bool { return cs[i].Uid < cs[j].Uid })
					for i := 0; i < len(cs); i++ {
						checkTestCaseEquality(cs[i], c.wantTestCases[i])
					}
				}
			})
		})
	}
}

func genFakeTestRun(label, prejobVerdict, lifeCycle, verdict string) *analytics.TestRun {
	return &analytics.TestRun{
		BuildId:     fakeTestRunnerBuildID,
		DisplayName: fmt.Sprintf("fake-test-run-%s", label),
		ExecutionUrl: fmt.Sprintf(
			"https://ci.chromium.org/p/%s/builders/%s/%s/b%d",
			fakeBuildbucketConfig.Project,
			fakeBuildbucketConfig.Bucket,
			fakeBuildbucketConfig.Builder,
			fakeTestRunnerBuildID,
		),
		ParentUid:  "TestPlanRuns/fake-build-id/fake-test-run",
		Timeline:   genAnalyticTimeline(),
		FullLogUrl: "gs://fakeLogURL",
		Prejob: &analytics.TestRun_Prejob{
			Verdict: &analytics.Verdict{
				Value: prejobVerdict,
			},
		},
		Status: &analytics.Status{
			Value: lifeCycle,
		},
		Verdict: &analytics.Verdict{
			Value: verdict,
		},
	}
}

func genFakeTestRunnerURL() string {
	return fmt.Sprintf(
		"https://ci.chromium.org/p/%s/builders/%s/%s/b%d",
		fakeBuildbucketConfig.Project,
		fakeBuildbucketConfig.Bucket,
		fakeBuildbucketConfig.Builder,
		fakeTestRunnerBuildID,
	)
}

func genFakeTestCaseResult(label, verdict string) *analytics.TestCaseResult {
	return &analytics.TestCaseResult{
		Uid:           fmt.Sprintf("testCaseResults/%d/%s/%s", fakeTestRunnerBuildID, label, "fake-test-case"),
		DisplayName:   fmt.Sprintf("%s/%s", label, "fake-test-case"),
		ParentBuildId: fakeTestRunnerBuildID,
		Verdict: &analytics.Verdict{
			Value: verdict,
		},
		HumanReadableSummary: fmt.Sprintf("fake summary for %s", label),
	}
}

func genFakeTestRunnerBuild(label string, status bbpb.Status, prejobVerdict runner.Result_Prejob_Step_Verdict, incomplete bool, cs ...*runner.Result_Autotest_TestCase) *bbpb.Build {
	res := &bbpb.Build{
		Id:         fakeTestRunnerBuildID,
		CreateTime: fakeCreateTime,
		StartTime:  fakeStartTime,
		EndTime:    fakeEndTime,
		Status:     status,
		Input: testRunnerRequestsToInputField(
			genFakeTestRunnerRequest(label),
		),
	}
	if int(status)&int(bbpb.Status_ENDED_MASK) != 0 {
		res.Output = pbToOutputField(
			genFakeTestRunnerResult(prejobVerdict, incomplete, cs...),
			"compressed_result",
		)
	}
	return res
}

func testRunnerRequestsToInputField(request *runner.Request) *bbpb.Build_Input {
	m, _ := marshalPB(request)
	return &bbpb.Build_Input{
		Properties: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"request": m,
			},
		},
	}
}

func genFakeTestRunnerRequest(label string) *runner.Request {
	return &runner.Request{
		Test: &runner.Request_Test{
			Harness: &runner.Request_Test_Autotest_{
				Autotest: &runner.Request_Test_Autotest{
					DisplayName: fmt.Sprintf("fake-test-run-%s", label),
				},
			},
		},
	}
}

func genFakeTestRunnerResult(prejobVerdict runner.Result_Prejob_Step_Verdict, incomplete bool, cs ...*runner.Result_Autotest_TestCase) *runner.Result {
	return &runner.Result{
		Prejob: &runner.Result_Prejob{
			Step: []*runner.Result_Prejob_Step{
				{
					Verdict: prejobVerdict,
				},
			},
		},
		Harness: &runner.Result_AutotestResult{
			AutotestResult: &runner.Result_Autotest{
				TestCases:  cs,
				Incomplete: incomplete,
			},
		},
	}
}

func genFakeAutotestTestCase(label string, verdict runner.Result_Autotest_TestCase_Verdict) *runner.Result_Autotest_TestCase {
	return &runner.Result_Autotest_TestCase{
		Name:                 fmt.Sprintf("%s/%s", label, "fake-test-case"),
		HumanReadableSummary: fmt.Sprintf("fake summary for %s", label),
		Verdict:              verdict,
	}
}

func checkTestRunEquality(want, got *analytics.TestRun) {
	So(got.BuildId, ShouldEqual, want.BuildId)
	So(got.DisplayName, ShouldEqual, want.DisplayName)
	So(got.ExecutionUrl, ShouldEqual, want.ExecutionUrl)
	So(got.ParentUid, ShouldEqual, want.ParentUid)
	So(got.GetVerdict().GetValue(), ShouldEqual, want.GetVerdict().GetValue())
	So(got.GetStatus().GetValue(), ShouldEqual, want.GetStatus().GetValue())
	So(got.GetPrejob().GetVerdict().GetValue(), ShouldEqual, want.GetPrejob().GetVerdict().GetValue())
	So(got.Timeline, ShouldResemble, want.Timeline)
}

func checkTestCaseEquality(want, got *analytics.TestCaseResult) {
	So(got.Uid, ShouldEqual, want.Uid)
	So(got.DisplayName, ShouldEqual, want.DisplayName)
	So(got.ParentBuildId, ShouldEqual, want.ParentBuildId)
	So(got.GetVerdict().GetValue(), ShouldEqual, want.GetVerdict().GetValue())
	So(got.HumanReadableSummary, ShouldEqual, want.HumanReadableSummary)
}
