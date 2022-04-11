// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"context"
	"crypto/sha256"
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
	"infra/appengine/weetbix/internal/services/testverdictingester"
	"infra/appengine/weetbix/internal/tasks/taskspb"
)

// For presubmit builds, proceeding to ingestion is conditional:
// we must wait for the both the CV run and Buildbucket build to complete.
// We define the following metrics to monitor the performance of that join.
var (
	cvPresubmitBuildInputCounter = metric.NewCounter(
		"weetbix/ingestion/join/cv_presubmit_builds_input",
		"The number of unique presubmit builds for which CV Run Completion was received."+
			" Broken down by project of the CV run.",
		nil,
		// The LUCI Project of the CV run.
		field.String("project"))

	cvPresubmitBuildOutputCounter = metric.NewCounter(
		"weetbix/ingestion/join/cv_presubmit_builds_output",
		"The number of presubmit builds which were successfully joined and for which ingestion was queued."+
			" Broken down by project of the CV run.",
		nil,
		// The LUCI Project of the CV run.
		field.String("project"))

	bbPresubmitBuildInputCounter = metric.NewCounter(
		"weetbix/ingestion/join/bb_presubmit_builds_input",
		"The number of unique presubmit build for which buildbucket build completion was received."+
			" Broken down by project of the buildbucket build.",
		nil,
		// The LUCI Project of the buildbucket run.
		field.String("project"))

	bbPresubmitBuildOutputCounter = metric.NewCounter(
		"weetbix/ingestion/join/bb_presubmit_builds_output",
		"The number of presubmit builds which were successfully joined and for which ingestion was queued."+
			" Broken down by project of the buildbucket build.",
		nil,
		// The LUCI Project of the buildbucket run.
		field.String("project"))
)

// For CI builds, no actual join needs to occur. So it is sufficient to
// monitor only the output flow (same as input flow).
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
func JoinBuildResult(ctx context.Context, buildID, buildProject string, isPresubmit bool, br *ctlpb.BuildResult) error {
	if br == nil {
		return errors.New("build result must be specified")
	}
	var saved bool
	var taskCreated bool
	var cvProject string
	f := func(ctx context.Context) error {
		// Clear variables to ensure nothing from a previous (failed)
		// try of this transaction leaks out to the outer context.
		saved = false
		taskCreated = false
		cvProject = ""

		entries, err := control.Read(ctx, []string{buildID})
		if err != nil {
			return err
		}
		entry := entries[0]
		// Record does not exist.
		if entry == nil {
			entry = &control.Entry{
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
		entry.BuildProject = buildProject
		entry.BuildResult = br
		if control.SetBuildResult(ctx, entry); err != nil {
			return err
		}
		saved = true
		taskCreated = createTasksIfNeeded(ctx, entry)

		// Will only populated if IsPresubmit is not empty.
		cvProject = entry.PresubmitProject
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
		bbPresubmitBuildInputCounter.Add(ctx, 1, buildProject)
	}
	if taskCreated {
		if isPresubmit {
			bbPresubmitBuildOutputCounter.Add(ctx, 1, buildProject)
			cvPresubmitBuildOutputCounter.Add(ctx, 1, cvProject)
		} else {
			outputCIBuildCounter.Add(ctx, 1, buildProject)
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
func JoinPresubmitResult(ctx context.Context, presubmitResultByBuildID map[string]*ctlpb.PresubmitResult, presubmitProject string) error {
	for id, result := range presubmitResultByBuildID {
		if result == nil {
			return fmt.Errorf("presubmit result for build %v must be specified", id)
		}
	}

	var buildIDsSkipped []string
	var buildsOutputByBuildProject map[string]int64
	f := func(ctx context.Context) error {
		// Clear variables to ensure nothing from a previous (failed)
		// try of this transaction leaks out to the outer context.
		buildIDsSkipped = nil
		buildsOutputByBuildProject = make(map[string]int64)

		var buildIDs []string
		for id := range presubmitResultByBuildID {
			buildIDs = append(buildIDs, id)
		}

		entries, err := control.Read(ctx, buildIDs)
		if err != nil {
			return err
		}
		for i, entry := range entries {
			buildID := buildIDs[i]
			if entry == nil {
				entry = &control.Entry{
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
			entry.PresubmitProject = presubmitProject
			entry.PresubmitResult = presubmitResultByBuildID[buildID]
			if err := control.SetPresubmitResult(ctx, entry); err != nil {
				return err
			}
			created := createTasksIfNeeded(ctx, entry)
			if created {
				buildsOutputByBuildProject[entry.BuildProject]++
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
	cvPresubmitBuildInputCounter.Add(ctx, int64(len(presubmitResultByBuildID)-len(buildIDsSkipped)), presubmitProject)
	for buildProject, count := range buildsOutputByBuildProject {
		bbPresubmitBuildOutputCounter.Add(ctx, count, buildProject)
		cvPresubmitBuildOutputCounter.Add(ctx, count, presubmitProject)
	}
	return nil
}

// createTaskIfNeeded creates a test-result-ingestion task if all necessary
// data for the ingestion is available. It also has a 1/256 chance to create a
// test-verdict-ingestion task.
// Returns true if the test-result-ingestion task is created.
func createTasksIfNeeded(ctx context.Context, e *control.Entry) (itrTaskCreated bool) {
	if e.BuildResult == nil || (e.IsPresubmit && e.PresubmitResult == nil) {
		return false
	}

	var itrTask *taskspb.IngestTestResults
	if e.IsPresubmit {
		itrTask = &taskspb.IngestTestResults{
			PartitionTime: e.PresubmitResult.CreationTime,
			Build:         e.BuildResult,
			PresubmitRun:  e.PresubmitResult,
		}
	} else {
		itrTask = &taskspb.IngestTestResults{
			PartitionTime: e.BuildResult.CreationTime,
			Build:         e.BuildResult,
		}
	}

	// Copy the task to avoid aliasing issues if the caller ever
	// decides the modify e.PresubmitResult or e.BuildResult
	// after we return.
	itrTask = proto.Clone(itrTask).(*taskspb.IngestTestResults)
	resultingester.Schedule(ctx, itrTask)

	// Only ingest 1/256 test verdicts to limit the amount the verdicts we ingest
	// during development phase.
	if sha256.Sum256([]byte(e.BuildID))[0] != 0 {
		return true
	}

	itvTask := &taskspb.IngestTestVerdicts{
		PartitionTime: itrTask.PartitionTime,
		Build:         itrTask.Build,
		PresubmitRun:  itrTask.PresubmitRun,
	}
	testverdictingester.Schedule(ctx, itvTask)
	return true
}
