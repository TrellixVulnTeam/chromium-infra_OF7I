// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package reclustering

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"infra/appengine/weetbix/internal/analysis"
	"infra/appengine/weetbix/internal/analysis/clusteredfailures"
	"infra/appengine/weetbix/internal/clustering"
	"infra/appengine/weetbix/internal/clustering/algorithms"
	"infra/appengine/weetbix/internal/clustering/algorithms/failurereason"
	"infra/appengine/weetbix/internal/clustering/algorithms/rulesalgorithm"
	"infra/appengine/weetbix/internal/clustering/algorithms/testname"
	cpb "infra/appengine/weetbix/internal/clustering/proto"
	"infra/appengine/weetbix/internal/clustering/rules"
	"infra/appengine/weetbix/internal/clustering/rules/cache"
	"infra/appengine/weetbix/internal/clustering/runs"
	"infra/appengine/weetbix/internal/clustering/state"
	spanutil "infra/appengine/weetbix/internal/span"
	"infra/appengine/weetbix/internal/tasks/taskspb"
	"infra/appengine/weetbix/internal/testutil"
	"infra/appengine/weetbix/pbutil"
	bqpb "infra/appengine/weetbix/proto/bq"
	pb "infra/appengine/weetbix/proto/v1"

	"cloud.google.com/go/spanner"

	"go.chromium.org/luci/server/caching"
	"go.chromium.org/luci/server/span"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/luci/common/clock/testclock"
	. "go.chromium.org/luci/common/testing/assertions"
)

const testProject = "testproject"

// scenario represents a Weetbix system state used for testing.
type scenario struct {
	// clusteringState stores the test result-cluster inclusions
	// for each test result in each chunk, and related metadata.
	clusteringState []*state.Entry
	// netBQExports are the test result-cluster insertions recorded
	// in BigQuery, net of any deletions/updates.
	netBQExports []*bqpb.ClusteredFailureRow
	// rulesVersion is last updated time of the most recently updated
	// failure association rule.
	rulesVersion time.Time
	// rules are the failure association rules.
	rules []*rules.FailureAssociationRule
	// testResults are the actual test failures ingested by Weetbix,
	// organised in chunks by chunk ID.
	testResultsByChunkID map[string]*cpb.Chunk
}

