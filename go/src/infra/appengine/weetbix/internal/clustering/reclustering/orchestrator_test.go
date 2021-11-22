// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package reclustering

import (
	"context"
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"go.chromium.org/luci/gae/impl/memory"
	"go.chromium.org/luci/server/span"
	"go.chromium.org/luci/server/tq"
	"go.chromium.org/luci/server/tq/tqtesting"
	_ "go.chromium.org/luci/server/tq/txn/spanner"

	"infra/appengine/weetbix/internal/clustering/algorithms"
	"infra/appengine/weetbix/internal/clustering/rules"
	"infra/appengine/weetbix/internal/clustering/runs"
	"infra/appengine/weetbix/internal/clustering/state"
	"infra/appengine/weetbix/internal/config"
	"infra/appengine/weetbix/internal/services/reclustering"
	"infra/appengine/weetbix/internal/tasks/taskspb"
	"infra/appengine/weetbix/internal/testutil"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/luci/common/clock/testclock"
	. "go.chromium.org/luci/common/testing/assertions"
)

func TestOrchestrator(t *testing.T) {
	reclustering.RegisterTaskClass()

	Convey(`With Spanner Test Database`, t, func() {
		ctx := testutil.SpannerTestContext(t)

		// Simulate the Orchestrator job running one second past the hour.
		startTime := testclock.TestRecentTimeUTC.Truncate(time.Hour).Add(time.Second)
		ctx, tc := testclock.UseTime(ctx, startTime)

		ctx = memory.Use(ctx) // For config cache.
		ctx, skdr := tq.TestingContext(ctx, nil)

		cfg := &config.Config{
			ReclusteringWorkers:         4,
			ReclusteringIntervalMinutes: 5,
		}
		config.SetTestConfig(ctx, cfg)

		testProjects := []string{"project-a", "project-b"}

		testOrchestratorDoesNothing := func() {
			beforeTasks := tasks(skdr)
			beforeRuns := readRuns(ctx, testProjects)

			err := CronHandler(ctx)
			So(err, ShouldBeNil)

			afterTasks := tasks(skdr)
			afterRuns := readRuns(ctx, testProjects)
			So(afterTasks, ShouldResembleProto, beforeTasks)
			So(afterRuns, ShouldResemble, beforeRuns)
		}

		Convey("Without Projects", func() {
			projectCfg := make(map[string]*config.ProjectConfig)
			config.SetTestProjectConfig(ctx, projectCfg)

			testOrchestratorDoesNothing()
		})
		Convey("With Projects", func() {
			// Orchestrator only looks at the projects that have config, not the config itself.
			projectCfg := make(map[string]*config.ProjectConfig)
			projectCfg["project-a"] = &config.ProjectConfig{}
			projectCfg["project-b"] = &config.ProjectConfig{}
			config.SetTestProjectConfig(ctx, projectCfg)

			// Create chunks in project-b. After this, the row estimates
			// for the projects should be:
			// project-a: ~100
			// project-b: ~450
			var entries []*state.Entry
			for i := 0; i < 450; i++ {
				entries = append(entries, state.NewEntry(i).WithProject("project-b").Build())
			}
			_, err := state.CreateEntriesForTesting(ctx, entries)
			So(err, ShouldBeNil)

			rulesVersionB := time.Date(2020, time.January, 10, 9, 8, 7, 0, time.UTC)
			rule := rules.NewRule(1).WithProject("project-b").WithLastUpdated(rulesVersionB).Build()
			err = rules.SetRulesForTesting(ctx, []*rules.FailureAssociationRule{rule})
			So(err, ShouldBeNil)

			expectedAttemptTime := tc.Now().Truncate(5 * time.Minute).Add(5 * time.Minute)
			expectedTasks := []*taskspb.ReclusterChunks{
				{
					Project:      "project-a",
					AttemptTime:  timestamppb.New(expectedAttemptTime),
					StartChunkId: "",
					EndChunkId:   state.EndOfTable,
				},
				{
					Project:      "project-b",
					AttemptTime:  timestamppb.New(expectedAttemptTime),
					StartChunkId: "",
					EndChunkId:   strings.Repeat("55", 15) + "54",
				},
				{
					Project:      "project-b",
					AttemptTime:  timestamppb.New(expectedAttemptTime),
					StartChunkId: strings.Repeat("55", 15) + "54",
					EndChunkId:   strings.Repeat("aa", 15) + "a9",
				},
				{
					Project:      "project-b",
					AttemptTime:  timestamppb.New(expectedAttemptTime),
					StartChunkId: strings.Repeat("aa", 15) + "a9",
					EndChunkId:   state.EndOfTable,
				},
			}
			expectedRunA := &runs.ReclusteringRun{
				Project:           "project-a",
				AttemptTimestamp:  expectedAttemptTime,
				AlgorithmsVersion: algorithms.AlgorithmsVersion,
				RulesVersion:      rules.StartingEpoch,
				ShardCount:        1,
				ShardsReported:    0,
				Progress:          0,
			}
			expectedRunB := &runs.ReclusteringRun{
				Project:           "project-b",
				AttemptTimestamp:  expectedAttemptTime,
				AlgorithmsVersion: algorithms.AlgorithmsVersion,
				RulesVersion:      rulesVersionB,
				ShardCount:        3,
				ShardsReported:    0,
				Progress:          0,
			}
			expectedRuns := make(map[string]*runs.ReclusteringRun)
			expectedRuns["project-a"] = expectedRunA
			expectedRuns["project-b"] = expectedRunB

			Convey("Disabled orchestrator does nothing", func() {
				Convey("Workers is zero", func() {
					cfg.ReclusteringWorkers = 0
					config.SetTestConfig(ctx, cfg)

					testOrchestratorDoesNothing()
				})
				Convey("Interval Minutes is zero", func() {
					cfg.ReclusteringIntervalMinutes = 0
					config.SetTestConfig(ctx, cfg)

					testOrchestratorDoesNothing()
				})
			})
			Convey("Schedules successfully without existing runs", func() {
				err := CronHandler(ctx)
				So(err, ShouldBeNil)

				actualTasks := tasks(skdr)
				So(actualTasks, ShouldResembleProto, expectedTasks)

				actualRuns := readRuns(ctx, testProjects)
				So(actualRuns, ShouldResemble, expectedRuns)
			})
			Convey("Schedules successfully with an existing complete run", func() {
				runB := &runs.ReclusteringRun{
					Project: "project-b",
					// So as not to overlap with the run that should be created.
					AttemptTimestamp:  expectedAttemptTime.Add(-5 * time.Minute),
					AlgorithmsVersion: 1,
					RulesVersion:      rulesVersionB.Add(-1 * time.Hour),
					ShardCount:        10,
					ShardsReported:    10,
					Progress:          10 * 1000,
				}
				err := runs.SetRunsForTesting(ctx, []*runs.ReclusteringRun{runB})
				So(err, ShouldBeNil)

				// A run scheduled after an existing complete run should
				// use the latest algorithms and rules available. So our
				// expectations are unchanged.
				err = CronHandler(ctx)
				So(err, ShouldBeNil)

				actualTasks := tasks(skdr)
				So(actualTasks, ShouldResembleProto, expectedTasks)

				actualRuns := readRuns(ctx, testProjects)
				So(actualRuns, ShouldResemble, expectedRuns)
			})
			Convey("Schedules successfully with an existing incomplete run", func() {
				runB := &runs.ReclusteringRun{
					Project: "project-b",
					// So as not to overlap with the run that should be created.
					AttemptTimestamp:  expectedAttemptTime.Add(-5 * time.Minute),
					AlgorithmsVersion: 1,
					RulesVersion:      rulesVersionB.Add(-1 * time.Hour),
					ShardCount:        10,
					ShardsReported:    10,
					Progress:          500,
				}
				err := runs.SetRunsForTesting(ctx, []*runs.ReclusteringRun{runB})
				So(err, ShouldBeNil)

				// Expect the same algorithms and rules version to be used as
				// the previous run, to ensure forward progress (if new rules
				// are being constantly created, we don't want to be
				// reclustering only the beginning of the workers' keyspaces).
				expectedRunB.AlgorithmsVersion = runB.AlgorithmsVersion
				expectedRunB.RulesVersion = runB.RulesVersion

				err = CronHandler(ctx)
				So(err, ShouldBeNil)

				actualTasks := tasks(skdr)
				So(actualTasks, ShouldResembleProto, expectedTasks)

				actualRuns := readRuns(ctx, testProjects)
				So(actualRuns, ShouldResemble, expectedRuns)
			})
			Convey("Does not schedule with an overlapping run", func() {
				// This can occur if the reclustering interval changes.
				runA := &runs.ReclusteringRun{
					Project: "project-a",
					// So as to overlap with the run that should be created.
					AttemptTimestamp:  expectedAttemptTime.Add(-1 * time.Minute),
					AlgorithmsVersion: 1,
					RulesVersion:      rulesVersionB.Add(-1 * time.Hour),
					ShardCount:        1,
					ShardsReported:    1,
					Progress:          500,
				}
				err := runs.SetRunsForTesting(ctx, []*runs.ReclusteringRun{runA})
				So(err, ShouldBeNil)

				// Expect only project-b to have been scheduled.
				expectedRuns["project-a"] = runA
				expectedTasks = expectedTasks[1:]

				err = CronHandler(ctx)
				So(err, ShouldErrLike, "an attempt which overlaps the proposed attempt already exists")

				actualTasks := tasks(skdr)
				So(actualTasks, ShouldResembleProto, expectedTasks)

				actualRuns := readRuns(ctx, testProjects)
				So(actualRuns, ShouldResemble, expectedRuns)
			})
			Convey("Does not schedule on off-interval minutes", func() {
				// The test uses a 5-minute re-clustering interval, so
				// minutes modulo 1, 2, 3 and 4 should not have runs start.
				for i := 0; i < 4; i++ {
					tc.Add(time.Minute)
					testOrchestratorDoesNothing()
				}

				tc.Add(time.Minute)
				err := CronHandler(ctx)
				So(err, ShouldBeNil)

				// Because we ran on the next scheduling interval five minutes later,
				// expect all tasks and runs to be shifted five minutes back.
				expectedAttemptTime = expectedAttemptTime.Add(5 * time.Minute)
				for _, task := range expectedTasks {
					task.AttemptTime = timestamppb.New(expectedAttemptTime)
				}
				expectedRunA.AttemptTimestamp = expectedAttemptTime
				expectedRunB.AttemptTimestamp = expectedAttemptTime

				actualTasks := tasks(skdr)
				So(actualTasks, ShouldResembleProto, expectedTasks)

				actualRuns := readRuns(ctx, testProjects)
				So(actualRuns, ShouldResemble, expectedRuns)
			})
		})
	})
}

func tasks(s *tqtesting.Scheduler) []*taskspb.ReclusterChunks {
	var tasks []*taskspb.ReclusterChunks
	for _, pl := range s.Tasks().Payloads() {
		task := pl.(*taskspb.ReclusterChunks)
		tasks = append(tasks, task)
	}
	return tasks
}

func readRuns(ctx context.Context, projects []string) map[string]*runs.ReclusteringRun {
	txn, cancel := span.ReadOnlyTransaction(ctx)
	defer cancel()

	result := make(map[string]*runs.ReclusteringRun)
	for _, project := range projects {
		run, err := runs.ReadLast(txn, project)
		So(err, ShouldBeNil)
		result[project] = run
	}
	return result
}
