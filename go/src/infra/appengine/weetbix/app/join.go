// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"context"
	"fmt"

	"infra/appengine/weetbix/internal/ingestion/control"
	ctlpb "infra/appengine/weetbix/internal/ingestion/control/proto"
	"infra/appengine/weetbix/internal/services/resultingester"
	"infra/appengine/weetbix/internal/tasks/taskspb"
	pb "infra/appengine/weetbix/proto/v1"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/server/span"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
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
	f := func(ctx context.Context) error {
		saved = false
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
		createTaskIfNeeded(ctx, entry)
		return nil
	}
	if _, err := span.ReadWriteTransaction(ctx, f); err != nil {
		return err
	}
	if !saved {
		logging.Warningf(ctx, "build result for ingestion %q was dropped as one was already recorded", buildID)
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
	f := func(ctx context.Context) error {
		buildIDsSkipped = nil
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
			createTaskIfNeeded(ctx, entry)
		}
		return nil
	}
	if _, err := span.ReadWriteTransaction(ctx, f); err != nil {
		return err
	}
	if len(buildIDsSkipped) > 0 {
		logging.Warningf(ctx, "presubmit result for builds %v were dropped as one was already recorded", buildIDsSkipped)
	}
	return nil
}

// createTaskIfNeeded creates the task if all necessary data for the ingestion is available.
func createTaskIfNeeded(ctx context.Context, e *control.Entry) {
	if e.BuildResult == nil || (e.IsPresubmit && e.PresubmitResult == nil) {
		return
	}

	var task *taskspb.IngestTestResults
	if e.IsPresubmit {
		task = &taskspb.IngestTestResults{
			Build: &taskspb.Build{
				Host: e.BuildResult.Host,
				Id:   e.BuildResult.Id,
			},
			PartitionTime:         proto.Clone(e.PresubmitResult.CreationTime).(*timestamppb.Timestamp),
			PresubmitRunId:        proto.Clone(e.PresubmitResult.PresubmitRunId).(*pb.PresubmitRunId),
			PresubmitRunSucceeded: e.PresubmitResult.PresubmitRunSucceeded,
		}
	} else {
		task = &taskspb.IngestTestResults{
			Build: &taskspb.Build{
				Host: e.BuildResult.Host,
				Id:   e.BuildResult.Id,
			},
			PartitionTime: proto.Clone(e.BuildResult.CreationTime).(*timestamppb.Timestamp),
		}
	}

	resultingester.Schedule(ctx, task)
}