func TestIngest(t *testing.T) {
	Convey(`With Worker`, t, func() {
		ctx := testutil.SpannerTestContext(t)
		ctx, tc := testclock.UseTime(ctx, testclock.TestRecentTimeUTC)
		ctx = caching.WithEmptyProcessCache(ctx) // For rules cache.

		chunkStore := &fakeChunkStore{
			chunksByID: make(map[string]*cpb.Chunk),
		}
		clusteredFailures := clusteredfailures.NewFakeClient()
		analysis := analysis.NewClusteringHandler(clusteredFailures)

		worker := NewWorker(chunkStore, analysis)

		attemptTime := tc.Now().Add(time.Minute * 10)
		run := &runs.ReclusteringRun{
			Project:           testProject,
			AttemptTimestamp:  attemptTime,
			AlgorithmsVersion: algorithms.AlgorithmsVersion,
			RulesVersion:      time.Time{}, // To be set by the test.
			ShardCount:        1,
			ShardsReported:    0,
			Progress:          0,
		}
		task := &taskspb.ReclusterChunks{
			Project:      testProject,
			AttemptTime:  timestamppb.New(attemptTime),
			StartChunkId: "",
			EndChunkId:   state.EndOfTable,
		}

		setupScenario := func(s *scenario) {
			// Create the run entry corresponding to the task.
			run.RulesVersion = s.rulesVersion
			So(runs.SetRunsForTesting(ctx, []*runs.ReclusteringRun{run}), ShouldBeNil)

			// Set stored failure association rules.
			So(rules.SetRulesForTesting(ctx, s.rules), ShouldBeNil)

			// Set stored test result chunks.
			chunkStore.chunksByID = s.testResultsByChunkID

			// Set stored clustering state.
			commitTime, err := state.CreateEntriesForTesting(ctx, s.clusteringState)
			for _, e := range s.clusteringState {
				e.LastUpdated = commitTime.In(time.UTC)
			}
			So(err, ShouldBeNil)
		}

		Convey(`No re-clustering`, func() {
			// Start with up-to-date clustering.
			s := newScenario().withOldClustering(false).build()
			setupScenario(s)

			// Run the task.
			err := worker.Do(ctx, task)
			So(err, ShouldBeNil)

			// Final clustering state should be equal starting state.
			actualState, err := state.ReadAllForTesting(ctx, testProject)
			So(err, ShouldBeNil)
			So(actualState, ShouldResemble, s.clusteringState)

			// No changes written to BigQuery.
			So(clusteredFailures.InsertionsByProject[testProject], ShouldBeEmpty)

			// Run is reported as complete.
			actualRun, err := runs.Read(span.Single(ctx), testProject, run.AttemptTimestamp)
			So(err, ShouldBeNil)
			So(actualRun.Progress, ShouldEqual, 1000)
		})
		Convey(`Significant re-clustering`, func() {
			expected := newScenario().withOldClustering(false).build()

			// Start with an out of date clustering.
			s := newScenario().withOldClustering(true).build()
			s.rules = expected.rules
			s.rulesVersion = expected.rulesVersion
			setupScenario(s)

			// Run the task.
			err := worker.Do(ctx, task)
			So(err, ShouldBeNil)

			// Final clustering state should be equal expected state.
			actualState, err := state.ReadAllForTesting(ctx, testProject)
			So(err, ShouldBeNil)
			for _, as := range actualState {
				// Clear last updated time to compare actual vs expected
				// state based on row contents, not when the row was updated.
				as.LastUpdated = time.Time{}
			}
			So(actualState, ShouldResemble, expected.clusteringState)

			// BigQuery exports should correctly reflect the new
			// test result-cluster inclusions.
			exports := clusteredFailures.InsertionsByProject[testProject]
			sortBQExport(exports)
			netExports := flattenBigQueryExports(append(s.netBQExports, exports...))
			So(netExports, ShouldResembleProto, expected.netBQExports)

			// Run is reported as complete.
			actualRun, err := runs.Read(span.Single(ctx), testProject, run.AttemptTimestamp)
			So(err, ShouldBeNil)
			So(actualRun.Progress, ShouldEqual, 1000)
		})
		Convey(`Minor re-clustering`, func() {
			s := newScenario().withOldClustering(false).build()

			// Add a new rule that needs to be matched against all test
			// results. (For simplicity, its definition is FALSE, which
			// means our expected clustering state need not be modified.)
			rule3 := rules.NewRule(100).
				WithProject(testProject).
				WithRuleDefinition("FALSE").
				WithLastUpdated(s.rulesVersion.Add(1 * time.Hour)).
				Build()
			s.rules = append(s.rules, rule3)
			s.rulesVersion = s.rulesVersion.Add(1 * time.Hour)

			setupScenario(s)

			// Run the task.
			err := worker.Do(ctx, task)
			So(err, ShouldBeNil)

			// Final clustering state should be equal starting state,
			// except that RulesVersion should now be later (and
			// LastUpdated may have changed).
			expected := newScenario().withOldClustering(false).build()
			for _, es := range expected.clusteringState {
				es.Clustering.RulesVersion = s.rulesVersion
			}
			actualState, err := state.ReadAllForTesting(ctx, testProject)
			So(err, ShouldBeNil)
			for _, as := range actualState {
				as.LastUpdated = time.Time{}
			}
			So(actualState, ShouldResemble, expected.clusteringState)

			// No changes written to BigQuery.
			So(clusteredFailures.InsertionsByProject[testProject], ShouldBeEmpty)

			// Run is reported as complete.
			actualRun, err := runs.Read(span.Single(ctx), testProject, run.AttemptTimestamp)
			So(err, ShouldBeNil)
			So(actualRun.Progress, ShouldEqual, 1000)
		})
		Convey(`Worker respects end time`, func() {
			expected := newScenario().withOldClustering(false).build()

			// Start with an out of date clustering.
			s := newScenario().withOldClustering(true).build()
			s.rules = expected.rules
			s.rulesVersion = expected.rulesVersion
			setupScenario(s)

			// Start the worker after the attempt time.
			tc.Add(11 * time.Minute)
			So(tc.Now(), ShouldHappenAfter, run.AttemptTimestamp)

			// Run the task.
			err := worker.Do(ctx, task)
			So(err, ShouldBeNil)

			// Clustering state should be same as the initial state.
			actualState, err := state.ReadAllForTesting(ctx, testProject)
			So(err, ShouldBeNil)
			So(actualState, ShouldResemble, s.clusteringState)

			// No changes written to BigQuery.
			So(clusteredFailures.InsertionsByProject[testProject], ShouldBeEmpty)

			// No progress is reported.
			actualRun, err := runs.Read(span.Single(ctx), testProject, run.AttemptTimestamp)
			So(err, ShouldBeNil)
			So(actualRun.Progress, ShouldEqual, 0)
		})
		Convey(`Handles update/update races`, func() {
			finalState := newScenario().withOldClustering(false).build()

			// Start with an out of date clustering.
			s := newScenario().withOldClustering(true).build()
			s.rules = finalState.rules
			s.rulesVersion = finalState.rulesVersion
			setupScenario(s)

			// Make reading a chunk's test results trigger updating
			// its clustering state Spanner, to simulate an update/update race.
			chunkIDByObjectID := make(map[string]string)
			for _, state := range s.clusteringState {
				chunkIDByObjectID[state.ObjectID] = state.ChunkID
			}
			chunkStore.cb = func(objectID string) {
				chunkID, ok := chunkIDByObjectID[objectID]

				// Only simulate the update/update race once per chunk.
				if !ok {
					return
				}
				delete(chunkIDByObjectID, objectID)

				_, err := span.ReadWriteTransaction(ctx, func(ctx context.Context) error {
					span.BufferWrite(ctx, spanutil.UpdateMap("ClusteringState", map[string]interface{}{
						"Project": testProject,
						"ChunkID": chunkID,
						// Simulate a race with another update, that
						// re-clustered the chunk to an algorithms version
						// later than the one we know about.
						"AlgorithmsVersion": algorithms.AlgorithmsVersion + 1,
						"LastUpdated":       spanner.CommitTimestamp,
					}))
					return nil
				})
				So(err, ShouldBeNil)
			}

			// Run the worker with time advancing, as the transaction retry
			// logic sets timers which must be triggered.
			runWithTimeAdvancing(tc, func() {
				err := worker.Do(ctx, task)
				So(err, ShouldBeNil)
			})

			// Because of update races, none of the chunks should have been
			// re-clustered further.
			expected := newScenario().withOldClustering(true).build()
			for _, es := range expected.clusteringState {
				es.Clustering.AlgorithmsVersion = algorithms.AlgorithmsVersion + 1
			}

			actualState, err := state.ReadAllForTesting(ctx, testProject)
			So(err, ShouldBeNil)
			for _, as := range actualState {
				as.LastUpdated = time.Time{}
			}
			So(actualState, ShouldResemble, expected.clusteringState)

			// No changes written to BigQuery.
			So(clusteredFailures.InsertionsByProject[testProject], ShouldBeEmpty)

			// Run is reported as complete.
			actualRun, err := runs.Read(span.Single(ctx), testProject, run.AttemptTimestamp)
			So(err, ShouldBeNil)
			So(actualRun.Progress, ShouldEqual, 1000)
		})
	})
}

