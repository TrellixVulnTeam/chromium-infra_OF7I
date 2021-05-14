// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"testing"

	"go.chromium.org/luci/appengine/gaetesting"
	"go.chromium.org/luci/gae/service/datastore"
)

func TestReadActionEntityFromEmptyDatastore(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContext()
	datastore.GetTestable(ctx).Consistent(true)
	entities, err := GetActionEntities(ctx, 100)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if len(entities) != 0 {
		t.Errorf("unexpected entities: %v", entities)
	}
}

func TestReadSingleActionEntityFromDatastore(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContext()
	datastore.GetTestable(ctx).Consistent(true)
	if err := PutActionEntities(ctx, &ActionEntity{
		ID: "hi",
	}); err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	entities, err := GetActionEntities(ctx, 100)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if len(entities) != 1 {
		t.Errorf("unexpected entities: %v", entities)
	}
}
