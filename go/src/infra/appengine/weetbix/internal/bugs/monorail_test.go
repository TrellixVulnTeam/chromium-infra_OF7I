// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package bugs

import (
	"context"
	"fmt"
	"testing"

	mpb "infra/monorailv2/api/v3/api_proto"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func newIssue(uniqifier int) *mpb.Issue {
	return &mpb.Issue{
		Name:     fmt.Sprintf("projects/monorailproject/issues/%v", uniqifier),
		Summary:  fmt.Sprintf("This is the summary of bug %v.", uniqifier),
		State:    mpb.IssueContentState_ACTIVE,
		Reporter: "meiring@google.com",
	}
}

func TestGetIssue(t *testing.T) {
	t.Parallel()

	issue1 := newIssue(1)
	issue2 := newIssue(2)
	issue3 := newIssue(3)
	f := &FakeIssuesClient{
		Issues: []*mpb.Issue{issue1, issue2, issue3},
		NextID: 4,
	}

	ctx := UseFakeIssuesClient(context.Background(), f)
	Convey("Get issue", t, func() {
		bc, err := NewMonorailClient(ctx, "monorailhost")
		So(err, ShouldBeNil)
		result, err := bc.GetIssue(ctx, "projects/monorailproject/issues/1")
		So(err, ShouldBeNil)
		So(result, ShouldResembleProto, issue1)
	})
	Convey("Get issues", t, func() {
		bc, err := NewMonorailClient(ctx, "monorailhost")
		So(err, ShouldBeNil)
		names := []string{
			"projects/monorailproject/issues/1",
			"projects/monorailproject/issues/2",
			"projects/monorailproject/issues/1",
			"projects/monorailproject/issues/2",
			"projects/monorailproject/issues/3",
		}
		result, err := bc.GetIssues(ctx, names)
		So(err, ShouldBeNil)
		So(result, ShouldResembleProto, []*mpb.Issue{issue1, issue2, issue1, issue2, issue3})
	})
	Convey("Make issue", t, func() {
		issue := newIssue(4)
		issue.Name = ""
		req := &mpb.MakeIssueRequest{
			Parent:      "projects/monorailproject",
			Issue:       issue,
			Description: "Description",
			NotifyType:  mpb.NotifyType_NO_NOTIFICATION,
		}
		issue, err := f.MakeIssue(ctx, req)
		So(err, ShouldBeNil)
		issue.Name = "project/monorailproject/issues/4"
		So(issue, ShouldResembleProto, issue)
	})
}
