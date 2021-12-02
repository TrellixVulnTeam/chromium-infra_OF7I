// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package runs

import (
	"context"
	"testing"
	"time"

	"go.chromium.org/luci/server/span"

	"infra/appengine/weetbix/internal/clustering/algorithms"
	"infra/appengine/weetbix/internal/clustering/rules"
	"infra/appengine/weetbix/internal/testutil"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func TestSpan(t *testing.T) {
	Convey(`With Spanner Test Database`, t, func() {
		ctx := testutil.SpannerTestContext(t)

		Convey(`Reads`, func() {
			reference := time.Date(2020, time.January, 1, 1, 0, 0, 0, time.UTC)
			runs := []*ReclusteringRun{
				NewRun(0).WithProject("otherproject").WithAttemptTimestamp(reference).WithCompletedProgress().Build(),
				NewRun(1).WithAttemptTimestamp(reference.Add(-5 * time.Minute)).Build(),
				NewRun(2).WithAttemptTimestamp(reference.Add(-10 * time.Minute)).Build(),
				NewRun(3).WithAttemptTimestamp(reference.Add(-20 * time.Minute)).WithReportedProgress(500).Build(),
				NewRun(4).WithAttemptTimestamp(reference.Add(-30 * time.Minute)).WithReportedProgress(500).Build(),
				NewRun(5).WithAttemptTimestamp(reference.Add(-40 * time.Minute)).WithCompletedProgress().Build(),
				NewRun(6).WithAttemptTimestamp(reference.Add(-50 * time.Minute)).WithCompletedProgress().Build(),
			}
			err := SetRunsForTesting(ctx, runs)
			So(err, ShouldBeNil)

			// For ReadLast... methods, this is the fake row that is expected
			// to be returned if no row exists.
			expectedFake := &ReclusteringRun{
				Project:           "emptyproject",
				AttemptTimestamp:  time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC),
				AlgorithmsVersion: 1,
				RulesVersion:      rules.StartingEpoch,
				ShardCount:        1,
				ShardsReported:    1,
				Progress:          1000,
			}

			Convey(`Read`, func() {
				Convey(`Not Exists`, func() {
					run, err := Read(span.Single(ctx), testProject, reference)
					So(err, ShouldEqual, NotFound)
					So(run, ShouldBeNil)
				})
				Convey(`Exists`, func() {
					run, err := Read(span.Single(ctx), testProject, runs[2].AttemptTimestamp)
					So(err, ShouldBeNil)
					So(run, ShouldResemble, runs[2])
				})
			})
			Convey(`ReadLast`, func() {
				Convey(`Not Exists`, func() {
					run, err := ReadLast(span.Single(ctx), "emptyproject")
					So(err, ShouldBeNil)
					So(run, ShouldResemble, expectedFake)
				})
				Convey(`Exists`, func() {
					run, err := ReadLast(span.Single(ctx), testProject)
					So(err, ShouldBeNil)
					So(run, ShouldResemble, runs[1])
				})
			})
			Convey(`ReadLastWithProgress`, func() {
				Convey(`Not Exists`, func() {
					run, err := ReadLastWithProgress(span.Single(ctx), "emptyproject")
					So(err, ShouldBeNil)
					So(run, ShouldResemble, expectedFake)
				})
				Convey(`Exists`, func() {
					run, err := ReadLastWithProgress(span.Single(ctx), testProject)
					So(err, ShouldBeNil)
					So(run, ShouldResemble, runs[3])
				})
			})
			Convey(`ReadLastComplete`, func() {
				Convey(`Not Exists`, func() {
					run, err := ReadLastComplete(span.Single(ctx), "emptyproject")
					So(err, ShouldBeNil)
					So(run, ShouldResemble, expectedFake)
				})
				Convey(`Exists`, func() {
					run, err := ReadLastComplete(span.Single(ctx), testProject)
					So(err, ShouldBeNil)
					So(run, ShouldResemble, runs[5])
				})
			})
		})
		Convey(`Query Progress`, func() {
			Convey(`Rule Progress`, func() {
				rulesVersion := time.Date(2021, time.January, 1, 1, 0, 0, 0, time.UTC)

				reference := time.Date(2020, time.January, 1, 1, 0, 0, 0, time.UTC)
				runs := []*ReclusteringRun{
					NewRun(0).WithAttemptTimestamp(reference.Add(-5 * time.Minute)).WithRulesVersion(rulesVersion).WithNoReportedProgress().Build(),
					NewRun(1).WithAttemptTimestamp(reference.Add(-10 * time.Minute)).WithRulesVersion(rulesVersion).WithReportedProgress(500).Build(),
					NewRun(2).WithAttemptTimestamp(reference.Add(-20 * time.Minute)).WithRulesVersion(rulesVersion.Add(-1 * time.Hour)).WithCompletedProgress().Build(),
				}
				err := SetRunsForTesting(ctx, runs)
				So(err, ShouldBeNil)

				progress, err := ReadReclusteringProgress(ctx, testProject)
				So(err, ShouldBeNil)

				So(progress.IncorporatesRulesVersion(rulesVersion.Add(1*time.Hour)), ShouldBeFalse)
				So(progress.IncorporatesRulesVersion(rulesVersion), ShouldBeFalse)
				So(progress.IncorporatesRulesVersion(rulesVersion.Add(-1*time.Minute)), ShouldBeFalse)
				So(progress.IncorporatesRulesVersion(rulesVersion.Add(-1*time.Hour)), ShouldBeTrue)
				So(progress.IncorporatesRulesVersion(rulesVersion.Add(-2*time.Hour)), ShouldBeTrue)
			})
			Convey(`Algorithms Upgrading (started)`, func() {
				reference := time.Date(2020, time.January, 1, 1, 0, 0, 0, time.UTC)
				runs := []*ReclusteringRun{
					NewRun(0).WithAttemptTimestamp(reference.Add(-5 * time.Minute)).WithAlgorithmsVersion(algorithms.AlgorithmsVersion + 1).WithNoReportedProgress().Build(),
					NewRun(1).WithAttemptTimestamp(reference.Add(-10 * time.Minute)).WithAlgorithmsVersion(algorithms.AlgorithmsVersion + 1).WithReportedProgress(500).Build(),
					NewRun(2).WithAttemptTimestamp(reference.Add(-20 * time.Minute)).WithAlgorithmsVersion(algorithms.AlgorithmsVersion).WithCompletedProgress().Build(),
				}
				err := SetRunsForTesting(ctx, runs)
				So(err, ShouldBeNil)

				progress, err := ReadReclusteringProgress(ctx, testProject)
				So(err, ShouldBeNil)

				So(progress.LatestAlgorithmsVersion, ShouldEqual, algorithms.AlgorithmsVersion+1)
				So(progress.IncorporatesLatestAlgorithms(), ShouldBeFalse)
			})
			Convey(`Algorithms Upgrading (not yet started)`, func() {
				reference := time.Date(2020, time.January, 1, 1, 0, 0, 0, time.UTC)
				runs := []*ReclusteringRun{
					NewRun(0).WithAttemptTimestamp(reference.Add(-5 * time.Minute)).WithAlgorithmsVersion(algorithms.AlgorithmsVersion - 1).WithNoReportedProgress().Build(),
					NewRun(1).WithAttemptTimestamp(reference.Add(-10 * time.Minute)).WithAlgorithmsVersion(algorithms.AlgorithmsVersion - 1).WithReportedProgress(500).Build(),
					NewRun(2).WithAttemptTimestamp(reference.Add(-20 * time.Minute)).WithAlgorithmsVersion(algorithms.AlgorithmsVersion - 1).WithCompletedProgress().Build(),
				}
				err := SetRunsForTesting(ctx, runs)
				So(err, ShouldBeNil)

				progress, err := ReadReclusteringProgress(ctx, testProject)
				So(err, ShouldBeNil)

				So(progress.LatestAlgorithmsVersion, ShouldEqual, algorithms.AlgorithmsVersion)
				So(progress.IncorporatesLatestAlgorithms(), ShouldBeFalse)
			})
			Convey(`Algorithms Stable`, func() {
				reference := time.Date(2020, time.January, 1, 1, 0, 0, 0, time.UTC)
				runs := []*ReclusteringRun{
					NewRun(0).WithAttemptTimestamp(reference.Add(-5 * time.Minute)).WithAlgorithmsVersion(algorithms.AlgorithmsVersion).WithNoReportedProgress().Build(),
					NewRun(1).WithAttemptTimestamp(reference.Add(-10 * time.Minute)).WithAlgorithmsVersion(algorithms.AlgorithmsVersion).WithReportedProgress(500).Build(),
					NewRun(2).WithAttemptTimestamp(reference.Add(-20 * time.Minute)).WithAlgorithmsVersion(algorithms.AlgorithmsVersion).WithCompletedProgress().Build(),
				}
				err := SetRunsForTesting(ctx, runs)
				So(err, ShouldBeNil)

				progress, err := ReadReclusteringProgress(ctx, testProject)
				So(err, ShouldBeNil)

				So(progress.LatestAlgorithmsVersion, ShouldEqual, algorithms.AlgorithmsVersion)
				So(progress.IncorporatesLatestAlgorithms(), ShouldBeTrue)
			})
		})
		Convey(`Reporting Progress`, func() {
			reference := time.Date(2020, time.January, 1, 1, 0, 0, 0, time.UTC)
			assertProgress := func(shardsReported, progress int64) {
				run, err := Read(span.Single(ctx), testProject, reference)
				So(err, ShouldBeNil)
				So(run.ShardsReported, ShouldEqual, shardsReported)
				So(run.Progress, ShouldEqual, progress)
			}

			runs := []*ReclusteringRun{
				NewRun(0).WithAttemptTimestamp(reference).WithShardCount(2).WithNoReportedProgress().Build(),
			}
			err := SetRunsForTesting(ctx, runs)
			So(err, ShouldBeNil)

			token1 := NewProgressToken(testProject, reference)
			token2 := NewProgressToken(testProject, reference)
			assertProgress(0, 0)

			So(token1.ReportProgress(ctx, 0), ShouldBeNil)
			assertProgress(1, 0)

			So(token1.ReportProgress(ctx, 150), ShouldBeNil)
			assertProgress(1, 150)

			So(token2.ReportProgress(ctx, 200), ShouldBeNil)
			assertProgress(2, 350)

			So(token1.ReportProgress(ctx, 200), ShouldBeNil)
			assertProgress(2, 400)

			So(token2.ReportProgress(ctx, 1000), ShouldBeNil)
			assertProgress(2, 1200)

			So(token1.ReportProgress(ctx, 1000), ShouldBeNil)
			assertProgress(2, 2000)
		})
		Convey(`Create`, func() {
			testCreate := func(bc *ReclusteringRun) error {
				_, err := span.ReadWriteTransaction(ctx, func(ctx context.Context) error {
					return Create(ctx, bc)
				})
				return err
			}
			r := NewRun(100).Build()
			Convey(`Valid`, func() {
				testExists := func(expectedRun *ReclusteringRun) {
					txn, cancel := span.ReadOnlyTransaction(ctx)
					defer cancel()
					run, err := Read(txn, expectedRun.Project, expectedRun.AttemptTimestamp)

					So(err, ShouldBeNil)
					So(run, ShouldResemble, expectedRun)
				}

				err := testCreate(r)
				So(err, ShouldBeNil)
				testExists(r)
			})
			Convey(`With invalid Project`, func() {
				Convey(`Missing`, func() {
					r.Project = ""
					err := testCreate(r)
					So(err, ShouldErrLike, "project must be valid")
				})
				Convey(`Invalid`, func() {
					r.Project = "!"
					err := testCreate(r)
					So(err, ShouldErrLike, "project must be valid")
				})
			})
			Convey(`With invalid Attempt Timestamp`, func() {
				r.AttemptTimestamp = time.Time{}
				err := testCreate(r)
				So(err, ShouldErrLike, "attempt timestamp must be set")
			})
			Convey(`With invalid Algorithms Version`, func() {
				r.AlgorithmsVersion = 0
				err := testCreate(r)
				So(err, ShouldErrLike, "algorithms version must be valid")
			})
			Convey(`With invalid Rules Version`, func() {
				r.RulesVersion = time.Time{}
				err := testCreate(r)
				So(err, ShouldErrLike, "rules version must be valid")
			})
			Convey(`With invalid Shard Count`, func() {
				r.ShardCount = 0
				err := testCreate(r)
				So(err, ShouldErrLike, "shard count must be valid")
			})
			Convey(`With invalid Shards Reported`, func() {
				r.ShardsReported = r.ShardCount + 1
				err := testCreate(r)
				So(err, ShouldErrLike, "shards reported must be valid")
			})
			Convey(`With invalid Progress`, func() {
				r.Progress = r.ShardCount*1000 + 1
				err := testCreate(r)
				So(err, ShouldErrLike, "progress must be valid")
			})
		})
	})
}
