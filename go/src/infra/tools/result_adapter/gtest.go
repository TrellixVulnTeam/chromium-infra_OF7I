// Copyright 2020 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/resultdb/pbutil"
	pb "go.chromium.org/luci/resultdb/proto/v1"
	sinkpb "go.chromium.org/luci/resultdb/sink/proto/v1"
)

var (
	// Prefixes that may be present in the test name and must be stripped before forming the base id.
	prefixes = []string{"MANUAL_", "PRE_"}

	// Java base ids aren't actually GTest but use the same launcher output format.
	// TODO(chanli@): remove java test related logic when crbug.com/1104245 and crbug.com/1104238 are done.
	javaIDRe = regexp.MustCompile(`^[\w.]+#`)

	// Test base ids look like FooTest.DoesBar: "FooTest" is the suite and "DoesBar" the test name.
	baseIDRE = regexp.MustCompile(`^(\w+)\.(\w+)$`)

	// Type parametrized test examples:
	// - MyInstantiation/FooTest/1.DoesBar
	// - FooTest/1.DoesBar
	// - FooType/MyType.DoesBar
	//
	// In the above examples, "FooTest" is the suite, "DoesBar" the test name, "MyInstantiation" the
	// optional instantiation, "1" the index of the type on which the test has been instantiated, if
	// no string representation for the type has been provided, and "MyType" is the user-provided
	// string representation of the type on which the test has been instantiated.
	typeParamRE = regexp.MustCompile(`^((\w+)/)?(\w+)/(\w+)\.(\w+)$`)

	// Value parametrized tests examples:
	// - MyInstantiation/FooTest.DoesBar/1
	// - FooTest.DoesBar/1
	// - FooTest.DoesBar/TestValue
	//
	// In the above examples, "FooTest" is the suite, "DoesBar" the test name, "MyInstantiation" the
	// optional instantiation, "1" the index of the value on which the test has been instantiated, if
	// no string representation for the value has been provided, and "TestValue" is the user-provided
	// string representation of the value on which the test has been instantiated.
	valueParamRE = regexp.MustCompile(`^((\w+)/)?(\w+)\.(\w+)/(\w+)$`)

	// TODO(chanli@): Remove this after crbug.com/1045846 is fixed.
	// This is a synthetic test created by test launcher, not a real test.
	syntheticTestRE = regexp.MustCompile(`^GoogleTestVerification.Uninstantiated(?:Type)?ParamaterizedTestSuite<\w+>$`)

	syntheticTestTag = errors.BoolTag{
		Key: errors.NewTagKey("synthetic test"),
	}

	// fatalMessageRE extracts fatal log lines, and is used as a fallback to
	// extract failure reasons where result parts are not available (such as
	// when the test crashes.)
	// The first capture group captures the name of the file which generated
	// the fatal message, the second captures the message itself. Derived with
	// the help of experiments to properly extract the message and file from
	// log lines which contained the word "FATAL" or "Check failed:" in ~99%
	// of cases.
	// The main known category of crash this regexp does not cater for is when
	// the Chrome crashes due to signals received by the OS (e.g. SEGV_MAPERR,
	// EXCEPTION_ACCESS_VIOLATION). No attempt is made to extract these are
	// the messages do not appear to be specific enough to be useful for
	// clustering. Further work may improve failure reason extraction for
	// these types of errors.
	fatalMessageRE = regexp.MustCompile(`.*FATAL.*?([a-zA-Z0-9_.]+\.[a-zA-Z0-9_]+\([0-9]+\))]? (.*)`)

	// checkFailedRE extracts fatal log lines and is used in addition to
	// fatalMessageRE. This results in an additional ~1% of fatal failure
	// messages being extracted. Both regular expressions overlap
	// significantly.
	checkFailedRE = regexp.MustCompile(`.*?([a-zA-Z0-9_.]+\.[a-zA-Z0-9_]+\([0-9]+\))]? (Check failed:.*)`)

	// Extracts failed Google Test expectations from test snippets.
	// Used as a fallback where result parts are not available
	// (such as when the test crashes.)
	//
	// Example (from a Ubuntu machine):
	// ../../content/public/test/browser_test_base.cc:718: Failure
	// Expected equality of these values:
	//   expected_exit_code_
	// 	Which is: 0
	//   ContentMain(std::move(params))
	// 	Which is: 1
	// Stack trace:
	// #0 0x5640a41b448b content::BrowserTestBase::SetUp()
	//
	// Note that gtest can produce different kinds of expectation
	// failures that do not start with
	// "Expected equality of these values:", so no attempt was made
	// to match on this part of the text.
	gtestExpectationRE = regexp.MustCompile(`(?s)[a-zA-Z0-9_.]+\.[a-zA-Z0-9_]+:[0-9]+: Failure\n(.*?)Stack trace:`)

	// As above, but for output generated on Windows machines.
	// Example:
	// ../../chrome/browser/net/network_context_configuration_browsertest.cc(984): error: Expected equality of these values:
	//   net::ERR_CONNECTION_REFUSED
	//     Which is: -102
	//   simple_loader2->NetError()
	//     Which is: -21
	// Stack trace:
	// Backtrace:
	//         std::__1::unique_ptr<network::ResourceRequest,std::__1::default_delete<network::ResourceRequest> >::reset [0x007A3C5B+7709]
	gtestExpectationWindowsRE = regexp.MustCompile(`(?s)[a-zA-Z0-9_.]+\.[a-zA-Z0-9_]+\([0-9]+\): error: (.*?)Stack trace:`)

	// googleTestTraceRE identifies output from SCOPED_TRACE calls in GTest
	// so that they can be removed from the primary error message.
	googleTestTraceRE = regexp.MustCompile(`(?s)Google Test trace:.*$`)

	// ResultSink limits the failure reason primary error message to 1024 bytes in UTF-8.
	maxPrimaryErrorBytes = 1024
)

