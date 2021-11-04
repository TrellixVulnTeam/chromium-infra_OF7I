// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package ingestion

import (
	"encoding/hex"
	"fmt"
	"sort"
	"testing"
	"time"

	"infra/appengine/weetbix/internal/analysis"
	"infra/appengine/weetbix/internal/analysis/clusteredfailures"
	"infra/appengine/weetbix/internal/clustering"
	"infra/appengine/weetbix/internal/clustering/algorithms/testname"
	"infra/appengine/weetbix/internal/clustering/chunkstore"
	"infra/appengine/weetbix/internal/testutil"
	bqpb "infra/appengine/weetbix/proto/bq"
	pb "infra/appengine/weetbix/proto/v1"

	rdbpb "go.chromium.org/luci/resultdb/proto/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func TestIngest(t *testing.T) {
	Convey(`With Ingestor`, t, func() {
		ctx := testutil.SpannerTestContext(t)
		chunkStore := chunkstore.NewFakeClient()
		clusteredFailures := clusteredfailures.NewFakeClient()
		analysis := analysis.NewClusteringHandler(clusteredFailures)
		ingestor := New(chunkStore, analysis)

		opts := Options{
			Project:        "chromium",
			PartitionTime:  time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC),
			Realm:          "chromium:ci",
			InvocationID:   "build-123456790123456",
			PresubmitRunID: &pb.PresubmitRunId{System: "luci-cv", Id: "cq-run-123"},
		}
		testIngestion := func(input []*rdbpb.TestVariant, expectedCFs []*bqpb.ClusteredFailureRow) {
			ingestion := ingestor.Open(opts)
			err := ingestion.Put(ctx, input)
			So(err, ShouldBeNil)
			err = ingestion.Flush(ctx)
			So(err, ShouldBeNil)

			insertions := clusteredFailures.InsertionsByProject["chromium"]
			So(len(insertions), ShouldEqual, len(expectedCFs))

			// Sort both actuals and expectations by key so that we compare corresponding rows.
			sortClusteredFailures(insertions)
			sortClusteredFailures(expectedCFs)
			for i, exp := range expectedCFs {
				actual := insertions[i]
				So(actual, ShouldNotBeNil)

				// Chunk ID and index is assigned by ingestion.
				copyExp := proto.Clone(exp).(*bqpb.ClusteredFailureRow)
				So(actual.ChunkId, ShouldNotBeEmpty)
				So(actual.ChunkIndex, ShouldBeGreaterThanOrEqualTo, 0)
				copyExp.ChunkId = actual.ChunkId
				copyExp.ChunkIndex = actual.ChunkIndex

				// LastUpdated time is assigned by Spanner.
				So(actual.LastUpdated, ShouldNotBeZeroValue)
				copyExp.LastUpdated = actual.LastUpdated

				So(actual, ShouldResembleProto, copyExp)
			}
		}

		Convey(`Ingest one failure`, func() {
			const uniqifier = 1
			const testRunCount = 1
			const resultsPerTestRun = 1
			tv := newTestVariant(uniqifier, testRunCount, resultsPerTestRun)
			tvs := []*rdbpb.TestVariant{tv}

			// Expect the test result to be clustered by both reason and test name.
			const testRunNum = 0
			const resultNum = 0
			regexpCF := expectedClusteredFailure(uniqifier, testRunCount, testRunNum, resultsPerTestRun, resultNum)
			setRegexpClustered(regexpCF)
			testnameCF := expectedClusteredFailure(uniqifier, testRunCount, testRunNum, resultsPerTestRun, resultNum)
			setTestNameClustered(testnameCF)
			expectedCFs := []*bqpb.ClusteredFailureRow{regexpCF, testnameCF}

			Convey(`Unexpected failure`, func() {
				tv.Results[0].Result.Status = rdbpb.TestStatus_FAIL
				tv.Results[0].Result.Expected = false

				testIngestion(tvs, expectedCFs)
				So(len(chunkStore.Blobs), ShouldEqual, 1)
			})
			Convey(`Expected failure`, func() {
				tv.Results[0].Result.Status = rdbpb.TestStatus_FAIL
				tv.Results[0].Result.Expected = true

				// Expect no test results ingested for an expected
				// failure.
				expectedCFs = nil

				testIngestion(tvs, expectedCFs)
				So(len(chunkStore.Blobs), ShouldEqual, 0)
			})
			Convey(`Unexpected pass`, func() {
				tv.Results[0].Result.Status = rdbpb.TestStatus_PASS
				tv.Results[0].Result.Expected = false

				// Expect no test results ingested for a passed test
				// (even if unexpected).
				expectedCFs = nil
				testIngestion(tvs, expectedCFs)
				So(len(chunkStore.Blobs), ShouldEqual, 0)
			})
			Convey(`Unexpected skip`, func() {
				tv.Results[0].Result.Status = rdbpb.TestStatus_SKIP
				tv.Results[0].Result.Expected = false

				// Expect no test results ingested for a skipped test
				// (even if unexpected).
				expectedCFs = nil

				testIngestion(tvs, expectedCFs)
				So(len(chunkStore.Blobs), ShouldEqual, 0)
			})
			Convey(`Failure without variant`, func() {
				// Tests are allowed to have no variant.
				tv.Variant = nil
				tv.Results[0].Result.Variant = nil

				regexpCF.Variant = nil
				testnameCF.Variant = nil

				testIngestion(tvs, expectedCFs)
				So(len(chunkStore.Blobs), ShouldEqual, 1)
			})
			Convey(`Failure without failure reason`, func() {
				// Failures may not have a failure reason.
				tv.Results[0].Result.FailureReason = nil
				testnameCF.FailureReason = nil
				expectedCFs = []*bqpb.ClusteredFailureRow{testnameCF}

				testIngestion(tvs, expectedCFs)
				So(len(chunkStore.Blobs), ShouldEqual, 1)
			})
			Convey(`Failure without presubmit run`, func() {
				opts.PresubmitRunID = nil
				regexpCF.PresubmitRunId = nil
				testnameCF.PresubmitRunId = nil

				testIngestion(tvs, expectedCFs)
				So(len(chunkStore.Blobs), ShouldEqual, 1)
			})
			Convey(`Failure with exoneration`, func() {
				tv.Exonerations = []*rdbpb.TestExoneration{
					{
						Name:            fmt.Sprintf("invocations/testrun-mytestrun/tests/test-name-%v/exonerations/exon-1", uniqifier),
						TestId:          tv.TestId,
						Variant:         proto.Clone(tv.Variant).(*rdbpb.Variant),
						VariantHash:     "hash",
						ExonerationId:   "exon-1",
						ExplanationHtml: "<p>Known flake affecting CQ</p>",
					},
				}
				testnameCF.IsExonerated = true
				regexpCF.IsExonerated = true

				testIngestion(tvs, expectedCFs)
				So(len(chunkStore.Blobs), ShouldEqual, 1)
			})
		})
		Convey(`Ingest multiple failures`, func() {
			const uniqifier = 1
			const testRunsPerVariant = 2
			const resultsPerTestRun = 2
			tv := newTestVariant(uniqifier, testRunsPerVariant, resultsPerTestRun)
			tvs := []*rdbpb.TestVariant{tv}

			var expectedCFs []*bqpb.ClusteredFailureRow
			var expectedCFsByTestRun [][]*bqpb.ClusteredFailureRow
			for t := 0; t < testRunsPerVariant; t++ {
				var testRunExp []*bqpb.ClusteredFailureRow
				for j := 0; j < resultsPerTestRun; j++ {
					regexpCF := expectedClusteredFailure(uniqifier, testRunsPerVariant, t, resultsPerTestRun, j)
					setRegexpClustered(regexpCF)
					testnameCF := expectedClusteredFailure(uniqifier, testRunsPerVariant, t, resultsPerTestRun, j)
					setTestNameClustered(testnameCF)
					testRunExp = append(testRunExp, regexpCF, testnameCF)
				}
				expectedCFsByTestRun = append(expectedCFsByTestRun, testRunExp)
				expectedCFs = append(expectedCFs, testRunExp...)
			}

			Convey(`Test run and presubmit run blocked`, func() {
				for _, exp := range expectedCFs {
					exp.IsIngestedInvocationBlocked = true
					exp.IsTestRunBlocked = true
				}
				testIngestion(tvs, expectedCFs)
				So(len(chunkStore.Blobs), ShouldEqual, 1)
			})
			Convey(`Some test runs blocked and presubmit run not blocked`, func() {
				// Let the last retry of the last test run pass.
				tv.Results[3].Result.Status = rdbpb.TestStatus_PASS
				// First test run should be blocked.
				for _, exp := range expectedCFsByTestRun[0] {
					exp.IsIngestedInvocationBlocked = false
					exp.IsTestRunBlocked = true
				}
				// Second test run should not be blocked.
				for _, exp := range expectedCFsByTestRun[1] {
					exp.IsIngestedInvocationBlocked = false
					exp.IsTestRunBlocked = false
				}
			})
		})
		Convey(`Ingest many failures`, func() {
			var tvs []*rdbpb.TestVariant
			var expectedCFs []*bqpb.ClusteredFailureRow

			const variantCount = 20
			const testRunsPerVariant = 10
			const resultsPerTestRun = 10
			for uniqifier := 0; uniqifier < variantCount; uniqifier++ {
				tv := newTestVariant(uniqifier, testRunsPerVariant, resultsPerTestRun)
				tvs = append(tvs, tv)
				for t := 0; t < testRunsPerVariant; t++ {
					for j := 0; j < resultsPerTestRun; j++ {
						regexpCF := expectedClusteredFailure(uniqifier, testRunsPerVariant, t, resultsPerTestRun, j)
						setRegexpClustered(regexpCF)
						testnameCF := expectedClusteredFailure(uniqifier, testRunsPerVariant, t, resultsPerTestRun, j)
						setTestNameClustered(testnameCF)
						expectedCFs = append(expectedCFs, regexpCF, testnameCF)
					}
				}
			}
			// Verify more than one chunk is ingested.
			testIngestion(tvs, expectedCFs)
			So(len(chunkStore.Blobs), ShouldBeGreaterThan, 1)
		})
	})
}

