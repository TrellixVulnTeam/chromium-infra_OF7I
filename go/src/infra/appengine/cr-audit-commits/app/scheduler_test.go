// Copyright 20208 The Chromium Authors. All rights reserved.
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

	"infra/appengine/cr-audit-commits/app/config"
	"infra/appengine/cr-audit-commits/app/rules"

	gax "github.com/googleapis/gax-go/v2"
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

		// Nil out DynamicRefFunctions so the Schedule call doesn't try to actually
		// make network calls to them.
		for _, cfg := range config.GetRuleMap() {
			cfg.DynamicRefFunction = nil
		}

		config.GetRuleMap()["new-repo"] = &rules.RefConfig{
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
		}

		Convey("CreateTask fails", func() {
			sched := &scheduler{
				createTask: func(ctx context.Context, req *taskspb.CreateTaskRequest, opts ...gax.CallOption) (*taskspb.Task, error) {
					return nil, fmt.Errorf("not implemented")
				},
			}
			sched.Schedule(c)
			So(w.Code, ShouldEqual, http.StatusInternalServerError)
		})

		Convey("CreateTask succeeds", func() {
			sched := &scheduler{
				createTask: func(ctx context.Context, req *taskspb.CreateTaskRequest, opts ...gax.CallOption) (*taskspb.Task, error) {
					return &taskspb.Task{}, nil
				},
			}
			sched.Schedule(c)
			So(w.Code, ShouldEqual, http.StatusOK)
		})
	})
}
