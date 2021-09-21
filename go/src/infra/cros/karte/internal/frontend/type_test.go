// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/timestamppb"

	kartepb "infra/cros/karte/api"
)

// TestConvertActionEntitySmokeTest tests that an action entity can be converted to an action.
func TestConvertActionEntitySmokeTest(t *testing.T) {
	t.Parallel()
	entity := &ActionEntity{
		ID: "FAKE-ENTITY-ID",
	}
	expectedAction := &kartepb.Action{
		Name: "FAKE-ENTITY-ID",
	}
	action := entity.ConvertToAction()
	if action == nil {
		t.Errorf("action unexpectedly nil")
	}
	if diff := cmp.Diff(expectedAction, action, protocmp.Transform()); diff != "" {
		t.Errorf("unexpected diff: %s", diff)
	}
}

// TestConvertActionEntityToActionNilAction tests that converting a nil action entity succeeds.
func TestConvertActionEntityToActionNilAction(t *testing.T) {
	t.Parallel()
	var e *ActionEntity
	if e.ConvertToAction() != nil {
		t.Errorf("converting nil action failed")
	}
}

// TestConvertActionEntityToAction tests converting an action entity to an action.
func TestConvertActionEntityToAction(t *testing.T) {
	cases := []struct {
		name string
		in   *ActionEntity
		out  *kartepb.Action
	}{
		{
			name: "empty",
			in:   &ActionEntity{},
			out:  &kartepb.Action{},
		},
		{
			name: "seal time",
			in: &ActionEntity{
				SealTime: time.Unix(1, 2),
			},
			out: &kartepb.Action{
				SealTime: timestamppb.New(time.Unix(1, 2)),
			},
		},
		{
			name: "erorr reason",
			in: &ActionEntity{
				ErrorReason: "aaaa",
			},
			out: &kartepb.Action{
				FailReason: "aaaa",
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt := tt
			expected := tt.out
			actual := tt.in.ConvertToAction()
			if diff := cmp.Diff(expected, actual, protocmp.Transform()); diff != "" {
				t.Errorf("unexpected error (-want +got): %s", diff)
			}
			expectedRoundTrip := tt.in
			actualRoundTrip, err := convertActionToActionEntity(actual)
			if err != nil {
				t.Errorf("unexpected error during round trip conversion: %s", err)
			}
			roundTripDiff := cmp.Diff(expectedRoundTrip, actualRoundTrip, cmp.AllowUnexported(ActionEntity{}))
			if roundTripDiff != "" {
				t.Errorf("unexpected diff during round trip (-want +got): %s", roundTripDiff)
			}
		})
	}
}
