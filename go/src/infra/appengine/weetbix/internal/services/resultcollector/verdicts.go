// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package resultcollector

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/spanner"
	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/resultdb/pbutil"
	rdbpb "go.chromium.org/luci/resultdb/proto/v1"
	"go.chromium.org/luci/server/span"

	"infra/appengine/weetbix/internal"
	spanutil "infra/appengine/weetbix/internal/span"
	"infra/appengine/weetbix/internal/tasks/taskspb"
)

func createVerdicts(ctx context.Context, task *taskspb.CollectTestResults, tvs []*rdbpb.TestVariant) error {
	ms := make([]*spanner.Mutation, 0, len(tvs))
	// Each batch of verdicts use the same ingestion time.
	now := clock.Now(ctx)
	for _, tv := range tvs {
		if tv.Status == rdbpb.TestVariantStatus_UNEXPECTEDLY_SKIPPED {
			continue
		}
		m := insertVerdict(task, tv, now)
		if m == nil {
			continue
		}
		ms = append(ms, m)
	}
	_, err := span.ReadWriteTransaction(ctx, func(ctx context.Context) error {
		span.BufferWrite(ctx, ms...)
		return nil
	})
	return err
}

func insertVerdict(task *taskspb.CollectTestResults, tv *rdbpb.TestVariant, ingestionTime time.Time) *spanner.Mutation {
	inv := task.Resultdb.Invocation
	invId, err := pbutil.ParseInvocationName(inv.Name)
	if err != nil {
		// This should never happen:inv was originally from ResultDB.
		panic(err)
	}
	row := map[string]interface{}{
		"Realm":                        inv.Realm,
		"InvocationId":                 invId,
		"InvocationCreationTime":       inv.CreateTime,
		"IngestionTime":                ingestionTime,
		"TestId":                       tv.TestId,
		"VariantHash":                  tv.VariantHash,
		"Status":                       deriveVerdictStatus(tv),
		"Exonerated":                   tv.Status == rdbpb.TestVariantStatus_EXONERATED,
		"IsPreSubmit":                  task.IsPreSubmit,
		"HasContributedToClSubmission": task.ContributedToClSubmission,
	}
	row["UnexpectedResultCount"], row["TotalResultCount"] = countResults(tv)

	if row["TotalResultCount"] == 0 {
		// No results in the verdict can be counted (skips?), so no need to save
		// this verdict.
		return nil
	}
	return spanner.InsertOrUpdateMap("Verdicts", spanutil.ToSpannerMap(row))
}

func deriveVerdictStatus(tv *rdbpb.TestVariant) internal.VerdictStatus {
	switch tv.Status {
	case rdbpb.TestVariantStatus_FLAKY:
		return internal.VerdictStatus_VERDICT_FLAKY
	case rdbpb.TestVariantStatus_EXPECTED:
		return internal.VerdictStatus_EXPECTED
	case rdbpb.TestVariantStatus_UNEXPECTED:
		return internal.VerdictStatus_UNEXPECTED
	case rdbpb.TestVariantStatus_EXONERATED:
		return internal.VerdictStatus_UNEXPECTED
	default:
		panic(fmt.Sprintf("impossible verdict status: %d", tv.Status))
	}
}

func countResults(tv *rdbpb.TestVariant) (unexpected, total int64) {
	for _, trb := range tv.Results {
		tr := trb.Result
		if tr.Status == rdbpb.TestStatus_SKIP {
			// Skips are not counted into total nor unexpected.
			continue
		}
		total++
		if !tr.Expected && tr.Status != rdbpb.TestStatus_PASS {
			// Count unexpected failures.
			unexpected++
		}
	}
	return
}
