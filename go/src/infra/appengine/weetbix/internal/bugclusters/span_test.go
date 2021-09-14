// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package bugclusters

import (
	"context"
	"fmt"
	"testing"

	"cloud.google.com/go/spanner"
	"go.chromium.org/luci/server/span"

	"infra/appengine/weetbix/internal/testutil"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func TestRead(t *testing.T) {
	ctx := testutil.SpannerTestContext(t)
	Convey(`Read`, t, func() {
		Convey(`Empty`, func() {
			setBugClusters(ctx, nil)

			clusters, err := ReadActive(span.Single(ctx))
			So(err, ShouldBeNil)
			So(clusters, ShouldResemble, []*BugCluster{})
		})
		Convey(`Multiple`, func() {
			clustersToCreate := []*BugCluster{
				newBugCluster(0),
				newBugCluster(1),
				newBugCluster(2),
			}
			clustersToCreate[1].IsActive = false
			setBugClusters(ctx, clustersToCreate)

			clusters, err := ReadActive(span.Single(ctx))
			So(err, ShouldBeNil)
			So(clusters, ShouldResemble, []*BugCluster{
				newBugCluster(0),
				newBugCluster(2),
			})
		})
	})
	Convey(`Create`, t, func() {
		testCreate := func(bc *BugCluster) error {
			_, err := span.ReadWriteTransaction(ctx, func(ctx context.Context) error {
				return Create(ctx, bc)
			})
			return err
		}
		Convey(`Valid`, func() {
			bc := newBugCluster(103)
			err := testCreate(bc)
			So(err, ShouldBeNil)
			// Create followed by read already tested as part of Read tests.
		})
		Convey(`With missing Project`, func() {
			bc := newBugCluster(100)
			bc.Project = ""
			err := testCreate(bc)
			So(err, ShouldErrLike, "project must be specified")
		})
		Convey(`With missing Bug`, func() {
			bc := newBugCluster(101)
			bc.Bug = ""
			err := testCreate(bc)
			So(err, ShouldErrLike, "bug must be specified")
		})
		Convey(`With missing Associated Cluster`, func() {
			bc := newBugCluster(102)
			bc.AssociatedClusterID = ""
			err := testCreate(bc)
			So(err, ShouldErrLike, "associated cluster must be specified")
		})
	})
}

func newBugCluster(uniqifier int) *BugCluster {
	return &BugCluster{
		Project:             fmt.Sprintf("project%v", uniqifier),
		Bug:                 fmt.Sprintf("monorail/project/%v", uniqifier),
		AssociatedClusterID: fmt.Sprintf("some-cluster-id%v", uniqifier),
		IsActive:            true,
	}
}

// setBugClusters replaces the set of stored bug clusters to match the given set.
func setBugClusters(ctx context.Context, bcs []*BugCluster) {
	testutil.MustApply(ctx,
		spanner.Delete("BugClusters", spanner.AllKeys()))
	// Insert some BugClusters.
	_, err := span.ReadWriteTransaction(ctx, func(ctx context.Context) error {
		for _, bc := range bcs {
			if err := Create(ctx, bc); err != nil {
				return err
			}
		}
		return nil
	})
	So(err, ShouldBeNil)
}