func setTestNameClustered(e *bqpb.ClusteredFailureRow) {
	e.ClusterAlgorithm = "testname-v1"
	e.ClusterId = hex.EncodeToString((&testname.Algorithm{}).Cluster(&clustering.Failure{
		TestID: e.TestId,
	}))
}

func setRegexpClustered(e *bqpb.ClusteredFailureRow) {
	e.ClusterAlgorithm = "failurereason-v1"
	e.ClusterId = "5b4886907ba205f9ee2d8815452cb6e7" // Cluster ID for "Failure reason."
}

func sortClusteredFailures(cfs []*bqpb.ClusteredFailureRow) {
	sort.Slice(cfs, func(i, j int) bool {
		return clusteredFailureKey(cfs[i]) < clusteredFailureKey(cfs[j])
	})
}

func clusteredFailureKey(cf *bqpb.ClusteredFailureRow) string {
	return fmt.Sprintf("%q/%q/%q/%q", cf.ClusterAlgorithm, cf.ClusterId, cf.TestResultSystem, cf.TestResultId)
}

func newTestVariant(uniqifier int, testRunCount int, resultsPerTestRun int) *rdbpb.TestVariant {
	testID := fmt.Sprintf("ninja://test_name/%v", uniqifier)
	variant := &rdbpb.Variant{
		Def: map[string]string{
			"k1": "v1",
		},
	}
	tv := &rdbpb.TestVariant{
		TestId:       testID,
		Variant:      variant,
		VariantHash:  "hash",
		Status:       rdbpb.TestVariantStatus_UNEXPECTED,
		Exonerations: nil,
		TestMetadata: &rdbpb.TestMetadata{},
	}
	for i := 0; i < testRunCount; i++ {
		for j := 0; j < resultsPerTestRun; j++ {
			tr := newTestResult(uniqifier, i, j)
			// Test ID, Variant, VariantHash are not populated on the test
			// results of a Test Variant as it is present on the parent record.
			tr.TestId = ""
			tr.Variant = nil
			tr.VariantHash = ""
			tv.Results = append(tv.Results, &rdbpb.TestResultBundle{Result: tr})
		}
	}
	return tv
}

