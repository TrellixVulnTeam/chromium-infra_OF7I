// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// +build !windows

package dutstate

import (
	"testing"
)

func TestResultsDir(t *testing.T) {
	got := ResultsDir("/autotest", "deadbeef1")
	want := "/autotest/results/swarming-deadbeef0/1"
	if got != want {
		t.Fatalf("ResultsDir = %s; want = %s", got, want)
	}
}
