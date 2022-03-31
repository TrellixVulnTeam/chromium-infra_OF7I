// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cron

import (
	"context"
	"os"
	"strings"
	"time"

	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/data/rand/mathrand"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/runtime/paniccatcher"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type DurationType int

const (
	EVERY    = iota // Run the task every minInterval
	HOURLY          // Run the task every hour. At minInterval(<60 Minutes) after the hour
	DAILY           // Run the task every day. At minInterval(<24 Hours) after 00:00
	WEEKDAYS        // Run the task every weekday. At minInterval(<24 Hours) after 00:00
	WEEKEND         // Run the task everyweekend. At minInterval(<48Hours) after 00:00
)

// CronTab describes the job to be run by cron
type CronTab struct {
	Name     string                          // Name of the job
	Time     time.Duration                   // Min inteval between triggers
	TrigType DurationType                    // Refer to the const above for available options
	Job      func(ctx context.Context) error // Target routine to trigger
	preempt  chan int                        // Int channel to preempt timer and trigger the job
}

// estimateTriggerTime checks to see if start + interval > start + quanta. If that happens, (ex: Hourly mode
// triggered with 65 minutes of interval) it throws a warning and returns trigger for next available trigger
// window without interval. If the estimated trigger time has already passed, it returns next available one.
func estimateTriggerTime(ctx context.Context, start time.Time, interval, quanta time.Duration) time.Time {
	if interval >= quanta {
		logging.Warningf(ctx, "Ignoring %v interval (>= %v)", interval, quanta)
		// Trigger the next quanta as we don't know if we can trigger this quanta
		return truncateInZone(ctx, start, quanta).Add(quanta)
	}
	tt := truncateInZone(ctx, start, quanta).Add(interval) // Time to trigger
	if !tt.After(start) {
		logging.Warningf(ctx, "Missed trigger window for %v. Trying %v", tt, tt.Add(quanta))
		// If the trigger time has already passed. Try next quanta
		tt = tt.Add(quanta)
	}
	return tt
}

// truncateInZone truncates the given time to quanta without assuming UTC first.
func truncateInZone(ctx context.Context, t time.Time, quanta time.Duration) time.Time {
	switch quanta {
	case 24 * time.Hour:
		// If truncating for a day remove all hours, minutes, seconds and nanoseconds
		d := time.Duration(t.Hour())*time.Hour + time.Duration(t.Minute())*time.Minute + time.Duration(t.Second())*time.Second + time.Duration(t.Nanosecond())*time.Nanosecond
		return t.Add(-d)
	case time.Hour:
		// If truncating for an hour remove  minutes, seconds and nanoseconds
		d := time.Duration(t.Minute())*time.Minute + time.Duration(t.Second())*time.Second + time.Duration(t.Nanosecond())*time.Nanosecond
		return t.Add(-d)
	default:
		logging.Warningf(ctx, "Using truncate with respect to UTC. Might result in estimates being ~7 hours off")
		return t.Truncate(quanta)
	}
}

// skipDays is a helper function to esimate Weekdays and Weekends mode.
func skipDays(tt time.Time, weekend bool) time.Time {
	switch tt.Weekday() {
	case time.Monday:
		if !weekend {
			return tt
		}
		return tt.Add(5 * 24 * time.Hour)
	case time.Tuesday:
		if !weekend {
			return tt
		}
		return tt.Add(4 * 24 * time.Hour)
	case time.Wednesday:
		if !weekend {
			return tt
		}
		return tt.Add(3 * 24 * time.Hour)
	case time.Thursday:
		if !weekend {
			return tt
		}
		return tt.Add(2 * 24 * time.Hour)
	case time.Friday:
		if !weekend {
			return tt
		}
		return tt.Add(1 * 24 * time.Hour)
	case time.Saturday:
		if weekend {
			return tt
		}
		return tt.Add(2 * 24 * time.Hour)
	case time.Sunday:
		if weekend {
			return tt
		}
		return tt.Add(1 * 24 * time.Hour)
	}
	// ideally this will never execute
	return tt
}

// Trigger triggers a crontab immediately.
func Trigger(cronTab *CronTab) (err error) {
	defer func() {
		// The write to channel panics if the channel is closed.
		if r := recover(); r != nil {
			err = status.Errorf(codes.AlreadyExists,
				"Cannot trigger %s. Job might already be running. %v", cronTab.Name, r)
			return
		}
	}()
	// Send a signal on the preempt channel to trigger the job.
	cronTab.preempt <- 1
	return nil
}

// Run runs cronTab.Job repeatedly, until the context is cancelled..
func Run(ctx context.Context, cronTab *CronTab) {
	defer logging.Warningf(ctx, "Exiting cron")

	// call calls f and catches a panic, will stop once the whole context is cancelled.
	call := func(ctx context.Context) error {
		defer paniccatcher.Catch(func(p *paniccatcher.Panic) {
			logging.Errorf(ctx, "Caught panic: %s\n%s", p.Reason, p.Stack)
		})
		return cronTab.Job(ctx)
	}
	// Run all tasks with MTV time ref.
	location, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		panic(err)
	}

	var count = 0
	for {
		start := clock.Now(ctx)
		start = start.In(location)
		var trigTime time.Time
		switch cronTab.TrigType {
		case EVERY:
			// Introduce env DUMPER_IMMEDIATELY_TRIGGER for local test
			// Otherwise, just add the interval specified to the start time.
			if strings.ToLower(os.Getenv("DUMPER_IMMEDIATELY_TRIGGER")) == "true" && count == 0 {
				trigTime = start
			} else {
				trigTime = start.Add(cronTab.Time)
			}

		case HOURLY:
			trigTime = estimateTriggerTime(ctx, start, cronTab.Time, 1*time.Hour)

		case DAILY:
			trigTime = estimateTriggerTime(ctx, start, cronTab.Time, 24*time.Hour)

		case WEEKDAYS:
			trigTime = estimateTriggerTime(ctx, start, cronTab.Time, 24*time.Hour)
			trigTime = skipDays(trigTime, false)

		case WEEKEND:
			trigTime = estimateTriggerTime(ctx, start, cronTab.Time, 24*time.Hour)
			trigTime = skipDays(trigTime, true)

		default:
			// Don't start the cron if the tab is bad
			logging.Errorf(ctx, "Unable to trigger %s. Bad type of trigger", cronTab.Name)
			return
		}

		// Wait until trigTime.
		if sleep := time.Until(trigTime); sleep > 0 {
			// Add jitter: +5% of sleep time to desynchronize cron jobs.
			sleep = sleep + time.Duration(mathrand.Intn(ctx, int(sleep/20)))
			timer := time.NewTimer(sleep)
			cronTab.preempt = make(chan int)
			select {
			case <-timer.C:
			case <-cronTab.preempt:
				// Stop the timer
				timer.Stop()
			case <-ctx.Done():
				return
			}
			// Close the channel. This will disable trigger when the job is running..
			close(cronTab.preempt)
		}

		if err := call(ctx); err != nil {
			logging.Errorf(ctx, "Iteration failed: %s", err)
		}
		count++
	}
}
