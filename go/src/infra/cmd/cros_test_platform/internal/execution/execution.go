// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package execution

import (
	"context"
	"fmt"
	"infra/cmd/cros_test_platform/internal/execution/skylab"
	"time"

	"go.chromium.org/chromiumos/infra/proto/go/test_platform/config"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/steps"
	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/errors"
)

// Runner manages task sets for multiple cros_test_platform requests.
type Runner struct {
	requestTaskSets map[string]*RequestTaskSet
	running         bool
}

// NewRunnerWithRequestTaskSets returns a new Runner to manage the provided
// RequestTaskSets.
//
// This constructor is only used by unittests.
func NewRunnerWithRequestTaskSets(requestTaskSets ...*RequestTaskSet) *Runner {
	r := &Runner{
		requestTaskSets: make(map[string]*RequestTaskSet),
	}
	for i, ts := range requestTaskSets {
		r.requestTaskSets[fmt.Sprintf("task%d", i)] = ts
	}
	return r
}

// NewRunner returns a Runner that will execute the given requests.
func NewRunner(workerConfig *config.Config_SkylabWorker, parentTaskID string, deadline time.Time, requests map[string]*steps.ExecuteRequest) (*Runner, error) {
	ts := make(map[string]*RequestTaskSet)
	for t, r := range requests {
		var err error
		ts[t], err = NewRequestTaskSet(r.Enumeration.AutotestInvocations, r.RequestParams, workerConfig, parentTaskID, deadline)
		if err != nil {
			return nil, errors.Annotate(err, "new skylab runner").Err()
		}
	}
	return &Runner{
		requestTaskSets: ts,
	}, nil
}

// LaunchAndWait launches a skylab execution and waits for it to complete,
// polling for new results periodically, and retrying tests that need retry,
// based on retry policy.
//
// If the supplied context is cancelled prior to completion, or some other error
// is encountered, this method returns whatever partial execution response
// was visible to it prior to that error.
func (r *Runner) LaunchAndWait(ctx context.Context, c skylab.Client) error {
	defer func() { r.running = false }()

	if err := r.launchTasks(ctx, c); err != nil {
		return err
	}
	for {
		if err := r.checkTasksAndRetry(ctx, c); err != nil {
			return err
		}
		if r.completed() {
			return nil
		}

		select {
		case <-ctx.Done():
			// A timeout while waiting for tests to complete is reported as
			// aborts when summarizing individual tests' results.
			// The execute step completes without errors.
			return nil
		case <-clock.After(ctx, 15*time.Second):
		}
	}
}

func (r *Runner) launchTasks(ctx context.Context, c skylab.Client) error {
	for t, ts := range r.requestTaskSets {
		if err := ts.LaunchTasks(ctx, c); err != nil {
			return errors.Annotate(err, "launch tasks for %s", t).Err()
		}
	}
	return nil
}

func (r *Runner) checkTasksAndRetry(ctx context.Context, c skylab.Client) error {
	for t, ts := range r.requestTaskSets {
		if err := ts.CheckTasksAndRetry(ctx, c); err != nil {
			return errors.Annotate(err, "check tasks and retry for %s", t).Err()
		}
	}
	return nil
}

func (r *Runner) completed() bool {
	for _, ts := range r.requestTaskSets {
		if !ts.Completed() {
			return false
		}
	}
	return true
}

// Responses constructs responses for each request managed by the Runner.
func (r *Runner) Responses() map[string]*steps.ExecuteResponse {
	running := r.running
	resps := make(map[string]*steps.ExecuteResponse)
	for t, ts := range r.requestTaskSets {
		resps[t] = ts.response(running)
	}
	return resps
}
