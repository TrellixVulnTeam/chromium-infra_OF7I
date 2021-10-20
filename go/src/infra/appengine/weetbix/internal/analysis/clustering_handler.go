// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package analysis

import (
	"context"
	"encoding/hex"
	"time"

	"infra/appengine/weetbix/internal/clustering"
	cpb "infra/appengine/weetbix/internal/clustering/proto"
	bqpb "infra/appengine/weetbix/proto/bq"
	pb "infra/appengine/weetbix/proto/v1"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ClusteringHandler handles test result (re-)clustering events, to
// ensure analysis remains up-to-date.
type ClusteringHandler struct {
	cfClient ClusteredFailuresClient
}

// ClusteredFailuresClient exports clustered failures to BigQuery for
// further analysis.
type ClusteredFailuresClient interface {
	// Insert inserts the given rows into BigQuery.
	Insert(ctx context.Context, luciProject string, rows []*bqpb.ClusteredFailureRow) error
}

func NewClusteringHandler(cf ClusteredFailuresClient) *ClusteringHandler {
	return &ClusteringHandler{
		cfClient: cf,
	}
}

// HandleUpdatedClusters handles (re-)clustered test results. It is called
// after the spanner transaction effecting the (re-)clustering has committed.
// commitTime is the Spanner time the transaction committed.
//
// If this method fails, it will not be retried and data loss or inconsistency
// (in this method's BigQuery export) may occur. This could be improved in
// future with a two-stage apply process (journalling the BigQuery updates
// to be applied as part of the original transaction and retrying them at
// a later point if they do not succeed).
func (r *ClusteringHandler) HandleUpdatedClusters(ctx context.Context, updates *clustering.Update, commitTime time.Time) error {
	rowUpdates := prepareInserts(updates, commitTime)
	return r.cfClient.Insert(ctx, updates.Project, rowUpdates)
}

// prepareInserts prepares entries into the BigQuery clustered failures table
// in response to a (re-)clustering. For efficiency, only the updated rows are
// returned.
func prepareInserts(updates *clustering.Update, commitTime time.Time) []*bqpb.ClusteredFailureRow {
	var result []*bqpb.ClusteredFailureRow
	for _, u := range updates.Updates {
		deleted := make(map[string]*clustering.ClusterID)
		retained := make(map[string]*clustering.ClusterID)
		new := make(map[string]*clustering.ClusterID)

		previousInBugCluster := false
		for _, pc := range u.PreviousClusters {
			deleted[pc.Key()] = pc
			if isBugCluster(pc) {
				previousInBugCluster = true
			}
		}
		newInBugCluster := false
		for _, nc := range u.NewClusters {
			key := nc.Key()
			if _, ok := deleted[key]; ok {
				delete(deleted, key)
				retained[key] = nc
			} else {
				new[key] = nc
			}
			if isBugCluster(nc) {
				newInBugCluster = true
			}
		}
		// Create rows for deletions.
		for _, dc := range deleted {
			isIncluded := false
			isIncludedWithHighPriority := false
			row := entryFromUpdate(updates.Project, updates.ChunkID, dc, u.TestResult, isIncluded, isIncludedWithHighPriority, commitTime)
			result = append(result, row)
		}
		// Create rows for retained clusters for which inclusion was modified.
		for _, rc := range retained {
			isIncluded := true
			// A failure will appear with high priority in any bug clusters
			// it appears in, and if it appears in no bug clusters, it will
			// appear with high priority in any suggested clusters it appears
			// in.
			previousIncludedWithHighPriority := isBugCluster(rc) || !previousInBugCluster
			newIncludedWithHighPriority := isBugCluster(rc) || !newInBugCluster
			if previousIncludedWithHighPriority == newIncludedWithHighPriority {
				// The inclusion status of the test result in the cluster has not changed.
				// For efficiency, do not stream an update.
				continue
			}
			row := entryFromUpdate(updates.Project, updates.ChunkID, rc, u.TestResult, isIncluded, newIncludedWithHighPriority, commitTime)
			result = append(result, row)
		}
		// Create rows for new clusters.
		for _, nc := range new {
			isIncluded := true
			// A failure will appear with high priority in any bug clusters
			// it appears in, and if it appears in no bug clusters, it will
			// appear with high priority in any suggested clusters it appears
			// in.
			isIncludedWithHighPriority := isBugCluster(nc) || !newInBugCluster
			row := entryFromUpdate(updates.Project, updates.ChunkID, nc, u.TestResult, isIncluded, isIncludedWithHighPriority, commitTime)
			result = append(result, row)
		}
	}
	return result
}

func isBugCluster(c *clustering.ClusterID) bool {
	// TODO(crbug.com/1243174): When failure association rules are implemented,
	// return whether the clustering algorithm is the failure association rule
	// clustering algorithm.
	return false
}

func entryFromUpdate(project, chunkID string, cluster *clustering.ClusterID, failure *cpb.Failure, included, includedWithHighPriority bool, commitTime time.Time) *bqpb.ClusteredFailureRow {
	// Copy the failure, to ensure the returned ClusteredFailure does not
	// alias any of the original failure's nested message protos.
	failure = proto.Clone(failure).(*cpb.Failure)
	entry := &bqpb.ClusteredFailureRow{
		ClusterAlgorithm: cluster.Algorithm,
		ClusterId:        hex.EncodeToString(cluster.ID),
		TestResultSystem: failure.TestResultId.System,
		TestResultId:     failure.TestResultId.Id,
		LastUpdated:      timestamppb.New(commitTime),

		PartitionTime: failure.PartitionTime,

		IsIncluded:                 included,
		IsIncludedWithHighPriority: includedWithHighPriority,

		ChunkId:    chunkID,
		ChunkIndex: failure.ChunkIndex,

		Realm:                failure.Realm,
		TestId:               failure.TestId,
		Variant:              variantPairs(failure.Variant),
		VariantHash:          failure.VariantHash,
		FailureReason:        failure.FailureReason,
		BugTrackingComponent: failure.BugTrackingComponent,
		StartTime:            failure.StartTime,
		Duration:             failure.Duration,
		IsExonerated:         failure.IsExonerated,

		PresubmitRunId:                failure.PresubmitRunId,
		IngestedInvocationId:          failure.IngestedInvocationId,
		IngestedInvocationResultIndex: failure.IngestedInvocationResultIndex,
		IngestedInvocationResultCount: failure.IngestedInvocationResultCount,
		IsIngestedInvocationBlocked:   failure.IsIngestedInvocationBlocked,
		TestRunId:                     failure.TestRunId,
		TestRunResultIndex:            failure.TestRunResultIndex,
		TestRunResultCount:            failure.TestRunResultCount,
		IsTestRunBlocked:              failure.IsTestRunBlocked,
	}
	return entry
}

func variantPairs(v *pb.Variant) []*pb.StringPair {
	var result []*pb.StringPair
	for k, v := range v.Def {
		result = append(result, &pb.StringPair{
			Key:   k,
			Value: v,
		})
	}
	return result
}
