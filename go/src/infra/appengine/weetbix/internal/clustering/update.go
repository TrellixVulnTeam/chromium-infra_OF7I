// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package clustering

import (
	cpb "infra/appengine/weetbix/internal/clustering/proto"
)

// Update describes changes made to the clustering of a chunk.
type Update struct {
	// Project is the LUCI Project containing the chunk which is being
	// (re-)clustered.
	Project string
	// ChunkID is the identity of the chunk which is being (re-)clustered.
	ChunkID string
	// Updates describes how each failure in the cluster was (re)clustered.
	// It contains one entry for each failure in the cluster.
	Updates []*FailureUpdate
}

// FailureUpdate describes the changes made to the clustering
// of a specific test failure.
type FailureUpdate struct {
	// TestResult is the failure that was re-clustered.
	TestResult *cpb.Failure
	// PreviousClusters are the clusters the failure was previously in.
	PreviousClusters []*ClusterID
	// PreviousClusters are the clusters the failure is now in.
	NewClusters []*ClusterID
}
