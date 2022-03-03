// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package control

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/spanner"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/server/span"
	"google.golang.org/protobuf/proto"

	"infra/appengine/weetbix/internal/config"
	ctlpb "infra/appengine/weetbix/internal/ingestion/control/proto"
	spanutil "infra/appengine/weetbix/internal/span"
)

// PresubmitJoinStatsHours is the number of previous hours
// ReadPresubmitJoinStats reads statistics for.
const PresubmitJoinStatsHours = 24

// Entry is an ingestion control record, used to de-duplicate build ingestions
// and synchronise them with presubmit results (if required).
type Entry struct {
	// Project is the LUCI Project the chunk belongs to.
	Project string

	// The identity of the build which is being ingested.
	// The scheme is: {buildbucket host name}/{build id}.
	BuildID string

	// BuildResult is the result of the build bucket build, to be passed
	// to the result ingestion task. This is nil if the result is
	// not yet known.
	BuildResult *ctlpb.BuildResult

	// IsPresubmit records whether the build is part of a presubmit run.
	// If true, ingestion should wait for the presubmit result to be
	// populated (in addition to the build result) before commencing
	// ingestion.
	IsPresubmit bool

	// PresubmitResult is result of the presubmit run, to be passed to the
	// result ingestion task. This is nil if the result is
	// not yet known.
	PresubmitResult *ctlpb.PresubmitResult

	// LastUpdated is the Spanner commit time the row was last updated.
	LastUpdated time.Time

	// CreationTime is the Spanner commit time the row was created.
	CreationTime time.Time
}

// Read reads ingestion control records for the specified build IDs.
// Exactly one *Entry is returned for each build ID. The result entry
// at index i corresponds to the buildIDs[i].
// If a record does not exist for the given build ID, an *Entry of
// nil is returned for that build ID.
func Read(ctx context.Context, project string, buildIDs []string) ([]*Entry, error) {
	uniqueIDs := make(map[string]struct{})
	var keys []spanner.Key
	for _, buildID := range buildIDs {
		keys = append(keys, spanner.Key{project, buildID})
		if _, ok := uniqueIDs[buildID]; ok {
			return nil, fmt.Errorf("duplicate build ID %s", buildID)
		}
		uniqueIDs[buildID] = struct{}{}
	}
	cols := []string{
		"BuildID",
		"BuildResult",
		"IsPresubmit",
		"PresubmitResult",
		"LastUpdated",
		"CreationTime",
	}
	entryByBuildID := make(map[string]*Entry)
	rows := span.Read(ctx, "IngestionControl", spanner.KeySetFromKeys(keys...), cols)
	f := func(r *spanner.Row) error {
		var buildID string
		var buildResultBytes []byte
		var isPresubmit spanner.NullBool
		var presubmitResultBytes []byte
		var lastUpdated time.Time
		var creationTime spanner.NullTime

		err := r.Columns(&buildID,
			&buildResultBytes,
			&isPresubmit,
			&presubmitResultBytes,
			&lastUpdated,
			&creationTime)
		if err != nil {
			return errors.Annotate(err, "read IngestionControl row").Err()
		}
		var buildResult *ctlpb.BuildResult
		if buildResultBytes != nil {
			buildResult = &ctlpb.BuildResult{}
			if err := proto.Unmarshal(buildResultBytes, buildResult); err != nil {
				return errors.Annotate(err, "unmarshal build result").Err()
			}
		}
		var presubmitResult *ctlpb.PresubmitResult
		if presubmitResultBytes != nil {
			presubmitResult = &ctlpb.PresubmitResult{}
			if err := proto.Unmarshal(presubmitResultBytes, presubmitResult); err != nil {
				return errors.Annotate(err, "unmarshal presubmit result").Err()
			}
		}

		entryByBuildID[buildID] = &Entry{
			Project:     project,
			BuildID:     buildID,
			BuildResult: buildResult,
			// IsPresubmit uses NULL to indicate false.
			IsPresubmit:     isPresubmit.Valid && isPresubmit.Bool,
			PresubmitResult: presubmitResult,
			LastUpdated:     lastUpdated,
			CreationTime:    creationTime.Time,
		}
		return nil
	}

	if err := rows.Do(f); err != nil {
		return nil, err
	}

	var result []*Entry
	for _, buildID := range buildIDs {
		// If the entry does not exist, return nil for that build ID.
		entry := entryByBuildID[buildID]
		result = append(result, entry)
	}
	return result, nil
}

// Update updates an existing ingestion control record to match the
// specified details.
func Update(ctx context.Context, e *Entry) error {
	if err := validateEntry(e); err != nil {
		return err
	}
	m := spanutil.UpdateMap("IngestionControl", map[string]interface{}{
		"Project":         e.Project,
		"BuildId":         e.BuildID,
		"BuildResult":     e.BuildResult,
		"IsPresubmit":     spanner.NullBool{Valid: e.IsPresubmit, Bool: e.IsPresubmit},
		"PresubmitResult": e.PresubmitResult,
		"LastUpdated":     spanner.CommitTimestamp,
	})
	span.BufferWrite(ctx, m)
	return nil
}

