// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package models

// Commit represents a document in datastore. Commit is generated and persisted
// exclusively by the backend service, during initial repository import or
// while receiving pubsub messages. Once persisted, commit document shouldn't
// be changed.
// The frontend service queries commits either by {CommitHash} or by
// {Repository, PositionRef, PositionNumber}.
type Commit struct {
	ID            string `gae:"$id"`
	Host          string
	Repository    string
	CommitHash    string
	CommitMessage string `gae:",noindex"`

	// PositionRef is extracted from Git footer. If the footer is not
	// present, it has zero value. If non-zero, PositionNumber is also
	// non-zero.
	PositionRef string

	// PositionNumber is extracted from Git footer. If the footer is not
	// present, it has zero value. If non-zero, PositionRef is also
	// non-zero.
	PositionNumber int
}
