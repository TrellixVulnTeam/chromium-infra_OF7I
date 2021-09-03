// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package bugclusters

import (
	"testing"

	"cloud.google.com/go/spanner"
	"go.chromium.org/luci/server/span"

	"infra/appengine/weetbix/internal/testutil"

	. "github.com/smartystreets/goconvey/convey"
)

func TestRead(t *testing.T) {
	Convey(`Read`, t, func() {
		ctx := testutil.SpannerTestContext(t)

		testutil.MustApply(ctx,
			spanner.Delete("BugClusters", spanner.AllKeys()))

		Convey(`Empty`, func() {
			clusters, err := ReadActive(span.Single(ctx))
			So(err, ShouldBeNil)
			So(clusters, ShouldBeNil)
		})

		Convey(`Multiple`, func() {
			// Insert some BugClusters.
			testutil.MustApply(ctx,
				spanner.InsertOrUpdateMap("BugClusters", map[string]interface{}{
					"Project":             "project1",
					"Bug":                 "monorail/project/1",
					"AssociatedClusterId": "some-cluster-id1",
					"IsActive":            true,
				}),
				spanner.InsertOrUpdateMap("BugClusters", map[string]interface{}{
					"Project":             "project2",
					"Bug":                 "monorail/project/2",
					"AssociatedClusterId": "some-cluster-id2",
					"IsActive":            false,
				}),
				spanner.InsertOrUpdateMap("BugClusters", map[string]interface{}{
					"Project":             "project3",
					"Bug":                 "monorail/project/3",
					"AssociatedClusterId": "some-cluster-id3",
					"IsActive":            true,
				}),
			)

			clusters, err := ReadActive(span.Single(ctx))
			So(err, ShouldBeNil)
			So(clusters, ShouldResemble, []*BugCluster{
				{
					Project:             "project1",
					Bug:                 "monorail/project/1",
					AssociatedClusterID: "some-cluster-id1",
				},
				{
					Project:             "project3",
					Bug:                 "monorail/project/3",
					AssociatedClusterID: "some-cluster-id3",
				},
			})
		})
	})
}
