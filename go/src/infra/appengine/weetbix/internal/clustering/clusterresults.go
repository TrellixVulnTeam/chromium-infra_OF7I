// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package clustering

import (
	"time"
)

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

// AlgorithmsAndClustersEqual returns whether the algorithms and clusters of
// two cluster results are equivalent.
func AlgorithmsAndClustersEqual(a *ClusterResults, b *ClusterResults) bool {
	if !setsEqual(a.Algorithms, b.Algorithms) {
		return false
	}
	if len(a.Clusters) != len(b.Clusters) {
		return false
	}
	for i, aClusters := range a.Clusters {
		bClusters := b.Clusters[i]
		if !ClusterSetsEqual(aClusters, bClusters) {
			return false
		}
	}
	return true
}

// ClusterSetsEqual returns whether the the set of clusters represented
// by the slice `as` is equivalent to the set of clusters represented
// by the slice `bs`.
// Order of the slices is not considered. Each cluster in as and bs
// should be unique.
func ClusterSetsEqual(as []*ClusterID, bs []*ClusterID) bool {
	if len(as) != len(bs) {
		return false
	}
	aKeys := make(map[string]struct{})
	for _, a := range as {
		aKeys[a.Key()] = struct{}{}
	}
	bKeys := make(map[string]struct{})
	for _, b := range bs {
		bKeys[b.Key()] = struct{}{}
	}
	return setsEqual(aKeys, bKeys)
}

// setsEqual returns whether two sets are equal.
func setsEqual(a map[string]struct{}, b map[string]struct{}) bool {
	if len(a) != len(b) {
		return false
	}
	for key := range a {
		if _, ok := b[key]; !ok {
			return false
		}
	}
	return true
}
