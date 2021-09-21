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
)

func TestClient(t *testing.T) {
	t.Parallel()

	issue1 := NewIssueData(1)
	issue2 := NewIssueData(2)
	issue3 := NewIssueData(3)
	f := &FakeIssuesStore{
		Issues: []*IssueData{issue1, issue2, issue3},
		NextID: 4,
	}
	ctx := UseFakeIssuesClient(context.Background(), f, "user@chromium.org")

	Convey("Get issue", t, func() {
		c, err := NewClient(ctx, "monorailhost")
		So(err, ShouldBeNil)
		result, err := c.GetIssue(ctx, "projects/monorailproject/issues/1")
		So(err, ShouldBeNil)
		So(result, ShouldResembleProto, issue1.Issue)
	})
	Convey("Get issues", t, func() {
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
	Convey("Make issue", t, func() {
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
	})
}
