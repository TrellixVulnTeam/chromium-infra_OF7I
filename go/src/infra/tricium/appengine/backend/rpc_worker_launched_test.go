// Copyright 2017 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	ds "go.chromium.org/luci/gae/service/datastore"

	admin "infra/tricium/api/admin/v1"
	tricium "infra/tricium/api/v1"
	"infra/tricium/appengine/common/track"
	"infra/tricium/appengine/common/triciumtest"
)

func TestWorkerLaunchedRequest(t *testing.T) {
	Convey("Test Environment", t, func() {
		ctx := triciumtest.Context()
		helloUbuntu := "Hello_Ubuntu"

		workflowProvider := &mockWorkflowProvider{
			Workflow: &admin.Workflow{
				Workers: []*admin.Worker{
					{
						Name:     helloUbuntu,
						Needs:    tricium.Data_GIT_FILE_DETAILS,
						Provides: tricium.Data_RESULTS,
					},
				},
			},
		}

		// Add pending workflow run entity.
		request := &track.AnalyzeRequest{}
		So(ds.Put(ctx, request), ShouldBeNil)
		requestKey := ds.KeyForObj(ctx, request)
		workflowRun := &track.WorkflowRun{ID: 1, Parent: requestKey}
		So(ds.Put(ctx, workflowRun), ShouldBeNil)
		workflowRunKey := ds.KeyForObj(ctx, workflowRun)
		So(ds.Put(ctx, &track.WorkflowRunResult{
			ID:     1,
			Parent: workflowRunKey,
			State:  tricium.State_PENDING,
		}), ShouldBeNil)

		// Mark workflow as launched and add tracking entities for workers.
		err := workflowLaunched(ctx, &admin.WorkflowLaunchedRequest{
			RunId: request.ID,
		}, workflowProvider)
		So(err, ShouldBeNil)

		// Mark worker as launched.
		err = workerLaunched(ctx, &admin.WorkerLaunchedRequest{
			RunId:  request.ID,
			Worker: helloUbuntu,
		})
		So(err, ShouldBeNil)

		Convey("Marks worker as launched", func() {
			functionName, _, err := track.ExtractFunctionPlatform(helloUbuntu)
			So(err, ShouldBeNil)
			functionRunKey := ds.NewKey(ctx, "FunctionRun", functionName, 0, workflowRunKey)
			workerKey := ds.NewKey(ctx, "WorkerRun", helloUbuntu, 0, functionRunKey)
			wr := &track.WorkerRunResult{ID: 1, Parent: workerKey}
			err = ds.Get(ctx, wr)
			So(err, ShouldBeNil)
			So(wr.State, ShouldEqual, tricium.State_RUNNING)
			fr := &track.FunctionRunResult{ID: 1, Parent: functionRunKey}
			err = ds.Get(ctx, fr)
			So(err, ShouldBeNil)
			So(fr.State, ShouldEqual, tricium.State_RUNNING)
		})

		Convey("Validates request", func() {
			// Validate run ID.
			s := &trackerServer{}
			_, err = s.WorkerLaunched(ctx, &admin.WorkerLaunchedRequest{})
			So(err.Error(), ShouldEqual, "rpc error: code = InvalidArgument desc = missing run ID")

			// Validate worker.
			_, err = s.WorkerLaunched(ctx, &admin.WorkerLaunchedRequest{
				RunId: request.ID,
			})
			So(err.Error(), ShouldEqual, "rpc error: code = InvalidArgument desc = missing worker")

			// Validate buildbucket.
			_, err = s.WorkerLaunched(ctx, &admin.WorkerLaunchedRequest{
				RunId:              request.ID,
				Worker:             helloUbuntu,
				BuildbucketBuildId: 12,
			})
			So(err, ShouldBeNil)
		})
	})
}
