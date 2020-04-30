// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cron

import (
	"context"
	"time"

	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/data/rand/mathrand"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/runtime/paniccatcher"
)

// Run runs f repeatedly, until the context is cancelled.
//
// Ensures f is not called too often (minInterval).
func Run(ctx context.Context, minInterval time.Duration, f func(context.Context) error) {
	defer logging.Warningf(ctx, "Exiting cron")

	// call calls f and catches a panic, will stop once the whole context is cancelled.
	call := func(ctx context.Context) error {
		defer paniccatcher.Catch(func(p *paniccatcher.Panic) {
			logging.Errorf(ctx, "Caught panic: %s\n%s", p.Reason, p.Stack)
		})
		return f(ctx)
	}

	for {
		start := clock.Now(ctx)
		if err := call(ctx); err != nil {
			logging.Errorf(ctx, "Iteration failed: %s", err)
		}

		// Ensure minInterval between iterations.
		if sleep := minInterval - clock.Since(ctx, start); sleep > 0 {
			// Add jitter: +-10% of sleep time to desynchronize cron jobs.
			sleep = sleep - sleep/10 + time.Duration(mathrand.Intn(ctx, int(sleep/5)))
			select {
			case <-time.After(sleep):
			case <-ctx.Done():
				return
			}
		}
	}
}
