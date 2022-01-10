// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package handlers

import (
	"context"
	"fmt"
	"net/http"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/server/router"
	"go.chromium.org/luci/server/span"

	"infra/appengine/weetbix/internal/analysis"
	"infra/appengine/weetbix/internal/clustering"
	"infra/appengine/weetbix/internal/clustering/rules"
	configpb "infra/appengine/weetbix/internal/config/proto"
)

// ClusterSummary provides a summary of a Weetbix cluster.
type ClusterSummary struct {
	ClusterID          clustering.ClusterID `json:"clusterId"`
	Title              string               `json:"title"`
	BugLink            *BugLink             `json:"bugLink"`
	PresubmitRejects1d analysis.Counts      `json:"presubmitRejects1d"`
	PresubmitRejects3d analysis.Counts      `json:"presubmitRejects3d"`
	PresubmitRejects7d analysis.Counts      `json:"presubmitRejects7d"`
	TestRunFails1d     analysis.Counts      `json:"testRunFailures1d"`
	TestRunFails3d     analysis.Counts      `json:"testRunFailures3d"`
	TestRunFails7d     analysis.Counts      `json:"testRunFailures7d"`
	Failures1d         analysis.Counts      `json:"failures1d"`
	Failures3d         analysis.Counts      `json:"failures3d"`
	Failures7d         analysis.Counts      `json:"failures7d"`
}

// ListClusters serves a GET request for /api/projects/:project/clusters.
func (h *Handlers) ListClusters(ctx *router.Context) {
	projectID, projectCfg, ok := obtainProjectConfigOrError(ctx)
	if !ok {
		return
	}
	cs, err := h.listClustersInternal(ctx.Context, projectID, projectCfg)
	if err != nil {
		logging.Errorf(ctx.Context, "Listing Clusters: %s", err)
		http.Error(ctx.Writer, "Internal server error.", http.StatusInternalServerError)
		return
	}
	respondWithJSON(ctx, cs)
}

func (h *Handlers) listClustersInternal(ctx context.Context, projectID string, projectCfg *configpb.ProjectConfig) ([]*ClusterSummary, error) {
	ac, err := analysis.NewClient(ctx, h.cloudProject)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := ac.Close(); err != nil {
			logging.Warningf(ctx, "Closing analysis client: %v", err)
		}
	}()
	opts := analysis.ImpactfulClusterReadOptions{
		Project:    projectID,
		Thresholds: projectCfg.BugFilingThreshold,
	}
	clusters, err := ac.ReadImpactfulClusters(ctx, opts)
	if err != nil {
		return nil, errors.Annotate(err, "read impactful clusters").Err()
	}
	var ids []string
	for _, c := range clusters {
		if c.ClusterID.IsBugCluster() {
			ids = append(ids, c.ClusterID.ID)
		}
	}
	rules, err := rules.ReadMany(span.Single(ctx), projectID, ids)
	if err != nil {
		return nil, errors.Annotate(err, "read rules").Err()
	}
	var ruleIndex int

	var result []*ClusterSummary
	for _, c := range clusters {
		var title string
		var link *BugLink
		switch {
		case c.ClusterID.IsBugCluster():
			rule := rules[ruleIndex]
			title = rule.RuleDefinition
			link = createBugLink(rule.BugID, projectCfg)
			ruleIndex++
		case c.ClusterID.IsTestNameCluster():
			title = c.ExampleTestID
		case c.ClusterID.IsFailureReasonCluster():
			title = c.ExampleFailureReason.StringVal
		default:
			// Fallback
			title = fmt.Sprintf("%s/%s", c.ClusterID.Algorithm, c.ClusterID.ID)
		}

		cs := &ClusterSummary{
			ClusterID:          c.ClusterID,
			Title:              title,
			BugLink:            link,
			PresubmitRejects1d: c.PresubmitRejects1d,
			PresubmitRejects3d: c.PresubmitRejects3d,
			PresubmitRejects7d: c.PresubmitRejects7d,
			TestRunFails1d:     c.TestRunFails1d,
			TestRunFails3d:     c.TestRunFails3d,
			TestRunFails7d:     c.TestRunFails7d,
			Failures1d:         c.Failures1d,
			Failures3d:         c.Failures3d,
			Failures7d:         c.Failures7d,
		}
		result = append(result, cs)
	}
	return result, nil
}

// GetCluster serves a GET request for
// api/projects/:project/clusters/:algorithm/:id.
func (h *Handlers) GetCluster(ctx *router.Context) {
	projectID, ok := obtainProjectOrError(ctx)
	if !ok {
		return
	}
	clusterID := clustering.ClusterID{
		Algorithm: ctx.Params.ByName("algorithm"),
		ID:        ctx.Params.ByName("id"),
	}
	if err := clusterID.Validate(); err != nil {
		http.Error(ctx.Writer, "Please supply a valid cluster ID.", http.StatusBadRequest)
		return
	}
	ac, err := analysis.NewClient(ctx.Context, h.cloudProject)
	if err != nil {
		logging.Errorf(ctx.Context, "Creating new analysis client: %v", err)
		http.Error(ctx.Writer, "Internal server error.", http.StatusInternalServerError)
		return
	}
	defer func() {
		if err := ac.Close(); err != nil {
			logging.Warningf(ctx.Context, "Closing analysis client: %v", err)
		}
	}()

	clusters, err := ac.ReadCluster(ctx.Context, projectID, clusterID)
	if err != nil {
		logging.Errorf(ctx.Context, "Reading Cluster from BigQuery: %s", err)
		http.Error(ctx.Writer, "Internal server error.", http.StatusInternalServerError)
		return
	}

	respondWithJSON(ctx, clusters)
}

// GetClusterFailures handles a GET request for
// /api/projects/:project/clusters/:algorithm/:id/failures.
func (h *Handlers) GetClusterFailures(ctx *router.Context) {
	projectID, ok := obtainProjectOrError(ctx)
	if !ok {
		return
	}
	clusterID := clustering.ClusterID{
		Algorithm: ctx.Params.ByName("algorithm"),
		ID:        ctx.Params.ByName("id"),
	}
	if err := clusterID.Validate(); err != nil {
		http.Error(ctx.Writer, "Please supply a valid cluster ID.", http.StatusBadRequest)
		return
	}
	ac, err := analysis.NewClient(ctx.Context, h.cloudProject)
	if err != nil {
		logging.Errorf(ctx.Context, "Creating new analysis client: %v", err)
		http.Error(ctx.Writer, "Internal server error.", http.StatusInternalServerError)
		return
	}
	defer func() {
		if err := ac.Close(); err != nil {
			logging.Warningf(ctx.Context, "Closing analysis client: %v", err)
		}
	}()

	failures, err := ac.ReadClusterFailures(ctx.Context, projectID, clusterID)
	if err != nil {
		logging.Errorf(ctx.Context, "Reading Cluster from BigQuery: %s", err)
		http.Error(ctx.Writer, "Internal server error.", http.StatusInternalServerError)
		return
	}

	respondWithJSON(ctx, failures)
}
