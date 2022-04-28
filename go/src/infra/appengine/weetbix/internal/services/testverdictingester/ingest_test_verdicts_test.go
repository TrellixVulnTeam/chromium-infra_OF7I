// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package testverdictingester

import (
	"context"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
	"google.golang.org/genproto/protobuf/field_mask"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	bbpb "go.chromium.org/luci/buildbucket/proto"
	"go.chromium.org/luci/common/clock"
	. "go.chromium.org/luci/common/testing/assertions"
	rdbpbutil "go.chromium.org/luci/resultdb/pbutil"
	rdbpb "go.chromium.org/luci/resultdb/proto/v1"
	"go.chromium.org/luci/server/span"
	"go.chromium.org/luci/server/tq"
	_ "go.chromium.org/luci/server/tq/txn/spanner"

	"infra/appengine/weetbix/internal/buildbucket"
	ctrlpb "infra/appengine/weetbix/internal/ingestion/control/proto"
	"infra/appengine/weetbix/internal/resultdb"
	"infra/appengine/weetbix/internal/tasks/taskspb"
	"infra/appengine/weetbix/internal/testutil"
	"infra/appengine/weetbix/internal/testverdicts"
	"infra/appengine/weetbix/pbutil"
)

func init() {
	RegisterTaskClass()
}

func TestSchedule(t *testing.T) {
	Convey(`TestSchedule`, t, func() {
		ctx := testutil.SpannerTestContext(t)
		ctx, skdr := tq.TestingContext(ctx, nil)

		task := &taskspb.IngestTestVerdicts{
			Build:         &ctrlpb.BuildResult{},
			PartitionTime: timestamppb.New(time.Date(2025, time.January, 1, 12, 0, 0, 0, time.UTC)),
		}
		expected := proto.Clone(task).(*taskspb.IngestTestVerdicts)

		_, err := span.ReadWriteTransaction(ctx, func(ctx context.Context) error {
			Schedule(ctx, task)
			return nil
		})
		So(err, ShouldBeNil)
		So(skdr.Tasks().Payloads()[0], ShouldResembleProto, expected)
	})
}

