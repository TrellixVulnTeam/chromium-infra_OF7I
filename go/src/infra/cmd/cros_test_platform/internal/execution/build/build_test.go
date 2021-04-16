// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package build

import (
	"fmt"
	"infra/cmd/cros_test_platform/internal/execution/testrunner"
	"testing"

	buildbucketpb "go.chromium.org/luci/buildbucket/proto"

	"github.com/google/go-cmp/cmp"
)

var testSummaryData = []struct {
	tasks       []*testrunner.Build
	wantSummary string
}{
	{
		[]*testrunner.Build{
			testrunner.NewBuildForTesting("task1", "url1"),
			testrunner.NewBuildForTesting("task2", "url2"),
			testrunner.NewBuildForTesting("task3", "url3"),
		},
		`*    [latest attempt](url3)
*    previous failed attempts: [1](url1), [2](url2)`,
	},
	{
		[]*testrunner.Build{
			testrunner.NewBuildForTesting("task1", "url1"),
			testrunner.NewBuildForTesting("task2", "url2"),
		},
		`*    [latest attempt](url2)
*    previous failed attempts: [1](url1)`,
	},
	{
		[]*testrunner.Build{
			testrunner.NewBuildForTesting("task1", "url1"),
		},
		"*    [latest attempt](url1)",
	},
}

func TestSummary(t *testing.T) {
	t.Parallel()
	for _, tt := range testSummaryData {
		tt := tt
		t.Run(fmt.Sprintf("(%s)", tt.wantSummary), func(t *testing.T) {
			t.Parallel()
			updater := &InvocationStepUpdater{tasks: tt.tasks}
			if diff := cmp.Diff(tt.wantSummary, updater.summary()); diff != "" {
				t.Errorf("unexpected diff (%s)", diff)
			}
		})
	}
}

var testNotifyNewTaskData = []struct {
	currentTasks            []*testrunner.Build
	newTask                 *testrunner.Build
	wantStepSummaryMarkdown string
}{
	{
		[]*testrunner.Build{
			testrunner.NewBuildForTesting("task1", "url1"),
			testrunner.NewBuildForTesting("task2", "url2"),
		},
		testrunner.NewBuildForTesting("task3", "url3"),
		`*    [latest attempt](url3)
*    previous failed attempts: [1](url1), [2](url2)`,
	},
	{
		[]*testrunner.Build{
			testrunner.NewBuildForTesting("task1", "url1"),
		},
		testrunner.NewBuildForTesting("task2", "url2"),
		`*    [latest attempt](url2)
*    previous failed attempts: [1](url1)`,
	},
	{
		[]*testrunner.Build{},
		testrunner.NewBuildForTesting("task1", "url1"),
		"*    [latest attempt](url1)",
	},
}

func TestNotifyNewTask(t *testing.T) {
	t.Parallel()
	for _, tt := range testNotifyNewTaskData {
		tt := tt
		t.Run(fmt.Sprintf("(%s)", tt.wantStepSummaryMarkdown), func(t *testing.T) {
			t.Parallel()
			updater := &InvocationStepUpdater{
				step:  &buildbucketpb.Step{},
				tasks: tt.currentTasks,
			}
			updater.NotifyNewTask(tt.newTask)

			if diff := cmp.Diff(tt.wantStepSummaryMarkdown, updater.step.SummaryMarkdown); diff != "" {
				t.Errorf("unexpected diff (%s)", diff)
			}
		})
	}
}
