// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package metrics

import (
	"context"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/tsmon"
	"go.chromium.org/luci/common/tsmon/distribution"
	"go.chromium.org/luci/common/tsmon/field"
	"go.chromium.org/luci/common/tsmon/metric"
	"go.chromium.org/luci/common/tsmon/types"
	"go.chromium.org/luci/server/span"

	"infra/appengine/weetbix/internal/clustering/rules"
	"infra/appengine/weetbix/internal/config"
	"infra/appengine/weetbix/internal/ingestion/control"
)

var (
	activeRulesGauge = metric.NewInt(
		"weetbix/clustering/active_rules",
		"The total number of active rules, by LUCI project.",
		&types.MetricMetadata{Units: "rules"},
		// The LUCI Project.
		field.String("project"))

	creationTimeStatusDist = metric.NewNonCumulativeDistribution(
		"weetbix/ingestion/join/presubmit_creation_time_status",
		"The distribution of creation times for presubmit build ingestion "+
			"control records. "+
			"Filtered to control records created in the last 24 hours, "+
			"broken down by LUCI Project and join status.",
		&types.MetricMetadata{Units: "hours ago"},
		distribution.FixedWidthBucketer(1, 24),
		// The LUCI Project.
		field.String("project"),
		// Status: "joined", "awaiting_presubmit", "awaiting_build".
		field.String("status"))
)

func init() {
	// Register metrics as global metrics, which has the effort of
	// resetting them after every flush.
	tsmon.RegisterGlobalCallback(func(ctx context.Context) {
		// Do nothing -- the metrics will be populated by the cron
		// job itself and does not need to be triggered externally.
	}, activeRulesGauge, creationTimeStatusDist)
}

// GlobalMetrics handles the "global-metrics" cron job. It reports
// metrics related to overall system state (that are not logically
// reported as part of individual task or cron job executions).
func GlobalMetrics(ctx context.Context) error {
	projectConfigs, err := config.Projects(ctx)
	if err != nil {
		return errors.Annotate(err, "obtain project configs").Err()
	}

	// Total number of active rules, broken down by project.
	activeRules, err := rules.ReadTotalActiveRules(span.Single(ctx))
	if err != nil {
		return errors.Annotate(err, "collect total active rules").Err()
	}
	for project := range projectConfigs {
		// If there is no entry in activeRules for this project
		// (e.g. because there are no rules in that project),
		// the read count defaults to zero, which is the correct
		// behaviour.
		count := activeRules[project]
		activeRulesGauge.Set(ctx, count, project)
	}

	// Performance joining presubmit (build + presubmit) completion data
	// in ingestion.
	joinStats, err := control.ReadPresubmitJoinStatistics(span.Single(ctx))
	if err != nil {
		return errors.Annotate(err, "collect presubmit build join statistics").Err()
	}
	for project := range projectConfigs {
		joinedDist := distribution.New(creationTimeStatusDist.Bucketer())
		awaitingBuildDist := distribution.New(creationTimeStatusDist.Bucketer())
		awaitingPresubmitDist := distribution.New(creationTimeStatusDist.Bucketer())

		if stats, ok := joinStats[project]; ok {
			for hoursAgo := 0; hoursAgo < control.PresubmitJoinStatsHours; hoursAgo++ {
				awaitingPresubmit := stats.AwaitingPresubmitResultByHour[hoursAgo]
				awaitingBuild := stats.AwaitingBuildByHour[hoursAgo]
				joinedBuilds := stats.TotalBuildsByHour[hoursAgo] - awaitingBuild - awaitingPresubmit
				for i := int64(0); i < joinedBuilds; i++ {
					joinedDist.Add(float64(hoursAgo))
				}
				for i := int64(0); i < awaitingPresubmit; i++ {
					awaitingPresubmitDist.Add(float64(hoursAgo))
				}
				for i := int64(0); i < awaitingBuild; i++ {
					awaitingBuildDist.Add(float64(hoursAgo))
				}
			}
		}

		creationTimeStatusDist.Set(ctx, joinedDist, project, "joined")
		creationTimeStatusDist.Set(ctx, awaitingBuildDist, project, "awaiting_build")
		creationTimeStatusDist.Set(ctx, awaitingPresubmitDist, project, "awaiting_presubmit")
	}
	return nil
}
