// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package monorail

import (
	"infra/appengine/weetbix/internal/clustering"
	mpb "infra/monorailv2/api/v3/api_proto"
	"testing"

	"cloud.google.com/go/bigquery"
	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func TestPrepareNew(t *testing.T) {
	t.Parallel()

	c := &clustering.Cluster{
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
	ig := NewIssueGenerator("reporter@chromium.org")

	Convey("PrepareNew", t, func() {
		Convey("With reason-based failure cluster", func() {
			reason := `Expected equality of these values:
				"Expected_Value"
				my_expr.evaluate(123)
				    Which is: "Unexpected_Value"`
			c.ClusterID = "ClusterIDShouldNotAppearInOutput"
			c.ExampleFailureReason = bigquery.NullString{StringVal: reason, Valid: true}

			result := ig.PrepareNew(c)
			So(result.Parent, ShouldEqual, "projects/chromium")
			So(result.Issue, ShouldResembleProto, &mpb.Issue{
				Summary:  "Tests are failing: Expected equality of these values: \"Expected_Value\" my_expr.evaluate(123) Which is: \"Unexpected_Value\"",
				Reporter: "reporter@chromium.org",
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
						Value: "Pri-0",
					},
				},
				Labels: []*mpb.Issue_LabelValue{{
					Label: "Restrict-View-Google",
				}, {
					Label: "Weetbix-Managed",
				}},
			})
			So(result.Description, ShouldContainSubstring, reason)
			So(result.Description, ShouldNotContainSubstring, "ClusterIDShouldNotAppearInOutput")
			So(result.NotifyType, ShouldEqual, mpb.NotifyType_NO_NOTIFICATION)
		})
		Convey("With test name failure cluster", func() {
			c.ClusterID = "ninja://:blink_web_tests/media/my-suite/my-test.html"
			c.ExampleFailureReason = bigquery.NullString{Valid: false}

			result := ig.PrepareNew(c)
			So(result.Parent, ShouldEqual, "projects/chromium")
			So(result.Issue, ShouldResembleProto, &mpb.Issue{
				Summary:  "Tests are failing: ninja://:blink_web_tests/media/my-suite/my-test.html",
				Reporter: "reporter@chromium.org",
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
						Value: "Pri-0",
					},
				},
				Labels: []*mpb.Issue_LabelValue{{
					Label: "Restrict-View-Google",
				}, {
					Label: "Weetbix-Managed",
				}},
			})
			So(result.Description, ShouldContainSubstring, "ninja://:blink_web_tests/media/my-suite/my-test.html")
			So(result.NotifyType, ShouldEqual, mpb.NotifyType_NO_NOTIFICATION)
		})
	})
}
