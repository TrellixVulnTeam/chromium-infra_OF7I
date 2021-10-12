// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package resultcollector

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"

	"go.chromium.org/luci/common/logging"
	rdbpb "go.chromium.org/luci/resultdb/proto/v1"
	"go.chromium.org/luci/server/tq"

	"infra/appengine/weetbix/internal/tasks/taskspb"
)

const (
	resultCollectionTaskClass = "result-collection"
	resultCollectionQueue     = "result-collection"
)

// RegisterTasksClass registers the task class for tq dispatcher.
func RegisterTasksClass() {
	tq.RegisterTaskClass(tq.TaskClass{
		ID:        resultCollectionTaskClass,
		Prototype: &taskspb.CollectTestResults{},
		Queue:     resultCollectionQueue,
		Kind:      tq.NonTransactional,
		Handler: func(ctx context.Context, payload proto.Message) error {
			task := payload.(*taskspb.CollectTestResults)
			return collectTestResults(ctx, task)
		},
	})
}

// Schedule enqueues a task to get test results of interesting test variants
// from an invocation.
//
// Interesting test variants are the analyzed test variants with any unexpected
// results.
func Schedule(ctx context.Context, inv *rdbpb.Invocation, rdbHost, builder string, isPreSubmit, contributedToCLSubmission bool) error {
	return tq.AddTask(ctx, &tq.Task{
		Title: fmt.Sprintf("%s", inv.Name),
		Payload: &taskspb.CollectTestResults{
			Resultdb: &taskspb.ResultDB{
				Invocation: inv,
				Host:       rdbHost,
			},
			Builder:                   builder,
			IsPreSubmit:               isPreSubmit,
			ContributedToClSubmission: contributedToCLSubmission,
		},
	})
}

func collectTestResults(ctx context.Context, task *taskspb.CollectTestResults) error {
	logging.Debugf(ctx, "ResultCollection task for invocation %s", task.Resultdb.Invocation)
	return nil
}
