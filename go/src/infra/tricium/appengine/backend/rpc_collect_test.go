// Copyright 2016 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	tq "go.chromium.org/luci/gae/service/taskqueue"

	admin "infra/tricium/api/admin/v1"
	tricium "infra/tricium/api/v1"
	"infra/tricium/appengine/common"
	"infra/tricium/appengine/common/triciumtest"
)

// Mock task server API that returns a canned task result.
type mockTaskServer struct {
	State common.ResultState
}

func (mockTaskServer) Trigger(c context.Context, params *common.TriggerParameters) (*common.TriggerResult, error) {
	return &common.TriggerResult{}, nil
}
func (m mockTaskServer) Collect(c context.Context, params *common.CollectParameters) (*common.CollectResult, error) {
	return &common.CollectResult{
		State: m.State,
	}, nil
}

func TestCollectRequest(t *testing.T) {
	Convey("Test Environment", t, func() {
		ctx := triciumtest.Context()
		runID := int64(123456789)

		workflowProvider := &mockWorkflowProvider{
			Workflow: &admin.Workflow{
				Workers: []*admin.Worker{
					{
						Name:     "Hello",
						Needs:    tricium.Data_GIT_FILE_DETAILS,
						Provides: tricium.Data_RESULTS,
						Impl:     &admin.Worker_Recipe{},
					},
				},
			},
		}

		Convey("Driver collect request for worker without successors", func() {
			err := collect(ctx, &admin.CollectRequest{
				RunId:  runID,
				Worker: "Hello",
			}, workflowProvider, common.MockTaskServerAPI)
			So(err, ShouldBeNil)

			Convey("Enqueues track request", func() {
				So(len(tq.GetTestable(ctx).GetScheduledTasks()[common.TrackerQueue]), ShouldEqual, 1)
			})

			Convey("Enqueues no driver request", func() {
				So(len(tq.GetTestable(ctx).GetScheduledTasks()[common.DriverQueue]), ShouldEqual, 0)
			})
		})
	})
}

func TestValidateCollectRequest(t *testing.T) {
	Convey("Test Environment", t, func() {
		Convey("A request with run ID and worker name is valid", func() {
			So(validateCollectRequest(&admin.CollectRequest{
				RunId:  int64(1234),
				Worker: "Hello",
			}), ShouldBeNil)
		})

		Convey("A request missing either run ID or worker name is not valid", func() {
			So(validateCollectRequest(&admin.CollectRequest{
				Worker: "Hello",
			}), ShouldNotBeNil)
			So(validateCollectRequest(&admin.CollectRequest{
				RunId: int64(1234),
			}), ShouldNotBeNil)
		})
	})
}
