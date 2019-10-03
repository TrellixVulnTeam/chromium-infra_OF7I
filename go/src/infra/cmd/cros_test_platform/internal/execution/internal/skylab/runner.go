// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package skylab

import (
	"context"
	"infra/cmd/cros_test_platform/internal/execution/isolate"
	"infra/cmd/cros_test_platform/internal/execution/swarming"
	"time"

	"go.chromium.org/chromiumos/infra/proto/go/test_platform/steps"
	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/errors"
)

// Runner is a new Skylab task set runner.
type Runner struct {
	taskSet *TaskSet

	running bool
}

// NewRunner returns a new Runner for executing the provided TaskSet.
func NewRunner(taskSet *TaskSet) *Runner {
	return &Runner{
		taskSet: taskSet,
	}
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

	if err := r.taskSet.LaunchTasks(ctx, client); err != nil {
		return err
	}
	for {
		if err := r.taskSet.CheckTasksAndRetry(ctx, client, gf); err != nil {
			return err
		}
		if r.taskSet.Completed() {
			return nil
		}

		select {
		case <-ctx.Done():
			return errors.Annotate(ctx.Err(), "wait for tests").Err()
		case <-clock.After(ctx, 15*time.Second):
		}
	}
}

// Response constructs a response based on the current state of the Runner.
func (r *Runner) Response(urler swarming.URLer) *steps.ExecuteResponse {
	return r.taskSet.response(urler, r.running)
}
