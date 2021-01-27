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
	"context"
	"fmt"
	"net/url"
	"strings"

	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/gae/service/taskqueue"
)

const repairBotsQueue = "repair-bots"
const resetBotsQueue = "reset-bots"
const repairLabstationQueue = "repair-labstations"
const auditBotsQueue = "audit-bots"

// PushRepairLabstations pushes BOT ids to taskqueue repairLabstationQueue for
// upcoming repair jobs.
func PushRepairLabstations(ctx context.Context, botIDs []string) error {
	return pushDUTs(ctx, repairLabstationQueue, createTasks(botIDs, labstationRepairTask))
}

// PushRepairDUTs pushes BOT ids to taskqueue repairBotsQueue for upcoming repair
// jobs.
func PushRepairDUTs(ctx context.Context, botIDs []string) error {
	return pushDUTs(ctx, repairBotsQueue, createTasks(botIDs, crosRepairTask))
}

// PushResetDUTs pushes BOT ids to taskqueue resetBotsQueue for upcoming reset
// jobs.
func PushResetDUTs(ctx context.Context, botIDs []string) error {
	return pushDUTs(ctx, resetBotsQueue, createTasks(botIDs, resetTask))
}

// PushAuditDUTs pushes BOT ids to taskqueue auditBotsQueue for upcoming audit jobs.
func PushAuditDUTs(ctx context.Context, botIDs, actions []string) error {
	actionsCSV := strings.Join(actions, ",")
	actionsStr := strings.Join(actions, "-")
	tasks := make([]*taskqueue.Task, 0, len(botIDs))
	for _, id := range botIDs {
		tasks = append(tasks, crosAuditTask(id, actionsCSV, actionsStr))
	}
	return pushDUTs(ctx, auditBotsQueue, tasks)
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

func crosAuditTask(botID, actionsCSV, actionsStr string) *taskqueue.Task {
	values := url.Values{}
	values.Set("botID", botID)
	values.Set("actions", actionsCSV)
	return taskqueue.NewPOSTTask(fmt.Sprintf("/internal/task/audit/%s/%s", botID, actionsStr), values)
}

func createTasks(botIDs []string, taskGenerator func(string) *taskqueue.Task) []*taskqueue.Task {
	tasks := make([]*taskqueue.Task, 0, len(botIDs))
	for _, id := range botIDs {
		tasks = append(tasks, taskGenerator(id))
	}
	return tasks
}
func pushDUTs(ctx context.Context, queueName string, tasks []*taskqueue.Task) error {
	if err := taskqueue.Add(ctx, queueName, tasks...); err != nil {
		return err
	}
	logging.Infof(ctx, "enqueued %d tasks", len(tasks))
	return nil
}
