// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"testing"

	"go.chromium.org/luci/appengine/gaetesting"
	"go.chromium.org/luci/gae/service/datastore"

	// See https://bugs.chromium.org/p/chromium/issues/detail?id=1242998 for details.
	// TODO(gregorynisbet): Remove this once new behavior is default.
	_ "go.chromium.org/luci/gae/service/datastore/crbug1242998safeget"
)

// TestListActionsWithFilter tests listing actions with a simple filter.
func TestListActionsWithFilter(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContext()
	datastore.GetTestable(ctx).Consistent(true)
	if err := PutActionEntities(
		ctx,
		&ActionEntity{ID: "hi", Kind: "w"},
		&ActionEntity{ID: "hi2", Kind: "w"},
		&ActionEntity{ID: "hi3", Kind: "a"},
	); err != nil {
		t.Errorf("putting entities: %s", err)
	}
	q, err := newActionEntitiesQuery("", "kind == \"w\"")
	if err != nil {
		t.Errorf("building query: %s", err)
	}
	es, _, err := q.Next(ctx, 10)
	if err != nil {
		t.Errorf("running query: %s", err)
	}
	if len(es) != 2 {
		t.Errorf("unexpected entities: %v", es)
	}
}
