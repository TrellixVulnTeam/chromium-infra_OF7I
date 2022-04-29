// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package testvariantupdator

import (
	"context"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"go.chromium.org/luci/gae/impl/memory"
	"go.chromium.org/luci/resultdb/pbutil"
	"go.chromium.org/luci/server/span"
	"go.chromium.org/luci/server/tq"
	"google.golang.org/protobuf/types/known/durationpb"

	"infra/appengine/weetbix/internal"
	"infra/appengine/weetbix/internal/analyzedtestvariants"
	"infra/appengine/weetbix/internal/config"
	configpb "infra/appengine/weetbix/internal/config/proto"
	"infra/appengine/weetbix/internal/tasks/taskspb"
	"infra/appengine/weetbix/internal/testutil"
	"infra/appengine/weetbix/internal/testutil/insert"
	pb "infra/appengine/weetbix/proto/v1"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/luci/common/clock"
	. "go.chromium.org/luci/common/testing/assertions"
)

func init() {
	RegisterTaskClass()
}

func TestSchedule(t *testing.T) {
	Convey(`TestSchedule`, t, func() {
		ctx, skdr := tq.TestingContext(testutil.SpannerTestContext(t), nil)

		realm := "chromium:ci"
		testID := "ninja://test"
		variantHash := "deadbeef"
		now := clock.Now(ctx)
		task := &taskspb.UpdateTestVariant{
			TestVariantKey: &taskspb.TestVariantKey{
				Realm:       realm,
				TestId:      testID,
				VariantHash: variantHash,
			},
			EnqueueTime: pbutil.MustTimestampProto(now),
		}
		_, err := span.ReadWriteTransaction(ctx, func(ctx context.Context) error {
			Schedule(ctx, realm, testID, variantHash, durationpb.New(time.Hour), now)
			return nil
		})
		So(err, ShouldBeNil)
		So(skdr.Tasks().Payloads()[0], ShouldResembleProto, task)
	})
}

func TestCheckTask(t *testing.T) {
	Convey(`checkTask`, t, func() {
		ctx := testutil.SpannerTestContext(t)
		realm := "chromium:ci"
		tID := "ninja://test"
		vh := "varianthash"
		now := clock.Now(ctx)
		ms := []*spanner.Mutation{
			insert.AnalyzedTestVariant(realm, tID, vh, pb.AnalyzedTestVariantStatus_CONSISTENTLY_EXPECTED,
				map[string]interface{}{
					"NextUpdateTaskEnqueueTime": now,
				}),
			insert.AnalyzedTestVariant(realm, "anothertest", vh, pb.AnalyzedTestVariantStatus_CONSISTENTLY_EXPECTED, nil),
		}
		testutil.MustApply(ctx, ms...)

		task := &taskspb.UpdateTestVariant{
			TestVariantKey: &taskspb.TestVariantKey{
				Realm:       realm,
				TestId:      tID,
				VariantHash: vh,
			},
		}
		Convey(`match`, func() {
			task.EnqueueTime = pbutil.MustTimestampProto(now)
			_, err := checkTask(span.Single(ctx), task)
			So(err, ShouldBeNil)
		})

		Convey(`mismatch`, func() {
			anotherTime := now.Add(time.Hour)
			task.EnqueueTime = pbutil.MustTimestampProto(anotherTime)
			_, err := checkTask(span.Single(ctx), task)
			So(err, ShouldEqual, errUnknownTask)
		})

		Convey(`no schedule`, func() {
			task.TestVariantKey.TestId = "anothertest"
			_, err := checkTask(span.Single(ctx), task)
			So(err, ShouldEqual, errShouldNotSchedule)
		})
	})
}

func createProjectsConfig() map[string]*configpb.ProjectConfig {
	return map[string]*configpb.ProjectConfig{
		"chromium": {
			Realms: []*configpb.RealmConfig{
				{
					Name: "ci",
					TestVariantAnalysis: &configpb.TestVariantAnalysisConfig{
						UpdateTestVariantTask: &configpb.UpdateTestVariantTask{
							UpdateTestVariantTaskInterval:   durationpb.New(time.Hour),
							TestVariantStatusUpdateDuration: durationpb.New(24 * time.Hour),
						},
					},
				},
			},
		},
	}
}

