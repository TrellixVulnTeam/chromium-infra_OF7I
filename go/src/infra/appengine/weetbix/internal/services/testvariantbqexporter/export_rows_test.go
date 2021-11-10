// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package testvariantbqexporter

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"go.chromium.org/luci/common/bq"
	"go.chromium.org/luci/common/clock"

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
		tID := "ninja://test1"
		tID3 := "ninja://test3"
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
		ms := []*spanner.Mutation{
			insert.AnalyzedTestVariant(realm, tID, vh, pb.AnalyzedTestVariantStatus_FLAKY, map[string]interface{}{
				"Variant":          pbutil.Variant("builder", "Linux Tests"),
				"Tags":             tags,
				"TestMetadata":     span.Compressed(tmdM),
				"StatusUpdateTime": start.Add(-time.Hour),
			}),
			// Flaky test with no verdicts in time range.
			insert.AnalyzedTestVariant(realm, tID3, vh, pb.AnalyzedTestVariantStatus_FLAKY, map[string]interface{}{
				"Variant":          pbutil.Variant("builder", "Linux Tests"),
				"Tags":             tags,
				"TestMetadata":     span.Compressed(tmdM),
				"StatusUpdateTime": start.Add(-time.Hour),
			}),
			// Test variant with another status is not exported.
			insert.AnalyzedTestVariant(realm, "ninja://test4", vh, pb.AnalyzedTestVariantStatus_CONSISTENTLY_UNEXPECTED, map[string]interface{}{
				"Variant":          pbutil.Variant("builder", "Linux Tests"),
				"Tags":             tags,
				"TestMetadata":     span.Compressed(tmdM),
				"StatusUpdateTime": start.Add(-time.Hour),
			}),
			insert.Verdict(realm, tID, vh, "build-0", pb.VerdictStatus_EXPECTED, twoAndHalfHAgo, map[string]interface{}{
				"IngestionTime":         oneAndHalfHAgo,
				"UnexpectedResultCount": 0,
				"TotalResultCount":      1,
			}),
			insert.Verdict(realm, tID, vh, "build-1", pb.VerdictStatus_VERDICT_FLAKY, twoAndHalfHAgo, map[string]interface{}{
				"IngestionTime":         halfHAgo,
				"UnexpectedResultCount": 1,
				"TotalResultCount":      2,
			}),
			insert.Verdict(realm, tID, vh, "build-2", pb.VerdictStatus_EXPECTED, oneAndHalfHAgo, map[string]interface{}{
				"IngestionTime":         halfHAgo,
				"UnexpectedResultCount": 0,
				"TotalResultCount":      1,
			}),
		}
		testutil.MustApply(ctx, ms...)

		op := &Options{
			Realm:        realm,
			CloudProject: "cloud_project",
			Dataset:      "dataset",
			Table:        "table",
			Predicate: &pb.AnalyzedTestVariantPredicate{
				Status: pb.AnalyzedTestVariantStatus_FLAKY,
			},
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

		Convey("insert succeeded", func() {
			ins := &mockPassInserter{}
			err := br.exportTestVariantRows(ctx, ins)
			So(err, ShouldBeNil)

			rows := make([]*bqpb.TestVariantRow, len(ins.insertedMessages))
			for i, m := range ins.insertedMessages {
				rows[i] = m.Message.(*bqpb.TestVariantRow)
			}

			expRows := []*bqpb.TestVariantRow{
				{
					Name:         testVariantName(realm, tID3, vh),
					Realm:        realm,
					TestId:       tID3,
					VariantHash:  vh,
					Variant:      pbutil.StringPairs("builder", "Linux Tests"),
					Tags:         tags,
					TestMetadata: tmd,
					TimeRange: &pb.TimeRange{
						Earliest: op.TimeRange.Earliest,
						Latest:   op.TimeRange.Latest,
					},
					Status: "FLAKY",
					FlakeStatistics: &pb.FlakeStatistics{
						FlakyVerdictRate:      0.0,
						FlakyVerdictCount:     int64(0),
						TotalVerdictCount:     int64(0),
						UnexpectedResultRate:  0.0,
						UnexpectedResultCount: int64(0),
						TotalResultCount:      int64(0),
					},
					PartitionTime: timestamppb.New(now),
				},
				{
					Name:         testVariantName(realm, tID, vh),
					Realm:        realm,
					TestId:       tID,
					VariantHash:  vh,
					Variant:      pbutil.StringPairs("builder", "Linux Tests"),
					Tags:         tags,
					TestMetadata: tmd,
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
						{
							Invocation: "build-2",
							Status:     "EXPECTED",
							CreateTime: timestamppb.New(oneAndHalfHAgo),
						},
						{
							Invocation: "build-1",
							Status:     "VERDICT_FLAKY",
							CreateTime: timestamppb.New(twoAndHalfHAgo),
						},
					},
					PartitionTime: timestamppb.New(now),
				},
			}
			So(rows, ShouldResembleProto, expRows)
		})

	})
}
