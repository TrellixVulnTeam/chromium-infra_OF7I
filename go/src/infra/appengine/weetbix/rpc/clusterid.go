// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package rpc

import (
	"infra/appengine/weetbix/internal/clustering"
	pb "infra/appengine/weetbix/proto/v1"
)

func createClusterIdPB(b clustering.ClusterID) *pb.ClusterId {
	if b.IsBugCluster() {
		// Drop the version number from the rules algorithm,
		// e.g. "rules-v2" -> "rules".
		// Clients may want to detect the rules algorithm
		// to identify clusters for which they can lookup the
		// corresponding rule.
		// Hiding the version information avoids clients
		// accidentally depending on it in their code, which would
		// make changing the version of the rules-based clustering
		// algorithm breaking for clients.
		// It is anticipated that future updates to the rules-based
		// clustering algorithm will mostly be about tweaking the
		// failure-matching semantics, and will retain the property
		// that the Cluster ID corresponds to the Rule ID.
		return &pb.ClusterId{
			Algorithm: "rules",
			Id:        b.ID,
		}
	}
	// Suggested clustering algorithm.
	return &pb.ClusterId{
		Algorithm: b.Algorithm,
		Id:        b.ID,
	}
}
