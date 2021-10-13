// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package testvariantupdator

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/proto"

	"go.chromium.org/luci/server/tq"

	"infra/appengine/weetbix/internal/tasks/taskspb"
	"infra/appengine/weetbix/internal/verdicts"
	pb "infra/appengine/weetbix/proto/v1"
)

const (
	taskClass = "update-test-variant"
	queue     = "update-test-variant"
)

// RegisterTaskClass registers the task class for tq dispatcher.
func RegisterTaskClass() {
	tq.RegisterTaskClass(tq.TaskClass{
		ID:        taskClass,
		Prototype: &taskspb.UpdateTestVariant{},
		Queue:     queue,
		Kind:      tq.NonTransactional,
		Handler: func(ctx context.Context, payload proto.Message) error {
			task := payload.(*taskspb.UpdateTestVariant)
			return updateTestVariant(ctx, task)
		},
	})
}

// Schedule enqueues a task to update an AnalyzedTestVariant row.
func Schedule(ctx context.Context, realm, testID, variantHash string) error {
	return tq.AddTask(ctx, &tq.Task{
		Title: fmt.Sprintf("%s-%s-%s", realm, testID, variantHash),
		Payload: &taskspb.UpdateTestVariant{
			TestVariantKey: &taskspb.TestVariantKey{
				Realm:       realm,
				TestId:      testID,
				VariantHash: variantHash,
			},
		},
		Delay: time.Hour,
	})
}

func updateTestVariant(ctx context.Context, task *taskspb.UpdateTestVariant) error {
	status, err := verdicts.ComputeTestVariantStatusFromVerdicts(ctx, task.TestVariantKey)
	if err != nil {
		return err
	}
	return updateTestVariantStatus(ctx, task.TestVariantKey, status)
}

// updateTestVariantStatus updates the Status and StatusUpdateTime of the
// AnalyzedTestVariants row if the provided status is different from the one
// in the row.
func updateTestVariantStatus(ctx context.Context, tvKey *taskspb.TestVariantKey, status pb.AnalyzedTestVariantStatus) error {
	return fmt.Errorf("not implemented")
}
