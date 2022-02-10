// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package gitiles

import (
	"context"
	"fmt"
	"time"

	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/common/api/gitiles"
	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/errors"
	gitilespb "go.chromium.org/luci/common/proto/gitiles"
	"go.chromium.org/luci/common/retry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Client provides the gitiles-oriented operations required for bootstrapping.
type Client struct {
	clients map[string]GitilesClient
	factory GitilesClientFactory
}

// GitilesClient provides a subset of the generated gitiles RPC client.
type GitilesClient interface {
	Log(context.Context, *gitilespb.LogRequest, ...grpc.CallOption) (*gitilespb.LogResponse, error)
	DownloadFile(context.Context, *gitilespb.DownloadFileRequest, ...grpc.CallOption) (*gitilespb.DownloadFileResponse, error)
	DownloadDiff(context.Context, *gitilespb.DownloadDiffRequest, ...grpc.CallOption) (*gitilespb.DownloadDiffResponse, error)
}

// Enforce that the GitilesClient interface is a subset of the generated client
// interface.
var _ GitilesClient = (gitilespb.GitilesClient)(nil)

// GitilesClientFactory creates clients for accessing each necessary gitiles
// instance.
type GitilesClientFactory func(ctx context.Context, host string) (GitilesClient, error)

var ctxKey = "infra/chromium/bootstrapper/gitiles.GitilesClientFactory"

// UseGitilesClientFactory returns a context that causes new Client instances to
// use the given factory when getting gitiles clients.
func UseGitilesClientFactory(ctx context.Context, factory GitilesClientFactory) context.Context {
	return context.WithValue(ctx, &ctxKey, factory)
}

// NewClient returns a new gitiles client.
//
// If ctx is a context returned from UseGitilesClientFactory, then the returned
// client will use the factory that was passed to UseGitilesClientFactory when
// creating gitiles clients. Otherwise, a factory that creates gitiles clients
// using gitiles.NewRESTClient and http.DefaultClient will be used.
func NewClient(ctx context.Context) *Client {
	factory, _ := ctx.Value(&ctxKey).(GitilesClientFactory)
	if factory == nil {
		factory = func(ctx context.Context, host string) (GitilesClient, error) {
			authClient, err := auth.NewAuthenticator(ctx, auth.SilentLogin, auth.Options{Scopes: []string{gitiles.OAuthScope}}).Client()
			if err != nil {
				return nil, fmt.Errorf("could not initialize auth client: %w", err)
			}
			return gitiles.NewRESTClient(authClient, host, true)
		}
	}
	return &Client{map[string]GitilesClient{}, factory}
}

func (c *Client) gitilesClientForHost(ctx context.Context, host string) (GitilesClient, error) {
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

// FetchLatestRevision returns the git commit hash for the latest change to the
// given ref of the given project on the given host.
func (c *Client) FetchLatestRevision(ctx context.Context, host, project, ref string) (string, error) {
	gitilesClient, err := c.gitilesClientForHost(ctx, host)
	if err != nil {
		return "", err
	}
	request := &gitilespb.LogRequest{
		Project:    project,
		Committish: ref,
		PageSize:   1,
	}
	response, err := gitilesClient.Log(ctx, request)
	if err != nil {
		return "", err
	}
	return response.Log[0].GetId(), nil
}

type downloadFileRetryIterator struct {
	backoff retry.ExponentialBackoff
}

func (i *downloadFileRetryIterator) Next(ctx context.Context, err error) time.Duration {
	s, ok := status.FromError(err)
	if ok && s.Code() == codes.NotFound {
		return i.backoff.Next(ctx, err)
	}
	return retry.Stop
}

// DownloadFile returns the contents of the file at the given path at the given
// revision of the given project on the given host.
func (c *Client) DownloadFile(ctx context.Context, host, project, revision, path string) (string, error) {
	gitilesClient, err := c.gitilesClientForHost(ctx, host)
	if err != nil {
		return "", err
	}
	request := &gitilespb.DownloadFileRequest{
		Project:    project,
		Committish: revision,
		Path:       path,
	}

	var response *gitilespb.DownloadFileResponse
	retryFactory := func() retry.Iterator {
		return &downloadFileRetryIterator{
			backoff: retry.ExponentialBackoff{
				Limited: retry.Limited{
					Delay:   time.Second,
					Retries: 5,
				},
				Multiplier: 2,
			},
		}
	}
	clockCtx := clock.Tag(ctx, "gob-retry") // used by tests
	err = retry.Retry(clockCtx, retryFactory, func() error {
		var err error
		response, err = gitilesClient.DownloadFile(ctx, request)
		return err
	}, retry.LogCallback(ctx, "DownloadFile"))
	if err != nil {
		return "", err
	}

	return response.Contents, nil
}

// DownloadDiff returns the diff between a given revision and its parent.
func (c *Client) DownloadDiff(ctx context.Context, host, project, revision string) (string, error) {
	gitilesClient, err := c.gitilesClientForHost(ctx, host)
	if err != nil {
		return "", err
	}
	request := &gitilespb.DownloadDiffRequest{
		Project:    project,
		Committish: revision,
	}
	response, err := gitilesClient.DownloadDiff(ctx, request)
	if err != nil {
		return "", err
	}
	return response.Contents, nil
}
