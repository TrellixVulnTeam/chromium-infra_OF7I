// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package engine

import (
	"context"
	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/logger/metrics"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

// TestRecordAction tests recording an action in Karte.
func TestRecordAction(t *testing.T) {
	t.Parallel()

	expected := []*metrics.Action{
		{
			ActionKind: "action:aaaa",
			Status:     "success",
		},
		{
			ActionKind: "action:aaaa",
			Status:     "success",
		},
	}
	ctx := context.Background()
	m := newFakeMetrics()
	r := &recoveryEngine{
		args: &execs.RunArgs{
			Metrics: m,
		},
	}
	a := &metrics.Action{}
	closer := r.recordAction(ctx, "aaaa", a)
	if closer == nil {
		t.Errorf("closer is unexpectedly nil")
	}
	closer(nil)

	// TODO(gregorynisbet): Mock the time.Now() function everywhere instead of removing times
	// from test cases.
	for i := range m.actions {
		var zero time.Time
		m.actions[i].StartTime = zero
	}

	if diff := cmp.Diff(expected, m.actions); diff != "" {
		t.Errorf("unexpected diff (-want +got): %s", diff)
	}
}
