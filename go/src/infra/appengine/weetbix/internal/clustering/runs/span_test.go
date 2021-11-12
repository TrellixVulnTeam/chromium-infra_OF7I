// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package runs

import (
	"context"
	"testing"
	"time"

	"go.chromium.org/luci/server/span"

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
				NewRun(3).WithAttemptTimestamp(reference.Add(-20 * time.Minute)).WithReportedProgress().Build(),
				NewRun(4).WithAttemptTimestamp(reference.Add(-30 * time.Minute)).WithReportedProgress().Build(),
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
