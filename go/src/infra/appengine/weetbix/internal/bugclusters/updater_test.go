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
	"infra/appengine/weetbix/internal/config"
	"infra/appengine/weetbix/internal/testutil"

	"cloud.google.com/go/bigquery"
	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/luci/server/span"
	"google.golang.org/protobuf/proto"
)

func TestRun(t *testing.T) {
	ctx := testutil.SpannerTestContext(t)
	Convey("Run bug updates", t, func() {
		setBugClusters(ctx, nil)

		f := &monorail.FakeIssuesStore{
			NextID:            100,
			PriorityFieldName: "projects/chromium/fieldDefs/11",
		}
		user := monorail.AutomationUsers[0]
		mc, err := monorail.NewClient(monorail.UseFakeIssuesClient(ctx, f, user), "myhost")
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

		mgrs := make(map[string]BugManager)
		mgrs[monorail.ManagerName] = monorail.NewBugManager(mc, monorail.ChromiumTestConfig())

		thres := map[string]*config.ImpactThreshold{
			"chromium": {
				UnexpectedFailures_1D: proto.Int64(10),
				UnexpectedFailures_3D: proto.Int64(30),
				UnexpectedFailures_7D: proto.Int64(70),
			},
		}

		Convey("With no impactful clusters", func() {
			bu := NewBugUpdater(mgrs, cc, thres)
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
				bu := NewBugUpdater(mgrs, cc, thres)
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
				So(f.Issues[0].Issue.Name, ShouldEqual, "projects/chromium/issues/100")
				So(f.Issues[0].Issue.Summary, ShouldContainSubstring, "Test failure reason.")
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
			expectBugClusters := func(count int) {
				bugClusters, err := ReadActive(span.Single(ctx))
				So(err, ShouldBeNil)
				So(len(bugClusters), ShouldEqual, count)
				So(len(f.Issues), ShouldEqual, count)
			}
			clusters[1].UnexpectedFailures1d = 200
			clusters[2].UnexpectedFailures3d = 30
			clusters[3].UnexpectedFailures7d = 70

			bu := NewBugUpdater(mgrs, cc, thres)
			// Limit to one bug filed each time, so that
			// we test change throttling.
			bu.MaxBugsFiledPerRun = 1

			err = bu.Run(ctx)
			So(err, ShouldBeNil)
			expectBugClusters(1)

			err = bu.Run(ctx)
			So(err, ShouldBeNil)
			expectBugClusters(2)

			err = bu.Run(ctx)
			So(err, ShouldBeNil)

			expectFinalBugClusters := func() {
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
			}
			expectFinalBugClusters()

			// Further updates do nothing.
			originalIssues := monorail.CopyIssuesStore(f)
			err = bu.Run(ctx)
			So(err, ShouldBeNil)
			So(f, monorail.ShouldResembleIssuesStore, originalIssues)
			expectFinalBugClusters()

			Convey("Changing cluster priority updates issue priority", func() {
				issue := f.Issues[2].Issue
				So(issue.Name, ShouldEqual, "projects/chromium/issues/102")
				So(monorail.ChromiumTestIssuePriority(issue), ShouldEqual, "3")

				clusters[3].UnexpectedFailures1d = 10000
				err = bu.Run(ctx)
				So(err, ShouldBeNil)

				So(len(f.Issues), ShouldEqual, 3)
				issue = f.Issues[2].Issue
				So(issue.Name, ShouldEqual, "projects/chromium/issues/102")
				So(monorail.ChromiumTestIssuePriority(issue), ShouldEqual, "0")

				expectFinalBugClusters()
			})
			Convey("Deleting cluster closes issue", func() {
				issue := f.Issues[0].Issue
				So(issue.Name, ShouldEqual, "projects/chromium/issues/100")
				So(monorail.ChromiumTestIssuePriority(issue), ShouldEqual, "2")

				// Drop the cluster at index 1.
				cc.clusters = []*clustering.Cluster{cc.clusters[0], cc.clusters[2], cc.clusters[3]}
				err = bu.Run(ctx)
				So(err, ShouldBeNil)

				So(len(f.Issues), ShouldEqual, 3)
				issue = f.Issues[0].Issue
				So(issue.Name, ShouldEqual, "projects/chromium/issues/100")
				So(issue.Status.Status, ShouldEqual, monorail.VerifiedStatus)
			})
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
			(opts.Thresholds.UnexpectedFailures_1D != nil && int64(c.UnexpectedFailures1d) >= *opts.Thresholds.UnexpectedFailures_1D) ||
			(opts.Thresholds.UnexpectedFailures_3D != nil && int64(c.UnexpectedFailures3d) >= *opts.Thresholds.UnexpectedFailures_3D) ||
			(opts.Thresholds.UnexpectedFailures_7D != nil && int64(c.UnexpectedFailures7d) >= *opts.Thresholds.UnexpectedFailures_7D)
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