func TestProgress(t *testing.T) {
	Convey(`Task assigned entire keyspace`, t, func() {
		task := &taskspb.ReclusterChunks{
			StartChunkId: "",
			EndChunkId:   strings.Repeat("ff", 16),
		}

		progress, err := calculateProgress(task, strings.Repeat("00", 16))
		So(err, ShouldBeNil)
		So(progress, ShouldEqual, 0)

		progress, err = calculateProgress(task, "80"+strings.Repeat("00", 15))
		So(err, ShouldBeNil)
		So(progress, ShouldEqual, 500)

		progress, err = calculateProgress(task, strings.Repeat("ff", 16))
		So(err, ShouldBeNil)
		So(progress, ShouldEqual, 999)
	})
	Convey(`Task assigned partial keyspace`, t, func() {
		// Consistent with the second shard, if the keyspace is split into
		// three.
		task := &taskspb.ReclusterChunks{
			StartChunkId: strings.Repeat("55", 15) + "54",
			EndChunkId:   strings.Repeat("aa", 15) + "a9",
		}

		progress, err := calculateProgress(task, strings.Repeat("55", 16))
		So(err, ShouldBeNil)
		So(progress, ShouldEqual, 0)

		progress, err = calculateProgress(task, strings.Repeat("77", 16))
		So(err, ShouldBeNil)
		So(progress, ShouldEqual, 400)

		progress, err = calculateProgress(task, strings.Repeat("aa", 15)+"a9")
		So(err, ShouldBeNil)
		So(progress, ShouldEqual, 999)
	})
}

