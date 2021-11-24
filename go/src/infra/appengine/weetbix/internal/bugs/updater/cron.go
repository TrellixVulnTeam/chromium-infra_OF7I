// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package updater

import (
	"context"

	"infra/appengine/weetbix/internal/analysis"
	"infra/appengine/weetbix/internal/bugs"
	"infra/appengine/weetbix/internal/bugs/monorail"
	"infra/appengine/weetbix/internal/clustering/runs"
	"infra/appengine/weetbix/internal/config"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
)

// AnalysisClient is an interface for building and accessing cluster analysis.
type AnalysisClient interface {
	// RebuildAnalysis rebuilds analysis from the latest clustered test
	// results.
	RebuildAnalysis(ctx context.Context, project string) error
	// ReadImpactfulClusters reads analysis for clusters matching the
	// specified criteria.
	ReadImpactfulClusters(ctx context.Context, opts analysis.ImpactfulClusterReadOptions) ([]*analysis.ClusterSummary, error)
}

// UpdateAnalysisAndBugs updates BigQuery analysis, and then updates bugs
// to reflect this analysis.
// Simulate, if true, avoids any changes being applied to monorail and logs
// the changes which would be made instead. This must be set when running
// on developer computers as Weetbix-initiated monorail changes will appear
// on monorail as the developer themselves rather than the Weetbix service.
// This leads to bugs errounously being detected as having manual priority
// changes.
func UpdateAnalysisAndBugs(ctx context.Context, monorailHost, projectID string, simulate bool) error {
	mc, err := monorail.NewClient(ctx, monorailHost)
	if err != nil {
		return err
	}
	ac, err := analysis.NewClient(ctx, projectID)
	if err != nil {
		return err
	}
	projectCfg, err := config.Projects(ctx)
	if err != nil {
		return err
	}
	var errs []error
	// In future, the 10 minute GAE request limit may mean we need to
	// parallelise these tasks or fan them out as separate tasks.
	for project, cfg := range projectCfg {
		opts := updateOptions{
			project:            project,
			analysisClient:     ac,
			monorailClient:     mc,
			projectConfig:      cfg,
			simulateBugUpdates: simulate,
			maxBugsFiledPerRun: 1,
		}
		// Isolate other projects from bug update errors
		// in one project.
		err := updateAnalysisAndBugsForProject(ctx, opts)
		if err != nil {
			err = errors.Annotate(err, "in project %v", project).Err()
			logging.Errorf(ctx, "Updating analysis and bugs: %s", err)
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errors.NewMultiError(errs...)
	}
	return nil
}

type updateOptions struct {
	project            string
	analysisClient     AnalysisClient
	monorailClient     *monorail.Client
	projectConfig      *config.ProjectConfig
	simulateBugUpdates bool
	maxBugsFiledPerRun int
}

// updateAnalysisAndBugsForProject updates BigQuery analysis, and
// Weetbix-managed bugs for a particular LUCI project.
func updateAnalysisAndBugsForProject(ctx context.Context, opts updateOptions) error {
	// Capture the current state of re-clustering before running analysis.
	// This will reflect how up-to-date our analysis is when it completes.
	progress, err := runs.ReadReclusteringProgress(ctx, opts.project)
	if err != nil {
		return errors.Annotate(err, "read re-clustering progress").Err()
	}

	if err := opts.analysisClient.RebuildAnalysis(ctx, opts.project); err != nil {
		return errors.Annotate(err, "update cluster summaries").Err()
	}

	monorailCfg := opts.projectConfig.Monorail
	thresholds := opts.projectConfig.BugFilingThreshold
	mgrs := make(map[string]BugManager)

	mbm := monorail.NewBugManager(opts.monorailClient, monorailCfg)
	mbm.Simulate = opts.simulateBugUpdates
	mgrs[bugs.MonorailSystem] = mbm

	bu := NewBugUpdater(opts.project, mgrs, opts.analysisClient, thresholds)
	bu.MaxBugsFiledPerRun = opts.maxBugsFiledPerRun
	if err := bu.Run(ctx, progress); err != nil {
		return errors.Annotate(err, "update bugs").Err()
	}
	return nil
}
