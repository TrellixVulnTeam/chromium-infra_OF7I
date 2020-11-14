// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package tasks

import (
	"context"
	"fmt"

	gerritpb "go.chromium.org/luci/common/proto/gerrit"
	"go.chromium.org/luci/server/tq"
	"google.golang.org/protobuf/proto"

	"infra/appengine/rubber-stamper/tasks/taskspb"
)

func init() {
	tq.RegisterTaskClass(tq.TaskClass{
		ID:        "change-review-task",
		Prototype: (*taskspb.ChangeReviewTask)(nil),
		Queue:     "change-review-queue",
		Handler: func(ctx context.Context, payload proto.Message) error {
			// TODO: Handle change review task.
			return nil
		},
	})
}

// EnqueueChangeReviewTask enqueues a change review task.
func EnqueueChangeReviewTask(ctx context.Context, host string, cl *gerritpb.ChangeInfo) error {
	t := &taskspb.ChangeReviewTask{
		Host:     host,
		Number:   cl.Number,
		Revision: cl.CurrentRevision,
	}
	dedupKey := fmt.Sprintf("change(%s,%d,%s)", t.Host, t.Number, t.Revision)

	return tq.AddTask(ctx, &tq.Task{
		Payload:          t,
		Title:            dedupKey,
		DeduplicationKey: dedupKey,
	})
}
