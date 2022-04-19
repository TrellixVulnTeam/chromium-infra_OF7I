// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package execs

import (
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"infra/cros/recovery/logger"
	"infra/cros/recovery/logger/metrics"
)

// TestNewMetric tests that we can create a new metric with the appropriate kind and swarmingID.
func TestNewMetric(t *testing.T) {
	t.Parallel()
	Convey("swarming ID", t, func() {
		ctx := context.Background()
		// Create a harmless metrics implementation using the logger.
		lg := logger.NewLogger()
		m := metrics.NewLogMetrics(lg)
		a := &RunArgs{
			Metrics:        m,
			SwarmingTaskID: "f2ef3b36-1985-4b11-9381-f7de82c49bd6",
		}
		action := &metrics.Action{}
		closer, err := a.NewMetric(ctx, "ssh-attempt", action)
		So(err, ShouldBeNil)
		So(action, ShouldNotBeNil)
		So(action.SwarmingTaskID, ShouldEqual, "f2ef3b36-1985-4b11-9381-f7de82c49bd6")
		So(closer, ShouldNotBeNil)
	})
	Convey("buildbucket ID", t, func() {
		ctx := context.Background()
		// Create a harmless metrics implementation using the logger.
		lg := logger.NewLogger()
		m := metrics.NewLogMetrics(lg)
		a := &RunArgs{
			Metrics:       m,
			BuildbucketID: "35510b33-5c0e-44ef-a81d-9bce1ed4137e",
		}
		action := &metrics.Action{}
		closer, err := a.NewMetric(ctx, "ssh-attempt", action)
		So(err, ShouldBeNil)
		So(action, ShouldNotBeNil)
		So(action.BuildbucketID, ShouldEqual, "35510b33-5c0e-44ef-a81d-9bce1ed4137e")
		So(closer, ShouldNotBeNil)
	})
}
