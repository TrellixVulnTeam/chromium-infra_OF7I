// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package execs

import (
	"context"
	"infra/cros/recovery/logger"
	"infra/cros/recovery/logger/metrics"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestRunExec(t *testing.T) {
	ctx := context.Background()
	const actionExecWrong = "wrong_name"
	const actionExecGood = "sample_pass"
	const actionExecBad = "sample_fail"
	const actionExecMetricsAction = "sample_metrics_action"
	args := &RunArgs{}
	actionArgs := []string{"action", "args"}
	t.Run("Incorrect name", func(t *testing.T) {
		t.Parallel()
		err := Run(ctx, actionExecWrong, args, actionArgs)
		if err == nil {
			t.Errorf("Expected to fail")
		}
		if err.Error() != "exec \"wrong_name\": not found" {
			t.Errorf("Did not have expected explanation. Got: %q", err.Error())
		}
	})
	t.Run("Good sample", func(t *testing.T) {
		t.Parallel()
		if err := Run(ctx, actionExecGood, args, actionArgs); err != nil {
			t.Errorf("Expected to pass")
		}
	})
	t.Run("Bad sample", func(t *testing.T) {
		t.Parallel()
		if err := Run(ctx, actionExecBad, args, actionArgs); err == nil {
			t.Errorf("Expected to have status Fail")
		}
	})
	t.Run("Send metrics action", func(t *testing.T) {
		t.Parallel()
		if err := Run(ctx, actionExecMetricsAction, args, actionArgs); err != nil {
			t.Errorf("Expected to pass")
		}
	})
}

// TestNewMetric tests that we can create a new metric with the appropriate kind and swarmingID.
func TestNewMetric(t *testing.T) {
	t.Parallel()
	Convey("swarming ID", t, func() {
		ctx := context.Background()
		// Create a harmless metrics implementation using the logger.
		lg := logger.NewLogger()
		metrics := metrics.NewLogMetrics(lg)
		a := &RunArgs{
			Metrics:        metrics,
			SwarmingTaskID: "f2ef3b36-1985-4b11-9381-f7de82c49bd6",
		}
		action, closer := a.NewMetric(ctx, "ssh-attempt")
		So(action, ShouldNotBeNil)
		So(action.SwarmingTaskID, ShouldEqual, "f2ef3b36-1985-4b11-9381-f7de82c49bd6")
		So(closer, ShouldNotBeNil)
	})
}
