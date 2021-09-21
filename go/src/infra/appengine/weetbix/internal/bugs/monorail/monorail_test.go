// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package monorail

import (
	"context"
	"testing"

	mpb "infra/monorailv2/api/v3/api_proto"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
	"google.golang.org/genproto/protobuf/field_mask"
)

func TestClient(t *testing.T) {
	t.Parallel()

	Convey("With Existing Issue Data", t, func() {
		issue1 := NewIssueData(1)
		issue2 := NewIssueData(2)
		issue3 := NewIssueData(3)
		f := &FakeIssuesStore{
			Issues: []*IssueData{issue1, issue2, issue3},
			NextID: 4,
		}
		ctx := UseFakeIssuesClient(context.Background(), f, "user@chromium.org")

		Convey("Get issue", func() {
			c, err := NewClient(ctx, "monorailhost")
			So(err, ShouldBeNil)
			result, err := c.GetIssue(ctx, "projects/monorailproject/issues/1")
			So(err, ShouldBeNil)
			So(result, ShouldResembleProto, issue1.Issue)
		})
		Convey("Get issues", func() {
			c, err := NewClient(ctx, "monorailhost")
			So(err, ShouldBeNil)
			names := []string{
				"projects/monorailproject/issues/1",
				"projects/monorailproject/issues/2",
				"projects/monorailproject/issues/1",
				"projects/monorailproject/issues/2",
				"projects/monorailproject/issues/3",
			}
			result, err := c.GetIssues(ctx, names)
			So(err, ShouldBeNil)
			So(result, ShouldResembleProto, []*mpb.Issue{issue1.Issue, issue2.Issue, issue1.Issue, issue2.Issue, issue3.Issue})
		})
		Convey("Make issue", func() {
			issue := NewIssue(4)
			issue.Name = ""
			req := &mpb.MakeIssueRequest{
				Parent:      "projects/monorailproject",
				Issue:       issue,
				Description: "Description",
				NotifyType:  mpb.NotifyType_NO_NOTIFICATION,
			}

			c, err := NewClient(ctx, "monorailhost")
			So(err, ShouldBeNil)
			result, err := c.MakeIssue(ctx, req)
			So(err, ShouldBeNil)
			So(result, ShouldResembleProto, NewIssue(4))

			comments, err := c.ListComments(ctx, result.Name)
			So(err, ShouldBeNil)
			So(len(comments), ShouldEqual, 1)
			So(comments[0].Content, ShouldEqual, "Description")
		})
		Convey("List comments", func() {
			Convey("Single comment", func() {
				c, err := NewClient(ctx, "monorailhost")
				So(err, ShouldBeNil)
				comments, err := c.ListComments(ctx, "projects/monorailproject/issues/1")
				So(err, ShouldBeNil)
				So(len(comments), ShouldEqual, 1)
				So(comments, ShouldResembleProto, issue1.Comments)
			})
			Convey("Many comments", func() {
				issue := NewIssueData(4)
				for i := 2; i <= 3*maxCommentPageSize; i++ {
					issue.Comments = append(issue.Comments, NewComment(issue.Issue.Name, i))
				}
				f.Issues = append(f.Issues, issue)

				c, err := NewClient(ctx, "monorailhost")
				So(err, ShouldBeNil)
				comments, err := c.ListComments(ctx, issue.Issue.Name)
				So(err, ShouldBeNil)
				So(comments, ShouldResembleProto, issue.Comments)
			})
		})
		Convey("Modify issue", func() {
			issue1.Issue.Labels = []*mpb.Issue_LabelValue{
				{Label: "Test-Label1"},
			}

			c, err := NewClient(ctx, "monorailhost")
			So(err, ShouldBeNil)

			req := &mpb.ModifyIssuesRequest{
				Deltas: []*mpb.IssueDelta{
					{
						Issue: &mpb.Issue{
							Name: issue1.Issue.Name,
							Labels: []*mpb.Issue_LabelValue{
								{
									Label: "Test-Label2",
								},
							},
						},
						UpdateMask: &field_mask.FieldMask{
							Paths: []string{"labels"},
						},
					},
				},
				CommentContent: "Changing priority.",
			}
			err = c.ModifyIssues(ctx, req)
			So(err, ShouldBeNil)

			expectedData := NewIssueData(1)
			expectedData.Issue.Labels = []*mpb.Issue_LabelValue{
				{Label: "Test-Label1"},
				{Label: "Test-Label2"},
			}

			read, err := c.GetIssue(ctx, issue1.Issue.Name)
			So(err, ShouldBeNil)
			So(read, ShouldResembleProto, expectedData.Issue)

			comments, err := c.ListComments(ctx, issue1.Issue.Name)
			So(err, ShouldBeNil)
			So(len(comments), ShouldEqual, 2)
			So(comments[0], ShouldResembleProto, expectedData.Comments[0])
			So(comments[1].Content, ShouldEqual, "Changing priority.")
		})
	})
}
