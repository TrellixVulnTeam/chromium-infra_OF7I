// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package bugs

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
	"google.golang.org/protobuf/proto"

	configpb "infra/appengine/weetbix/internal/config/proto"
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
		Convey("MeetsThreshold", func() {
			t := &configpb.ImpactThreshold{}
			Convey("No cluster meets empty threshold", func() {
				So(cl.MeetsThreshold(t), ShouldBeFalse)
			})
			Convey("Test results failed thresholding", func() {
				t.TestResultsFailed = &configpb.MetricThreshold{OneDay: proto.Int64(100)}
				So(cl.MeetsThreshold(t), ShouldBeTrue)

				t.TestResultsFailed = &configpb.MetricThreshold{OneDay: proto.Int64(101)}
				So(cl.MeetsThreshold(t), ShouldBeFalse)
			})
			Convey("Test runs failed thresholding", func() {
				t.TestRunsFailed = &configpb.MetricThreshold{OneDay: proto.Int64(30)}
				So(cl.MeetsThreshold(t), ShouldBeTrue)

				t.TestRunsFailed = &configpb.MetricThreshold{OneDay: proto.Int64(31)}
				So(cl.MeetsThreshold(t), ShouldBeFalse)
			})
			Convey("Presubmit runs failed thresholding", func() {
				t.PresubmitRunsFailed = &configpb.MetricThreshold{OneDay: proto.Int64(3)}
				So(cl.MeetsThreshold(t), ShouldBeTrue)

				t.PresubmitRunsFailed = &configpb.MetricThreshold{OneDay: proto.Int64(4)}
				So(cl.MeetsThreshold(t), ShouldBeFalse)
			})
			Convey("One day threshold", func() {
				t.TestResultsFailed = &configpb.MetricThreshold{OneDay: proto.Int64(100)}
				So(cl.MeetsThreshold(t), ShouldBeTrue)

				t.TestResultsFailed = &configpb.MetricThreshold{OneDay: proto.Int64(101)}
				So(cl.MeetsThreshold(t), ShouldBeFalse)
			})
			Convey("Three day threshold", func() {
				t.TestResultsFailed = &configpb.MetricThreshold{ThreeDay: proto.Int64(300)}
				So(cl.MeetsThreshold(t), ShouldBeTrue)

				t.TestResultsFailed = &configpb.MetricThreshold{ThreeDay: proto.Int64(301)}
				So(cl.MeetsThreshold(t), ShouldBeFalse)
			})
			Convey("Seven day threshold", func() {
				t.TestResultsFailed = &configpb.MetricThreshold{SevenDay: proto.Int64(700)}
				So(cl.MeetsThreshold(t), ShouldBeTrue)

				t.TestResultsFailed = &configpb.MetricThreshold{SevenDay: proto.Int64(701)}
				So(cl.MeetsThreshold(t), ShouldBeFalse)
			})
		})
		Convey("InflateThreshold", func() {
			t := &configpb.ImpactThreshold{}
			Convey("Empty threshold", func() {
				result := InflateThreshold(t, 15)
				So(result, ShouldResembleProto, &configpb.ImpactThreshold{})
			})
			Convey("Test results failed", func() {
				t.TestResultsFailed = &configpb.MetricThreshold{OneDay: proto.Int64(100)}
				result := InflateThreshold(t, 15)
				So(result, ShouldResembleProto, &configpb.ImpactThreshold{
					TestResultsFailed: &configpb.MetricThreshold{OneDay: proto.Int64(115)},
				})
			})
			Convey("Test runs failed", func() {
				t.TestRunsFailed = &configpb.MetricThreshold{OneDay: proto.Int64(100)}
				result := InflateThreshold(t, 15)
				So(result, ShouldResembleProto, &configpb.ImpactThreshold{
					TestRunsFailed: &configpb.MetricThreshold{OneDay: proto.Int64(115)},
				})
			})
			Convey("Presubmit runs failed", func() {
				t.PresubmitRunsFailed = &configpb.MetricThreshold{OneDay: proto.Int64(100)}
				result := InflateThreshold(t, 15)
				So(result, ShouldResembleProto, &configpb.ImpactThreshold{
					PresubmitRunsFailed: &configpb.MetricThreshold{OneDay: proto.Int64(115)},
				})
			})
			Convey("One day threshold", func() {
				t.TestResultsFailed = &configpb.MetricThreshold{OneDay: proto.Int64(100)}
				result := InflateThreshold(t, 15)
				So(result, ShouldResembleProto, &configpb.ImpactThreshold{
					TestResultsFailed: &configpb.MetricThreshold{OneDay: proto.Int64(115)},
				})
			})
			Convey("Three day threshold", func() {
				t.TestResultsFailed = &configpb.MetricThreshold{ThreeDay: proto.Int64(100)}
				result := InflateThreshold(t, 15)
				So(result, ShouldResembleProto, &configpb.ImpactThreshold{
					TestResultsFailed: &configpb.MetricThreshold{ThreeDay: proto.Int64(115)},
				})
			})
			Convey("Seven day threshold", func() {
				t.TestResultsFailed = &configpb.MetricThreshold{SevenDay: proto.Int64(100)}
				result := InflateThreshold(t, 15)
				So(result, ShouldResembleProto, &configpb.ImpactThreshold{
					TestResultsFailed: &configpb.MetricThreshold{SevenDay: proto.Int64(115)},
				})
			})
		})
		Convey("ExplainThresholdMet", func() {
			t := &configpb.ImpactThreshold{
				TestResultsFailed: &configpb.MetricThreshold{
					OneDay:   proto.Int64(101), // Not met.
					ThreeDay: proto.Int64(299), // Met.
					SevenDay: proto.Int64(699), // Met.
				},
			}
			explanation := cl.ExplainThresholdMet(t)
			So(explanation, ShouldResemble, ThresholdExplanation{
				Metric:        "Test Results Failed",
				TimescaleDays: 3,
				Threshold:     299,
			})
		})
		Convey("ExplainThresholdNotMet", func() {
			t := &configpb.ImpactThreshold{
				TestResultsFailed: &configpb.MetricThreshold{
					OneDay: proto.Int64(101), // Not met.
				},
				TestRunsFailed: &configpb.MetricThreshold{
					ThreeDay: proto.Int64(301), // Not met.
				},
				PresubmitRunsFailed: &configpb.MetricThreshold{
					SevenDay: proto.Int64(701), // Not met.
				},
			}
			explanation := ExplainThresholdNotMet(t)
			So(explanation, ShouldResemble, []ThresholdExplanation{
				{
					Metric:        "Presubmit Runs Failed",
					TimescaleDays: 7,
					Threshold:     701,
				},
				{
					Metric:        "Test Runs Failed",
					TimescaleDays: 3,
					Threshold:     301,
				},
				{
					Metric:        "Test Results Failed",
					TimescaleDays: 1,
					Threshold:     101,
				},
			})
		})
		Convey("MergeThresholdMetExplanations", func() {
			input := []ThresholdExplanation{
				{
					Metric:        "Presubmit Runs Failed",
					TimescaleDays: 7,
					Threshold:     20,
				},
				{
					Metric:        "Test Runs Failed",
					TimescaleDays: 3,
					Threshold:     100,
				},
				{
					Metric:        "Presubmit Runs Failed",
					TimescaleDays: 7,
					Threshold:     10,
				},
				{
					Metric:        "Test Runs Failed",
					TimescaleDays: 3,
					Threshold:     200,
				},
				{
					Metric:        "Test Runs Failed",
					TimescaleDays: 7,
					Threshold:     700,
				},
			}
			result := MergeThresholdMetExplanations(input)
			So(result, ShouldResemble, []ThresholdExplanation{
				{
					Metric:        "Presubmit Runs Failed",
					TimescaleDays: 7,
					Threshold:     20,
				},
				{
					Metric:        "Test Runs Failed",
					TimescaleDays: 3,
					Threshold:     200,
				},
				{
					Metric:        "Test Runs Failed",
					TimescaleDays: 7,
					Threshold:     700,
				},
			})
		})
	})
}
