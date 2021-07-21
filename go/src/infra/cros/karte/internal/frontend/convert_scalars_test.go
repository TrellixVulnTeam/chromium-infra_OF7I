// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"testing"
	"testing/quick"
	"time"
)

// Test that conversions to and from timestamps and time.Time values roundtrip.
func TestRoundTripTimeToTimestampToTime(t *testing.T) {
	t.Parallel()
	// time.Time contains unexported fields, so we can't test it directly.
	f := func(d time.Duration) bool {
		var zero time.Time
		x := zero.Add(d)
		x2 := convertTimestampPtrToTime(convertTimeToTimestampPtr(x))
		return x.Equal(x2)
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}