func runWithTimeAdvancing(tc testclock.TestClock, cb func()) {
	ticker := time.NewTicker(time.Millisecond)
	done := make(chan bool)
	go func() {
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				// Run with time advancing at 100 times speed, to
				// avoid holding up tests unnecessarily.
				tc.Add(time.Millisecond * 100)
			}
		}
	}()

	cb()

	ticker.Stop()
	done <- true
}

// flattenBigQueryExports returns the latest inclusion row for
// each test result-cluster, from a list of BigQuery exports.
// The returned set of rows do not have last updated time set.
func flattenBigQueryExports(exports []*bqpb.ClusteredFailureRow) []*bqpb.ClusteredFailureRow {
	keyValue := make(map[string]*bqpb.ClusteredFailureRow)
	for _, row := range exports {
		key := bigQueryKey(row)
		existingRow, ok := keyValue[key]
		if ok && existingRow.LastUpdated.AsTime().After(row.LastUpdated.AsTime()) {
			continue
		}
		keyValue[key] = row
	}
	var result []*bqpb.ClusteredFailureRow
	for _, row := range keyValue {
		if row.IsIncluded {
			clonedRow := proto.Clone(row).(*bqpb.ClusteredFailureRow)
			clonedRow.LastUpdated = nil
			result = append(result, clonedRow)
		}
	}
	sortBQExport(result)
	return result
}

func bigQueryKey(row *bqpb.ClusteredFailureRow) string {
	return fmt.Sprintf("%q/%q/%q/%q", row.ClusterAlgorithm, row.ClusterId, row.TestResultSystem, row.TestResultId)
}

type testResultBuilder struct {
	uniqifier     int
	failureReason *pb.FailureReason
	testName      string
}

func newTestResult(uniqifier int) *testResultBuilder {
	return &testResultBuilder{
		uniqifier: uniqifier,
		testName:  fmt.Sprintf("ninja://test_name/%v", uniqifier),
		failureReason: &pb.FailureReason{
			PrimaryErrorMessage: fmt.Sprintf("Failure reason %v.", uniqifier),
		},
	}
}

func (b *testResultBuilder) withTestName(name string) *testResultBuilder {
	b.testName = name
	return b
}

func (b *testResultBuilder) withFailureReason(reason *pb.FailureReason) *testResultBuilder {
	b.failureReason = reason
	return b
}

func (b *testResultBuilder) buildFailure() *cpb.Failure {
	keyHash := sha256.Sum256([]byte("variantkey:value\n"))
	return &cpb.Failure{
		TestResultId:  pbutil.TestResultIDFromResultDB(fmt.Sprintf("invocations/testrun-%v/tests/test-name-%v/results/%v", b.uniqifier, b.uniqifier, b.uniqifier)),
		PartitionTime: timestamppb.New(time.Date(2020, time.April, 1, 2, 3, 4, 0, time.UTC)),
		ChunkIndex:    -1, // To be populated by caller.
		Realm:         "testproject:realm",
		TestId:        b.testName,
		Variant:       &pb.Variant{Def: map[string]string{"variantkey": "value"}},
		VariantHash:   hex.EncodeToString(keyHash[:]),
		FailureReason: b.failureReason,
		BugTrackingComponent: &pb.BugTrackingComponent{
			System:    "monorail",
			Component: "Component>MyComponent",
		},
		StartTime:                     timestamppb.New(time.Date(2025, time.March, 2, 2, 2, 2, b.uniqifier, time.UTC)),
		Duration:                      durationpb.New(time.Duration(b.uniqifier) * time.Second),
		IsExonerated:                  b.uniqifier%4 == 0,
		IngestedInvocationId:          fmt.Sprintf("invocation-%v", b.uniqifier),
		IngestedInvocationResultIndex: int64(b.uniqifier + 1),
		IngestedInvocationResultCount: int64(b.uniqifier*2 + 1),
		IsIngestedInvocationBlocked:   b.uniqifier%3 == 0,
		TestRunId:                     fmt.Sprintf("test-run-%v", b.uniqifier),
		TestRunResultIndex:            int64((int64(b.uniqifier) / 2) + 1),
		TestRunResultCount:            int64(b.uniqifier + 1),
		IsTestRunBlocked:              b.uniqifier%2 == 0,
		PresubmitRunId: &pb.PresubmitRunId{
			System: "luci-cv",
			Id:     fmt.Sprintf("run-%v", b.uniqifier),
		},
	}
}

