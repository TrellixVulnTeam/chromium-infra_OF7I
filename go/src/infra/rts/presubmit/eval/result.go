// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package eval

import (
	"fmt"
	"io"
	"math"
	"time"

	"infra/rts"
)

// Result is the result of evaluation of a selection strategy.
type Result struct {
	// Thresholds are considered thresholds and their results.
	// Sorted by ascending ChangeScore.
	Thresholds []*Threshold

	// TotalRejections is the number of analyzed rejections.
	TotalRejections int
	// TotalTestFailures is the number of analyzed test failures.
	TotalTestFailures int
	// TotalDuration is the sum of analyzed test durations.
	TotalDuration time.Duration
}

// Threshold is distance threshold and its scores.
type Threshold struct {
	Value rts.Affectedness

	// PreservedRejections is the number of rejections where at least one failed
	// test was selected.
	PreservedRejections int

	// PreservedTestFailures is the number of selected failed tests.
	PreservedTestFailures int

	// SavedDuration is the sum of test durations for skipped tests.
	SavedDuration time.Duration

	// ChangeRecall is the fraction of rejections that were preserved.
	// May be NaN.
	ChangeRecall float64

	// TestRecall is the fraction of test failures that were preserved.
	// May return NaN.
	TestRecall float64

	// Savings is the fraction of test duration that was cut.
	// May return NaN.
	Savings float64
}

// Print prints the results to w.
func (r *Result) Print(w io.Writer, minChangeRecall float64) error {
	p := newPrinter(w)

	p.printf("ChangeRecall | Savings | TestRecall | Distance\n")
	p.printf("----------------------------------------------\n")
	for _, t := range r.Thresholds {
		if t.ChangeRecall < minChangeRecall {
			continue
		}
		p.printf(
			"%7s      | % 7s | %7s    | %6.3f\n",
			scoreString(t.ChangeRecall),
			scoreString(t.Savings),
			scoreString(t.TestRecall),
			t.Value.Distance,
		)
	}

	p.printf("\nbased on %d rejections, %d test failures, %s testing time\n", r.TotalRejections, r.TotalTestFailures, r.TotalDuration)

	return p.err
}

func scoreString(score float64) string {
	percentage := score * 100
	switch {
	case math.IsNaN(percentage):
		return "?"
	case percentage > 0 && percentage < 0.01:
		// Do not print it as 0.00%.
		return "<0.01%"
	case percentage > 99.99 && percentage < 100:
		// Do not print it as 100.00%.
		return ">99.99%"
	default:
		return fmt.Sprintf("%02.2f%%", percentage)
	}
}
