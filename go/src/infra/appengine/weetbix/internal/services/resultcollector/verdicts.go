// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package resultcollector

import (
	"context"
	"fmt"

	"cloud.google.com/go/spanner"
	"go.chromium.org/luci/resultdb/pbutil"
	rdbpb "go.chromium.org/luci/resultdb/proto/v1"
	"go.chromium.org/luci/server/span"

	spanutil "infra/appengine/weetbix/internal/span"
	"infra/appengine/weetbix/internal/tasks/taskspb"
	pb "infra/appengine/weetbix/proto/v1"
)

func createVerdicts(ctx context.Context, task *taskspb.CollectTestResults, tvs []*rdbpb.TestVariant) error {
	ms := make([]*spanner.Mutation, 0, len(tvs))
	for _, tv := range tvs {
		m := insertVerdict(task, tv)
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

func insertVerdict(task *taskspb.CollectTestResults, tv *rdbpb.TestVariant) *spanner.Mutation {
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

func deriveVerdictStatus(tv *rdbpb.TestVariant) pb.VerdictStatus {
	switch tv.Status {
	case rdbpb.TestVariantStatus_FLAKY:
		return pb.VerdictStatus_VERDICT_FLAKY
	case rdbpb.TestVariantStatus_EXPECTED:
		return pb.VerdictStatus_EXPECTED
	case rdbpb.TestVariantStatus_UNEXPECTED:
		return pb.VerdictStatus_UNEXPECTED
	case rdbpb.TestVariantStatus_EXONERATED:
		return pb.VerdictStatus_UNEXPECTED
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
