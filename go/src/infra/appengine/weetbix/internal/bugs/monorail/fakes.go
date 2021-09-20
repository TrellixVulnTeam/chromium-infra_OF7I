// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package monorail

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"

	mpb "infra/monorailv2/api/v3/api_proto"
)

// projectsRE matches valid monorail project references.
var projectsRE = regexp.MustCompile(`projects/[a-z0-9\-_]+`)

// FakeIssuesClient provides a fake implementation of a monorail client, for testing. See:
// https://source.chromium.org/chromium/infra/infra/+/main:appengine/monorail/api/v3/api_proto/issues.proto
type FakeIssuesClient struct {
	Issues []*mpb.Issue
	NextID int
}

// UseFakeIssuesClient installs the given FakeIssuesClient into the context so that
// it is used instead of making RPCs to monorail.
func UseFakeIssuesClient(ctx context.Context, f *FakeIssuesClient) context.Context {
	return context.WithValue(context.Background(), &testMonorailClientKey, mpb.IssuesClient(f))
}

func (f *FakeIssuesClient) GetIssue(ctx context.Context, in *mpb.GetIssueRequest, opts ...grpc.CallOption) (*mpb.Issue, error) {
	issue := f.issueByName(in.Name)
	if issue == nil {
		return nil, errors.New("issue not found")
	}
	return issue, nil
}

func (f *FakeIssuesClient) issueByName(name string) *mpb.Issue {
	for _, issue := range f.Issues {
		if issue.Name == name {
			// Copy proto so that if the consumer modifies the proto,
			// the stored proto does not change.
			result := &mpb.Issue{}
			proto.Merge(result, issue)
			return result
		}
	}
	return nil
}

func (f *FakeIssuesClient) BatchGetIssues(ctx context.Context, in *mpb.BatchGetIssuesRequest, opts ...grpc.CallOption) (*mpb.BatchGetIssuesResponse, error) {
	result := &mpb.BatchGetIssuesResponse{}
	for _, name := range in.Names {
		issue := f.issueByName(name)
		if issue == nil {
			return nil, fmt.Errorf("issue %q not found", name)
		}
		result.Issues = append(result.Issues, issue)
	}
	return result, nil
}

func (f *FakeIssuesClient) SearchIssues(ctx context.Context, in *mpb.SearchIssuesRequest, opts ...grpc.CallOption) (*mpb.SearchIssuesResponse, error) {
	return nil, errors.New("not implemented")
}

func (f *FakeIssuesClient) ListComments(ctx context.Context, in *mpb.ListCommentsRequest, opts ...grpc.CallOption) (*mpb.ListCommentsResponse, error) {
	return nil, errors.New("not implemented")
}

func (f *FakeIssuesClient) ModifyIssues(ctx context.Context, in *mpb.ModifyIssuesRequest, opts ...grpc.CallOption) (*mpb.ModifyIssuesResponse, error) {
	return nil, errors.New("not implemented")
}

func (f *FakeIssuesClient) ModifyIssueApprovalValues(ctx context.Context, in *mpb.ModifyIssueApprovalValuesRequest, opts ...grpc.CallOption) (*mpb.ModifyIssueApprovalValuesResponse, error) {
	return nil, errors.New("not implemented")
}

func (f *FakeIssuesClient) ListApprovalValues(ctx context.Context, in *mpb.ListApprovalValuesRequest, opts ...grpc.CallOption) (*mpb.ListApprovalValuesResponse, error) {
	return nil, errors.New("not implemented")
}

func (f *FakeIssuesClient) ModifyCommentState(ctx context.Context, in *mpb.ModifyCommentStateRequest, opts ...grpc.CallOption) (*mpb.ModifyCommentStateResponse, error) {
	return nil, errors.New("not implemented")
}

func (f *FakeIssuesClient) MakeIssueFromTemplate(ctx context.Context, in *mpb.MakeIssueFromTemplateRequest, opts ...grpc.CallOption) (*mpb.Issue, error) {
	return nil, errors.New("not implemented")
}

func (f *FakeIssuesClient) MakeIssue(ctx context.Context, in *mpb.MakeIssueRequest, opts ...grpc.CallOption) (*mpb.Issue, error) {
	// Copy the proto so that if the request proto is later modified, the save proto is not changed.
	saved := &mpb.Issue{}
	proto.Merge(saved, in.Issue)

	if !projectsRE.MatchString(in.Parent) {
		return nil, errors.New("parent project must be specified and match the form 'projects/{project_id}'")
	}

	saved.Name = fmt.Sprintf("%s/issues/%v", in.Parent, f.NextID)
	f.NextID++
	f.Issues = append(f.Issues, saved)

	// Copy the proto so that if the consumer modifies it, the saved proto is not changed.
	result := &mpb.Issue{}
	proto.Merge(result, saved)
	return result, nil
}
