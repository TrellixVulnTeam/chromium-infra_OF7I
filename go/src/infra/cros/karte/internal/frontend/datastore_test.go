// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"go.chromium.org/luci/appengine/gaetesting"
	"go.chromium.org/luci/gae/service/datastore"

	// See https://bugs.chromium.org/p/chromium/issues/detail?id=1242998 for details.
	// TODO(gregorynisbet): Remove this once new behavior is default.
	_ "go.chromium.org/luci/gae/service/datastore/crbug1242998safeget"
)

// TestReadActionEntityFromEmptyDatastore check that a read from a consistent datastore with
// nothing in it consistently produces no action entities.
func TestReadActionEntityFromEmptyDatastore(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContext()
	datastore.GetTestable(ctx).Consistent(true)
	q, err := newActionEntitiesQuery("", "")
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	es, d, err := q.Next(ctx, 100)
	token := q.Token
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if len(es) != 0 {
		t.Errorf("unexpected entities: %v", es)
	}
	if token != "" {
		t.Errorf("expected tok to be empty not %s", token)
	}
	if diff := cmp.Diff(ActionQueryAncillaryData{}, d); diff != "" {
		t.Errorf("unexpected ancillary data (-want +got): %s", diff)
	}
}

// TestReadObservationEntityFromEmptyDatastore check that a read from a consistent datastore with
// nothing in it consistently produces no observation entities.
func TestReadObservationEntityFromEmptyDatastore(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContext()
	datastore.GetTestable(ctx).Consistent(true)
	q, err := newObservationEntitiesQuery("", "")
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	es, err := q.Next(ctx, 100)
	token := q.Token
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if len(es) != 0 {
		t.Errorf("unexpected entities: %v", es)
	}
	if token != "" {
		t.Errorf("expected tok to be empty not %s", token)
	}
}

// TestReadSingleActionEntityFromDatastore tests putting a single action entity into datastore
// and then reading it back out.
func TestReadSingleActionEntityFromDatastore(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContext()
	datastore.GetTestable(ctx).Consistent(true)
	if err := PutActionEntities(ctx, &ActionEntity{
		ID: "zzzz-hi",
	}); err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	q, err := newActionEntitiesQuery("", "")
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	es, d, err := q.Next(ctx, 100)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if len(es) != 1 {
		t.Errorf("unexpected entities: %v", es)
	}
	if diff := cmp.Diff(ActionQueryAncillaryData{
		BiggestVersion:  "zzzz",
		SmallestVersion: "zzzz",
	},
		d,
	); diff != "" {
		t.Errorf("unexpected ancillary data (-want +got): %s", diff)
	}
}

// TestReadTwoActionEntitiesFromDatastore tests putting a two action entities into datastore
// and then reading them back out.
func TestReadTwoActionEntitiesFromDatastore(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContext()
	datastore.GetTestable(ctx).Consistent(true)
	if err := PutActionEntities(ctx, &ActionEntity{
		ID: "zzzz-hi",
	}); err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if err := PutActionEntities(ctx, &ActionEntity{
		ID: "aaaa-hi",
	}); err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	q, err := newActionEntitiesQuery("", "")
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	es, d, err := q.Next(ctx, 100)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if len(es) != 2 {
		t.Errorf("unexpected entities: %v", es)
	}
	if diff := cmp.Diff(ActionQueryAncillaryData{
		BiggestVersion:  "zzzz",
		SmallestVersion: "aaaa",
	},
		d,
	); diff != "" {
		t.Errorf("unexpected ancillary data (-want +got): %s", diff)
	}
}

// TestReadSingleObservationEntityFromDatastore tests putting a single observation entity into datastore
// and then reading it back out.
func TestReadSingleObservationEntityFromDatastore(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContext()
	datastore.GetTestable(ctx).Consistent(true)
	if err := PutObservationEntities(ctx, &ObservationEntity{
		ID: "hi",
	}); err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	q, err := newObservationEntitiesQuery("", "")
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	es, err := q.Next(ctx, 100)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if len(es) != 1 {
		t.Errorf("unexpected entities: %v", es)
	}
}

