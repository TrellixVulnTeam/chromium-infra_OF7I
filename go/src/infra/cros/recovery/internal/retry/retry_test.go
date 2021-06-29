// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package retry

import (
	"context"
	"testing"
	"time"

	"go.chromium.org/luci/common/errors"
)

func TestLimitCount(t *testing.T) {
	ctx := context.Background()
	t.Run("Fail as reached limit", func(t *testing.T) {
		t.Parallel()
		if err := LimitCount(ctx, 2, time.Nanosecond, createFunc(2), "Fail with reached limit"); err == nil {
			t.Errorf("Expected to fail")
		}
	})
	t.Run("Passed as not reached limit", func(t *testing.T) {
		t.Parallel()
		if err := LimitCount(ctx, 2, time.Nanosecond, createFunc(1), "Passed as not reached limit"); err != nil {
			t.Errorf("Expected to pass: %s", err)
		}
	})
	t.Run("Passed with first attemp", func(t *testing.T) {
		t.Parallel()
		if err := LimitCount(ctx, 0, time.Nanosecond, createFunc(0), "Passed with first attemp"); err != nil {
			t.Errorf("Expected to pass: %s", err)
		}
	})
}

func TestLimitTime(t *testing.T) {
	ctx := context.Background()
	t.Run("Fail as reached limit", func(t *testing.T) {
		t.Parallel()
		if err := WithTimeout(ctx, time.Millisecond, 50*time.Millisecond, createFunc(50), "Fail as reached limit"); err == nil {
			t.Errorf("Expected to fail")
		}
	})
	t.Run("Passed as not reached limit", func(t *testing.T) {
		t.Parallel()
		if err := WithTimeout(ctx, time.Millisecond, 50*time.Millisecond, createFunc(10), "Passed as not reached limit"); err != nil {
			t.Errorf("Expected to pass: %s", err)
		}
	})
	t.Run("Passed with first attemp", func(t *testing.T) {
		t.Parallel()
		if err := WithTimeout(ctx, time.Millisecond, 50*time.Millisecond, createFunc(0), "Passed with first attemp"); err != nil {
			t.Errorf("Expected to pass: %s", err)
		}
	})
}

func createFunc(count int) func() error {
	var i int
	return func() error {
		if i < count {
			i++
			return errors.Reason("err for attempt: %d", i).Err()
		}
		return nil
	}
}
