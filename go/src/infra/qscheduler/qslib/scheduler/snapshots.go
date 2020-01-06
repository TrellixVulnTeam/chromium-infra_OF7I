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

package scheduler

import (
	"infra/qscheduler/qslib/protos/metrics"
	"sort"
)

// Transfer the state struct to metrics.SchedulerState proto.
func (s *state) toMetricsSchedulerState(poolID string) *metrics.SchedulerState {
	accounts := make([]*metrics.SchedulerState_Account, 0, len(s.balances))
	for aid, bal := range s.balances {
		accounts = append(accounts, toMetricsAccount(aid, bal))
	}

	queuedTasks := make([]*metrics.SchedulerState_Task, 0, len(s.queuedRequests))
	for _, rq := range s.queuedRequests {
		queuedTasks = append(queuedTasks, toMetricsTask(rq))
	}

	runningTasks := make([]*metrics.SchedulerState_Task, 0, len(s.workers))
	runningWorkers := make([]*metrics.SchedulerState_Worker, 0, len(s.workers))
	idleWorkers := make([]*metrics.SchedulerState_Worker, 0, len(s.workers))
	for _, worker := range s.workers {
		if worker.IsIdle() {
			idleWorkers = append(idleWorkers, toMetricsWorker(worker))
			continue
		}
		runningWorkers = append(runningWorkers, toMetricsWorker(worker))
		runningTasks = append(runningTasks, toMetricsTask(worker.runningTask.request))
	}

	return &metrics.SchedulerState{
		QueuedTasks:    sortTaskByID(queuedTasks),
		RunningTasks:   sortTaskByID(runningTasks),
		IdleWorkers:    sortWorkerByID(idleWorkers),
		RunningWorkers: sortWorkerByID(runningWorkers),
		Accounts:       sortAccountByID(accounts),
		PoolId:         poolID,
	}
}

func toMetricsAccount(id AccountID, balance Balance) *metrics.SchedulerState_Account {
	bCopy := balance
	return &metrics.SchedulerState_Account{Id: string(id), Balance: bCopy[:]}
}

func toMetricsTask(rq *TaskRequest) *metrics.SchedulerState_Task {
	return &metrics.SchedulerState_Task{
		Id:                  string(rq.ID),
		AccountId:           string(rq.AccountID),
		BaseLabels:          sortLabels(rq.BaseLabels),
		ProvisionableLabels: sortLabels(rq.ProvisionableLabels),
	}
}

func toMetricsWorker(w *Worker) *metrics.SchedulerState_Worker {
	var taskID string
	if !w.IsIdle() {
		taskID = string(w.runningTask.request.ID)
	}
	return &metrics.SchedulerState_Worker{
		Id:     string(w.ID),
		Labels: sortLabels(w.Labels.ToSlice()),
		TaskId: taskID,
	}
}

func sortAccountByID(accounts []*metrics.SchedulerState_Account) []*metrics.SchedulerState_Account {
	sort.Slice(accounts, func(i, j int) bool {
		return accounts[i].GetId() < accounts[j].GetId()
	})
	return accounts
}

func sortTaskByID(tasks []*metrics.SchedulerState_Task) []*metrics.SchedulerState_Task {
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].GetId() < tasks[j].GetId()
	})
	return tasks
}

func sortWorkerByID(workers []*metrics.SchedulerState_Worker) []*metrics.SchedulerState_Worker {
	sort.Slice(workers, func(i, j int) bool {
		return workers[i].GetId() < workers[j].GetId()
	})
	return workers
}

// A wrapper to sort labels.
func sortLabels(labels []string) []string {
	sort.Strings(labels)
	return labels
}
