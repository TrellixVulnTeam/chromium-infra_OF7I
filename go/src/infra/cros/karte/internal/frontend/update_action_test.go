// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.chromium.org/luci/appengine/gaetesting"
	"go.chromium.org/luci/gae/service/datastore"

	// See https://bugs.chromium.org/p/chromium/issues/detail?id=1242998 for details.
	// TODO(gregorynisbet): Remove this once new behavior is default.
	_ "go.chromium.org/luci/gae/service/datastore/crbug1242998safeget"
)

// TestUpdateActionEntity tests writing an action entity to datastore, updating it, and then reading it back.
func TestUpdateActionEntity(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContext()
	datastore.GetTestable(ctx).Consistent(true)
	if err := PutActionEntities(
		ctx,
		&ActionEntity{ID: "hi", Kind: "w"},
	); err != nil {
		t.Errorf("putting entities: %s", err)
	}
	// We record actual1 to check whether UpdateActionEntity returns the correct entity.
	// UpdateActionEntity does not perform a get to datastore after writing.
	actual1, err := UpdateActionEntity(
		ctx,
		&ActionEntity{
			ID:   "hi",
			Kind: "x",
		},
		[]string{"kind"},
	)
	if err != nil {
		t.Errorf("updating entities: %s", err)
	}
	// Actual is the read value.
	actual2, err := GetActionEntityByID(ctx, "hi")
	if err != nil {
		t.Errorf("reading back entities: %s", err)
	}
	expected := &ActionEntity{
		ID:   "hi",
		Kind: "x",
	}
	for _, actual := range []*ActionEntity{actual1, actual2} {
		if diff := cmp.Diff(expected, actual, cmp.AllowUnexported(ActionEntity{})); diff != "" {
			t.Errorf("unexpected error: %s", diff)
		}
	}
}
