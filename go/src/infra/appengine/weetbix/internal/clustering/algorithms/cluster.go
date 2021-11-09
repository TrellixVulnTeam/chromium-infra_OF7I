// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package algorithms

import (
	"encoding/hex"
	"errors"

	"infra/appengine/weetbix/internal/clustering"
	"infra/appengine/weetbix/internal/clustering/algorithms/failurereason"
	"infra/appengine/weetbix/internal/clustering/algorithms/testname"
	"infra/appengine/weetbix/internal/clustering/rules"
)

// Algorithm represents the interface that each clustering algorithm must
// implement.
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
const AlgorithmsVersion = 1 + failurereason.AlgorithmVersion + testname.AlgorithmVersion

// algorithms is the set of clustering algorithms known to Weetbix.
// When this an algorithm is added or removed from the set,
// or when an algorithm is updated, ensure the AlgorithmsVersion
// above increments.
var algorithms = []Algorithm{
	&failurereason.Algorithm{},
	&testname.Algorithm{},
}

// Cluster clusters the given test failures using all registered
// clustering algorithms.
func Cluster(failures []*clustering.Failure) clustering.ClusterResults {
	var result [][]*clustering.ClusterID
	for _, f := range failures {
		var ids []*clustering.ClusterID
		for _, a := range algorithms {
			id := a.Cluster(f)
			if id == nil {
				continue
			}
			ids = append(ids, &clustering.ClusterID{
				Algorithm: a.Name(),
				ID:        hex.EncodeToString(id),
			})
		}

		result = append(result, ids)
	}

	algorithmNames := make(map[string]struct{})
	for _, a := range algorithms {
		algorithmNames[a.Name()] = struct{}{}
	}

	return clustering.ClusterResults{
		AlgorithmsVersion: AlgorithmsVersion,
		// TODO(crbug.com/1243174): Update when failure association rules
		// are implemented.
		RulesVersion: rules.StartingEpoch,
		Algorithms:   algorithmNames,
		Clusters:     result,
	}
}

// ErrAlgorithmNotExist is returned if an algorithm with the given
// name does not exist. This may indicate the algorithm
// is newer or older than the current version.
var ErrAlgorithmNotExist = errors.New("algorithm does not exist")

// ByName returns the algorithm with the given name. If the algorithm
// does not exist, ErrAlgorithmNotExist is returned.
func ByName(algorithm string) (Algorithm, error) {
	for _, a := range algorithms {
		if a.Name() == algorithm {
			return a, nil
		}
	}
	// We may be running old code, or the caller may be asking
	// for an old (version of an) algorithm.
	return nil, ErrAlgorithmNotExist
}
