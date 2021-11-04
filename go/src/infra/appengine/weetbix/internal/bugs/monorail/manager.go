// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package monorail

import (
	"context"
	"fmt"
	"regexp"

	"infra/appengine/weetbix/internal/bugs"
	"infra/appengine/weetbix/internal/clustering"
	"infra/appengine/weetbix/internal/config"
	mpb "infra/monorailv2/api/v3/api_proto"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"google.golang.org/protobuf/encoding/prototext"
)

// monorailRe matches monorail issue names, like
// "monorail/{monorail_project}/{numeric_id}".
var monorailRe = regexp.MustCompile(`^projects/([a-z0-9\-_]+)/issues/([0-9]+)$`)

var textPBMultiline = prototext.MarshalOptions{
	Multiline: true,
}

// monorailPageSize is the maximum number of issues that can be requested
// through GetIssues at a time. This limit is set by monorail.
const monorailPageSize = 100

// BugManager controls the creation of, and updates to, monorail bugs
// for clusters.
type BugManager struct {
	client *Client
	// The snapshot of monorail configuration to use for each project.
	monorailCfgs map[string]*config.MonorailProject
	// Simulate, if set, tells BugManager not to make mutating changes
	// to monorail but only log the changes it would make. Must be set
	// when running locally as RPCs made from developer systems will
	// appear as that user, which breaks the detection of user-made
	// priority changes vs system-made priority changes.
	Simulate bool
}

// NewBugManager initialises a new bug manager, using the specified
// monorail client.
func NewBugManager(client *Client, monorailCfgs map[string]*config.MonorailProject) *BugManager {
	return &BugManager{
		client:       client,
		monorailCfgs: monorailCfgs,
		Simulate:     false,
	}
}

// Create creates a new bug for the given cluster, returning its name, or
// any encountered error.
func (m *BugManager) Create(ctx context.Context, cluster *clustering.Cluster) (string, error) {
	monorailCfg, ok := m.monorailCfgs[cluster.Project]
	if !ok {
		return "", fmt.Errorf("no monorail configuration exists for project %q", cluster.Project)
	}
	g, err := NewGenerator(cluster, monorailCfg)
	if err != nil {
		return "", errors.Annotate(err, "create issue generator").Err()
	}
	req := g.PrepareNew()
	if m.Simulate {
		logging.Debugf(ctx, "Would create Monorail issue: %s", textPBMultiline.Format(req))
		return "", bugs.ErrCreateSimulated
	}
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

type clusterIssue struct {
	cluster *clustering.Cluster
	issue   *mpb.Issue
}

// Update updates the specified list of bugs.
func (m *BugManager) Update(ctx context.Context, bugs []*bugs.BugToUpdate) error {
	// Fetch issues for bugs to update.
	cis, err := m.fetchIssues(ctx, bugs)
	if err != nil {
		return err
	}
	for _, ci := range cis {
		monorailCfg, ok := m.monorailCfgs[ci.cluster.Project]
		if !ok {
			return fmt.Errorf("no monorail configuration exists for project %q", ci.cluster.Project)
		}
		g, err := NewGenerator(ci.cluster, monorailCfg)
		if err != nil {
			return errors.Annotate(err, "create issue generator").Err()
		}
		if g.NeedsUpdate(ci.issue) {
			comments, err := m.client.ListComments(ctx, ci.issue.Name)
			if err != nil {
				return err
			}
			req := g.MakeUpdate(ci.issue, comments)
			if m.Simulate {
				logging.Debugf(ctx, "Would update Monorail issue: %s", textPBMultiline.Format(req))
			} else {
				if err := m.client.ModifyIssues(ctx, req); err != nil {
					return errors.Annotate(err, "failed to update to issue %s", ci.issue.Name).Err()
				}
			}
		}
	}
	return nil
}

func (m *BugManager) fetchIssues(ctx context.Context, updates []*bugs.BugToUpdate) ([]*clusterIssue, error) {
	// Calculate the number of requests required, rounding up
	// to the nearest page.
	pages := (len(updates) + (monorailPageSize - 1)) / monorailPageSize

	var clusterIssues []*clusterIssue
	for i := 0; i < pages; i++ {
		// Divide bug clusters into pages of monorailPageSize.
		pageEnd := i*monorailPageSize + (monorailPageSize - 1)
		if pageEnd > len(updates) {
			pageEnd = len(updates)
		}
		updatesPage := updates[i*monorailPageSize : pageEnd]

		var names []string
		for _, upd := range updatesPage {
			name, err := toMonorailIssueName(upd.BugName)
			if err != nil {
				return nil, err
			}
			names = append(names, name)
		}
		// Guarantees result array in 1:1 correspondence to requested names.
		issues, err := m.client.BatchGetIssues(ctx, names)
		if err != nil {
			return nil, err
		}
		for i, upd := range updatesPage {
			clusterIssues = append(clusterIssues, &clusterIssue{
				cluster: upd.Cluster,
				issue:   issues[i],
			})
		}
	}
	return clusterIssues, nil
}

// toMonorailIssueName converts an internal bug name like
// "{monorail_project}/{numeric_id}" to a monorail issue name like
// "projects/{project}/issues/{numeric_id}".
func toMonorailIssueName(bug string) (string, error) {
	parts := bugs.MonorailBugIDRe.FindStringSubmatch(bug)
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
