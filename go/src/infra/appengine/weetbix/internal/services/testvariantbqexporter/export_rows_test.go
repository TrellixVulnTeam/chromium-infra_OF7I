// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package testvariantbqexporter

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"go.chromium.org/luci/common/bq"
	"go.chromium.org/luci/common/clock"

	"infra/appengine/weetbix/internal"
	"infra/appengine/weetbix/internal/span"
	"infra/appengine/weetbix/internal/testutil"
	"infra/appengine/weetbix/internal/testutil/insert"
	"infra/appengine/weetbix/pbutil"
	bqpb "infra/appengine/weetbix/proto/bq"
	pb "infra/appengine/weetbix/proto/v1"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

type mockPassInserter struct {
	insertedMessages []*bq.Row
	mu               sync.Mutex
}

func (i *mockPassInserter) PutWithRetries(ctx context.Context, src []*bq.Row) error {
	i.mu.Lock()
	i.insertedMessages = append(i.insertedMessages, src...)
	i.mu.Unlock()
	return nil
}

type mockFailInserter struct {
}

func (i *mockFailInserter) PutWithRetries(ctx context.Context, src []*bq.Row) error {
	return fmt.Errorf("some error")
}

func TestQueryTestVariantsToExport(t *testing.T) {
	Convey(`queryTestVariantsToExport`, t, func() {
		ctx := testutil.SpannerTestContext(t)
		realm := "chromium:ci"
		tID1 := "ninja://test1"
		tID2 := "ninja://test2"
		tID3 := "ninja://test3"
		tID4 := "ninja://test4"
		tID5 := "ninja://test5"
		tID6 := "ninja://test6"
		variant := pbutil.Variant("builder", "Linux Tests")
		vh := "varianthash"
		tags := pbutil.StringPairs("k1", "v1")
		tmd := &pb.TestMetadata{
			Location: &pb.TestLocation{
				Repo:     "https://chromium.googlesource.com/chromium/src",
				FileName: "//a_test.go",
			},
		}
		tmdM, _ := proto.Marshal(tmd)
		now := clock.Now(ctx).Round(time.Microsecond)
		start := clock.Now(ctx).Add(-time.Hour)
		twoAndHalfHAgo := now.Add(-150 * time.Minute)
		oneAndHalfHAgo := now.Add(-90 * time.Minute)
		halfHAgo := now.Add(-30 * time.Minute)
		m46Ago := now.Add(-46 * time.Minute)
		ms := []*spanner.Mutation{
			insert.AnalyzedTestVariant(realm, tID1, vh, pb.AnalyzedTestVariantStatus_FLAKY, map[string]interface{}{
				"Variant":          variant,
				"Tags":             tags,
				"TestMetadata":     span.Compressed(tmdM),
				"StatusUpdateTime": start.Add(-time.Hour),
			}),
			// New flaky test variant.
			insert.AnalyzedTestVariant(realm, tID2, vh, pb.AnalyzedTestVariantStatus_FLAKY, map[string]interface{}{
				"Variant":          variant,
				"Tags":             tags,
				"TestMetadata":     span.Compressed(tmdM),
				"StatusUpdateTime": halfHAgo,
			}),
			// Flaky test with no verdicts in time range.
			insert.AnalyzedTestVariant(realm, tID3, vh, pb.AnalyzedTestVariantStatus_FLAKY, map[string]interface{}{
				"Variant":          variant,
				"Tags":             tags,
				"TestMetadata":     span.Compressed(tmdM),
				"StatusUpdateTime": start.Add(-time.Hour),
			}),
			// Test variant with another status is not exported.
			insert.AnalyzedTestVariant(realm, tID4, vh, pb.AnalyzedTestVariantStatus_CONSISTENTLY_UNEXPECTED, map[string]interface{}{
				"Variant":          variant,
				"Tags":             tags,
				"TestMetadata":     span.Compressed(tmdM),
				"StatusUpdateTime": start.Add(-time.Hour),
			}),
			// Test variant has multiple status updates.
			insert.AnalyzedTestVariant(realm, tID5, vh, pb.AnalyzedTestVariantStatus_FLAKY, map[string]interface{}{
				"Variant":          variant,
				"Tags":             tags,
				"TestMetadata":     span.Compressed(tmdM),
				"StatusUpdateTime": halfHAgo,
				"PreviousStatuses": []pb.AnalyzedTestVariantStatus{
					pb.AnalyzedTestVariantStatus_CONSISTENTLY_EXPECTED,
					pb.AnalyzedTestVariantStatus_FLAKY},
				"PreviousStatusUpdateTimes": []time.Time{
					m46Ago,
					now.Add(-24 * time.Hour)},
			}),
			// Test variant with different variant.
			insert.AnalyzedTestVariant(realm, tID6, "c467ccce5a16dc72", pb.AnalyzedTestVariantStatus_CONSISTENTLY_EXPECTED, map[string]interface{}{
				"Variant":          pbutil.Variant("a", "b"),
				"Tags":             tags,
				"TestMetadata":     span.Compressed(tmdM),
				"StatusUpdateTime": twoAndHalfHAgo,
			}),
			insert.Verdict(realm, tID1, vh, "build-0", internal.VerdictStatus_EXPECTED, twoAndHalfHAgo, map[string]interface{}{
				"IngestionTime":         oneAndHalfHAgo,
				"UnexpectedResultCount": 0,
				"TotalResultCount":      1,
			}),
			insert.Verdict(realm, tID1, vh, "build-1", internal.VerdictStatus_VERDICT_FLAKY, twoAndHalfHAgo, map[string]interface{}{
				"IngestionTime":         halfHAgo,
				"UnexpectedResultCount": 1,
				"TotalResultCount":      2,
			}),
			insert.Verdict(realm, tID1, vh, "build-2", internal.VerdictStatus_EXPECTED, oneAndHalfHAgo, map[string]interface{}{
				"IngestionTime":         halfHAgo,
				"UnexpectedResultCount": 0,
				"TotalResultCount":      1,
			}),
			insert.Verdict(realm, tID2, vh, "build-2", internal.VerdictStatus_VERDICT_FLAKY, oneAndHalfHAgo, map[string]interface{}{
				"IngestionTime":         halfHAgo,
				"UnexpectedResultCount": 1,
				"TotalResultCount":      2,
			}),
			insert.Verdict(realm, tID5, vh, "build-1", internal.VerdictStatus_EXPECTED, twoAndHalfHAgo, map[string]interface{}{
				"IngestionTime":         now.Add(-45 * time.Minute),
				"UnexpectedResultCount": 0,
				"TotalResultCount":      1,
			}),
			insert.Verdict(realm, tID5, vh, "build-2", internal.VerdictStatus_VERDICT_FLAKY, oneAndHalfHAgo, map[string]interface{}{
				"IngestionTime":         halfHAgo,
				"UnexpectedResultCount": 1,
				"TotalResultCount":      2,
			}),
		}
		testutil.MustApply(ctx, ms...)

		verdicts := map[string]map[string]*bqpb.Verdict{
			tID1: {
				"build-0": {
					Invocation: "build-0",
					Status:     "EXPECTED",
					CreateTime: timestamppb.New(twoAndHalfHAgo),
				},
				"build-1": {
					Invocation: "build-1",
					Status:     "VERDICT_FLAKY",
					CreateTime: timestamppb.New(twoAndHalfHAgo),
				},
				"build-2": {
					Invocation: "build-2",
					Status:     "EXPECTED",
					CreateTime: timestamppb.New(oneAndHalfHAgo),
				},
			},
			tID2: {
				"build-2": {
					Invocation: "build-2",
					Status:     "VERDICT_FLAKY",
					CreateTime: timestamppb.New(oneAndHalfHAgo),
				},
			},
			tID5: {
				"build-1": {
					Invocation: "build-1",
					Status:     "EXPECTED",
					CreateTime: timestamppb.New(twoAndHalfHAgo),
				},
				"build-2": {
					Invocation: "build-2",
					Status:     "VERDICT_FLAKY",
					CreateTime: timestamppb.New(oneAndHalfHAgo),
				},
			},
		}

		op := &Options{
			Realm:        realm,
			CloudProject: "cloud_project",
			Dataset:      "dataset",
			Table:        "table",
			TimeRange: &pb.TimeRange{
				Earliest: timestamppb.New(start),
				Latest:   timestamppb.New(now),
			},
		}
		br := CreateBQExporter(op)

		// To check when encountering an error, the test can run to the end
		// without hanging, or race detector does not detect anything.
		Convey(`insert fail`, func() {
			err := br.exportTestVariantRows(ctx, &mockFailInserter{})
			So(err, ShouldErrLike, "some error")
		})

		simplifyF := func(rows []*bqpb.TestVariantRow) []*bqpb.TestVariantRow {
			simpleRows := make([]*bqpb.TestVariantRow, len(rows))
			for i, r := range rows {
				simpleRows[i] = &bqpb.TestVariantRow{
					TestId:          r.TestId,
					Status:          r.Status,
					TimeRange:       r.TimeRange,
					FlakeStatistics: r.FlakeStatistics,
					Verdicts:        r.Verdicts,
				}
			}
			return simpleRows
		}

		sortF := func(rows []*bqpb.TestVariantRow) {
			sort.Slice(rows, func(i, j int) bool {
				switch {
				case rows[i].TestId != rows[j].TestId:
					return rows[i].TestId < rows[j].TestId
				default:
					earliestI, _ := pbutil.AsTime(rows[i].TimeRange.Earliest)
					earliestJ, _ := pbutil.AsTime(rows[j].TimeRange.Earliest)
					return earliestI.Before(earliestJ)
				}
			})
		}

		test := func(predicate *pb.AnalyzedTestVariantPredicate, expRows []*bqpb.TestVariantRow) {
			op.Predicate = predicate
			ins := &mockPassInserter{}
			err := br.exportTestVariantRows(ctx, ins)
			So(err, ShouldBeNil)

			rows := make([]*bqpb.TestVariantRow, len(ins.insertedMessages))
			for i, m := range ins.insertedMessages {
				rows[i] = m.Message.(*bqpb.TestVariantRow)
			}
			rows = simplifyF(rows)
			sortF(rows)
			sortF(expRows)
			So(rows, ShouldResembleProto, expRows)
		}

		Convey(`no predicate`, func() {
			expRows := []*bqpb.TestVariantRow{
				{
					TestId: tID1,
					TimeRange: &pb.TimeRange{
						Earliest: op.TimeRange.Earliest,
						Latest:   op.TimeRange.Latest,
					},
					Status: "FLAKY",
					FlakeStatistics: &pb.FlakeStatistics{
						FlakyVerdictRate:      0.5,
						FlakyVerdictCount:     1,
						TotalVerdictCount:     2,
						UnexpectedResultRate:  float32(1) / 3,
						UnexpectedResultCount: 1,
						TotalResultCount:      3,
					},
					Verdicts: []*bqpb.Verdict{
						verdicts[tID1]["build-2"],
						verdicts[tID1]["build-1"],
					},
				},
				{
					TestId: tID4,
					TimeRange: &pb.TimeRange{
						Earliest: op.TimeRange.Earliest,
						Latest:   op.TimeRange.Latest,
					},
					Status:          "CONSISTENTLY_UNEXPECTED",
					FlakeStatistics: zeroFlakyStatistics(),
				},
				{
					TestId: tID5,
					TimeRange: &pb.TimeRange{
						Earliest: timestamppb.New(halfHAgo),
						Latest:   op.TimeRange.Latest,
					},
					Status: "FLAKY",
					FlakeStatistics: &pb.FlakeStatistics{
						FlakyVerdictRate:      1.0,
						FlakyVerdictCount:     1,
						TotalVerdictCount:     1,
						UnexpectedResultRate:  0.5,
						UnexpectedResultCount: 1,
						TotalResultCount:      2,
					},
					Verdicts: []*bqpb.Verdict{
						verdicts[tID5]["build-2"],
					},
				},
				{
					TestId: tID5,
					TimeRange: &pb.TimeRange{
						Earliest: timestamppb.New(m46Ago),
						Latest:   timestamppb.New(halfHAgo),
					},
					Status: "CONSISTENTLY_EXPECTED",
					FlakeStatistics: &pb.FlakeStatistics{
						FlakyVerdictRate:      0.0,
						FlakyVerdictCount:     0,
						TotalVerdictCount:     1,
						UnexpectedResultRate:  0.0,
						UnexpectedResultCount: 0,
						TotalResultCount:      1,
					},
					Verdicts: []*bqpb.Verdict{
						verdicts[tID5]["build-1"],
					},
				},
				{
					TestId: tID5,
					TimeRange: &pb.TimeRange{
						Earliest: op.TimeRange.Earliest,
						Latest:   timestamppb.New(m46Ago),
					},
					Status:          "FLAKY",
					FlakeStatistics: zeroFlakyStatistics(),
				},
				{
					TestId: tID2,
					TimeRange: &pb.TimeRange{
						Earliest: timestamppb.New(halfHAgo),
						Latest:   op.TimeRange.Latest,
					},
					Status: "FLAKY",
					FlakeStatistics: &pb.FlakeStatistics{
						FlakyVerdictRate:      1.0,
						FlakyVerdictCount:     1,
						TotalVerdictCount:     1,
						UnexpectedResultRate:  0.5,
						UnexpectedResultCount: 1,
						TotalResultCount:      2,
					},
					Verdicts: []*bqpb.Verdict{
						verdicts[tID2]["build-2"],
					},
				},
				{
					TestId: tID6,
					TimeRange: &pb.TimeRange{
						Earliest: op.TimeRange.Earliest,
						Latest:   op.TimeRange.Latest,
					},
					Status:          "CONSISTENTLY_EXPECTED",
					FlakeStatistics: zeroFlakyStatistics(),
				},
				{
					TestId: tID3,
					TimeRange: &pb.TimeRange{
						Earliest: op.TimeRange.Earliest,
						Latest:   op.TimeRange.Latest,
					},
					Status:          "FLAKY",
					FlakeStatistics: zeroFlakyStatistics(),
				},
			}
			test(nil, expRows)
		})

		Convey(`status predicate`, func() {
			predicate := &pb.AnalyzedTestVariantPredicate{
				Status: pb.AnalyzedTestVariantStatus_FLAKY,
			}

			expRows := []*bqpb.TestVariantRow{
				{
					TestId: tID2,
					TimeRange: &pb.TimeRange{
						Earliest: timestamppb.New(halfHAgo),
						Latest:   op.TimeRange.Latest,
					},
					Status: "FLAKY",
					FlakeStatistics: &pb.FlakeStatistics{
						FlakyVerdictRate:      1.0,
						FlakyVerdictCount:     1,
						TotalVerdictCount:     1,
						UnexpectedResultRate:  0.5,
						UnexpectedResultCount: 1,
						TotalResultCount:      2,
					},
					Verdicts: []*bqpb.Verdict{
						verdicts[tID2]["build-2"],
					},
				},
				{
					TestId: tID1,
					TimeRange: &pb.TimeRange{
						Earliest: op.TimeRange.Earliest,
						Latest:   op.TimeRange.Latest,
					},
					Status: "FLAKY",
					FlakeStatistics: &pb.FlakeStatistics{
						FlakyVerdictRate:      0.5,
						FlakyVerdictCount:     1,
						TotalVerdictCount:     2,
						UnexpectedResultRate:  float32(1) / 3,
						UnexpectedResultCount: 1,
						TotalResultCount:      3,
					},
					Verdicts: []*bqpb.Verdict{
						verdicts[tID1]["build-2"],
						verdicts[tID1]["build-1"],
					},
				},
				{
					TestId: tID3,
					TimeRange: &pb.TimeRange{
						Earliest: op.TimeRange.Earliest,
						Latest:   op.TimeRange.Latest,
					},
					Status:          "FLAKY",
					FlakeStatistics: zeroFlakyStatistics(),
				},
				{
					TestId: tID5,
					TimeRange: &pb.TimeRange{
						Earliest: timestamppb.New(halfHAgo),
						Latest:   op.TimeRange.Latest,
					},
					Status: "FLAKY",
					FlakeStatistics: &pb.FlakeStatistics{
						FlakyVerdictRate:      1.0,
						FlakyVerdictCount:     1,
						TotalVerdictCount:     1,
						UnexpectedResultRate:  0.5,
						UnexpectedResultCount: 1,
						TotalResultCount:      2,
					},
					Verdicts: []*bqpb.Verdict{
						{
							Invocation: "build-2",
							Status:     "VERDICT_FLAKY",
							CreateTime: timestamppb.New(oneAndHalfHAgo),
						},
					},
				},
				{
					TestId: tID5,
					TimeRange: &pb.TimeRange{
						Earliest: op.TimeRange.Earliest,
						Latest:   timestamppb.New(m46Ago),
					},
					Status:          "FLAKY",
					FlakeStatistics: zeroFlakyStatistics(),
				},
			}

			test(predicate, expRows)
		})

		Convey(`testIdRegexp`, func() {
			predicate := &pb.AnalyzedTestVariantPredicate{
				TestIdRegexp: tID1,
			}
			expRows := []*bqpb.TestVariantRow{
				{
					TestId: tID1,
					TimeRange: &pb.TimeRange{
						Earliest: op.TimeRange.Earliest,
						Latest:   op.TimeRange.Latest,
					},
					Status: "FLAKY",
					FlakeStatistics: &pb.FlakeStatistics{
						FlakyVerdictRate:      0.5,
						FlakyVerdictCount:     1,
						TotalVerdictCount:     2,
						UnexpectedResultRate:  float32(1) / 3,
						UnexpectedResultCount: 1,
						TotalResultCount:      3,
					},
					Verdicts: []*bqpb.Verdict{
						verdicts[tID1]["build-2"],
						verdicts[tID1]["build-1"],
					},
				},
			}

			test(predicate, expRows)
		})

		variantExpRows := []*bqpb.TestVariantRow{
			{
				TestId: tID6,
				TimeRange: &pb.TimeRange{
					Earliest: op.TimeRange.Earliest,
					Latest:   op.TimeRange.Latest,
				},
				Status:          "CONSISTENTLY_EXPECTED",
				FlakeStatistics: zeroFlakyStatistics(),
			},
		}
		Convey(`variantEqual`, func() {
			predicate := &pb.AnalyzedTestVariantPredicate{
				Variant: &pb.VariantPredicate{
					Predicate: &pb.VariantPredicate_Equals{
						Equals: pbutil.Variant("a", "b"),
					},
				},
			}
			test(predicate, variantExpRows)
		})

		Convey(`variantContain`, func() {
			predicate := &pb.AnalyzedTestVariantPredicate{
				Variant: &pb.VariantPredicate{
					Predicate: &pb.VariantPredicate_Contains{
						Contains: pbutil.Variant("a", "b"),
					},
				},
			}
			test(predicate, variantExpRows)
		})
	})
}
