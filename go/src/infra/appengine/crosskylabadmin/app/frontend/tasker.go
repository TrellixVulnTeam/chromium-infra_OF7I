// Copyright 2018 The LUCI Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package frontend

import (
	"fmt"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"golang.org/x/net/context"

	fleet "infra/appengine/crosskylabadmin/api/fleet/v1"
	"infra/appengine/crosskylabadmin/app/clients"
	"infra/appengine/crosskylabadmin/app/config"
	"infra/appengine/crosskylabadmin/app/frontend/internal/swarming"
	"infra/appengine/crosskylabadmin/app/frontend/internal/worker"
)

// CreateRepairTask kicks off a repair job.
func CreateRepairTask(ctx context.Context, dutName string) (string, error) {
	at := worker.AdminTaskForType(ctx, fleet.TaskType_Repair)
	sc, err := clients.NewSwarmingClient(ctx, config.Get(ctx).Swarming.Host)
	if err != nil {
		return "", errors.Annotate(err, "failed to obtain swarming client").Err()
	}
	taskURL, err := runTaskByDUTName(ctx, at, sc, dutName)
	if err != nil {
		return "", errors.Annotate(err, "fail to create repair task for %s", dutName).Err()
	}
	return taskURL, nil
}

// CreateResetTask kicks off a reset job.
func CreateResetTask(ctx context.Context, dutName string) (string, error) {
	at := worker.AdminTaskForType(ctx, fleet.TaskType_Reset)
	sc, err := clients.NewSwarmingClient(ctx, config.Get(ctx).Swarming.Host)
	if err != nil {
		return "", errors.Annotate(err, "failed to obtain swarming client").Err()
	}
	taskURL, err := runTaskByDUTName(ctx, at, sc, dutName)
	if err != nil {
		return "", errors.Annotate(err, "fail to create reset task for %s", dutName).Err()
	}
	return taskURL, nil
}

func runTaskByDUTName(ctx context.Context, at worker.Task, sc clients.SwarmingClient, dutName string) (string, error) {
	cfg := config.Get(ctx)
	tags := swarming.AddCommonTags(
		ctx,
		fmt.Sprintf("%s:%s", at.Name, dutName),
		fmt.Sprintf("task:%s", at.Name),
	)
	tags = append(tags, at.Tags...)
	a := swarming.SetCommonTaskArgs(ctx, &clients.SwarmingCreateTaskArgs{
		Cmd:                  at.Cmd,
		DutName:              dutName,
		ExecutionTimeoutSecs: cfg.Tasker.BackgroundTaskExecutionTimeoutSecs,
		ExpirationSecs:       cfg.Tasker.BackgroundTaskExpirationSecs,
		Priority:             cfg.Cron.FleetAdminTaskPriority,
		Tags:                 tags,
	})
	tid, err := sc.CreateTask(ctx, at.Name, a)
	if err != nil {
		return "", errors.Annotate(err, "failed to create task for dut %s", dutName).Err()
	}
	logging.Infof(ctx, "successfully kick off task %s for dut %s", tid, dutName)
	return swarming.URLForTask(ctx, tid), nil
}

var dutStateForTask = map[fleet.TaskType]string{
	fleet.TaskType_Cleanup: "needs_cleanup",
	fleet.TaskType_Repair:  "needs_repair",
	fleet.TaskType_Reset:   "needs_reset",
}
