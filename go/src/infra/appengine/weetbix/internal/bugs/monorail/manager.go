// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package monorail

import (
	"context"
	"fmt"
	"regexp"

	"infra/appengine/weetbix/internal/bugs"
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
	// The snapshot of monorail configuration to use for the project.
	monorailCfg *config.MonorailProject
	// Simulate, if set, tells BugManager not to make mutating changes
	// to monorail but only log the changes it would make. Must be set
	// when running locally as RPCs made from developer systems will
	// appear as that user, which breaks the detection of user-made
	// priority changes vs system-made priority changes.
	Simulate bool
}

// NewBugManager initialises a new bug manager, using the specified
// monorail client.
func NewBugManager(client *Client, monorailCfg *config.MonorailProject) *BugManager {
	return &BugManager{
		client:      client,
		monorailCfg: monorailCfg,
		Simulate:    false,
	}
}

// Create creates a new bug for the given request, returning its name, or
// any encountered error.
func (m *BugManager) Create(ctx context.Context, request *bugs.CreateRequest) (string, error) {
	g, err := NewGenerator(request.Impact, m.monorailCfg)
	if err != nil {
		return "", errors.Annotate(err, "create issue generator").Err()
	}
	req := g.PrepareNew(request.Description)
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
	impact *bugs.ClusterImpact
	issue  *mpb.Issue
}

// Update updates the specified list of bugs.
func (m *BugManager) Update(ctx context.Context, bugsToUpdate []*bugs.BugToUpdate) error {
	// Fetch issues for bugs to update.
	cis, err := m.fetchIssues(ctx, bugsToUpdate)
	if err != nil {
		return err
	}
	for _, ci := range cis {
		g, err := NewGenerator(ci.impact, m.monorailCfg)
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
				impact: upd.Impact,
				issue:  issues[i],
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
