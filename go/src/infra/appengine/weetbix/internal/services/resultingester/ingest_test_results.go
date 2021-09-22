// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package resultingester

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"

	"go.chromium.org/luci/common/logging"
	cvpb "go.chromium.org/luci/cv/api/v0"
	"go.chromium.org/luci/server/tq"

	"infra/appengine/weetbix/internal/tasks/taskspb"
)

const (
	resultIngestionTaskClass = "result-ingestion"
	resultIngestionQueue     = "result-ingestion"
)

// RegisterResultIngestionTasksClass registers the task class for tq dispatcher.
func RegisterResultIngestionTasksClass() {
	tq.RegisterTaskClass(tq.TaskClass{
		ID:        resultIngestionTaskClass,
		Prototype: &taskspb.IngestTestResults{},
		Queue:     resultIngestionQueue,
		Kind:      tq.NonTransactional,
		Handler: func(ctx context.Context, payload proto.Message) error {
			task := payload.(*taskspb.IngestTestResults)
			return ingestTestResults(ctx, task)
		},
	})
}

// Schedule enqueues a task to ingest test results from a build.
func Schedule(ctx context.Context, cvRun *cvpb.Run, build *taskspb.Build) error {
	// Note that currently we don't need to deduplicate tasks, because for
	// Chromium use case Weetbix only ingest test results of the try builds that
	// contribute to CL submission, so each build should be processed only once.
	// This may not be true in ChromeOS use case where Weetbix ingests test
	// of all try builds.
	return tq.AddTask(ctx, &tq.Task{
		Title: fmt.Sprintf("%s-%d", build.Host, build.Id),
		Payload: &taskspb.IngestTestResults{
			CvRun: cvRun,
			Build: build,
		},
	})
}

func ingestTestResults(ctx context.Context, payload *taskspb.IngestTestResults) error {
	logging.Debugf(ctx, "ResultIngestion task for build %d", payload.Build.Id)
	return nil
}
