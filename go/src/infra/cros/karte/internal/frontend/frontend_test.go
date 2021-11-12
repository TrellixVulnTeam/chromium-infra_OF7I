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
	"google.golang.org/protobuf/testing/protocmp"

	// See https://bugs.chromium.org/p/chromium/issues/detail?id=1242998 for details.
	// TODO(gregorynisbet): Remove this once new behavior is default.
	_ "go.chromium.org/luci/gae/service/datastore/crbug1242998safeget"

	kartepb "infra/cros/karte/api"
	"infra/cros/karte/internal/idstrategy"
	"infra/cros/karte/internal/scalars"
)

// TestCreateAction makes sure that CreateAction returns the action it created and that the action is present in datastore.
func TestCreateAction(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContext()
	ctx = idstrategy.Use(ctx, idstrategy.NewNaive())
	datastore.GetTestable(ctx).Consistent(true)
	k := NewKarteFrontend()
	resp, err := k.CreateAction(ctx, &kartepb.CreateActionRequest{
		Action: &kartepb.Action{
			Name:       "",
			Kind:       "ssh-attempt",
			CreateTime: scalars.ConvertTimeToTimestampPtr(time.Unix(1, 2)),
		},
	})
	expected := &kartepb.Action{
		Name:       "entity1",
		Kind:       "ssh-attempt",
		CreateTime: scalars.ConvertTimeToTimestampPtr(time.Unix(1, 2)),
	}
	if err != nil {
		t.Error(err)
	}
	if diff := cmp.Diff(expected, resp, protocmp.Transform()); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	// Here we inspect the contents of datastore.
	q, err := newActionEntitiesQuery("", "")
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	datastoreActionEntities, err := q.Next(ctx, 0)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if datastoreActionEntities == nil {
		t.Errorf("action entities should not be nil")
	}
	switch len(datastoreActionEntities) {
	case 0:
		t.Errorf("datastore should not be empty")
	case 1:
	default:
		t.Errorf("datastore should not have more than 1 item")
	}
}

// TestRejectActionWithUserDefinedName tests that an action with a user-defined name is rejected.
func TestRejectActionWithUserDefinedName(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContext()
	datastore.GetTestable(ctx).Consistent(true)
	k := NewKarteFrontend()
	resp, err := k.CreateAction(ctx, &kartepb.CreateActionRequest{
		Action: &kartepb.Action{
			Name: "aaaaa",
			Kind: "ssh-attempt",
		},
	})
	if resp != nil {
		t.Errorf("unexpected response: %s", resp.String())
	}
	if err == nil {
		t.Errorf("expected response to be rejected")
	}
}

// TestCreateObservation makes sure that that CreateObservation fails because
// it isn't implemented.
func TestCreateObservation(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContext()
	datastore.GetTestable(ctx).Consistent(true)
	k := NewKarteFrontend()
	_, err := k.CreateObservation(ctx, &kartepb.CreateObservationRequest{})
	if err == nil {
		t.Error("expected Create Observation to fail")
	}
}

// TestListActionsSmokeTest tests that ListActions does not error.
func TestListActionsSmokeTest(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContext()
	datastore.GetTestable(ctx).Consistent(true)
	k := NewKarteFrontend()
	resp, err := k.ListActions(ctx, &kartepb.ListActionsRequest{})
	if resp == nil {
		t.Errorf("expected resp to not be nil")
	}
	if len(resp.GetActions()) != 0 {
		t.Errorf("expected actions to be trivial")
	}
	if err != nil {
		t.Errorf("expected error to be nil not %s", err)
	}
}

// TestListActions tests that ListActions errors.
func TestListActions(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContext()
	datastore.GetTestable(ctx).Consistent(true)
	if err := PutActionEntities(
		ctx,
		&ActionEntity{
			ID: "aaaa",
		},
	); err != nil {
		t.Error(err)
	}
	k := NewKarteFrontend()
	resp, err := k.ListActions(ctx, &kartepb.ListActionsRequest{})
	if err != nil {
		t.Errorf("expected error to be nil not %s", err)
	}
	if resp == nil {
		t.Errorf("expected resp to not be nil")
	}
	if resp.GetActions() == nil {
		t.Errorf("expected actions to not be nil")
	}
	if len(resp.GetActions()) != 1 {
		t.Errorf("expected len(actions) to be 1 not %d", len(resp.GetActions()))
	}
}

// TestListObservations tests that ListObservations errors.
func TestListObservations(t *testing.T) {
	t.Parallel()
	k := NewKarteFrontend()
	ctx := gaetesting.TestingContext()
	datastore.GetTestable(ctx).Consistent(true)
	resp, err := k.ListObservations(ctx, &kartepb.ListObservationsRequest{})
	if resp == nil {
		t.Errorf("expected resp to not be nil")
	}
	if err != nil {
		t.Errorf("expected error to be nil not %s", err)
	}
}