// GTestResults represents the structure as described to be generated in
// https://cs.chromium.org/chromium/src/base/test/launcher/test_results_tracker.h?l=83&rcl=96020cfd447cb285acfa1a96c37a67ed22fa2499
// (base::TestResultsTracker::SaveSummaryAsJSON)
//
// Fields not used by Test Results are omitted.
type GTestResults struct {
	AllTests      []string `json:"all_tests"`
	DisabledTests []string `json:"disabled_tests"`
	GlobalTags    []string `json:"global_tags"`

	// PerIterationData is a vector of run iterations, each mapping test names to a list of test data.
	PerIterationData []map[string][]*GTestRunResult `json:"per_iteration_data"`

	// TestLocations maps test names to their location in code.
	TestLocations map[string]*Location `json:"test_locations"`
}

// GTestRunResult represents the per_iteration_data as described in
// https://cs.chromium.org/chromium/src/base/test/launcher/test_results_tracker.h?l=83&rcl=96020cfd447cb285acfa1a96c37a67ed22fa2499
// (base::TestResultsTracker::SaveSummaryAsJSON)
//
// Fields not used by Test Results are omitted.
type GTestRunResult struct {
	Status        string  `json:"status"`
	ElapsedTimeMs float64 `json:"elapsed_time_ms"`

	LosslessSnippet     bool   `json:"losless_snippet"`
	OutputSnippetBase64 string `json:"output_snippet_base64"`

	// Links are not generated by test_launcher, but harnesses built on top may add them to the json.
	Links map[string]json.RawMessage `json:"links"`

	ResultParts []*GTestRunResultPart `json:"result_parts"`
}

// GTestRunResultPart represents the result_parts as described in
// https://cs.chromium.org/chromium/src/base/test/launcher/test_results_tracker.h?l=83&rcl=96020cfd447cb285acfa1a96c37a67ed22fa2499
// (base::TestResultsTracker::SaveSummaryAsJSON)
//
// This is the result of a single gtest SUCCEED, SKIP, or failed EXPECT or ASSERT.
//
// Fields not used by Test Results are omitted.
type GTestRunResultPart struct {
	// The type of result part. Can be one of:
	// - "success" (if a test used SUCCEED()),
	// - "failure" (for non-fatal failure, i.e. from EXPECT()),
	// - "fatal_failure" (for fatal failure, i.e. from ASSERT()) or
	// - "skip" (from GTEST_SKIP()).
	Type          string `json:"type"`
	SummaryBase64 string `json:"summary_base64"`
}

// Location describes a code location.
type Location struct {
	File string `json:"file"`
	Line int    `json:"line"`
}

// Link represents a link content.
type Link struct {
	Content string `json:"content"`
}

// ConvertFromJSON reads the provided reader into the receiver.
//
// The receiver is cleared and its fields overwritten.
func (r *GTestResults) ConvertFromJSON(reader io.Reader) error {
	*r = GTestResults{}
	if err := json.NewDecoder(reader).Decode(r); err != nil {
		return err
	}

	return nil
}

