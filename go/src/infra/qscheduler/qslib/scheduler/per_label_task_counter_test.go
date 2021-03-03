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
	"fmt"
	"github.com/google/go-cmp/cmp"
	"go.chromium.org/luci/common/data/stringset"
	"testing"
)

var testCountData = []struct {
	accountConfigs map[AccountID]*AccountConfig
	newLabels      stringset.Set
	wantTaskCount  map[AccountID]tasksPerLabelKeyVal
}{
	{ // Account not found
		nil,
		stringset.NewFromSlice("foo:bar"),
		map[AccountID]tasksPerLabelKeyVal{},
	},
	{ // No labels limited
		map[AccountID]*AccountConfig{"test-id": {}},
		stringset.NewFromSlice("foo:bar"),
		map[AccountID]tasksPerLabelKeyVal{"test-id": {}},
	},
	{ // Labels limited but not found
		map[AccountID]*AccountConfig{
			"test-id": {PerLabelTaskLimits: map[string]int32{"baz": 2}},
		},
		stringset.NewFromSlice("foo:bar"),
		map[AccountID]tasksPerLabelKeyVal{"test-id": {}},
	},
	{ // Labels limited and found
		map[AccountID]*AccountConfig{
			"test-id": {PerLabelTaskLimits: map[string]int32{"foo": 2}},
		},
		stringset.NewFromSlice("foo:bar"),
		map[AccountID]tasksPerLabelKeyVal{
			"test-id": {"foo:bar": 1},
		},
	},
}

func TestCount(t *testing.T) {
	t.Parallel()
	for _, tt := range testCountData {
		tt := tt
		t.Run(fmt.Sprintf("(%s)", tt.newLabels), func(t *testing.T) {
			tc := newPerLabelTaskCounter(&Config{
				AccountConfigs: tt.accountConfigs,
			})
			tc.count(tt.newLabels, "test-id")
			if diff := cmp.Diff(tt.wantTaskCount, tc.taskCount); diff != "" {
				t.Errorf("unexpected diff (%s)", diff)
			}
		})
	}
}

var testIsAtAnyLimitData = []struct {
	accountConfigs   map[AccountID]*AccountConfig
	currentTaskCount map[AccountID]tasksPerLabelKeyVal
	newLabels        stringset.Set
	wantAnswer       bool
}{
	{ // Account not found
		nil,
		map[AccountID]tasksPerLabelKeyVal{},
		stringset.NewFromSlice("foo:bar"),
		false,
	},
	{ // No labels limited
		map[AccountID]*AccountConfig{"test-id": {}},
		map[AccountID]tasksPerLabelKeyVal{},
		stringset.NewFromSlice("foo:bar"),
		false,
	},
	{ // Labels at limit but not found
		map[AccountID]*AccountConfig{
			"test-id": {PerLabelTaskLimits: map[string]int32{"foo": 2}},
		},
		map[AccountID]tasksPerLabelKeyVal{
			"test-id": {"foo:baz": 2},
		},
		stringset.NewFromSlice("foo:bar"),
		false,
	},
	{ // Labels found but not at limit
		map[AccountID]*AccountConfig{
			"test-id": {PerLabelTaskLimits: map[string]int32{"foo": 2}},
		},
		map[AccountID]tasksPerLabelKeyVal{
			"test-id": {"foo:bar": 1, "foo:baz": 2},
		},
		stringset.NewFromSlice("foo:bar"),
		false,
	},
	{ // Labels found but not at limit
		map[AccountID]*AccountConfig{
			"test-id": {PerLabelTaskLimits: map[string]int32{"foo": 2}},
		},
		map[AccountID]tasksPerLabelKeyVal{
			"test-id": {"foo:bar": 2},
		},
		stringset.NewFromSlice("foo:bar"),
		true,
	},
}

func TestIsAtAnyLimit(t *testing.T) {
	t.Parallel()
	for _, tt := range testIsAtAnyLimitData {
		tt := tt
		t.Run(fmt.Sprintf("(%s)", tt.newLabels), func(t *testing.T) {
			tc := newPerLabelTaskCounter(&Config{
				AccountConfigs: tt.accountConfigs,
			})
			tc.taskCount = tt.currentTaskCount

			gotAnswer := tc.isAtAnyLimit(tt.newLabels, "test-id")
			if tt.wantAnswer != gotAnswer {
				t.Errorf("unexpected error: wanted %v, got %v", tt.wantAnswer, gotAnswer)
			}
		})
	}
}