func TestWriteAndReadObservationEntitiesFromDatastore(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContext()
	datastore.GetTestable(ctx).Consistent(true)
	data := []struct {
		name     string
		entities []*ObservationEntity
	}{
		{
			"empty",
			[]*ObservationEntity{},
		},
		{
			"a few observations",
			[]*ObservationEntity{
				{
					ID:         "a",
					ActionID:   "b",
					MetricKind: "c",
				},
				{
					ID:          "d",
					ActionID:    "e",
					MetricKind:  "f",
					ValueString: "g",
				},
			},
		},
		{
			"many observations",
			[]*ObservationEntity{
				{
					ID:         "a",
					ActionID:   "b",
					MetricKind: "c",
				},
				{
					ID:          "d",
					ActionID:    "e",
					MetricKind:  "f",
					ValueString: "g",
				},
				{
					ID: "1",
				},
				{
					ID: "2",
				},
				{
					ID: "3",
				},
				{
					ID: "4",
				},
			},
		},
	}

	for _, tt := range data {
		t.Run(tt.name, func(t *testing.T) {
			if err := PutObservationEntities(ctx, tt.entities...); err != nil {
				t.Errorf("test case %s %s", tt.name, err)
			}
			q, err := newObservationEntitiesQuery("", "")
			if err != nil {
				t.Errorf("test case %s %s", tt.name, err)
			}
			es, err := q.Next(ctx, 100)
			if err != nil {
				t.Errorf("test case %s %s", tt.name, err)
			}
			if diff := cmp.Diff(
				tt.entities,
				es,
				cmp.AllowUnexported(ObservationEntity{}),
				cmpopts.SortSlices(func(a *ObservationEntity, b *ObservationEntity) bool {
					return a.cmp(b) == -1
				}),
			); diff != "" {
				if len(es) == 0 && len(tt.entities) == 0 {
					// Both are tt.entities and es are empty, forgive this case.
				} else {
					t.Errorf("test case %s %s", tt.name, diff)
				}
			}
		})
	}
}

// TestReadActionEntitiesFromDatastoreOneAtAtime reads several action entities out of datastore one at a time
// using pagination.
func TestReadActionEntitiesFromDatastoreOneAtAtime(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContext()
	datastore.GetTestable(ctx).Consistent(true)
	if err := PutActionEntities(
		ctx,
		&ActionEntity{
			ID: "entity1",
		},
		&ActionEntity{
			ID: "entity2",
		},
	); err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	// Test that extracting the first ActionEntity produces some action entity.
	var resultTok string
	{
		q, err := newActionEntitiesQuery("", "")
		if err != nil {
			t.Errorf("unexpected error: %s", err)
		}
		es, _, err := q.Next(ctx, 1)
		resultTok = q.Token
		if err != nil {
			t.Errorf("unexpected error: %s", err)
		}
		if len(es) != 1 {
			t.Errorf("unexpected entities: %v", es)
		}
		if q.Token == "" {
			t.Errorf("token should not be empty!")
		}
	}
	// Test that extracting the second ActionEntity produces some action entity.
	{
		q, err := newActionEntitiesQuery(resultTok, "")
		if err != nil {
			t.Errorf("token should not be empty!")
		}
		if err != nil {
			t.Errorf("unexpected error: %s", err)
		}
		es, _, err := q.Next(ctx, 10)
		if err != nil {
			t.Errorf("unexpected error: %s", err)
		}
		if len(es) != 1 {
			t.Errorf("unexpected entities: %v", es)
		}
		// Even though there are no further entities to extract,
		// we haven't observed the end of the query yet and therefore
		// should not see an end token.
		if q.Token != "" {
			t.Errorf("unexpected token: %q", q.Token)
		}
	}
}

