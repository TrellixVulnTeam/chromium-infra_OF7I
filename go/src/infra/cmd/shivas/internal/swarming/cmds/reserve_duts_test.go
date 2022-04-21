// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package tasks

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"

	"infra/cmd/shivas/site"
)

// TestScheduleReserveBuilder tests that scheduling a repair builder produces the correct
// taskID and the right URL. This test does NOT emulate the buildbucket client on a deep level.
func TestScheduleReserveBuilder(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	client := &fakeClient{}

	taskInfo, err := scheduleReserveBuilder(ctx, client, site.Environment{}, "fake-labstation1", "admin-session:bla bla")
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	expected := "buildbucket:1"
	actual := taskInfo.ID
	if diff := cmp.Diff(expected, actual); diff != "" {
		t.Errorf("unexpected diff: %s", diff)
	}
	expected = "https://ci.chromium.org/p/chromeos/builders/labpack/labpack/b1"
	actual = taskInfo.TaskURL
	if diff := cmp.Diff(expected, actual); diff != "" {
		t.Errorf("unexpected diff: %s", diff)
	}
}
