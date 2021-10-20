// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package testvariantupdator

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/protobuf/proto"

	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/resultdb/pbutil"
	"go.chromium.org/luci/server/span"
	"go.chromium.org/luci/server/tq"
	_ "go.chromium.org/luci/server/tq/txn/spanner"

	"infra/appengine/weetbix/internal/analyzedtestvariants"
	spanutil "infra/appengine/weetbix/internal/span"
	"infra/appengine/weetbix/internal/tasks/taskspb"
	"infra/appengine/weetbix/internal/verdicts"
	pb "infra/appengine/weetbix/proto/v1"
)

const (
	taskClass = "update-test-variant"
	queue     = "update-test-variant"
)

// errShouldNotSchedule returned if the AnalyzedTestVariant spanner row
// does not have timestamp.
var errShouldNotSchedule = fmt.Errorf("should not schedule update task")

// errUnknownTask is returned if the task has a mismatched timestamp.
var errUnknownTask = fmt.Errorf("the task is unknown")

// RegisterTaskClass registers the task class for tq dispatcher.
func RegisterTaskClass() {
	tq.RegisterTaskClass(tq.TaskClass{
		ID:        taskClass,
		Prototype: &taskspb.UpdateTestVariant{},
		Queue:     queue,
		Kind:      tq.Transactional,
		Handler: func(ctx context.Context, payload proto.Message) error {
			task := payload.(*taskspb.UpdateTestVariant)
			tvKey := task.TestVariantKey
			_, err := checkTask(span.Single(ctx), task)
			switch {
			case err == errShouldNotSchedule:
				// Ignore the task.
				logging.Errorf(ctx, "test variant %s/%s/%s should not have any update task", tvKey.Realm, tvKey.TestId, tvKey.VariantHash)
				return nil
			case err == errUnknownTask:
				// Ignore the task.
				logging.Errorf(ctx, "unknown task found for test variant %s/%s/%s", tvKey.Realm, tvKey.TestId, tvKey.VariantHash)
				return nil
			case err != nil:
				return err
			}

			return updateTestVariant(ctx, task)
		},
	})
}

// Schedule enqueues a task to update an AnalyzedTestVariant row.
func Schedule(ctx context.Context, realm, testID, variantHash string, enqTime time.Time) {
	tq.MustAddTask(ctx, &tq.Task{
		Title: fmt.Sprintf("%s-%s-%s", realm, testID, variantHash),
		Payload: &taskspb.UpdateTestVariant{
			TestVariantKey: &taskspb.TestVariantKey{
				Realm:       realm,
				TestId:      testID,
				VariantHash: variantHash,
			},
			EnqueueTime: pbutil.MustTimestampProto(enqTime),
		},
		Delay: time.Hour,
	})
}

// checkTask checks if the task has the same timestamp as the one saved in the
// row.
// Task has a mismatched timestamp will be ignored.
func checkTask(ctx context.Context, task *taskspb.UpdateTestVariant) (pb.AnalyzedTestVariantStatus, error) {
	var status pb.AnalyzedTestVariantStatus
	var enqTime spanner.NullTime
	err := analyzedtestvariants.Read(ctx, spanner.KeySets(toSpannerKey(task.TestVariantKey)), func(atv *pb.AnalyzedTestVariant, t spanner.NullTime) error {
		status = atv.Status
		enqTime = t
		return nil
	})
	switch {
	case err != nil:
		return pb.AnalyzedTestVariantStatus_STATUS_UNSPECIFIED, err
	case enqTime.IsNull():
		return status, errShouldNotSchedule
	case enqTime.Time != pbutil.MustTimestamp(task.EnqueueTime):
		return status, errUnknownTask
	default:
		return status, nil
	}
}

func updateTestVariant(ctx context.Context, task *taskspb.UpdateTestVariant) error {
	status, err := verdicts.ComputeTestVariantStatusFromVerdicts(span.Single(ctx), task.TestVariantKey)
	if err != nil {
		return err
	}
	return updateTestVariantStatus(ctx, task, status)
}

// updateTestVariantStatus updates the Status and StatusUpdateTime of the
// AnalyzedTestVariants row if the provided status is different from the one
// in the row.
func updateTestVariantStatus(ctx context.Context, task *taskspb.UpdateTestVariant, newStatus pb.AnalyzedTestVariantStatus) error {
	tvKey := task.TestVariantKey
	_, err := span.ReadWriteTransaction(ctx, func(ctx context.Context) error {
		// Get the old status, and check the token once again.
		oldStatus, err := checkTask(ctx, task)
		if err != nil {
			return err
		}

		// Update the Spanner row.
		vals := map[string]interface{}{
			"Realm":       tvKey.Realm,
			"TestId":      tvKey.TestId,
			"VariantHash": tvKey.VariantHash,
		}
		now := clock.Now(ctx)

		if oldStatus == newStatus {
			if newStatus == pb.AnalyzedTestVariantStatus_CONSISTENTLY_EXPECTED || newStatus == pb.AnalyzedTestVariantStatus_NO_NEW_RESULTS {
				// This should never happen. But it doesn't have a huge negative impact,
				// so just log an error and return immediately.
				logging.Errorf(ctx, "UpdateTestVariant task runs for a test variant without any new unexpected failures: %s/%s/%s", tvKey.Realm, tvKey.TestId, tvKey.VariantHash)
				return nil
			}
			vals["NextUpdateTaskEnqueueTime"] = now
		} else {
			vals["Status"] = int64(newStatus)
			vals["StatusUpdateTime"] = spanner.CommitTimestamp
			if newStatus != pb.AnalyzedTestVariantStatus_CONSISTENTLY_EXPECTED && newStatus != pb.AnalyzedTestVariantStatus_NO_NEW_RESULTS {
				// Only schedule the next UpdateTestVariant task if the test variant
				// still has unexpected failures.
				vals["NextUpdateTaskEnqueueTime"] = now
			}
		}
		span.BufferWrite(ctx, spanutil.UpdateMap("AnalyzedTestVariants", vals))

		// Enqueue the next task.
		if _, ok := vals["NextUpdateTaskEnqueueTime"]; ok {
			Schedule(ctx, tvKey.Realm, tvKey.TestId, tvKey.VariantHash, now)
		}
		return nil
	})
	return err
}

func toSpannerKey(tvKey *taskspb.TestVariantKey) spanner.Key {
	return spanner.Key{tvKey.Realm, tvKey.TestId, tvKey.VariantHash}
}
