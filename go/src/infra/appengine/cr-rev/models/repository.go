// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package models

import (
	"fmt"
	"strings"
	"time"

	"go.chromium.org/luci/gae/service/datastore"
)

const (
	// RepositoryStaleIndexingDuration defines maximum duration between
	// full_scan_lock refreshes after which indexing will be considered
	// incomplete. At that point, any backend instance of cr-rev can take
	// over indexing.
	RepositoryStaleIndexingDuration = 1 * time.Hour
)

// RepoID uniquely identifies a repository. It's used as a primary key for
// the Repository kind.
type RepoID struct {
	Host       string
	Repository string
}

// ToProperty serializes RepoID into string representation, separated by slash.
func (rID *RepoID) ToProperty() (p datastore.Property, err error) {
	if rID.Host == "" || rID.Repository == "" {
		return p, fmt.Errorf("Host or Repository not defined")
	}
	p.SetValue(
		fmt.Sprintf("%s/%s", rID.Host, rID.Repository),
		true)
	return
}

// FromProperty deserializes RepoID from its string representation.
func (rID *RepoID) FromProperty(p datastore.Property) error {
	id, ok := p.Value().(string)
	if !ok {
		return fmt.Errorf("Failed to cast property value to string")
	}
	idx := strings.Index(id, "/")
	// idx is set to -1 if not found. Valid property value can't be start
	// with slash (index 0), nor have first slash as last element.
	if idx <= 0 || idx+1 == len(id) {
		return fmt.Errorf("Unexpected ID format: %s", id)
	}
	rID.Host, rID.Repository = id[:idx], id[idx+1:]
	return nil
}

// Repository represents a document in datastore. It stores information about
// indexed repository. Before new repository is indexed, cr-rev creates entry
// in datastore with FullScanLock set to current time and FullScanLastRun to
// zero value. On successful execution, FullScanLastRun is set to current time.
type Repository struct {
	ID RepoID `gae:"$id"`

	// FullScanLastRun holds information when was last successful full
	// import. Zero value means the repository was never fully indexed.
	FullScanLastRun time.Time `gae:",noindex"`

	// FullScanLeaseStartTime, if non-zero value, indicates on-going
	// indexing. The indexing process should periodically update this
	// value. If the last update was longer than
	// RepositoryStaleIndexingDuration, the import process is considered
	// stalled (e.g. crashed, networking partition) and a new import job
	// can be triggered.
	FullScanLeaseStartTime time.Time

	// FullScanLeaseHostname tracks which hostname has the lease for the
	// repository.
	FullScanLeaseHostname string `gae:",noindex"`
}

// IsFullScanStalled returns true if the indexing didn't update Repository
// document more than RepositoryStaleIndexingDuration.
func (r *Repository) IsFullScanStalled(currentTime time.Time) bool {
	if r.FullScanLeaseStartTime.IsZero() {
		return false
	}

	return r.FullScanLastRun.Add(RepositoryStaleIndexingDuration).After(currentTime)
}

// SetIndexingCompleted marks Repository as successfully indexed and sets
// FullScanLock to zero-value.
func (r *Repository) SetIndexingCompleted(currentTime time.Time) {
	r.FullScanLastRun = currentTime
	r.FullScanLeaseStartTime = time.Time{}
	r.FullScanLeaseHostname = ""
}
