// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package verdicts

import (
	"context"
	"fmt"

	"cloud.google.com/go/spanner"
	"go.chromium.org/luci/server/span"
	"google.golang.org/api/iterator"
	"google.golang.org/protobuf/types/known/durationpb"

	"infra/appengine/weetbix/internal"
	spanutil "infra/appengine/weetbix/internal/span"
	"infra/appengine/weetbix/internal/tasks/taskspb"
	pb "infra/appengine/weetbix/proto/v1"
)

func statusCalculationDuration(du *durationpb.Duration) int {
	return int(du.AsDuration().Hours())
}

// ComputeTestVariantStatusFromVerdicts computes the test variant's status based
// on its verdicts within a time range.
//
// Currently the time range is the past one day, but should be configurable.
// TODO(crbug.com/1259374): Use the value in configurations.
func ComputeTestVariantStatusFromVerdicts(ctx context.Context, tvKey *taskspb.TestVariantKey, du *durationpb.Duration) (pb.AnalyzedTestVariantStatus, error) {
	st := spanner.NewStatement(`
		SELECT Status
		FROM Verdicts@{FORCE_INDEX=VerdictsByTestVariantAndIngestionTime, spanner_emulator.disable_query_null_filtered_index_check=true}
		WHERE Realm = @realm
		AND TestId = @testID
		AND VariantHash = @variantHash
		AND IngestionTime >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL @numHours HOUR)
	`)
	st.Params = map[string]interface{}{
		"realm":       tvKey.Realm,
		"testID":      tvKey.TestId,
		"variantHash": tvKey.VariantHash,
		"numHours":    statusCalculationDuration(du),
	}

	totalCount := 0
	unexpectedCount := 0

	itr := span.Query(ctx, st)
	defer itr.Stop()
	var b spanutil.Buffer
	for {
		row, err := itr.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return pb.AnalyzedTestVariantStatus_STATUS_UNSPECIFIED, err
		}
		var verdictStatus internal.VerdictStatus
		if err = b.FromSpanner(row, &verdictStatus); err != nil {
			return pb.AnalyzedTestVariantStatus_STATUS_UNSPECIFIED, err
		}

		totalCount++
		switch verdictStatus {
		case internal.VerdictStatus_VERDICT_FLAKY:
			// Any flaky verdict means the test variant is flaky.
			// Return status right away.
			itr.Stop()
			return pb.AnalyzedTestVariantStatus_FLAKY, nil
		case internal.VerdictStatus_UNEXPECTED:
			unexpectedCount++
		case internal.VerdictStatus_EXPECTED:
		default:
			panic(fmt.Sprintf("got unsupported verdict status %d", int(verdictStatus)))
		}
	}

	return computeTestVariantStatus(totalCount, unexpectedCount), nil
}

func computeTestVariantStatus(total, unexpected int) pb.AnalyzedTestVariantStatus {
	switch {
	case total == 0:
		// No new results of the test variant.
		return pb.AnalyzedTestVariantStatus_NO_NEW_RESULTS
	case unexpected == 0:
		return pb.AnalyzedTestVariantStatus_CONSISTENTLY_EXPECTED
	case unexpected == total:
		return pb.AnalyzedTestVariantStatus_CONSISTENTLY_UNEXPECTED
	default:
		return pb.AnalyzedTestVariantStatus_HAS_UNEXPECTED_RESULTS
	}
}
