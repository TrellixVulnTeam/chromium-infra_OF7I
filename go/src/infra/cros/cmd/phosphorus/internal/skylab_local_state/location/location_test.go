// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// +build !windows

package location_test

import (
	"testing"

	"infra/cros/cmd/phosphorus/internal/skylab_local_state/location"
)

func TestResultsDir(t *testing.T) {
	got := location.ResultsDir("/autotest", "fooRunID1", "testID1")
	want := "/autotest/results/swarming-fooRunID0/1/testID1"
	if got != want {
		t.Fatalf("ResultsDir = %s; want = %s", got, want)
	}
}

func TestResultsDirWithoutTestId(t *testing.T) {
	got := location.ResultsDir("/autotest", "fooRunID1", "")
	want := "/autotest/results/swarming-fooRunID0/1"
	if got != want {
		t.Fatalf("ResultsDir = %s; want = %s", got, want)
	}
}
