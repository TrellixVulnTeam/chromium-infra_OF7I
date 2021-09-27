// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package karte

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"

	kartepb "infra/cros/karte/api"
	"infra/cros/recovery/logger"
)

// TestConvertActionToKarteAction tests conversion from an action internal to
// logger to Karte's notion of an action.
func TestConvertActionToKarteAction(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		input  *logger.Action
		output *kartepb.Action
	}{
		{
			name:   "nil action",
			input:  nil,
			output: nil,
		},
		{
			name:   "empty action",
			input:  &logger.Action{},
			output: &kartepb.Action{},
		},
		{
			name: "full action",
			input: &logger.Action{
				Name:           "name",
				ActionKind:     "a",
				SwarmingTaskID: "b",
				AssetTag:       "c",
				StartTime:      time.Unix(1, 2),
				StopTime:       time.Unix(3, 4),
				Status:         logger.ActionStatusFail,
				FailReason:     "w",
				Observations: []*logger.Observation{
					{
						MetricKind: "aa",
						ValueType:  "bb",
						Value:      "cc",
					},
				},
			},
			output: &kartepb.Action{
				Name:           "name",
				Kind:           "a",
				SwarmingTaskId: "b",
				AssetTag:       "c",
				StartTime:      convertTimeToProtobufTimestamp(time.Unix(1, 2)),
				StopTime:       convertTimeToProtobufTimestamp(time.Unix(3, 4)),
				FailReason:     "w",
				Status:         kartepb.Action_FAIL,
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			expected := tt.output
			actual := convertActionToKarteAction(tt.input)
			if diff := cmp.Diff(expected, actual, protocmp.Transform()); diff != "" {
				t.Errorf("unexpected diff (-want +got): %s", diff)
			}
		})
	}
}

// TestConvertKarteActionToAction tests conversion from a Karte action back to an action.
func TestConvertKarteActionToAction(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		input  *kartepb.Action
		output *logger.Action
	}{
		{
			name:   "nil action",
			input:  nil,
			output: nil,
		},
		{
			name:   "empty action",
			input:  &kartepb.Action{},
			output: &logger.Action{},
		},
		{
			name: "non-empty action",
			input: &kartepb.Action{
				Name:       "a",
				FailReason: "w",
			},
			output: &logger.Action{
				Name:       "a",
				FailReason: "w",
			},
		},
		{
			name: "full action",
			input: &kartepb.Action{
				Name:           "name",
				Kind:           "a",
				SwarmingTaskId: "b",
				AssetTag:       "c",
				StartTime:      convertTimeToProtobufTimestamp(time.Unix(1, 2)),
				StopTime:       convertTimeToProtobufTimestamp(time.Unix(3, 4)),
				FailReason:     "w",
				Status:         kartepb.Action_FAIL,
			},
			output: &logger.Action{
				Name:           "name",
				ActionKind:     "a",
				SwarmingTaskID: "b",
				AssetTag:       "c",
				StartTime:      time.Unix(1, 2),
				StopTime:       time.Unix(3, 4),
				Status:         logger.ActionStatusFail,
				FailReason:     "w",
				Observations:   nil,
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			expected := tt.output
			actual := convertKarteActionToAction(tt.input)
			if diff := cmp.Diff(expected, actual, cmp.AllowUnexported(logger.Action{})); diff != "" {
				t.Errorf("unexpected diff (-want +got): %s", diff)
			}
		})
	}
}

// TestConvertActionRoundTrip tests that converting an action to a karte action and back results in the same action.
func TestConvertActionRoundTrip(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		input *logger.Action
	}{
		{
			name: "simple action",
			input: &logger.Action{
				Name: "aaaaa",
			},
		},
		{
			name: "complex action",
			input: &logger.Action{
				Name:           "aaaaaaaaaa",
				ActionKind:     "a",
				SwarmingTaskID: "b",
				AssetTag:       "c",
				StartTime:      time.Unix(1, 2),
				StopTime:       time.Unix(3, 4),
				Status:         logger.ActionStatusFail,
				FailReason:     "w",
			},
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			expected := tt.input
			actual := convertKarteActionToAction(convertActionToKarteAction(tt.input))
			if diff := cmp.Diff(expected, actual, cmp.AllowUnexported(logger.Action{})); diff != "" {
				t.Errorf("unexpected diff (-want +got): %s", diff)
			}
		})
	}
}