func newTestResult(uniqifier, testRunNum, resultNum int) *rdbpb.TestResult {
	resultID := fmt.Sprintf("result-%v-%v", testRunNum, resultNum)
	return &rdbpb.TestResult{
		Name:        fmt.Sprintf("invocations/testrun-%v/tests/test-name-%v/results/%s", testRunNum, uniqifier, resultID),
		ResultId:    resultID,
		Expected:    false,
		Status:      rdbpb.TestStatus_CRASH,
		SummaryHtml: "<p>Some SummaryHTML</p>",
		StartTime:   timestamppb.New(time.Date(2022, time.February, 12, 0, 0, 0, 0, time.UTC)),
		Duration:    durationpb.New(time.Second * 10),
		Tags: []*rdbpb.StringPair{
			{
				Key:   "monorail_component",
				Value: "Component>MyComponent",
			},
		},
		TestMetadata: &rdbpb.TestMetadata{},
		FailureReason: &rdbpb.FailureReason{
			PrimaryErrorMessage: "Failure reason.",
		},
	}
}

func expectedClusteredFailure(uniqifier, testRunCount, testRunNum, resultsPerTestRun, resultNum int) *bqpb.ClusteredFailureRow {
	resultID := fmt.Sprintf("result-%v-%v", testRunNum, resultNum)
	return &bqpb.ClusteredFailureRow{
		ClusterAlgorithm: "", // Determined by clustering algorithm.
		ClusterId:        "", // Determined by clustering algorithm.
		TestResultSystem: "resultdb",
		TestResultId:     fmt.Sprintf("invocations/testrun-%v/tests/test-name-%v/results/%s", testRunNum, uniqifier, resultID),
		LastUpdated:      nil, // Only known at runtime, Spanner commit timestamp.

		PartitionTime:              timestamppb.New(time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC)),
		IsIncluded:                 true,
		IsIncludedWithHighPriority: true,

		ChunkId:    "",
		ChunkIndex: 0, // To be set by caller as needed.

		Realm:  "chromium:ci",
		TestId: fmt.Sprintf("ninja://test_name/%v", uniqifier),
		Variant: []*pb.StringPair{
			{
				Key:   "k1",
				Value: "v1",
			},
		},
		VariantHash:                   "hash",
		FailureReason:                 &pb.FailureReason{PrimaryErrorMessage: "Failure reason."},
		BugTrackingComponent:          &pb.BugTrackingComponent{System: "monorail", Component: "Component>MyComponent"},
		StartTime:                     timestamppb.New(time.Date(2022, time.February, 12, 0, 0, 0, 0, time.UTC)),
		Duration:                      durationpb.New(time.Second * 10),
		IsExonerated:                  false,
		PresubmitRunId:                &pb.PresubmitRunId{System: "luci-cv", Id: "cq-run-123"},
		IngestedInvocationId:          "build-123456790123456",
		IngestedInvocationResultIndex: int64(testRunNum*resultsPerTestRun + resultNum),
		IngestedInvocationResultCount: int64(testRunCount * resultsPerTestRun),
		IsIngestedInvocationBlocked:   true,
		TestRunId:                     fmt.Sprintf("testrun-%v", testRunNum),
		TestRunResultIndex:            int64(resultNum),
		TestRunResultCount:            int64(resultsPerTestRun),
		IsTestRunBlocked:              true,
	}
}
