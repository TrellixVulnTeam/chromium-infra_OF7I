// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package transform

import (
	"context"
	"fmt"

	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/golang/protobuf/ptypes/timestamp"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/analytics"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/result_flow"
	runner "go.chromium.org/chromiumos/infra/proto/go/test_platform/skylab_test_runner"
	bbpb "go.chromium.org/luci/buildbucket/proto"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
)

// TestRunnerBuild contains test build data and can be transformed to
// analytics.TestRun and analytics.TestCaseResult.
type TestRunnerBuild interface {
	ToTestRun(context.Context) *analytics.TestRun
	ToTestCaseResults() []*analytics.TestCaseResult
}

type testRunnerBuild struct {
	bb         *result_flow.BuildbucketConfig
	id         int64
	parentID   string
	status     bbpb.Status
	createTime *timestamp.Timestamp
	startTime  *timestamp.Timestamp
	endTime    *timestamp.Timestamp
	model      string

	req *runner.Request
	res *runner.Result
}

// LoadTestRunnerBuild loads a Test Runner build from Buildbucket response.
func LoadTestRunnerBuild(ctx context.Context, parentUID string, b *bbpb.Build, bb *result_flow.BuildbucketConfig) (TestRunnerBuild, error) {
	if b == nil {
		return nil, fmt.Errorf("empty build")
	}
	c := &testRunnerBuild{
		bb:         bb,
		id:         b.GetId(),
		parentID:   parentUID,
		status:     b.GetStatus(),
		createTime: b.GetCreateTime(),
		startTime:  b.GetStartTime(),
		endTime:    b.GetEndTime(),
	}
	var err error
	prop := b.GetInput().GetProperties()
	if prop == nil {
		return nil, fmt.Errorf("Build has no input properties found, %d", b.GetId())
	}
	setDefaultStructValues(prop)
	op := prop.GetFields()
	if rValue, ok := op["request"]; ok {
		if c.req, err = structPBToTestRunnerRequest(rValue); err != nil {
			return nil, errors.Annotate(err, "failed to extract test runner request").Err()
		}
	}
	v, ok := getOutputPropertiesValue(b, "compressed_result")
	if !ok {
		logging.Infof(ctx, "Build has no output properties yet, %d", b.GetId())
		return c, nil
	}

	if c.model = fetchModelName(b); c.model == "" {
		logging.Infof(ctx, "failed to find label-model from build %d", b.GetId())
	}
	setDefaultStructValues(b.GetOutput().GetProperties())
	c.res, err = extractTestRunnerResult(v)
	return c, err
}

func structPBToTestRunnerRequest(from *structpb.Value) (*runner.Request, error) {
	res, err := unmarshalStructPB(from, &runner.Request{})
	if err != nil {
		return nil, errors.Annotate(err, "struct PB to TestRunner request").Err()
	}
	request, ok := res.(*runner.Request)
	if !ok {
		return nil, errors.Annotate(err, "struct PB to TestRunner request").Err()
	}
	return request, nil
}

func extractTestRunnerResult(rs *structpb.Value) (*runner.Result, error) {
	if b64 := rs.GetStringValue(); b64 != "" {
		pb, err := unmarshalCompressedString(b64, &runner.Result{})
		if err != nil {
			return nil, errors.Annotate(err, "extract TestRunner Build Data").Err()
		}
		result, ok := pb.(*runner.Result)
		if !ok {
			return nil, errors.Annotate(err, "extract TestRunner Build Data").Err()
		}
		return result, nil
	}
	return nil, fmt.Errorf("Failed to find result field")
}

// ToTestRun derives a TestRun entity from a test runner build.
func (c *testRunnerBuild) ToTestRun(ctx context.Context) *analytics.TestRun {
	r := &analytics.TestRun{
		BuildId:      c.id,
		DisplayName:  c.req.GetTest().GetAutotest().GetDisplayName(),
		ExecutionUrl: inferExecutionURL(c.bb, c.id),
		ParentUid:    c.parentID,
		Timeline: &analytics.Timeline{
			CreateTime: c.createTime,
			StartTime:  c.startTime,
			EndTime:    c.endTime,
		},
		Model:  c.model,
		Status: inferTestRunStatus(c.status),
	}
	if c.res != nil {
		r.FullLogUrl = c.res.GetAutotestResult().GetSynchronousLogDataUrl()
		r.Verdict = &analytics.Verdict{
			Value: inferTestRunVerdict(c.status, c.res),
		}
		r.Prejob =
			&analytics.TestRun_Prejob{
				Verdict: &analytics.Verdict{
					Value: inferPrejobVerdict(c.res),
				},
			}
	}
	return r
}

func inferTestRunStatus(s bbpb.Status) *analytics.Status {
	// analytics.Status is agnostic of whether the task was
	// successful - that information is contained in analytics.Verdict.
	m := map[bbpb.Status]string{
		bbpb.Status_STATUS_UNSPECIFIED: "UNKNOWN",
		bbpb.Status_SCHEDULED:          "PENDING",
		bbpb.Status_STARTED:            "RUNNING",
		bbpb.Status_SUCCESS:            "COMPLETED",
		bbpb.Status_FAILURE:            "COMPLETED",
		bbpb.Status_INFRA_FAILURE:      "COMPLETED",
		bbpb.Status_CANCELED:           "CANCELLED",
	}
	return &analytics.Status{
		Value: m[s],
	}
}

func inferTestRunVerdict(status bbpb.Status, res *runner.Result) string {
	if int(status)&int(bbpb.Status_ENDED_MASK) == 0 {
		return ""
	}
	// The tasks that ran no test cases do not have a definite verdict.
	if res.GetAutotestResult().GetIncomplete() {
		return "FAILED"
	}

	if s := res.GetAutotestResult().GetTestCases(); len(s) > 0 {
		for _, v := range s {
			if v.GetVerdict() != runner.Result_Autotest_TestCase_VERDICT_PASS {
				return "FAILED"
			}
		}
		return "PASSED"
	}

	return "NO_VERDICT"
}

func inferPrejobVerdict(res *runner.Result) string {
	m := map[runner.Result_Prejob_Step_Verdict]string{
		runner.Result_Prejob_Step_VERDICT_UNDEFINED: "UNKNOWN",
		runner.Result_Prejob_Step_VERDICT_PASS:      "PASSED",
		runner.Result_Prejob_Step_VERDICT_FAIL:      "FAILED",
	}
	if s := res.GetPrejob().GetStep(); len(s) > 0 {
		return m[s[0].GetVerdict()]
	}
	return "UNKNOWN"
}

func (c *testRunnerBuild) ToTestCaseResults() []*analytics.TestCaseResult {
	if c.res == nil {
		return nil
	}
	var r []*analytics.TestCaseResult
	for _, v := range c.res.GetAutotestResult().GetTestCases() {
		r = append(r, &analytics.TestCaseResult{
			Uid:           fmt.Sprintf("testCaseResults/%d/%s", c.id, v.GetName()),
			DisplayName:   v.GetName(),
			ParentBuildId: c.id,
			Verdict: &analytics.Verdict{
				Value: v.GetVerdict().String(),
			},
			HumanReadableSummary: v.GetHumanReadableSummary(),
		})
	}
	return r
}

func fetchModelName(b *bbpb.Build) string {
	botDimensions := b.GetInfra().GetSwarming().GetBotDimensions()
	for _, d := range botDimensions {
		if d.GetKey() == "label-model" {
			return d.GetValue()
		}
	}
	return ""
}
