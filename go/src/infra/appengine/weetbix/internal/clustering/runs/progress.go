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

// ReclusteringTarget captures the rules and algorithms a re-clustering run
// is re-clustering to.
type ReclusteringTarget struct {
	// RulesVersion is the rules version the re-clustering run is attempting
	// to achieve.
	RulesVersion time.Time `json:"rulesVersion"`
	// AlgorithmsVersion is the algorithms version the re-clustering run is
	// attempting to achieve.
	AlgorithmsVersion int64 `json:"algorithmsVersion"`
}

// ReclusteringProgress captures the progress re-clustering a
// given LUCI project's test results using specific rules
// versions or algorithms versions.
type ReclusteringProgress struct {
	// ProgressPerMille is the progress of the current re-clustering run,
	// measured in thousandths (per mille).
	ProgressPerMille int `json:"progressPerMille"`
	// LatestAlgorithmsVersion is the latest version of algorithms known to
	// Weetbix.
	LatestAlgorithmsVersion int64 `json:"latestAlgorithmsVersion"`
	// Next is the goal of the current re-clustering run. (For which
	// ProgressPerMille is specified.)
	Next ReclusteringTarget `json:"next"`
	// Last is the goal of the last completed re-clustering run.
	Last ReclusteringTarget `json:"last"`
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

	// Scale run progress to being from 0 to 1000.
	runProgress := int(lastWithProgress.Progress / lastWithProgress.ShardCount)

	latestAlgorithmsVersion := int64(algorithms.AlgorithmsVersion)
	if last.AlgorithmsVersion > latestAlgorithmsVersion {
		// This GAE instance is running old code, use the
		// algorithms version on the last re-clustering run instead.
		// Note that the AlgorithmsVersion in each subsequent
		// re-clustering run is guaranteed to be non-decreasing, so
		// the AlgorithmsVersion in the latest run is guaranteed
		// to be the highest of all runs so far.
		latestAlgorithmsVersion = last.AlgorithmsVersion
	}

	return &ReclusteringProgress{
		ProgressPerMille:        runProgress,
		LatestAlgorithmsVersion: latestAlgorithmsVersion,
		Next: ReclusteringTarget{
			RulesVersion:      lastWithProgress.RulesVersion,
			AlgorithmsVersion: lastWithProgress.AlgorithmsVersion,
		},
		Last: ReclusteringTarget{
			RulesVersion:      lastCompleted.RulesVersion,
			AlgorithmsVersion: lastCompleted.AlgorithmsVersion,
		},
	}, nil
}

// IncorporatesLatestAlgorithms returns whether only the latest
// algorithms are in Weetbix's clustering output.
func (p *ReclusteringProgress) IncorporatesLatestAlgorithms() bool {
	return p.Last.AlgorithmsVersion >= p.LatestAlgorithmsVersion
}

// IncorporatesRulesVersion returns returns whether only rules
// of at least the given rules version are used in Weetbix's clustering
// output.
func (p *ReclusteringProgress) IncorporatesRulesVersion(rulesVersion time.Time) bool {
	return !rulesVersion.After(p.Last.RulesVersion)
}
