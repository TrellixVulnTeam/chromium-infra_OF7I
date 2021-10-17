// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package state

import (
	"context"
	"fmt"
	"testing"
	"time"

	"go.chromium.org/luci/server/span"

	"infra/appengine/weetbix/internal/clustering"
	"infra/appengine/weetbix/internal/testutil"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func TestSpanner(t *testing.T) {
	Convey(`Create`, t, func() {
		ctx := testutil.SpannerTestContext(t)
		testCreate := func(e *Entry) error {
			_, err := span.ReadWriteTransaction(ctx, func(ctx context.Context) error {
				return Create(ctx, e)
			})
			return err
		}
		e := newEntry(100)
		Convey(`Valid`, func() {
			err := testCreate(e)
			So(err, ShouldBeNil)

			txn := span.Single(ctx)
			actual, err := Read(txn, e.Project, e.ChunkID)
			So(err, ShouldBeNil)

			// Check the LastUpdated time is set, but ignore it for
			// further comparisons.
			So(actual.LastUpdated, ShouldNotBeZeroValue)
			actual.LastUpdated = time.Time{}

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
			Convey(`Rule Version missing`, func() {
				var t time.Time
				e.RuleVersion = t
				err := testCreate(e)
				So(err, ShouldErrLike, "rule version must be specified")
			})
			Convey(`Algorithms Version missing`, func() {
				e.AlgorithmsVersion = 0
				err := testCreate(e)
				So(err, ShouldErrLike, "algorithms version must be specified")
			})
			Convey(`Clusters missing`, func() {
				e.Clusters = nil
				err := testCreate(e)
				So(err, ShouldErrLike, "there must be clustered test results in the chunk")
			})
			Convey(`Clusters invalid`, func() {
				Convey(`Algorithm missing`, func() {
					e.Clusters[1][1].Algorithm = ""
					err := testCreate(e)
					So(err, ShouldErrLike, `test result 1: cluster 1: algorithm name ("") is not valid`)
				})
				Convey("Algorithm invalid", func() {
					e.Clusters[1][1].Algorithm = "!!!"
					err := testCreate(e)
					So(err, ShouldErrLike, `test result 1: cluster 1: algorithm name ("!!!") is not valid`)
				})
				Convey("ID missing", func() {
					e.Clusters[1][1].ID = []byte{}
					err := testCreate(e)
					So(err, ShouldErrLike, `test result 1: cluster 1: cluster ID must be specified`)
				})
			})
		})
	})
}

func newEntry(uniqifier int) *Entry {
	return &Entry{
		Project:           fmt.Sprintf("project-%v", uniqifier),
		ChunkID:           fmt.Sprintf("c%v", uniqifier),
		PartitionTime:     time.Date(2030, 1, 1, 1, 1, 1, uniqifier, time.UTC),
		ObjectID:          "abcdef1234567890abcdef1234567890",
		AlgorithmsVersion: int64(uniqifier),
		RuleVersion:       time.Date(2025, 1, 1, 1, 1, 1, uniqifier, time.UTC),
		Clusters: [][]*clustering.ClusterID{
			{
				{
					Algorithm: fmt.Sprintf("alg-%v", uniqifier),
					ID:        []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
				},
			},
			{
				{
					Algorithm: fmt.Sprintf("alg-%v", uniqifier),
					ID:        []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
				},
				{
					Algorithm: fmt.Sprintf("alg-%v", uniqifier),
					ID:        []byte{2},
				},
			},
		},
	}
}
