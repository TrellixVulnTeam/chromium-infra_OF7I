// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package resultingester

import (
	"sort"
	"testing"

	"cloud.google.com/go/spanner"
	"github.com/golang/mock/gomock"
	"google.golang.org/protobuf/proto"

	"go.chromium.org/luci/resultdb/pbutil"
	rdbpb "go.chromium.org/luci/resultdb/proto/v1"
	"go.chromium.org/luci/server/span"
	"go.chromium.org/luci/server/tq"
	_ "go.chromium.org/luci/server/tq/txn/spanner"

	"infra/appengine/weetbix/internal/buildbucket"
	"infra/appengine/weetbix/internal/resultdb"
	"infra/appengine/weetbix/internal/services/resultcollector"
	"infra/appengine/weetbix/internal/services/testvariantupdator"
	spanutil "infra/appengine/weetbix/internal/span"
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
		RegisterTasksClass()

		build := &taskspb.Build{}
		So(Schedule(ctx, nil, build), ShouldBeNil)
		So(skdr.Tasks().Payloads()[0], ShouldResembleProto, &taskspb.IngestTestResults{Build: build})
	})
}

func TestIngestTestResults(t *testing.T) {
	Convey(`TestIngestTestResults`, t, func() {
		ctx := testutil.SpannerTestContext(t)
		ctx, skdr := tq.TestingContext(ctx, nil)
		resultcollector.RegisterTaskClass()
		testvariantupdator.RegisterTaskClass()

		ctl := gomock.NewController(t)
		defer ctl.Finish()

		mrc := resultdb.NewMockedClient(ctx, ctl)
		mbc := buildbucket.NewMockedClient(mrc.Ctx, ctl)
		ctx = mbc.Ctx

		bID := int64(87654321)
		inv := "invocations/build-87654321"
		realm := "chromium:ci"

		mbc.GetBuildWithBuilderAndRDBInfo(bID, mockedGetBuildRsp(inv))

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
			PageSize:    1000,
			Predicate: &rdbpb.TestVariantPredicate{
				Status: rdbpb.TestVariantStatus_UNEXPECTED_MASK,
			},
		}
		mrc.QueryTestVariants(tvReq, mockedQueryTestVariantsRsp())

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
					Id:   bID,
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
			sampleTestId := "ninja://test_new_failure"
			expProto := &pb.AnalyzedTestVariant{
				Realm:        realm,
				TestId:       sampleTestId,
				VariantHash:  "hash",
				Status:       pb.AnalyzedTestVariantStatus_HAS_UNEXPECTED_RESULTS,
				Variant:      sampleVar,
				Tags:         pbutil.StringPairs("monorail_component", "Monorail>Component"),
				TestMetadata: sampleTmd,
			}

			var testIDsWithNextTask []string
			fields := []string{"Realm", "TestId", "VariantHash", "Status", "Variant", "Tags", "TestMetadata", "NextUpdateTaskEnqueueTime"}
			var actProto *pb.AnalyzedTestVariant
			var b spanutil.Buffer
			err = span.Read(ctx, "AnalyzedTestVariants", spanner.AllKeys(), fields).Do(
				func(row *spanner.Row) error {
					tv := &pb.AnalyzedTestVariant{}
					var tmd spanutil.Compressed
					var enqTime spanner.NullTime
					err = b.FromSpanner(row, &tv.Realm, &tv.TestId, &tv.VariantHash, &tv.Status, &tv.Variant, &tv.Tags, &tmd, &enqTime)
					So(err, ShouldBeNil)
					So(tv.Realm, ShouldEqual, realm)

					if len(tmd) > 0 {
						tv.TestMetadata = &rdbpb.TestMetadata{}
						err = proto.Unmarshal(tmd, tv.TestMetadata)
						So(err, ShouldBeNil)
					}

					act[tv.TestId] = tv.Status
					if tv.TestId == sampleTestId {
						actProto = tv
					}

					if !enqTime.IsNull() {
						testIDsWithNextTask = append(testIDsWithNextTask, tv.TestId)
					}
					return nil
				},
			)
			So(err, ShouldBeNil)
			So(act, ShouldResemble, exp)
			So(actProto, ShouldResembleProto, expProto)
			sort.Strings(testIDsWithNextTask)

			// Should have enqueued 1 CollectTestResults task, 3 UpdateTestVariant tasks.
			So(len(skdr.Tasks().Payloads()), ShouldEqual, 4)
			expColTask := &taskspb.CollectTestResults{
				Resultdb: &taskspb.ResultDB{
					Invocation: &rdbpb.Invocation{
						Name:  inv,
						Realm: realm,
					},
					Host: "results.api.cr.dev",
				},
				Builder:                   "builder",
				IsPreSubmit:               false,
				ContributedToClSubmission: false,
			}
			var actTestIDsWithTasks []string
			for _, pl := range skdr.Tasks().Payloads() {
				switch pl.(type) {
				case *taskspb.UpdateTestVariant:
					plp := pl.(*taskspb.UpdateTestVariant)
					actTestIDsWithTasks = append(actTestIDsWithTasks, plp.TestVariantKey.TestId)
				case *taskspb.CollectTestResults:
					plp := pl.(*taskspb.CollectTestResults)
					So(plp, ShouldResembleProto, expColTask)
				default:
				}
			}
			sort.Strings(actTestIDsWithTasks)
			So(len(actTestIDsWithTasks), ShouldEqual, 3)
			So(actTestIDsWithTasks, ShouldResemble, testIDsWithNextTask)
		})
	})
}