// buildBQExport returns the expected test result-cluster inclusion rows that
// would appear in BigQuery, if the test result was in the given clusters.
// Note that deletions are not returned; these are simply the 'net' rows that
// would be expected.
func (b *testResultBuilder) buildBQExport(clusterIDs []*clustering.ClusterID) []*bqpb.ClusteredFailureRow {
	keyHash := sha256.Sum256([]byte("variantkey:value\n"))
	var inBugCluster bool
	for _, cID := range clusterIDs {
		if cID.IsBugCluster() {
			inBugCluster = true
		}
	}

	var results []*bqpb.ClusteredFailureRow
	for _, cID := range clusterIDs {
		result := &bqpb.ClusteredFailureRow{
			ClusterAlgorithm: cID.Algorithm,
			ClusterId:        cID.ID,
			TestResultSystem: "resultdb",
			TestResultId:     fmt.Sprintf("invocations/testrun-%v/tests/test-name-%v/results/%v", b.uniqifier, b.uniqifier, b.uniqifier),
			LastUpdated:      nil, // To be set by caller.

			PartitionTime:              timestamppb.New(time.Date(2020, time.April, 1, 2, 3, 4, 0, time.UTC)),
			IsIncluded:                 true,
			IsIncludedWithHighPriority: cID.IsBugCluster() || !inBugCluster,

			ChunkId:    "", // To be set by caller.
			ChunkIndex: 0,  // To be set by caller.

			Realm:  "testproject:realm",
			TestId: b.testName,
			Variant: []*pb.StringPair{
				{
					Key:   "variantkey",
					Value: "value",
				},
			},
			VariantHash:          hex.EncodeToString(keyHash[:]),
			FailureReason:        b.failureReason,
			BugTrackingComponent: &pb.BugTrackingComponent{System: "monorail", Component: "Component>MyComponent"},
			StartTime:            timestamppb.New(time.Date(2025, time.March, 2, 2, 2, 2, b.uniqifier, time.UTC)),
			Duration:             durationpb.New(time.Duration(b.uniqifier) * time.Second),
			IsExonerated:         b.uniqifier%4 == 0,

			PresubmitRunId: &pb.PresubmitRunId{
				System: "luci-cv",
				Id:     fmt.Sprintf("run-%v", b.uniqifier),
			},
			IngestedInvocationId:          fmt.Sprintf("invocation-%v", b.uniqifier),
			IngestedInvocationResultIndex: int64(b.uniqifier + 1),
			IngestedInvocationResultCount: int64(b.uniqifier*2 + 1),
			IsIngestedInvocationBlocked:   b.uniqifier%3 == 0,

			TestRunId:          fmt.Sprintf("test-run-%v", b.uniqifier),
			TestRunResultIndex: int64((int64(b.uniqifier) / 2) + 1),
			TestRunResultCount: int64(b.uniqifier + 1),
			IsTestRunBlocked:   b.uniqifier%2 == 0,
		}
		results = append(results, result)
	}
	return results
}

// buildClusters returns the clusters that would be expected for this test
// result, if current clustering algorithms were used.
func (b *testResultBuilder) buildClusters(rules *cache.Ruleset) []*clustering.ClusterID {
	var clusters []*clustering.ClusterID
	failure := &clustering.Failure{
		TestID: b.testName,
		Reason: b.failureReason,
	}
	testNameAlg := &testname.Algorithm{}
	clusters = append(clusters, &clustering.ClusterID{
		Algorithm: testNameAlg.Name(),
		ID:        hex.EncodeToString(testNameAlg.Cluster(failure)),
	})
	if b.failureReason != nil && b.failureReason.PrimaryErrorMessage != "" {
		failureReasonAlg := &failurereason.Algorithm{}
		clusters = append(clusters, &clustering.ClusterID{
			Algorithm: failureReasonAlg.Name(),
			ID:        hex.EncodeToString(failureReasonAlg.Cluster(failure)),
		})
	}
	vals := map[string]string{
		"test":   b.testName,
		"reason": b.failureReason.GetPrimaryErrorMessage(),
	}
	for _, rule := range rules.ActiveRulesSorted {
		if rule.Expr.Evaluate(vals) {
			clusters = append(clusters, &clustering.ClusterID{
				Algorithm: rulesalgorithm.AlgorithmName,
				ID:        rule.RuleID,
			})
		}
	}
	clustering.SortClusters(clusters)
	return clusters
}

