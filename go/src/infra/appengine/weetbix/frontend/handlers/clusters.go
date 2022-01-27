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
	"infra/appengine/weetbix/internal/clustering/algorithms"
	"infra/appengine/weetbix/internal/clustering/rules"
	"infra/appengine/weetbix/internal/config/compiledcfg"
	weetbixpb "infra/appengine/weetbix/proto/v1"
)

// ClusterCommon captures common cluster fields used in ClusterSummary
// and Cluster.
type ClusterCommon struct {
	ClusterID          clustering.ClusterID `json:"clusterId"`
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

// ClusterSummary provides a summary of a Weetbix cluster.
// It is used in the ListClusters response.
type ClusterSummary struct {
	ClusterCommon
	// Title is a one-line description of the cluster.
	Title string `json:"title"`
	// BugLink is the link to the bug associated with the cluster.
	// Set only for rule-based clusters.
	BugLink *BugLink `json:"bugLink"`
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

func (h *Handlers) listClustersInternal(ctx context.Context, projectID string, projectCfg *compiledcfg.ProjectConfig) ([]*ClusterSummary, error) {
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
		Thresholds: projectCfg.Config.BugFilingThreshold,
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
		cs := &ClusterSummary{}
		cs.ClusterCommon = newClusterCommon(c)
		if c.ClusterID.IsBugCluster() {
			rule := rules[ruleIndex]
			cs.Title = rule.RuleDefinition
			cs.BugLink = createBugLink(rule.BugID, projectCfg.Config)
			ruleIndex++
		} else {
			// Suggested cluster.
			cs.Title = suggestedClusterTitle(c, projectCfg)
		}

		result = append(result, cs)
	}
	return result, nil
}

func newClusterCommon(cs *analysis.ClusterSummary) ClusterCommon {
	return ClusterCommon{
		ClusterID:          cs.ClusterID,
		PresubmitRejects1d: cs.PresubmitRejects1d,
		PresubmitRejects3d: cs.PresubmitRejects3d,
		PresubmitRejects7d: cs.PresubmitRejects7d,
		TestRunFails1d:     cs.TestRunFails1d,
		TestRunFails3d:     cs.TestRunFails3d,
		TestRunFails7d:     cs.TestRunFails7d,
		Failures1d:         cs.Failures1d,
		Failures3d:         cs.Failures3d,
		Failures7d:         cs.Failures7d,
	}
}

func suggestedClusterTitle(cs *analysis.ClusterSummary, cfg *compiledcfg.ProjectConfig) string {
	var title string

	// Ignore error, it is only returned if algorithm cannot be found.
	alg, _ := algorithms.SuggestingAlgorithm(cs.ClusterID.Algorithm)
	switch {
	case alg != nil:
		example := &clustering.Failure{
			TestID: cs.ExampleTestID,
			Reason: &weetbixpb.FailureReason{
				PrimaryErrorMessage: cs.ExampleFailureReason.StringVal,
			},
		}
		title = alg.ClusterTitle(cfg, example)
	case cs.ClusterID.IsTestNameCluster():
		// Fallback for old test name clusters.
		title = cs.ExampleTestID
	case cs.ClusterID.IsFailureReasonCluster():
		// Fallback for old reason-based clusters.
		title = cs.ExampleFailureReason.StringVal
	default:
		// Fallback for all other cases.
		title = fmt.Sprintf("%s/%s", cs.ClusterID.Algorithm, cs.ClusterID.ID)
	}
	return title
}

// Cluster is the type provided by the GetCluster response.
type Cluster struct {
	ClusterCommon
	// Title is a one-line description of the cluster.
	// Populated only for suggested clusters.
	Title string `json:"title"`
	// The equivalent failure association rule to use if filing a new bug.
	// Populated only for suggested clusters, where the algorithm is still
	// known by Weetbix and there are recent examples.
	FailureAssociationRule string `json:"failureAssociationRule"`
}

func newCluster(cs *analysis.ClusterSummary, cfg *compiledcfg.ProjectConfig) *Cluster {
	result := &Cluster{}
	result.ClusterCommon = newClusterCommon(cs)
	if !cs.ClusterID.IsBugCluster() {
		result.Title = suggestedClusterTitle(cs, cfg)

		// Ignore error, it is only returned if algorithm cannot be found.
		alg, _ := algorithms.SuggestingAlgorithm(cs.ClusterID.Algorithm)
		if alg != nil {
			example := &clustering.Failure{
				TestID: cs.ExampleTestID,
				Reason: &weetbixpb.FailureReason{
					PrimaryErrorMessage: cs.ExampleFailureReason.StringVal,
				},
			}
			result.FailureAssociationRule = alg.FailureAssociationRule(cfg, example)
		}
	}
	return result
}

func newEmptyCluster(clusterID clustering.ClusterID) *Cluster {
	result := &Cluster{}
	result.ClusterID = clusterID
	if !clusterID.IsBugCluster() {
		result.Title = "(cluster no longer exists)"
	}
	return result
}

// GetCluster serves a GET request for
// api/projects/:project/clusters/:algorithm/:id.
func (h *Handlers) GetCluster(ctx *router.Context) {
	projectID, projectCfg, ok := obtainProjectConfigOrError(ctx)
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

	cs, err := ac.ReadCluster(ctx.Context, projectID, clusterID)
	if err != nil && err != analysis.NotExistsErr {
		logging.Errorf(ctx.Context, "Reading Cluster from BigQuery: %s", err)
		http.Error(ctx.Writer, "Internal server error.", http.StatusInternalServerError)
		return
	}
	var response *Cluster
	if err != analysis.NotExistsErr {
		response = newCluster(cs, projectCfg)
	} else {
		// Return a placeholder cluster with zero impact.
		response = newEmptyCluster(clusterID)
	}

	respondWithJSON(ctx, response)
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
