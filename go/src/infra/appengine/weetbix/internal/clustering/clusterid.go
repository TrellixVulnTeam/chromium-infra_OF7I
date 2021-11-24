// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package clustering

import (
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"go.chromium.org/luci/common/errors"
)

// MaxClusterIDBytes is the maximum number of bytes the algorithm-determined
// cluster ID may occupy. This is the raw number of bytes; if the ID is hex-
// encoded (e.g. for use in a BigQuery table), its length in characters may
// be double this number.
const MaxClusterIDBytes = 16

// RulesAlgorithmPrefix is the algorithm name prefix used by all versions
// of the rules-based clustering algorithm.
const RulesAlgorithmPrefix = "rules-"

// ClusterID represents the identity of a cluster. The LUCI Project is
// omitted as it is assumed to be implicit from the context.
type ClusterID struct {
	// Algorithm is the name of the clustering algorithm that identified
	// the cluster.
	Algorithm string `json:"algorithm"`
	// ID is the cluster identifier returned by the algorithm. The underlying
	// identifier is at most 16 bytes, but is represented here as a hexadecimal
	// string of up to 32 lowercase hexadecimal characters.
	ID string `json:"id"`
}

// Key returns a value that can be used to uniquely identify the Cluster.
// This is designed for cases where it is desirable for cluster IDs
// to be used as keys in a map.
func (c ClusterID) Key() string {
	return fmt.Sprintf("%s:%s", c.Algorithm, c.ID)
}

// Validate validates the cluster ID is valid.
func (c ClusterID) Validate() error {
	if !AlgorithmRe.MatchString(c.Algorithm) {
		return errors.New("algorithm not valid")
	}
	b, err := hex.DecodeString(c.ID)
	if err != nil {
		return errors.New("ID is not valid hexadecimal")
	}
	// ID must be always be stored in lowercase, so that string equality can
	// be used to determine if IDs are the same.
	if c.ID != strings.ToLower(c.ID) {
		return errors.New("ID must be in lowercase")
	}
	if len(b) > MaxClusterIDBytes {
		return fmt.Errorf("ID is too long (got %v bytes, want at most %v bytes)", len(b), MaxClusterIDBytes)
	}
	if len(b) == 0 {
		return errors.New("ID is empty")
	}
	return nil
}

// IsEmpty returns whether the cluster ID is equal to its
// zero value.
func (c ClusterID) IsEmpty() bool {
	return c.Algorithm == "" && c.ID == ""
}

// IsBugCluster returns whether this cluster is backed by a failure
// association rule, and produced by a version of the failure association
// rule based clustering algorithm.
func (c ClusterID) IsBugCluster() bool {
	return strings.HasPrefix(c.Algorithm, RulesAlgorithmPrefix)
}

// SortClusters sorts the given clusters in ascending algorithm and then ID
// order.
func SortClusters(cs []*ClusterID) {
	sort.Slice(cs, func(i, j int) bool {
		if cs[i].Algorithm == cs[j].Algorithm {
			return cs[i].ID < cs[j].ID
		}
		return cs[i].Algorithm < cs[j].Algorithm
	})
}
