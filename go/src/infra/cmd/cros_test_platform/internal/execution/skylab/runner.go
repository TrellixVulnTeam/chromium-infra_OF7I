// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package skylab

import (
	"context"
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
	taskSets []*TaskSet

	running bool
}

// NewRunnerWithTaskSets returns a new Runner for executing the provided
// TaskSets.
//
// This constructor is only used by unittests.
func NewRunnerWithTaskSets(taskSets ...*TaskSet) *Runner {
	return &Runner{
		taskSets: taskSets,
	}
}

// NewRunner returns a Runner that will execute the given tests.
func NewRunner(workerConfig *config.Config_SkylabWorker, parentTaskID string, requests []*steps.ExecuteRequest) (*Runner, error) {
	ts := make([]*TaskSet, len(requests))
	for i, r := range requests {
		var err error
		ts[i], err = NewTaskSet(r.Enumeration.AutotestInvocations, r.RequestParams, workerConfig, parentTaskID)
		if err != nil {
			return nil, errors.Annotate(err, "new skylab runner").Err()
		}
	}
	return &Runner{
		taskSets: ts,
	}, nil
}

// LaunchAndWait launches a skylab execution and waits for it to complete,
// polling for new results periodically, and retrying tests that need retry,
// based on retry policy.
//
// If the supplied context is cancelled prior to completion, or some other error
// is encountered, this method returns whatever partial execution response
// was visible to it prior to that error.
func (r *Runner) LaunchAndWait(ctx context.Context, client swarming.Client, gf isolate.GetterFactory) error {
	defer func() { r.running = false }()

	if err := r.launchTasks(ctx, client); err != nil {
		return err
	}
	for {
		if err := r.checkTasksAndRetry(ctx, client, gf); err != nil {
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

func (r *Runner) launchTasks(ctx context.Context, client swarming.Client) error {
	for _, ts := range r.taskSets {
		if err := ts.LaunchTasks(ctx, client); err != nil {
			return errors.Annotate(err, "launch tasks for request [%v]", ts).Err()
		}
	}
	return nil
}

func (r *Runner) checkTasksAndRetry(ctx context.Context, client swarming.Client, gf isolate.GetterFactory) error {
	for _, ts := range r.taskSets {
		if err := ts.CheckTasksAndRetry(ctx, client, gf); err != nil {
			return errors.Annotate(err, "check tasks and retry for request [%v]", ts).Err()
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
func (r *Runner) Responses(urler swarming.URLer) []*steps.ExecuteResponse {
	running := r.running
	resps := make([]*steps.ExecuteResponse, len(r.taskSets))
	for i, ts := range r.taskSets {
		resps[i] = ts.response(urler, running)
	}
	return resps
}
