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

package metrics

import (
	"sync"

	"go.chromium.org/luci/common/data/stringset"

	"infra/qscheduler/qslib/scheduler"
)

type labelUtilization struct {
	// RunningBots is the number of running bots that possess a given label.
	RunningBots uint32
	// IdleBots is the number of idle bots that possess a given label.
	IdleBots uint32

	// RunningRequests is the number of running task requests that
	// requested a given label.
	RunningRequests uint32
	// WaitingRequests is the number of waiting tasks that requested a
	// given label.
	WaitingRequests uint32
}

// labelUtilizationCounter computes utilization rate for labels that have
// been requested as non-provisionable labels by tasks.
//
// When sending gauge-based metrics with a dynamic collection of field values,
// a common problem encountered is that an application's logic may stop setting
// values for a field that it previously had a value for, if that field value
// disappears. Monarch interprets this by holding the metric for that field at
// it's old value, rather than setting it to zero. This structure avoids that by
// keeping cache of all field values (in this case, labels) that have been sent
// and explicitly setting their metric values to zero in the absence of another
// value.
//
// labelUtilizationCounter is concurrency safe.
type labelUtilizationCounter struct {
	// lock is held while mutating labelUtilizationCounter
	lock sync.Mutex

	// knownLabels caches all the non-provisionable labels that have been
	// encountered for a particular scheduler_id.
	knownLabels map[string]stringset.Set
}

func newUtilizationCounter() *labelUtilizationCounter {
	return &labelUtilizationCounter{
		knownLabels: make(map[string]stringset.Set),
	}
}

// labelDigest is simplified representation of the tasks and labels
// contained in a scheduler. This is used to aid testability of
// labelUtilizationCounter
type labelDigest struct {
	WaitingTaskBaseLabels [][]string
	RunningTaskBaseLabels [][]string
	IdleWorkerLabels      []stringset.Set
	RunningWorkerLabels   []stringset.Set
}

// digest converts a scheduler's label information into a simplified labelDigest.
func digest(s *scheduler.Scheduler) labelDigest {
	var d labelDigest
	for _, task := range s.GetWaitingRequests() {
		d.WaitingTaskBaseLabels = append(d.WaitingTaskBaseLabels, task.BaseLabels)
	}

	for _, w := range s.GetWorkers() {
		if w.IsIdle() {
			d.IdleWorkerLabels = append(d.IdleWorkerLabels, w.Labels)
		} else {
			d.RunningWorkerLabels = append(d.RunningWorkerLabels, w.Labels)
			d.RunningTaskBaseLabels = append(d.RunningTaskBaseLabels, w.RunningRequest().BaseLabels)
		}
	}

	return d
}

// Compute returns the utilization rate of all non-provisionable labels for a
// given scheduler ID and its label digest.
//
// It is concurrency safe.
func (c *labelUtilizationCounter) Compute(schedulerID string, d labelDigest) map[string]*labelUtilization {
	c.lock.Lock()
	defer c.lock.Unlock()

	known := c.getKnownLabels(schedulerID, d)

	utils := make(map[string]*labelUtilization, len(known))
	for l := range known {
		utils[l] = &labelUtilization{}
	}

	for _, r := range d.RunningTaskBaseLabels {
		for _, l := range r {
			utils[l].RunningRequests++
		}
	}

	for _, r := range d.WaitingTaskBaseLabels {
		for _, l := range r {
			utils[l].WaitingRequests++
		}
	}

	for _, w := range d.IdleWorkerLabels {
		for l := range w {
			// Only count bot labels if they correspond to some task's base
			// labels.
			if known.Has(l) {
				utils[l].IdleBots++
			}
		}
	}

	for _, w := range d.RunningWorkerLabels {
		for l := range w {
			// Only count bot labels if they correspond to some task's base
			// labels.
			if known.Has(l) {
				utils[l].RunningBots++
			}
		}
	}

	return utils
}

// getKnownLabels updates and returns the known labels set for the given scheduler.
//
// Not concurrency safe. Caller must hold c.lock.
func (c *labelUtilizationCounter) getKnownLabels(schedulerID string, d labelDigest) stringset.Set {
	known, ok := c.knownLabels[schedulerID]
	if !ok {
		known = stringset.New(0)
		c.knownLabels[schedulerID] = known
	}

	for _, r := range d.WaitingTaskBaseLabels {
		for _, l := range r {
			known.Add(l)
		}
	}

	for _, r := range d.RunningTaskBaseLabels {
		for _, l := range r {
			known.Add(l)
		}
	}

	return known
}
