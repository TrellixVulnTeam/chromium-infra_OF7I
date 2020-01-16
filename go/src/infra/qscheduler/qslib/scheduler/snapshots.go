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
	"infra/qscheduler/qslib/tutils"
	"sort"
)

// Snapshot represents the scheduler state at a specified timestamp.
type Snapshot struct {
	Accounts []*metrics.Account
	Tasks    []*metrics.Task
	Workers  []*metrics.Worker
}

func (s *state) snapshot(poolID string) *Snapshot {
	return &Snapshot{
		s.accountSnapshot(poolID),
		s.taskSnapshot(poolID),
		s.workerSnapshot(poolID),
	}
}

func (s *state) accountSnapshot(PoolID string) []*metrics.Account {
	accounts := make([]*metrics.Account, 0, len(s.balances))
	for aid, bal := range s.balances {
		bCopy := bal
		accounts = append(accounts, &metrics.Account{
			Id:           &metrics.Account_ID{Name: string(aid)},
			Pool:         &metrics.Pool{Id: PoolID},
			Balance:      bCopy[:],
			SnapshotTime: tutils.TimestampProto(s.lastUpdateTime),
		})
	}
	return sortAccountByID(accounts)
}

func (s *state) taskSnapshot(PoolID string) []*metrics.Task {
	tasks := make([]*metrics.Task, 0, len(s.queuedRequests)+len(s.workers))
	for _, rq := range s.queuedRequests {
		tasks = append(tasks, &metrics.Task{
			Id:                  &metrics.Task_ID{Name: string(rq.ID)},
			Pool:                &metrics.Pool{Id: string(PoolID)},
			AccountId:           &metrics.Account_ID{Name: string(rq.AccountID)},
			WorkerId:            &metrics.Worker_ID{Name: ""},
			BaseLabels:          sortLabels(rq.BaseLabels),
			ProvisionableLabels: sortLabels(rq.ProvisionableLabels),
			SnapshotTime:        tutils.TimestampProto(s.lastUpdateTime),
		})
	}
	for _, w := range s.workers {
		if !w.IsIdle() {
			rq := w.runningTask.request
			tasks = append(tasks, &metrics.Task{
				Id:                  &metrics.Task_ID{Name: string(rq.ID)},
				Pool:                &metrics.Pool{Id: string(PoolID)},
				AccountId:           &metrics.Account_ID{Name: string(rq.AccountID)},
				WorkerId:            &metrics.Worker_ID{Name: string(w.ID)},
				BaseLabels:          sortLabels(rq.BaseLabels),
				ProvisionableLabels: sortLabels(rq.ProvisionableLabels),
				SnapshotTime:        tutils.TimestampProto(s.lastUpdateTime),
			})
		}
	}
	return sortTaskByID(tasks)
}

func (s *state) workerSnapshot(PoolID string) []*metrics.Worker {
	workers := make([]*metrics.Worker, 0, len(s.workers))
	for _, w := range s.workers {
		taskID := ""
		if !w.IsIdle() {
			taskID = string(w.runningTask.request.ID)
		}
		workers = append(workers, &metrics.Worker{
			Id:           &metrics.Worker_ID{Name: string(w.ID)},
			Pool:         &metrics.Pool{Id: string(PoolID)},
			TaskId:       &metrics.Task_ID{Name: taskID},
			Labels:       sortLabels(w.Labels.ToSlice()),
			SnapshotTime: tutils.TimestampProto(s.lastUpdateTime),
		})

	}
	return sortWorkerByID(workers)
}

func sortAccountByID(accounts []*metrics.Account) []*metrics.Account {
	sort.Slice(accounts, func(i, j int) bool {
		return accounts[i].GetId().GetName() < accounts[j].GetId().GetName()
	})
	return accounts
}

func sortTaskByID(tasks []*metrics.Task) []*metrics.Task {
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].GetId().GetName() < tasks[j].GetId().GetName()
	})
	return tasks
}

func sortWorkerByID(workers []*metrics.Worker) []*metrics.Worker {
	sort.Slice(workers, func(i, j int) bool {
		return workers[i].GetId().GetName() < workers[j].GetId().GetName()
	})
	return workers
}

// A wrapper to sort labels.
func sortLabels(labels []string) []string {
	sort.Strings(labels)
	return labels
}
