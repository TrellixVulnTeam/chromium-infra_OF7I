// Copyright 2021 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"

	pb "go.chromium.org/luci/resultdb/proto/v1"
	sinkpb "go.chromium.org/luci/resultdb/sink/proto/v1"
)

// Following CrOS test_runner's convention, test_case represents a single test
// executed in an Autotest run. Described in
// http://cs/chromeos_public/infra/proto/src/test_platform/skylab_test_runner/result.proto
// Fields not used by ResultSink Test Results are omitted.
type TestRunnerResult struct {
	Autotest TestRunnerAutotest `json:"autotest_result"`
}

type TestRunnerAutotest struct {
	TestCases []TestRunnerTestCase `json:"test_cases"`
}

type TestRunnerTestCase struct {
	Name                 string `json:"name"`
	Verdict              string `json:"verdict"`
	HumanReadableSummary string `json:"human_readable_summary"`
}

// ConvertFromJSON reads the provided reader into the receiver.
//
// The receiver is cleared and its fields overwritten.
func (r *TestRunnerResult) ConvertFromJSON(reader io.Reader) error {
	*r = TestRunnerResult{}
	if err := json.NewDecoder(reader).Decode(r); err != nil {
		return err
	}
	return nil
}

// ToProtos converts test results in r to []*sinkpb.TestResult.
func (r *TestRunnerResult) ToProtos(ctx context.Context) ([]*sinkpb.TestResult, error) {
	var ret []*sinkpb.TestResult
	for _, c := range r.Autotest.TestCases {
		status := genTestCaseStatus(c)
		tr := &sinkpb.TestResult{
			TestId: c.Name,
			// Phosphorus treats any status other than PASS as a failure.
			// http://cs/chromeos_public/infra/recipes/recipe_modules/dut_interface/phosphorus_results.py?l=100
			Expected: status == pb.TestStatus_PASS,
			Status:   status,
		}
		if c.HumanReadableSummary != "" {
			tr.SummaryHtml = fmt.Sprintf("<pre>%s</pre>", html.EscapeString(c.HumanReadableSummary))
		}
		ret = append(ret, tr)
	}
	return ret, nil
}

// Convert a TestCase Verdict into a ResultSink Status.
func genTestCaseStatus(c TestRunnerTestCase) pb.TestStatus {
	if c.Verdict == "VERDICT_PASS" {
		return pb.TestStatus_PASS
	}
	if c.Verdict == "VERDICT_NO_VERDICT" {
		return pb.TestStatus_SKIP
	}
	return pb.TestStatus_FAIL
}
