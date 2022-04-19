// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package convert

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"

	pinpoint_proto "infra/chromeperf/pinpoint/proto"

	. "github.com/smartystreets/goconvey/convey"
	"google.golang.org/protobuf/proto"
)

const testPriority = 42

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
	t.Parallel()
	job := &pinpoint_proto.JobSpec{
		Config:    "some-config",
		Target:    "some-build-target",
		UserAgent: "pinpoint/unittest",
	}

	Convey("We support Bisections without a Patch", t, func() {
		job.JobKind = &pinpoint_proto.JobSpec_Bisection{
			Bisection: &pinpoint_proto.Bisection{
				CommitRange: &pinpoint_proto.GitilesCommitRange{
					Host:         "gitiles-host",
					Project:      "gitiles-project",
					StartGitHash: "c0dec0de",
					EndGitHash:   "f00dc0de",
				}}}

		Convey("Creating a Performance mode job", func() {
			job.ComparisonMode = pinpoint_proto.JobSpec_PERFORMANCE

			Convey("We support Telemetry specifying a story", func() {
				telemetryJob :=
					&pinpoint_proto.JobSpec{
						ComparisonMagnitude: 1000.0,
						Arguments: &pinpoint_proto.JobSpec_TelemetryBenchmark{
							TelemetryBenchmark: &pinpoint_proto.TelemetryBenchmark{
								Benchmark: "some-benchmark",
								StorySelection: &pinpoint_proto.TelemetryBenchmark_Story{
									Story: "some-story"},
								Measurement:   "some-metric",
								GroupingLabel: "some-grouping-label",
								Statistic:     pinpoint_proto.TelemetryBenchmark_NONE}}}
				proto.Merge(telemetryJob, job)
				v, err := JobToValues(telemetryJob, "user@example.com")
				So(err, ShouldBeNil)

				// Check that we have the user agent.
				So(v, shouldContainMap, map[string]interface{}{
					"user_agent": "pinpoint/unittest",
				})

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

				// Check that priority is unset
				So(v, shouldContainMap, map[string]interface{}{
					"priority": "0"})
			})

			Convey("We support Telemetry specifying priority", func() {
				telemetryJob :=
					&pinpoint_proto.JobSpec{
						Priority:            testPriority,
						ComparisonMagnitude: 1000.0,
						Arguments: &pinpoint_proto.JobSpec_TelemetryBenchmark{
							TelemetryBenchmark: &pinpoint_proto.TelemetryBenchmark{
								Benchmark: "some-benchmark",
								StorySelection: &pinpoint_proto.TelemetryBenchmark_Story{
									Story: "some-story"},
								Measurement:   "some-metric",
								GroupingLabel: "some-grouping-label",
								Statistic:     pinpoint_proto.TelemetryBenchmark_NONE}}}
				proto.Merge(telemetryJob, job)
				v, err := JobToValues(telemetryJob, "user@example.com")
				So(err, ShouldBeNil)

				So(v, shouldContainMap, map[string]interface{}{
					"priority": fmt.Sprintf("%d", testPriority)})
			})

			Convey("We support Telemetry specifying story tags", func() {
				telemetryJob :=
					&pinpoint_proto.JobSpec{
						ComparisonMagnitude: 1000.0,
						Arguments: &pinpoint_proto.JobSpec_TelemetryBenchmark{
							TelemetryBenchmark: &pinpoint_proto.TelemetryBenchmark{
								Benchmark: "some-benchmark",
								StorySelection: &pinpoint_proto.TelemetryBenchmark_StoryTags{
									StoryTags: &pinpoint_proto.TelemetryBenchmark_StoryTagList{
										StoryTags: []string{"some-tag", "some-other-tag"},
									}},
								Measurement:   "some-metric",
								GroupingLabel: "some-grouping-label",
								Statistic:     pinpoint_proto.TelemetryBenchmark_NONE}}}
				proto.Merge(telemetryJob, job)
				v, err := JobToValues(telemetryJob, "user@example.com")
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
				gtestJob := &pinpoint_proto.JobSpec{
					ComparisonMagnitude: 1000.0,
					Arguments: &pinpoint_proto.JobSpec_GtestBenchmark{
						GtestBenchmark: &pinpoint_proto.GTestBenchmark{
							Benchmark:   "some-benchmark",
							Measurement: "some-metric",
							Test:        "some-test"}}}
				proto.Merge(gtestJob, job)
				v, err := JobToValues(gtestJob, "user@example.com")
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
			job.ComparisonMode = pinpoint_proto.JobSpec_FUNCTIONAL

			Convey("We support Telemetry specifying a story", func() {
				telemetryJob :=
					&pinpoint_proto.JobSpec{
						ComparisonMagnitude: 0.2,
						Arguments: &pinpoint_proto.JobSpec_TelemetryBenchmark{
							TelemetryBenchmark: &pinpoint_proto.TelemetryBenchmark{
								Benchmark: "some-benchmark",
								StorySelection: &pinpoint_proto.TelemetryBenchmark_Story{
									Story: "some-story"},
								Measurement:   "some-metric",
								GroupingLabel: "some-grouping-label",
								Statistic:     pinpoint_proto.TelemetryBenchmark_NONE}}}
				proto.Merge(telemetryJob, job)
				v, err := JobToValues(telemetryJob, "user@example.com")
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
					&pinpoint_proto.JobSpec{
						ComparisonMagnitude: 0.2,
						Arguments: &pinpoint_proto.JobSpec_TelemetryBenchmark{
							TelemetryBenchmark: &pinpoint_proto.TelemetryBenchmark{
								Benchmark: "some-benchmark",
								StorySelection: &pinpoint_proto.TelemetryBenchmark_StoryTags{
									StoryTags: &pinpoint_proto.TelemetryBenchmark_StoryTagList{
										StoryTags: []string{"some-tag", "some-other-tag"},
									}},
								Measurement:   "some-metric",
								GroupingLabel: "some-grouping-label",
								Statistic:     pinpoint_proto.TelemetryBenchmark_NONE}}}
				proto.Merge(telemetryJob, job)
				v, err := JobToValues(telemetryJob, "user@example.com")
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
				gtestJob := &pinpoint_proto.JobSpec{
					ComparisonMagnitude: 0.2,
					Arguments: &pinpoint_proto.JobSpec_GtestBenchmark{
						GtestBenchmark: &pinpoint_proto.GTestBenchmark{
							Benchmark:   "some-benchmark",
							Measurement: "some-metric",
							Test:        "some-test"}}}
				proto.Merge(gtestJob, job)
				v, err := JobToValues(gtestJob, "user@example.com")
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
		job.JobKind = &pinpoint_proto.JobSpec_Bisection{
			Bisection: &pinpoint_proto.Bisection{
				CommitRange: &pinpoint_proto.GitilesCommitRange{
					Host:         "gitiles-host",
					Project:      "gitiles-project",
					StartGitHash: "c0dec0de",
					EndGitHash:   "f00dc0de",
				},
				Patch: &pinpoint_proto.GerritChange{
					Host:     "some-gerrit-host",
					Project:  "some-project",
					Change:   12345,
					Patchset: 1}}}

		Convey("Creating a Performance mode job", func() {
			job.ComparisonMode = pinpoint_proto.JobSpec_PERFORMANCE

			Convey("We support Telemetry specifying a story", func() {
				telemetryJob :=
					&pinpoint_proto.JobSpec{
						ComparisonMagnitude: 1000.0,
						Arguments: &pinpoint_proto.JobSpec_TelemetryBenchmark{
							TelemetryBenchmark: &pinpoint_proto.TelemetryBenchmark{
								Benchmark: "some-benchmark",
								StorySelection: &pinpoint_proto.TelemetryBenchmark_Story{
									Story: "some-story"},
								Measurement:   "some-metric",
								GroupingLabel: "some-grouping-label",
								Statistic:     pinpoint_proto.TelemetryBenchmark_NONE}}}
				proto.Merge(telemetryJob, job)
				v, err := JobToValues(telemetryJob, "user@example.com")
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
					"patch": "https://some-gerrit-host/c/12345/1"})
			})

			Convey("We support Telemetry specifying story tags", func() {
				telemetryJob :=
					&pinpoint_proto.JobSpec{
						ComparisonMagnitude: 1000.0,
						Arguments: &pinpoint_proto.JobSpec_TelemetryBenchmark{
							TelemetryBenchmark: &pinpoint_proto.TelemetryBenchmark{
								Benchmark: "some-benchmark",
								StorySelection: &pinpoint_proto.TelemetryBenchmark_StoryTags{
									StoryTags: &pinpoint_proto.TelemetryBenchmark_StoryTagList{
										StoryTags: []string{"some-tag", "some-other-tag"},
									}},
								Measurement:   "some-metric",
								GroupingLabel: "some-grouping-label",
								Statistic:     pinpoint_proto.TelemetryBenchmark_NONE}}}
				proto.Merge(telemetryJob, job)
				v, err := JobToValues(telemetryJob, "user@example.com")
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
					"patch": "https://some-gerrit-host/c/12345/1"})

			})
			Convey("We support GTest", func() {
				gtestJob := &pinpoint_proto.JobSpec{
					ComparisonMagnitude: 1000.0,
					Arguments: &pinpoint_proto.JobSpec_GtestBenchmark{
						GtestBenchmark: &pinpoint_proto.GTestBenchmark{
							Benchmark:   "some-benchmark",
							Measurement: "some-metric",
							Test:        "some-test"}}}
				proto.Merge(gtestJob, job)
				v, err := JobToValues(gtestJob, "user@example.com")
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
					"patch": "https://some-gerrit-host/c/12345/1"})
			})

		})

		Convey("Creating a Functional Comparison", func() {
			job.ComparisonMode = pinpoint_proto.JobSpec_FUNCTIONAL

			Convey("We support Telemetry specifying a story", func() {
				telemetryJob :=
					&pinpoint_proto.JobSpec{
						ComparisonMagnitude: 0.2,
						Arguments: &pinpoint_proto.JobSpec_TelemetryBenchmark{
							TelemetryBenchmark: &pinpoint_proto.TelemetryBenchmark{
								Benchmark: "some-benchmark",
								StorySelection: &pinpoint_proto.TelemetryBenchmark_Story{
									Story: "some-story"},
								Measurement:   "some-metric",
								GroupingLabel: "some-grouping-label",
								Statistic:     pinpoint_proto.TelemetryBenchmark_NONE}}}
				proto.Merge(telemetryJob, job)
				v, err := JobToValues(telemetryJob, "user@example.com")
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
					"patch": "https://some-gerrit-host/c/12345/1"})

			})

			Convey("We support Telemetry specifying story tags", func() {
				telemetryJob :=
					&pinpoint_proto.JobSpec{
						ComparisonMagnitude: 0.2,
						Arguments: &pinpoint_proto.JobSpec_TelemetryBenchmark{
							TelemetryBenchmark: &pinpoint_proto.TelemetryBenchmark{
								Benchmark: "some-benchmark",
								StorySelection: &pinpoint_proto.TelemetryBenchmark_StoryTags{
									StoryTags: &pinpoint_proto.TelemetryBenchmark_StoryTagList{
										StoryTags: []string{"some-tag", "some-other-tag"},
									}},
								Measurement:   "some-metric",
								GroupingLabel: "some-grouping-label",
								Statistic:     pinpoint_proto.TelemetryBenchmark_NONE}}}
				proto.Merge(telemetryJob, job)
				v, err := JobToValues(telemetryJob, "user@example.com")
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
					"patch": "https://some-gerrit-host/c/12345/1"})

			})

			Convey("We support GTest", func() {
				gtestJob := &pinpoint_proto.JobSpec{
					ComparisonMagnitude: 0.2,
					Arguments: &pinpoint_proto.JobSpec_GtestBenchmark{
						GtestBenchmark: &pinpoint_proto.GTestBenchmark{
							Benchmark:   "some-benchmark",
							Measurement: "some-metric",
							Test:        "some-test"}}}
				proto.Merge(gtestJob, job)
				v, err := JobToValues(gtestJob, "user@example.com")
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

	Convey("We fail on experiments with missing inputs", t, func() {
		job.JobKind = &pinpoint_proto.JobSpec_Experiment{
			Experiment: &pinpoint_proto.Experiment{
				BaseCommit: &pinpoint_proto.GitilesCommit{
					Host:    "some-gitiles-host",
					Project: "some-gitiles-project",
					GitHash: "c0dec0de",
				},
				ExperimentPatch: &pinpoint_proto.GerritChange{
					Host:     "some-gerrit-host",
					Project:  "some-gerrit-project",
					Change:   12345,
					Patchset: 0,
				},
			}}
		Convey("Creating a performance mode job", func() {
			job.ComparisonMode = pinpoint_proto.JobSpec_PERFORMANCE
			telemetryJob :=
				&pinpoint_proto.JobSpec{
					Arguments: &pinpoint_proto.JobSpec_TelemetryBenchmark{
						TelemetryBenchmark: &pinpoint_proto.TelemetryBenchmark{
							Benchmark: "some-benchmark",
							StorySelection: &pinpoint_proto.TelemetryBenchmark_Story{
								Story: "some-story",
							},
							Measurement:   "some-metric",
							GroupingLabel: "some-grouping-label",
							Statistic:     pinpoint_proto.TelemetryBenchmark_NONE}}}
			proto.Merge(telemetryJob, job)
			Convey("No base commit", func() {
				telemetryJob.GetExperiment().BaseCommit = nil
				_, err := JobToValues(telemetryJob, "user@example.com")
				So(err, ShouldBeError)
			})
			Convey("No user configuration", func() {
				telemetryJob.Config = ""
				_, err := JobToValues(telemetryJob, "user@example.com")
				So(err, ShouldBeError)
			})
			Convey("No target", func() {
				telemetryJob.Target = ""
				_, err := JobToValues(telemetryJob, "user@example.com")
				So(err, ShouldBeError)
			})
		})
	})

	Convey("We support experiments with base commit and experiment patch", t, func() {
		job.JobKind = &pinpoint_proto.JobSpec_Experiment{
			Experiment: &pinpoint_proto.Experiment{
				BaseCommit: &pinpoint_proto.GitilesCommit{
					Host:    "some-gitiles-host",
					Project: "some-gitiles-project",
					GitHash: "c0dec0de",
				},
				ExperimentPatch: &pinpoint_proto.GerritChange{
					Host:     "some-gerrit-host",
					Project:  "some-gerrit-project",
					Change:   23456,
					Patchset: 1,
				}}}

		Convey("Creating a Performance mode job", func() {
			job.ComparisonMode = pinpoint_proto.JobSpec_PERFORMANCE

			Convey("We support Telemetry specifying a story", func() {
				telemetryJob :=
					&pinpoint_proto.JobSpec{
						Arguments: &pinpoint_proto.JobSpec_TelemetryBenchmark{
							TelemetryBenchmark: &pinpoint_proto.TelemetryBenchmark{
								Benchmark: "some-benchmark",
								StorySelection: &pinpoint_proto.TelemetryBenchmark_Story{
									Story: "some-story",
								},
								Measurement:   "some-metric",
								GroupingLabel: "some-grouping-label",
								Statistic:     pinpoint_proto.TelemetryBenchmark_NONE}}}
				proto.Merge(telemetryJob, job)
				v, err := JobToValues(telemetryJob, "user@example.com")
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
					"base_git_hash":    "c0dec0de",
					"experiment_patch": "https://some-gerrit-host/c/23456/1"})

			})

			Convey("We support having both the base commit and experiment commit", func() {
				job.GetExperiment().ExperimentCommit = &pinpoint_proto.GitilesCommit{
					Host:    "some-gitiles-host",
					Project: "some-gitiles-project",
					GitHash: "60061ec0de",
				}
				telemetryJob := &pinpoint_proto.JobSpec{
					Arguments: &pinpoint_proto.JobSpec_TelemetryBenchmark{
						TelemetryBenchmark: &pinpoint_proto.TelemetryBenchmark{
							Benchmark: "some-benchmark",
							StorySelection: &pinpoint_proto.TelemetryBenchmark_Story{
								Story: "some-story",
							},
							Measurement:   "some-metric",
							GroupingLabel: "some-grouping-label",
							Statistic:     pinpoint_proto.TelemetryBenchmark_NONE}}}
				proto.Merge(telemetryJob, job)
				v, err := JobToValues(telemetryJob, "user@example.com")
				So(err, ShouldBeNil)
				So(v, shouldContainMap, map[string]interface{}{
					"end_git_hash": "60061ec0de",
				})
			})

			Convey("We support Telemetry specifying story tags", func() {
				telemetryJob :=
					&pinpoint_proto.JobSpec{
						Arguments: &pinpoint_proto.JobSpec_TelemetryBenchmark{
							TelemetryBenchmark: &pinpoint_proto.TelemetryBenchmark{
								Benchmark: "some-benchmark",
								StorySelection: &pinpoint_proto.TelemetryBenchmark_StoryTags{
									StoryTags: &pinpoint_proto.TelemetryBenchmark_StoryTagList{
										StoryTags: []string{"some-tag", "some-other-tag"},
									}},
								Measurement:   "some-metric",
								GroupingLabel: "some-grouping-label",
								Statistic:     pinpoint_proto.TelemetryBenchmark_NONE}}}
				proto.Merge(telemetryJob, job)
				v, err := JobToValues(telemetryJob, "user@example.com")
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
					"base_git_hash":    "c0dec0de",
					"experiment_patch": "https://some-gerrit-host/c/23456/1"})

			})

			Convey("We support Telemetry specifying story tags and extra args", func() {
				telemetryJob :=
					&pinpoint_proto.JobSpec{
						Arguments: &pinpoint_proto.JobSpec_TelemetryBenchmark{
							TelemetryBenchmark: &pinpoint_proto.TelemetryBenchmark{
								Benchmark: "some-benchmark",
								StorySelection: &pinpoint_proto.TelemetryBenchmark_StoryTags{
									StoryTags: &pinpoint_proto.TelemetryBenchmark_StoryTagList{
										StoryTags: []string{"some-tag", "some-other-tag"},
									}},
								Measurement:   "some-metric",
								GroupingLabel: "some-grouping-label",
								Statistic:     pinpoint_proto.TelemetryBenchmark_NONE,
								ExtraArgs:     []string{"--browser", "some-browser"},
							}}}
				proto.Merge(telemetryJob, job)
				v, err := JobToValues(telemetryJob, "user@example.com")
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
					"base_git_hash":    "c0dec0de",
					"experiment_patch": "https://some-gerrit-host/c/23456/1",
					"extra_test_args":  `["--browser","some-browser"]`,
				})

			})

			Convey("We support jobs with a Batch ID and Attempts", func() {
				telemetryJob :=
					&pinpoint_proto.JobSpec{
						BatchId:             "defined-job-id",
						InitialAttemptCount: 42,
						Arguments: &pinpoint_proto.JobSpec_TelemetryBenchmark{
							TelemetryBenchmark: &pinpoint_proto.TelemetryBenchmark{
								Benchmark: "some-benchmark",
								StorySelection: &pinpoint_proto.TelemetryBenchmark_StoryTags{
									StoryTags: &pinpoint_proto.TelemetryBenchmark_StoryTagList{
										StoryTags: []string{"some-tag", "some-other-tag"},
									}},
								Measurement:   "some-metric",
								GroupingLabel: "some-grouping-label",
								Statistic:     pinpoint_proto.TelemetryBenchmark_NONE,
								ExtraArgs:     []string{"--browser", "some-browser"},
							}}}
				proto.Merge(telemetryJob, job)
				v, err := JobToValues(telemetryJob, "user@example.com")
				So(err, ShouldBeNil)

				// Check that we support the required fields for all Pinpoint jobs.
				So(v, shouldContainMap, map[string]interface{}{
					"target":        "some-build-target",
					"configuration": "some-config",
					"batch_id":      "defined-job-id",
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
					"base_git_hash":    "c0dec0de",
					"experiment_patch": "https://some-gerrit-host/c/23456/1",
					"extra_test_args":  `["--browser","some-browser"]`,
				})

				So(v, shouldContainMap, map[string]interface{}{
					"initial_attempt_count": "42",
				})
			})

			Convey("We support GTest", func() {
				gtestJob := &pinpoint_proto.JobSpec{
					Arguments: &pinpoint_proto.JobSpec_GtestBenchmark{
						GtestBenchmark: &pinpoint_proto.GTestBenchmark{
							Benchmark:   "some-benchmark",
							Measurement: "some-metric",
							Test:        "some-test"}}}
				proto.Merge(gtestJob, job)
				v, err := JobToValues(gtestJob, "user@example.com")
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
					"base_git_hash":    "c0dec0de",
					"experiment_patch": "https://some-gerrit-host/c/23456/1"})
			})

		})

		Convey("Creating a Functional mode job", func() {
			job.ComparisonMode = pinpoint_proto.JobSpec_FUNCTIONAL

			Convey("Fails for Telemetry (unsupported)", func() {
				telemetryJob :=
					&pinpoint_proto.JobSpec{
						Arguments: &pinpoint_proto.JobSpec_TelemetryBenchmark{
							TelemetryBenchmark: &pinpoint_proto.TelemetryBenchmark{
								Benchmark: "some-benchmark",
								StorySelection: &pinpoint_proto.TelemetryBenchmark_Story{
									Story: "some-story"},
								Measurement:   "some-metric",
								GroupingLabel: "some-grouping-label",
								Statistic:     pinpoint_proto.TelemetryBenchmark_NONE}}}
				proto.Merge(telemetryJob, job)
				_, err := JobToValues(telemetryJob, "user@example.com")
				So(err, ShouldNotBeNil)
				So(fmt.Sprintf("%v", err), ShouldContainSubstring, "functional experiments not supported")
			})

			Convey("Fails for GTest (unsupported)", func() {
				gtestJob := &pinpoint_proto.JobSpec{
					Arguments: &pinpoint_proto.JobSpec_GtestBenchmark{
						GtestBenchmark: &pinpoint_proto.GTestBenchmark{
							Benchmark:   "some-benchmark",
							Measurement: "some-metric",
							Test:        "some-test"}}}
				proto.Merge(gtestJob, job)
				_, err := JobToValues(gtestJob, "user@example.com")
				So(err, ShouldNotBeNil)
				So(fmt.Sprintf("%v", err), ShouldContainSubstring, "functional experiments not supported")
			})
		})

	})

}

