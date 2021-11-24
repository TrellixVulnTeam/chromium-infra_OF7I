// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package tq

import (
	"context"
	"net/url"

	"go.chromium.org/luci/gae/service/taskqueue"
)

// Task is a taskqueue task to be executed.
type Task = taskqueue.Task

// Testable is a testable interface for the taskqueue library.
type Testable = taskqueue.Testable

// QueueData is support data for task queues in tests.
type QueueData = taskqueue.QueueData

// NewPOSTTask creates a task that will post to a path with a given payload in its URLs.
func NewPOSTTask(path string, params url.Values) *Task {
	return taskqueue.NewPOSTTask(path, params)
}

// Add a task to the specified task queue.
func Add(ctx context.Context, queueName string, tasks ...*Task) error {
	return taskqueue.Add(ctx, queueName, tasks...)
}

// GetTestable gets a testable version of the current task queue.
func GetTestable(ctx context.Context) Testable {
	return taskqueue.GetTestable(ctx)
}
