// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package bugclusters

import (
	"context"
	"fmt"
	"testing"

	"infra/appengine/weetbix/internal/bugs/monorail"
	"infra/appengine/weetbix/internal/clustering"
	"infra/appengine/weetbix/internal/testutil"

	"cloud.google.com/go/bigquery"
	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/luci/server/span"
)

func TestRun(t *testing.T) {
	ctx := testutil.SpannerTestContext(t)
	Convey("Run bug updates", t, func() {
		setBugClusters(ctx, nil)

		f := &monorail.FakeIssuesClient{
			NextID: 100,
		}
		mc, err := monorail.NewClient(monorail.UseFakeIssuesClient(ctx, f), "myhost")
		So(err, ShouldBeNil)

		clusters := []*clustering.Cluster{
			makeCluster(0),
			makeCluster(1),
			makeCluster(2),
			makeCluster(3),
		}
		cc := &fakeClusterClient{
			clusters: clusters,
		}

		ig := monorail.NewIssueGenerator("reporter@google.com")
		thres := clustering.ImpactThresholds{
			UnexpectedFailures1d: 10,
			UnexpectedFailures3d: 30,
			UnexpectedFailures7d: 70,
		}

		Convey("With no impactful clusters", func() {
			bu := NewBugUpdater(mc, cc, ig, thres)
			err = bu.Run(ctx)
			So(err, ShouldBeNil)

			// No bug clusters.
			bugClusters, err := ReadActive(span.Single(ctx))
			So(err, ShouldBeNil)
			So(bugClusters, ShouldResemble, []*BugCluster{})

			// No monorail issues.
			So(f.Issues, ShouldBeNil)
		})
		Convey("With a cluster above impact thresold", func() {
			clusters[1].ExampleFailureReason = bigquery.NullString{StringVal: "Test failure reason.", Valid: true}

			test := func() {
				bu := NewBugUpdater(mc, cc, ig, thres)
				err = bu.Run(ctx)
				So(err, ShouldBeNil)

				bugClusters, err := ReadActive(span.Single(ctx))
				So(err, ShouldBeNil)
				So(bugClusters, ShouldResemble, []*BugCluster{
					{
						Project:             "chromium",
						Bug:                 "monorail/chromium/100",
						AssociatedClusterID: clusterID(1),
						IsActive:            true,
					},
				})
				So(len(f.Issues), ShouldEqual, 1)
				So(f.Issues[0].Name, ShouldEqual, "projects/chromium/issues/100")
				So(f.Issues[0].Reporter, ShouldEqual, "reporter@google.com")
				So(f.Issues[0].Summary, ShouldContainSubstring, "Test failure reason.")
			}
			Convey("1d unexpected failures", func() {
				clusters[1].UnexpectedFailures1d = 10
				test()
			})
			Convey("3d unexpected failures", func() {
				clusters[1].UnexpectedFailures3d = 30
				test()
			})
			Convey("7d unexpected failures", func() {
				clusters[1].UnexpectedFailures7d = 70
				test()
			})
		})
		Convey("With multiple clusters above impact thresold", func() {
			expectBugs := func(count int) {
				bugClusters, err := ReadActive(span.Single(ctx))
				So(err, ShouldBeNil)
				So(len(bugClusters), ShouldEqual, count)
				So(len(f.Issues), ShouldEqual, count)
			}
			clusters[1].UnexpectedFailures1d = 10
			clusters[2].UnexpectedFailures3d = 30
			clusters[3].UnexpectedFailures7d = 70

			bu := NewBugUpdater(mc, cc, ig, thres)
			// Limit to one bug filed each time, so that
			// we test change throttling.
			bu.MaxBugsFiledPerRun = 1

			err = bu.Run(ctx)
			So(err, ShouldBeNil)
			expectBugs(1)

			err = bu.Run(ctx)
			So(err, ShouldBeNil)
			expectBugs(2)

			err = bu.Run(ctx)
			So(err, ShouldBeNil)

			// Check final set of bugs is as expected.
			bugClusters, err := ReadActive(span.Single(ctx))
			So(err, ShouldBeNil)
			So(bugClusters, ShouldResemble, []*BugCluster{
				{
					Project:             "chromium",
					Bug:                 "monorail/chromium/100",
					AssociatedClusterID: clusterID(1),
					IsActive:            true,
				},
				{
					Project:             "chromium",
					Bug:                 "monorail/chromium/101",
					AssociatedClusterID: clusterID(2),
					IsActive:            true,
				},
				{
					Project:             "chromium",
					Bug:                 "monorail/chromium/102",
					AssociatedClusterID: clusterID(3),
					IsActive:            true,
				},
			})
			So(len(f.Issues), ShouldEqual, 3)
		})
	})
}

func makeCluster(uniqifier int) *clustering.Cluster {
	return &clustering.Cluster{
		Project:                "chromium",
		ClusterID:              clusterID(uniqifier),
		UnexpectedFailures1d:   9,
		UnexpectedFailures3d:   29,
		UnexpectedFailures7d:   69,
		UnexoneratedFailures1d: 10000,
		UnexoneratedFailures3d: 10000,
		UnexoneratedFailures7d: 10000,
		AffectedRuns1d:         10000,
		AffectedRuns3d:         10000,
		AffectedRuns7d:         10000,
	}
}

func clusterID(uniqifier int) string {
	return fmt.Sprintf("test-cluster-id-%v", uniqifier)
}

type fakeClusterClient struct {
	clusters []*clustering.Cluster
}

func (f *fakeClusterClient) ReadImpactfulClusters(ctx context.Context, opts clustering.ImpactfulClusterReadOptions) ([]*clustering.Cluster, error) {
	var results []*clustering.Cluster
	for _, c := range f.clusters {
		include := containsValue(opts.AlwaysIncludeClusterIDs, c.ClusterID) ||
			c.UnexpectedFailures1d >= opts.Thresholds.UnexpectedFailures1d ||
			c.UnexpectedFailures3d >= opts.Thresholds.UnexpectedFailures3d ||
			c.UnexpectedFailures7d >= opts.Thresholds.UnexpectedFailures7d
		if include {
			results = append(results, c)
		}
	}
	return results, nil
}

func containsValue(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}
