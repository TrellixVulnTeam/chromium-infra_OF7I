// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package reclustering

import (
	"context"

	"google.golang.org/protobuf/proto"

	"go.chromium.org/luci/server/tq"

	"infra/appengine/weetbix/internal/tasks/taskspb"
)

const (
	taskClass = "reclustering"
	queue     = "reclustering"
)

// RegisterTaskClass registers the task class with the Task Queue dispatcher.
func RegisterTaskClass() {
	tq.RegisterTaskClass(tq.TaskClass{
		ID:        taskClass,
		Prototype: &taskspb.ReclusterChunks{},
		Queue:     queue,
		Kind:      tq.NonTransactional,
		Handler: func(ctx context.Context, payload proto.Message) error {
			task := payload.(*taskspb.ReclusterChunks)
			return reclusterTestResults(ctx, task)
		},
	})
}

// Schedule enqueues a task to recluster a range of chunks in a LUCI
// Project.
func Schedule(ctx context.Context, title string, task *taskspb.ReclusterChunks) error {
	return tq.AddTask(ctx, &tq.Task{
		Title: title,
		// Copy the task to avoid the caller retaining an alias to
		// the task proto passed to tq.AddTask.
		Payload: proto.Clone(task).(*taskspb.ReclusterChunks),
	})
}

func reclusterTestResults(ctx context.Context, task *taskspb.ReclusterChunks) error {
	return nil
}
