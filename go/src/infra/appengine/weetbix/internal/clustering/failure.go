// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package clustering

import (
	cpb "infra/appengine/weetbix/internal/clustering/proto"
	pb "infra/appengine/weetbix/proto/v1"

	"google.golang.org/protobuf/proto"
)

// Failure captures the minimal information required to cluster a failure.
// This is a subset of the information captured by Weetbix for failures.
type Failure struct {
	// The name of the test that failed.
	TestID string
	// The failure reason explaining the reason why the test failed.
	Reason *pb.FailureReason
}

// FailureFromProto extracts failure information relevant for clustering from
// a Weetbix failure proto.
func FailureFromProto(f *cpb.Failure) *Failure {
	result := &Failure{
		TestID: f.TestId,
	}
	if f.FailureReason != nil {
		result.Reason = proto.Clone(f.FailureReason).(*pb.FailureReason)
	}
	return result
}

// FailuresFromProtos extracts failure information relevant for clustering
// from a set of Weetbix failure protos.
func FailuresFromProtos(protos []*cpb.Failure) []*Failure {
	result := make([]*Failure, len(protos))
	for i, p := range protos {
		result[i] = FailureFromProto(p)
	}
	return result
}
