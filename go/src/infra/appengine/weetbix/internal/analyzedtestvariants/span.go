// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package analyzedtestvariants

import (
	"context"

	"cloud.google.com/go/spanner"

	"go.chromium.org/luci/server/span"

	spanutil "infra/appengine/weetbix/internal/span"
	pb "infra/appengine/weetbix/proto/v1"
)

// Read reads AnalyzedTestVariant rows by keys.
func Read(ctx context.Context, ks spanner.KeySet, f func(*pb.AnalyzedTestVariant) error) error {
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

// QueryTestVariantsByBuilder queries AnalyzedTestVariants with unexpected
// results on the given builder.
func QueryTestVariantsByBuilder(ctx context.Context, realm, builder string, f func(*pb.AnalyzedTestVariant) error) error {
	st := spanner.NewStatement(`
		SELECT TestId, VariantHash
		FROM AnalyzedTestVariants@{FORCE_INDEX=AnalyzedTestVariantsPerBuilderAndStatus, spanner_emulator.disable_query_null_filtered_index_check=true}
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
