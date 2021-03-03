// Copyright 2021 The LUCI Authors.
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
	"go.chromium.org/luci/common/data/stringset"
	"strings"
)

// tasksPerLabelKeyVal tracks how many tasks are currently running for unique
// label keyval combinations.
type tasksPerLabelKeyVal map[string]int32

// perLabelTaskCounter counts how many tasks per unique label keyval are
// currently running for each account that has per-label task limits set for
// any labels.
//
// NOTE: an individual perLabelTaskCounter is designed to be attached to an
// individual schedulerRun.
type perLabelTaskCounter struct {
	taskCount    map[AccountID]tasksPerLabelKeyVal
	configGetter accountConfigGetter
}

// newPerLabelTaskCounter initializes a perLabelTaskCounter, for attaching to
// a new schedulerRun.
func newPerLabelTaskCounter(config *Config) *perLabelTaskCounter {
	return &perLabelTaskCounter{
		taskCount:    map[AccountID]tasksPerLabelKeyVal{},
		configGetter: acGetter{config},
	}
}

// count increments the per-label task count for all labels configured with
// limits on the given account, with any matching label keyvals found in the
// given label set.
func (tc *perLabelTaskCounter) count(labels stringset.Set, id AccountID) {
	config, ok := tc.configGetter.getAccountConfig(id)
	if !ok {
		return
	}
	if tc.taskCount[id] == nil {
		tc.taskCount[id] = map[string]int32{}
	}
	for key := range config.PerLabelTaskLimits {
		if keyval, ok := findLabelVal(labels, key); ok {
			tc.taskCount[id][keyval]++
		}
	}
}

// isAtAnyLimit returns true if any per-label task limit on the given account
// is already reached for the given label keyvals.
func (tc *perLabelTaskCounter) isAtAnyLimit(labels stringset.Set, id AccountID) bool {
	config, ok := tc.configGetter.getAccountConfig(id)
	if !ok {
		return false
	}
	if tc.taskCount[id] == nil {
		tc.taskCount[id] = map[string]int32{}
	}
	for key, taskLimit := range config.PerLabelTaskLimits {
		keyval, ok := findLabelVal(labels, key)
		if !ok {
			continue
		}
		if tc.taskCount[id][keyval] >= taskLimit {
			return true
		}
	}
	return false
}

// findLabelVal returns the matching label keyval string for the given label
// key, or returns false if no label matching the key is found.
func findLabelVal(labels stringset.Set, key string) (string, bool) {
	for keyval := range labels {
		if strings.HasPrefix(keyval, key) {
			return keyval, true
		}
	}
	return "", false
}
