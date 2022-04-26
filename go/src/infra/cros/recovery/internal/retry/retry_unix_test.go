// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// These tests are not fundamentally platform-specific. However, they are
// more brittle than one would expect on Windows under high load for mysterious
// reasons that haven't been determined yet.
//
// See b:230128605 for more details.

package retry

import (
	"context"
	"strings"
	"testing"
	"time"

	"go.chromium.org/luci/common/errors"
)

// TestLimitCount tests retries with limit to count of retries.
//
// Test based on testing returned errors and check error messages.
func TestLimitCount(t *testing.T) {
	// t.Parallel() -- b:227523207
	ctx := context.Background()
	t.Run("Fail as reached limit", func(t *testing.T) {
		// t.Parallel() -- b:227523207
		if err := LimitCount(ctx, 4, time.Nanosecond, createFunc(4, 0, 0), "Retry test by count"); err == nil {
			t.Errorf("Expected to fail")
		} else if !strings.Contains(err.Error(), simpleErrorMsg) {
			t.Errorf("Expected to finish with error from func: %s", err)
		} else if !strings.Contains(err.Error(), "attempt: 4") {
			t.Errorf("Expected to stop by abort: %s", err)
		}
	})
	t.Run("Passed as not reached limit", func(t *testing.T) {
		// t.Parallel() -- b:227523207
		if err := LimitCount(ctx, 2, time.Nanosecond, createFunc(1, 0, 0), "Retry test by count"); err != nil {
			t.Errorf("Expected to pass: %s", err)
		}
	})
	t.Run("Passed with first attempt", func(t *testing.T) {
		// t.Parallel() -- b:227523207
		if err := LimitCount(ctx, 0, time.Nanosecond, createFunc(0, 0, 0), "Retry test by count"); err != nil {
			t.Errorf("Expected to pass: %s", err)
		}
	})
	t.Run("Break before reached the time count", func(t *testing.T) {
		// t.Parallel() -- b:227523207
		if err := LimitCount(ctx, 10, time.Nanosecond, createFunc(8, 5, 0), "Retry test by count"); err == nil {
			t.Errorf("Expected to fail: %s", err)
		} else if !strings.Contains(err.Error(), abortErrorMsg) {
			t.Errorf("Expected to stop by abort: %s", err)
		} else if !strings.Contains(err.Error(), "attempt: 5") {
			t.Errorf("Expected to stop by abort: %s", err)
		}
	})
	t.Run("Cancel by parent context", func(t *testing.T) {
		// t.Parallel() -- b:227523207
		ctx, cancel := context.WithTimeout(ctx, time.Nanosecond)
		defer func() { cancel() }()
		if err := LimitCount(ctx, 10, time.Second, createFunc(10, 0, 0), "Retry test by count"); err == nil {
			t.Errorf("Expected to fail by pass")
		} else if !strings.Contains(err.Error(), "attempts took 0.00 seconds: context deadline exceeded") {
			t.Errorf("Expected to stop by abort: %s", err)
		}
	})
}

// TestLimitTime tests limit retries limited by time execution.
//
// Test based on testing returned errors and check error messages.
func TestLimitTime(t *testing.T) {
	// t.Parallel() -- b:227523207
	ctx := context.Background()
	t.Run("Fail as reached limit", func(t *testing.T) {
		// t.Parallel() -- b:227523207
		if err := WithTimeout(ctx, time.Millisecond, 50*time.Millisecond, createFunc(50, 0, 0), "Retry test by time"); err == nil {
			t.Errorf("Expected to fail")
		} else if !strings.Contains(err.Error(), "context deadline exceeded") {
			t.Errorf("Expected to finish with error from func: %s", err)
		}
	})
	t.Run("Passed as not reached limit", func(t *testing.T) {
		// t.Parallel() -- b:227523207
		if err := WithTimeout(ctx, time.Millisecond, time.Second, createFunc(10, 0, 0), "Retry test by time"); err != nil {
			t.Errorf("Expected to pass: %s", err)
		}
	})
	t.Run("Passed with first attempt", func(t *testing.T) {
		// t.Parallel() -- b:227523207
		if err := WithTimeout(ctx, time.Millisecond, 50*time.Millisecond, createFunc(0, 0, 0), "Retry test by time"); err != nil {
			t.Errorf("Expected to pass: %s", err)
		}
	})
	t.Run("Break before reached the time count", func(t *testing.T) {
		// t.Parallel() -- b:227523207
		if err := WithTimeout(ctx, time.Millisecond, 5*time.Second, createFunc(8, 5, 0), "Retry test by time"); err == nil {
			t.Errorf("Expected to fail: %s", err)
		} else if !strings.Contains(err.Error(), abortErrorMsg) {
			t.Errorf("Expected to stop by abort: %s", err)
		}
	})
	t.Run("Cancel by parent context", func(t *testing.T) {
		// t.Parallel() -- b:227523207
		ctx, cancel := context.WithTimeout(ctx, time.Nanosecond)
		defer func() { cancel() }()
		if err := WithTimeout(ctx, time.Second, time.Second, createFunc(10, 0, 10*time.Millisecond), "Retry test by time"); err == nil {
			t.Errorf("Expected to fail by pass")
		} else if !strings.Contains(err.Error(), "attempts took 0.00 seconds: context deadline exceeded") {
			t.Errorf("Expected to stop by abort: %s", err)
		}
	})
}

const (
	abortErrorMsg  = "err !abort! for attempt"
	simpleErrorMsg = "err for attempt"
)

// create target function which will several times per request.
//
// count: 			How many times function will return simple error, after that it will rerun nil
// abortOnAttempt:	When generate error fo abort. if 0 then never.
func createFunc(count, abortOnAttempt int, initialSleep time.Duration) func() error {
	var i int
	return func() error {
		time.Sleep(initialSleep)
		if abortOnAttempt != 0 && i == abortOnAttempt {
			return errors.Reason("%s: %d", abortErrorMsg, i).Tag(LoopBreakTag()).Err()
		}
		if i < count {
			i++
			return errors.Reason("%s: %d", simpleErrorMsg, i).Err()
		}
		return nil
	}
}
