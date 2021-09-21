// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package monorail

import (
	"fmt"

	mpb "infra/monorailv2/api/v3/api_proto"

	"google.golang.org/protobuf/proto"
)

// IssueData is a representation of all data stored for an issue, used by
// FakeIssuesClient.
type IssueData struct {
	Issue    *mpb.Issue
	Comments []*mpb.Comment
}

// FakeIssuesSystem stores the state of bugs for a fake implementation of monorail.
type FakeIssuesStore struct {
	Issues []*IssueData
	NextID int
}

// NewIssueData creates new monorail issue data for testing.
func NewIssueData(uniqifier int) *IssueData {
	result := &IssueData{}
	result.Issue = NewIssue(uniqifier)
	result.Comments = []*mpb.Comment{
		NewComment(result.Issue.Name, 1),
	}
	return result
}

// NewIssue returns a new monorail issue proto for testing.
func NewIssue(uniqifier int) *mpb.Issue {
	return &mpb.Issue{
		Name:     fmt.Sprintf("projects/monorailproject/issues/%v", uniqifier),
		Summary:  fmt.Sprintf("This is the summary of bug %v.", uniqifier),
		State:    mpb.IssueContentState_ACTIVE,
		Reporter: "user@chromium.org",
		FieldValues: []*mpb.FieldValue{
			{
				Field: priorityFieldName,
				Value: "1",
			},
			{
				Field: typeFieldName,
				Value: "Bug",
			},
		},
		Labels: []*mpb.Issue_LabelValue{
			{
				Label: "Weetbix-Managed",
			},
		},
	}
}

// NewComment returns a new monorail comment proto for testing.
func NewComment(issueName string, number int) *mpb.Comment {
	return &mpb.Comment{
		Name:      fmt.Sprintf("%s/comment/%v", issueName, number),
		State:     mpb.IssueContentState_ACTIVE,
		Type:      mpb.Comment_DESCRIPTION,
		Content:   "Issue Description.",
		Commenter: "user@chromium.org",
	}
}

// CopyIssuesStore performs a deep copy of the given FakeIssuesStore.
func CopyIssuesStore(s *FakeIssuesStore) *FakeIssuesStore {
	var issues []*IssueData
	for _, iss := range s.Issues {
		issues = append(issues, CopyIssueData(iss))
	}
	return &FakeIssuesStore{
		Issues: issues,
		NextID: s.NextID,
	}
}

// CopyIssuesStore performs a deep copy of the given IssueData.
func CopyIssueData(d *IssueData) *IssueData {
	return &IssueData{
		Issue:    CopyIssue(d.Issue),
		Comments: CopyComments(d.Comments),
	}
}

// CopyIssue performs a deep copy of the given Issue.
func CopyIssue(issue *mpb.Issue) *mpb.Issue {
	result := &mpb.Issue{}
	proto.Merge(result, issue)
	return result
}

// CopyComments performs a deep copy of the given Comment.
func CopyComments(comments []*mpb.Comment) []*mpb.Comment {
	var result []*mpb.Comment
	for _, c := range comments {
		copy := &mpb.Comment{}
		proto.Merge(copy, c)
		result = append(result, copy)
	}
	return result
}
