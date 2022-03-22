// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package retry provides retry methods.
package retry

import (
	"context"
	"fmt"
	"time"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/log"
)

var stopRetryLoopTag = errors.BoolTag{Key: errors.NewTagKey("break retry loop")}

// LoopBreakTag returns tags to break to retry loop per request.
func LoopBreakTag() errors.BoolTag {
	return stopRetryLoopTag
}

// TODO(otabek@): Need to pass logger interface.
// Note: Context is required for all retries and will be used with new logger in further CLs.

// WithTimeout retries execute function in giving time duration.
//
// Example: Check if device is reachable, try during 1 hour with intervals 2 seconds.
//	 return retry.WithTimeout(ctx, time.Hour,  2*time.Second, func() error {
//	 	return  <-- return err if device is not reachable.
//	 }, "check if a device is reachable")
//
func WithTimeout(ctx context.Context, interval, duration time.Duration, f func() error, opName string) (err error) {
	ctx, cancel := context.WithTimeout(ctx, duration)
	defer func() { cancel() }()
	startTime := time.Now()
	var attempts int
	var abort bool
	err = retry(ctx, &retryOptions{
		next: func(ctx context.Context) error {
			attempts++
			err := f()
			if err == nil {
				log.Debug(ctx, getSuccessMessage(opName, attempts, startTime))
			}
			spentTime := time.Now().Sub(startTime).Seconds()
			log.Debug(ctx, "Retry %q: attempt %d (used %0.2f of %0.2f seconds), error: %s", opName, attempts, spentTime, duration.Seconds(), err)
			return err
		},
		hasNext: func(ctx context.Context) bool {
			// Time tracking by context timeout.
			return !abort
		},
		abort: func(ctx context.Context) {
			abort = true
			log.Debug(ctx, "Retry %q: aborted!", opName)
		},
		interval: interval,
	})
	return errors.Annotate(err, getEndErrorMessage(opName, attempts, startTime)).Err()
}

// LimitCount retries execute function with limit by numbers attempts.
//
// Example: Check if device is reachable, only try 5 times with interval 2 seconds.
//	 return retry.LimitCount(ctx, 5, 2*time.Second, func() error {
//	 	return  <-- return err if device is not reachable.
//	 }, "check if a device is reachable")
//
func LimitCount(ctx context.Context, count int, interval time.Duration, f func() error, opName string) (err error) {
	startTime := time.Now()
	var attempts int
	var abort bool
	err = retry(ctx, &retryOptions{
		next: func(ctx context.Context) error {
			attempts++
			err := f()
			if err == nil {
				log.Debug(ctx, getSuccessMessage(opName, attempts, startTime))
			}
			log.Debug(ctx, "Retry %q: attempts %d of %d, error: %s", opName, attempts, count, err)
			return err
		},
		hasNext: func(ctx context.Context) bool {
			if abort {
				return false
			}
			return attempts < count
		},
		abort: func(ctx context.Context) {
			abort = true
			log.Debug(ctx, "Retry %q: aborted!", opName)
		},
		interval: interval,
	})
	return errors.Annotate(err, getEndErrorMessage(opName, attempts, startTime)).Err()
}

type retryOptions struct {
	// Run next iteration of retry.
	next func(ctx context.Context) error
	// Check if retry has next iteration.
	hasNext func(ctx context.Context) bool
	// Abort retry function.
	abort func(ctx context.Context)
	// Interval between retries.
	interval time.Duration
}

// retry execute retry logic to run retries.
// If context report Done() the retry will be aborted.
func retry(ctx context.Context, o *retryOptions) error {
	// Buffered channels needed to not block writing when it is not longer reading in select.
	c := make(chan error, 1)
	go func() {
		var err error
		defer func() { c <- err }()
		for o.hasNext(ctx) {
			err = o.next(ctx)
			// If iteration finished with success we break the loop.
			if err == nil {
				return
			} else if stopRetryLoopTag.In(err) {
				log.Debug(ctx, "Retry received request for abort!")
				o.abort(ctx)
				// Removing tag from the error to void recursion stop.
				stopRetryLoopTag.Off().Apply(err)
				return
			}
			time.Sleep(o.interval)
		}
	}()
	select {
	case err := <-c:
		// Retry finished.
		return err
	case <-ctx.Done():
		// Allort loop as parent closed the context.
		o.abort(ctx)
		return ctx.Err()
	}
}

// getSuccessMessage creates a message for retry when it succeeded.
func getSuccessMessage(opName string, attempts int, startTime time.Time) string {
	spentTime := time.Now().Sub(startTime).Seconds()
	if attempts == 1 {
		return fmt.Sprintf("Retry %q: succeeded in first try. Spent %0.2f seconds.", opName, spentTime)
	}
	return fmt.Sprintf("Retry %q: succeeded in %d attempts. Spent %0.2f seconds.", opName, attempts, spentTime)
}

// getEndErrorMessage creates an error message for each attempts in retry.
func getEndErrorMessage(opName string, attempts int, startTime time.Time) string {
	return fmt.Sprintf("%s: failed %d attempts took %0.2f seconds", opName, attempts, time.Now().Sub(startTime).Seconds())
}
