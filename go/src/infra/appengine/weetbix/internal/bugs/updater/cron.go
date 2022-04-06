// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package updater

import (
	"context"
	"time"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/tsmon"
	"go.chromium.org/luci/common/tsmon/field"
	"go.chromium.org/luci/common/tsmon/metric"
	"go.chromium.org/luci/common/tsmon/types"

	"infra/appengine/weetbix/internal/analysis"
	"infra/appengine/weetbix/internal/bugs"
	"infra/appengine/weetbix/internal/bugs/monorail"
	"infra/appengine/weetbix/internal/clustering/runs"
	"infra/appengine/weetbix/internal/config"
	"infra/appengine/weetbix/internal/config/compiledcfg"
)

var (
	// statusGauge reports the status of the bug updater job.
	// Reports either "success" or "failure".
	statusGauge = metric.NewString("weetbix/bug_updater/status",
		"Whether automatic bug updates are succeeding, by LUCI Project.",
		nil,
		// The LUCI project.
		field.String("project"),
	)

	durationGauge = metric.NewFloat("weetbix/bug_updater/duration",
		"How long it is taking to update bugs, by LUCI Project.",
		&types.MetricMetadata{
			Units: types.Seconds,
		},
		// The LUCI project.
		field.String("project"))
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

func init() {
	// Register metrics as global metrics, which has the effort of
	// resetting them after every flush.
	tsmon.RegisterGlobalCallback(func(ctx context.Context) {
		// Do nothing -- the metrics will be populated by the cron
		// job itself and does not need to be triggered externally.
	}, statusGauge, durationGauge)
}

// UpdateAnalysisAndBugs updates BigQuery analysis, and then updates bugs
// to reflect this analysis.
// Simulate, if true, avoids any changes being applied to monorail and logs
// the changes which would be made instead. This must be set when running
// on developer computers as Weetbix-initiated monorail changes will appear
// on monorail as the developer themselves rather than the Weetbix service.
// This leads to bugs errounously being detected as having manual priority
// changes.
func UpdateAnalysisAndBugs(ctx context.Context, monorailHost, gcpProject string, simulate, enable bool) (retErr error) {
	projectCfg, err := config.Projects(ctx)
	if err != nil {
		return err
	}

	statusByProject := make(map[string]string)
	for project := range projectCfg {
		// Until each project succeeds, report "failure".
		statusByProject[project] = "failure"
	}
	defer func() {
		for project, status := range statusByProject {
			statusGauge.Set(ctx, status, project)
		}
	}()

	mc, err := monorail.NewClient(ctx, monorailHost)
	if err != nil {
		return err
	}

	ac, err := analysis.NewClient(ctx, gcpProject)
	if err != nil {
		return err
	}
	defer func() {
		if err := ac.Close(); err != nil && retErr == nil {
			retErr = errors.Annotate(err, "closing analysis client").Err()
		}
	}()

	projectsWithDataset, err := ac.ProjectsWithDataset(ctx)
	if err != nil {
		return errors.Annotate(err, "querying projects with dataset").Err()
	}

	var errs []error
	// In future, the 10 minute GAE request limit may mean we need to
	// parallelise these tasks or fan them out as separate tasks.
	for project := range projectCfg {
		start := time.Now()
		if _, ok := projectsWithDataset[project]; !ok {
			// Dataset not provisioned for project.
			continue
		}

		opts := updateOptions{
			appID:              gcpProject,
			project:            project,
			analysisClient:     ac,
			monorailClient:     mc,
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
		} else {
			statusByProject[project] = "success"
		}
		elapsed := time.Since(start)
		durationGauge.Set(ctx, elapsed.Seconds(), project)
	}
	if len(errs) > 0 {
		return errors.NewMultiError(errs...)
	}
	return nil
}

type updateOptions struct {
	appID              string
	project            string
	analysisClient     AnalysisClient
	monorailClient     *monorail.Client
	enableBugUpdates   bool
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

	projectCfg, err := compiledcfg.Project(ctx, opts.project, progress.LatestConfigVersion)
	if err != nil {
		return errors.Annotate(err, "read project config").Err()
	}

	if err := opts.analysisClient.RebuildAnalysis(ctx, opts.project); err != nil {
		return errors.Annotate(err, "update cluster summaries").Err()
	}

	if !opts.enableBugUpdates {
		return nil
	}

	monorailCfg := projectCfg.Config.Monorail
	mgrs := make(map[string]BugManager)

	mbm := monorail.NewBugManager(opts.monorailClient, opts.appID, opts.project, monorailCfg)
	mbm.Simulate = opts.simulateBugUpdates
	mgrs[bugs.MonorailSystem] = mbm

	bu := NewBugUpdater(opts.project, mgrs, opts.analysisClient, projectCfg)
	bu.MaxBugsFiledPerRun = opts.maxBugsFiledPerRun
	if err := bu.Run(ctx, progress); err != nil {
		return errors.Annotate(err, "update bugs").Err()
	}
	return nil
}
