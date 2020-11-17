// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package scheduler

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/luci/common/proto"
	gerritpb "go.chromium.org/luci/common/proto/gerrit"
	. "go.chromium.org/luci/common/testing/assertions"
	"go.chromium.org/luci/gae/impl/memory"
	"go.chromium.org/luci/server/auth"
	"go.chromium.org/luci/server/auth/signing"
	"go.chromium.org/luci/server/auth/signing/signingtest"
	"go.chromium.org/luci/server/tq"
	"go.chromium.org/luci/server/tq/tqtesting"

	"infra/appengine/rubber-stamper/config"
	rsgerrit "infra/appengine/rubber-stamper/internal/gerrit"
	"infra/appengine/rubber-stamper/tasks/taskspb"
)

func TestScheduleReviews(t *testing.T) {
	Convey("schedule reviews", t, func() {
		cfg := &config.Config{
			HostConfigs: map[string]*config.HostConfig{
				"test-host": {},
			},
		}
		ctx := memory.Use(context.Background())
		ctx = rsgerrit.Setup(ctx)
		So(config.SetTestConfig(ctx, cfg), ShouldBeNil)
		ctx, sched := tq.TestingContext(ctx, nil)
		ctx = auth.ModifyConfig(ctx, func(cfg auth.Config) auth.Config {
			cfg.Signer = signingtest.NewSigner(&signing.ServiceInfo{
				ServiceAccountName: "srv-account@example.com",
			})
			return cfg
		})

		ctl := gomock.NewController(t)
		defer ctl.Finish()
		gerritMock := gerritpb.NewMockGerritClient(ctl)
		clientMap := map[string]rsgerrit.Client{
			getGerritHostURL("test-host"): gerritMock,
		}
		ctx = rsgerrit.SetTestClientFactory(ctx, clientMap)

		var succeeded tqtesting.TaskList
		sched.TaskSucceeded = tqtesting.TasksCollector(&succeeded)
		sched.TaskFailed = func(ctx context.Context, task *tqtesting.Task) { panic("should not fail") }

		gerritMock.EXPECT().ListChanges(gomock.Any(), proto.MatcherEqual(&gerritpb.ListChangesRequest{
			Query:   "status:open r:srv-account@example.com",
			Options: []gerritpb.QueryOption{gerritpb.QueryOption_CURRENT_REVISION},
		})).Return(&gerritpb.ListChangesResponse{
			Changes: []*gerritpb.ChangeInfo{
				{
					Number:          00000,
					CurrentRevision: "123abc",
				},
				{
					Number:          00001,
					CurrentRevision: "234abc",
				},
			},
			MoreChanges: false,
		}, nil)

		err := ScheduleReviews(ctx)
		So(err, ShouldBeNil)

		sched.Run(ctx, tqtesting.StopWhenDrained())
		So(succeeded.Payloads(), ShouldResembleProto, []*taskspb.ChangeReviewTask{
			{
				Host:     "test-host",
				Number:   00000,
				Revision: "123abc",
			},
			{
				Host:     "test-host",
				Number:   00001,
				Revision: "234abc",
			},
		})
	})
}
