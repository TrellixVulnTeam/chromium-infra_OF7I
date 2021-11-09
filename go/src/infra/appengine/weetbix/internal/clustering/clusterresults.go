// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package clustering

import "time"

// ClusterResults represents the results of clustering a list of
// test failures.
type ClusterResults struct {
	// AlgorithmsVersion is the version of clustering algorithms used to
	// cluster test results in this chunk. (This is a version over the
	// set of algorithms, distinct from the versions of a single algorithm,
	// e.g.: v1 -> {failurereason-v1}, v2 -> {failurereason-v1, testname-v1},
	// v3 -> {failurereason-v2, testname-v1}.)
	AlgorithmsVersion int64
	// RulesVersion is the version of failure association rules used
	// to cluster test results.  This is the RulesLastUpdated
	// time of the most-recently-updated failure association rule in
	// the snapshot of failure association rules used to cluster
	// the test results.
	RulesVersion time.Time
	// Algorithms is the set of algorithms that were used to cluster
	// the test results. Each entry is an algorithm name.
	// When stored alongside the clustered test results, this allows only
	// the new algorithms to be run when re-clustering (for efficiency).
	Algorithms map[string]struct{}
	// Clusters records the clusters each test result is in;
	// one slice of ClusterIDs for each test result.
	Clusters [][]*ClusterID
}
