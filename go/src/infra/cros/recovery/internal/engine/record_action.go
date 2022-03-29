// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package engine

import (
	"context"
	"fmt"
	"infra/cros/recovery/internal/log"
	"infra/cros/recovery/logger/metrics"
	"time"
)

// recordActionCloser is a function that takes an error (the ultimate error produced by an action) and records
// it inside a defer block.
type recordActionCloser = func(error)

// recordAction takes a context and an action name and records the initial action for a record.
// The parameter action is assumed NOT to be nil. Also, this function indirectly mutates its parameter action.
func (r *recoveryEngine) recordAction(ctx context.Context, actionName string, action *metrics.Action) recordActionCloser {
	if r == nil {
		log.Debugf(ctx, "RecoveryEngine is nil, skipping")
		return nil
	}
	if r.args == nil {
		log.Debugf(ctx, "Metrics is nil, skipping")
		return nil
	}
	if r.args.Metrics != nil {
		log.Debugf(ctx, "Recording metrics for action %q", actionName)
		// Create the metric up front. Allow 30 seconds to talk to Karte.
		createMetricCtx, createMetricCloser := context.WithTimeout(ctx, 30*time.Second)
		defer createMetricCloser()
		u, err := r.args.NewMetric(
			createMetricCtx,
			// TODO(gregorynisbet): Consider adding a new field to Karte to explicitly track the name
			//                      assigned to an action by recoverylib.
			fmt.Sprintf("action:%s", actionName),
			action,
		)
		if err != nil {
			log.Errorf(ctx, "Encountered error when creating action: %s", err)
			return nil
		}
		// Here we intentionally close over the context "early", before the deadline is applied inside
		// runAction.
		return func(rErr error) {
			// Update the metric. This contains information that we will not know until after the action ran.
			updateMetricCtx, updateMetricCloser := context.WithTimeout(ctx, 30*time.Second)
			defer updateMetricCloser()
			u(updateMetricCtx, rErr)
		}
	} else {
		log.Debugf(ctx, "Skipping metrics for action %q", actionName)
		return nil
	}
}
