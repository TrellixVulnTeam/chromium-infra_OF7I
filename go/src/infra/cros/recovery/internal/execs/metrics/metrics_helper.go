// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package metrics

import (
	"context"
	"fmt"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/logger/metrics"
)

// CountFailedRepairFromMetrics determines the number of failed PARIS repair task
// since the last successful PARIS repair task.
func CountFailedRepairFromMetrics(ctx context.Context, taskName string, info *execs.ExecInfo) (int, error) {
	metric := info.RunArgs.Metrics
	if metric == nil {
		return 0, errors.Reason("count failed repair from karte: karte metric has not been initialized").Err()
	}
	karteQuery := &metrics.Query{
		//TODO(gregorynisbet): When karte's Search API is capable of taking in asset tag,
		// change the query to use asset tag instead of using hostname.
		Hostname:   info.RunArgs.DUT.Name,
		ActionKind: fmt.Sprintf(metrics.PerResourceTaskKindGlob, taskName),
	}
	queryRes, err := metric.Search(ctx, karteQuery)
	if err != nil {
		return 0, errors.Annotate(err, "count failed repair from karte").Err()
	}
	matchedQueryResCount := len(queryRes.Actions)
	if matchedQueryResCount == 0 {
		return 0, nil
	}
	var failedRepairCount int
	for i := 0; i < matchedQueryResCount; i++ {
		if queryRes.Actions[i].Status == metrics.ActionStatusSuccess {
			// since we are counting the number of failed repair tasks after last successful task.
			// when we are encountering the successful record,that mean we reached latest success task
			// and we need stop counting it.
			break
		}
		failedRepairCount += 1
	}
	return failedRepairCount, nil
}
