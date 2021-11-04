// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package clustering

// ClusterDescription captures the description of a cluster, for
// use in bug filing.
type ClusterDescription struct {
	// Title is a short, one-line description of the cluster, for use
	// in the bug title.
	Title string
	// Description is a human-readable description of the cluster.
	Description string
}
