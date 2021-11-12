// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package analyzedtestvariants

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/spanner"

	"go.chromium.org/luci/server/span"

	spanutil "infra/appengine/weetbix/internal/span"
	pb "infra/appengine/weetbix/proto/v1"
)

// ReadStatus reads AnalyzedTestVariant rows by keys.
func ReadStatus(ctx context.Context, ks spanner.KeySet, f func(*pb.AnalyzedTestVariant) error) error {
	fields := []string{"Realm", "TestId", "VariantHash", "Status"}
	var b spanutil.Buffer
	return span.Read(ctx, "AnalyzedTestVariants", ks, fields).Do(
		func(row *spanner.Row) error {
			tv := &pb.AnalyzedTestVariant{}
			if err := b.FromSpanner(row, &tv.Realm, &tv.TestId, &tv.VariantHash, &tv.Status); err != nil {
				return err
			}
			return f(tv)
		},
	)
}

// StatusHistory contains all the information related to a test variant's status changes.
type StatusHistory struct {
	Status                    pb.AnalyzedTestVariantStatus
	StatusUpdateTime          time.Time
	PreviousStatuses          []pb.AnalyzedTestVariantStatus
	PreviousStatusUpdateTimes []time.Time
}

// ReadStatusHistory reads AnalyzedTestVariant rows by keys and returns the test variant's status related info.
func ReadStatusHistory(ctx context.Context, k spanner.Key) (*StatusHistory, spanner.NullTime, error) {
	fields := []string{"Status", "StatusUpdateTime", "NextUpdateTaskEnqueueTime", "PreviousStatuses", "PreviousStatusUpdateTimes"}
	var b spanutil.Buffer
	si := &StatusHistory{}
	var enqTime, t spanner.NullTime
	err := span.Read(ctx, "AnalyzedTestVariants", spanner.KeySets(k), fields).Do(
		func(row *spanner.Row) error {
			if err := b.FromSpanner(row, &si.Status, &t, &enqTime, &si.PreviousStatuses, &si.PreviousStatusUpdateTimes); err != nil {
				return err
			}
			if !t.Valid {
				return fmt.Errorf("invalid status update time")
			}
			si.StatusUpdateTime = t.Time
			return nil
		},
	)
	return si, enqTime, err
}

// ReadNextUpdateTaskEnqueueTime reads the NextUpdateTaskEnqueueTime from the
// requested test variant.
func ReadNextUpdateTaskEnqueueTime(ctx context.Context, k spanner.Key) (spanner.NullTime, error) {
	row, err := span.ReadRow(ctx, "AnalyzedTestVariants", k, []string{"NextUpdateTaskEnqueueTime"})
	if err != nil {
		return spanner.NullTime{}, err
	}

	var t spanner.NullTime
	err = row.Column(0, &t)
	return t, err
}

// QueryTestVariantsByBuilder queries AnalyzedTestVariants with unexpected
// results on the given builder.
func QueryTestVariantsByBuilder(ctx context.Context, realm, builder string, f func(*pb.AnalyzedTestVariant) error) error {
	st := spanner.NewStatement(`
		SELECT TestId, VariantHash
		FROM AnalyzedTestVariants@{FORCE_INDEX=AnalyzedTestVariantsByBuilderAndStatus, spanner_emulator.disable_query_null_filtered_index_check=true}
		WHERE Realm = @realm
		AND Builder = @builder
		AND Status in UNNEST(@statuses)
		ORDER BY TestId, VariantHash
	`)
	st.Params = map[string]interface{}{
		"realm":    realm,
		"builder":  builder,
		"statuses": []int{int(pb.AnalyzedTestVariantStatus_FLAKY), int(pb.AnalyzedTestVariantStatus_CONSISTENTLY_UNEXPECTED), int(pb.AnalyzedTestVariantStatus_HAS_UNEXPECTED_RESULTS)},
	}

	var b spanutil.Buffer
	return span.Query(ctx, st).Do(
		func(row *spanner.Row) error {
			tv := &pb.AnalyzedTestVariant{}
			if err := b.FromSpanner(row, &tv.TestId, &tv.VariantHash); err != nil {
				return err
			}
			return f(tv)
		},
	)
}
