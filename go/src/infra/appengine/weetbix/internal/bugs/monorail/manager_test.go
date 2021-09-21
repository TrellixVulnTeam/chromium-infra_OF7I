// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package monorail

import (
	"context"
	"infra/appengine/weetbix/internal/clustering"
	mpb "infra/monorailv2/api/v3/api_proto"
	"testing"

	"cloud.google.com/go/bigquery"
	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func NewCluster() *clustering.Cluster {
	return &clustering.Cluster{
		ClusterID:              "ClusterID",
		UnexpectedFailures1d:   1300,
		UnexpectedFailures3d:   3300,
		UnexpectedFailures7d:   7300,
		UnexoneratedFailures1d: 120,
		UnexoneratedFailures3d: 320,
		UnexoneratedFailures7d: 720,
		AffectedRuns1d:         11,
		AffectedRuns3d:         31,
		AffectedRuns7d:         71,
		ExampleFailureReason:   bigquery.NullString{StringVal: "Some failure reason.", Valid: true},
	}
}

func TestManager(t *testing.T) {
	t.Parallel()

	Convey("Create", t, func() {
		ctx := context.Background()
		f := &FakeIssuesStore{
			NextID: 100,
		}
		user := AutomationUsers[0]
		cl, err := NewClient(UseFakeIssuesClient(ctx, f, user), "myhost")
		So(err, ShouldBeNil)
		bm := NewBugManager(cl)

		c := NewCluster()
		Convey("With reason-based failure cluster", func() {
			reason := `Expected equality of these values:
				"Expected_Value"
				my_expr.evaluate(123)
					Which is: "Unexpected_Value"`
			c.ClusterID = "ClusterIDShouldNotAppearInOutput"
			c.ExampleFailureReason = bigquery.NullString{StringVal: reason, Valid: true}

			bug, err := bm.Create(ctx, c)
			So(err, ShouldBeNil)
			So(bug, ShouldEqual, "chromium/100")
			So(len(f.Issues), ShouldEqual, 1)
			issue := f.Issues[0]

			So(issue.Issue, ShouldResembleProto, &mpb.Issue{
				Name:     "projects/chromium/issues/100",
				Summary:  "Tests are failing: Expected equality of these values: \"Expected_Value\" my_expr.evaluate(123) Which is: \"Unexpected_Value\"",
				Reporter: AutomationUsers[0],
				State:    mpb.IssueContentState_ACTIVE,
				Status:   &mpb.Issue_StatusValue{Status: "Untriaged"},
				FieldValues: []*mpb.FieldValue{
					{
						// Type field.
						Field: "projects/chromium/fieldDefs/10",
						Value: "Bug",
					},
					{
						// Priority field.
						Field: "projects/chromium/fieldDefs/11",
						Value: "0",
					},
				},
				Labels: []*mpb.Issue_LabelValue{{
					Label: "Restrict-View-Google",
				}, {
					Label: "Weetbix-Managed",
				}},
			})
			So(len(issue.Comments), ShouldEqual, 1)
			So(issue.Comments[0].Content, ShouldContainSubstring, reason)
			So(issue.Comments[0].Content, ShouldNotContainSubstring, "ClusterIDShouldNotAppearInOutput")
		})
		Convey("With test name failure cluster", func() {
			c.ClusterID = "ninja://:blink_web_tests/media/my-suite/my-test.html"
			c.ExampleFailureReason = bigquery.NullString{Valid: false}

			bug, err := bm.Create(ctx, c)
			So(err, ShouldBeNil)
			So(bug, ShouldEqual, "chromium/100")
			So(len(f.Issues), ShouldEqual, 1)
			issue := f.Issues[0]

			So(issue.Issue, ShouldResembleProto, &mpb.Issue{
				Name:     "projects/chromium/issues/100",
				Summary:  "Tests are failing: ninja://:blink_web_tests/media/my-suite/my-test.html",
				Reporter: AutomationUsers[0],
				State:    mpb.IssueContentState_ACTIVE,
				Status:   &mpb.Issue_StatusValue{Status: "Untriaged"},
				FieldValues: []*mpb.FieldValue{
					{
						// Type field.
						Field: "projects/chromium/fieldDefs/10",
						Value: "Bug",
					},
					{
						// Priority field.
						Field: "projects/chromium/fieldDefs/11",
						Value: "0",
					},
				},
				Labels: []*mpb.Issue_LabelValue{{
					Label: "Restrict-View-Google",
				}, {
					Label: "Weetbix-Managed",
				}},
			})
			So(len(issue.Comments), ShouldEqual, 1)
			So(issue.Comments[0].Content, ShouldContainSubstring, "ninja://:blink_web_tests/media/my-suite/my-test.html")
		})
	})
}