// ToProtos converts test results in r to []*sinkpb.TestResult.
func (r *GTestResults) ToProtos(ctx context.Context) ([]*sinkpb.TestResult, error) {
	var ret []*sinkpb.TestResult
	var testNames []string

	globalTags := make([]*pb.StringPair, len(r.GlobalTags)+1)
	for i, tag := range r.GlobalTags {
		globalTags[i] = pbutil.StringPair("gtest_global_tag", tag)
	}
	globalTags[len(r.GlobalTags)] = pbutil.StringPair(originalFormatTagKey, formatGTest)

	for _, name := range r.DisabledTests {
		testID, err := extractGTestParameters(name)
		switch {
		case syntheticTestTag.In(err):
			continue
		case err != nil:
			return nil, errors.Annotate(err,
				"failed to extract test id and parameters from %q", name).Err()
		}
		tr := &sinkpb.TestResult{
			TestId:   testID,
			Expected: true,
			Status:   pb.TestStatus_SKIP,
			Tags: pbutil.StringPairs(
				// Store the original Gtest test name.
				"test_name", name,
				"disabled_test", "true",
			),
			TestMetadata: &pb.TestMetadata{Name: name},
		}
		tr.Tags = append(tr.Tags, globalTags...)

		ret = append(ret, tr)
	}

	var buf bytes.Buffer
	for _, data := range r.PerIterationData {
		// Sort the test name to make the output deterministic.
		testNames = testNames[:0]
		for name := range data {
			testNames = append(testNames, name)
		}
		sort.Strings(testNames)

		for _, name := range testNames {
			testID, err := extractGTestParameters(name)
			switch {
			case syntheticTestTag.In(err):
				continue
			case err != nil:
				return nil, errors.Annotate(err,
					"failed to extract test id and parameters from %q", name).Err()
			}

			for i, result := range data[name] {
				// Store the processed test result into the correct part of the overall map.
				rpb, err := r.convertTestResult(ctx, &buf, testID, name, result)
				if err != nil {
					return nil, errors.Annotate(err,
						"iteration %d of test %s failed to convert run result", i, name).Err()
				}
				rpb.Tags = append(rpb.Tags, globalTags...)

				ret = append(ret, rpb)
			}
		}
	}

	return ret, nil
}

func fromGTestStatus(s string) (status pb.TestStatus, expected bool, err error) {
	switch s {
	case "SUCCESS":
		return pb.TestStatus_PASS, true, nil
	case "FAILURE":
		return pb.TestStatus_FAIL, false, nil
	case "FAILURE_ON_EXIT":
		return pb.TestStatus_FAIL, false, nil
	case "TIMEOUT":
		return pb.TestStatus_ABORT, false, nil
	case "CRASH":
		return pb.TestStatus_CRASH, false, nil
	case "SKIPPED":
		return pb.TestStatus_SKIP, true, nil
	case "EXCESSIVE_OUTPUT":
		return pb.TestStatus_FAIL, false, nil
	case "NOTRUN":
		return pb.TestStatus_SKIP, false, nil
	case "UNKNOWN":
		return pb.TestStatus_ABORT, false, nil
	default:
		// This would only happen if the set of possible GTest result statuses change and resultsdb has
		// not been updated to match.
		return pb.TestStatus_STATUS_UNSPECIFIED, false, errors.Reason("unknown GTest status %q", s).Err()
	}
}

// extractGTestParameters extracts parameters from a test id as a mapping with "param/" keys.
func extractGTestParameters(testID string) (baseID string, err error) {
	var suite, name, instantiation, id string

	// If this is a JUnit tests, don't try to extract parameters.
	if match := javaIDRe.FindStringSubmatch(testID); match != nil {
		baseID = testID
		return
	}

	// Tests can be only one of type- or value-parametrized, if parametrized at all.
	if match := typeParamRE.FindStringSubmatch(testID); match != nil {
		// Extract type parameter.
		suite = match[3]
		name = match[5]
		instantiation = match[2]
		id = match[4]
	} else if match := valueParamRE.FindStringSubmatch(testID); match != nil {
		// Extract value parameter.
		suite = match[3]
		name = match[4]
		instantiation = match[2]
		id = match[5]
	} else if match := baseIDRE.FindStringSubmatch(testID); match != nil {
		// Otherwise our test id should not be parametrized, so extract the suite
		// and name.
		suite = match[1]
		name = match[2]
	} else if syntheticTestRE.MatchString(testID) {
		// A synthetic test, skip.
		err = errors.Reason("not a real test").Tag(syntheticTestTag).Err()
	} else {
		// Otherwise test id format is unrecognized.
		err = errors.Reason("test id of unknown format").Err()
		return
	}

	// Strip prefixes from test name if necessary.
	name = stripRepeatedPrefixes(name, prefixes...)

	switch {
	case id == "":
		baseID = fmt.Sprintf("%s.%s", suite, name)
	case instantiation == "":
		baseID = fmt.Sprintf("%s.%s/%s", suite, name, id)
	default:
		baseID = fmt.Sprintf("%s.%s/%s.%s", suite, name, instantiation, id)
	}

	return
}

