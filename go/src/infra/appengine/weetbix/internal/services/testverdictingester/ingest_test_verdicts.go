// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package testverdictingester

import (
	"context"
	"errors"
	"fmt"
	"time"

	"golang.org/x/sync/errgroup"
	"google.golang.org/genproto/protobuf/field_mask"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	bbpb "go.chromium.org/luci/buildbucket/proto"
	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/logging"
	rdbpb "go.chromium.org/luci/resultdb/proto/v1"
	"go.chromium.org/luci/server/tq"

	"infra/appengine/weetbix/internal/buildbucket"
	"infra/appengine/weetbix/internal/resultdb"
	"infra/appengine/weetbix/internal/tasks/taskspb"
	"infra/appengine/weetbix/utils"
)

const (
	taskClass = "test-verdict-ingestion"
	queue     = "test-verdict-ingestion"

	// ingestionEarliest is the oldest data that may be ingested by Weetbix.
	// This is an offset relative to the current time, and should be kept
	// in sync with the data retention period in Spanner and BigQuery.
	ingestionEarliest = -90 * 24 * time.Hour

	// ingestionLatest is the newest data that may be ingested by Weetbix.
	// This is an offset relative to the current time. It is designed to
	// allow for clock drift.
	ingestionLatest = 24 * time.Hour

	// maxResultDBPages is the maximum number of pages of test verdicts to ingest
	// from ResultDB, per build. The page size is 1000 test verdicts.
	maxResultDBPages = int(^uint(0) >> 1) // set to max int
)

// RegisterTaskClass registers the task class for tq dispatcher.
func RegisterTaskClass() {
	tq.RegisterTaskClass(tq.TaskClass{
		ID:        taskClass,
		Prototype: &taskspb.IngestTestVerdicts{},
		Queue:     queue,
		Kind:      tq.Transactional,
		Handler: func(ctx context.Context, payload proto.Message) error {
			task := payload.(*taskspb.IngestTestVerdicts)
			return ingestTestVerdicts(ctx, task)
		},
	})
}

// Schedule enqueues a task to get all the test results from an invocation,
// group them into test verdicts, and save them to the TestVerdicts table.
func Schedule(ctx context.Context, task *taskspb.IngestTestVerdicts) error {
	return tq.AddTask(ctx, &tq.Task{
		Title:   fmt.Sprintf("%s-%d", task.Build.Host, task.Build.Id),
		Payload: task,
	})
}

func ingestTestVerdicts(ctx context.Context, payload *taskspb.IngestTestVerdicts) error {
	if err := validateRequest(ctx, payload); err != nil {
		return err
	}

	// Buildbucket build only has input.gerrit_changes, infra.resultdb, status populated.
	build, err := retrieveBuild(ctx, payload)
	code := status.Code(err)
	if code == codes.NotFound {
		// Build not found, end the task gracefully.
		logging.Warningf(ctx, "Buildbucket build %s/%d for project %s not found (or Weetbix does not have access to read it).",
			payload.Build.Host, payload.Build.Id, payload.Build.Project)
		return nil
	}
	if err != nil {
		return err
	}

	if build.Infra.GetResultdb().GetInvocation() == "" {
		// Build does not have a ResultDB invocation to ingest.
		logging.Debugf(ctx, "Skipping ingestion of build %s-%d because it has no ResultDB invocation.",
			payload.Build.Host, payload.Build.Id)
		return nil
	}

	rdbHost := build.Infra.Resultdb.Hostname
	invName := build.Infra.Resultdb.Invocation
	rc, err := resultdb.NewClient(ctx, rdbHost)
	if err != nil {
		return err
	}
	inv, err := rc.GetInvocation(ctx, invName)
	if err != nil {
		return err
	}
	project, _ := utils.SplitRealm(inv.Realm)
	if project == "" {
		return fmt.Errorf("invocation has invalid realm: %q", inv.Realm)
	}

	err = recordIngestedInvocation(ctx, payload, build, inv)
	if err != nil {
		return err
	}

	eg, ctx := errgroup.WithContext(ctx)

	// Query test variants from ResultDB.
	tvsC := make(chan []*rdbpb.TestVariant)
	eg.Go(func() error {
		defer close(tvsC)

		req := &rdbpb.QueryTestVariantsRequest{
			Invocations: []string{invName},
			PageSize:    10000,
		}
		return rc.QueryTestVariants(ctx, req, func(tvs []*rdbpb.TestVariant) error {
			tvsC <- tvs
			return nil
		}, maxResultDBPages)
	})

	// Record the test verdicts.
	eg.Go(func() error {
		return recordTestVerdicts(ctx, payload, build, inv, tvsC)
	})

	// If any transaction failed, the task will be retried and the tables will be
	// eventual-consistent.
	return eg.Wait()
}

func validateRequest(ctx context.Context, payload *taskspb.IngestTestVerdicts) error {
	if !payload.PartitionTime.IsValid() {
		return tq.Fatal.Apply(errors.New("partition time must be specified and valid"))
	}
	t := payload.PartitionTime.AsTime()
	now := clock.Now(ctx)
	if t.Before(now.Add(ingestionEarliest)) {
		return tq.Fatal.Apply(fmt.Errorf("partition time (%v) is too long ago", t))
	} else if t.After(now.Add(ingestionLatest)) {
		return tq.Fatal.Apply(fmt.Errorf("partition time (%v) is too far in the future", t))
	}
	if payload.Build == nil {
		return tq.Fatal.Apply(errors.New("build must be specified"))
	}
	return nil
}

func retrieveBuild(ctx context.Context, payload *taskspb.IngestTestVerdicts) (*bbpb.Build, error) {
	bbHost := payload.Build.Host
	id := payload.Build.Id
	bc, err := buildbucket.NewClient(ctx, bbHost)
	if err != nil {
		return nil, err
	}
	request := &bbpb.GetBuildRequest{
		Id: id,
		Mask: &bbpb.BuildMask{
			Fields: &field_mask.FieldMask{
				Paths: []string{"input.gerrit_changes", "infra.resultdb", "status"},
			},
		},
	}
	b, err := bc.GetBuild(ctx, request)
	switch {
	case err != nil:
		return nil, err
	}
	return b, nil
}
