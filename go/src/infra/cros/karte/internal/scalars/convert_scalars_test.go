// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package scalars

import (
	"errors"
	"testing"
	"testing/quick"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// TestTimestampConversionRoundTrips checks that converting a time.Time to a timestampPtr and back round trips successfully.
func TestTimestampConversionRoundTrips(t *testing.T) {
	t.Parallel()
	roundTrip := func(seconds int64, nanoseconds int64) bool {
		original := time.Unix(seconds, nanoseconds)
		converted := ConvertTimeToTimestampPtr(original)
		reverted := ConvertTimestampPtrToTime(converted)
		return original.Equal(reverted)
	}
	if err := quick.Check(roundTrip, nil); err != nil {
		t.Errorf("unexpected error: %s", err)
	}
}

// TestTimestampConversionRoundTripsFromTimestampPtr tests that timestamp conversions starting from timestamppb.Timestamp roundtrip.
func TestTimestampConversionRoundTripsFromTimestampPtr(t *testing.T) {
	t.Parallel()
	roundTrip := func(seconds int64, nanoseconds int64) bool {
		original := &timestamppb.Timestamp{}
		original.Seconds = seconds
		// Enforce well-formedness of nanoseconds.
		nanos, err := mod(nanoseconds, 1000*1000*1000)
		if err != nil {
			panic(err.Error())
		}
		original.Nanos = int32(nanos)
		converted := ConvertTimestampPtrToTime(original)
		reverted := ConvertTimeToTimestampPtr(converted)
		return (original.GetSeconds() == reverted.GetSeconds()) &&
			(original.GetNanos() == reverted.GetNanos())
	}
	if err := quick.Check(roundTrip, nil); err != nil {
		t.Errorf("unexpected error: %s", err)
	}
}

// mod is like %, but guarantees that num % denom is in the range [0, denom).
func mod(num int64, denom int64) (int64, error) {
	if denom <= 0 {
		return 0, errors.New("denominator must be positive")
	}
	out := num % denom
	if out < 0 {
		return out + denom, nil
	}
	return out, nil
}
