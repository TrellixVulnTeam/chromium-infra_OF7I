// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package tasks

import (
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	gerritpb "go.chromium.org/luci/common/proto/gerrit"
	"go.chromium.org/luci/gae/impl/memory"
	"go.chromium.org/luci/server/tq"
	"go.chromium.org/luci/server/tq/tqtesting"

	"infra/appengine/rubber-stamper/tasks/taskspb"
)

func TestQueue(t *testing.T) {
	Convey("Chain works", t, func() {
		ctx, sched := tq.TestingContext(context.Background(), nil)
		ctx = memory.Use(ctx)

		var succeeded tqtesting.TaskList

		sched.TaskSucceeded = tqtesting.TasksCollector(&succeeded)
		sched.TaskFailed = func(ctx context.Context, task *tqtesting.Task) { panic("should not fail") }

		Convey("Test deduplication", func() {
			host := "host"
			cls := []*gerritpb.ChangeInfo{
				{
					Number:          12345,
					CurrentRevision: "123abc",
				},
				{
					Number:          12345,
					CurrentRevision: "456789",
				},
			}
			So(EnqueueChangeReviewTask(ctx, host, cls[0]), ShouldBeNil)
			So(EnqueueChangeReviewTask(ctx, host, cls[0]), ShouldBeNil)
			So(EnqueueChangeReviewTask(ctx, host, cls[1]), ShouldBeNil)
			So(EnqueueChangeReviewTask(ctx, host, cls[1]), ShouldBeNil)
			sched.Run(ctx, tqtesting.StopWhenDrained())
			So(len(succeeded.Payloads()), ShouldEqual, 2)
			So(succeeded.Payloads(), ShouldContain, &taskspb.ChangeReviewTask{
				Host:     "host",
				Number:   12345,
				Revision: "123abc",
			})
			So(succeeded.Payloads(), ShouldContain, &taskspb.ChangeReviewTask{
				Host:     "host",
				Number:   12345,
				Revision: "456789",
			})
		})
	})
}
