// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package rpc

import (
	"strings"

	"infra/appengine/weetbix/internal/clustering"
	"infra/appengine/weetbix/internal/clustering/algorithms/rulesalgorithm"
	pb "infra/appengine/weetbix/proto/v1"
)

func createClusterIdPB(b clustering.ClusterID) *pb.ClusterId {
	return &pb.ClusterId{
		Algorithm: aliasAlgorithm(b.Algorithm),
		Id:        b.ID,
	}
}

func aliasAlgorithm(algorithm string) string {
	// Drop the version number from the rules algorithm,
	// e.g. "rules-v2" -> "rules".
	// Clients may want to identify if the rules algorithm
	// was used to cluster, to identify clusters for which they
	// can lookup the corresponding rule.
	// Hiding the version information avoids clients
	// accidentally depending on it in their code, which would
	// make changing the version of the rules-based clustering
	// algorithm breaking for clients.
	// It is anticipated that future updates to the rules-based
	// clustering algorithm will mostly be about tweaking the
	// failure-matching semantics, and will retain the property
	// that the Cluster ID corresponds to the Rule ID.
	if strings.HasPrefix(algorithm, clustering.RulesAlgorithmPrefix) {
		return "rules"
	}
	return algorithm
}

func resolveAlgorithm(algorithm string) string {
	// Resolve an alias to the rules algorithm to the concrete
	// implementation, e.g. "rules" -> "rules-v2".
	if algorithm == "rules" {
		return rulesalgorithm.AlgorithmName
	}
	return algorithm
}
