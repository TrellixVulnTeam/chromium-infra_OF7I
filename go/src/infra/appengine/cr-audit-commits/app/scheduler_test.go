// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.chromium.org/luci/gae/impl/memory"
	"go.chromium.org/luci/server/router"

	"infra/appengine/cr-audit-commits/app/rules"
	cloudtasksmodule "infra/libs/grpcclient/cloudtasks"

	taskspb "google.golang.org/genproto/googleapis/cloud/tasks/v2"

	. "github.com/smartystreets/goconvey/convey"
)

// Remaining cases that can fail, which we probably want to verify:
// DynamicRefFunction call fails.
// Datastore RepoState fetch fails.

func TestScheduler(t *testing.T) {
	Convey("Scheduler", t, func() {
		ctx := memory.Use(context.Background())
		w := &httptest.ResponseRecorder{}
		c := &router.Context{Context: ctx, Writer: w}

		// Mock global configuration.
		expectedRuleMap := map[string]*rules.RefConfig{
			"new-repo": {
				BaseRepoURL:    "https://new.googlesource.com/new.git",
				GerritURL:      "https://new-review.googlesource.com",
				BranchName:     "master",
				StartingCommit: "000000",
				Rules: map[string]rules.AccountRules{"rules": {
					Account: "new@test.com",
					Rules: []rules.Rule{
						rules.DummyRule{
							Name: "DummyRule",
							Result: &rules.RuleResult{
								RuleName:         "Dummy rule",
								RuleResultStatus: rules.RulePassed,
								Message:          "",
								MetaData:         "",
							},
						},
					},
				}},
			},
		}
		configGetOld := configGet
		configGet = func(context.Context) map[string]*rules.RefConfig {
			return expectedRuleMap
		}
		defer func() {
			configGet = configGetOld
		}()

		Convey("CreateTask fails", func() {
			fakeCloudTasks := &cloudtasksmodule.FakeServer{
				CreateTaskError: fmt.Errorf("default error for testing"),
			}
			fakeServer, err := fakeCloudTasks.Start(ctx)
			defer fakeServer.Stop()
			So(err, ShouldBeNil)

			tasksClient, err := fakeCloudTasks.NewClient(ctx)
			So(err, ShouldBeNil)

			appServer := &app{
				cloudTasksClient: tasksClient,
			}
			appServer.Schedule(c)
			So(w.Code, ShouldEqual, http.StatusInternalServerError)
		})

		Convey("CreateTask succeeds", func() {
			fakeCloudTasks := &cloudtasksmodule.FakeServer{
				CreateTaskResponse: &taskspb.Task{},
			}
			fakeServer, err := fakeCloudTasks.Start(ctx)
			defer fakeServer.Stop()
			So(err, ShouldBeNil)

			tasksClient, err := fakeCloudTasks.NewClient(ctx)
			So(err, ShouldBeNil)
			appServer := &app{
				cloudTasksClient: tasksClient,
			}
			appServer.Schedule(c)

			So(w.Code, ShouldEqual, http.StatusOK)
		})
	})
}
