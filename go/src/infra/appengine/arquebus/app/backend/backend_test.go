// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package backend

import (
	"testing"
	"time"

	"infra/appengine/arquebus/app/backend/model"
	monorail "infra/monorailv2/api/api_proto"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/gae/service/datastore"
)

func TestBackend(t *testing.T) {
	t.Parallel()
	assignerID := "test-assigner"

	Convey("scheduleAssignerTaskHandler", t, func() {
		c := createTestContextWithTQ()

		// create a sample assigner with tasks.
		createAssigner(c, assignerID)
		tasks := triggerScheduleTaskHandler(c, assignerID)
		So(tasks, ShouldNotBeNil)

		Convey("works", func() {
			for _, task := range tasks {
				So(task.Status, ShouldEqual, model.TaskStatus_Scheduled)
			}
		})

		Convey("doesn't schedule new tasks for a drained assigner.", func() {
			// TODO(crbug/967519): implement me.
		})
	})

	Convey("runAssignerTaskHandler", t, func() {
		c := createTestContextWithTQ()
		assigner := createAssigner(c, assignerID)
		tasks := triggerScheduleTaskHandler(c, assignerID)
		So(tasks, ShouldNotBeNil)

		Convey("works", func() {
			mockGetAndListIssues(
				c, &monorail.Issue{ProjectName: "test", LocalId: 123},
			)

			for _, task := range tasks {
				So(task.Status, ShouldEqual, model.TaskStatus_Scheduled)
				task = triggerRunTaskHandler(c, assignerID, task.ID)

				So(task.Status, ShouldEqual, model.TaskStatus_Succeeded)
				So(task.Started.IsZero(), ShouldBeFalse)
				So(task.Ended.IsZero(), ShouldBeFalse)
			}
		})

		Convey("cancelling stale tasks.", func() {
			// make one stale schedule
			task := tasks[0]
			task.ExpectedStart = task.ExpectedStart.Add(-10 * time.Hour)
			So(task.ExpectedStart.Before(clock.Now(c).UTC()), ShouldBeTrue)
			So(task.Status, ShouldEqual, model.TaskStatus_Scheduled)
			So(datastore.Put(c, task), ShouldBeNil)

			// It should be marked as cancelled after runTaskHandler().
			processedTask := triggerRunTaskHandler(c, assignerID, task.ID)
			So(processedTask.Status, ShouldEqual, model.TaskStatus_Cancelled)
			So(processedTask.Started.IsZero(), ShouldBeFalse)
			So(processedTask.Ended.IsZero(), ShouldBeFalse)
		})

		Convey("task status is kept as original, if not scheduled.", func() {
			// make one with an invalid status. TaskStatus_Scheduled is the
			// the only status valid for runTaskHandler()
			task := tasks[0]
			task.Status = model.TaskStatus_Failed
			task.Started = time.Date(2000, 1, 1, 2, 3, 4, 0, time.UTC)
			task.Ended = task.Started.AddDate(0, 1, 2)
			So(datastore.Put(c, task), ShouldBeNil)

			// The task should stay the same after runTaskHandler().
			processedTask := triggerRunTaskHandler(c, assignerID, task.ID)
			So(processedTask.Status, ShouldEqual, task.Status)
			So(processedTask.Started, ShouldEqual, task.Started)
			So(processedTask.Ended, ShouldEqual, task.Ended)
		})

		Convey("skips assigners with stale format", func() {
			assigner.FormatVersion = 0
			datastore.Put(c, assigner)

			for _, task := range tasks {
				task = triggerRunTaskHandler(c, assignerID, task.ID)
				So(task.Status, ShouldEqual, model.TaskStatus_Cancelled)
			}
		})

		Convey("cancelling tasks, if the assigner has been drained.", func() {
			// TODO(crbug/967519): implement me.
		})
	})

	Convey("RemoveNoopTasks", t, func() {
		c := createTestContextWithTQ()
		assigner := createAssigner(c, assignerID)
		addTasks := func(n int, noop bool) error {
			tks := make([]*model.Task, n)
			for i := 0; i < n; i++ {
				tks[i] = &model.Task{
					AssignerKey:    model.GenAssignerKey(c, assigner),
					Status:         model.TaskStatus_Succeeded,
					WasNoopSuccess: noop,
				}
			}
			return datastore.Put(c, tks)
		}
		getTasks := func(n int32) []*model.Task {
			tks, err := model.GetTasks(c, assigner, n, true)
			So(err, ShouldBeNil)
			return tks
		}

		Convey("With noop tasks", func() {
			// nTask == 0
			So(RemoveNoopTasks(c, assigner, 5), ShouldBeNil)
			So(len(getTasks(5)), ShouldEqual, 0)

			// nTask < nDel
			addTasks(3, true)
			So(RemoveNoopTasks(c, assigner, 5), ShouldBeNil)
			So(len(getTasks(5)), ShouldEqual, 0)

			// nTask > nDel
			addTasks(7, true)
			So(RemoveNoopTasks(c, assigner, 5), ShouldBeNil)
			So(getTasks(5), ShouldHaveLength, 2)
		})

		Convey("w/o noop tasks", func() {
			addTasks(7, false)
			So(RemoveNoopTasks(c, assigner, 5), ShouldBeNil)
			So(getTasks(7), ShouldHaveLength, 7)
		})
	})
}
