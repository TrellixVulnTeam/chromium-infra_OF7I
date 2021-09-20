// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package monorail

import (
	"context"
	"fmt"
	"net/http"

	mpb "infra/monorailv2/api/v3/api_proto"

	"google.golang.org/protobuf/proto"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/grpc/prpc"
	"go.chromium.org/luci/server/auth"
)

var testMonorailClientKey = "used in tests only for setting the monorail client test double"

func newClient(ctx context.Context, host string) (mpb.IssuesClient, error) {
	if testClient, ok := ctx.Value(&testMonorailClientKey).(mpb.IssuesClient); ok {
		return testClient, nil
	}

	// Reference: go/dogfood-monorail-v3-api
	apiHost := fmt.Sprintf("api-dot-%v", host)
	audience := fmt.Sprintf("https://%v", host)
	t, err := auth.GetRPCTransport(ctx, auth.AsSelf, auth.WithIDTokenAudience(audience))
	if err != nil {
		return nil, err
	}
	// httpClient is able to make HTTP requests authenticated with
	// ID tokens.
	httpClient := &http.Client{Transport: t}
	monorailPRPCClient := &prpc.Client{
		C:    httpClient,
		Host: apiHost,
	}
	return mpb.NewIssuesPRPCClient(monorailPRPCClient), nil
}

// Creates a new Monorail client. Host is the monorail host to use,
// e.g. monorail-prod.appspot.com.
func NewClient(ctx context.Context, host string) (*Client, error) {
	client, err := newClient(ctx, host)
	if err != nil {
		return nil, err
	}

	return &Client{
		client: client,
	}, nil
}

// Client is a client to communicate with the Monorail issue tracker.
type Client struct {
	client mpb.IssuesClient
}

// GetIssue retrieves the details of a monorail issue. Name should
// follow the format "projects/<projectid>/issues/<issueid>".
func (c *Client) GetIssue(ctx context.Context, name string) (*mpb.Issue, error) {
	req := mpb.GetIssueRequest{Name: name}
	resp, err := c.client.GetIssue(ctx, &req)
	if err != nil {
		return nil, errors.Annotate(err, "GetIssue %q", name).Err()
	}
	return resp, nil
}

// GetIssues gets the details of the specified monorail issues.
// At most 100 issues can be queried at once. It is guaranteed
// that the i_th issue in the result will match the i_th issue
// requested. It is valid to request the same issue multiple
// times in the same request.
func (c *Client) GetIssues(ctx context.Context, names []string) ([]*mpb.Issue, error) {
	var deduplicatedNames []string
	requestedNames := make(map[string]bool)
	for _, name := range names {
		if !requestedNames[name] {
			deduplicatedNames = append(deduplicatedNames, name)
			requestedNames[name] = true
		}
	}
	req := mpb.BatchGetIssuesRequest{Names: deduplicatedNames}
	resp, err := c.client.BatchGetIssues(ctx, &req)
	if err != nil {
		return nil, errors.Annotate(err, "BatchGetIssues %v", deduplicatedNames).Err()
	}
	issuesByName := make(map[string]*mpb.Issue)
	for _, issue := range resp.Issues {
		issuesByName[issue.Name] = issue
	}
	var result []*mpb.Issue
	for _, name := range names {
		// Copy the proto to avoid an issue being aliased in
		// the result if the same issue is requested multiple times.
		// The caller should be able to assume each issue returned
		// is a distinct object.
		issue := &mpb.Issue{}
		proto.Merge(issue, issuesByName[name])
		result = append(result, issue)
	}
	return result, nil
}

// MakeIssue creates the given issue in monorail, adding the specified
// description.
func (c *Client) MakeIssue(ctx context.Context, req *mpb.MakeIssueRequest) (*mpb.Issue, error) {
	issue, err := c.client.MakeIssue(ctx, req)
	return issue, err
}
