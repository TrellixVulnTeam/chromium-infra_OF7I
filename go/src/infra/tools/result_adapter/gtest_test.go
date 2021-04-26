// Copyright 2020 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"go.chromium.org/luci/resultdb/pbutil"
	pb "go.chromium.org/luci/resultdb/proto/v1"
	sinkpb "go.chromium.org/luci/resultdb/sink/proto/v1"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

const snippetInSummaryHtml = `<p><text-artifact artifact-id="snippet" /></p>`

func TestGTestConversions(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	textArt := func(s string) *sinkpb.Artifact {
		a := &sinkpb.Artifact{
			Body:        &sinkpb.Artifact_Contents{Contents: []byte(s)},
			ContentType: "text/plain",
		}
		return a
	}

	Convey(`From JSON works`, t, func() {
		str := `{
				"all_tests": [
					"FooTest.TestDoBar",
					"FooTest.TestDoBaz"
				],
				"disabled_tests": [
					"FooTest.TestDoBarDisabled"
				],
				"global_tags": ["CPU_64_BITS","MODE_RELEASE","OS_WIN"],
				"per_iteration_data": [{
					"FooTest.TestDoBar": [
						{
							"elapsed_time_ms": 1837,
							"losless_snippet": true,
							"output_snippet": "[ RUN      ] FooTest.TestDoBar",
							"output_snippet_base64": "WyBSVU4gICAgICBdIEZvb1Rlc3QuVGVzdERvQmFy",
							"status": "CRASH"
						},
						{
							"elapsed_time_ms": 1856,
							"losless_snippet": false,
							"output_snippet_base64": "c29tZSBkYXRhIHdpdGggACBhbmQg77u/",
							"status": "FAILURE"
						}
					],
					"FooTest.TestDoBaz": [
						{
							"elapsed_time_ms": 837,
							"losless_snippet": true,
							"output_snippet": "[ RUN      ] FooTest.TestDoBaz",
							"output_snippet_base64": "WyBSVU4gICAgICBdIEZvb1Rlc3QuVGVzdERvQmF6",
							"status": "SUCCESS"
						},
						{
							"elapsed_time_ms": 856,
							"losless_snippet": false,
							"output_snippet_base64": "c29tZSBkYXRhIHdpdGggACBhbmQg77u/",
							"status": "SUCCESS",
							"links": {
								"logcat": "https://luci-logdog.appspot.com/v/?s=logcat"
							}
						}
					]
				}],
				"test_locations": {
					"FooTest.TestDoBar": {
						"file": "../../chrome/browser/foo/test.cc",
						"line": 287
					},
					"FooTest.TestDoBaz": {
						"file": "../../chrome/browser/foo/test.cc",
						"line": 293
					}
				}
			}`

		results := &GTestResults{
			buf: &strings.Builder{},
		}
		err := results.ConvertFromJSON(strings.NewReader(str))
		So(err, ShouldBeNil)
		So(results.AllTests, ShouldResemble, []string{"FooTest.TestDoBar", "FooTest.TestDoBaz"})
		So(results.DisabledTests, ShouldResemble, []string{"FooTest.TestDoBarDisabled"})
		So(results.GlobalTags, ShouldResemble, []string{"CPU_64_BITS", "MODE_RELEASE", "OS_WIN"})
		So(results.PerIterationData, ShouldResemble, []map[string][]*GTestRunResult{
			{
				"FooTest.TestDoBar": {
					{
						Status:              "CRASH",
						ElapsedTimeMs:       1837,
						LosslessSnippet:     true,
						OutputSnippetBase64: "WyBSVU4gICAgICBdIEZvb1Rlc3QuVGVzdERvQmFy",
					},
					{
						Status:              "FAILURE",
						ElapsedTimeMs:       1856,
						OutputSnippetBase64: "c29tZSBkYXRhIHdpdGggACBhbmQg77u/",
					},
				},
				"FooTest.TestDoBaz": {
					{
						Status:              "SUCCESS",
						ElapsedTimeMs:       837,
						LosslessSnippet:     true,
						OutputSnippetBase64: "WyBSVU4gICAgICBdIEZvb1Rlc3QuVGVzdERvQmF6",
					},
					{
						Status:              "SUCCESS",
						ElapsedTimeMs:       856,
						OutputSnippetBase64: "c29tZSBkYXRhIHdpdGggACBhbmQg77u/",
						Links: map[string]json.RawMessage{
							"logcat": json.RawMessage(
								`"https://luci-logdog.appspot.com/v/?s=logcat"`),
						},
					},
				},
			},
		})
		So(results.TestLocations, ShouldResemble, map[string]*Location{
			"FooTest.TestDoBar": {File: "../../chrome/browser/foo/test.cc", Line: 287},
			"FooTest.TestDoBaz": {File: "../../chrome/browser/foo/test.cc", Line: 293},
		})
	})

	Convey(`all_tests can be empty`, t, func() {
		str := `{
				"all_tests": [],
				"disabled_tests": [],
				"global_tags": ["CPU_64_BITS","MODE_RELEASE","OS_WIN"],
				"per_iteration_data": [{}],
				"test_locations": {}
			}`

		results := &GTestResults{
			buf: &strings.Builder{},
		}
		err := results.ConvertFromJSON(strings.NewReader(str))
		So(err, ShouldBeNil)
		So(len(results.AllTests), ShouldEqual, 0)
	})

	Convey("convertTestResult", t, func() {
		convert := func(result *GTestRunResult) *sinkpb.TestResult {
			r := &GTestResults{
				buf: &strings.Builder{},
			}
			tr, err := r.convertTestResult(ctx, "testId", "TestName", result)
			So(err, ShouldBeNil)
			return tr
		}
		Convey("EXCESSIVE_OUTPUT", func() {
			tr := convert(&GTestRunResult{Status: "EXCESSIVE_OUTPUT"})
			So(tr.Status, ShouldEqual, pb.TestStatus_FAIL)
			So(pbutil.StringPairsContain(tr.Tags, pbutil.StringPair("gtest_status", "EXCESSIVE_OUTPUT")), ShouldBeTrue)
		})

		Convey("NOTRUN", func() {
			tr := convert(&GTestRunResult{Status: "NOTRUN"})
			So(tr.Status, ShouldEqual, pb.TestStatus_SKIP)
			So(tr.Expected, ShouldBeFalse)
			So(pbutil.StringPairsContain(tr.Tags, pbutil.StringPair("gtest_status", "NOTRUN")), ShouldBeTrue)
		})

		Convey("SKIPPED", func() {
			tr := convert(&GTestRunResult{Status: "SKIPPED"})
			So(tr.Status, ShouldEqual, pb.TestStatus_SKIP)
			So(tr.Expected, ShouldBeTrue)
			So(pbutil.StringPairsContain(tr.Tags, pbutil.StringPair("gtest_status", "SKIPPED")), ShouldBeTrue)
		})

		Convey("Duration", func() {
			tr := convert(&GTestRunResult{
				Status:        "SUCCESS",
				ElapsedTimeMs: 1e3,
			})
			So(tr.Duration.GetSeconds(), ShouldEqual, 1)
			So(tr.Duration.GetNanos(), ShouldEqual, 0)
		})

		Convey("snippet", func() {
			Convey("valid", func() {
				tr := convert(&GTestRunResult{
					Status:              "SUCCESS",
					LosslessSnippet:     true,
					OutputSnippetBase64: "WyBSVU4gICAgICBdIEZvb1Rlc3QuVGVzdERvQmFyCigxMCBtcyk=",
				})
				So(tr.SummaryHtml, ShouldEqual, snippetInSummaryHtml)
				So(tr.Artifacts["snippet"], ShouldResembleProto, textArt(
					"[ RUN      ] FooTest.TestDoBar\n(10 ms)",
				))
			})

			Convey("invalid does not cause a fatal error", func() {
				tr := convert(&GTestRunResult{
					Status:              "SUCCESS",
					LosslessSnippet:     true,
					OutputSnippetBase64: "invalid base64",
				})
				So(tr.SummaryHtml, ShouldEqual, "")
			})

			Convey("stack trace in artifact", func() {
				tr := convert(&GTestRunResult{
					Status:              "FAILURE",
					LosslessSnippet:     true,
					OutputSnippetBase64: "WyBSVU4gICAgICBdIEZvb1Rlc3QuVGVzdERvQmFyCigxMCBtcyk=",
				})
				So(tr.SummaryHtml, ShouldEqual, snippetInSummaryHtml)
				So(tr.Artifacts, ShouldHaveLength, 1)
				So(tr.Artifacts["snippet"], ShouldResemble, textArt(
					"[ RUN      ] FooTest.TestDoBar\n(10 ms)",
				))
			})
		})

		Convey("testLocations", func() {
			Convey(`Works`, func() {
				results := &GTestResults{
					TestLocations: map[string]*Location{
						"TestName": {
							File: "..\\\\TestFile",
							Line: 54,
						},
					},
					buf: &strings.Builder{},
				}
				tr, err := results.convertTestResult(ctx, "testId", "TestName", &GTestRunResult{Status: "SUCCESS"})
				So(err, ShouldBeNil)
				So(tr.TestMetadata, ShouldResembleProto, &pb.TestMetadata{
					Name: "TestName",
					Location: &pb.TestLocation{
						Repo:     chromiumSrcRepo,
						FileName: "//TestFile",
						Line:     54,
					},
				})
			})
			Convey(`Clean path`, func() {
				results := &GTestResults{
					TestLocations: map[string]*Location{
						"TestName": {
							File: "../../TestFile",
							Line: 54,
						},
					},
					buf: &strings.Builder{},
				}
				tr, err := results.convertTestResult(ctx, "testId", "TestName", &GTestRunResult{Status: "SUCCESS"})
				So(err, ShouldBeNil)
				So(tr.TestMetadata.Location, ShouldResembleProto, &pb.TestLocation{
					Repo:     chromiumSrcRepo,
					FileName: "//TestFile",
					Line:     54,
				})
			})
		})

		Convey("links", func() {
			tr := convert(&GTestRunResult{
				Status:              "SUCCESS",
				LosslessSnippet:     true,
				OutputSnippetBase64: "invalid base64",
				Links: map[string]json.RawMessage{
					"logcat": json.RawMessage(`{"content": "https://luci-logdog.appspot.com/v/?s=logcat"}`),
				},
			})
			So(tr.SummaryHtml, ShouldEqual, `<ul><li><a href="https://luci-logdog.appspot.com/v/?s=logcat">logcat</a></li></ul>`)
		})
	})

	Convey(`extractGTestParameters`, t, func() {
		Convey(`type parametrized`, func() {
			Convey(`with instantiation`, func() {
				baseID, err := extractGTestParameters("MyInstantiation/FooTest/1.DoesBar")
				So(err, ShouldBeNil)
				So(baseID, ShouldEqual, "FooTest.DoesBar/MyInstantiation.1")
			})

			Convey(`without instantiation`, func() {
				baseID, err := extractGTestParameters("FooTest/1.DoesBar")
				So(err, ShouldBeNil)
				So(baseID, ShouldEqual, "FooTest.DoesBar/1")
			})
		})

		Convey(`value parametrized`, func() {
			Convey(`with instantiation`, func() {
				baseID, err := extractGTestParameters("MyInstantiation/FooTest.DoesBar/1")
				So(err, ShouldBeNil)
				So(baseID, ShouldEqual, "FooTest.DoesBar/MyInstantiation.1")
			})

			Convey(`without instantiation`, func() {
				baseID, err := extractGTestParameters("FooTest.DoesBar/1")
				So(err, ShouldBeNil)
				So(baseID, ShouldEqual, "FooTest.DoesBar/1")
			})
		})

		Convey(`not parametrized`, func() {
			baseID, err := extractGTestParameters("FooTest.DoesBar")
			So(err, ShouldBeNil)
			So(baseID, ShouldEqual, "FooTest.DoesBar")
		})

		Convey(`with magic prefixes`, func() {
			baseID, err := extractGTestParameters("FooTest.PRE_PRE_MANUAL_DoesBar")
			So(err, ShouldBeNil)
			So(baseID, ShouldEqual, "FooTest.DoesBar")
		})

		Convey(`with JUnit tests`, func() {
			baseID, err := extractGTestParameters("org.chromium.tests#testFoo_sub__param=val")
			So(err, ShouldBeNil)
			So(baseID, ShouldEqual, "org.chromium.tests#testFoo_sub__param=val")
		})

		Convey(`synthetic parameterized test`, func() {
			_, err := extractGTestParameters("GoogleTestVerification.UninstantiatedParamaterizedTestSuite<Suite>")
			So(err, ShouldErrLike, "not a real test")
			So(syntheticTestTag.In(err), ShouldBeTrue)
		})

		Convey(`synthetic type parameterized test`, func() {
			_, err := extractGTestParameters("GoogleTestVerification.UninstantiatedTypeParamaterizedTestSuite<Suite>")
			So(err, ShouldErrLike, "not a real test")
			So(syntheticTestTag.In(err), ShouldBeTrue)
		})

		Convey(`with unrecognized format`, func() {
			_, err := extractGTestParameters("not_gtest_test")
			So(err, ShouldErrLike, "test id of unknown format")
		})
	})

	Convey(`ToProtos`, t, func() {
		Convey("Works", func() {
			results := &GTestResults{
				DisabledTests: []string{"FooTest.TestDoBarDisabled"},
				GlobalTags: []string{
					"OS_LINUX",
				},
				PerIterationData: []map[string][]*GTestRunResult{
					{
						"BazTest.DoesQux": {
							{
								Status: "SUCCESS",
							},
							{
								Status: "FAILURE",
							},
						},
						"GoogleTestVerification.UninstantiatedTypeParamaterizedTestSuite<Suite>": {
							{
								Status: "SUCCESS",
							},
						},
						"FooTest.DoesBar": {
							{
								Status: "EXCESSIVE_OUTPUT",
							},
							{
								Status: "FAILURE_ON_EXIT",
							},
						},
					},
					{
						"BazTest.DoesQux": {
							{
								Status: "SUCCESS",
							},
							{
								Status: "SUCCESS",
							},
						},
						"FooTest.DoesBar": {
							{
								Status: "FAILURE",
							},
							{
								Status: "FAILURE_ON_EXIT",
							},
						},
					},
				},
				buf: &strings.Builder{},
			}

			testResults, err := results.ToProtos(ctx)
			So(err, ShouldBeNil)

			expected := []*sinkpb.TestResult{
				// Disabled tests.
				{
					TestId:   "FooTest.TestDoBarDisabled",
					Expected: true,
					Status:   pb.TestStatus_SKIP,
					Tags: pbutil.StringPairs(
						"test_name", "FooTest.TestDoBarDisabled",
						"disabled_test", "true",
						"gtest_global_tag", "OS_LINUX",
						"orig_format", "chromium_gtest",
					),
					TestMetadata: &pb.TestMetadata{
						Name: "FooTest.TestDoBarDisabled",
					},
				},
				// Iteration 1.
				{
					TestId:   "BazTest.DoesQux",
					Expected: true,
					Status:   pb.TestStatus_PASS,
					Tags: pbutil.StringPairs(
						"test_name", "BazTest.DoesQux",
						"gtest_status", "SUCCESS",
						"lossless_snippet", "false",
						"gtest_global_tag", "OS_LINUX",
						"orig_format", "chromium_gtest",
					),
					TestMetadata: &pb.TestMetadata{
						Name: "BazTest.DoesQux",
					},
				},
				{
					TestId: "BazTest.DoesQux",
					Status: pb.TestStatus_FAIL,
					Tags: pbutil.StringPairs(
						"test_name", "BazTest.DoesQux",
						"gtest_status", "FAILURE",
						"lossless_snippet", "false",
						"gtest_global_tag", "OS_LINUX",
						"orig_format", "chromium_gtest",
					),
					TestMetadata: &pb.TestMetadata{
						Name: "BazTest.DoesQux",
					},
				},
				{
					TestId: "FooTest.DoesBar",
					Status: pb.TestStatus_FAIL,
					Tags: pbutil.StringPairs(
						"test_name", "FooTest.DoesBar",
						"gtest_status", "EXCESSIVE_OUTPUT",
						"lossless_snippet", "false",
						"gtest_global_tag", "OS_LINUX",
						"orig_format", "chromium_gtest",
					),
					TestMetadata: &pb.TestMetadata{
						Name: "FooTest.DoesBar",
					},
				},
				{
					TestId: "FooTest.DoesBar",
					Status: pb.TestStatus_FAIL,
					Tags: pbutil.StringPairs(
						"test_name", "FooTest.DoesBar",
						"gtest_status", "FAILURE_ON_EXIT",
						"lossless_snippet", "false",
						"gtest_global_tag", "OS_LINUX",
						"orig_format", "chromium_gtest",
					),
					TestMetadata: &pb.TestMetadata{
						Name: "FooTest.DoesBar",
					},
				},

				// Iteration 2.
				{
					TestId:   "BazTest.DoesQux",
					Expected: true,
					Status:   pb.TestStatus_PASS,
					Tags: pbutil.StringPairs(
						"test_name", "BazTest.DoesQux",
						"gtest_status", "SUCCESS",
						"lossless_snippet", "false",
						"gtest_global_tag", "OS_LINUX",
						"orig_format", "chromium_gtest",
					),
					TestMetadata: &pb.TestMetadata{
						Name: "BazTest.DoesQux",
					},
				},
				{
					TestId:   "BazTest.DoesQux",
					Expected: true,
					Status:   pb.TestStatus_PASS,
					Tags: pbutil.StringPairs(
						"test_name", "BazTest.DoesQux",
						"gtest_status", "SUCCESS",
						"lossless_snippet", "false",
						"gtest_global_tag", "OS_LINUX",
						"orig_format", "chromium_gtest",
					),
					TestMetadata: &pb.TestMetadata{
						Name: "BazTest.DoesQux",
					},
				},
				{
					TestId: "FooTest.DoesBar",
					Status: pb.TestStatus_FAIL,
					Tags: pbutil.StringPairs(
						"test_name", "FooTest.DoesBar",
						"gtest_status", "FAILURE",
						"lossless_snippet", "false",
						"gtest_global_tag", "OS_LINUX",
						"orig_format", "chromium_gtest",
					),
					TestMetadata: &pb.TestMetadata{
						Name: "FooTest.DoesBar",
					},
				},
				{
					TestId: "FooTest.DoesBar",
					Status: pb.TestStatus_FAIL,
					Tags: pbutil.StringPairs(
						"test_name", "FooTest.DoesBar",
						"gtest_status", "FAILURE_ON_EXIT",
						"lossless_snippet", "false",
						"gtest_global_tag", "OS_LINUX",
						"orig_format", "chromium_gtest",
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
