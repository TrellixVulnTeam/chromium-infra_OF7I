// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package skylab

import (
	"context"
	"fmt"
	"infra/cmd/cros_test_platform/internal/execution/isolate"
	"infra/cmd/cros_test_platform/internal/execution/swarming"
	"time"

	"go.chromium.org/chromiumos/infra/proto/go/test_platform/config"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/steps"
	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/errors"
)

// Runner is a new Skylab task set runner.
type Runner struct {
	taskSets map[string]*TaskSet
	running  bool
}

// NewRunnerWithTaskSets returns a new Runner for executing the provided
// TaskSets.
//
// This constructor is only used by unittests.
func NewRunnerWithTaskSets(taskSets ...*TaskSet) *Runner {
	r := &Runner{
		taskSets: make(map[string]*TaskSet),
	}
	for i, ts := range taskSets {
		r.taskSets[fmt.Sprintf("task%d", i)] = ts
	}
	return r
}

// NewRunner returns a Runner that will execute the given tests.
func NewRunner(workerConfig *config.Config_SkylabWorker, parentTaskID string, requests map[string]*steps.ExecuteRequest) (*Runner, error) {
	ts := make(map[string]*TaskSet)
	for t, r := range requests {
		var err error
		ts[t], err = NewTaskSet(r.Enumeration.AutotestInvocations, r.RequestParams, workerConfig, parentTaskID)
		if err != nil {
			return nil, errors.Annotate(err, "new skylab runner").Err()
		}
	}
	return &Runner{
		taskSets: ts,
	}, nil
}

// Clients bundles local interfaces to various remote services used by Runner.
type Clients struct {
	Swarming      swarming.Client
	IsolateGetter isolate.GetterFactory
}

// LaunchAndWait launches a skylab execution and waits for it to complete,
// polling for new results periodically, and retrying tests that need retry,
// based on retry policy.
//
// If the supplied context is cancelled prior to completion, or some other error
// is encountered, this method returns whatever partial execution response
// was visible to it prior to that error.
func (r *Runner) LaunchAndWait(ctx context.Context, clients Clients) error {
	defer func() { r.running = false }()

	if err := r.launchTasks(ctx, clients); err != nil {
		return err
	}
	for {
		if err := r.checkTasksAndRetry(ctx, clients); err != nil {
			return err
		}
		if r.completed() {
			return nil
		}

		select {
		case <-ctx.Done():
			return errors.Annotate(ctx.Err(), "wait for tests").Err()
		case <-clock.After(ctx, 15*time.Second):
		}
	}
}

func (r *Runner) launchTasks(ctx context.Context, clients Clients) error {
	for t, ts := range r.taskSets {
		if err := ts.LaunchTasks(ctx, clients); err != nil {
			return errors.Annotate(err, "launch tasks for %s", t).Err()
		}
	}
	return nil
}

func (r *Runner) checkTasksAndRetry(ctx context.Context, clients Clients) error {
	for t, ts := range r.taskSets {
		if err := ts.CheckTasksAndRetry(ctx, clients); err != nil {
			return errors.Annotate(err, "check tasks and retry for %s", t).Err()
		}
	}
	return nil
}

func (r *Runner) completed() bool {
	for _, ts := range r.taskSets {
		if !ts.Completed() {
			return false
		}
	}
	return true
}

// Responses constructs responses for each taskSet managed by the Runner.
func (r *Runner) Responses(urler swarming.URLer) map[string]*steps.ExecuteResponse {
	running := r.running
	resps := make(map[string]*steps.ExecuteResponse)
	for t, ts := range r.taskSets {
		resps[t] = ts.response(urler, running)
	}
	return resps
}
