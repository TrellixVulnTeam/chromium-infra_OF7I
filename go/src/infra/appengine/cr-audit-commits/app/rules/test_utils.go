// Copyright 2017 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package rules

import (
	"fmt"
	"strconv"
	"time"

	"context"

	"github.com/golang/protobuf/ptypes"
	google_protobuf "github.com/golang/protobuf/ptypes/timestamp"

	"google.golang.org/grpc"

	ds "go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/common/api/gerrit"

	mr "infra/monorail"
)

type mockGerritClient struct {
	q     map[string][]*gerrit.Change
	pr    map[string]bool
	e     error
	calls []string
}

func (c mockGerritClient) ChangeQuery(ctx context.Context, r gerrit.ChangeQueryParams) ([]*gerrit.Change, bool, error) {
	ret := c.q[r.Query]
	return ret, false, c.e
}
func (c *mockGerritClient) SetReview(ctx context.Context, changeid string, revision string, r *gerrit.ReviewInput) (*gerrit.ReviewResult, error) {
	c.calls = append(c.calls, "SetReview")
	return &gerrit.ReviewResult{}, nil
}

func (c mockGerritClient) ChangeDetails(ctx context.Context, cid string, p gerrit.ChangeDetailsParams) (*gerrit.Change, error) {
	ret := c.q[cid]
	return ret[0], c.e
}

func (c mockGerritClient) IsChangePureRevert(ctx context.Context, cid string) (bool, error) {
	// Say a revert is a pure revert if present in c.pr, and its value is
	// true.
	val, ok := c.pr[cid]
	return ok && val, c.e
}

func fakeRelevantCommits(n int, k *ds.Key, bh string, s AuditStatus, t time.Time, d time.Duration, a, c string) []*RelevantCommit {
	result := []*RelevantCommit{}
	for i := 0; i < n; i++ {
		prevHash := bh + strconv.Itoa(i+1)
		if i == n-1 {
			prevHash = ""
		}
		result = append(result, &RelevantCommit{
			RepoStateKey:           k,
			CommitHash:             bh + strconv.Itoa(i),
			Status:                 s,
			CommitTime:             t,
			CommitterAccount:       c,
			AuthorAccount:          a,
			CommitMessage:          fmt.Sprintf("Fake commit %d", i),
			PreviousRelevantCommit: prevHash,
		})
		t = t.Add(d)
	}
	return result
}

// MockMonorailClient is a mock monorail client that used for testing.
type MockMonorailClient struct {
	Il *mr.IssuesListResponse
	Cl *mr.ListCommentsResponse
	Ic *mr.InsertCommentResponse
	Ii *mr.InsertIssueResponse
	Gi *mr.Issue
	E  error
}

// InsertIssue pretends to create an issue.
func (c MockMonorailClient) InsertIssue(ctx context.Context, in *mr.InsertIssueRequest, opts ...grpc.CallOption) (*mr.InsertIssueResponse, error) {
	return c.Ii, c.E
}

// InsertComment pretends to post comments to an issue.
func (c MockMonorailClient) InsertComment(ctx context.Context, in *mr.InsertCommentRequest, opts ...grpc.CallOption) (*mr.InsertCommentResponse, error) {
	return c.Ic, c.E
}

// IssuesList pretends to list issues from a project.
func (c MockMonorailClient) IssuesList(ctx context.Context, in *mr.IssuesListRequest, opts ...grpc.CallOption) (*mr.IssuesListResponse, error) {
	return c.Il, c.E
}

// GetIssue pretends to get an issue by id.
func (c MockMonorailClient) GetIssue(ctx context.Context, in *mr.GetIssueRequest, opts ...grpc.CallOption) (*mr.Issue, error) {
	return c.Gi, c.E
}

// ListComments pretends to return comments of an issue.
func (c MockMonorailClient) ListComments(ctx context.Context, in *mr.ListCommentsRequest, opts ...grpc.CallOption) (*mr.ListCommentsResponse, error) {
	return c.Cl, c.E
}

// DummyRule is a rule that always returns the value of the result field when run.
type DummyRule struct {
	Name   string
	Result *RuleResult
}

// GetName returns the name of the rule.
func (rule DummyRule) GetName() string {
	return rule.Name
}

// Run returns the result struct field.
func (rule DummyRule) Run(c context.Context, ap *AuditParams, rc *RelevantCommit, cs *Clients) (*RuleResult, error) {
	return rule.Result, nil
}

// MustGitilesTime convert time string into golang protobuf timestamp.
func MustGitilesTime(v string) *google_protobuf.Timestamp {
	var t time.Time
	t, err := time.Parse(time.ANSIC, v)
	if err != nil {
		t, err = time.Parse(time.ANSIC+" -0700", v)
	}
	if err != nil {
		panic(fmt.Errorf("could not parse time %q: %v", v, err))

	}
	r, err := ptypes.TimestampProto(t)
	if err != nil {
		panic(fmt.Errorf("could not convert time %s to google_protobuf.Timestamp: %v", t, err))

	}
	return r
}
