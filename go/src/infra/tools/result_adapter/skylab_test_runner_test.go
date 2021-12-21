// Copyright 2021 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package main

import (
	"context"
	"strings"
	"testing"

	pb "go.chromium.org/luci/resultdb/proto/v1"
	sinkpb "go.chromium.org/luci/resultdb/sink/proto/v1"

	. "github.com/smartystreets/goconvey/convey"
)

func TestSkylabTestRunnerConversions(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tc := []TestRunnerTestCase{{
		Name:    "test1",
		Verdict: "VERDICT_PASS",
	},
		{
			Name:    "test2",
			Verdict: "NO_VERDICT",
		},
		{
			Name:                 "test3",
			Verdict:              "VERDICT_FAIL",
			HumanReadableSummary: "test failure",
		},
	}

	results := TestRunnerResult{Autotest: TestRunnerAutotest{
		TestCases: tc,
	}}

	Convey(`From JSON works`, t, func() {
		str := `{
			"autotest_result": {
			  "test_cases": [
				{
				  "verdict": "VERDICT_PASS",
				  "name": "test1"
				},
				{
				  "verdict": "NO_VERDICT",
				  "name": "test2"
				},
				{
				  "verdict": "VERDICT_FAIL",
				  "name": "test3",
				  "human_readable_summary": "test failure"
				}
			  ]
			}
		  }`

		results := &TestRunnerResult{}
		err := results.ConvertFromJSON(strings.NewReader(str))
		So(err, ShouldBeNil)
		So(results.Autotest.TestCases, ShouldResemble, tc)
	})

	Convey(`ToProtos`, t, func() {
		Convey("test passes", func() {

			testResults, err := results.ToProtos(ctx)
			So(err, ShouldBeNil)

			expected := []*sinkpb.TestResult{
				{
					TestId:   "test1",
					Expected: true,
					Status:   pb.TestStatus_PASS,
				},
				{
					TestId:   "test2",
					Expected: false,
					Status:   pb.TestStatus_FAIL,
				},
				{
					TestId:      "test3",
					Expected:    false,
					Status:      pb.TestStatus_FAIL,
					SummaryHtml: "<pre>test failure</pre>",
				},
			}
			So(testResults, ShouldHaveLength, 3)
			So(testResults, ShouldResemble, expected)
		})
	})
}
