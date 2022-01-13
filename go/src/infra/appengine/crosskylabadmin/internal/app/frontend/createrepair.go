// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"context"
	"fmt"
	"math"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"

	fleet "infra/appengine/crosskylabadmin/api/fleet/v1"
	"infra/appengine/crosskylabadmin/internal/app/clients"
	"infra/appengine/crosskylabadmin/internal/app/config"
	"infra/appengine/crosskylabadmin/internal/app/frontend/internal/worker"
	"infra/libs/skylab/common/heuristics"
)

const paris = "paris"

// RouteRepairTask routes a repair task for a given bot.
//
// The possible return values are:
// - ""      (for legacy, which is the default)
// - "paris" (for PARIS, which is new)
//
// RouteRepairTask takes as an argument randFloat (which is a float64 in the closed interval [0, 1]).
// This argument is, by design, all the entropy that randFloat will need. Taking this as an argument allows
// RouteRepairTask itself to be deterministic because the caller is responsible for generating the random
// value.
//
// TODO(gregorynisbet): This function is not finished; we need to take the labstation pool into account as well.
func RouteRepairTask(ctx context.Context, botID string, expectedState string, randFloat float64) (string, error) {
	if !(0.0 <= randFloat && randFloat <= 1.0) {
		return "", fmt.Errorf("route repair task: randfloat %f is not in [0, 1]", randFloat)
	}
	parisCfg := config.Get(ctx).GetParis()
	if !heuristics.LooksLikeLabstation(botID) {
		logging.Infof(ctx, "Non-labstations always use legacy flow")
		return "", nil
	}
	if !parisCfg.GetEnableLabstationRecovery() {
		logging.Infof(ctx, "Labstation recovery is not enabled at all")
		return "", nil
	}
	if parisCfg.GetOptinAllLabstations() {
		return paris, nil
	}
	threshold := parisCfg.GetLabstationRecoveryPermille()
	if threshold == 0 {
		return "", fmt.Errorf("route repair task: a threshold of zero implies that optinAllLabstations should be set, but optinAllLabstations is not set")
	}
	// If we make it this far, then it's possible for us to use the new flow.
	// However, we should only actually do it if our value exceeds the threshold.
	myValue := math.Round(1000.0 * randFloat)
	if myValue >= float64(threshold) {
		return paris, nil
	}
	return "", nil
}

// CreateRepairTask kicks off a repair job.
//
// This function will either schedule a legacy repair task or a PARIS repair task.
func CreateRepairTask(ctx context.Context, botID string, expectedState string, randFloat float64) (string, error) {
	logging.Infof(ctx, "Creating repair task for %q expected state %q with random input %f", botID, expectedState, randFloat)
	// If we encounter an error picking paris or legacy, do the safe thing and use legacy.
	taskType, err := RouteRepairTask(ctx, botID, expectedState, randFloat)
	if err != nil {
		logging.Infof(ctx, "Create repair task: falling back to legacy repair by default: %s", err)
	}
	switch taskType {
	case "paris":
		return createBuildbucketRepairTask(ctx, botID, expectedState)
	default:
		return createLegacyRepairTask(ctx, botID, expectedState)
	}
}

// CreateBuildbucketRepairTask creates a new repair task for a labstation. Not yet implemented.
func createBuildbucketRepairTask(ctx context.Context, botID string, expectedState string) (string, error) {
	logging.Infof(ctx, "Using new repair flow for bot %q", botID)
	return "", fmt.Errorf("not yet implemented")
}

// CreateLegacyRepairTask creates a legacy repair task for a labstation.
func createLegacyRepairTask(ctx context.Context, botID string, expectedState string) (string, error) {
	logging.Infof(ctx, "Using legacy repair flow for bot %q", botID)
	at := worker.AdminTaskForType(ctx, fleet.TaskType_Repair)
	sc, err := clients.NewSwarmingClient(ctx, config.Get(ctx).Swarming.Host)
	if err != nil {
		return "", errors.Annotate(err, "failed to obtain swarming client").Err()
	}
	cfg := config.Get(ctx)
	taskURL, err := runTaskByBotID(ctx, at, sc, botID, expectedState, cfg.Tasker.BackgroundTaskExpirationSecs, cfg.Tasker.BackgroundTaskExecutionTimeoutSecs)
	if err != nil {
		return "", errors.Annotate(err, "fail to create repair task for %s", botID).Err()
	}
	return taskURL, nil
}

// IsRecoveryEnabledForLabstation returns whether recovery is enabled for labstations.
// TODO(gregorynisbet): Expand this to take into account other relevant factors like the pools that the labstation is in.
func isRecoveryEnabledForLabstation(ctx context.Context) bool {
	enabled := true
	enabled = enabled && config.Get(ctx).GetParis().GetEnableLabstationRecovery()
	return enabled
}
