// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"context"
	"fmt"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/tsmon/field"
	"go.chromium.org/luci/common/tsmon/metric"
	"go.chromium.org/luci/server/span"
	"google.golang.org/protobuf/proto"

	"infra/appengine/weetbix/internal/ingestion/control"
	ctlpb "infra/appengine/weetbix/internal/ingestion/control/proto"
	"infra/appengine/weetbix/internal/services/resultingester"
	"infra/appengine/weetbix/internal/tasks/taskspb"
)

// For presubmit builds, proceeding to ingestion is not unconditional:
// we must wait for the build to be notified over both a CV and Buildbucket.
// We define the following metrics to monitor the performance of that join.
var (
	cvPresubmitBuildCounter = metric.NewCounter(
		"weetbix/ingestion/join/cv_presubmit_builds_input",
		"The number of unique presubmit builds for which CV Run Completion was received.",
		nil,
		// The LUCI Project.
		field.String("project"))

	bbPresubmitBuildCounter = metric.NewCounter(
		"weetbix/ingestion/join/bb_presubmit_builds_input",
		"The number of unique presubmit build for which buildbucket build completion was received.",
		nil,
		// The LUCI Project.
		field.String("project"))

	outputPresubmitBuildCounter = metric.NewCounter(
		"weetbix/ingestion/join/presubmit_builds_output",
		"The number of presubmit builds which were successfully joined and for which ingestion was queued.",
		nil,
		// The LUCI Project.
		field.String("project"))
)

// For CI builds, no actual join needs to occur. So it is sufficient to
// monitor total flow.
var (
	outputCIBuildCounter = metric.NewCounter(
		"weetbix/ingestion/join/ci_builds_output",
		"The number of CI builds for which ingestion was queued.",
		nil,
		// The LUCI Project.
		field.String("project"))
)

// JoinBuildResult sets the build result for the given build.
//
// An ingestion task is created if all required data for the
// ingestion is available (for builds part of a presubmit run,
// this is only after the presubmit result has joined, for
// all other builds, this is straight away).
//
// If the build result has already been provided for a build,
// this method has no effect.
func JoinBuildResult(ctx context.Context, project, buildID string, isPresubmit bool, br *ctlpb.BuildResult) error {
	if br == nil {
		return errors.New("build result must be specified")
	}
	var saved bool
	var joined bool
	f := func(ctx context.Context) error {
		saved = false
		joined = false
		entries, err := control.Read(ctx, project, []string{buildID})
		if err != nil {
			return err
		}
		entry := entries[0]
		// Record does not exist.
		if entry == nil {
			entry = &control.Entry{
				Project:     project,
				BuildID:     buildID,
				IsPresubmit: isPresubmit,
			}
		}
		if entry.IsPresubmit != isPresubmit {
			return fmt.Errorf("disagreement about whether ingestion is presubmit run (got %v, want %v)", isPresubmit, entry.IsPresubmit)
		}
		if entry.BuildResult != nil {
			// Build result already recorded. Do not modify and do not
			// create a duplicate ingestion.
			return nil
		}
		entry.BuildResult = br
		if err := control.InsertOrUpdate(ctx, entry); err != nil {
			return err
		}
		saved = true
		joined = createTaskIfNeeded(ctx, entry)
		return nil
	}
	if _, err := span.ReadWriteTransaction(ctx, f); err != nil {
		return err
	}
	if !saved {
		logging.Warningf(ctx, "build result for ingestion %q was dropped as one was already recorded", buildID)
	}

	// Export metrics.
	if saved && isPresubmit {
		bbPresubmitBuildCounter.Add(ctx, 1, project)
	}
	if joined {
		if isPresubmit {
			outputPresubmitBuildCounter.Add(ctx, 1, project)
		} else {
			outputCIBuildCounter.Add(ctx, 1, project)
		}
	}
	return nil
}

// JoinPresubmitResult sets the presubmit result for the given builds.
//
// Ingestion task(s) are created for builds where all required data
// is available (i.e. after the build result has also joined).
//
// If the presubmit result has already been provided for a build,
// this method has no effect.
func JoinPresubmitResult(ctx context.Context, project string, buildIDs []string, pr *ctlpb.PresubmitResult) error {
	if pr == nil {
		return errors.New("presubmit result must be specified")
	}
	var buildIDsSkipped []string
	var buildsOutput int64
	f := func(ctx context.Context) error {
		buildIDsSkipped = nil
		buildsOutput = 0
		entries, err := control.Read(ctx, project, buildIDs)
		if err != nil {
			return err
		}
		for i, entry := range entries {
			buildID := buildIDs[i]
			if entry == nil {
				entry = &control.Entry{
					Project:     project,
					BuildID:     buildID,
					IsPresubmit: true,
				}
			}
			if !entry.IsPresubmit {
				return fmt.Errorf("attempt to save presubmit result on build (%q) not marked as presubmit", buildID)
			}
			if entry.PresubmitResult != nil {
				// Presubmit result already recorded. Do not modify and do not
				// create a duplicate ingestion.
				buildIDsSkipped = append(buildIDsSkipped, buildID)
				return nil
			}
			entry.PresubmitResult = pr
			if err := control.InsertOrUpdate(ctx, entry); err != nil {
				return err
			}
			created := createTaskIfNeeded(ctx, entry)
			if created {
				buildsOutput++
			}
		}
		return nil
	}
	if _, err := span.ReadWriteTransaction(ctx, f); err != nil {
		return err
	}
	if len(buildIDsSkipped) > 0 {
		logging.Warningf(ctx, "presubmit result for builds %v were dropped as one was already recorded", buildIDsSkipped)
	}

	// Export metrics.
	cvPresubmitBuildCounter.Add(ctx, int64(len(buildIDs)-len(buildIDsSkipped)), project)
	if buildsOutput > 0 {
		outputPresubmitBuildCounter.Add(ctx, buildsOutput, project)
	}
	return nil
}

// createTaskIfNeeded creates the task if all necessary data for the ingestion is available.
func createTaskIfNeeded(ctx context.Context, e *control.Entry) bool {
	if e.BuildResult == nil || (e.IsPresubmit && e.PresubmitResult == nil) {
		return false
	}

	var task *taskspb.IngestTestResults
	if e.IsPresubmit {
		task = &taskspb.IngestTestResults{
			PartitionTime: e.PresubmitResult.CreationTime,
			Build:         e.BuildResult,
			PresubmitRun:  e.PresubmitResult,
		}
	} else {
		task = &taskspb.IngestTestResults{
			PartitionTime: e.BuildResult.CreationTime,
			Build:         e.BuildResult,
		}
	}

	// Copy the task to avoid aliasing issues if the caller ever
	// decides the modify e.PresubmitResult or e.BuildResult
	// after we return.
	task = proto.Clone(task).(*taskspb.IngestTestResults)

	resultingester.Schedule(ctx, task)
	return true
}
