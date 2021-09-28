// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package execs

import (
	"context"
	"infra/cros/recovery/logger/metrics"
	"time"

	"go.chromium.org/luci/common/errors"
)

// samplePassActionExec provides example to run action which always pass.
func samplePassActionExec(ctx context.Context, args *RunArgs, actionArgs []string) error {
	return nil
}

// sampleFailActionExec provides example to run action which always fail.
func sampleFailActionExec(ctx context.Context, args *RunArgs, actionArgs []string) error {
	return errors.Reason("failed").Err()
}

// sampleMetricsAction sends a record to the metrics service.
func sampleMetricsAction(ctx context.Context, args *RunArgs, actionArgs []string) error {
	// TODO(gregorynisbet): Add more interesting information to the action.
	action := &metrics.Action{}
	if args.Metrics != nil {
		action.StartTime = time.Now()
		action, _ = args.Metrics.Create(ctx, action)
		// TODO(gregorynisbet): Uncomment when update lands.
		// defer args.Metrics.Update(ctx, action)
	}
	// Test sleeping for one nanosecond. This will cause time to pass, which will be
	// reflected in the action and therefore in Karte.
	time.Sleep(time.Nanosecond)
	action.StopTime = time.Now()
	return nil
}

func init() {
	Register("sample_pass", samplePassActionExec)
	Register("sample_fail", sampleFailActionExec)
	Register("sample_metrics_action", sampleMetricsAction)
}
