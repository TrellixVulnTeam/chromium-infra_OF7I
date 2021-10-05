// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package clustering

import (
	"infra/appengine/weetbix/internal/config"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"google.golang.org/protobuf/proto"
)

func TestThresholding(t *testing.T) {
	t.Parallel()

	Convey("With Cluster", t, func() {
		cl := &Cluster{
			UnexpectedFailures1d: 100,
			UnexpectedFailures3d: 300,
			UnexpectedFailures7d: 700,
		}
		t := &config.ImpactThreshold{}
		Convey("no cluster meets empty threshold", func() {
			So(cl.MeetsThreshold(t), ShouldBeFalse)
		})
		Convey("1d unexpected failures threshold", func() {
			t.UnexpectedFailures_1D = proto.Int64(100)
			So(cl.MeetsThreshold(t), ShouldBeTrue)

			t.UnexpectedFailures_1D = proto.Int64(101)
			So(cl.MeetsThreshold(t), ShouldBeFalse)
		})
		Convey("3d unexpected failures threshold", func() {
			t.UnexpectedFailures_3D = proto.Int64(300)
			So(cl.MeetsThreshold(t), ShouldBeTrue)

			t.UnexpectedFailures_3D = proto.Int64(301)
			So(cl.MeetsThreshold(t), ShouldBeFalse)
		})
		Convey("7d unexpected failures threshold", func() {
			t.UnexpectedFailures_7D = proto.Int64(700)
			So(cl.MeetsThreshold(t), ShouldBeTrue)

			t.UnexpectedFailures_7D = proto.Int64(701)
			So(cl.MeetsThreshold(t), ShouldBeFalse)
		})
	})
}