// truncateString truncates a UTF-8 string to the given number of bytes.
// If the string is truncated, ellipsis ("...") are added.
// Truncation is aware of UTF-8 runes and will only truncate whole runes.
// length must be at least 3 (to leave space for ellipsis, if needed).
func truncateString(s string, length int) string {
	if len(s) <= length {
		return s
	}
	// The index (in bytes) at which to begin truncating the string.
	lastIndex := 0
	// Find the point where we must truncate from. We only want to
	// start truncation at the start/end of a rune, not in the middle.
	// See https://blog.golang.org/strings.
	for i := range s {
		if i <= (length - 3) {
			lastIndex = i
		}
	}
	return s[:lastIndex] + "..."
}

// extractPrimaryFailure returns the first fatal result part, or if it does not exist,
// the first non-fatal failure. If there is no failure, it returns nil.
func extractPrimaryFailure(ctx context.Context, parts []*GTestRunResultPart) *GTestRunResultPart {
	var primaryFailure *GTestRunResultPart
	for _, part := range parts {
		switch part.Type {
		case "success":
		case "failure":
			// Select the first non-fatal failure.
			if primaryFailure == nil {
				primaryFailure = part
			}
		case "fatal_failure":
			// Or the first fatal failure, if it exists.
			return part
		case "skip":
		default:
			logging.Warningf(ctx, "Unknown GTest result part status %q", part.Type)
		}
	}
	return primaryFailure
}

// extractFailureReasonFromResultParts identifies a test failure reason from
// structured test result output provided by Google Test. This is the
// preferred way of identifying a test's failure reason, but this way may not
// always be possible (e.g. if the test crashed).
func extractFailureReasonFromResultParts(ctx context.Context, parts []*GTestRunResultPart) *pb.FailureReason {
	f := extractPrimaryFailure(ctx, parts)
	if f == nil {
		// No failure part.
		return nil
	}
	summaryBytes, err := base64.StdEncoding.DecodeString(f.SummaryBase64)
	if err != nil {
		// Log the error, but we shouldn't fail to convert a file just because we can't
		// convert a summary.
		logging.Warningf(ctx, "Failed to convert SummaryBase64 %q", f.SummaryBase64)
		return nil
	}
	if !utf8.Valid(summaryBytes) {
		// summaryBytes may not be valid UTF-8 (this is permitted on the Chrome side).
		// If so, drop it, but log a message. In future, we could consider escaping
		// characters.
		logging.Warningf(ctx, "SummaryBase64 is not valid UTF-8 %q", f.SummaryBase64)
		return nil
	}
	summary := strings.TrimSpace(string(summaryBytes))
	return &pb.FailureReason{
		PrimaryErrorMessage: truncateString(trimGoogleTestTrace(summary), maxPrimaryErrorBytes),
	}
}

func trimGoogleTestTrace(message string) string {
	return googleTestTraceRE.ReplaceAllString(message, "")
}

// extractFailureReasonFromSnippet implements a fallback approach to
// extracting test failure reasons, based on analysing the test snippet.
// It tries to identify fatal log messages (including DCheck failures)
// and failed GTest expectations. This fallback is usually used if
// the test crashed and GTest does not report structured failure data.
func extractFailureReasonFromSnippet(ctx context.Context, snippet string) *pb.FailureReason {
	// Try to find fatal log messages.
	match := fatalMessageRE.FindStringSubmatchIndex(snippet)
	checkFailedMatch := checkFailedRE.FindStringSubmatchIndex(snippet)
	// Pick whichever match exists, or the first (if both matched).
	if match == nil || (checkFailedMatch != nil && checkFailedMatch[0] < match[0]) {
		match = checkFailedMatch
	}
	// A fatal error was found. Return it.
	if match != nil {
		// File name and line, e.g. "tls_handshaker.cc(123)".
		fileName := snippet[match[2]:match[3]]
		// The failure message, e.g. "Check failed: !condition. "
		message := strings.TrimSpace(snippet[match[4]:match[5]])

		// Include the location of the fatal error as sometimes "Check failed: "
		// errors are non specific. E.g. "Check failed: false".
		primaryError := fmt.Sprintf("%v: %v", fileName, message)
		return &pb.FailureReason{
			PrimaryErrorMessage: truncateString(primaryError, maxPrimaryErrorBytes),
		}
	}
	// As a second approach, we will try to extract GTest expectation failures.
	// Normally if these were fatal (i.e. GTest assertion failures), the test
	// would have failed immediately and returned them to as the result parts
	// (without crashing). The fact we haven't suggests they were non-fatal
	// failures, or something went wrong in between the fatal error and
	// returning them to us. Nonetheless, any failure is more useful than none
	// in terms of investigating the cause of a test failure.
	match = gtestExpectationRE.FindStringSubmatchIndex(snippet)
	windowsMatch := gtestExpectationWindowsRE.FindStringSubmatchIndex(snippet)
	if match == nil || (windowsMatch != nil && windowsMatch[0] < match[0]) {
		match = windowsMatch
	}
	if match == nil {
		return nil
	}
	message := strings.TrimSpace(snippet[match[2]:match[3]])
	primaryError := trimGoogleTestTrace(message)
	return &pb.FailureReason{
		PrimaryErrorMessage: truncateString(primaryError, maxPrimaryErrorBytes),
	}
}

