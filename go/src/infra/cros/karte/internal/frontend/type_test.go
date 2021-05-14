// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"

	kartepb "infra/cros/karte/api"
)

func TestConvertActionEntity(t *testing.T) {
	t.Parallel()
	entity := &ActionEntity{
		ID: "FAKE-ENTITY-ID",
	}
	expectedAction := &kartepb.Action{
		Name: "FAKE-ENTITY-ID",
	}
	action := ConvertActionEntityToAction(entity)
	if action == nil {
		t.Errorf("action unexpectedly nil")
	}
	if diff := cmp.Diff(expectedAction, action, protocmp.Transform()); diff != "" {
		t.Errorf("unexpected diff: %s", diff)
	}
}

func TestConvertActionEntities(t *testing.T) {
	t.Parallel()
	entities := []*ActionEntity{
		{
			ID: "FAKE-ENTITY-ID",
		},
	}
	expectedActions := []*kartepb.Action{
		{
			Name: "FAKE-ENTITY-ID",
		},
	}
	actions := ConvertActionEntitiesToActions(entities...)
	if actions == nil {
		t.Errorf("actions unexpectedly nil")
	}
	if len(actions) == 0 {
		t.Errorf("actions unexpectedly empty")
	}
	if actions[0] == nil {
		t.Errorf("bad")
	}
	if diff := cmp.Diff(expectedActions, actions, protocmp.Transform()); diff != "" {
		t.Errorf("unexpected diff: %s", diff)
	}
}
