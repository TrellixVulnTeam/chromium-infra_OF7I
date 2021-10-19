// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package failurereason

import (
	"testing"

	cpb "infra/appengine/weetbix/internal/clustering/proto"
	pb "infra/appengine/weetbix/proto/v1"

	. "github.com/smartystreets/goconvey/convey"
)

func TestAlgorithm(t *testing.T) {
	Convey(`Does not cluster test result without failure reason`, t, func() {
		a := &Algorithm{}
		id := a.Cluster(&cpb.Failure{})
		So(id, ShouldBeNil)
	})
	Convey(`ID of appropriate length`, t, func() {
		a := &Algorithm{}
		id := a.Cluster(&cpb.Failure{
			FailureReason: &pb.FailureReason{
				PrimaryErrorMessage: "abcd this is a test failure message",
			},
		})
		// IDs may be 16 bytes at most.
		So(len(id), ShouldBeGreaterThan, 0)
		So(len(id), ShouldBeLessThanOrEqualTo, 16)
	})
	Convey(`Same ID for same cluster with different numbers`, t, func() {
		a := &Algorithm{}
		id1 := a.Cluster(&cpb.Failure{
			FailureReason: &pb.FailureReason{
				PrimaryErrorMessage: "Null pointer exception at ip 0x45637271",
			},
		})
		id2 := a.Cluster(&cpb.Failure{
			FailureReason: &pb.FailureReason{
				PrimaryErrorMessage: "Null pointer exception at ip 0x12345678",
			},
		})
		So(id2, ShouldResemble, id1)
	})
	Convey(`Different ID for different clusters`, t, func() {
		a := &Algorithm{}
		id1 := a.Cluster(&cpb.Failure{
			FailureReason: &pb.FailureReason{
				PrimaryErrorMessage: "Exception in TestMethod",
			},
		})
		id2 := a.Cluster(&cpb.Failure{
			FailureReason: &pb.FailureReason{
				PrimaryErrorMessage: "Exception in MethodUnderTest",
			},
		})
		So(id2, ShouldNotResemble, id1)
	})
}
