// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package failurereason contains the failure reason clustering algorithm
// for Weetbix.
//
// This algorithm removes ips, temp file names, numbers and other such tokens
// to cluster similar reasons together.
package failurereason

import (
	"crypto/sha256"
	"fmt"
	"regexp"

	cpb "infra/appengine/weetbix/internal/clustering/proto"
)

// AlgorithmVersion is the version of the clustering algorithm. The algorithm
// version should be incremented whenever existing test results may be
// clustered differently (i.e. Cluster(f) returns a different value for some
// f that may have been already ingested).
const AlgorithmVersion = 1

// AlgorithmName is the identifier for the clustering algorithm.
// Weetbix requires all clustering algorithms to have a unique identifier.
// Must match the pattern ^[a-z0-9-.]{1,32}$.
//
// The AlgorithmName must encode the algorithm version, so that each version
// of an algorithm has a different name.
var AlgorithmName = fmt.Sprintf("failurereason-v%v", AlgorithmVersion)

// To match any 1 or more digit numbers, or hex values (often appear in temp
// file names or prints of pointers), which will be replaced.
var clusterExp = regexp.MustCompile(`[0-9]+|[\-0-9a-fA-F\s]{16,}|[0-9a-fA-Fx]{8,}|[/+0-9a-zA-Z]{10,}=+`)

// Algorithm represents an instance of the reason-based clustering
// algorithm.
type Algorithm struct{}

// Name returns the identifier of the clustering algorithm.
func (a *Algorithm) Name() string {
	return AlgorithmName
}

// Cluster clusters the given test failure and returns its cluster ID (if it
// can be clustered) or nil otherwise.
func (a *Algorithm) Cluster(failure *cpb.Failure) []byte {
	if failure.FailureReason == nil || failure.FailureReason.PrimaryErrorMessage == "" {
		return nil
	}
	// Replace numbers and hex values.
	id := clusterExp.ReplaceAllString(failure.FailureReason.PrimaryErrorMessage, "0")
	// sha256 hash the resulting string.
	h := sha256.Sum256([]byte(id))
	// Take first 16 bytes as the ID. (Risk of collision is
	// so low as to not warrant full 32 bytes.)
	return h[0:16]
}