func (r *GTestResults) convertTestResult(ctx context.Context, buf *bytes.Buffer, testID, name string, result *GTestRunResult) (*sinkpb.TestResult, error) {
	status, expected, err := fromGTestStatus(result.Status)
	if err != nil {
		return nil, err
	}

	tr := &sinkpb.TestResult{
		TestId:   testID,
		Expected: expected,
		Status:   status,
		Tags: pbutil.StringPairs(
			// Store the original Gtest test name.
			"test_name", name,
			// Store the original GTest status.
			"gtest_status", result.Status,
			// Store the correct output snippet.
			"lossless_snippet", strconv.FormatBool(result.LosslessSnippet),
		),
		TestMetadata:  &pb.TestMetadata{Name: name},
		FailureReason: extractFailureReasonFromResultParts(ctx, result.ResultParts),
	}

	// Do not set duration if it is unknown.
	if result.ElapsedTimeMs != 0 {
		tr.Duration = msToDuration(result.ElapsedTimeMs)
	}

	summaryData := map[string]interface{}{}

	// snippet
	if result.OutputSnippetBase64 != "" {
		outputBytes, err := base64.StdEncoding.DecodeString(result.OutputSnippetBase64)
		if err != nil {
			// Log the error, but we shouldn't fail to convert a file just because we can't
			// convert a summary.
			logging.Warningf(ctx, "Failed to convert OutputSnippetBase64 %q", result.OutputSnippetBase64)
		} else {
			failed := status == pb.TestStatus_FAIL || status == pb.TestStatus_CRASH || status == pb.TestStatus_ABORT
			if tr.FailureReason == nil && failed {
				tr.FailureReason = extractFailureReasonFromSnippet(ctx, string(outputBytes))
			}
			tr.Artifacts = map[string]*sinkpb.Artifact{"snippet": {
				Body:        &sinkpb.Artifact_Contents{Contents: outputBytes},
				ContentType: "text/plain",
			}}
			summaryData["text_artifacts"] = []string{"snippet"}
		}
	}

	// links
	if len(result.Links) > 0 {
		links := make(map[string]string, len(result.Links))
		l := new(Link)
		s := new(string)

		for lName, link := range result.Links {
			switch {
			case json.Unmarshal(link, l) == nil:
				links[lName] = l.Content
			case json.Unmarshal(link, s) == nil:
				links[lName] = *s
			default:
				return nil, errors.Reason("unsupported data format for a link: %q", string(link)).Err()
			}
		}
		summaryData["links"] = links
	}

	// Write the summary html
	if len(summaryData) > 0 {
		buf.Reset()
		if err := summaryTmpl.ExecuteTemplate(buf, "gtest", summaryData); err != nil {
			return nil, err
		}
		tr.SummaryHtml = buf.String()
	}

	// Store the test code location.
	if loc, ok := r.TestLocations[name]; ok {
		file := normalizePath(loc.File)
		// For some reason, many file paths start with "../../", followed by
		// the correct path. Strip the prefix.
		file = stripRepeatedPrefixes(file, "../")
		file = ensureLeadingDoubleSlash(file)
		tr.TestMetadata.Location = &pb.TestLocation{
			Repo:     chromiumSrcRepo,
			FileName: file,
			Line:     int32(loc.Line),
		}
	}

	return tr, nil
}

// stripRepeatedPrefixes strips prefixes.
func stripRepeatedPrefixes(str string, prefixes ...string) string {
	for {
		stripped := str
		for _, prefix := range prefixes {
			stripped = strings.TrimPrefix(stripped, prefix)
		}
		if stripped == str {
			return str
		}
		str = stripped
	}
}
