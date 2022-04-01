// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package resultingester

import (
	"context"
	"sort"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
	bbpb "go.chromium.org/luci/buildbucket/proto"
	"go.chromium.org/luci/common/clock"
	. "go.chromium.org/luci/common/testing/assertions"
	"go.chromium.org/luci/gae/impl/memory"
	rdbpb "go.chromium.org/luci/resultdb/proto/v1"
	"go.chromium.org/luci/server/caching"
	"go.chromium.org/luci/server/span"
	"go.chromium.org/luci/server/tq"
	_ "go.chromium.org/luci/server/tq/txn/spanner"
	"google.golang.org/genproto/protobuf/field_mask"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"infra/appengine/weetbix/internal/analysis"
	"infra/appengine/weetbix/internal/analysis/clusteredfailures"
	"infra/appengine/weetbix/internal/buildbucket"
	"infra/appengine/weetbix/internal/clustering/chunkstore"
	"infra/appengine/weetbix/internal/clustering/ingestion"
	"infra/appengine/weetbix/internal/config"
	configpb "infra/appengine/weetbix/internal/config/proto"
	ctrlpb "infra/appengine/weetbix/internal/ingestion/control/proto"
	"infra/appengine/weetbix/internal/resultdb"
	"infra/appengine/weetbix/internal/services/resultcollector"
	"infra/appengine/weetbix/internal/services/testvariantupdator"
	spanutil "infra/appengine/weetbix/internal/span"
	"infra/appengine/weetbix/internal/tasks/taskspb"
	"infra/appengine/weetbix/internal/testutil"
	"infra/appengine/weetbix/internal/testutil/insert"
	"infra/appengine/weetbix/pbutil"
	pb "infra/appengine/weetbix/proto/v1"
)

func TestSchedule(t *testing.T) {
	Convey(`TestSchedule`, t, func() {
		ctx := testutil.SpannerTestContext(t)
		ctx, skdr := tq.TestingContext(ctx, nil)

		task := &taskspb.IngestTestResults{
			Build:         &ctrlpb.BuildResult{},
			PartitionTime: timestamppb.New(time.Date(2025, time.January, 1, 12, 0, 0, 0, time.UTC)),
		}
		expected := proto.Clone(task).(*taskspb.IngestTestResults)

		_, err := span.ReadWriteTransaction(ctx, func(ctx context.Context) error {
			Schedule(ctx, task)
			return nil
		})
		So(err, ShouldBeNil)
		So(skdr.Tasks().Payloads()[0], ShouldResembleProto, expected)
	})
}

