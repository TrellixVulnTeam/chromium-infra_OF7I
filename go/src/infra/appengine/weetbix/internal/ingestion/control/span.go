// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package control

import (
	"context"
	"fmt"
	"time"

	"infra/appengine/weetbix/internal/config"
	ctlpb "infra/appengine/weetbix/internal/ingestion/control/proto"
	spanutil "infra/appengine/weetbix/internal/span"

	"cloud.google.com/go/spanner"
	"github.com/golang/protobuf/proto"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/server/span"
)

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
	}
	entryByBuildID := make(map[string]*Entry)
	rows := span.Read(ctx, "IngestionControl", spanner.KeySetFromKeys(keys...), cols)
	f := func(r *spanner.Row) error {
		var buildID string
		var buildResultBytes []byte
		var isPresubmit bool
		var presubmitResultBytes []byte
		var lastUpdated time.Time

		err := r.Columns(&buildID,
			&buildResultBytes,
			&isPresubmit,
			&presubmitResultBytes,
			&lastUpdated)
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
			Project:         project,
			BuildID:         buildID,
			BuildResult:     buildResult,
			IsPresubmit:     isPresubmit,
			PresubmitResult: presubmitResult,
			LastUpdated:     lastUpdated,
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

// InsertOrUpdate creates or updates an ingestion control record to
// match the specified details. To avoid clobbering an existing record,
// this should be performed in the same transaction as a Read() of
// the record.
func InsertOrUpdate(ctx context.Context, e *Entry) error {
	if err := validateEntry(e); err != nil {
		return err
	}
	m := spanutil.InsertOrUpdateMap("IngestionControl", map[string]interface{}{
		"Project":         e.Project,
		"BuildId":         e.BuildID,
		"BuildResult":     e.BuildResult,
		"IsPresubmit":     e.IsPresubmit,
		"PresubmitResult": e.PresubmitResult,
		"LastUpdated":     spanner.CommitTimestamp,
	})
	span.BufferWrite(ctx, m)
	return nil
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
