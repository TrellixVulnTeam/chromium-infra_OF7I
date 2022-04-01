// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package testverdictingester

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"

	"go.chromium.org/luci/server/tq"

	"infra/appengine/weetbix/internal/tasks/taskspb"
)

const (
	taskClass = "test-verdict-ingestion"
	queue     = "test-verdict-ingestion"
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

func ingestTestVerdicts(ctx context.Context, task *taskspb.IngestTestVerdicts) error {
	// TODO(crbug.com/1266759): query ResultDB and collects the test results to
	// the TestVerdicts table.
	return nil
}