func TestShouldIngestForTestVariants(t *testing.T) {
	t.Parallel()
	Convey(`With realm config`, t, func() {
		realm := &configpb.RealmConfig{
			Name: "ci",
			TestVariantAnalysis: &configpb.TestVariantAnalysisConfig{
				UpdateTestVariantTask: &configpb.UpdateTestVariantTask{
					UpdateTestVariantTaskInterval:   durationpb.New(time.Hour),
					TestVariantStatusUpdateDuration: durationpb.New(24 * time.Hour),
				},
			},
		}
		payload := &taskspb.IngestTestResults{
			Build: &ctrlpb.BuildResult{
				Host: "host",
				Id:   int64(1),
			},
			PartitionTime: timestamppb.New(time.Date(2025, time.January, 1, 12, 0, 0, 0, time.UTC)),
		}
		Convey(`CI`, func() {
			So(shouldIngestForTestVariants(realm, payload), ShouldBeTrue)
		})
		Convey(`CQ run`, func() {
			payload.PresubmitRun = &ctrlpb.PresubmitResult{
				PresubmitRunId: &pb.PresubmitRunId{
					System: "luci-cv",
					Id:     "chromium/1111111111111-1-1111111111111111",
				},
				PresubmitRunSucceeded: true,
				Mode:                  "FULL_RUN",
			}
			Convey(`Successful full run`, func() {
				So(shouldIngestForTestVariants(realm, payload), ShouldBeTrue)
			})
			Convey(`Successful dry run`, func() {
				payload.PresubmitRun.Mode = "DRY_RUN"
				So(shouldIngestForTestVariants(realm, payload), ShouldBeFalse)
			})
			Convey(`Failed run`, func() {
				payload.PresubmitRun.PresubmitRunSucceeded = false
				So(shouldIngestForTestVariants(realm, payload), ShouldBeFalse)
			})
		})
		Convey(`Test Variant analysis not configured`, func() {
			realm.TestVariantAnalysis = nil
			So(shouldIngestForTestVariants(realm, payload), ShouldBeFalse)
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

func TestIngestTestResults(t *testing.T) {
	resultcollector.RegisterTaskClass()
	testvariantupdator.RegisterTaskClass()

	Convey(`TestIngestTestResults`, t, func() {
		ctx := testutil.SpannerTestContext(t)
		ctx = caching.WithEmptyProcessCache(ctx) // For failure association rules cache.
		ctx, skdr := tq.TestingContext(ctx, nil)
		ctx = memory.Use(ctx)
		config.SetTestProjectConfig(ctx, createProjectsConfig())

		chunkStore := chunkstore.NewFakeClient()
		clusteredFailures := clusteredfailures.NewFakeClient()
		analysis := analysis.NewClusteringHandler(clusteredFailures)
		ri := &resultIngester{
			clustering: ingestion.New(chunkStore, analysis),
		}

		Convey(`partition time`, func() {
			payload := &taskspb.IngestTestResults{
				Build: &ctrlpb.BuildResult{
					Host: "host",
					Id:   13131313,
				},
				PartitionTime: timestamppb.New(clock.Now(ctx).Add(-1 * time.Hour)),
			}
			Convey(`too early`, func() {
				payload.PartitionTime = timestamppb.New(clock.Now(ctx).Add(25 * time.Hour))
				err := ri.ingestTestResults(ctx, payload)
				So(err, ShouldErrLike, "too far in the future")
			})
			Convey(`too late`, func() {
				payload.PartitionTime = timestamppb.New(clock.Now(ctx).Add(-91 * 24 * time.Hour))
				err := ri.ingestTestResults(ctx, payload)
				So(err, ShouldErrLike, "too long ago")
			})
		})
		Convey(`valid payload`, func() {
			config.SetTestProjectConfig(ctx, createProjectsConfig())

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
						Paths: []string{"builder", "infra.resultdb", "status"},
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
				PageSize:    1000,
				Predicate: &rdbpb.TestVariantPredicate{
					Status: rdbpb.TestVariantStatus_UNEXPECTED_MASK,
				},
			}
			mrc.QueryTestVariants(tvReq, mockedQueryTestVariantsRsp())

			// Prepare some existing analyzed test variants to update.
			ms := []*spanner.Mutation{
				// Known flake's status should remain unchanged.
				insert.AnalyzedTestVariant(realm, "ninja://test_known_flake", "hash", pb.AnalyzedTestVariantStatus_FLAKY, map[string]interface{}{
					"Tags": pbutil.StringPairs("test_name", "test_known_flake", "monorail_component", "Monorail>OldComponent"),
				}),
				// Non-flake test variant's status will change when see a flaky occurrence.
				insert.AnalyzedTestVariant(realm, "ninja://test_has_unexpected", "hash", pb.AnalyzedTestVariantStatus_HAS_UNEXPECTED_RESULTS, nil),
				// Consistently failed test variant.
				insert.AnalyzedTestVariant(realm, "ninja://test_consistent_failure", "hash", pb.AnalyzedTestVariantStatus_CONSISTENTLY_UNEXPECTED, nil),
				// Stale test variant has new failure.
				insert.AnalyzedTestVariant(realm, "ninja://test_no_new_results", "hash", pb.AnalyzedTestVariantStatus_NO_NEW_RESULTS, nil),
			}
			testutil.MustApply(ctx, ms...)

			payload := &taskspb.IngestTestResults{
				Build: &ctrlpb.BuildResult{
					Host: "host",
					Id:   bID,
				},
				PartitionTime: timestamppb.New(clock.Now(ctx).Add(-1 * time.Hour)),
			}
			err := ri.ingestTestResults(ctx, payload)
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
			expProtos := map[string]*pb.AnalyzedTestVariant{
				"ninja://test_new_failure": {
					Realm:        realm,
					TestId:       "ninja://test_new_failure",
					VariantHash:  "hash",
					Status:       pb.AnalyzedTestVariantStatus_HAS_UNEXPECTED_RESULTS,
					Variant:      pbutil.VariantFromResultDB(sampleVar),
					Tags:         pbutil.StringPairs("monorail_component", "Monorail>Component"),
					TestMetadata: pbutil.TestMetadataFromResultDB(sampleTmd),
				},
				"ninja://test_known_flake": {
					Realm:       realm,
					TestId:      "ninja://test_known_flake",
					VariantHash: "hash",
					Status:      pb.AnalyzedTestVariantStatus_FLAKY,
					Tags:        pbutil.StringPairs("test_name", "test_known_flake", "monorail_component", "Monorail>Component", "os", "Mac"),
				},
			}

			var testIDsWithNextTask []string
			fields := []string{"Realm", "TestId", "VariantHash", "Status", "Variant", "Tags", "TestMetadata", "NextUpdateTaskEnqueueTime"}
			actProtos := make(map[string]*pb.AnalyzedTestVariant, len(expProtos))
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
						tv.TestMetadata = &pb.TestMetadata{}
						err = proto.Unmarshal(tmd, tv.TestMetadata)
						So(err, ShouldBeNil)
					}

					act[tv.TestId] = tv.Status
					if _, ok := expProtos[tv.TestId]; ok {
						actProtos[tv.TestId] = tv
					}

					if !enqTime.IsNull() {
						testIDsWithNextTask = append(testIDsWithNextTask, tv.TestId)
					}
					return nil
				},
			)
			So(err, ShouldBeNil)
			So(act, ShouldResemble, exp)
			for k, actProto := range actProtos {
				v, ok := expProtos[k]
				So(ok, ShouldBeTrue)
				So(actProto, ShouldResembleProto, v)
			}
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

			// Confirm chunks have been written to GCS.
			So(len(chunkStore.Contents), ShouldEqual, 1)

			// Confirm clustering has occurred, with each test result in at
			// least one cluster.
			actualClusteredFailures := make(map[string]int)
			for project, insertions := range clusteredFailures.InsertionsByProject {
				So(project, ShouldEqual, "chromium")
				for _, f := range insertions {
					actualClusteredFailures[f.TestId] += 1
				}
			}
			expectedClusteredFailures := map[string]int{
				"ninja://test_new_failure":        1,
				"ninja://test_known_flake":        1,
				"ninja://test_consistent_failure": 1,
				"ninja://test_no_new_results":     1,
				"ninja://test_new_flake":          1,
				"ninja://test_has_unexpected":     1,
			}
			So(actualClusteredFailures, ShouldResemble, expectedClusteredFailures)
		})
		Convey(`no project config`, func() {
			ctl := gomock.NewController(t)
			defer ctl.Finish()

			mbc := buildbucket.NewMockedClient(ctx, ctl)
			ctx = mbc.Ctx

			bID := int64(87654321)
			inv := "invocations/build-87654321"

			request := &bbpb.GetBuildRequest{
				Id: bID,
				Mask: &bbpb.BuildMask{
					Fields: &field_mask.FieldMask{
						Paths: []string{"builder", "infra.resultdb", "status"},
					},
				},
			}
			mbc.GetBuild(request, mockedGetBuildRsp(inv))

			payload := &taskspb.IngestTestResults{
				Build: &ctrlpb.BuildResult{
					Host: "host",
					Id:   bID,
				},
				PartitionTime: timestamppb.New(clock.Now(ctx).Add(-1 * time.Hour)),
			}
			err := ri.ingestTestResults(ctx, payload)
			So(err, ShouldBeNil)

			// Confirm no tasks have been scheduled.
			So(len(skdr.Tasks().Payloads()), ShouldEqual, 0)
			// Confirm no chunks have been written to GCS.
			So(len(chunkStore.Contents), ShouldEqual, 0)
			// Confirm no clustering has occurred.
			So(clusteredFailures.InsertionsByProject, ShouldHaveLength, 0)
		})
	})
}
