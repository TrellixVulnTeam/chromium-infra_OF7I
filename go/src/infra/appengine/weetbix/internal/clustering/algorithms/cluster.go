// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package algorithms

import (
	"encoding/hex"
	"errors"

	"infra/appengine/weetbix/internal/clustering"
	"infra/appengine/weetbix/internal/clustering/algorithms/failurereason"
	"infra/appengine/weetbix/internal/clustering/algorithms/rulesalgorithm"
	"infra/appengine/weetbix/internal/clustering/algorithms/testname"
	"infra/appengine/weetbix/internal/clustering/rules"
	"infra/appengine/weetbix/internal/clustering/rules/cache"
)

// Algorithm represents the interface that each clustering algorithm
// generating suggested clusters must implement.
type Algorithm interface {
	// Name returns the identifier of the clustering algorithm.
	Name() string
	// Cluster clusters the given test failure and returns its cluster ID (if
	// it can be clustered) or nil otherwise. THe returned cluster ID must be
	// at most 16 bytes.
	Cluster(failure *clustering.Failure) []byte
	// FailureAssociationRule returns a failure association rule that
	// captures the definition of the cluster containing the given example.
	FailureAssociationRule(example *clustering.Failure) string
	// ClusterDescription returns a description of the cluster, for use when
	// filing bugs, with the help of the given example failure.
	ClusterDescription(example *clustering.Failure) *clustering.ClusterDescription
}

// AlgorithmsVersion is the version of the set of algorithms used.
// Changing the set of algorithms below (including add, update or
// deletion of an algorithm) should result in this version being
// incremented.
//
// In case of algorithm deletion, make sure to update this constant
// appropriately to ensure the AlgorithmsVersion still increases
// (I.E. DO NOT simply delete "+ <myalgorithm>.AlgorithmVersion"
// when deleting an algorithm without rolling its value (plus one)
// into the constant.)
const AlgorithmsVersion = 1 + failurereason.AlgorithmVersion +
	testname.AlgorithmVersion + rulesalgorithm.AlgorithmVersion

// suggestingAlgorithms is the set of clustering algorithms used by
// Weetbix to generate suggested clusters.
// When an algorithm is added or removed from the set,
// or when an algorithm is updated, ensure the AlgorithmsVersion
// above increments.
var suggestingAlgorithms = []Algorithm{
	&failurereason.Algorithm{},
	&testname.Algorithm{},
}

// rulesAlgorithm is the rules-based clustering algorithm used by
// Weetbix. When this algorithm is changed, ensure the AlgorithmsVersion
// above increments.
var rulesAlgorithm = rulesalgorithm.Algorithm{}

// The set of all algorithms known by Weetbix.
var algorithmNames map[string]struct{}

func init() {
	algorithmNames = make(map[string]struct{})
	algorithmNames[rulesalgorithm.AlgorithmName] = struct{}{}
	for _, a := range suggestingAlgorithms {
		algorithmNames[a.Name()] = struct{}{}
	}
}

