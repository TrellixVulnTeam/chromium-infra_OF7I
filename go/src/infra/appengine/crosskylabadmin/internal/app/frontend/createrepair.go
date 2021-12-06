// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"context"
	"fmt"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"

	fleet "infra/appengine/crosskylabadmin/api/fleet/v1"
	"infra/appengine/crosskylabadmin/internal/app/clients"
	"infra/appengine/crosskylabadmin/internal/app/config"
	"infra/appengine/crosskylabadmin/internal/app/frontend/internal/worker"
	"infra/libs/skylab/common/heuristics"
)

// CreateRepairTask kicks off a repair job.
// This function will either schedule a legacy repair task or a PARIS repair task.
//
// TODO(gregorynisbet): Make rules more complicated.
func CreateRepairTask(ctx context.Context, botID, expectedState string) (string, error) {
	logging.Infof(ctx, "Creating repair task for %q expected state %q", botID, expectedState)
	useBuildbucketFlow := false
	labstationRecoveryEnabled := isRecoveryEnabledForLabstation(ctx)
	if heuristics.LooksLikeLabstation(botID) && labstationRecoveryEnabled {
		useBuildbucketFlow = true
	}
	if useBuildbucketFlow {
		return createBuildbucketRepairTask(ctx, botID, expectedState)
	}
	return createLegacyRepairTask(ctx, botID, expectedState)
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
