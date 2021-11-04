// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package testname

import (
	"testing"

	"infra/appengine/weetbix/internal/clustering"
	pb "infra/appengine/weetbix/proto/v1"

	. "github.com/smartystreets/goconvey/convey"
)

func TestAlgorithm(t *testing.T) {
	Convey(`ID of appropriate length`, t, func() {
		a := &Algorithm{}
		id := a.Cluster(&clustering.Failure{
			TestID: "ninja://test_name",
		})
		// IDs may be 16 bytes at most.
		So(len(id), ShouldBeGreaterThan, 0)
		So(len(id), ShouldBeLessThanOrEqualTo, clustering.MaxClusterIDBytes)
	})
	Convey(`Same ID for same test name`, t, func() {
		a := &Algorithm{}
		id1 := a.Cluster(&clustering.Failure{
			TestID: "ninja://test_name_one/",
			Reason: &pb.FailureReason{PrimaryErrorMessage: "A"},
		})
		id2 := a.Cluster(&clustering.Failure{
			TestID: "ninja://test_name_one/",
			Reason: &pb.FailureReason{PrimaryErrorMessage: "B"},
		})
		So(id2, ShouldResemble, id1)
	})
	Convey(`Different ID for different clusters`, t, func() {
		a := &Algorithm{}
		id1 := a.Cluster(&clustering.Failure{
			TestID: "ninja://test_name_one/",
		})
		id2 := a.Cluster(&clustering.Failure{
			TestID: "ninja://test_name_two/",
		})
		So(id2, ShouldNotResemble, id1)
	})
}
