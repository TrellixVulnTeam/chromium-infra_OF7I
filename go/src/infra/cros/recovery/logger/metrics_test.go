// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package logger

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// TestNewMetrics verifies that the default metrics logger writes a serialized message to
// the provided logger at the debug level.
//
// TODO(gregorynisbet): Drop this test once the default logger implementation is more substantial.
func TestNewMetrics(t *testing.T) {
	ctx := context.Background()
	expected := []string{
		lines(
			`Create action "b": {`,
			`    "ActionKind": "b",`,
			`    "SwarmingTaskID": "a",`,
			`    "AssetTag": "",`,
			`    "StartTime": "0001-01-01T00:00:00Z",`,
			`    "StopTime": "0001-01-01T00:00:00Z",`,
			`    "Status": "",`,
			`    "FailReason": "",`,
			`    "Observations": [`,
			`        {`,
			`            "MetricKind": "c",`,
			`            "ValueType": "",`,
			`            "Value": ""`,
			`        }`,
			`    ]`,
			`}`,
		),
	}
	l := newFakeLogger().(*fakeLogger)
	m := NewLogMetrics(l)

	m.Create(
		ctx,
		&Action{
			SwarmingTaskID: "a",
			ActionKind:     "b",
			Observations: []*Observation{
				{
					MetricKind: "c",
				},
			},
		},
	)

	actual := l.messages["debug"]

	if diff := cmp.Diff(expected, actual); diff != "" {
		t.Errorf("unexpected diff: %s", diff)
	}
}

// Join a sequence of lines together to make a string with newlines inserted after
// each element.
func lines(a ...string) string {
	return fmt.Sprintf("%s\n", strings.Join(a, "\n"))
}
