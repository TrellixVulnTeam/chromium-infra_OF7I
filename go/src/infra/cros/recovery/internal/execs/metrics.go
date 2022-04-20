// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package execs

import (
	"context"
	"time"

	"infra/cros/recovery/internal/log"
	"infra/cros/recovery/logger/metrics"

	"go.chromium.org/luci/common/errors"
)

// CloserFunc is a function that updates an action and is NOT safe to use in a defer block WITHOUT CHECKING FOR NIL.
// The following ways of using a CloserFunc are both correct.
//
// `ctx` and `err` are bound by the surrounding context.
//
//   action, closer := someFunction(...)
//   if closer != nil {
//     defer closer(ctx, err)
//   }
//
//   action, closer := someFunction(...)
//   defer func() {
//     if closer != nil {
//       defer closer(ctx, err)
//     }
//   }()
//
type CloserFunc = func(context.Context, error)

// NewMetric takes a reference to an action and populates it as a new action of kind `kind`.
// NewMetric mutates its argument action.
func (a *RunArgs) NewMetric(ctx context.Context, kind string, action *metrics.Action) (CloserFunc, error) {
	// Keep this function up to date with the call to args.Metrics.Create in recovery.go
	if a == nil {
		return nil, errors.Reason("new metrics: run args cannot be nil").Err()
	}
	if action == nil {
		return nil, errors.Reason("new metrics: action cannot be nil").Err()
	}
	dutName := ""
	if a.DUT != nil {
		dutName = a.DUT.Name
	}
	startTime := time.Now()
	*action = metrics.Action{
		ActionKind:     kind,
		StartTime:      startTime,
		SwarmingTaskID: a.SwarmingTaskID,
		BuildbucketID:  a.BuildbucketID,
		Hostname:       dutName,
	}
	c := createMetric(ctx, a.Metrics, action)
	return c, nil
}

// CreateMetric creates a metric with an actionKind, and a startTime.
// It returns an action and a closer function.
// CreateMetric mutates its argument action.
//
// Intended usage:
//
//  err is managed by the containing function.
//
//  Note that it is necessary to explicitly defer evaluation of err to the
//  end of the function.
//
//  closer := createMetric(ctx, ...)
//  if closer != nil {
//    defer func() {
//      closer(ctx, err)
//    }()
//  }
//
func createMetric(ctx context.Context, m metrics.Metrics, action *metrics.Action) func(context.Context, error) {
	if m == nil {
		return nil
	}
	if err := m.Create(ctx, action); err != nil {
		log.Errorf(ctx, err.Error())
	}
	closer := func(ctx context.Context, e error) {
		if m == nil {
			log.Debugf(ctx, "Forgivable error while creating metric, nil metrics")
			return
		}
		if action == nil {
			log.Debugf(ctx, "Forgivable error while creating metric, action reference points to nil action")
			return
		}
		action.Status = metrics.ActionStatusUnspecified
		// TODO(gregorynisbet): Consider strategies for multiple fail reasons.
		if e != nil {
			log.Debugf(ctx, "Updating action %q of kind %q during close failed with reason %q", action.Name, action.ActionKind, e.Error())
			action.Status = metrics.ActionStatusFail
			action.FailReason = e.Error()
		} else {
			action.Status = metrics.ActionStatusSuccess
			log.Debugf(ctx, "Updating action %q of kind %q during close was successful", action.Name, action.ActionKind)
		}
		if uErr := m.Update(ctx, action); uErr != nil {
			log.Errorf(ctx, "Updating action %q during close had error during upload: %s", action.Name, uErr.Error())
		}
	}
	return closer
}
