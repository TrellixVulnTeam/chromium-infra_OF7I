// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package retry provides retry methods.
package retry

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.chromium.org/luci/common/errors"
)

// TODO(otabek@): Migrate to custom logger interface.
// Note: Context is required for all retries and will be used with new logger in further CLs.

// WithTimeout retries execute function in giving time duration.
// TODO(otabek@): Add example of usage the documentation.
func WithTimeout(ctx context.Context, interval, duration time.Duration, f func() error, opName string) (err error) {
	startTime := time.Now()
	maxTime := startTime.Add(duration)
	var attempts int
	for {
		attempts++
		if err = f(); err == nil {
			log.Printf(getSuccessMessage(opName, attempts, startTime))
			return
		}
		log.Printf(
			"Retry %q: attempt %d (used %0.2f of %0.2f seconds), error: %s",
			opName,
			attempts,
			time.Now().Sub(startTime).Seconds(),
			duration.Seconds(),
			err)
		if time.Now().Add(interval).After(maxTime) {
			break
		}
		time.Sleep(interval)
	}
	return errors.Annotate(err, getEndErrorMessage(opName, attempts, startTime)).Err()
}

// LimitCount retries execute function with limit by numbers attempts.
// TODO(otabek@): Add example of usage the documentation.
func LimitCount(ctx context.Context, count int, interval time.Duration, f func() error, opName string) (err error) {
	startTime := time.Now()
	var attempts int
	for {
		attempts++
		if err = f(); err == nil {
			log.Printf(getSuccessMessage(opName, attempts, startTime))
			return
		}
		log.Printf("Retry %q: attempts %d of %d, error: %s", opName, attempts, count, err)
		if attempts >= count {
			break
		}
		time.Sleep(interval)
	}
	return errors.Annotate(err, getEndErrorMessage(opName, attempts, startTime)).Err()
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
