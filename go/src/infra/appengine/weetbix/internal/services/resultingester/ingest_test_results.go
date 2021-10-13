// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package resultingester

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"

	bbpb "go.chromium.org/luci/buildbucket/proto"
	"go.chromium.org/luci/common/errors"
	cvpb "go.chromium.org/luci/cv/api/v0"
	"go.chromium.org/luci/server/tq"

	"infra/appengine/weetbix/internal/buildbucket"
	"infra/appengine/weetbix/internal/resultdb"
	"infra/appengine/weetbix/internal/tasks/taskspb"
)

const (
	resultIngestionTaskClass = "result-ingestion"
	resultIngestionQueue     = "result-ingestion"
)

// RegisterTasksClass registers the task class for tq dispatcher.
func RegisterTasksClass() {
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
	b, err := getBuilderAndResultDBInfo(ctx, payload)
	if err != nil {
		return err
	}

	rdbHost := b.Infra.Resultdb.Hostname
	invName := b.Infra.Resultdb.Invocation
	builder := b.Builder.Builder
	rc, err := resultdb.NewClient(ctx, rdbHost)
	if err != nil {
		return err
	}
	inv, err := rc.GetInvocation(ctx, invName)
	if err != nil {
		return err
	}
	tvs, err := rc.QueryTestVariants(ctx, invName)
	if err != nil {
		return err
	}

	if err = createOrUpdateAnalyzedTestVariants(ctx, inv.Realm, builder, tvs); err != nil {
		return err
	}

	return nil
}

func getBuilderAndResultDBInfo(ctx context.Context, payload *taskspb.IngestTestResults) (*bbpb.Build, error) {
	bbHost := payload.Build.Host
	bId := payload.Build.Id
	bc, err := buildbucket.NewClient(ctx, bbHost)
	if err != nil {
		return nil, err
	}
	b, err := bc.GetBuildWithBuilderAndRDBInfo(ctx, bId)
	switch {
	case err != nil:
		return nil, err
	case b.GetInfra().GetResultdb() == nil || b.Infra.Resultdb.GetInvocation() == "":
		return nil, errors.Reason("build %s-%d not have ResultDB invocation", bbHost, bId).Err()
	}
	return b, nil
}
