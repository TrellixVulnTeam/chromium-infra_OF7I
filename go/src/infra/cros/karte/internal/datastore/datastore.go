// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// This package exposes a subset of the functionality of the LUCI datastore library.
// This is the only place in the Karte project that directly uses this library.
// This package makes it easy to add additional logging or validation on datastore API
// calls.

package datastore

import (
	"context"

	"go.chromium.org/luci/gae/service/datastore"
)

// Query is a potentially invalid datastore query.
type Query = datastore.Query

// Cursor is a wrapper around a datastore cursor.
type Cursor = datastore.Cursor

// CursorCB is a cursor callback.
type CursorCB = datastore.CursorCB

// DecodeCursor converts a stringified cursor to a cursor instance.
func DecodeCursor(ctx context.Context, s string) (Cursor, error) {
	return datastore.DecodeCursor(ctx, s)
}

// NewQuery creates a new query for the kind in question.
func NewQuery(kind string) *Query {
	return datastore.NewQuery(kind)
}

// Run runs a query.
func Run(ctx context.Context, q *Query, cb interface{}) error {
	return datastore.Run(ctx, q, cb)
}

// Put adds new items into datastore.
func Put(ctx context.Context, src ...interface{}) error {
	return datastore.Put(ctx, src...)
}

// Get gets new items from datastore.
func Get(ctx context.Context, dst ...interface{}) error {
	return datastore.Get(ctx, dst...)
}
