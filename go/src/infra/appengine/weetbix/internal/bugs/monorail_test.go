// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package bugs

import (
	"context"
	"errors"
	mpb "infra/monorailv2/api/v3/api_proto"
	"testing"

	"google.golang.org/grpc"

	"google.golang.org/protobuf/proto"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func TestGetIssue(t *testing.T) {
	t.Parallel()

	Convey("Get issue", t, func() {
		issue := &mpb.Issue{
			Name:     "projects/chromium/issues/123",
			Summary:  "This is a bug summary.",
			State:    mpb.IssueContentState_ACTIVE,
			Reporter: "meiring@google.com",
		}
		f := &fakeIssuesClient{
			Issues: []*mpb.Issue{issue},
		}

		ctx := context.WithValue(context.Background(), &testMonorailClientKey, mpb.IssuesClient(f))

		bc, err := NewMonorailClient(ctx, "monorailhost")
		So(err, ShouldBeNil)
		result, err := bc.GetIssue(ctx, "projects/chromium/issues/123")
		So(err, ShouldBeNil)
		So(result, ShouldResembleProto, issue)
	})
}

type fakeIssuesClient struct {
	Issues []*mpb.Issue
}

func (f *fakeIssuesClient) GetIssue(ctx context.Context, in *mpb.GetIssueRequest, opts ...grpc.CallOption) (*mpb.Issue, error) {
	for _, issue := range f.Issues {
		if issue.Name == in.Name {
			// Copy proto so that if the consumer modifies the proto,
			// the stored proto does not change.
			result := &mpb.Issue{}
			proto.Merge(result, issue)
			return result, nil
		}
	}
	return nil, errors.New("Issue not found.")
}

func (f *fakeIssuesClient) BatchGetIssues(ctx context.Context, in *mpb.BatchGetIssuesRequest, opts ...grpc.CallOption) (*mpb.BatchGetIssuesResponse, error) {
	return nil, errors.New("Not implemented.")
}

func (f *fakeIssuesClient) SearchIssues(ctx context.Context, in *mpb.SearchIssuesRequest, opts ...grpc.CallOption) (*mpb.SearchIssuesResponse, error) {
	return nil, errors.New("Not implemented.")
}

func (f *fakeIssuesClient) ListComments(ctx context.Context, in *mpb.ListCommentsRequest, opts ...grpc.CallOption) (*mpb.ListCommentsResponse, error) {
	return nil, errors.New("Not implemented.")
}

func (f *fakeIssuesClient) ModifyIssues(ctx context.Context, in *mpb.ModifyIssuesRequest, opts ...grpc.CallOption) (*mpb.ModifyIssuesResponse, error) {
	return nil, errors.New("Not implemented.")
}

func (f *fakeIssuesClient) ModifyIssueApprovalValues(ctx context.Context, in *mpb.ModifyIssueApprovalValuesRequest, opts ...grpc.CallOption) (*mpb.ModifyIssueApprovalValuesResponse, error) {
	return nil, errors.New("Not implemented.")
}

func (f *fakeIssuesClient) ListApprovalValues(ctx context.Context, in *mpb.ListApprovalValuesRequest, opts ...grpc.CallOption) (*mpb.ListApprovalValuesResponse, error) {
	return nil, errors.New("Not implemented.")
}

func (f *fakeIssuesClient) ModifyCommentState(ctx context.Context, in *mpb.ModifyCommentStateRequest, opts ...grpc.CallOption) (*mpb.ModifyCommentStateResponse, error) {
	return nil, errors.New("Not implemented.")
}

func (f *fakeIssuesClient) MakeIssueFromTemplate(ctx context.Context, in *mpb.MakeIssueFromTemplateRequest, opts ...grpc.CallOption) (*mpb.Issue, error) {
	return nil, errors.New("Not implemented.")
}

func (f *fakeIssuesClient) MakeIssue(ctx context.Context, in *mpb.MakeIssueRequest, opts ...grpc.CallOption) (*mpb.Issue, error) {
	return nil, errors.New("Not implemented.")
}
