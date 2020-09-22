// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package models

import (
	"fmt"
	"os"
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

// IsScanRequired returns true if full repository scan is required. This
// happens if repository was never indexed or if previous lease expired and
// left this repository partially indexed.
func (r *Repository) IsScanRequired(currentTime time.Time) bool {
	if !r.FullScanLastRun.IsZero() {
		return false
	}
	if r.FullScanLeaseStartTime.IsZero() {
		return true
	}

	deadline := r.FullScanLeaseStartTime.Add(RepositoryStaleIndexingDuration)
	fmt.Printf("\n%v %v %v\n", currentTime, r.FullScanLeaseStartTime, deadline)
	return currentTime.After(deadline)
}

// SetIndexingCompleted marks Repository as successfully indexed and removes
// lease.
func (r *Repository) SetIndexingCompleted(currentTime time.Time) {
	r.FullScanLastRun = currentTime
	r.FullScanLeaseStartTime = time.Time{}
	r.FullScanLeaseHostname = ""
}

// SetStartIndexing marks Repository for initial import and sets lease
// information. It is on client to ensure lease is renewed periodically, by
// calling ExtendLease
func (r *Repository) SetStartIndexing(currentTime time.Time, hostname string) {
	r.FullScanLastRun, r.FullScanLeaseStartTime = time.Time{}, currentTime
	r.FullScanLeaseHostname = os.Getenv("GAE_INSTANCE")
}

// ExtendLease extends currently active lease. This function doesn't check
// lease ownership.
func (r *Repository) ExtendLease(currentTime time.Time) {
	r.FullScanLeaseStartTime = currentTime
}
