// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package monorail

import (
	"context"
	"fmt"
	"regexp"

	"infra/appengine/weetbix/internal/clustering"

	"go.chromium.org/luci/common/errors"
)

// ManagerName is the name of the monorail bug manager. It is used to
// namespace bugs created by the manager.
const ManagerName = "monorail"

// bugRe matches internal bug names, like
// "{monorail_project}/{numeric_id}".
var bugRe = regexp.MustCompile(`^([a-z0-9\-_]+)/([0-9]+)$`)

// monorailRe matches monorail issue names, like
// "monorail/{monorail_project}/{numeric_id}".
var monorailRe = regexp.MustCompile(`^projects/([a-z0-9\-_]+)/issues/([0-9]+)$`)

// BugManager controls the creation of, and updates to, monorail bugs
// for clusters.
type BugManager struct {
	client *Client
}

// NewBugManager initialises a new bug manager, using the specified
// monorail client.
func NewBugManager(client *Client) *BugManager {
	return &BugManager{
		client: client,
	}
}

// Create creates a new bug for the given cluster, returning its name, or
// any encountered error.
func (m *BugManager) Create(ctx context.Context, cluster *clustering.Cluster) (string, error) {
	req := PrepareNew(cluster)

	// Save the issue in Monorail.
	issue, err := m.client.MakeIssue(ctx, req)
	if err != nil {
		return "", errors.Annotate(err, "create issue in monorail").Err()
	}
	bug, err := fromMonorailIssueName(issue.Name)
	if err != nil {
		return "", errors.Annotate(err, "parsing monorail issue name").Err()
	}
	return bug, err
}

// toMonorailIssueName converts an internal bug name like
// "{monorail_project}/{numeric_id}" to a monorail issue name like
// "projects/{project}/issues/{numeric_id}".
func toMonorailIssueName(bug string) (string, error) {
	parts := bugRe.FindStringSubmatch(bug)
	if parts == nil {
		return "", fmt.Errorf("invalid bug %q", bug)
	}
	return fmt.Sprintf("projects/%s/issues/%s", parts[1], parts[2]), nil
}

// fromMonorailIssueName converts a monorail issue name like
// "projects/{project}/issues/{numeric_id}" to an internal bug name like
// "{monorail_project}/{numeric_id}".
func fromMonorailIssueName(name string) (string, error) {
	parts := monorailRe.FindStringSubmatch(name)
	if parts == nil {
		return "", fmt.Errorf("invalid monorail issue name %q", name)
	}
	return fmt.Sprintf("%s/%s", parts[1], parts[2]), nil
}
