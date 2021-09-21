// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package monorail

import (
	"context"
	"fmt"
	"regexp"

	"go.chromium.org/luci/common/errors"
	"google.golang.org/grpc"

	mpb "infra/monorailv2/api/v3/api_proto"
)

// projectsRE matches valid monorail project references.
var projectsRE = regexp.MustCompile(`projects/[a-z0-9\-_]+`)

// fakeIssuesClient provides a fake implementation of a monorail client, for testing. See:
// https://source.chromium.org/chromium/infra/infra/+/main:appengine/monorail/api/v3/api_proto/issues.proto
type fakeIssuesClient struct {
	store *FakeIssuesStore
	// User is the identity of the user interacting with monorail.
	user string
}

// UseFakeIssuesClient installs a given fake IssuesClient into the context so that
// it is used instead of making RPCs to monorail. The client will behave as if
// the given user is authenticated.
func UseFakeIssuesClient(ctx context.Context, store *FakeIssuesStore, user string) context.Context {
	f := &fakeIssuesClient{store: store, user: user}
	return context.WithValue(context.Background(), &testMonorailClientKey, mpb.IssuesClient(f))
}

func (f *fakeIssuesClient) GetIssue(ctx context.Context, in *mpb.GetIssueRequest, opts ...grpc.CallOption) (*mpb.Issue, error) {
	issue := f.issueByName(in.Name)
	if issue == nil {
		return nil, errors.New("issue not found")
	}
	// Copy proto so that if the consumer modifies the proto,
	// the stored proto does not change.
	return CopyIssue(issue.Issue), nil
}

func (f *fakeIssuesClient) issueByName(name string) *IssueData {
	for _, issue := range f.store.Issues {
		if issue.Issue.Name == name {
			return issue
		}
	}
	return nil
}

func (f *fakeIssuesClient) BatchGetIssues(ctx context.Context, in *mpb.BatchGetIssuesRequest, opts ...grpc.CallOption) (*mpb.BatchGetIssuesResponse, error) {
	result := &mpb.BatchGetIssuesResponse{}
	for _, name := range in.Names {
		issue := f.issueByName(name)
		if issue == nil {
			return nil, fmt.Errorf("issue %q not found", name)
		}
		// Copy proto so that if the consumer modifies the proto,
		// the stored proto does not change.
		result.Issues = append(result.Issues, CopyIssue(issue.Issue))
	}
	return result, nil
}

func (f *fakeIssuesClient) SearchIssues(ctx context.Context, in *mpb.SearchIssuesRequest, opts ...grpc.CallOption) (*mpb.SearchIssuesResponse, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeIssuesClient) ListComments(ctx context.Context, in *mpb.ListCommentsRequest, opts ...grpc.CallOption) (*mpb.ListCommentsResponse, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeIssuesClient) ModifyIssues(ctx context.Context, in *mpb.ModifyIssuesRequest, opts ...grpc.CallOption) (*mpb.ModifyIssuesResponse, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeIssuesClient) ModifyIssueApprovalValues(ctx context.Context, in *mpb.ModifyIssueApprovalValuesRequest, opts ...grpc.CallOption) (*mpb.ModifyIssueApprovalValuesResponse, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeIssuesClient) ListApprovalValues(ctx context.Context, in *mpb.ListApprovalValuesRequest, opts ...grpc.CallOption) (*mpb.ListApprovalValuesResponse, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeIssuesClient) ModifyCommentState(ctx context.Context, in *mpb.ModifyCommentStateRequest, opts ...grpc.CallOption) (*mpb.ModifyCommentStateResponse, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeIssuesClient) MakeIssueFromTemplate(ctx context.Context, in *mpb.MakeIssueFromTemplateRequest, opts ...grpc.CallOption) (*mpb.Issue, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeIssuesClient) MakeIssue(ctx context.Context, in *mpb.MakeIssueRequest, opts ...grpc.CallOption) (*mpb.Issue, error) {
	if !projectsRE.MatchString(in.Parent) {
		return nil, errors.New("parent project must be specified and match the form 'projects/{project_id}'")
	}
	// Copy the proto so that if the request proto is later modified, the save proto is not changed.
	saved := CopyIssue(in.Issue)
	saved.Name = fmt.Sprintf("%s/issues/%v", in.Parent, f.store.NextID)
	saved.Reporter = f.user

	f.store.NextID++
	issue := &IssueData{
		Issue: saved,
		Comments: []*mpb.Comment{
			{
				Name:      fmt.Sprintf("%s/comment/1", saved.Name),
				State:     mpb.IssueContentState_ACTIVE,
				Type:      mpb.Comment_DESCRIPTION,
				Content:   in.Description,
				Commenter: in.Issue.Reporter,
			},
		},
	}

	f.store.Issues = append(f.store.Issues, issue)

	// Copy the proto so that if the consumer modifies it, the saved proto is not changed.
	return CopyIssue(saved), nil
}
