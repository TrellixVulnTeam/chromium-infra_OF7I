// Copyright 2022 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"go.chromium.org/luci/resultdb/pbutil"
	pb "go.chromium.org/luci/resultdb/proto/v1"
	sinkpb "go.chromium.org/luci/resultdb/sink/proto/v1"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func TestGTestJsonConversions(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	Convey(`From JSON works`, t, func() {
		str := `{
				"testsuites": [
					{
						"testsuite": [
							{
								"name": "TestDoBar",
								"classname": "FooTest",
								"time": "0.12s",
								"result": "COMPLETED",
								"failures":[{
									"failure":"file.cc:5\nThis is a failure message."
								}]
							},
							{
								"name": "TestDoBarDisabled",
								"classname": "FooTest",
								"time": "0s",
								"result": "SUPPRESSED"
							}
						]
					},
					{
						"testsuite": [
							{
								"name": "TestDoBaz1",
								"classname": "FooTest",
								"time": "37s",
								"result": "SKIPPED"
							},
							{
								"name": "TestDoBaz2",
								"classname": "FooTest",
								"time": "6s",
								"result": "COMPLETED",
								"failures":[{}]
							}
						]
					}
				]
			}`

		results := &GTestJsonResults{}
		err := results.ConvertFromJSON(strings.NewReader(str))
		So(err, ShouldBeNil)
		So(results.TestSuites, ShouldResemble, []*GTestJsonTestSuites{
			{
				TestSuite: []*GTestJsonTestSuite{
					{
						ClassName: "FooTest",
						Name:      "TestDoBar",
						Result:    "COMPLETED",
						Time:      "0.12s",
						Failures: []*GTestJsonFailures{
							{
								Failure: "file.cc:5\nThis is a failure message.",
							},
						},
					},
					{
						ClassName: "FooTest",
						Name:      "TestDoBarDisabled",
						Result:    "SUPPRESSED",
						Time:      "0s",
					},
				},
			},
			{
				TestSuite: []*GTestJsonTestSuite{
					{
						ClassName: "FooTest",
						Name:      "TestDoBaz1",
						Result:    "SKIPPED",
						Time:      "37s",
					},
					{
						ClassName: "FooTest",
						Name:      "TestDoBaz2",
						Result:    "COMPLETED",
						Time:      "6s",
						Failures:  []*GTestJsonFailures{{}},
					},
				},
			},
		})
	})

	Convey("convertTestResult", t, func() {
		convert := func(result *GTestJsonTestSuite) *sinkpb.TestResult {
			r := &GTestJsonResults{}
			buf := &bytes.Buffer{}
			tr, err := r.convertTestResult(ctx, buf, "testId", result)
			So(err, ShouldBeNil)
			return tr
		}

		Convey("SUPPRESSED", func() {
			tr := convert(&GTestJsonTestSuite{Result: "SUPPRESSED"})
			So(tr.Status, ShouldEqual, pb.TestStatus_SKIP)
			So(tr.Expected, ShouldBeTrue)
		})

		Convey("Duration", func() {
			tr := convert(&GTestJsonTestSuite{
				Result: "COMPLETED",
				Time:   "1s",
			})
			So(tr.Duration.GetSeconds(), ShouldEqual, 1)
			So(tr.Duration.GetNanos(), ShouldEqual, 0)
		})

		Convey("failure reason", func() {
			Convey("first failure takes precedence", func() {
				tr := convert(&GTestJsonTestSuite{
					Result: "COMPLETED",
					Failures: []*GTestJsonFailures{
						{
							Failure: "This is a failure message.",
						},
						{
							Failure: "This is a second failure message.",
						},
					},
				})
				So(tr.FailureReason.PrimaryErrorMessage, ShouldEqual, `This is a failure message.`)
			})
			Convey("empty", func() {
				tr := convert(&GTestJsonTestSuite{
					Result:   "COMPLETED",
					Failures: []*GTestJsonFailures{},
				})
				So(tr.FailureReason, ShouldEqual, nil)
			})
		})
	})

	Convey(`ToProtos`, t, func() {
		Convey("Works", func() {
			results := &GTestJsonResults{
				TestSuites: []*GTestJsonTestSuites{
					{
						TestSuite: []*GTestJsonTestSuite{
							{
								ClassName: "BazTest",
								Name:      "DoesQux1",
								Result:    "COMPLETED",
							},
							{
								ClassName: "BazTest",
								Name:      "DoesQux2",
								Result:    "COMPLETED",
								Failures: []*GTestJsonFailures{
									{
										Failure: "file.cc:5\nThis is a failure message.",
									},
								},
							},
							{
								ClassName: "FooTest",
								Name:      "TestDoBarDisabled",
								Result:    "SUPPRESSED",
							},
						},
					},
					{
						TestSuite: []*GTestJsonTestSuite{
							{
								ClassName: "FooTest",
								Name:      "DoesBar",
								Result:    "SKIPPED",
							},
						},
					},
				},
			}

			testResults, err := results.ToProtos(ctx)
			So(err, ShouldBeNil)

			expected := []*sinkpb.TestResult{
				{
					TestId:   "BazTest.DoesQux1",
					Expected: true,
					Status:   pb.TestStatus_PASS,
					Tags: pbutil.StringPairs(
						"test_name", "BazTest.DoesQux1",
					),
					TestMetadata: &pb.TestMetadata{
						Name: "BazTest.DoesQux1",
					},
				},
				{
					TestId: "BazTest.DoesQux2",
					Status: pb.TestStatus_FAIL,
					Tags: pbutil.StringPairs(
						"test_name", "BazTest.DoesQux2",
					),
					TestMetadata: &pb.TestMetadata{
						Name: "BazTest.DoesQux2",
						Location: &pb.TestLocation{
							Repo:     "https://webrtc.googlesource.com/src/",
							FileName: "//file.cc",
							Line:     5,
						},
					},
					FailureReason: &pb.FailureReason{
						PrimaryErrorMessage: "file.cc:5\nThis is a failure message.",
					},
					SummaryHtml: "<p>file.cc:5\nThis is a failure message.</p>",
				},
				{
					TestId:   "FooTest.TestDoBarDisabled",
					Expected: true,
					Status:   pb.TestStatus_SKIP,
					Tags: pbutil.StringPairs(
						"test_name", "FooTest.TestDoBarDisabled",
						"disabled_test", "true",
					),
					TestMetadata: &pb.TestMetadata{
						Name: "FooTest.TestDoBarDisabled",
					},
				},
				{
					TestId: "FooTest.DoesBar",
					Status: pb.TestStatus_SKIP,
					Tags: pbutil.StringPairs(
						"test_name", "FooTest.DoesBar",
					),
					TestMetadata: &pb.TestMetadata{
						Name: "FooTest.DoesBar",
					},
				},
			}
			So(testResults, ShouldHaveLength, len(expected))
			for i := range testResults {
				So(testResults[i], ShouldResembleProto, expected[i])
			}
		})
	})
}
