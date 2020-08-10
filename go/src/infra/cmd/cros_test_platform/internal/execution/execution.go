// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package execution

import (
	"context"
	"fmt"
	"infra/cmd/cros_test_platform/internal/execution/skylab"
	"time"

	"go.chromium.org/luci/luciexe/exe"

	"go.chromium.org/chromiumos/infra/proto/go/test_platform"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/config"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/steps"
	bbpb "go.chromium.org/luci/buildbucket/proto"
	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/errors"
)

// ctpRequestUIDTemplate is the template to generate the UID of
// a test plan run, a.k.a. CTP request.
const ctpRequestUIDTemplate = "TestPlanRuns/%d/%s"

// Runner manages task sets for multiple cros_test_platform requests.
type Runner struct {
	requestTaskSets map[string]*RequestTaskSet
	send            exe.BuildSender
	waiting         bool
}

// NewRunner returns a Runner that will execute the given requests.
func NewRunner(buildInstance *bbpb.Build, send exe.BuildSender, workerConfig *config.Config_SkylabWorker, parentTaskID string, deadline time.Time, request steps.ExecuteRequests) (*Runner, error) {
	ts := make(map[string]*RequestTaskSet)
	for t, r := range request.GetTaggedRequests() {
		var err error
		ts[t], err = NewRequestTaskSet(
			t,
			buildInstance,
			workerConfig,
			&TaskSetConfig{
				parentTaskID,
				constructRequestUID(request.GetBuild().GetId(), t),
				deadline,
			},
			r.RequestParams,
			r.Enumeration.AutotestInvocations,
		)
		if err != nil {
			return nil, errors.Annotate(err, "new skylab runner").Err()
		}
	}
	return &Runner{
		requestTaskSets: ts,
		send:            send,
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
	defer func() { r.waiting = false }()

	// TODO(pprabhu): We may fail to Close() the individual requests if we fail
	// between a call to NewRunner() and this function.
	// To fix this, merge NewRunner() and LaunchAndWait() into a single method.
	for _, ts := range r.requestTaskSets {
		defer ts.Close()
	}

	if err := r.launchTasks(ctx, c); err != nil {
		r.send()
		return err
	}
	for {
		allDone, err := r.checkTasksAndRetry(ctx, c)

		// Each call to checkTasksAndRetry() potentially updates the Build.
		// We unconditionally send() the updated build so that we reflect the
		// update irrespective of abnormal exits.
		// Since this loop sleeps between iterations, the load generated on
		// the buildbucket service is bounded.
		r.send()

		if err != nil {
			return err
		}
		if allDone {
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

// Returns whether all tasks are complete (so future calls to this function are
// unnecessary)
func (r *Runner) checkTasksAndRetry(ctx context.Context, c skylab.Client) (bool, error) {
	allDone := true
	for t, ts := range r.requestTaskSets {
		c, err := ts.CheckTasksAndRetry(ctx, c)
		if err != nil {
			return false, errors.Annotate(err, "check tasks and retry for %s", t).Err()
		}
		allDone = allDone && c
	}
	return allDone, nil
}

// Responses constructs responses for each request managed by the Runner.
func (r *Runner) Responses() map[string]*steps.ExecuteResponse {
	resps := make(map[string]*steps.ExecuteResponse)
	for t, ts := range r.requestTaskSets {
		resps[t] = ts.Response()
		// The test hasn't completed, but we're not waiting for it to complete
		// anymore.
		if !r.waiting && resps[t].GetState().LifeCycle == test_platform.TaskState_LIFE_CYCLE_RUNNING {
			resps[t].State.LifeCycle = test_platform.TaskState_LIFE_CYCLE_ABORTED
		}
	}
	return resps
}

func constructRequestUID(buildID int64, key string) string {
	return fmt.Sprintf(ctpRequestUIDTemplate, buildID, key)
}