// Create creates an ingestion control record with the specified details.
func Create(ctx context.Context, e *Entry) error {
	if err := validateEntry(e); err != nil {
		return err
	}
	m := spanutil.InsertMap("IngestionControl", map[string]interface{}{
		"Project":         e.Project,
		"BuildId":         e.BuildID,
		"BuildResult":     e.BuildResult,
		"IsPresubmit":     spanner.NullBool{Valid: e.IsPresubmit, Bool: e.IsPresubmit},
		"PresubmitResult": e.PresubmitResult,
		"LastUpdated":     spanner.CommitTimestamp,
		"CreationTime":    spanner.CommitTimestamp,
	})
	span.BufferWrite(ctx, m)
	return nil
}

// PresubmitJoinStatistics captures indicators of how well CV Run completions
// are being joined with buildbucket build completions.
type PresubmitJoinStatistics struct {
	// TotalBuildsByHour captures the total number of presubmit builds notified
	// to ingestion, either by way of a CV run completion notification and/or
	// buildbucket completion notification, by hours since the control
	// record for the build was first created. Index 0 indicates the period
	// from ]-1 hour, now], index 1 indicates [-2 hour, -1 hour] and so on.
	TotalBuildsByHour []int64
	// AwaitingBuildByHour is the number of presubmit builds which are
	// not ingested because they are pending a build completion notification
	// from buildbucket, by hours since the control record for the
	// build was first created. See TotalBuildsByHour for how to index into
	// this slice.
	AwaitingBuildByHour []int64
	// AwaitingPresubmitResultByHour is the number of presubmit builds which
	// are not ingested because they are pending a presubmit run completion
	// notification, by hours since the control record for the build was
	// first created. See TotalBuildsByHour for how to index into
	// this slice.
	AwaitingPresubmitResultByHour []int64
}

// ReadPresubmitJoinStatistics reads indicators of how well CV Run completions
// are being joined with buildbucket build completions. The last 24 hours of
// data for each project is returned.
func ReadPresubmitJoinStatistics(ctx context.Context) (map[string]PresubmitJoinStatistics, error) {
	stmt := spanner.NewStatement(`
		SELECT
		  project,
		  TIMESTAMP_DIFF(CURRENT_TIMESTAMP(), CreationTime, HOUR) as hour,
		  COUNT(*) as total,
		  COUNTIF(NOT HasBuildResult) as no_build_result,
		  COUNTIF(NOT HasPresubmitResult) as no_presubmit_result
		FROM IngestionControl@{FORCE_INDEX=IngestionControlByIsPresubmit, spanner_emulator.disable_query_null_filtered_index_check=true}
		WHERE IsPresubmit
		  AND CreationTime >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL @hours HOUR)
		GROUP BY project, hour
	`)
	stmt.Params["hours"] = PresubmitJoinStatsHours

	result := make(map[string]PresubmitJoinStatistics)
	it := span.Query(ctx, stmt)
	err := it.Do(func(r *spanner.Row) error {
		var project string
		var hour int64
		var total, noBuildResult, noPresubmitResult int64

		err := r.Columns(&project, &hour, &total, &noBuildResult, &noPresubmitResult)
		if err != nil {
			return errors.Annotate(err, "read row").Err()
		}

		stats, ok := result[project]
		if !ok {
			stats = PresubmitJoinStatistics{
				// Add zero data for all hours.
				TotalBuildsByHour:             make([]int64, PresubmitJoinStatsHours),
				AwaitingBuildByHour:           make([]int64, PresubmitJoinStatsHours),
				AwaitingPresubmitResultByHour: make([]int64, PresubmitJoinStatsHours),
			}
		}
		stats.TotalBuildsByHour[hour] = total
		stats.AwaitingBuildByHour[hour] = noBuildResult
		stats.AwaitingPresubmitResultByHour[hour] = noPresubmitResult

		result[project] = stats
		return nil
	})
	if err != nil {
		return nil, errors.Annotate(err, "query presubmit join stats by project").Err()
	}
	return result, nil
}

func validateEntry(e *Entry) error {
	switch {
	case !config.ProjectRe.MatchString(e.Project):
		return errors.New("project must be valid")
	case e.BuildID == "":
		return errors.New("build ID must be specified")
	}
	if e.BuildResult != nil {
		if err := validateBuildResult(e.BuildResult); err != nil {
			return errors.Annotate(err, "build result").Err()
		}
	}
	if e.PresubmitResult != nil {
		if !e.IsPresubmit {
			return errors.New("presubmit result must not be set unless IsPresubmit is set")
		}
		if err := validatePresubmitResult(e.PresubmitResult); err != nil {
			return errors.Annotate(err, "presubmit result").Err()
		}
	}
	return nil
}

func validateBuildResult(r *ctlpb.BuildResult) error {
	switch {
	case r.Host == "":
		return errors.New("host must be specified")
	case r.Id == 0:
		return errors.New("id must be specified")
	case !r.CreationTime.IsValid():
		return errors.New("creation time must be specified")
	}
	return nil
}

func validatePresubmitResult(r *ctlpb.PresubmitResult) error {
	switch {
	case r.PresubmitRunId == nil:
		return errors.New("presubmit run ID must be specified")
	case r.PresubmitRunId.System != "luci-cv":
		// LUCI CV is currently the only supported system.
		return errors.New("presubmit run system must be 'luci-cv'")
	case r.PresubmitRunId.Id == "":
		return errors.New("presubmit run system-specific ID must be specified")
	case !r.CreationTime.IsValid():
		return errors.New("creation time must be specified and valid")
	}
	return nil
}
