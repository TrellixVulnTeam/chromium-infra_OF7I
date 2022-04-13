// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package testverdictingester

import (
	"context"
	"time"

	"cloud.google.com/go/spanner"
	bbpb "go.chromium.org/luci/buildbucket/proto"
	"go.chromium.org/luci/common/sync/parallel"
	rdbpbutil "go.chromium.org/luci/resultdb/pbutil"
	rdbpb "go.chromium.org/luci/resultdb/proto/v1"
	"go.chromium.org/luci/server/span"

	"infra/appengine/weetbix/internal/tasks/taskspb"
	"infra/appengine/weetbix/internal/testverdicts"
	"infra/appengine/weetbix/pbutil"
	"infra/appengine/weetbix/utils"
)

func recordIngestedInvocation(ctx context.Context, task *taskspb.IngestTestVerdicts, build *bbpb.Build, inv *rdbpb.Invocation) error {
	invID, err := rdbpbutil.ParseInvocationName(inv.Name)
	if err != nil {
		// This should never happen. Inv was originated from ResultDB.
		panic(err)
	}

	proj, subRealm := utils.SplitRealm(inv.Realm)
	contributedToCLSubmission := task.GetPresubmitRun().GetPresubmitRunSucceeded()
	hasUnsubmittedChanges := len(build.GetInput().GetGerritChanges()) != 0

	// Update the IngestedInvocations table.
	ingestedInv := &testverdicts.IngestedInvocation{
		Project:                      proj,
		IngestedInvocationID:         invID,
		SubRealm:                     subRealm,
		PartitionTime:                task.PartitionTime.AsTime(),
		HasUnsubmittedChanges:        hasUnsubmittedChanges,
		HasContributedToClSubmission: contributedToCLSubmission,
	}
	_, err = span.ReadWriteTransaction(ctx, func(ctx context.Context) error {
		ingestedInv.SaveUnverified(ctx)
		return nil
	})
	return err
}

// recordTestVerdicts records test verdicts from an test-verdict-ingestion task.
func recordTestVerdicts(ctx context.Context, task *taskspb.IngestTestVerdicts, build *bbpb.Build, inv *rdbpb.Invocation, tvsC chan []*rdbpb.TestVariant) error {
	const (
		workerCount = 8
		batchSize   = 500
	)

	invId, err := rdbpbutil.ParseInvocationName(inv.Name)
	if err != nil {
		return err
	}

	proj, subRealm := utils.SplitRealm(inv.Realm)
	contributedToCLSubmission := task.GetPresubmitRun().GetPresubmitRunSucceeded()
	hasUnsubmittedChanges := len(build.GetInput().GetGerritChanges()) != 0

	// recordBatch updates TestVerdicts table and TestVariantRealms table from a
	// batch of test variants. Must be called in a spanner RW transactional
	// context.
	recordBatch := func(ctx context.Context, batch []*rdbpb.TestVariant) error {
		for _, tv := range batch {
			// Record the test verdict.
			expectedCount, unexpectedCount, skippedCount := countResults(tv)
			verdict := &testverdicts.TestVerdict{
				Project:                      proj,
				TestID:                       tv.TestId,
				PartitionTime:                task.PartitionTime.AsTime(),
				VariantHash:                  tv.VariantHash,
				IngestedInvocationID:         invId,
				SubRealm:                     subRealm,
				ExpectedCount:                expectedCount,
				UnexpectedCount:              unexpectedCount,
				SkippedCount:                 skippedCount,
				IsExonerated:                 tv.Status == rdbpb.TestVariantStatus_EXONERATED,
				PassedAvgDuration:            calcPassedAvgDuration(tv),
				HasUnsubmittedChanges:        hasUnsubmittedChanges,
				HasContributedToClSubmission: contributedToCLSubmission,
			}
			verdict.SaveUnverified(ctx)

			// Record the test variant realm.
			tvr := &testverdicts.TestVariantRealm{
				Project:           proj,
				TestID:            tv.TestId,
				VariantHash:       tv.VariantHash,
				SubRealm:          subRealm,
				Variant:           pbutil.VariantFromResultDB(tv.Variant),
				LastIngestionTime: spanner.CommitTimestamp,
			}
			tvr.SaveUnverified(ctx)
		}
		return nil
	}

	return parallel.WorkPool(workerCount, func(c chan<- func() error) {
		batchC := make(chan []*rdbpb.TestVariant, workerCount)

		// Split test variants into smaller batches so we have less than 20k
		// mutations in a single spanner transaction.
		c <- func() error {
			defer close(batchC)
			for tvs := range tvsC {
				for i := 0; i < len(tvs); i += batchSize {
					end := i + batchSize
					if end > len(tvs) {
						end = len(tvs)
					}
					batchC <- tvs[i:end]
				}
			}
			return nil
		}

		for batch := range tvsC {
			// Bind to a local variable so it can be used in a goroutine without being
			// overwritten. See https://go.dev/doc/faq#closures_and_goroutines
			batch := batch

			c <- func() error {
				_, err := span.ReadWriteTransaction(ctx, func(ctx context.Context) error {
					return recordBatch(ctx, batch)
				})
				return err
			}
		}
	})
}

func countResults(tv *rdbpb.TestVariant) (expected, unexpected, skipped int64) {
	for _, trb := range tv.Results {
		tr := trb.Result
		if tr.Status == rdbpb.TestStatus_SKIP {
			skipped++
		}
		if tr.Expected {
			expected++
		} else {
			unexpected++
		}
	}
	return
}

// calcPassedAvgDuration calculates the average duration of passed results.
// Return nil if there's no passed results.
func calcPassedAvgDuration(tv *rdbpb.TestVariant) *time.Duration {
	count := 0
	totalDuration := time.Duration(0)
	for _, trb := range tv.Results {
		tr := trb.Result
		if tr.Status != rdbpb.TestStatus_PASS {
			// Only calculate passed test results
			continue
		}
		count++
		totalDuration += tr.Duration.AsDuration()
	}
	if count == 0 {
		return nil
	}
	avgDuration := totalDuration / time.Duration(count)
	return &avgDuration
}
