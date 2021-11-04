// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package bugs

import (
	"infra/appengine/weetbix/internal/config"
	"math"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"google.golang.org/protobuf/proto"
)

func TestThresholding(t *testing.T) {
	t.Parallel()

	Convey("With Cluster", t, func() {
		cl := &ClusterImpact{
			Failures1d: 100,
			Failures3d: 300,
			Failures7d: 700,
		}
		t := &config.ImpactThreshold{}
		Convey("No cluster meets empty threshold", func() {
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
		Convey("Threshold with deflation", func() {
			// With 15% hysteresis, leads to effective threshold of
			// 100 / 1.15 = 86.
			t.UnexpectedFailures_1D = proto.Int64(100)
			cl.Failures1d = 86
			So(cl.MeetsInflatedThreshold(t, -15), ShouldBeTrue)
			cl.Failures1d = 85
			So(cl.MeetsInflatedThreshold(t, -15), ShouldBeFalse)
		})
		Convey("Threshold with inflation", func() {
			// With 15% hysteresis, leads to effective threshold of
			// 100 * 1.15 = 115.
			t.UnexpectedFailures_1D = proto.Int64(100)
			cl.Failures1d = 115
			So(cl.MeetsInflatedThreshold(t, 15), ShouldBeTrue)
			cl.Failures1d = 114
			So(cl.MeetsInflatedThreshold(t, 15), ShouldBeFalse)
		})
		Convey("Thresholding of values near overflow", func() {
			t.UnexpectedFailures_1D = proto.Int64(math.MaxInt64)
			cl.Failures1d = math.MaxInt64
			So(cl.MeetsThreshold(t), ShouldBeTrue)
			// Thresholding has loss of precision towards the max value of
			// an int64, so make sure observed number of failures noticibily less.
			cl.Failures1d = math.MaxInt64 - 10000
			So(cl.MeetsThreshold(t), ShouldBeFalse)
		})
		Convey("Thresholding with inflation near overflow", func() {
			t.UnexpectedFailures_1D = proto.Int64(math.MaxInt64)
			cl.Failures1d = math.MaxInt64 / 11
			So(cl.MeetsInflatedThreshold(t, -1000), ShouldBeTrue)
			// Thresholding has loss of precision towards the max value of
			// an int64, so make sure observed number of failures noticibily less.
			cl.Failures1d = math.MaxInt64/11 - 10000
			So(cl.MeetsInflatedThreshold(t, -1000), ShouldBeFalse)
		})
		Convey("Thresholding with deflation near overflow", func() {
			t.UnexpectedFailures_1D = proto.Int64(math.MaxInt64 / 11)
			cl.Failures1d = math.MaxInt64
			So(cl.MeetsInflatedThreshold(t, 1000), ShouldBeTrue)
			// Thresholding has loss of precision towards the max value of
			// an int64, so make sure observed number of failures noticibily less.
			cl.Failures1d = math.MaxInt64 - 10000
			So(cl.MeetsInflatedThreshold(t, 1000), ShouldBeFalse)
		})
	})
}