func TestUpdateTestVariantStatus(t *testing.T) {
	Convey(`updateTestVariant`, t, func() {
		ctx, skdr := tq.TestingContext(testutil.SpannerTestContext(t), nil)
		ctx = memory.Use(ctx)
		config.SetTestProjectConfig(ctx, createProjectsConfig())
		realm := "chromium:ci"
		vh := "varianthash"
		now := clock.Now(ctx).UTC()
		tID1 := "ninja://test1"
		tID2 := "ninja://test2"
		statuses := []pb.AnalyzedTestVariantStatus{
			pb.AnalyzedTestVariantStatus_FLAKY,
			pb.AnalyzedTestVariantStatus_CONSISTENTLY_UNEXPECTED,
		}
		times := []time.Time{
			now.Add(-24 * time.Hour),
			now.Add(-240 * time.Hour),
		}
		ms := []*spanner.Mutation{
			insert.AnalyzedTestVariant(realm, tID1, vh, pb.AnalyzedTestVariantStatus_CONSISTENTLY_EXPECTED, map[string]interface{}{
				"NextUpdateTaskEnqueueTime": now,
				"StatusUpdateTime":          now,
			}),
			insert.AnalyzedTestVariant(realm, tID2, vh, pb.AnalyzedTestVariantStatus_CONSISTENTLY_EXPECTED, map[string]interface{}{
				"NextUpdateTaskEnqueueTime": now,
				"StatusUpdateTime":          now,
				"PreviousStatuses":          statuses,
				"PreviousStatusUpdateTimes": times,
			}),
			insert.Verdict(realm, tID1, vh, "build-1", internal.VerdictStatus_VERDICT_FLAKY, now.Add(-2*time.Hour), nil),
			insert.Verdict(realm, tID2, vh, "build-1", internal.VerdictStatus_VERDICT_FLAKY, now.Add(-2*time.Hour), nil),
		}
		testutil.MustApply(ctx, ms...)

		test := func(tID string, pStatuses []pb.AnalyzedTestVariantStatus, pUpdateTimes []time.Time, i int) {
			statusHistory, enqTime, err := analyzedtestvariants.ReadStatusHistory(span.Single(ctx), spanner.Key{realm, tID, vh})
			So(err, ShouldBeNil)
			statusUpdateTime := statusHistory.StatusUpdateTime

			task := &taskspb.UpdateTestVariant{
				TestVariantKey: &taskspb.TestVariantKey{
					Realm:       realm,
					TestId:      tID,
					VariantHash: vh,
				},
				EnqueueTime: pbutil.MustTimestampProto(now),
			}
			err = updateTestVariant(ctx, task)
			So(err, ShouldBeNil)

			// Read the test variant to confirm the updates.
			statusHistory, enqTime, err = analyzedtestvariants.ReadStatusHistory(span.Single(ctx), spanner.Key{realm, tID, vh})
			So(err, ShouldBeNil)
			So(statusHistory.Status, ShouldEqual, pb.AnalyzedTestVariantStatus_FLAKY)
			So(statusHistory.PreviousStatuses, ShouldResemble, append([]pb.AnalyzedTestVariantStatus{pb.AnalyzedTestVariantStatus_CONSISTENTLY_EXPECTED}, pStatuses...))
			So(statusHistory.PreviousStatusUpdateTimes, ShouldResemble, append([]time.Time{statusUpdateTime}, pUpdateTimes...))

			nextTask := skdr.Tasks().Payloads()[i].(*taskspb.UpdateTestVariant)
			So(pbutil.MustTimestamp(nextTask.EnqueueTime), ShouldEqual, enqTime.Time)
		}

		test(tID1, []pb.AnalyzedTestVariantStatus{}, []time.Time{}, 0)
		test(tID2, statuses, times, 1)
	})
}
