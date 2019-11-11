// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package git

import (
	"context"
	"fmt"
	"net/http"

	gerritapi "go.chromium.org/luci/common/api/gerrit"
	gitilesapi "go.chromium.org/luci/common/api/gitiles"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/proto/gerrit"
	"go.chromium.org/luci/common/proto/gitiles"
	gitilespb "go.chromium.org/luci/common/proto/gitiles"
)

// Client consists of resources needed for querying gitiles and gerrit.
type Client struct {
	gerritC    gerrit.GerritClient
	gitilesC   gitiles.GitilesClient
	project    string
	branch     string
	latestSHA1 string
}

// NewClient produces a new client using only simple types available in a command line context
func NewClient(ctx context.Context, hc *http.Client, gerritHost, gitilesHost, project, branch string) (*Client, error) {
	if err := validateNewClientParams(gerritHost, gitilesHost, project, branch); err != nil {
		return nil, err
	}

	c := &Client{}
	if err := c.Init(ctx, hc, gerritHost, gitilesHost, project, branch); err != nil {
		return nil, err
	}

	return c, nil
}

// Init takes an http Client, hostnames, a project name, and a branch and populates the fields of the client
func (c *Client) Init(ctx context.Context, hc *http.Client, gerritHost string, gitilesHost string, project string, branch string) error {
	var err error
	c.project = project
	c.branch = branch
	c.gerritC, err = gerritapi.NewRESTClient(hc, gerritHost, true)
	if err != nil {
		return err
	}

	c.gitilesC, err = gitilesapi.NewRESTClient(hc, gitilesHost, true)
	if err != nil {
		return err
	}

	c.latestSHA1, err = c.fetchLatestSHA1(ctx)
	if err != nil {
		return err
	}

	return nil
}

// GetFile returns the contents of the file located at a given path within the project
func (c *Client) GetFile(ctx context.Context, path string) (string, error) {
	if c.latestSHA1 == "" {
		return "", fmt.Errorf("Client::GetFile: stableversion git client not initialized")
	}
	req := &gitilespb.DownloadFileRequest{
		Project:    c.project,
		Committish: c.latestSHA1,
		Path:       path,
		Format:     gitilespb.DownloadFileRequest_TEXT,
	}
	res, err := c.gitilesC.DownloadFile(ctx, req)
	if err != nil {
		return "", err
	}
	if res == nil {
		panic(fmt.Sprintf("gitiles.DownloadFile unexpectedly returned nil on success path (%s)", path))
	}
	return res.Contents, nil
}

func (c *Client) fetchLatestSHA1(ctx context.Context) (string, error) {
	resp, err := c.gitilesC.Log(ctx, &gitilespb.LogRequest{
		Project:    c.project,
		Committish: fmt.Sprintf("refs/heads/%s", c.branch),
		PageSize:   1,
	})
	if err != nil {
		return "", errors.Annotate(err, "fetch sha1 for %s branch of %s", c.branch, c.project).Err()
	}

	if len(resp.Log) == 0 {
		return "", fmt.Errorf("fetch sha1 for %s branch of %s: empty git-log", c.branch, c.project)
	}

	return resp.Log[0].GetId(), nil
}

func validateNewClientParams(gerritHost string, gitilesHost string, project string, branch string) error {
	if gerritHost == "" {
		return fmt.Errorf("gerritHost cannot be empty")
	}

	if gitilesHost == "" {
		return fmt.Errorf("gitilesHost cannot be empty")
	}

	if project == "" {
		return fmt.Errorf("project cannot be empty")
	}

	if branch == "" {
		return fmt.Errorf("branch cannot be empty")
	}

	return nil
}
