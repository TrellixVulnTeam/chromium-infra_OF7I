// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package gob

import (
	"context"
	"time"

	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/clock/testclock"
	"go.chromium.org/luci/common/retry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type retryIterator struct {
	backoff retry.ExponentialBackoff
}

func (i *retryIterator) Next(ctx context.Context, err error) time.Duration {
	s, ok := status.FromError(err)
	if ok {
		switch s.Code() {
		case codes.NotFound, codes.Unavailable, codes.DeadlineExceeded:
			return i.backoff.Next(ctx, err)
		}
	}
	return retry.Stop
}

// Retry attempts a GoB operation with retries.
//
// Retry mitigates the effects of short-lived outages and replication lag by
// retrying operations with a 404 or 503 status code. The service client's error
// should be returned in order to correctly detect this situation. The retries
// will use exponential backoff with a context with the clock tagged with
// "gob-retry". When performing retries, a log will be emitted that uses opName
// to identify the operation that is being retried.
func Retry(ctx context.Context, opName string, fn func() error) error {
	retryFactory := func() retry.Iterator {
		return &retryIterator{
			backoff: retry.ExponentialBackoff{
				Limited: retry.Limited{
					Delay:   time.Second,
					Retries: 5,
				},
				Multiplier: 2,
			},
		}
	}
	ctx = clock.Tag(ctx, "gob-retry")
	return retry.Retry(ctx, retryFactory, fn, retry.LogCallback(ctx, opName))
}

func CtxForTest(ctx context.Context) context.Context {
	tc, ok := clock.Get(ctx).(testclock.TestClock)
	if !ok {
		ctx, tc = testclock.UseTime(ctx, testclock.TestTimeUTC)
	}

	tc.SetTimerCallback(func(d time.Duration, t clock.Timer) {
		if testclock.HasTags(t, "gob-retry") {
			tc.Add(d) // Fast-forward through sleeps in the test.
		}
	})
	return ctx
}
