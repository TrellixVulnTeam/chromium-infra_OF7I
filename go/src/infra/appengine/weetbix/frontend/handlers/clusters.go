// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package handlers

import (
	"net/http"

	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/server/router"

	"infra/appengine/weetbix/internal/analysis"
	"infra/appengine/weetbix/internal/clustering"
)

// ListClusters serves a GET request for /api/projects/:project/clusters.
func (h *Handlers) ListClusters(ctx *router.Context) {
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
	projectID, projectCfg, ok := obtainProjectConfigOrError(ctx)
	if !ok {
		return
	}
	opts := analysis.ImpactfulClusterReadOptions{
		Project:    projectID,
		Thresholds: projectCfg.BugFilingThreshold,
	}
	clusters, err := ac.ReadImpactfulClusters(ctx.Context, opts)
	if err != nil {
		logging.Errorf(ctx.Context, "Reading Clusters from BigQuery: %s", err)
		http.Error(ctx.Writer, "Internal server error.", http.StatusInternalServerError)
		return
	}
	respondWithJSON(ctx, clusters)
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
