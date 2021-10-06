// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package resultingester

import (
	"context"
	"testing"

	"cloud.google.com/go/spanner"
	"github.com/golang/mock/gomock"
	"google.golang.org/genproto/protobuf/field_mask"
	"google.golang.org/grpc"

	bbpb "go.chromium.org/luci/buildbucket/proto"
	rdbpb "go.chromium.org/luci/resultdb/proto/v1"
	"go.chromium.org/luci/server/span"
	"go.chromium.org/luci/server/tq"

	"infra/appengine/weetbix/internal/analyzedtestvariants"
	"infra/appengine/weetbix/internal/buildbucket"
	"infra/appengine/weetbix/internal/resultdb"
	"infra/appengine/weetbix/internal/tasks/taskspb"
	"infra/appengine/weetbix/internal/testutil"
	"infra/appengine/weetbix/internal/testutil/insert"
	pb "infra/appengine/weetbix/proto/v1"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func TestSchedule(t *testing.T) {
	Convey(`TestSchedule`, t, func() {
		ctx, skdr := tq.TestingContext(testutil.TestingContext(), nil)
		RegisterResultIngestionTasksClass()

		build := &taskspb.Build{}
		So(Schedule(ctx, nil, build), ShouldBeNil)
		So(skdr.Tasks().Payloads()[0], ShouldResembleProto, &taskspb.IngestTestResults{Build: build})
	})
}

func TestIngestTestResults(t *testing.T) {
	Convey(`TestIngestTestResults`, t, func() {
		ctl := gomock.NewController(t)
		defer ctl.Finish()

		mrc := resultdb.NewMockedClient(testutil.SpannerTestContext(t), ctl)
		mbc := buildbucket.NewMockedClient(mrc.Ctx, ctl)
		ctx := mbc.Ctx

		bId := int64(87654321)
		inv := "invocations/build-87654321"
		realm := "chromium:ci"

		bbReq := &bbpb.GetBuildRequest{
			Id: bId,
			Fields: &field_mask.FieldMask{
				Paths: []string{"infra.resultdb"},
			},
		}
		bbResF := func(ctx context.Context, in *bbpb.GetBuildRequest, opt grpc.CallOption) (*bbpb.Build, error) {
			return mockedGetBuildRsp(inv), nil
		}
		mbc.GetBuild(bbReq, bbResF)

		tvReq := &rdbpb.QueryTestVariantsRequest{
			Invocations: []string{inv},
			PageSize:    1000,
		}
		tvResF := func(ctx context.Context, in *rdbpb.QueryTestVariantsRequest, opt grpc.CallOption) (*rdbpb.QueryTestVariantsResponse, error) {
			return mockedQueryTestVariantsRsp(), nil
		}
		mrc.QueryTestVariants(tvReq, tvResF)
		mrc.GetRealm(inv, realm)

		// Prepare some existing analyzed test variants to update.
		ms := []*spanner.Mutation{
			// Known flake's status should remain unchanged.
			insert.AnalyzedTestVariant(realm, "ninja://test_known_flake", "hash", pb.AnalyzedTestVariantStatus_FLAKY, nil),
			// Non-flake test variant's status will change when see a flaky occurrence.
			insert.AnalyzedTestVariant(realm, "ninja://test_has_unexpected", "hash", pb.AnalyzedTestVariantStatus_HAS_UNEXPECTED_RESULTS, nil),
			// Consistently failed test variant.
			insert.AnalyzedTestVariant(realm, "ninja://test_consistent_failure", "hash", pb.AnalyzedTestVariantStatus_CONSISTENTLY_UNEXPECTED, nil),
			// Stale test variant has new failure.
			insert.AnalyzedTestVariant(realm, "ninja://test_no_new_results", "hash", pb.AnalyzedTestVariantStatus_NO_NEW_RESULTS, nil),
		}
		testutil.MustApply(ctx, ms...)

		Convey(`ingest test variants`, func() {
			payload := &taskspb.IngestTestResults{
				Build: &taskspb.Build{
					Host: "host",
					Id:   bId,
				},
			}
			err := ingestTestResults(ctx, payload)
			So(err, ShouldBeNil)

			// Read rows from Spanner to confirm the analyzed test variants are saved.
			ctx, cancel := span.ReadOnlyTransaction(ctx)
			defer cancel()

			exp := map[string]pb.AnalyzedTestVariantStatus{
				"ninja://test_new_failure":        pb.AnalyzedTestVariantStatus_HAS_UNEXPECTED_RESULTS,
				"ninja://test_known_flake":        pb.AnalyzedTestVariantStatus_FLAKY,
				"ninja://test_consistent_failure": pb.AnalyzedTestVariantStatus_CONSISTENTLY_UNEXPECTED,
				"ninja://test_no_new_results":     pb.AnalyzedTestVariantStatus_HAS_UNEXPECTED_RESULTS,
				"ninja://test_new_flake":          pb.AnalyzedTestVariantStatus_FLAKY,
				"ninja://test_has_unexpected":     pb.AnalyzedTestVariantStatus_FLAKY,
			}
			act := make(map[string]pb.AnalyzedTestVariantStatus)
			err = analyzedtestvariants.Read(ctx, spanner.AllKeys(), func(atv *pb.AnalyzedTestVariant) error {
				So(atv.Realm, ShouldEqual, realm)
				So(atv.TestId, ShouldNotEqual, "ninja://test_skip")
				act[atv.TestId] = atv.Status
				return nil
			})
			So(err, ShouldBeNil)
			So(len(act), ShouldEqual, 6)
			So(act, ShouldResemble, exp)
		})
	})
}