// chunkBuilder is used to build a chunk with test results, clustering state
// and BigQuery exports, for testing.
type chunkBuilder struct {
	project       string
	chunkID       string
	objectID      string
	testResults   []*testResultBuilder
	ruleset       *cache.Ruleset
	oldClustering bool
}

// newChunk returns a new chunkBuilder for creating a new chunk. Uniqifier
// is used to generate a chunk ID.
func newChunk(uniqifier int) *chunkBuilder {
	chunkID := sha256.Sum256([]byte(fmt.Sprintf("chunk-%v", uniqifier)))
	objectID := sha256.Sum256([]byte(fmt.Sprintf("object-%v", uniqifier)))
	return &chunkBuilder{
		project:       "testproject",
		chunkID:       hex.EncodeToString(chunkID[:16]),
		objectID:      hex.EncodeToString(objectID[:16]),
		ruleset:       cache.NewRuleset("", nil, rules.StartingEpoch, time.Time{}),
		oldClustering: false,
	}
}

func (b *chunkBuilder) withProject(project string) *chunkBuilder {
	b.project = project
	return b
}

func (b *chunkBuilder) withTestResults(tr ...*testResultBuilder) *chunkBuilder {
	b.testResults = tr
	return b
}

// withOldClustering sets whether old clusters (i.e. out of date algorithms
// and rules) should be used instead of current clustering.
func (b *chunkBuilder) withOldClustering(old bool) *chunkBuilder {
	b.oldClustering = old
	return b
}

// withRuleset sets the ruleset to use to determine current clustering
// (only used if out-of-date clustering is not set).
func (b *chunkBuilder) withRuleset(ruleset *cache.Ruleset) *chunkBuilder {
	b.ruleset = ruleset
	return b
}

func (b *chunkBuilder) buildTestResults() (chunk *cpb.Chunk) {
	var failures []*cpb.Failure
	for i, tr := range b.testResults {
		failure := tr.buildFailure()
		failure.ChunkIndex = int64(i)
		failures = append(failures, failure)
	}
	return &cpb.Chunk{
		Failures: failures,
	}
}

func (b *chunkBuilder) buildState() *state.Entry {
	var crs clustering.ClusterResults
	if b.oldClustering {
		algs := make(map[string]struct{})
		algs["testname-v0"] = struct{}{}
		algs["rules-v0"] = struct{}{}
		var clusters [][]*clustering.ClusterID
		for range b.testResults {
			cs := []*clustering.ClusterID{
				{
					Algorithm: "testname-v0",
					ID:        "01dc151e01dc151e01dc151e01dc151e",
				},
				{
					Algorithm: "rules-v0",
					ID:        "12341234123412341234123412341234",
				},
			}
			clusters = append(clusters, cs)
		}
		crs = clustering.ClusterResults{
			AlgorithmsVersion: 1,
			RulesVersion:      b.ruleset.RulesVersion,
			Algorithms:        algs,
			Clusters:          clusters,
		}
	} else {
		algs := make(map[string]struct{})
		algs[testname.AlgorithmName] = struct{}{}
		algs[failurereason.AlgorithmName] = struct{}{}
		algs[rulesalgorithm.AlgorithmName] = struct{}{}
		var clusters [][]*clustering.ClusterID
		for _, tr := range b.testResults {
			clusters = append(clusters, tr.buildClusters(b.ruleset))
		}
		crs = clustering.ClusterResults{
			AlgorithmsVersion: algorithms.AlgorithmsVersion,
			RulesVersion:      b.ruleset.RulesVersion,
			Algorithms:        algs,
			Clusters:          clusters,
		}
	}

	return &state.Entry{
		Project:       b.project,
		ChunkID:       b.chunkID,
		PartitionTime: time.Date(2020, time.April, 1, 2, 3, 4, 0, time.UTC),
		ObjectID:      b.objectID,
		Clustering:    crs,
	}
}