func TestGerritChangeToURL(t *testing.T) {
	t.Parallel()
	Convey("Given valid GerritChange", t, func() {
		c := &pinpoint_proto.GerritChange{
			Host:    "host",
			Project: "project",
			Change:  123456,
		}
		Convey("When the patchset is provided", func() {
			c.Patchset = 1
			Convey("Then we see the patchset in the URL", func() {
				u, err := gerritChangeToURL(c)
				So(err, ShouldBeNil)
				So(u, ShouldEqual, "https://host/c/123456/1")
			})
		})
		Convey("When the patset is not provided", func() {
			Convey("Then we see no patchset in the URL", func() {
				u, err := gerritChangeToURL(c)
				So(err, ShouldBeNil)
				So(u, ShouldNotEndWith, "/1")
				So(u, ShouldEqual, "https://host/c/123456")
			})
		})
	})
	Convey("Given an invalidly GerritChange", t, func() {
		c := &pinpoint_proto.GerritChange{
			Host:     "host",
			Project:  "project",
			Change:   123456,
			Patchset: 7,
		}

		Convey("When it is missing a host", func() {
			c.Host = ""
			Convey("Then conversion fails", func() {
				_, err := gerritChangeToURL(c)
				So(err, ShouldBeError)
				So(err.Error(), ShouldContainSubstring, "host")
			})
		})
		Convey("When it is missing a change", func() {
			c.Change = 0
			Convey("Then conversion fails", func() {
				_, err := gerritChangeToURL(c)
				So(err, ShouldBeError)
				So(err.Error(), ShouldContainSubstring, "change")
			})
		})
	})
}

