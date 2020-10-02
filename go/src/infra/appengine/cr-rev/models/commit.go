// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package models

import (
	"context"
	"infra/appengine/cr-rev/common"

	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/gae/service/datastore"
)

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

// SameRepoAs compares itself against commit c2 and returns true if host and
// repository are identical.
func (c1 Commit) SameRepoAs(c2 Commit) bool {
	return c1.Host == c2.Host && c1.Repository == c2.Repository
}

// FindCommitsByHash returns all commits that match exact hash. Same hash is
// likely to happen only on mirrors and forks.
func FindCommitsByHash(ctx context.Context, hash string) ([]*Commit, error) {
	commits := []*Commit{}
	q := datastore.NewQuery("Commit").Eq("CommitHash", hash)
	err := datastore.GetAll(ctx, q, &commits)
	if err != nil {
		return nil, err
	}
	return commits, nil
}

// PersistCommits converts list of commits to Datastore structs and stores them
// in Datastore. It returns (true, nil) if last commit in the list is already
// in database, indicating that further traversal may not be needed.
func PersistCommits(ctx context.Context, commits []*common.GitCommit) (bool, error) {
	if len(commits) == 0 {
		return true, nil
	}

	docs := make([]*Commit, len(commits), len(commits))
	for i, commit := range commits {
		docs[i] = &Commit{
			ID:            commit.ID(),
			CommitHash:    commit.Hash,
			CommitMessage: commit.CommitMessage,
			Host:          commit.Repository.Host,
			Repository:    commit.Repository.Name,
		}
		position, err := commit.GetPositionNumber()
		switch err {
		case nil:
			docs[i].PositionRef = position.Name
			docs[i].PositionNumber = position.Number
		case common.ErrNoPositionFooter:
			logging.Debugf(ctx, "No position footer for commit: %s", docs[i].ID)
		case common.ErrInvalidPositionFooter:
			logging.Warningf(ctx, "Malformed position footer for commit: %s", docs[i].ID)
		}
	}

	// If last entry is already in the database, it's safe to stop import.
	safeToStopImport := true
	lastDoc := &Commit{
		ID: docs[len(docs)-1].ID,
	}
	err := datastore.Get(ctx, lastDoc)
	if err == datastore.ErrNoSuchEntity || err != nil {
		safeToStopImport = false
	}

	err = datastore.Put(ctx, docs)
	if err != nil {
		return false, err
	}
	return safeToStopImport, nil
}
