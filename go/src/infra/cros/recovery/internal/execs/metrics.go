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

// NewMetric creates a new metric. Neither the action nor the closer function that NewMetrics returns will
// ever be nil.
// TODO(gregorynisbet): Consider adding a time parameter.
func (a *RunArgs) NewMetric(ctx context.Context, kind string) (*metrics.Action, CloserFunc, error) {
	if a == nil {
		return nil, nil, errors.Reason("new metrics: run args cannot be nil").Err()
	}
	startTime := time.Now()
	action := &metrics.Action{
		ActionKind:     kind,
		StartTime:      startTime,
		SwarmingTaskID: a.SwarmingTaskID,
		BuildbucketID:  a.BuildbucketID,
	}
	m, c := createMetric(ctx, a.Metrics, action)
	return m, c, nil
}

// CreateMetric creates a metric with an actionKind, and a startTime.
// It returns an action and a closer function.
//
// Intended usage:
//
//  err is managed by the containing function.
//
//  Note that it is necessary to explicitly defer evaluation of err to the
//  end of the function.
//
//  action, closer := createMetric(ctx, ...)
//  if closer != nil {
//    defer func() {
//      closer(ctx, err)
//    }()
//  }
//
func createMetric(ctx context.Context, m metrics.Metrics, action *metrics.Action) (*metrics.Action, func(context.Context, error)) {
	if m == nil {
		return nil, nil
	}
	a, err := m.Create(ctx, action)
	if err != nil {
		log.Error(ctx, err.Error())
	}
	closer := func(ctx context.Context, e error) {
		if m == nil {
			log.Debug(ctx, "error while creating metric, nil metrics")
			return
		}
		if a == nil {
			log.Debug(ctx, "error while creating metric, nil action")
			return
		}
		a.Status = metrics.ActionStatusUnspecified
		// TODO(gregorynisbet): Consider strategies for multiple fail reasons.
		if e != nil {
			log.Debug(ctx, "Updating action %q of kind %q during close failed with reason %q", action.Name, action.ActionKind, e.Error())
			a.Status = metrics.ActionStatusFail
			a.FailReason = e.Error()
		} else {
			a.Status = metrics.ActionStatusSuccess
			log.Debug(ctx, "Updating action %q of kind %q during close was successful", action.Name, action.ActionKind)
		}
		_, err := m.Update(ctx, a)
		if err != nil {
			log.Error(ctx, "Updating action %q during close had error during upload: %s", action.Name, err.Error())
		}
		return
	}
	return a, closer
}