func TestIngestTestVerdicts(t *testing.T) {
	Convey(`TestIngestTestVerdicts`, t, func() {
		ctx := testutil.SpannerTestContext(t)

		Convey(`partition time`, func() {
			payload := &taskspb.IngestTestVerdicts{
				Build: &ctrlpb.BuildResult{
					Host: "host",
					Id:   13131313,
				},
				PartitionTime: timestamppb.New(clock.Now(ctx).Add(-1 * time.Hour)),
			}
			Convey(`too early`, func() {
				payload.PartitionTime = timestamppb.New(clock.Now(ctx).Add(25 * time.Hour))
				err := ingestTestVerdicts(ctx, payload)
				So(err, ShouldErrLike, "too far in the future")
			})
			Convey(`too late`, func() {
				payload.PartitionTime = timestamppb.New(clock.Now(ctx).Add(-91 * 24 * time.Hour))
				err := ingestTestVerdicts(ctx, payload)
				So(err, ShouldErrLike, "too long ago")
			})
		})

		Convey(`valid payload`, func() {
			ctl := gomock.NewController(t)
			defer ctl.Finish()

			mrc := resultdb.NewMockedClient(ctx, ctl)
			mbc := buildbucket.NewMockedClient(mrc.Ctx, ctl)
			ctx = mbc.Ctx

			bID := int64(87654321)
			inv := "invocations/build-87654321"
			realm := "chromium:ci"

			request := &bbpb.GetBuildRequest{
				Id: bID,
				Mask: &bbpb.BuildMask{
					Fields: &field_mask.FieldMask{
						Paths: []string{"input.gerrit_changes", "infra.resultdb", "status"},
					},
				},
			}
			mbc.GetBuild(request, mockedGetBuildRsp(inv))

			invReq := &rdbpb.GetInvocationRequest{
				Name: inv,
			}
			invRes := &rdbpb.Invocation{
				Name:  inv,
				Realm: realm,
			}
			mrc.GetInvocation(invReq, invRes)

			tvReq := &rdbpb.QueryTestVariantsRequest{
				Invocations: []string{inv},
				PageSize:    10000,
				ReadMask: &fieldmaskpb.FieldMask{
					Paths: []string{
						"test_id",
						"variant_hash",
						"status",
						"variant",
						"results.*.result.status",
						"results.*.result.expected",
						"results.*.result.duration",
					},
				},
			}
			mrc.QueryTestVariants(tvReq, mockedQueryTestVariantsRsp())

			payload := &taskspb.IngestTestVerdicts{
				Build: &ctrlpb.BuildResult{
					Host: "cr-buildbucket-dev.appspot.com",
					Id:   bID,
				},
				PartitionTime: timestamppb.New(clock.Now(ctx)),
			}
			err := ingestTestVerdicts(ctx, payload)
			So(err, ShouldBeNil)

			// Validate IngestedInvocations table is populated.
			err = testverdicts.ReadIngestedInvocations(span.Single(ctx), spanner.AllKeys(), func(inv *testverdicts.IngestedInvocation) error {
				So(inv, ShouldResemble, &testverdicts.IngestedInvocation{
					Project:                      "chromium",
					IngestedInvocationID:         "build-87654321",
					SubRealm:                     "ci",
					PartitionTime:                payload.PartitionTime.AsTime(),
					HasUnsubmittedChanges:        false,
					HasContributedToClSubmission: false,
				})
				return nil
			})
			So(err, ShouldBeNil)

			// Validate TestVerdicts table is populated.
			tvs := make([]*testverdicts.TestVerdict, 0)
			err = testverdicts.ReadTestVerdicts(span.Single(ctx), spanner.AllKeys(), func(tv *testverdicts.TestVerdict) error {
				tvs = append(tvs, tv)
				return nil
			})
			So(err, ShouldBeNil)

			tvBuilder := testverdicts.NewTestVerdict().
				WithProject("chromium").
				WithPartitionTime(payload.PartitionTime.AsTime()).
				WithIngestedInvocationID("build-87654321").
				WithSubRealm("ci")
			So(tvs, ShouldResemble, []*testverdicts.TestVerdict{
				tvBuilder.WithTestID("test_id_1").
					WithVariantHash("hash_1").
					WithExpectedCount(0).
					WithUnexpectedCount(1).
					WithSkippedCount(0).
					WithIsExonerated(false).
					WithoutPassedAvgDuration().
					Build(),
				tvBuilder.WithTestID("test_id_1").
					WithVariantHash("hash_2").
					WithExpectedCount(1).
					WithUnexpectedCount(1).
					WithSkippedCount(0).
					WithIsExonerated(false).
					WithPassedAvgDuration(time.Second).
					Build(),
				tvBuilder.WithTestID("test_id_2").
					WithVariantHash("hash_1").
					WithExpectedCount(2).
					WithUnexpectedCount(1).
					WithSkippedCount(0).
					WithIsExonerated(false).
					WithPassedAvgDuration(time.Second * 2).
					Build(),
				tvBuilder.WithTestID("test_id_2").
					WithVariantHash("hash_2").
					WithExpectedCount(2).
					WithUnexpectedCount(1).
					WithSkippedCount(0).
					WithIsExonerated(true).
					WithPassedAvgDuration(time.Second).
					Build(),
			})

			// Validate TestVariantRealms table is populated.
			tvrs := make([]*testverdicts.TestVariantRealm, 0)
			err = testverdicts.ReadTestVariantRealms(span.Single(ctx), spanner.AllKeys(), func(tvr *testverdicts.TestVariantRealm) error {
				tvrs = append(tvrs, tvr)
				return nil
			})
			So(err, ShouldBeNil)
			So(tvrs[0].LastIngestionTime, ShouldNotBeZeroValue)
			So(tvrs[1].LastIngestionTime, ShouldNotBeZeroValue)
			So(tvrs[2].LastIngestionTime, ShouldNotBeZeroValue)
			So(tvrs[3].LastIngestionTime, ShouldNotBeZeroValue)
			So(tvrs, ShouldResemble, []*testverdicts.TestVariantRealm{
				{
					Project:           "chromium",
					TestID:            "test_id_1",
					VariantHash:       "hash_1",
					SubRealm:          "ci",
					Variant:           pbutil.VariantFromResultDB(rdbpbutil.Variant("k1", "v1")),
					LastIngestionTime: tvrs[0].LastIngestionTime,
				},
				{
					Project:           "chromium",
					TestID:            "test_id_1",
					VariantHash:       "hash_2",
					SubRealm:          "ci",
					Variant:           pbutil.VariantFromResultDB(rdbpbutil.Variant("k1", "v2")),
					LastIngestionTime: tvrs[1].LastIngestionTime,
				},
				{
					Project:           "chromium",
					TestID:            "test_id_2",
					VariantHash:       "hash_1",
					SubRealm:          "ci",
					Variant:           pbutil.VariantFromResultDB(rdbpbutil.Variant("k1", "v1")),
					LastIngestionTime: tvrs[2].LastIngestionTime,
				},
				{
					Project:           "chromium",
					TestID:            "test_id_2",
					VariantHash:       "hash_2",
					SubRealm:          "ci",
					Variant:           pbutil.VariantFromResultDB(rdbpbutil.Variant("k1", "v2")),
					LastIngestionTime: tvrs[3].LastIngestionTime,
				},
			})
		})
	})
}
