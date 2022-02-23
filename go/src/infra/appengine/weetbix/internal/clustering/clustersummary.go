// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package clustering

// ClusterSummary captures information about a cluster.
// This is a subset of the information captured by Weetbix for failures.
type ClusterSummary struct {
	// Example is an example failure contained within the cluster.
	Example Failure

	// TopTests is a list of up to 5 most commonly occurring tests
	// included in the cluster.
	TopTests []string
}
