// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package reclustering

import (
	"context"
	"errors"
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
// If the chunk does not not exist in Spanner, pass a *state.Entry
// with project, chunkID, objectID and partitionTime set
// but with LastUpdated set to its zero value. The chunk will be
// clustered for the first time.
//
// If the chunk does exist in Spanner, pass the state.Entry read
// from Spanner, along with the test results. The chunk will be
// re-clustered.
func Update(ctx context.Context, analysis Analysis, chunk *cpb.Chunk, existingState *state.Entry) error {
	// Obtain the set of failure association rules to use when clustering.
	ruleset, err := rulesCache.Ruleset(ctx, existingState.Project)
	if err != nil {
		return err
	}

	exists := !existingState.LastUpdated.IsZero()
	var existingClustering clustering.ClusterResults
	if !exists {
		existingClustering = algorithms.NewEmptyClusterResults(len(chunk.Failures))
	} else {
		// TODO(crbug.com/1243174) Implement reclustering.
		return errors.New("reclustering is not yet supported")
	}

	newClustering := algorithms.Cluster(ruleset, existingClustering, clustering.FailuresFromProtos(chunk.Failures))

	f := func(ctx context.Context) error {
		clusterState := &state.Entry{
			Project:       existingState.Project,
			ChunkID:       existingState.ChunkID,
			PartitionTime: existingState.PartitionTime,
			ObjectID:      existingState.ObjectID,
			Clustering:    newClustering,
		}
		if err := state.Create(ctx, clusterState); err != nil {
			return err
		}
		return nil
	}
	commitTime, err := span.ReadWriteTransaction(ctx, f)
	if err != nil {
		return err
	}

	update := &clustering.Update{
		Project: existingState.Project,
		ChunkID: existingState.ChunkID,
		Updates: prepareClusterUpdates(chunk, newClustering),
	}
	if err := analysis.HandleUpdatedClusters(ctx, update, commitTime); err != nil {
		return err
	}
	return nil
}

func prepareClusterUpdates(chunk *cpb.Chunk, clusterResults clustering.ClusterResults) []*clustering.FailureUpdate {
	var updates []*clustering.FailureUpdate
	for i, testResult := range chunk.Failures {
		update := &clustering.FailureUpdate{
			TestResult:  testResult,
			NewClusters: clusterResults.Clusters[i],
		}
		updates = append(updates, update)
	}
	return updates
}
