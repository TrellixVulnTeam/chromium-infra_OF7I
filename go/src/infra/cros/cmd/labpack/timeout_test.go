// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"strings"
	"testing"
	"time"
)

// TestCallFuncWithTimeout tests calling a function with a timeout.
func TestCallFuncWithTimeout(t *testing.T) {
	// t.Parallel() -- This thing tests goroutine timeouts.
	// Out of an abundance of caution, disable test parallelism.

	ctx := context.Background()

	// Be careful not to waste resources in the (unlikely?) event of a bug.
	// The callback should NOT be an infinite loop.
	status, err := callFuncWithTimeout(
		ctx,
		time.Nanosecond,
		func(ctx context.Context) error {
			time.Sleep(time.Second)
			return nil
		},
	)

	if status != interrupted {
		t.Errorf("bad status: %s", status)
	}
	if err == nil {
		t.Errorf("error is unexpectedly nil")
	}
	if !strings.Contains(err.Error(), "deadline exceeded") {
		t.Errorf("error message looks wrong: %s", err)
	}
}
