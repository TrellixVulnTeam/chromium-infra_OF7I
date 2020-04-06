// Copyright 2019 The LUCI Authors.
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

package clients

import (
	"fmt"
	"net/url"

	"go.chromium.org/gae/service/taskqueue"
	"go.chromium.org/luci/common/logging"
	"golang.org/x/net/context"
)

const repairBotsQueue = "repair-bots"
const resetBotsQueue = "reset-bots"
const repairLabstationQueue = "repair-labstations"

// PushRepairLabstations pushes BOT ids to taskqueue repairLabstationQueue for
// upcoming repair jobs.
func PushRepairLabstations(ctx context.Context, botIDs []string) error {
	return pushDUTs(ctx, botIDs, repairLabstationQueue, labstationRepairTask)
}

// PushRepairDUTs pushes BOT ids to taskqueue repairBotsQueue for upcoming repair
// jobs.
func PushRepairDUTs(ctx context.Context, botIDs []string) error {
	return pushDUTs(ctx, botIDs, repairBotsQueue, crosRepairTask)
}

// PushResetDUTs pushes BOT ids to taskqueue resetBotsQueue for upcoming reset
// jobs.
func PushResetDUTs(ctx context.Context, botIDs []string) error {
	return pushDUTs(ctx, botIDs, resetBotsQueue, resetTask)
}

func crosRepairTask(botID string) *taskqueue.Task {
	values := url.Values{}
	values.Set("botID", botID)
	return taskqueue.NewPOSTTask(fmt.Sprintf("/internal/task/cros_repair/%s", botID), values)
}

func labstationRepairTask(botID string) *taskqueue.Task {
	values := url.Values{}
	values.Set("botID", botID)
	return taskqueue.NewPOSTTask(fmt.Sprintf("/internal/task/labstation_repair/%s", botID), values)
}

func resetTask(botID string) *taskqueue.Task {
	values := url.Values{}
	values.Set("botID", botID)
	return taskqueue.NewPOSTTask(fmt.Sprintf("/internal/task/reset/%s", botID), values)
}

func pushDUTs(ctx context.Context, botIDs []string, queueName string, taskGenerator func(string) *taskqueue.Task) error {
	tasks := make([]*taskqueue.Task, 0, len(botIDs))
	for _, id := range botIDs {
		tasks = append(tasks, taskGenerator(id))
	}
	if err := taskqueue.Add(ctx, queueName, tasks...); err != nil {
		return err
	}
	logging.Infof(ctx, "enqueued %d tasks", len(tasks))
	return nil
}
