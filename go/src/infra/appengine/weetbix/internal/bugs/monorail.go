// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package bugs

import (
	"context"
	"fmt"
	"net/http"

	mpb "infra/monorailv2/api/v3/api_proto"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/grpc/prpc"
	"go.chromium.org/luci/server/auth"
)

var testMonorailClientKey = "used in tests only for setting the monorail client test double"

func newIssueClient(ctx context.Context, host string) (mpb.IssuesClient, error) {
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
func NewMonorailClient(ctx context.Context, host string) (*MonorailClient, error) {
	client, err := newIssueClient(ctx, host)
	if err != nil {
		return nil, err
	}

	return &MonorailClient{
		client: client,
	}, nil
}

// MonorailClient is a client to communicate with the Monorail issue tracker.
type MonorailClient struct {
	client mpb.IssuesClient
}

// GetIssue retrieves the details of a monorail issue. Name should
// follow the format "projects/<projectid>/issues/<issueid>".
func (c *MonorailClient) GetIssue(ctx context.Context, name string) (*mpb.Issue, error) {
	req := mpb.GetIssueRequest{Name: name}
	resp, err := c.client.GetIssue(ctx, &req)
	if err != nil {
		return nil, errors.Annotate(err, "GetIssue %q", name).Err()
	}
	return resp, nil
}
