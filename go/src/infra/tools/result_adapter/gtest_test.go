// Copyright 2020 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
	"go.chromium.org/luci/resultdb/pbutil"
	pb "go.chromium.org/luci/resultdb/proto/v1"
	sinkpb "go.chromium.org/luci/resultdb/sink/proto/v1"
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
							"status": "FAILURE",
							"result_parts":[{
								"summary_base64":"VGhpcyBpcyBhIGZhaWx1cmUgbWVzc2FnZS4=",
								"type":"failure"
							}, {
								"summary_base64":"VGhpcyBpcyBhIGZhdGFsIGZhaWx1cmUgbWVzc2FnZS4=",
								"type":"fatal_failure"
							}]
						}
					],
					"FooTest.TestDoBaz": [
						{
							"elapsed_time_ms": 837,
							"losless_snippet": true,
							"output_snippet": "[ RUN      ] FooTest.TestDoBaz",
							"output_snippet_base64": "WyBSVU4gICAgICBdIEZvb1Rlc3QuVGVzdERvQmF6",
							"status": "SUCCESS",
							"result_parts":[{
								"type":"success"
							}]
						},
						{
							"elapsed_time_ms": 856,
							"losless_snippet": false,
							"output_snippet_base64": "c29tZSBkYXRhIHdpdGggACBhbmQg77u/",
							"status": "SUCCESS",
							"links": {
								"logcat": "https://luci-logdog.appspot.com/v/?s=logcat"
							},
							"result_parts":[{}]
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

		results := &GTestResults{}
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
						ResultParts: []*GTestRunResultPart{
							{
								SummaryBase64: "VGhpcyBpcyBhIGZhaWx1cmUgbWVzc2FnZS4=",
								Type:          "failure",
							},
							{
								SummaryBase64: "VGhpcyBpcyBhIGZhdGFsIGZhaWx1cmUgbWVzc2FnZS4=",
								Type:          "fatal_failure",
							},
						},
					},
				},
				"FooTest.TestDoBaz": {
					{
						Status:              "SUCCESS",
						ElapsedTimeMs:       837,
						LosslessSnippet:     true,
						OutputSnippetBase64: "WyBSVU4gICAgICBdIEZvb1Rlc3QuVGVzdERvQmF6",
						ResultParts: []*GTestRunResultPart{
							{
								Type: "success",
							},
						},
					},
					{
						Status:              "SUCCESS",
						ElapsedTimeMs:       856,
						OutputSnippetBase64: "c29tZSBkYXRhIHdpdGggACBhbmQg77u/",
						Links: map[string]json.RawMessage{
							"logcat": json.RawMessage(
								`"https://luci-logdog.appspot.com/v/?s=logcat"`),
						},
						ResultParts: []*GTestRunResultPart{{}},
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

		results := &GTestResults{}
		err := results.ConvertFromJSON(strings.NewReader(str))
		So(err, ShouldBeNil)
		So(len(results.AllTests), ShouldEqual, 0)
	})

	Convey("convertTestResult", t, func() {
		var buf bytes.Buffer
		convert := func(result *GTestRunResult) *sinkpb.TestResult {
			r := &GTestResults{}
			buf := &bytes.Buffer{}
			tr, err := r.convertTestResult(ctx, buf, "testId", "TestName", result)
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
				}
				tr, err := results.convertTestResult(ctx, &buf, "testId", "TestName", &GTestRunResult{Status: "SUCCESS"})
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
				}
				tr, err := results.convertTestResult(ctx, &buf, "testId", "TestName", &GTestRunResult{Status: "SUCCESS"})
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

		Convey("failure reason", func() {
			Convey("first failure takes precedence", func() {
				tr := convert(&GTestRunResult{
					Status: "FAILURE",
					ResultParts: []*GTestRunResultPart{
						{
							// "This is a failure message."
							SummaryBase64: "VGhpcyBpcyBhIGZhaWx1cmUgbWVzc2FnZS4=",
							Type:          "failure",
						},
						{
							// "This is a second failure message."
							SummaryBase64: "VGhpcyBpcyBhIHNlY29uZCBmYWlsdXJlIG1lc3NhZ2Uu",
							Type:          "failure",
						},
					},
				})
				So(tr.FailureReason.PrimaryErrorMessage, ShouldEqual, `This is a failure message.`)
			})
			Convey("first fatal failure takes precedence", func() {
				tr := convert(&GTestRunResult{
					Status: "FAILURE",
					ResultParts: []*GTestRunResultPart{
						{
							// "This is a failure message."
							SummaryBase64: "VGhpcyBpcyBhIGZhaWx1cmUgbWVzc2FnZS4=",
							Type:          "failure",
						},
						{
							// "This is a fatal failure message."
							SummaryBase64: "VGhpcyBpcyBhIGZhdGFsIGZhaWx1cmUgbWVzc2FnZS4=",
							Type:          "fatal_failure",
						},
						{
							// "This is a second fatal failure message."
							SummaryBase64: "VGhpcyBpcyBhIHNlY29uZCBmYXRhbCBmYWlsdXJlIG1lc3NhZ2Uu",
							Type:          "fatal_failure",
						},
					},
				})
				So(tr.FailureReason.PrimaryErrorMessage, ShouldEqual, `This is a fatal failure message.`)
			})
			Convey("failure result parts take precedence over snippet", func() {
				tr := convert(&GTestRunResult{
					Status: "FAILURE",
					ResultParts: []*GTestRunResultPart{
						{
							// "This is a failure message."
							SummaryBase64: "VGhpcyBpcyBhIGZhaWx1cmUgbWVzc2FnZS4=",
							Type:          "failure",
						},
					},
					// [FATAL:file_name.cc(123)] Error message.
					OutputSnippetBase64: "W0ZBVEFMOmZpbGVfbmFtZS5jYygxMjMpXSBFcnJvciBtZXNzYWdlLg==",
				})
				So(tr.FailureReason.PrimaryErrorMessage, ShouldEqual, `This is a failure message.`)
			})
			Convey("Google Test trace is removed from failure reason", func() {
				input := "error message\nGoogle Test trace:\nRandom tracing output\n"
				tr := convert(&GTestRunResult{
					Status: "FAILURE",
					ResultParts: []*GTestRunResultPart{
						{
							SummaryBase64: base64.StdEncoding.EncodeToString([]byte(input)),
							Type:          "failure",
						},
					},
				})
				So(tr.FailureReason.PrimaryErrorMessage, ShouldEqual, "error message\n")
			})
			Convey("Leading and trailing spaces are removed from failure reason", func() {
				input := "  error\n message\n  "
				tr := convert(&GTestRunResult{
					Status: "FAILURE",
					ResultParts: []*GTestRunResultPart{
						{
							SummaryBase64: base64.StdEncoding.EncodeToString([]byte(input)),
							Type:          "failure",
						},
					},
				})
				So(tr.FailureReason.PrimaryErrorMessage, ShouldEqual, "error\n message")
			})
			Convey("empty", func() {
				tr := convert(&GTestRunResult{
					Status:      "FAILURE",
					ResultParts: []*GTestRunResultPart{},
				})
				So(tr.FailureReason, ShouldEqual, nil)
			})
			Convey("primary error message truncated at 1024 bytes", func() {
				var input bytes.Buffer
				var expected bytes.Buffer
				// Print 1020 bytes as 340 3-byte runes.
				for i := 0; i < 340; i++ {
					// Use swedish "Place of interest symbol", which encodes as three-bytes, e2 8c 98.
					// See https://blog.golang.org/strings.
					input.WriteRune('\u2318')
					expected.WriteRune('\u2318')
				}
				// Numbers and dots are one byte in UTF-8. Construct input to be 1025 bytes,
				// expected to be the 1024-byte truncation.
				input.WriteString("12345")
				expected.WriteString("1...")

				tr := convert(&GTestRunResult{
					Status: "FAILURE",
					ResultParts: []*GTestRunResultPart{
						{
							SummaryBase64: base64.StdEncoding.EncodeToString(input.Bytes()),
							Type:          "failure",
						},
					},
				})
				So(tr.FailureReason.PrimaryErrorMessage, ShouldEqual, expected.String())
			})
			Convey("invalid type does not cause a fatal error", func() {
				tr := convert(&GTestRunResult{
					Status: "FAILURE",
					ResultParts: []*GTestRunResultPart{
						{
							// "This is a failure message."
							SummaryBase64: "VGhpcyBpcyBhIGZhaWx1cmUgbWVzc2FnZS4=",
							Type:          "undefined",
						},
					},
				})
				So(tr.FailureReason, ShouldEqual, nil)
			})
			Convey("invalid UTF-8 does not cause a fatal error", func() {
				tr := convert(&GTestRunResult{
					Status: "FAILURE",
					ResultParts: []*GTestRunResultPart{
						{
							// Encodes (hexadecimal byte) ff, which is invalid UTF-8.
							// See https://www.cl.cam.ac.uk/~mgk25/ucs/examples/UTF-8-test.txt.
							SummaryBase64: "/w8=",
							Type:          "fatal_failure",
						},
					},
				})
				So(tr.FailureReason, ShouldEqual, nil)
			})
			Convey("invalid base64 does not cause a fatal error", func() {
				tr := convert(&GTestRunResult{
					Status: "FAILURE",
					ResultParts: []*GTestRunResultPart{
						{
							SummaryBase64: "Invalid base 64.",
							Type:          "fatal_failure",
						},
					},
				})
				So(tr.FailureReason, ShouldEqual, nil)
			})
			Convey("extracted from snippet", func() {
				tr := convert(&GTestRunResult{
					Status: "FAILURE",
					// [FATAL:file_name.cc(123)] Error message.
					OutputSnippetBase64: "W0ZBVEFMOmZpbGVfbmFtZS5jYygxMjMpXSBFcnJvciBtZXNzYWdlLg==",
				})
				So(tr.FailureReason, ShouldNotEqual, nil)
				So(tr.FailureReason.PrimaryErrorMessage, ShouldEqual, `file_name.cc(123): Error message.`)
			})
		})
	})

	Convey("extractFailureReasonFromSnippet", t, func() {
		test := func(input string, expected string) {
			result := extractFailureReasonFromSnippet(ctx, input)
			if expected != "" {
				So(result, ShouldNotEqual, nil)
				So(result.PrimaryErrorMessage, ShouldEqual, expected)
			} else {
				So(result, ShouldEqual, nil)
			}
		}
		Convey("fatal example without DCheck", func() {
			// This example does has in the message and and does not have "Check failed:".
			example := "[70297:775:0716/090328.691561:FATAL:sync_test.cc(928)] AwaitQuiescence() failed."
			test(example, "sync_test.cc(928): AwaitQuiescence() failed.")
		})
		Convey("fatal DCheck example #1", func() {
			example := "[722:259:FATAL:multiplex_router.cc(181)] Check failed: !client_. "
			test(example, "multiplex_router.cc(181): Check failed: !client_.")
		})
		Convey("fatal DCheck example #2", func() {
			example := "[27483:27483:0807/204616.124527:FATAL:video_source.mojom.cc(555)] Check failed: !connected. PushVideoStreamSubscription::GetPhotoStateCallback was destroyed"
			test(example, "video_source.mojom.cc(555): Check failed: !connected. PushVideoStreamSubscription::GetPhotoStateCallback was destroyed")
		})
		Convey("fatal DCheck example #3", func() {
			example := "FATAL ash_unittests[6813:6813]: [display_manager_test_api.cc(160)] Check failed: display_manager_->GetNumDisplays() >= 2U (1 vs. 2)"
			test(example, "display_manager_test_api.cc(160): Check failed: display_manager_->GetNumDisplays() >= 2U (1 vs. 2)")
		})
		Convey("fatal DCheck example #4", func() {
			example := "[FATAL:gl_context.cc(203)] Check failed: false. "
			test(example, "gl_context.cc(203): Check failed: false.")
		})
		Convey("non-fatal DCheck example", func() {
			// This example does not have FATAL in the message and relies upon matching "Check failed:".
			example := "../../base/allocator/partition_allocator/partition_root.h(998) Check failed: !slot_span->bucket->is_direct_mapped()"
			test(example, "partition_root.h(998): Check failed: !slot_span->bucket->is_direct_mapped()")
		})
		Convey("unix line endings", func() {
			example := "blah\n../../base/allocator/partition_allocator/partition_root.h(998) Check failed: !slot_span->bucket->is_direct_mapped()\nblah"
			test(example, "partition_root.h(998): Check failed: !slot_span->bucket->is_direct_mapped()")
		})
		Convey("windows line endings", func() {
			example := "blah\r\n../../base/allocator/partition_allocator/partition_root.h(998) Check failed: !slot_span->bucket->is_direct_mapped()\r\nblah"
			test(example, "partition_root.h(998): Check failed: !slot_span->bucket->is_direct_mapped()")
		})
		Convey("GTest Expectation", func() {
			// Test a log with multiple expectation failures, to make
			// sure only one gets picekd up.
			example := `Unrelated log line
../../content/public/test/browser_test_base.cc:718: Failure
Expected equality of these values:
  expected_exit_code_
    Which is: 0
  ContentMain(std::move(params))
    Which is: 1
Stack trace:
#0 0x5640a41b448b content::BrowserTestBase::SetUp()
../../content/public/test/browser_test_base.cc:719: Failure
Expected something else
Stack trace:
#0 0x5640a41b448b content::BrowserTestBase::SetUp()`
			expected := `Expected equality of these values:
  expected_exit_code_
    Which is: 0
  ContentMain(std::move(params))
    Which is: 1`
			test(example, expected)
		})
		Convey("GTest Expectation (Windows)", func() {
			example := `Unrelated log line
../../chrome/browser/net/network_context_configuration_browsertest.cc(984): error: Expected equality of these values:
  net::ERR_CONNECTION_REFUSED
    Which is: -102
  simple_loader2->NetError()
    Which is: -21
Stack trace:
Backtrace:
	std::__1::unique_ptr<network::ResourceRequest,std::__1::default_delete<network::ResourceRequest> >::reset [0x007A3C5B+7709]`
			expected := `Expected equality of these values:
  net::ERR_CONNECTION_REFUSED
    Which is: -102
  simple_loader2->NetError()
    Which is: -21`
			test(example, expected)
		})
		Convey("empty snippet", func() {
			example := ""
			test(example, "")
		})
		Convey("non-matching snippet", func() {
			example := "blah\nblah\n"
			test(example, "")
		})
		Convey("first fatal error extracted", func() {
			example := "blah\npath/to/file.cc(123) Check failed: bool_expression\n[FATAL:file2.cc(456)] Check failed: second_bool_expression\nblah"
			test(example, "file.cc(123): Check failed: bool_expression")
		})
	})

	Convey("truncateString", t, func() {
		Convey("one-byte runes", func() {
			Convey("longer than desired length", func() {
				result := truncateString("12345678", 5)
				So(result, ShouldEqual, "12...")
			})
			Convey("exactly desired length", func() {
				result := truncateString("12345", 5)
				So(result, ShouldEqual, "12345")
			})
			Convey("shorter than desired length", func() {
				result := truncateString("1234", 5)
				So(result, ShouldEqual, "1234")
			})
		})
		Convey("three-byte runes", func() {
			Convey("longer than desired length", func() {
				result := truncateString("\u2318\u2318\u2318", 7)
				So(result, ShouldEqual, "\u2318...")
			})
			Convey("exactly desired length", func() {
				result := truncateString("\u2318\u2318", 6)
				So(result, ShouldEqual, "\u2318\u2318")
			})
			Convey("shorter than desired length", func() {
				result := truncateString("\u2318\u2318", 7)
				So(result, ShouldEqual, "\u2318\u2318")
			})
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
