// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package testutil

import (
	"context"

	"go.chromium.org/luci/common/clock/testclock"
	"go.chromium.org/luci/common/logging/gologger"
)

func testingContext(mockClock bool) context.Context {
	ctx := context.Background()

	// Enable logging to stdout/stderr.
	ctx = gologger.StdConfig.Use(ctx)

	if mockClock {
		ctx, _ = testclock.UseTime(ctx, testclock.TestRecentTimeUTC)
	}

	return ctx
}

// TestingContext returns a context to be used in tests.
func TestingContext() context.Context {
	return testingContext(true)
}