func (b *chunkBuilder) buildBQExport() []*bqpb.ClusteredFailureRow {
	state := b.buildState()
	var result []*bqpb.ClusteredFailureRow
	for i, tr := range b.testResults {
		cIDs := state.Clustering.Clusters[i]
		rows := tr.buildBQExport(cIDs)
		for _, r := range rows {
			r.ChunkId = b.chunkID
			r.ChunkIndex = int64(i)
		}
		result = append(result, rows...)
	}
	return result
}

// scenarioBuilder is used to generate Weetbix system states used for testing.
// Each scenario represents a consistent state of the Weetbix system, i.e.
// - where the clustering state matches the configured rules, and
// - the BigQuery exports match the clustering state, and the test results
//   in the chunk store.
type scenarioBuilder struct {
	project       string
	chunkCount    int
	oldClustering bool
}

func newScenario() *scenarioBuilder {
	return &scenarioBuilder{
		project:    testProject,
		chunkCount: 2,
	}
}

func (b *scenarioBuilder) withOldClustering(value bool) *scenarioBuilder {
	b.oldClustering = value
	return b
}

func (b *scenarioBuilder) build() *scenario {
	var rs []*rules.FailureAssociationRule
	var activeRules []*cache.CachedRule
	rulesVersion := rules.StartingEpoch
	if !b.oldClustering {
		rulesVersion = time.Date(2020, time.August, 1, 8, 1, 8, 1000, time.UTC)
		ruleOne := rules.NewRule(0).WithProject(b.project).WithRuleDefinition(`test = "test_b"`).WithLastUpdated(rulesVersion).Build()
		ruleTwo := rules.NewRule(1).WithProject(b.project).WithRuleDefinition(`reason = "reason_b"`).WithLastUpdated(rulesVersion.Add(-1 * time.Hour)).Build()
		rs = []*rules.FailureAssociationRule{ruleOne, ruleTwo}

		for _, r := range rs {
			active, err := cache.NewCachedRule(r)
			So(err, ShouldBeNil)
			activeRules = append(activeRules, active)
		}
	}

	ruleset := cache.NewRuleset(b.project, activeRules, rulesVersion, time.Time{})

	var state []*state.Entry
	testResultsByChunkID := make(map[string]*cpb.Chunk)
	var bqExports []*bqpb.ClusteredFailureRow
	for i := 0; i < b.chunkCount; i++ {
		trOne := newTestResult(i * 2).withFailureReason(&pb.FailureReason{
			PrimaryErrorMessage: "reason_a",
		}).withTestName("test_a")
		trTwo := newTestResult(i*2 + 1).withFailureReason(&pb.FailureReason{
			PrimaryErrorMessage: "reason_b",
		}).withTestName("test_b")

		cb := newChunk(i).withProject(b.project).
			withOldClustering(b.oldClustering).
			withRuleset(ruleset).
			withTestResults(trOne, trTwo)

		s := cb.buildState()
		state = append(state, s)
		bqExports = append(bqExports, cb.buildBQExport()...)
		testResultsByChunkID[s.ObjectID] = cb.buildTestResults()
	}
	sortState(state)
	sortBQExport(bqExports)
	return &scenario{
		rulesVersion:         rulesVersion,
		rules:                rs,
		testResultsByChunkID: testResultsByChunkID,
		clusteringState:      state,
		netBQExports:         bqExports,
	}
}

// sortState sorts state.Entry elements in ascending ChunkID order.
func sortState(state []*state.Entry) {
	sort.Slice(state, func(i, j int) bool {
		return state[i].ChunkID < state[j].ChunkID
	})
}

// sortBQExport sorts BigQuery export rows in ascending key order.
func sortBQExport(rows []*bqpb.ClusteredFailureRow) {
	sort.Slice(rows, func(i, j int) bool {
		return bigQueryKey(rows[i]) < bigQueryKey(rows[j])
	})
}

// fakeChunkStore is a fake implementation of chunk store, for testing.
type fakeChunkStore struct {
	// The content of the store, by object ID.
	chunksByID map[string]*cpb.Chunk
	// A callback function to be called during Get(...). This allows
	// the test to change the environment during the processing of
	// a particular chunk.
	cb func(objectID string)
}

func (cs *fakeChunkStore) Get(ctx context.Context, project, objectID string) (*cpb.Chunk, error) {
	object := cs.chunksByID[objectID]
	if object == nil || project != testProject {
		return nil, errors.New("does not exist")
	}
	if cs.cb != nil {
		cs.cb(objectID)
	}

	return proto.Clone(object).(*cpb.Chunk), nil
}
