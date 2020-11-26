// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package conversion

import (
	"fmt"
	"net/url"
	"testing"

	"infra/chromeperf/pinpoint"

	"github.com/golang/protobuf/proto"
	. "github.com/smartystreets/goconvey/convey"
)

func shouldContainMap(actual interface{}, expected ...interface{}) string {
	v := actual.(url.Values)
	e := expected[0].(map[string]interface{})

	// Go through the list of expected keys and values and compare.
	for key, value := range e {
		actualValue, found := v[key]
		if !found {
			return fmt.Sprintf("expecting key '%s' (but is not there)", key)
		}
		switch value.(type) {
		case string:
			if actualValue[0] != value {
				return fmt.Sprintf("expecting actual['%s'] == %s (got %s instead)", key, value, actualValue[0])
			}
		case float64:
			if actualValue[0] != fmt.Sprintf("%f", value) {
				return fmt.Sprintf("expecting actual['%s'] == %f (got %s instead)", key, value, actualValue)
			}
		default:
			panic("Unsupported type!")
		}
	}
	return ""
}

func TestSimpleConversions(t *testing.T) {

	job := &pinpoint.JobSpec{
		Config: "some-config",
		Target: "some-build-target",
	}

	Convey("We support Bisections without a Patch", t, func() {
		job.JobKind = &pinpoint.JobSpec_Bisection{
			Bisection: &pinpoint.Bisection{
				CommitRange: &pinpoint.GitilesCommitRange{
					Host:         "gitiles-host",
					Project:      "gitiles-project",
					StartGitHash: "c0dec0de",
					EndGitHash:   "f00dc0de",
				}}}

		Convey("Creating a Performance mode job", func() {
			job.ComparisonMode = pinpoint.JobSpec_PERFORMANCE

			Convey("We support Telemetry specifying a story", func() {
				telemetryJob :=
					&pinpoint.JobSpec{
						ComparisonMagnitude: 1000.0,
						Arguments: &pinpoint.JobSpec_TelemetryBenchmark{
							TelemetryBenchmark: &pinpoint.TelemetryBenchmark{
								Benchmark: "some-benchmark",
								StorySelection: &pinpoint.TelemetryBenchmark_Story{
									Story: "some-story"},
								Measurement:   "some-metric",
								GroupingLabel: "some-grouping-label",
								Statistic:     pinpoint.TelemetryBenchmark_NONE}}}
				proto.Merge(telemetryJob, job)
				v, err := ConvertToValues(telemetryJob, "user@example.com")
				So(err, ShouldBeNil)

				// Check that we support the required fields for all Pinpoint jobs.
				So(v, shouldContainMap, map[string]interface{}{
					"target":        "some-build-target",
					"configuration": "some-config",
				})

				// Check that we have the required Telemetry fields in the JSON.
				So(v, shouldContainMap, map[string]interface{}{
					"benchmark":      "some-benchmark",
					"story":          "some-story",
					"metric":         "some-metric",
					"grouping_label": "some-grouping-label"})

				// Check that we have base job configurations are set.
				So(v, shouldContainMap, map[string]interface{}{
					"configuration":        "some-config",
					"comparison_mode":      "performance",
					"comparison_magnitude": 1000.0,
				})

				// Check that we also get the bisection details correct.
				So(v, shouldContainMap, map[string]interface{}{
					"start_git_hash": "c0dec0de",
					"end_git_hash":   "f00dc0de"})
			})

			Convey("We support Telemetry specifying story tags", func() {
				telemetryJob :=
					&pinpoint.JobSpec{
						ComparisonMagnitude: 1000.0,
						Arguments: &pinpoint.JobSpec_TelemetryBenchmark{
							TelemetryBenchmark: &pinpoint.TelemetryBenchmark{
								Benchmark: "some-benchmark",
								StorySelection: &pinpoint.TelemetryBenchmark_StoryTags{
									StoryTags: &pinpoint.TelemetryBenchmark_StoryTagList{
										StoryTags: []string{"some-tag", "some-other-tag"},
									}},
								Measurement:   "some-metric",
								GroupingLabel: "some-grouping-label",
								Statistic:     pinpoint.TelemetryBenchmark_NONE}}}
				proto.Merge(telemetryJob, job)
				v, err := ConvertToValues(telemetryJob, "user@example.com")
				So(err, ShouldBeNil)

				// Check that we support the required fields for all Pinpoint jobs.
				So(v, shouldContainMap, map[string]interface{}{
					"target":        "some-build-target",
					"configuration": "some-config",
				})

				// Check that we have the required Telemetry fields in the JSON.
				So(v, shouldContainMap, map[string]interface{}{
					"benchmark":      "some-benchmark",
					"metric":         "some-metric",
					"story_tags":     "some-tag,some-other-tag",
					"grouping_label": "some-grouping-label"})

				// Check that we have base job configurations are set.
				So(v, shouldContainMap, map[string]interface{}{
					"configuration":        "some-config",
					"comparison_mode":      "performance",
					"comparison_magnitude": 1000.0,
				})

				// Check that we also get the bisection details correct.
				So(v, shouldContainMap, map[string]interface{}{
					"start_git_hash": "c0dec0de",
					"end_git_hash":   "f00dc0de"})

			})

			Convey("We support GTest", func() {
				gtestJob := &pinpoint.JobSpec{
					ComparisonMagnitude: 1000.0,
					Arguments: &pinpoint.JobSpec_GtestBenchmark{
						GtestBenchmark: &pinpoint.GTestBenchmark{
							Benchmark:   "some-benchmark",
							Measurement: "some-metric",
							Test:        "some-test"}}}
				proto.Merge(gtestJob, job)
				v, err := ConvertToValues(gtestJob, "user@example.com")
				So(err, ShouldBeNil)

				// Check that we support the required fields for all Pinpoint jobs.
				So(v, shouldContainMap, map[string]interface{}{
					"target":        "some-build-target",
					"configuration": "some-config",
				})

				// Check the conversion of values to maps.
				So(v, shouldContainMap, map[string]interface{}{
					"benchmark": "some-benchmark",
					"trace":     "some-test",
					"chart":     "some-metric"})

				// Check that we have base job configurations are set.
				So(v, shouldContainMap, map[string]interface{}{
					"configuration":        "some-config",
					"comparison_mode":      "performance",
					"comparison_magnitude": 1000.0,
				})

				// Check that we also get the bisection details correct.
				So(v, shouldContainMap, map[string]interface{}{
					"start_git_hash": "c0dec0de",
					"end_git_hash":   "f00dc0de"})
			})

		})

		Convey("Creating a Functional Comparison", func() {
			job.ComparisonMode = pinpoint.JobSpec_FUNCTIONAL

			Convey("We support Telemetry specifying a story", func() {
				telemetryJob :=
					&pinpoint.JobSpec{
						ComparisonMagnitude: 0.2,
						Arguments: &pinpoint.JobSpec_TelemetryBenchmark{
							TelemetryBenchmark: &pinpoint.TelemetryBenchmark{
								Benchmark: "some-benchmark",
								StorySelection: &pinpoint.TelemetryBenchmark_Story{
									Story: "some-story"},
								Measurement:   "some-metric",
								GroupingLabel: "some-grouping-label",
								Statistic:     pinpoint.TelemetryBenchmark_NONE}}}
				proto.Merge(telemetryJob, job)
				v, err := ConvertToValues(telemetryJob, "user@example.com")
				So(err, ShouldBeNil)

				// Check that we support the required fields for all Pinpoint jobs.
				So(v, shouldContainMap, map[string]interface{}{
					"target":        "some-build-target",
					"configuration": "some-config",
				})

				// Check that we have the required Telemetry fields in the JSON.
				So(v, shouldContainMap, map[string]interface{}{
					"benchmark":      "some-benchmark",
					"story":          "some-story",
					"metric":         "some-metric",
					"grouping_label": "some-grouping-label"})

				// Check that we have base job configurations are set.
				So(v, shouldContainMap, map[string]interface{}{
					"configuration":        "some-config",
					"comparison_mode":      "functional",
					"comparison_magnitude": 0.2,
				})

				// Check that we also get the bisection details correct.
				So(v, shouldContainMap, map[string]interface{}{
					"start_git_hash": "c0dec0de",
					"end_git_hash":   "f00dc0de"})

			})

			Convey("We support Telemetry specifying story tags", func() {
				telemetryJob :=
					&pinpoint.JobSpec{
						ComparisonMagnitude: 0.2,
						Arguments: &pinpoint.JobSpec_TelemetryBenchmark{
							TelemetryBenchmark: &pinpoint.TelemetryBenchmark{
								Benchmark: "some-benchmark",
								StorySelection: &pinpoint.TelemetryBenchmark_StoryTags{
									StoryTags: &pinpoint.TelemetryBenchmark_StoryTagList{
										StoryTags: []string{"some-tag", "some-other-tag"},
									}},
								Measurement:   "some-metric",
								GroupingLabel: "some-grouping-label",
								Statistic:     pinpoint.TelemetryBenchmark_NONE}}}
				proto.Merge(telemetryJob, job)
				v, err := ConvertToValues(telemetryJob, "user@example.com")
				So(err, ShouldBeNil)

				// Check that we support the required fields for all Pinpoint jobs.
				So(v, shouldContainMap, map[string]interface{}{
					"target":        "some-build-target",
					"configuration": "some-config",
				})

				// Check that we have the required Telemetry fields in the JSON.
				So(v, shouldContainMap, map[string]interface{}{
					"benchmark":      "some-benchmark",
					"metric":         "some-metric",
					"story_tags":     "some-tag,some-other-tag",
					"grouping_label": "some-grouping-label"})

				// Check that we have base job configurations are set.
				So(v, shouldContainMap, map[string]interface{}{
					"configuration":        "some-config",
					"comparison_mode":      "functional",
					"comparison_magnitude": 0.2,
				})

				// Check that we also get the bisection details correct.
				So(v, shouldContainMap, map[string]interface{}{
					"start_git_hash": "c0dec0de",
					"end_git_hash":   "f00dc0de"})

			})

			Convey("We support GTest", func() {
				gtestJob := &pinpoint.JobSpec{
					ComparisonMagnitude: 0.2,
					Arguments: &pinpoint.JobSpec_GtestBenchmark{
						GtestBenchmark: &pinpoint.GTestBenchmark{
							Benchmark:   "some-benchmark",
							Measurement: "some-metric",
							Test:        "some-test"}}}
				proto.Merge(gtestJob, job)
				v, err := ConvertToValues(gtestJob, "user@example.com")
				So(err, ShouldBeNil)

				// Check that we support the required fields for all Pinpoint jobs.
				So(v, shouldContainMap, map[string]interface{}{
					"target":        "some-build-target",
					"configuration": "some-config",
				})

				// Check the conversion of values to maps.
				So(v, shouldContainMap, map[string]interface{}{
					"benchmark": "some-benchmark",
					"trace":     "some-test",
					"chart":     "some-metric"})

				// Check that we have base job configurations are set.
				So(v, shouldContainMap, map[string]interface{}{
					"configuration":        "some-config",
					"comparison_mode":      "functional",
					"comparison_magnitude": 0.2,
				})

				// Check that we also get the bisection details correct.
				So(v, shouldContainMap, map[string]interface{}{
					"start_git_hash": "c0dec0de",
					"end_git_hash":   "f00dc0de"})
			})

		})

	})

	Convey("We support Bisections with a Patch", t, func() {
		job.JobKind = &pinpoint.JobSpec_Bisection{
			Bisection: &pinpoint.Bisection{
				CommitRange: &pinpoint.GitilesCommitRange{
					Host:         "gitiles-host",
					Project:      "gitiles-project",
					StartGitHash: "c0dec0de",
					EndGitHash:   "f00dc0de",
				},
				Patch: &pinpoint.GerritChange{
					Host:     "some-gerrit-host",
					Project:  "some-project",
					Change:   12345,
					Patchset: 1}}}

		Convey("Creating a Performance mode job", func() {
			job.ComparisonMode = pinpoint.JobSpec_PERFORMANCE

			Convey("We support Telemetry specifying a story", func() {
				telemetryJob :=
					&pinpoint.JobSpec{
						ComparisonMagnitude: 1000.0,
						Arguments: &pinpoint.JobSpec_TelemetryBenchmark{
							TelemetryBenchmark: &pinpoint.TelemetryBenchmark{
								Benchmark: "some-benchmark",
								StorySelection: &pinpoint.TelemetryBenchmark_Story{
									Story: "some-story"},
								Measurement:   "some-metric",
								GroupingLabel: "some-grouping-label",
								Statistic:     pinpoint.TelemetryBenchmark_NONE}}}
				proto.Merge(telemetryJob, job)
				v, err := ConvertToValues(telemetryJob, "user@example.com")
				So(err, ShouldBeNil)

				// Check that we support the required fields for all Pinpoint jobs.
				So(v, shouldContainMap, map[string]interface{}{
					"target":        "some-build-target",
					"configuration": "some-config",
				})

				// Check that we have the required Telemetry fields in the JSON.
				So(v, shouldContainMap, map[string]interface{}{
					"benchmark":      "some-benchmark",
					"story":          "some-story",
					"metric":         "some-metric",
					"grouping_label": "some-grouping-label"})

				// Check that we have base job configurations are set.
				So(v, shouldContainMap, map[string]interface{}{
					"configuration":        "some-config",
					"comparison_mode":      "performance",
					"comparison_magnitude": 1000.0,
				})

				// Check that we also get the bisection details correct.
				So(v, shouldContainMap, map[string]interface{}{
					"start_git_hash": "c0dec0de",
					"end_git_hash":   "f00dc0de",
					// Here we're hard-coding the expected URL, as it's required by the legacy
					// Pinpoint API.
					"patch": "https://some-gerrit-host/c/some-project/+/12345/1"})
			})

			Convey("We support Telemetry specifying story tags", func() {
				telemetryJob :=
					&pinpoint.JobSpec{
						ComparisonMagnitude: 1000.0,
						Arguments: &pinpoint.JobSpec_TelemetryBenchmark{
							TelemetryBenchmark: &pinpoint.TelemetryBenchmark{
								Benchmark: "some-benchmark",
								StorySelection: &pinpoint.TelemetryBenchmark_StoryTags{
									StoryTags: &pinpoint.TelemetryBenchmark_StoryTagList{
										StoryTags: []string{"some-tag", "some-other-tag"},
									}},
								Measurement:   "some-metric",
								GroupingLabel: "some-grouping-label",
								Statistic:     pinpoint.TelemetryBenchmark_NONE}}}
				proto.Merge(telemetryJob, job)
				v, err := ConvertToValues(telemetryJob, "user@example.com")
				So(err, ShouldBeNil)

				// Check that we support the required fields for all Pinpoint jobs.
				So(v, shouldContainMap, map[string]interface{}{
					"target":        "some-build-target",
					"configuration": "some-config",
				})

				// Check that we have the required Telemetry fields in the JSON.
				So(v, shouldContainMap, map[string]interface{}{
					"benchmark":      "some-benchmark",
					"metric":         "some-metric",
					"story_tags":     "some-tag,some-other-tag",
					"grouping_label": "some-grouping-label"})

				// Check that we have base job configurations are set.
				So(v, shouldContainMap, map[string]interface{}{
					"configuration":        "some-config",
					"comparison_mode":      "performance",
					"comparison_magnitude": 1000.0,
				})

				// Check that we also get the bisection details correct.
				So(v, shouldContainMap, map[string]interface{}{
					"start_git_hash": "c0dec0de",
					"end_git_hash":   "f00dc0de",
					// Here we're hard-coding the expected URL, as it's required by the legacy
					// Pinpoint API.
					"patch": "https://some-gerrit-host/c/some-project/+/12345/1"})

			})
			Convey("We support GTest", func() {
				gtestJob := &pinpoint.JobSpec{
					ComparisonMagnitude: 1000.0,
					Arguments: &pinpoint.JobSpec_GtestBenchmark{
						GtestBenchmark: &pinpoint.GTestBenchmark{
							Benchmark:   "some-benchmark",
							Measurement: "some-metric",
							Test:        "some-test"}}}
				proto.Merge(gtestJob, job)
				v, err := ConvertToValues(gtestJob, "user@example.com")
				So(err, ShouldBeNil)

				// Check that we support the required fields for all Pinpoint jobs.
				So(v, shouldContainMap, map[string]interface{}{
					"target":        "some-build-target",
					"configuration": "some-config",
				})

				// Check the conversion of values to maps.
				So(v, shouldContainMap, map[string]interface{}{
					"benchmark": "some-benchmark",
					"trace":     "some-test",
					"chart":     "some-metric"})

				// Check that we have base job configurations are set.
				So(v, shouldContainMap, map[string]interface{}{
					"configuration":        "some-config",
					"comparison_mode":      "performance",
					"comparison_magnitude": 1000.0,
				})

				// Check that we also get the bisection details correct.
				So(v, shouldContainMap, map[string]interface{}{
					"start_git_hash": "c0dec0de",
					"end_git_hash":   "f00dc0de",
					// Here we're hard-coding the expected URL, as it's required by the legacy
					// Pinpoint API.
					"patch": "https://some-gerrit-host/c/some-project/+/12345/1"})
			})

		})

		Convey("Creating a Functional Comparison", func() {
			job.ComparisonMode = pinpoint.JobSpec_FUNCTIONAL

			Convey("We support Telemetry specifying a story", func() {
				telemetryJob :=
					&pinpoint.JobSpec{
						ComparisonMagnitude: 0.2,
						Arguments: &pinpoint.JobSpec_TelemetryBenchmark{
							TelemetryBenchmark: &pinpoint.TelemetryBenchmark{
								Benchmark: "some-benchmark",
								StorySelection: &pinpoint.TelemetryBenchmark_Story{
									Story: "some-story"},
								Measurement:   "some-metric",
								GroupingLabel: "some-grouping-label",
								Statistic:     pinpoint.TelemetryBenchmark_NONE}}}
				proto.Merge(telemetryJob, job)
				v, err := ConvertToValues(telemetryJob, "user@example.com")
				So(err, ShouldBeNil)

				// Check that we support the required fields for all Pinpoint jobs.
				So(v, shouldContainMap, map[string]interface{}{
					"target":        "some-build-target",
					"configuration": "some-config",
				})

				// Check that we have the required Telemetry fields in the JSON.
				So(v, shouldContainMap, map[string]interface{}{
					"benchmark":      "some-benchmark",
					"story":          "some-story",
					"metric":         "some-metric",
					"grouping_label": "some-grouping-label"})

				// Check that we have base job configurations are set.
				So(v, shouldContainMap, map[string]interface{}{
					"configuration":        "some-config",
					"comparison_mode":      "functional",
					"comparison_magnitude": 0.2,
				})

				// Check that we also get the bisection details correct.
				So(v, shouldContainMap, map[string]interface{}{
					"start_git_hash": "c0dec0de",
					"end_git_hash":   "f00dc0de",
					// Here we're hard-coding the expected URL, as it's required by the legacy
					// Pinpoint API.
					"patch": "https://some-gerrit-host/c/some-project/+/12345/1"})

			})

			Convey("We support Telemetry specifying story tags", func() {
				telemetryJob :=
					&pinpoint.JobSpec{
						ComparisonMagnitude: 0.2,
						Arguments: &pinpoint.JobSpec_TelemetryBenchmark{
							TelemetryBenchmark: &pinpoint.TelemetryBenchmark{
								Benchmark: "some-benchmark",
								StorySelection: &pinpoint.TelemetryBenchmark_StoryTags{
									StoryTags: &pinpoint.TelemetryBenchmark_StoryTagList{
										StoryTags: []string{"some-tag", "some-other-tag"},
									}},
								Measurement:   "some-metric",
								GroupingLabel: "some-grouping-label",
								Statistic:     pinpoint.TelemetryBenchmark_NONE}}}
				proto.Merge(telemetryJob, job)
				v, err := ConvertToValues(telemetryJob, "user@example.com")
				So(err, ShouldBeNil)

				// Check that we support the required fields for all Pinpoint jobs.
				So(v, shouldContainMap, map[string]interface{}{
					"target":        "some-build-target",
					"configuration": "some-config",
				})

				// Check that we have the required Telemetry fields in the JSON.
				So(v, shouldContainMap, map[string]interface{}{
					"benchmark":      "some-benchmark",
					"metric":         "some-metric",
					"story_tags":     "some-tag,some-other-tag",
					"grouping_label": "some-grouping-label"})

				// Check that we have base job configurations are set.
				So(v, shouldContainMap, map[string]interface{}{
					"configuration":        "some-config",
					"comparison_mode":      "functional",
					"comparison_magnitude": 0.2,
				})

				// Check that we also get the bisection details correct.
				// Check that we also get the bisection details correct.
				So(v, shouldContainMap, map[string]interface{}{
					"start_git_hash": "c0dec0de",
					"end_git_hash":   "f00dc0de",
					// Here we're hard-coding the expected URL, as it's required by the legacy
					// Pinpoint API.
					"patch": "https://some-gerrit-host/c/some-project/+/12345/1"})

			})

			Convey("We support GTest", func() {
				gtestJob := &pinpoint.JobSpec{
					ComparisonMagnitude: 0.2,
					Arguments: &pinpoint.JobSpec_GtestBenchmark{
						GtestBenchmark: &pinpoint.GTestBenchmark{
							Benchmark:   "some-benchmark",
							Measurement: "some-metric",
							Test:        "some-test"}}}
				proto.Merge(gtestJob, job)
				v, err := ConvertToValues(gtestJob, "user@example.com")
				So(err, ShouldBeNil)

				// Check that we support the required fields for all Pinpoint jobs.
				So(v, shouldContainMap, map[string]interface{}{
					"target":        "some-build-target",
					"configuration": "some-config",
				})

				// Check the conversion of values to maps.
				So(v, shouldContainMap, map[string]interface{}{
					"benchmark": "some-benchmark",
					"trace":     "some-test",
					"chart":     "some-metric"})

				// Check that we have base job configurations are set.
				So(v, shouldContainMap, map[string]interface{}{
					"configuration":        "some-config",
					"comparison_mode":      "functional",
					"comparison_magnitude": 0.2,
				})

				// Check that we also get the bisection details correct.
				So(v, shouldContainMap, map[string]interface{}{
					"start_git_hash": "c0dec0de",
					"end_git_hash":   "f00dc0de"})
			})

		})

	})

	Convey("We support experiments with base commit and experiment patch", t, func() {
		job.JobKind = &pinpoint.JobSpec_Experiment{
			Experiment: &pinpoint.Experiment{
				BaseCommit: &pinpoint.GitilesCommit{
					Host:    "some-gitiles-host",
					Project: "some-gitiles-project",
					GitHash: "c0dec0de",
				},
				ExperimentPatch: &pinpoint.GerritChange{
					Host:     "some-gerrit-host",
					Project:  "some-gerrit-project",
					Change:   23456,
					Patchset: 1,
				}}}

		Convey("Creating a Performance mode job", func() {
			job.ComparisonMode = pinpoint.JobSpec_PERFORMANCE

			Convey("We support Telemetry specifying a story", func() {
				telemetryJob :=
					&pinpoint.JobSpec{
						Arguments: &pinpoint.JobSpec_TelemetryBenchmark{
							TelemetryBenchmark: &pinpoint.TelemetryBenchmark{
								Benchmark: "some-benchmark",
								StorySelection: &pinpoint.TelemetryBenchmark_Story{
									Story: "some-story"},
								Measurement:   "some-metric",
								GroupingLabel: "some-grouping-label",
								Statistic:     pinpoint.TelemetryBenchmark_NONE}}}
				proto.Merge(telemetryJob, job)
				v, err := ConvertToValues(telemetryJob, "user@example.com")
				So(err, ShouldBeNil)

				// Check that we support the required fields for all Pinpoint jobs.
				So(v, shouldContainMap, map[string]interface{}{
					"target":        "some-build-target",
					"configuration": "some-config",
				})

				// Check that we have the required Telemetry fields in the JSON.
				So(v, shouldContainMap, map[string]interface{}{
					"benchmark":      "some-benchmark",
					"story":          "some-story",
					"metric":         "some-metric",
					"grouping_label": "some-grouping-label"})

				// Check that we have base job configurations are set.
				So(v, shouldContainMap, map[string]interface{}{
					"configuration": "some-config",
					// In legacy Pinpoint, an experiment is a "try" comparison mode.
					"comparison_mode": "try",
				})

				So(v, shouldContainMap, map[string]interface{}{
					"base_git_hash": "c0dec0de",
					"patch":         "https://some-gerrit-host/c/some-gerrit-project/+/23456/1"})

			})

			Convey("We support Telemetry specifying story tags", func() {
				telemetryJob :=
					&pinpoint.JobSpec{
						Arguments: &pinpoint.JobSpec_TelemetryBenchmark{
							TelemetryBenchmark: &pinpoint.TelemetryBenchmark{
								Benchmark: "some-benchmark",
								StorySelection: &pinpoint.TelemetryBenchmark_StoryTags{
									StoryTags: &pinpoint.TelemetryBenchmark_StoryTagList{
										StoryTags: []string{"some-tag", "some-other-tag"},
									}},
								Measurement:   "some-metric",
								GroupingLabel: "some-grouping-label",
								Statistic:     pinpoint.TelemetryBenchmark_NONE}}}
				proto.Merge(telemetryJob, job)
				v, err := ConvertToValues(telemetryJob, "user@example.com")
				So(err, ShouldBeNil)

				// Check that we support the required fields for all Pinpoint jobs.
				So(v, shouldContainMap, map[string]interface{}{
					"target":        "some-build-target",
					"configuration": "some-config",
				})

				// Check that we have the required Telemetry fields in the JSON.
				So(v, shouldContainMap, map[string]interface{}{
					"benchmark":      "some-benchmark",
					"metric":         "some-metric",
					"story_tags":     "some-tag,some-other-tag",
					"grouping_label": "some-grouping-label"})

				// Check that we have base job configurations are set.
				So(v, shouldContainMap, map[string]interface{}{
					"configuration": "some-config",
					// In legacy Pinpoint, an experiment is a "try" comparison mode.
					"comparison_mode": "try",
				})

				So(v, shouldContainMap, map[string]interface{}{
					"base_git_hash": "c0dec0de",
					"patch":         "https://some-gerrit-host/c/some-gerrit-project/+/23456/1"})

			})

			Convey("We support GTest", func() {
				gtestJob := &pinpoint.JobSpec{
					Arguments: &pinpoint.JobSpec_GtestBenchmark{
						GtestBenchmark: &pinpoint.GTestBenchmark{
							Benchmark:   "some-benchmark",
							Measurement: "some-metric",
							Test:        "some-test"}}}
				proto.Merge(gtestJob, job)
				v, err := ConvertToValues(gtestJob, "user@example.com")
				So(err, ShouldBeNil)

				// Check that we support the required fields for all Pinpoint jobs.
				So(v, shouldContainMap, map[string]interface{}{
					"target":        "some-build-target",
					"configuration": "some-config",
				})

				// Check the conversion of values to maps.
				So(v, shouldContainMap, map[string]interface{}{
					"benchmark": "some-benchmark",
					"trace":     "some-test",
					"chart":     "some-metric"})

				// Check that we have base job configurations are set.
				So(v, shouldContainMap, map[string]interface{}{
					"configuration":   "some-config",
					"comparison_mode": "try",
				})

				So(v, shouldContainMap, map[string]interface{}{
					"base_git_hash": "c0dec0de",
					"patch":         "https://some-gerrit-host/c/some-gerrit-project/+/23456/1"})
			})

		})

		Convey("Creating a Functional mode job", func() {
			job.ComparisonMode = pinpoint.JobSpec_FUNCTIONAL

			Convey("Fails for Telemetry (unsupported)", func() {
				telemetryJob :=
					&pinpoint.JobSpec{
						Arguments: &pinpoint.JobSpec_TelemetryBenchmark{
							TelemetryBenchmark: &pinpoint.TelemetryBenchmark{
								Benchmark: "some-benchmark",
								StorySelection: &pinpoint.TelemetryBenchmark_Story{
									Story: "some-story"},
								Measurement:   "some-metric",
								GroupingLabel: "some-grouping-label",
								Statistic:     pinpoint.TelemetryBenchmark_NONE}}}
				proto.Merge(telemetryJob, job)
				_, err := ConvertToValues(telemetryJob, "user@example.com")
				So(err, ShouldNotBeNil)
				So(fmt.Sprintf("%v", err), ShouldContainSubstring, "functional experiments not supported")
			})

			Convey("Fails for GTest (unsupported)", func() {
				gtestJob := &pinpoint.JobSpec{
					Arguments: &pinpoint.JobSpec_GtestBenchmark{
						GtestBenchmark: &pinpoint.GTestBenchmark{
							Benchmark:   "some-benchmark",
							Measurement: "some-metric",
							Test:        "some-test"}}}
				proto.Merge(gtestJob, job)
				_, err := ConvertToValues(gtestJob, "user@example.com")
				So(err, ShouldNotBeNil)
				So(fmt.Sprintf("%v", err), ShouldContainSubstring, "functional experiments not supported")
			})
		})

	})

}
