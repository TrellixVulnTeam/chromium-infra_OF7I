// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package ingestion

import (
	"regexp"
	"sort"

	rdbpb "go.chromium.org/luci/resultdb/proto/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	cpb "infra/appengine/weetbix/internal/clustering/proto"
	"infra/appengine/weetbix/pbutil"
	pb "infra/appengine/weetbix/proto/v1"
)

// weetbixExonerationReason matches exoneration explanations containing
// "weetbix" or "findit" (in a case-insenstiive way). It is used to
// detect exonerations based on FindIt or Weetbix analysis.
var weetbixExonerationReason = regexp.MustCompile(`(?i)(findit|weetbix)`)

func failuresFromTestVariants(opts Options, tvs []*rdbpb.TestVariant) []*cpb.Failure {
	var failures []*cpb.Failure
	for _, tv := range tvs {
		// Process results in order of StartTime.
		// This is to ensure test result indexes are later
		// assigned correctly w.r.t the actual execution order.
		results := sortResultsByStartTime(tv.Results)

		// Stores the test run for each test result.
		testRuns := make([]string, len(results))

		// Whether there were any passed or expected results.
		var hasPass bool
		// Whether there were any passed or expected results for a test run
		// (excluding skips).
		testRunHasPass := make(map[string]bool)
		// Total number of results by test run.
		countByTestRun := make(map[string]int64)
		for i, tr := range tv.Results {
			testRun := testRunFromResult(tr.Result)
			testRuns[i] = testRun
			countByTestRun[testRun] += 1

			if tr.Result.Status != rdbpb.TestStatus_SKIP &&
				(tr.Result.Status == rdbpb.TestStatus_PASS ||
					tr.Result.Expected) {
				hasPass = true
				testRunHasPass[testRun] = true
			}
		}

		seqByTestRun := make(map[string]int64)
		for i, tr := range tv.Results {
			testRun := testRuns[i]

			if tr.Result.Expected || !isFailure(tr.Result.Status) {
				// Only unexpected failures are ingested for clustering.
				continue
			}

			hasWeetbixExonerations := false
			hasOtherExonerations := false
			for _, e := range tv.Exonerations {
				if weetbixExonerationReason.MatchString(e.ExplanationHtml) {
					hasWeetbixExonerations = true
				} else {
					hasOtherExonerations = true
				}
			}

			var exoneration pb.ExonerationStatus
			switch {
			case hasOtherExonerations:
				exoneration = pb.ExonerationStatus_EXPLICIT
			case hasWeetbixExonerations:
				exoneration = pb.ExonerationStatus_WEETBIX
			case !hasPass && opts.ImplicitlyExonerateBlockingFailures:
				exoneration = pb.ExonerationStatus_IMPLICIT
			default:
				exoneration = pb.ExonerationStatus_NOT_EXONERATED
			}

			failure := failureFromResult(tv, tr.Result, opts, exoneration, testRun)
			failure.IngestedInvocationResultIndex = int64(i)
			failure.IngestedInvocationResultCount = int64(len(tv.Results))
			failure.IsIngestedInvocationBlocked = !hasPass
			failure.TestRunResultIndex = seqByTestRun[testRun]
			failure.TestRunResultCount = countByTestRun[testRun]
			failure.IsTestRunBlocked = !testRunHasPass[testRun]
			failures = append(failures, failure)

			seqByTestRun[testRun] += 1
		}
	}
	return failures
}

// testRunRe extracts the test run from the ResultDB test result name. This is
// the parent invocation the test result was included in, as distinct from
// the ingested invocation ID.
var testRunRe = regexp.MustCompile(`^invocations/([^/]+)/tests/[^/]+/results/[^/]+$`)

func testRunFromResult(r *rdbpb.TestResult) string {
	match := testRunRe.FindStringSubmatch(r.Name)
	if len(match) == 0 {
		return ""
	}
	return match[1]
}

func isFailure(s rdbpb.TestStatus) bool {
	return (s == rdbpb.TestStatus_ABORT ||
		s == rdbpb.TestStatus_CRASH ||
		s == rdbpb.TestStatus_FAIL)
}

func sortResultsByStartTime(results []*rdbpb.TestResultBundle) []*rdbpb.TestResultBundle {
	// Copy the results to avoid modifying parameter slice, which
	// the caller to IngestFromResultDB may not expect.
	sortedResults := make([]*rdbpb.TestResultBundle, len(results))
	for i, r := range results {
		sortedResults[i] = proto.Clone(r).(*rdbpb.TestResultBundle)
	}

	sort.Slice(sortedResults, func(i, j int) bool {
		aResult := results[i].Result
		bResult := results[j].Result
		aTime := aResult.StartTime.AsTime()
		bTime := bResult.StartTime.AsTime()
		if aTime.Equal(bTime) {
			// If start time the same, order by Result Name.
			// Needed to ensure the output of this sort is
			// deterministic given the input.
			return aResult.Name < bResult.Name
		}
		return aTime.Before(bTime)
	})
	return sortedResults
}

func failureFromResult(tv *rdbpb.TestVariant, tr *rdbpb.TestResult, opts Options, exonerationStatus pb.ExonerationStatus, testRunID string) *cpb.Failure {
	result := &cpb.Failure{
		TestResultId:                  pbutil.TestResultIDFromResultDB(tr.Name),
		PartitionTime:                 timestamppb.New(opts.PartitionTime),
		ChunkIndex:                    -1, // To be populated by chunking.
		Realm:                         opts.Realm,
		TestId:                        tv.TestId,                              // Get from variant, as it is not populated on each result.
		Variant:                       pbutil.VariantFromResultDB(tv.Variant), // Get from variant, as it is not populated on each result.
		VariantHash:                   tv.VariantHash,                         // Get from variant, as it is not populated on each result.
		FailureReason:                 pbutil.FailureReasonFromResultDB(tr.FailureReason),
		BugTrackingComponent:          extractBugTrackingComponent(tr.Tags),
		StartTime:                     tr.StartTime,
		Duration:                      tr.Duration,
		ExonerationStatus:             exonerationStatus,
		IngestedInvocationId:          opts.InvocationID,
		IngestedInvocationResultIndex: -1,    // To be populated by caller.
		IngestedInvocationResultCount: -1,    // To be populated by caller.
		IsIngestedInvocationBlocked:   false, // To be populated by caller.
		TestRunId:                     testRunID,
		TestRunResultIndex:            -1,    // To be populated by caller.
		TestRunResultCount:            -1,    // To be populated by caller.
		IsTestRunBlocked:              false, // To be populated by caller.
		PresubmitRunId:                opts.PresubmitRunID,
		PresubmitRunOwner:             opts.PresubmitRunOwner,
		PresubmitRunCls:               opts.PresubmitRunCls,
	}
	// Copy the result to avoid the result aliasing any of the protos used as input.
	return proto.Clone(result).(*cpb.Failure)
}

func extractBugTrackingComponent(tags []*rdbpb.StringPair) *pb.BugTrackingComponent {
	var value string
	for _, tag := range tags {
		if tag.Key == "monorail_component" {
			value = tag.Value
			break
		}
	}
	if value != "" {
		return &pb.BugTrackingComponent{
			System:    "monorail",
			Component: value,
		}
	}
	return nil
}
