// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package state

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"time"

	"go.chromium.org/luci/server/span"

	"infra/appengine/weetbix/internal/clustering"
)

const testProject = "myproject"

// EntryBuilder provides methods to build a new Entry.
type EntryBuilder struct {
	entry *Entry
}

// NewEntry creates a new entry builder with the given uniqifier.
// The uniqifier affects the ChunkID, AlgorithmVersion, RulesVersion
// and Algorithms.
func NewEntry(uniqifier int) *EntryBuilder {
	// Generate a 128-bit chunkID from the uniqifier.
	// Using a hash function ensures they will be approximately uniformly
	// distributed through the keyspace.
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], uint64(uniqifier))
	sum := sha256.Sum256(b[:])

	entry := &Entry{
		Project:       testProject,
		ChunkID:       hex.EncodeToString(sum[0:16]),
		PartitionTime: time.Date(2030, 1, 1, 1, 1, 1, uniqifier, time.UTC),
		ObjectID:      "abcdef1234567890abcdef1234567890",
		Clustering: clustering.ClusterResults{
			AlgorithmsVersion: int64(uniqifier + 1),
			ConfigVersion:     time.Date(2025, 2, 1, 1, 1, 1, uniqifier, time.UTC),
			RulesVersion:      time.Date(2025, 1, 1, 1, 1, 1, uniqifier, time.UTC),
			Algorithms: map[string]struct{}{
				fmt.Sprintf("alg-%v-v1", uniqifier): {},
				"alg-extra-v1":                      {},
			},
			Clusters: [][]clustering.ClusterID{
				{
					{
						Algorithm: fmt.Sprintf("alg-%v-v1", uniqifier),
						ID:        "00112233445566778899aabbccddeeff",
					},
				},
				{
					{
						Algorithm: fmt.Sprintf("alg-%v-v1", uniqifier),
						ID:        "00112233445566778899aabbccddeeff",
					},
					{
						Algorithm: fmt.Sprintf("alg-%v-v1", uniqifier),
						ID:        "22",
					},
				},
			},
		},
	}
	return &EntryBuilder{entry}
}

// WithChunkIDPrefix specifies the start of the ChunkID to use. The remaining
// ChunkID will be derived from the uniqifier.
func (b *EntryBuilder) WithChunkIDPrefix(prefix string) *EntryBuilder {
	b.entry.ChunkID = prefix + b.entry.ChunkID[len(prefix):]
	return b
}

// WithProject specifies the LUCI project for the entry.
func (b *EntryBuilder) WithProject(project string) *EntryBuilder {
	b.entry.Project = project
	return b
}

// WithAlgorithmsVersion specifies the algorithms version for the entry.
func (b *EntryBuilder) WithAlgorithmsVersion(version int64) *EntryBuilder {
	b.entry.Clustering.AlgorithmsVersion = version
	return b
}

// WithConfigVersion specifies the config version for the entry.
func (b *EntryBuilder) WithConfigVersion(version time.Time) *EntryBuilder {
	b.entry.Clustering.ConfigVersion = version
	return b
}

// WithRulesVersion specifies the rules version for the entry.
func (b *EntryBuilder) WithRulesVersion(version time.Time) *EntryBuilder {
	b.entry.Clustering.RulesVersion = version
	return b
}

// Build returns the built entry.
func (b *EntryBuilder) Build() *Entry {
	return b.entry
}

// CreateEntriesForTesting creates the given entries, for testing.
func CreateEntriesForTesting(ctx context.Context, entries []*Entry) (commitTimestamp time.Time, err error) {
	return span.ReadWriteTransaction(ctx, func(ctx context.Context) error {
		for _, e := range entries {
			if err := Create(ctx, e); err != nil {
				return err
			}
		}
		return nil
	})
}

// ReadAllForTesting reads all state entries in the given project
// (up to 1 million records) for testing.
func ReadAllForTesting(ctx context.Context, project string) ([]*Entry, error) {
	return readWhere(span.Single(ctx), project, "TRUE", nil, 1000*1000)
}
