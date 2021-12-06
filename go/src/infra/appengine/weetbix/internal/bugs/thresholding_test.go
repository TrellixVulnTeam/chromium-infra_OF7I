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
			TestResultsFailed: MetricImpact{
				OneDay:   100,
				ThreeDay: 300,
				SevenDay: 700,
			},
			TestRunsFailed: MetricImpact{
				OneDay:   30,
				ThreeDay: 90,
				SevenDay: 210,
			},
			PresubmitRunsFailed: MetricImpact{
				OneDay:   3,
				ThreeDay: 9,
				SevenDay: 21,
			},
		}
		t := &config.ImpactThreshold{}
		Convey("No cluster meets empty threshold", func() {
			So(cl.MeetsThreshold(t), ShouldBeFalse)
		})
		Convey("test results failed thresholding", func() {
			t.TestResultsFailed = &config.MetricThreshold{OneDay: proto.Int64(100)}
			So(cl.MeetsThreshold(t), ShouldBeTrue)

			t.TestResultsFailed = &config.MetricThreshold{OneDay: proto.Int64(101)}
			So(cl.MeetsThreshold(t), ShouldBeFalse)
		})
		Convey("test runs failed thresholding", func() {
			t.TestRunsFailed = &config.MetricThreshold{OneDay: proto.Int64(30)}
			So(cl.MeetsThreshold(t), ShouldBeTrue)

			t.TestRunsFailed = &config.MetricThreshold{OneDay: proto.Int64(31)}
			So(cl.MeetsThreshold(t), ShouldBeFalse)
		})
		Convey("presubmit runs failed thresholding", func() {
			t.PresubmitRunsFailed = &config.MetricThreshold{OneDay: proto.Int64(3)}
			So(cl.MeetsThreshold(t), ShouldBeTrue)

			t.PresubmitRunsFailed = &config.MetricThreshold{OneDay: proto.Int64(4)}
			So(cl.MeetsThreshold(t), ShouldBeFalse)
		})
		Convey("one day threshold", func() {
			t.TestResultsFailed = &config.MetricThreshold{OneDay: proto.Int64(100)}
			So(cl.MeetsThreshold(t), ShouldBeTrue)

			t.TestResultsFailed = &config.MetricThreshold{OneDay: proto.Int64(101)}
			So(cl.MeetsThreshold(t), ShouldBeFalse)
		})
		Convey("three day threshold", func() {
			t.TestResultsFailed = &config.MetricThreshold{ThreeDay: proto.Int64(300)}
			So(cl.MeetsThreshold(t), ShouldBeTrue)

			t.TestResultsFailed = &config.MetricThreshold{ThreeDay: proto.Int64(301)}
			So(cl.MeetsThreshold(t), ShouldBeFalse)
		})
		Convey("seven day threshold", func() {
			t.TestResultsFailed = &config.MetricThreshold{SevenDay: proto.Int64(700)}
			So(cl.MeetsThreshold(t), ShouldBeTrue)

			t.TestResultsFailed = &config.MetricThreshold{SevenDay: proto.Int64(701)}
			So(cl.MeetsThreshold(t), ShouldBeFalse)
		})
		Convey("Threshold with deflation", func() {
			// With 15% hysteresis, leads to effective threshold of
			// 100 / 1.15 = 86.
			t.TestResultsFailed = &config.MetricThreshold{OneDay: proto.Int64(100)}
			cl.TestResultsFailed.OneDay = 86
			So(cl.MeetsInflatedThreshold(t, -15), ShouldBeTrue)
			cl.TestResultsFailed.OneDay = 85
			So(cl.MeetsInflatedThreshold(t, -15), ShouldBeFalse)
		})
		Convey("Threshold with inflation", func() {
			// With 15% hysteresis, leads to effective threshold of
			// 100 * 1.15 = 115.
			t.TestResultsFailed = &config.MetricThreshold{OneDay: proto.Int64(100)}
			cl.TestResultsFailed.OneDay = 115
			So(cl.MeetsInflatedThreshold(t, 15), ShouldBeTrue)
			cl.TestResultsFailed.OneDay = 114
			So(cl.MeetsInflatedThreshold(t, 15), ShouldBeFalse)
		})
		Convey("Thresholding of values near overflow", func() {
			t.TestResultsFailed = &config.MetricThreshold{OneDay: proto.Int64(math.MaxInt64)}
			cl.TestResultsFailed.OneDay = math.MaxInt64
			So(cl.MeetsThreshold(t), ShouldBeTrue)
			// Thresholding has loss of precision towards the max value of
			// an int64, so make sure observed number of failures noticibily less.
			cl.TestResultsFailed.OneDay = math.MaxInt64 - 10000
			So(cl.MeetsThreshold(t), ShouldBeFalse)
		})
		Convey("Thresholding with inflation near overflow", func() {
			t.TestResultsFailed = &config.MetricThreshold{OneDay: proto.Int64(math.MaxInt64)}
			cl.TestResultsFailed.OneDay = math.MaxInt64 / 11
			So(cl.MeetsInflatedThreshold(t, -1000), ShouldBeTrue)
			// Thresholding has loss of precision towards the max value of
			// an int64, so make sure observed number of failures noticibily less.
			cl.TestResultsFailed.OneDay = math.MaxInt64/11 - 10000
			So(cl.MeetsInflatedThreshold(t, -1000), ShouldBeFalse)
		})
		Convey("Thresholding with deflation near overflow", func() {
			t.TestResultsFailed = &config.MetricThreshold{OneDay: proto.Int64(math.MaxInt64 / 11)}
			cl.TestResultsFailed.OneDay = math.MaxInt64
			So(cl.MeetsInflatedThreshold(t, 1000), ShouldBeTrue)
			// Thresholding has loss of precision towards the max value of
			// an int64, so make sure observed number of failures noticibily less.
			cl.TestResultsFailed.OneDay = math.MaxInt64 - 10000
			So(cl.MeetsInflatedThreshold(t, 1000), ShouldBeFalse)
		})
	})
}
