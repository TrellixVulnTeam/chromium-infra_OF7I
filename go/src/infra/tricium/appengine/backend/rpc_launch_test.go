// Copyright 2016 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	ds "go.chromium.org/luci/gae/service/datastore"
	tq "go.chromium.org/luci/gae/service/taskqueue"

	admin "infra/tricium/api/admin/v1"
	tricium "infra/tricium/api/v1"
	"infra/tricium/appengine/common"
	"infra/tricium/appengine/common/config"
	"infra/tricium/appengine/common/triciumtest"
)

const (
	hello   = "Hello"
	pylint  = "Pylint"
	project = "playground-gerrit-tricium"
)

func TestLaunchRequest(t *testing.T) {
	Convey("Test Environment", t, func() {
		ctx := triciumtest.Context()

		cp := &mockConfigProvider{
			Projects: map[string]*tricium.ProjectConfig{
				project: {
					Selections: []*tricium.Selection{
						{
							Function: hello,
							Platform: tricium.Platform_UBUNTU,
						},
						{
							Function: pylint,
							Platform: tricium.Platform_UBUNTU,
						},
					},
				},
			},
			ServiceConfig: &tricium.ServiceConfig{
				BuildbucketServerHost: "cr-buildbucket-dev",
				Platforms: []*tricium.Platform_Details{
					{
						Name:       tricium.Platform_UBUNTU,
						HasRuntime: true,
					},
				},
				DataDetails: []*tricium.Data_TypeDetails{
					{
						Type:               tricium.Data_GIT_FILE_DETAILS,
						IsPlatformSpecific: false,
					},
					{
						Type:               tricium.Data_RESULTS,
						IsPlatformSpecific: true,
					},
				},
				Functions: []*tricium.Function{
					{
						Type:     tricium.Function_ANALYZER,
						Name:     hello,
						Needs:    tricium.Data_GIT_FILE_DETAILS,
						Provides: tricium.Data_RESULTS,
						Impls: []*tricium.Impl{
							{
								ProvidesForPlatform: tricium.Platform_UBUNTU,
								RuntimePlatform:     tricium.Platform_UBUNTU,
								Impl: &tricium.Impl_Recipe{
									Recipe: &tricium.Recipe{
										Project: "chromium",
										Bucket:  "try",
										Builder: "hello-analysis",
									},
								},
							},
						},
					},
					{
						Type:     tricium.Function_ANALYZER,
						Name:     pylint,
						Needs:    tricium.Data_GIT_FILE_DETAILS,
						Provides: tricium.Data_RESULTS,
						Impls: []*tricium.Impl{
							{
								ProvidesForPlatform: tricium.Platform_UBUNTU,
								RuntimePlatform:     tricium.Platform_UBUNTU,
								Impl: &tricium.Impl_Recipe{
									Recipe: &tricium.Recipe{
										Project: "chromium",
										Bucket:  "try",
										Builder: "pylint-analysis",
									},
								},
							},
						},
					},
				},
			},
		}

		runID := int64(123456789)
		Convey("Launch request", func() {
			err := launch(ctx, &admin.LaunchRequest{
				RunId:   runID,
				Project: project,
				GitRef:  "ref/test",
				Files: []*tricium.Data_File{
					{Path: "README.md"},
					{Path: "README2.md"},
				},
				CommitMessage: "CL summary\n\nBug: 123\n",
			}, cp, common.MockPubSub)
			So(err, ShouldBeNil)

			Convey("Enqueues track request", func() {
				So(len(tq.GetTestable(ctx).GetScheduledTasks()[common.TrackerQueue]), ShouldEqual, 1)
			})

			Convey("Stores workflow config", func() {
				wf := &config.Workflow{ID: runID}
				So(ds.Get(ctx, wf), ShouldBeNil)
			})

			Convey("Enqueues driver requests", func() {
				So(len(tq.GetTestable(ctx).GetScheduledTasks()[common.DriverQueue]), ShouldEqual, 2)
			})

			// Check guard: one more launch request with the same run ID results in no added tasks.
			err = launch(ctx, &admin.LaunchRequest{
				RunId:   runID,
				Project: project,
				GitRef:  "ref/test",
				Files: []*tricium.Data_File{
					{Path: "README.md"},
					{Path: "README2.md"},
				},
				CommitMessage: "CL summary\n\nBug: 123\n",
			}, config.MockProvider, common.MockPubSub)
			So(err, ShouldBeNil)

			Convey("Succeeding launch request for the same run enqueues no track request", func() {
				So(len(tq.GetTestable(ctx).GetScheduledTasks()[common.TrackerQueue]), ShouldEqual, 1)
			})

		})
	})
}
