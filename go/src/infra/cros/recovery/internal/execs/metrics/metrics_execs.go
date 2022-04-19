// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package metrics

import (
	"context"
	"time"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/log"
	"infra/cros/recovery/logger/metrics"
)

const (
	// MetricsKind is the name/key stored in the karte metric.
	MetricsKind = "metrics_kind"
)

// metricsFoundAtLastTimeExec checks whether the specific actionKind has been found within the specified time window.
// By default, it is using DUT's ResourceName as the search parameter in Karte.
//
// @params: actionArgs should be in the format of:
// Ex: ["metrics_kind:x", "time_frame_hours:x"]
func metricsFoundAtLastTimeExec(ctx context.Context, info *execs.ExecInfo) error {
	argsMap := info.GetActionArgs(ctx)
	// metricsKind is the name/key stored in the karte metric.
	metricsKind := argsMap.AsString(ctx, MetricsKind, "")
	if metricsKind == "" {
		return errors.Reason("metrics found at last time: the provided metrics_kind is empty").Err()
	}
	// timeFrameHours is the time window for searching the last metric of this metricsKind being recorded.
	// Default to be 24h.
	timeFrameHours := argsMap.AsDuration(ctx, "time_frame_hours", 24, time.Hour)
	metric := info.RunArgs.Metrics
	if metric == nil {
		return errors.Reason("metrics found at last time: karte metric has not been initilized").Err()
	}
	karteQuery := &metrics.Query{
		// TODO: (@gregorynisbet): When karte' Search API is capable of taking in asset tag,
		// change the query to use asset tag instead of using hostname.
		Hostname:   info.RunArgs.DUT.Name,
		ActionKind: metricsKind,
	}
	queryRes, err := metric.Search(ctx, karteQuery)
	if err != nil {
		return errors.Annotate(err, "metrics found at last time").Err()
	}
	matchedQueryResCount := len(queryRes.Actions)
	if matchedQueryResCount == 0 {
		return errors.Reason("No match of the metrics kind: %q found in karte.", metricsKind).Err()
	}
	// Grabbing the most recent Karte response for this particular metrics kind.
	karteAction := queryRes.Actions[0]
	lastTime := karteAction.StopTime
	log.Infof(ctx, "Found last time: %v of metric kind: %q on the DUT: %v", lastTime, metricsKind)
	if time.Since(lastTime) < timeFrameHours {
		return nil
	}
	return errors.Reason("metrics found at last time: no metric kind of: %q found within the last %v", metricsKind, timeFrameHours).Err()
}

// checkTaskFailuresExec counts the number of times the task has
// failed, and verifies if it is greater than a threshold number.
func checkTaskFailuresExec(ctx context.Context, info *execs.ExecInfo) error {
	argsMap := info.GetActionArgs(ctx)
	taskName := argsMap.AsString(ctx, "task_name", "")
	repairFailedCountTarget := argsMap.AsInt(ctx, "repair_failed_count", 49)
	repairFailedCount, err := CountFailedRepairFromMetrics(ctx, taskName, info)
	if err != nil {
		return errors.Annotate(err, "check task failures").Err()
	}
	if repairFailedCount > repairFailedCountTarget {
		log.Infof(ctx, "The number of repair attempts %d exceeded the threshold of %d", repairFailedCount, repairFailedCountTarget)
		return nil
	}
	return errors.Reason("check task failures: Fail count: %d", repairFailedCount).Err()
}

func init() {
	execs.Register("metrics_found_at_last_time", metricsFoundAtLastTimeExec)
	execs.Register("metrics_check_task_failures", checkTaskFailuresExec)
}
