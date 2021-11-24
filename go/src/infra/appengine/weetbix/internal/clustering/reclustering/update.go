// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package reclustering

import (
	"context"
	"fmt"
	"time"

	"infra/appengine/weetbix/internal/clustering"
	"infra/appengine/weetbix/internal/clustering/algorithms"
	cpb "infra/appengine/weetbix/internal/clustering/proto"
	"infra/appengine/weetbix/internal/clustering/rules/cache"
	"infra/appengine/weetbix/internal/clustering/state"

	"go.chromium.org/luci/server/caching"
	"go.chromium.org/luci/server/span"
)

// TODO(crbug.com/1243174). Instrument the size of this cache so that we
// can monitor it.
var rulesCache = cache.NewRulesCache(caching.RegisterLRUCache(0))

// Ruleset returns the cached ruleset for the given project. If a minimum
// rules version is required, pass it as the minimumVersion. (Or otherwise,
// use time.Time{}).
func Ruleset(ctx context.Context, project string, minimumVersion time.Time) (*cache.Ruleset, error) {
	ruleset, err := rulesCache.Ruleset(ctx, project, minimumVersion)
	if err != nil {
		return nil, err
	}
	return ruleset, nil
}

// Analysis is the interface for cluster analysis.
type Analysis interface {
	// HandleUpdatedClusters handles (re-)clustered test results. It is called
	// after the spanner transaction effecting the (re-)clustering has
	// committed. commitTime is the Spanner time the transaction committed.
	HandleUpdatedClusters(ctx context.Context, updates *clustering.Update, commitTime time.Time) error
}

// Update will (re-)cluster the specific chunk of test results,
// committing the updated state to Spanner and exporting analysis.
//
// If the chunk does not exist in Spanner, pass a *state.Entry
// with project, chunkID, objectID and partitionTime set
// but with LastUpdated set to its zero value. The chunk will be
// clustered for the first time and saved to Spanner.
//
// If the chunk does exist in Spanner, pass the state.Entry read
// from Spanner, along with the test results. The chunk will be
// re-clustered and updated.
//
// In case of an update race, the chunk will not be modified, and
// the method returns state.UpdateRaceErr. The method should be retried
// with a more recent version of the chunk in this case.
//
// In case of a create conflicting with another create, the method
// will also fail (with spanner.ErrCode(err) == codes.AlreadyExists)
// but there is no need to retry as the chunk will have already been
// inserted.
func Update(ctx context.Context, ruleset *cache.Ruleset, analysis Analysis, chunk *cpb.Chunk, existingState *state.Entry) error {
	exists := !existingState.LastUpdated.IsZero()
	var existingClustering clustering.ClusterResults
	if !exists {
		existingClustering = algorithms.NewEmptyClusterResults(len(chunk.Failures))
	} else {
		if len(existingState.Clustering.Clusters) != len(chunk.Failures) {
			return fmt.Errorf("existing clustering does not match chunk; got clusters for %v test results, want %v", len(existingClustering.Clusters), len(chunk.Failures))
		}
		existingClustering = existingState.Clustering
	}

	newClustering := algorithms.Cluster(ruleset, existingClustering, clustering.FailuresFromProtos(chunk.Failures))

	f := func(ctx context.Context) error {
		if !exists {
			clusterState := &state.Entry{
				Project:       existingState.Project,
				ChunkID:       existingState.ChunkID,
				PartitionTime: existingState.PartitionTime,
				ObjectID:      existingState.ObjectID,
				Clustering:    newClustering,
			}
			// In case the row already exists, the enclosing ReadWriteTransaction
			// returns an err with spanner.ErrCode(err) == codes.AlreadyExists.
			if err := state.Create(ctx, clusterState); err != nil {
				return err
			}
		} else {
			// If an update race is detected, returns state.UpdateRaceErr.
			if err := state.UpdateClustering(ctx, existingState, &newClustering); err != nil {
				return err
			}
		}
		return nil
	}
	commitTime, err := span.ReadWriteTransaction(ctx, f)
	if err != nil {
		return err
	}

	updates := prepareClusterUpdates(chunk, existingClustering, newClustering)
	if len(updates) > 0 {
		update := &clustering.Update{
			Project: existingState.Project,
			ChunkID: existingState.ChunkID,
			Updates: updates,
		}
		if err := analysis.HandleUpdatedClusters(ctx, update, commitTime); err != nil {
			return err
		}
	}
	return nil
}

func prepareClusterUpdates(chunk *cpb.Chunk, previousClustering clustering.ClusterResults, newClustering clustering.ClusterResults) []*clustering.FailureUpdate {
	var updates []*clustering.FailureUpdate
	for i, testResult := range chunk.Failures {
		previousClusters := previousClustering.Clusters[i]
		newClusters := newClustering.Clusters[i]

		if !clustering.ClusterSetsEqual(previousClusters, newClusters) {
			update := &clustering.FailureUpdate{
				TestResult:       testResult,
				PreviousClusters: previousClusters,
				NewClusters:      newClusters,
			}
			updates = append(updates, update)
		}
	}
	return updates
}
