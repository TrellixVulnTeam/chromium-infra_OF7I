// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package gerrit

import (
	"context"
	"fmt"
	"infra/chromium/bootstrapper/gob"

	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/common/api/gerrit"
	"go.chromium.org/luci/common/errors"
	gerritpb "go.chromium.org/luci/common/proto/gerrit"
	"google.golang.org/grpc"
)

// TODO(gbeaty) Enforce uniqueness of change numbers for a host in the fake and
// remove project from the ID
type changeId struct {
	host    string
	project string
	change  int64
}

type Client struct {
	clients map[string]GerritClient
	factory GerritClientFactory

	// changeInfo maps host -> project -> change number -> change info
	changeInfo map[changeId]*gerritpb.ChangeInfo
}

// GerritClient provides a subset of the generated gerrit RPC client.
type GerritClient interface {
	GetChange(ctx context.Context, in *gerritpb.GetChangeRequest, opts ...grpc.CallOption) (*gerritpb.ChangeInfo, error)
}

// Enforce that the GerritClient interface is a subset of the generated client
// interface.
var _ GerritClient = (gerritpb.GerritClient)(nil)

// GerritClientFactory creates clients for accessing each necessary gerrit
// instance.
type GerritClientFactory func(ctx context.Context, host string) (GerritClient, error)

var ctxKey = "infra/chromium/bootstrapper/gerrit.GerritClientFactory"

// UseGerritClientFactory returns a context that causes new Client instances to
// use the given factory when getting gerrit clients.
func UseGerritClientFactory(ctx context.Context, factory GerritClientFactory) context.Context {
	return context.WithValue(ctx, &ctxKey, factory)
}

func NewClient(ctx context.Context) *Client {
	factory, _ := ctx.Value(&ctxKey).(GerritClientFactory)
	if factory == nil {
		factory = func(ctx context.Context, host string) (GerritClient, error) {
			authClient, err := auth.NewAuthenticator(ctx, auth.SilentLogin, auth.Options{Scopes: []string{gerrit.OAuthScope}}).Client()
			if err != nil {
				return nil, fmt.Errorf("could not initialize auth client: %w", err)
			}
			return gerrit.NewRESTClient(authClient, host, true)
		}
	}
	return &Client{
		clients: map[string]GerritClient{},
		factory: factory,
	}
}

func (c *Client) gerritClientForHost(ctx context.Context, host string) (GerritClient, error) {
	if client, ok := c.clients[host]; ok {
		return client, nil
	}
	client, err := c.factory(ctx, host)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, errors.Reason("returned client for %s is nil", host).Err()
	}
	c.clients[host] = client
	return client, nil
}

func (c *Client) getChangeInfo(ctx context.Context, host, project string, change int64) (*gerritpb.ChangeInfo, error) {
	id := changeId{host, project, change}
	if changeInfo, ok := c.changeInfo[id]; ok {
		return changeInfo, nil
	}
	gerritClient, err := c.gerritClientForHost(ctx, host)
	if err != nil {
		return nil, err
	}

	var changeInfo *gerritpb.ChangeInfo
	err = gob.Retry(ctx, "GetChange", func() error {
		var err error
		changeInfo, err = gerritClient.GetChange(ctx, &gerritpb.GetChangeRequest{
			Project: project,
			Number:  change,
			Options: []gerritpb.QueryOption{gerritpb.QueryOption_ALL_REVISIONS},
		})
		return err
	})
	if err != nil {
		return nil, err
	}

	if c.changeInfo == nil {
		c.changeInfo = map[changeId]*gerritpb.ChangeInfo{}
	}
	c.changeInfo[id] = changeInfo
	return changeInfo, nil
}

func (c *Client) GetTargetRef(ctx context.Context, host, project string, change int64) (string, error) {
	changeInfo, err := c.getChangeInfo(ctx, host, project, change)
	if err != nil {
		return "", err
	}
	return changeInfo.Ref, nil
}

func (c *Client) GetRevision(ctx context.Context, host, project string, change int64, patchset int32) (string, error) {
	changeInfo, err := c.getChangeInfo(ctx, host, project, change)
	if err != nil {
		return "", err
	}
	for rev, revInfo := range changeInfo.Revisions {
		if revInfo.Number == patchset {
			return rev, nil
		}
	}
	return "", errors.Reason("%s/c/%s/+/%d does not have patchset %d", host, project, change, patchset).Err()
}
