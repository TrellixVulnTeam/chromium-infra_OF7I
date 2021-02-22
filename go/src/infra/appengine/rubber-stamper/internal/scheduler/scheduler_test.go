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
	"go.chromium.org/luci/server/tq/tqtesting"

	"infra/appengine/rubber-stamper/config"
	"infra/appengine/rubber-stamper/internal/util"
	"infra/appengine/rubber-stamper/tasks/taskspb"
)

func TestScheduleReviews(t *testing.T) {
	Convey("schedule reviews", t, func() {
		cfg := &config.Config{
			HostConfigs: map[string]*config.HostConfig{
				"test-host": {
					RepoConfigs: map[string]*config.RepoConfig{
						"dummy": {
							BenignFilePattern: &config.BenignFilePattern{
								Paths: []string{"a/x", "a/q/y"},
							},
						},
					},
				},
			},
		}
		ctx := memory.Use(context.Background())
		ctx, gerritMock, sched := util.SetupTestingContext(ctx, cfg, "srv-account@example.com", "test-host", t)

		var succeeded tqtesting.TaskList
		sched.TaskSucceeded = tqtesting.TasksCollector(&succeeded)
		sched.TaskFailed = func(ctx context.Context, task *tqtesting.Task) { panic("should not fail") }

		gerritMock.EXPECT().ListChanges(gomock.Any(), proto.MatcherEqual(&gerritpb.ListChangesRequest{
			Query:   "status:open r:srv-account@example.com",
			Options: []gerritpb.QueryOption{gerritpb.QueryOption_ALL_REVISIONS, gerritpb.QueryOption_LABELS},
		})).Return(&gerritpb.ListChangesResponse{
			Changes: []*gerritpb.ChangeInfo{
				{
					Number:          00000,
					CurrentRevision: "123abc",
					Project:         "dummy",
					Revisions: map[string]*gerritpb.RevisionInfo{
						"123abc": {},
					},
					Labels: map[string]*gerritpb.LabelInfo{
						"Auto-Submit": {Approved: &gerritpb.AccountInfo{}},
					},
				},
				{
					Number:          00001,
					CurrentRevision: "234abc",
					Project:         "dummy",
					Revisions: map[string]*gerritpb.RevisionInfo{
						"234abc": {},
					},
				},
				{
					Number:             00002,
					CurrentRevision:    "789012",
					Project:            "dummy",
					CherryPickOfChange: 11234,
					Revisions: map[string]*gerritpb.RevisionInfo{
						"112233": {
							Number: 1,
						},
						"789012": {
							Number: 2,
						},
					},
				},
			},
			MoreChanges: false,
		}, nil)
		gerritMock.EXPECT().ListFiles(gomock.Any(), proto.MatcherEqual(&gerritpb.ListFilesRequest{
			Number:     00000,
			RevisionId: "123abc",
		})).Return(&gerritpb.ListFilesResponse{
			Files: map[string]*gerritpb.FileInfo{
				"a/x": nil,
			},
		}, nil)
		gerritMock.EXPECT().ListFiles(gomock.Any(), proto.MatcherEqual(&gerritpb.ListFilesRequest{
			Number:     00001,
			RevisionId: "234abc",
		})).Return(&gerritpb.ListFilesResponse{
			Files: map[string]*gerritpb.FileInfo{
				"a/q/y": nil,
			},
		}, nil)
		gerritMock.EXPECT().ListFiles(gomock.Any(), proto.MatcherEqual(&gerritpb.ListFilesRequest{
			Number:     00002,
			RevisionId: "789012",
			Base:       "1",
		})).Return(&gerritpb.ListFilesResponse{
			Files: map[string]*gerritpb.FileInfo{
				"/COMMIT_MSG": nil,
				"no.md":       nil,
				"invalid.go":  nil,
			},
		}, nil)
		gerritMock.EXPECT().ListFiles(gomock.Any(), proto.MatcherEqual(&gerritpb.ListFilesRequest{
			Number:     00002,
			RevisionId: "789012",
		})).Return(&gerritpb.ListFilesResponse{
			Files: map[string]*gerritpb.FileInfo{
				"a/q/zz": nil,
			},
		}, nil)
		gerritMock.EXPECT().SetReview(gomock.Any(), proto.MatcherEqual(&gerritpb.SetReviewRequest{
			Number:     00000,
			RevisionId: "123abc",
			Labels:     map[string]int32{"Bot-Commit": 1, "Commit-Queue": 2},
		})).Return(&gerritpb.ReviewResult{}, nil)
		gerritMock.EXPECT().SetReview(gomock.Any(), proto.MatcherEqual(&gerritpb.SetReviewRequest{
			Number:     00001,
			RevisionId: "234abc",
			Labels:     map[string]int32{"Bot-Commit": 1},
		})).Return(&gerritpb.ReviewResult{}, nil)
		gerritMock.EXPECT().SetReview(gomock.Any(), proto.MatcherEqual(&gerritpb.SetReviewRequest{
			Number:     00002,
			RevisionId: "789012",
			Message:    "The current revision changed the following files compared with the initial revision: invalid.go, no.md. Learn more: go/rubber-stamper-user-guide.",
		})).Return(&gerritpb.ReviewResult{}, nil)
		gerritMock.EXPECT().DeleteReviewer(gomock.Any(), proto.MatcherEqual(&gerritpb.DeleteReviewerRequest{
			Number:    00002,
			AccountId: "srv-account@example.com",
		})).Return(nil, nil)

		err := ScheduleReviews(ctx)
		So(err, ShouldBeNil)

		sched.Run(ctx, tqtesting.StopWhenDrained())
		So(succeeded.Payloads(), ShouldResembleProto, []*taskspb.ChangeReviewTask{
			{
				Host:           "test-host",
				Number:         00000,
				Revision:       "123abc",
				Repo:           "dummy",
				AutoSubmit:     true,
				RevisionsCount: 1,
			},
			{
				Host:           "test-host",
				Number:         00001,
				Revision:       "234abc",
				Repo:           "dummy",
				AutoSubmit:     false,
				RevisionsCount: 1,
			},
			{
				Host:               "test-host",
				Number:             00002,
				Revision:           "789012",
				Repo:               "dummy",
				AutoSubmit:         false,
				CherryPickOfChange: 11234,
				RevisionsCount:     2,
			},
		})
	})
}