func TestJobToProto(t *testing.T) {
	t.Parallel()
	Convey("Given a defined experiment", t, func() {
		lj, err := os.ReadFile("../testdata/defined-job-experiment.json")
		So(err, ShouldBeNil)
		Convey("When we convert the legacy JSON", func() {
			p, err := JobToProto(strings.NewReader(string(lj)))
			So(err, ShouldBeNil)
			So(p, ShouldNotBeNil)
			Convey("Then we find the experiment URLs", func() {
				results := p.GetAbExperimentResults()
				So(results, ShouldNotBeNil)
				So(results.AChangeResult.Attempts, ShouldHaveLength, 10)
				So(results.BChangeResult.Attempts, ShouldHaveLength, 10)

				// These are typical 3 steps for a legacy job
				quests := []string{"Build", "Test", "Get values"}

				// We know that legacy jobs have 2-3 executions per attempt. This corresponds with the Build, Test,
				// Value quest executions, which is defined for most Pinpoint A/B experiments.
				for _, a := range results.AChangeResult.Attempts {
					So(len(a.Executions), ShouldBeBetweenOrEqual, 2, 3)
					for i, e := range a.Executions {
						So(e.Label, ShouldEqual, quests[i])
					}
				}
				for _, a := range results.BChangeResult.Attempts {
					So(len(a.Executions), ShouldBeBetweenOrEqual, 2, 3)
					for i, e := range a.Executions {
						So(e.Label, ShouldEqual, quests[i])
					}
				}
			})
		})
	})
	Convey("Given an experiment with a batch id", t, func() {
		lj, err := os.ReadFile("../testdata/defined-job-experiment-with-batch-id.json")
		So(err, ShouldBeNil)
		Convey("When we convert the legacy JSON", func() {
			p, err := JobToProto(strings.NewReader(string(lj)))
			So(err, ShouldBeNil)
			So(p, ShouldNotBeNil)
			Convey("Then we find the batch ID", func() {
				So(p.JobSpec.BatchId, ShouldEqual, "batch-id-in-file")
			})
		})
	})
}
