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
	"sort"
	"strings"
	"testing"
	"time"

	"go.chromium.org/luci/server/span"

	"infra/appengine/weetbix/internal/clustering"
	"infra/appengine/weetbix/internal/testutil"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func TestSpanner(t *testing.T) {
	Convey(`With Spanner Test Database`, t, func() {
		ctx := testutil.SpannerTestContext(t)
		Convey(`Create`, func() {
			testCreate := func(e *Entry) error {
				_, err := span.ReadWriteTransaction(ctx, func(ctx context.Context) error {
					return Create(ctx, e)
				})
				return err
			}
			e := newEntry(100).build()
			Convey(`Valid`, func() {
				err := testCreate(e)
				So(err, ShouldBeNil)

				txn := span.Single(ctx)
				actual, err := Read(txn, e.Project, e.ChunkID)
				So(err, ShouldBeNil)

				// Check the LastUpdated time is set, but ignore it for
				// further comparisons.
				clearLastUpdatedTimestamps(actual)

				So(err, ShouldBeNil)
				So(actual, ShouldResemble, e)
			})
			Convey(`Invalid`, func() {
				Convey(`Project missing`, func() {
					e.Project = ""
					err := testCreate(e)
					So(err, ShouldErrLike, `project "" is not valid`)
				})
				Convey(`Chunk ID missing`, func() {
					e.ChunkID = ""
					err := testCreate(e)
					So(err, ShouldErrLike, `chunk ID "" is not valid`)
				})
				Convey(`Partition Time missing`, func() {
					var t time.Time
					e.PartitionTime = t
					err := testCreate(e)
					So(err, ShouldErrLike, "partition time must be specified")
				})
				Convey(`Object ID missing`, func() {
					e.ObjectID = ""
					err := testCreate(e)
					So(err, ShouldErrLike, "object ID must be specified")
				})
				Convey(`Rules Version missing`, func() {
					var t time.Time
					e.Clustering.RulesVersion = t
					err := testCreate(e)
					So(err, ShouldErrLike, "rules version must be specified")
				})
				Convey(`Algorithms Version missing`, func() {
					e.Clustering.AlgorithmsVersion = 0
					err := testCreate(e)
					So(err, ShouldErrLike, "algorithms version must be specified")
				})
				Convey(`Clusters missing`, func() {
					e.Clustering.Clusters = nil
					err := testCreate(e)
					So(err, ShouldErrLike, "there must be clustered test results in the chunk")
				})
				Convey(`Algorithms invalid`, func() {
					Convey(`Empty algorithm`, func() {
						e.Clustering.Algorithms[""] = struct{}{}
						err := testCreate(e)
						So(err, ShouldErrLike, `algorithm "" is not valid`)
					})
					Convey("Algorithm invalid", func() {
						e.Clustering.Algorithms["!!!"] = struct{}{}
						err := testCreate(e)
						So(err, ShouldErrLike, `algorithm "!!!" is not valid`)
					})
				})
				Convey(`Clusters invalid`, func() {
					Convey(`Algorithm missing`, func() {
						e.Clustering.Clusters[1][1].Algorithm = ""
						err := testCreate(e)
						So(err, ShouldErrLike, `clusters: test result 1: cluster 1: cluster ID is not valid: algorithm not valid`)
					})
					Convey("Algorithm invalid", func() {
						e.Clustering.Clusters[1][1].Algorithm = "!!!"
						err := testCreate(e)
						So(err, ShouldErrLike, `clusters: test result 1: cluster 1: cluster ID is not valid: algorithm not valid`)
					})
					Convey("Algorithm not in algorithms set", func() {
						e.Clustering.Algorithms = map[string]struct{}{
							"alg-extra": {},
						}
						err := testCreate(e)
						So(err, ShouldErrLike, `a test result was clustered with an unregistered algorithm`)
					})
					Convey("ID missing", func() {
						e.Clustering.Clusters[1][1].ID = ""
						err := testCreate(e)
						So(err, ShouldErrLike, `clusters: test result 1: cluster 1: cluster ID is not valid: ID is empty`)
					})
				})
			})
		})
		Convey(`ReadNext`, func() {
			targetRulesVersion := time.Date(2024, 1, 1, 1, 1, 1, 0, time.UTC)
			targetAlgorithmsVersion := 10
			entries := []*Entry{
				// Should not be read.
				newEntry(0).withChunkIDPrefix("11").withAlgorithmVersion(10).withRulesVersion(targetRulesVersion).build(),

				// Should be read (rulesVersion < targetRulesVersion).
				newEntry(1).withChunkIDPrefix("11").withAlgorithmVersion(10).withRulesVersion(targetRulesVersion.Add(-1 * time.Hour)).build(), // Should be read.
				newEntry(3).withChunkIDPrefix("11").withRulesVersion(targetRulesVersion.Add(-1 * time.Hour)).build(),

				// Should be read (algorithmsVersion < targetAlgorithmsVersion).
				newEntry(2).withChunkIDPrefix("11").withAlgorithmVersion(9).withRulesVersion(targetRulesVersion).build(),
				newEntry(4).withChunkIDPrefix("11").withAlgorithmVersion(2).build(),

				// Should not be read (other project).
				newEntry(5).withChunkIDPrefix("11").withAlgorithmVersion(2).withProject("other").build(),

				// Check handling of EndChunkID as an inclusive upper-bound.
				newEntry(6).withChunkIDPrefix("11" + strings.Repeat("ff", 15)).withAlgorithmVersion(2).build(), // Should be read.
				newEntry(7).withChunkIDPrefix("12" + strings.Repeat("00", 15)).withAlgorithmVersion(2).build(), // Should not be read.
			}

			err := createEntries(ctx, entries)
			So(err, ShouldBeNil)

			expectedEntries := []*Entry{
				entries[1],
				entries[2],
				entries[3],
				entries[4],
				entries[6],
			}
			sort.Slice(expectedEntries, func(i, j int) bool {
				return expectedEntries[i].ChunkID < expectedEntries[j].ChunkID
			})

			readOpts := ReadNextOptions{
				StartChunkID:      "11" + strings.Repeat("00", 15),
				EndChunkID:        "11" + strings.Repeat("ff", 15),
				AlgorithmsVersion: int64(targetAlgorithmsVersion),
				RulesVersion:      targetRulesVersion,
			}
			// Reads first page.
			rows, err := ReadNextN(span.Single(ctx), testProject, readOpts, 3)
			So(err, ShouldBeNil)
			clearLastUpdatedTimestamps(rows...)
			So(rows, ShouldResemble, expectedEntries[0:3])

			// Read second page.
			readOpts.StartChunkID = rows[2].ChunkID
			rows, err = ReadNextN(span.Single(ctx), testProject, readOpts, 3)
			So(err, ShouldBeNil)
			clearLastUpdatedTimestamps(rows...)
			So(rows, ShouldResemble, expectedEntries[3:])

			// Read empty last page.
			readOpts.StartChunkID = rows[1].ChunkID
			rows, err = ReadNextN(span.Single(ctx), testProject, readOpts, 3)
			So(err, ShouldBeNil)
			So(rows, ShouldBeEmpty)
		})
		Convey(`EstimateChunks`, func() {
			Convey(`Less than 100 chunks`, func() {
				est, err := EstimateChunks(span.Single(ctx), testProject)
				So(err, ShouldBeNil)
				So(est, ShouldBeLessThan, 100)
			})
			Convey(`At least 100 chunks`, func() {
				var entries []*Entry
				for i := 0; i < 200; i++ {
					entries = append(entries, newEntry(i).build())
				}
				err := createEntries(ctx, entries)
				So(err, ShouldBeNil)

				count, err := EstimateChunks(span.Single(ctx), testProject)
				So(err, ShouldBeNil)
				So(count, ShouldBeGreaterThan, 190)
				So(count, ShouldBeLessThan, 210)
			})
		})
	})
	Convey(`estimateChunksFromID`, t, func() {
		// Extremely full table. This is the minimum that the 100th ID
		// could be (considering 0x63 = 99).
		count, err := estimateChunksFromID("00000000000000000000000000000063")
		So(err, ShouldBeNil)
		// The maximum estimate.
		So(count, ShouldEqual, 1000*1000*1000)

		// The 100th ID is right in the middle of the keyspace.
		count, err = estimateChunksFromID("7fffffffffffffffffffffffffffffff")
		So(err, ShouldBeNil)
		So(count, ShouldEqual, 200)

		// The 100th ID is right at the end of the keyspace.
		count, err = estimateChunksFromID("ffffffffffffffffffffffffffffffff")
		So(err, ShouldBeNil)
		So(count, ShouldEqual, 100)
	})
}

