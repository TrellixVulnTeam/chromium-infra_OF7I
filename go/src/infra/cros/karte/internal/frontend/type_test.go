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
	action := entity.ConvertToAction()
	if action == nil {
		t.Errorf("action unexpectedly nil")
	}
	if diff := cmp.Diff(expectedAction, action, protocmp.Transform()); diff != "" {
		t.Errorf("unexpected diff: %s", diff)
	}
}
