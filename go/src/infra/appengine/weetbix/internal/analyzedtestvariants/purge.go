// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package analyzedtestvariants

import (
	"context"

	"cloud.google.com/go/spanner"

	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/server/span"

	pb "infra/appengine/weetbix/proto/v1"
)

func purge(ctx context.Context) (int64, error) {
	st := spanner.NewStatement(`
		DELETE FROM AnalyzedTestVariants
		WHERE Status in UNNEST(@statuses)
		AND StatusUpdateTime < TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 31 DAY)
	`)
	st.Params = map[string]interface{}{
		"statuses": []int{int(pb.AnalyzedTestVariantStatus_NO_NEW_RESULTS), int(pb.AnalyzedTestVariantStatus_CONSISTENTLY_EXPECTED)},
	}
	return span.PartitionedUpdate(ctx, st)
}

// Purge deletes AnalyzedTestVariants rows that have been in NO_NEW_RESULTS or
// CONSISTENTLY_EXPECTED status for over 1 month.
//
// Because Verdicts are interleaved with AnalyzedTestVariants, deleting
// AnalyzedTestVariants rows also deletes their verdicts.
func Purge(ctx context.Context) error {
	purged, err := purge(ctx)
	logging.Infof(ctx, "Purged %d test variants", purged)
	return err
}