func createEntries(ctx context.Context, entries []*Entry) error {
	_, err := span.ReadWriteTransaction(ctx, func(ctx context.Context) error {
		for _, e := range entries {
			if err := Create(ctx, e); err != nil {
				return err
			}
		}
		return nil
	})
	return err
}

func clearLastUpdatedTimestamps(entries ...*Entry) {
	for _, e := range entries {
		// Check the LastUpdated time is set, but ignore it for
		// further comparisons.
		So(e.LastUpdated, ShouldNotBeZeroValue)
		e.LastUpdated = time.Time{}
	}
}

const testProject = "myproject"

type EntryBuilder struct {
	entry *Entry
}

func newEntry(uniqifier int) *EntryBuilder {
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
			RulesVersion:      time.Date(2025, 1, 1, 1, 1, 1, uniqifier, time.UTC),
			Algorithms: map[string]struct{}{
				fmt.Sprintf("alg-%v", uniqifier): {},
				"alg-extra":                      {},
			},
			Clusters: [][]*clustering.ClusterID{
				{
					{
						Algorithm: fmt.Sprintf("alg-%v", uniqifier),
						ID:        "00112233445566778899aabbccddeeff",
					},
				},
				{
					{
						Algorithm: fmt.Sprintf("alg-%v", uniqifier),
						ID:        "00112233445566778899aabbccddeeff",
					},
					{
						Algorithm: fmt.Sprintf("alg-%v", uniqifier),
						ID:        "22",
					},
				},
			},
		},
	}
	return &EntryBuilder{entry}
}

func (b *EntryBuilder) withChunkIDPrefix(prefix string) *EntryBuilder {
	b.entry.ChunkID = prefix + b.entry.ChunkID[len(prefix):]
	return b
}

func (b *EntryBuilder) withProject(project string) *EntryBuilder {
	b.entry.Project = project
	return b
}

func (b *EntryBuilder) withAlgorithmVersion(version int64) *EntryBuilder {
	b.entry.Clustering.AlgorithmsVersion = version
	return b
}

func (b *EntryBuilder) withRulesVersion(version time.Time) *EntryBuilder {
	b.entry.Clustering.RulesVersion = version
	return b
}

func (b *EntryBuilder) build() *Entry {
	return b.entry
}