// TestGetActionEntityByID test retrieving an entity from the fake datastore instance by its ID.
func TestGetActionEntityByID(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContext()
	datastore.GetTestable(ctx).Consistent(true)
	// Confirm that an item does not exist should produce an error.
	t.Run("nonexistent entity", func(t *testing.T) {
		_, err := GetActionEntityByID(ctx, "hi")
		if err == nil {
			t.Errorf("getting action when datastore is empty: %s", err)
		}
		// Insert an action.
		if err := PutActionEntities(ctx, &ActionEntity{
			ID:   "hi",
			Kind: "step1",
		}); err != nil {
			t.Errorf("unexpected error: %s", err)
		}
	})
	// Confirm that we can successfully retrieve the action that we just put in.
	t.Run("read back inserted entity", func(t *testing.T) {
		expected := &ActionEntity{
			ID:   "hi",
			Kind: "step1",
		}
		actual, err := GetActionEntityByID(ctx, "hi")
		if err != nil {
			t.Errorf("getting action from non-empty datastore: %s", err)
		}
		if diff := cmp.Diff(expected, actual, cmp.AllowUnexported(ActionEntity{})); diff != "" {
			t.Errorf("unexpected diff: %s", diff)
		}
	})
	// Confirm that we get back the most recent item when writing.
	t.Run("overwrite entity", func(t *testing.T) {
		if err := PutActionEntities(ctx, &ActionEntity{
			ID:   "hi",
			Kind: "step2",
		}); err != nil {
			t.Errorf("unexpected error: %s", err)
		}
		expected := &ActionEntity{
			ID:   "hi",
			Kind: "step2",
		}
		actual, err := GetActionEntityByID(ctx, "hi")
		if err != nil {
			t.Errorf("getting action from non-empty datastore: %s", err)
		}
		if diff := cmp.Diff(expected, actual, cmp.AllowUnexported(ActionEntity{})); diff != "" {
			t.Errorf("unexpected diff: %s", diff)
		}
	})
}

// TestSetActionEntityFields tests updating action fields.
func TestSetActionEntityFields(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		fields   []string
		src      *ActionEntity
		dst      *ActionEntity
		expected *ActionEntity
	}{
		{
			name:     "nil",
			fields:   nil,
			src:      nil,
			dst:      nil,
			expected: nil,
		},
		{
			name:     "empty maps",
			fields:   nil,
			src:      &ActionEntity{},
			dst:      &ActionEntity{},
			expected: &ActionEntity{},
		},
		{
			name:     "update time -- nil fields",
			fields:   nil,
			src:      &ActionEntity{StopTime: time.Unix(1, 2)},
			dst:      &ActionEntity{StopTime: time.Unix(3, 4)},
			expected: &ActionEntity{StopTime: time.Unix(1, 2)},
		},
		{
			// Empty fields should be treated identically to nil fields.
			name:     "update time -- empty fields",
			fields:   []string{},
			src:      &ActionEntity{StopTime: time.Unix(1, 2)},
			dst:      &ActionEntity{StopTime: time.Unix(3, 4)},
			expected: &ActionEntity{StopTime: time.Unix(1, 2)},
		},
		{
			name:     "blocked update",
			fields:   []string{"swarming_task_id"},
			src:      &ActionEntity{CreateTime: time.Unix(1, 2)},
			dst:      &ActionEntity{CreateTime: time.Unix(3, 4)},
			expected: &ActionEntity{CreateTime: time.Unix(3, 4)},
		},
		{
			name:   "fail reason",
			fields: nil,
			src: &ActionEntity{
				FailReason: "a",
			},
			dst: &ActionEntity{
				FailReason: "b",
			},
			expected: &ActionEntity{
				FailReason: "a",
			},
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			setActionEntityFields(tt.fields, tt.src, tt.dst)
			expected := tt.expected
			actual := tt.dst
			if diff := cmp.Diff(expected, actual, cmp.AllowUnexported(ActionEntity{})); diff != "" {
				t.Errorf("unexpected diff (-want +got): %s", diff)
			}
		})
	}
}