// Cluster performs (incremental re-)clustering of the given test
// failures using all registered clustering algorithms and the
// specified set of failure association rules.
//
// If the test results have not been previously clustered, pass
// an existing ClusterResults of NewEmptyClusterResults(...)
// to cluster test results from scratch.
//
// If the test results have been previously clustered, pass the
// ClusterResults returned by the last call to Cluster.
func Cluster(ruleset *cache.Ruleset, existing clustering.ClusterResults, failures []*clustering.Failure) clustering.ClusterResults {
	if existing.AlgorithmsVersion > AlgorithmsVersion {
		// We are running out-of-date clustering algorithms. Do not
		// try to improve on the existing clustering. This can
		// happen if we are rolling out a new version of Weetbix.
		return existing
	}

	// For each suggesting algorithm, figure out whether it has already been
	// run previously and we can retain its results (for efficiency), or
	// if we need to run it again.
	var suggestedAlgorithmsToRun []Algorithm
	suggestedAlgorithmsToRetain := make(map[string]struct{})
	for _, alg := range suggestingAlgorithms {
		if _, ok := existing.Algorithms[alg.Name()]; ok {
			// The algorithm was run previously. Retain its results.
			suggestedAlgorithmsToRetain[alg.Name()] = struct{}{}
		} else {
			// The algorithm was not previously run. Run it now.
			suggestedAlgorithmsToRun = append(suggestedAlgorithmsToRun, alg)
		}
	}

	// For rule-based clustering.
	_, reuseRuleAlgorithmResults := existing.Algorithms[rulesalgorithm.AlgorithmName]
	existingRulesVersion := existing.RulesVersion
	if !reuseRuleAlgorithmResults {
		// Although we may have previously run rule-based clustering, we did
		// not run the current version of that algorithm. Invalidate all
		// previous analysis; match against all rules again.
		existingRulesVersion = rules.StartingEpoch
	}

	var result [][]*clustering.ClusterID
	for i, f := range failures {
		var newIDs []*clustering.ClusterID
		existingRuleIDs := make(map[string]struct{})

		existingIDs := existing.Clusters[i]
		for _, id := range existingIDs {
			if _, ok := suggestedAlgorithmsToRetain[id.Algorithm]; ok {
				// The algorithm was run previously. Retain its results.
				newIDs = append(newIDs, id)
			}
			if reuseRuleAlgorithmResults && id.Algorithm == rulesalgorithm.AlgorithmName {
				// The rules algorithm was previously run. Record the past results,
				// but separately. Some previously matched rules may have been
				// updated or made inactive since, so we need to treat these
				// separately (and pass them to the rules algorithm to filter
				// through).
				existingRuleIDs[id.ID] = struct{}{}
			}
		}

		// Run the suggested clustering algorithms.
		for _, a := range suggestedAlgorithmsToRun {
			id := a.Cluster(f)
			if id == nil {
				continue
			}
			newIDs = append(newIDs, &clustering.ClusterID{
				Algorithm: a.Name(),
				ID:        hex.EncodeToString(id),
			})
		}

		var newRuleIDs map[string]struct{}
		if ruleset.RulesVersion.After(existingRulesVersion) {
			// Match against the (incremental) set of rules.
			newRuleIDs = rulesAlgorithm.Cluster(ruleset, existingRulesVersion, existingRuleIDs, f)
		} else {
			// Test results were already clustered with an equal or later
			// version of rules.
			// This can happen if our cached ruleset is out of date. Re-use the
			// existing analysis; don't try to improve on it.
			newRuleIDs = existingRuleIDs
		}

		for rID := range newRuleIDs {
			id := &clustering.ClusterID{
				Algorithm: rulesalgorithm.AlgorithmName,
				ID:        rID,
			}
			newIDs = append(newIDs, id)
		}

		// Keep the output deterministic by sorting the clusters in the
		// output.
		clustering.SortClusters(newIDs)
		result = append(result, newIDs)
	}

	newRulesVersion := ruleset.RulesVersion
	if existingRulesVersion.After(newRulesVersion) {
		// If the existing rule-matching is more current than our current
		// ruleset allows, we will have kept its results, and should keep
		// its RulesVersion.
		// This can happen sometimes if our cached ruleset is out of date.
		// This is normal.
		newRulesVersion = existingRulesVersion
	}

	return clustering.ClusterResults{
		AlgorithmsVersion: AlgorithmsVersion,
		RulesVersion:      newRulesVersion,
		Algorithms:        algorithmNames,
		Clusters:          result,
	}
}

// ErrAlgorithmNotExist is returned if an algorithm with the given
// name does not exist. This may indicate the algorithm
// is newer or older than the current version.
var ErrAlgorithmNotExist = errors.New("algorithm does not exist")

// SuggestingAlgorithm returns the algorithm for generating
// suggested clusters with the given name. If the algorithm does
// not exist, ErrAlgorithmNotExist is returned.
func SuggestingAlgorithm(algorithm string) (Algorithm, error) {
	for _, a := range suggestingAlgorithms {
		if a.Name() == algorithm {
			return a, nil
		}
	}
	// We may be running old code, or the caller may be asking
	// for an old (version of an) algorithm.
	return nil, ErrAlgorithmNotExist
}

// NewEmptyClusterResults returns a new ClusterResults for a list of
// test results of length count. The ClusterResults will indicate the
// test results have not been clustered.
func NewEmptyClusterResults(count int) clustering.ClusterResults {
	return clustering.ClusterResults{
		// Algorithms version 0 is the empty set of clustering algorithms.
		AlgorithmsVersion: 0,
		// The RulesVersion StartingEpoch refers to the empty set of rules.
		RulesVersion: rules.StartingEpoch,
		Algorithms:   make(map[string]struct{}),
		Clusters:     make([][]*clustering.ClusterID, count),
	}
}
