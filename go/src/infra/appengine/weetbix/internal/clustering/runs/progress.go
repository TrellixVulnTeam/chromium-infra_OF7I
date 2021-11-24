// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package runs

import (
	"context"
	"infra/appengine/weetbix/internal/clustering/algorithms"
	"time"

	"go.chromium.org/luci/server/span"
)

// ProgressComplete is the maximum progress value returned by
// ProgressToRulesVersion and ProgressToLatestAlgorithmsVersion.
const ProgressComplete = 1000

// ReclusteringProgress captures the progress re-clustering a
// given LUCI project's test results using specific rules
// versions or algorithms versions.
type ReclusteringProgress struct {
	lastCompleted    *ReclusteringRun
	lastWithProgress *ReclusteringRun
	last             *ReclusteringRun
}

// ReadReclusteringProgress reads the re-clustering progress for
// the given LUCI project.
func ReadReclusteringProgress(ctx context.Context, project string) (*ReclusteringProgress, error) {
	txn, cancel := span.ReadOnlyTransaction(ctx)
	defer cancel()

	lastCompleted, err := ReadLastComplete(txn, project)
	if err != nil {
		return nil, err
	}

	lastWithProgress, err := ReadLastWithProgress(txn, project)
	if err != nil {
		return nil, err
	}

	last, err := ReadLast(txn, project)
	if err != nil {
		return nil, err
	}

	return &ReclusteringProgress{
		lastCompleted:    lastCompleted,
		lastWithProgress: lastWithProgress,
		last:             last,
	}, nil
}

// LatestAlgorithmsVersion returns the latest algorithms version
// in-use for the given project.
func (p *ReclusteringProgress) LatestAlgorithmsVersion() int64 {
	algorithmsVersion := int64(algorithms.AlgorithmsVersion)
	if p.last.AlgorithmsVersion > algorithmsVersion {
		// This GAE instance is running old code, use the
		// algorithms version on the last re-clustering run instead.
		// Note that the AlgorithmsVersion in each subsequent
		// re-clustering run is guaranteed to be non-decreasing, so
		// the AlgorithmsVersion in the latest run is guaranteed
		// to be the highest of all runs so far.
		algorithmsVersion = p.last.AlgorithmsVersion
	}
	return algorithmsVersion
}

// ProgressToRulesVersion returns progress towards using only the latest
// algorithms in Weetbix's clustering output. The returned value is
// between 0 (0.0%) and 1000 (100.0%).
func (p *ReclusteringProgress) ProgressToLatestAlgorithmsVersion() int {
	algorithmsVersion := p.LatestAlgorithmsVersion()
	return p.progressTo(func(r *ReclusteringRun) bool {
		return r.AlgorithmsVersion >= algorithmsVersion
	})
}

// ProgressToRulesVersion returns progress towards using only rules
// of at least the given rules version in Weetbix's clustering output.
// The returned value is between 0 (0.0%) and 1000 (100.0%).
func (p *ReclusteringProgress) ProgressToRulesVersion(rulesVersion time.Time) int {
	return p.progressTo(func(r *ReclusteringRun) bool {
		// run RulesVersion is after, or equal to, rulesVersion.
		return !rulesVersion.After(r.RulesVersion)
	})
}

// progressTo returns progress towards completing a re-clustering run
// satisfying the given predicate.
func (p *ReclusteringProgress) progressTo(predicate func(r *ReclusteringRun) bool) int {
	if predicate(p.lastCompleted) {
		// Completed.
		return 1000
	}
	if predicate(p.lastWithProgress) {
		// Scale run progress to being from 0 to 1000.
		runProgress := p.lastWithProgress.Progress / p.lastWithProgress.ShardCount

		// Starting a re-clustering run that will complete re-clustering
		// to the satisfaction of the specified predicate is 30% of the
		// work done. Finishing it is the other 70%.
		return int(300 + ((runProgress * 7) / 10))
	}
	// Scale run progress to being from 0 to 1000.
	runProgress := p.lastWithProgress.Progress / p.lastWithProgress.ShardCount

	// Finishing the current re-clustering goal is 30% of the
	// work, as it will enable the next reclustering runs to set
	// a goal that satisfies the specified predicate.
	return int(((runProgress * 3) / 10))
}
