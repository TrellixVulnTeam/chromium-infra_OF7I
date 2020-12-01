// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package main

import (
	"context"
	"log"

	"cloud.google.com/go/datastore"
)

// CursorSaver tracks previously consumed cursor and currently consumed cursor
// for given Kind. This is useful when Datastore Kind is being scanned
// sequentially.
type CursorSaver struct {
	DatastoreKey *datastore.Key
	dsClient     *datastore.Client
	oldCursor    *datastore.Cursor
	newCursor    *datastore.Cursor
}

// StaticGenCursors is document that is persisted into Datastore.
type StaticGenCursors struct {
	// Cursor is serialized interpretation of Datastore cursor
	Cursor string `datastore:",noindex"`
}

// NewCursorSaver initializes CursorSaver instance.
func NewCursorSaver(ctx context.Context, dsClient *datastore.Client, kind string) *CursorSaver {
	return &CursorSaver{
		DatastoreKey: &datastore.Key{
			Kind: "StaticGenCursors",
			Name: kind,
		},
		dsClient: dsClient,
	}
}

// UpdateCursor stores provided cursor and persists old cursor into Datastore,
// if present.
func (cs *CursorSaver) UpdateCursor(ctx context.Context, c *datastore.Cursor) {
	if cs.oldCursor != nil {
		sgc := &StaticGenCursors{
			Cursor: cs.oldCursor.String(),
		}
		_, err := cs.dsClient.Put(ctx, cs.DatastoreKey, sgc)
		if err != nil {
			panic(err)
		}
	}
	cs.oldCursor = cs.newCursor
	cs.newCursor = c
}

// RestoreCursor retrieves last saved cursor from Datastore, if any. If there
// is an error querying cursor, it returns nil.
func (cs *CursorSaver) RestoreCursor(ctx context.Context) *datastore.Cursor {
	sgc := &StaticGenCursors{}
	err := cs.dsClient.Get(ctx, cs.DatastoreKey, sgc)

	if err != nil {
		log.Printf(
			"Error querying datastore for cursor position: %s",
			err.Error())
		return nil
	}
	c, err := datastore.DecodeCursor(sgc.Cursor)
	if err != nil {
		return nil
	}
	return &c
}
