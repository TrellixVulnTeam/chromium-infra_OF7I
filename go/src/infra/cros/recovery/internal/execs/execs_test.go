// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package execs

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	. "github.com/smartystreets/goconvey/convey"

	"infra/cros/recovery/logger"
	"infra/cros/recovery/logger/metrics"
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
	Convey("buildbucket ID", t, func() {
		ctx := context.Background()
		// Create a harmless metrics implementation using the logger.
		lg := logger.NewLogger()
		metrics := metrics.NewLogMetrics(lg)
		a := &RunArgs{
			Metrics:       metrics,
			BuildbucketID: "35510b33-5c0e-44ef-a81d-9bce1ed4137e",
		}
		action, closer := a.NewMetric(ctx, "ssh-attempt")
		So(action, ShouldNotBeNil)
		So(action.BuildbucketID, ShouldEqual, "35510b33-5c0e-44ef-a81d-9bce1ed4137e")
		So(closer, ShouldNotBeNil)
	})
}

// Test cases for TestDUTPlans
var parseActionArgsCases = []struct {
	name     string
	args     []string
	splitter string
	expected map[string]string
}{
	{
		"empty",
		nil,
		DefaultSplitter,
		nil,
	},
	{
		"empty 2",
		[]string{" ", "", "    "},
		DefaultSplitter,
		nil,
	},
	{
		"simple args",
		[]string{"my", "1", "&&&", "---9433"},
		DefaultSplitter,
		map[string]string{
			"my": "", "1": "", "&&&": "", "---9433": "",
		},
	},
	{
		"pair values args",
		[]string{"my", "&&&:1234", "---9433: my value "},
		DefaultSplitter,
		map[string]string{
			"my":      "",
			"&&&":     "1234",
			"---9433": "my value",
		},
	},
	{
		"complicated cases",
		[]string{
			"my",
			"&&&:1234",
			"key: val:split "},
		DefaultSplitter,
		map[string]string{
			"my":  "",
			"&&&": "1234",
			"key": "val:split",
		},
	},
}

func TestParseActionArgs(t *testing.T) {
	t.Parallel()
	for _, c := range parseActionArgsCases {
		cs := c
		t.Run(cs.name, func(t *testing.T) {
			ctx := context.Background()
			got := ParseActionArgs(ctx, cs.args, cs.splitter)
			if len(cs.expected) == 0 && len(got) == len(cs.expected) {
				// Everything is good.
			} else {
				if !cmp.Equal(got, cs.expected) {
					t.Errorf("%q ->want: %v\n got: %v", cs.name, cs.expected, got)
				}
			}
		})
	}
}
