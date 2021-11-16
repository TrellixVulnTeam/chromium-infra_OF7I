// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"go.chromium.org/luci/appengine/gaetesting"
	"go.chromium.org/luci/gae/service/datastore"

	// See https://bugs.chromium.org/p/chromium/issues/detail?id=1242998 for details.
	// TODO(gregorynisbet): Remove this once new behavior is default.
	_ "go.chromium.org/luci/gae/service/datastore/crbug1242998safeget"

	kartepb "infra/cros/karte/api"
	"infra/cros/karte/internal/idstrategy"
	"infra/cros/karte/internal/scalars"
)

// TestGetMostRecentAction tests that we can get the most recent action of any kind in the datastore db.
func TestGetMostRecentAction(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContext()
	ctx = idstrategy.Use(ctx, idstrategy.NewNaive())

	datastore.GetTestable(ctx).Consistent(true)
	k := NewKarteFrontend()

	_, err := k.CreateAction(
		ctx,
		&kartepb.CreateActionRequest{
			Action: &kartepb.Action{
				Kind:       "foo",
				CreateTime: scalars.ConvertTimeToTimestampPtr(time.Unix(1, 2)),
			},
		},
	)
	if err != nil {
		t.Errorf("failed to insert")
	}

	_, err = k.CreateAction(
		ctx,
		&kartepb.CreateActionRequest{
			Action: &kartepb.Action{
				Kind:       "bar",
				CreateTime: scalars.ConvertTimeToTimestampPtr(time.Unix(1, 2)),
			},
		},
	)
	if err != nil {
		t.Errorf("failed to insert")
	}

	resp, err := k.ListActions(ctx, &kartepb.ListActionsRequest{
		PageSize: 1,
		Filter:   "",
	})
	if err != nil {
		t.Errorf("unexpected error while fetching actions: %s", err)
	}

	expected := "bar"
	actual := resp.GetActions()[0].GetKind()
	if diff := cmp.Diff(expected, actual); diff != "" {
		t.Errorf("unexpected diff (-want +got): %s", diff)
	}
}

// TestGetMostRecentActionInKind tests that we can get the most recent action of a given kind.
func TestGetMostRecentActionInKind(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContext()
	ctx = idstrategy.Use(ctx, idstrategy.NewNaive())

	datastore.GetTestable(ctx).Consistent(true)
	k := NewKarteFrontend()

	_, err := k.CreateAction(
		ctx,
		&kartepb.CreateActionRequest{
			Action: &kartepb.Action{
				Kind:       "ssh-attempt",
				FailReason: "1",
				CreateTime: scalars.ConvertTimeToTimestampPtr(time.Unix(1, 2)),
			},
		},
	)
	if err != nil {
		t.Errorf("failed to insert")
	}

	_, err = k.CreateAction(
		ctx,
		&kartepb.CreateActionRequest{
			Action: &kartepb.Action{
				Kind:       "ssh-attempt",
				FailReason: "2",
				CreateTime: scalars.ConvertTimeToTimestampPtr(time.Unix(1, 2)),
			},
		},
	)
	if err != nil {
		t.Errorf("failed to insert")
	}

	_, err = k.CreateAction(
		ctx,
		&kartepb.CreateActionRequest{
			Action: &kartepb.Action{
				Kind:       "flash-firmware",
				FailReason: "3",
				CreateTime: scalars.ConvertTimeToTimestampPtr(time.Unix(1, 2)),
			},
		},
	)
	if err != nil {
		t.Errorf("failed to insert")
	}

	resp, err := k.ListActions(ctx, &kartepb.ListActionsRequest{
		PageSize: 1,
		Filter:   `kind == "ssh-attempt"`,
	})
	if err != nil {
		t.Errorf("unexpected error while fetching actions: %s", err)
	}

	expected := "2"
	var actual string
	if len(resp.GetActions()) > 0 {
		actual = resp.GetActions()[0].GetFailReason()
	}
	if diff := cmp.Diff(expected, actual); diff != "" {
		t.Errorf("unexpected diff (-want +got): %s", diff)
	}
}
