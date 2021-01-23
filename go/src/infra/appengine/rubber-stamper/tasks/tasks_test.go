// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package tasks

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/luci/common/proto"
	gerritpb "go.chromium.org/luci/common/proto/gerrit"
	"go.chromium.org/luci/gae/impl/memory"
	"go.chromium.org/luci/server/tq/tqtesting"
	"google.golang.org/protobuf/types/known/timestamppb"

	"infra/appengine/rubber-stamper/config"
	"infra/appengine/rubber-stamper/internal/util"
	"infra/appengine/rubber-stamper/tasks/taskspb"
)

func TestQueue(t *testing.T) {
	Convey("Chain works", t, func() {
		cfg := &config.Config{
			DefaultTimeWindow: "7d",
			HostConfigs: map[string]*config.HostConfig{
				"host": {
					RepoConfigs: map[string]*config.RepoConfig{
						"dummy": {
							BenignFilePattern: &config.BenignFilePattern{
								Paths: []string{"a/b.txt"},
							},
						},
					},
				},
			},
		}

		ctx := memory.Use(context.Background())
		ctx, gerritMock, sched := util.SetupTestingContext(ctx, cfg, "srv-account@example.com", "host", t)

		var succeeded tqtesting.TaskList

		sched.TaskSucceeded = tqtesting.TasksCollector(&succeeded)
		sched.TaskFailed = func(ctx context.Context, task *tqtesting.Task) { panic("should not fail") }

		Convey("Test deduplication", func() {
			host := "host"
			cls := []*gerritpb.ChangeInfo{
				{
					Number:          12345,
					CurrentRevision: "123abc",
					Project:         "dummy",
					Labels: map[string]*gerritpb.LabelInfo{
						"Auto-Submit": {Approved: &gerritpb.AccountInfo{}},
					},
					Revisions: map[string]*gerritpb.RevisionInfo{
						"123abc": {},
					},
				},
				{
					Number:          12345,
					CurrentRevision: "456789",
					Project:         "dummy",
					Revisions: map[string]*gerritpb.RevisionInfo{
						"123abc": {},
					},
				},
				{
					Number:          12315,
					CurrentRevision: "112233",
					Project:         "dummy",
					RevertOf:        129380,
					Revisions: map[string]*gerritpb.RevisionInfo{
						"112233": {},
						"456123": {},
					},
				},
				{
					Number:             12387,
					CurrentRevision:    "111aaa",
					Project:            "dummy",
					CherryPickOfChange: 129380,
					Revisions: map[string]*gerritpb.RevisionInfo{
						"111aaa": {},
					},
				},
			}

			// cls[0]: benign file change with Auto-Submit
			gerritMock.EXPECT().ListFiles(gomock.Any(), proto.MatcherEqual(&gerritpb.ListFilesRequest{
				Number:     cls[0].Number,
				RevisionId: cls[0].CurrentRevision,
			})).Return(&gerritpb.ListFilesResponse{
				Files: map[string]*gerritpb.FileInfo{
					"a/b.txt": nil,
				},
			}, nil)
			gerritMock.EXPECT().SetReview(gomock.Any(), proto.MatcherEqual(&gerritpb.SetReviewRequest{
				Number:     cls[0].Number,
				RevisionId: cls[0].CurrentRevision,
				Labels:     map[string]int32{"Bot-Commit": 1, "Commit-Queue": 2},
			})).Return(&gerritpb.ReviewResult{}, nil)

			// cls[1]: benign file change
			gerritMock.EXPECT().ListFiles(gomock.Any(), proto.MatcherEqual(&gerritpb.ListFilesRequest{
				Number:     cls[1].Number,
				RevisionId: cls[1].CurrentRevision,
			})).Return(&gerritpb.ListFilesResponse{
				Files: map[string]*gerritpb.FileInfo{
					"a/b.txt": nil,
				},
			}, nil)
			gerritMock.EXPECT().SetReview(gomock.Any(), proto.MatcherEqual(&gerritpb.SetReviewRequest{
				Number:     cls[1].Number,
				RevisionId: cls[1].CurrentRevision,
				Labels:     map[string]int32{"Bot-Commit": 1},
			})).Return(&gerritpb.ReviewResult{}, nil)

			// cls[2]: clean revert
			gerritMock.EXPECT().GetPureRevert(gomock.Any(), proto.MatcherEqual(&gerritpb.GetPureRevertRequest{
				Number:  cls[2].Number,
				Project: cls[2].Project,
			})).Return(&gerritpb.PureRevertInfo{
				IsPureRevert: true,
			}, nil)
			gerritMock.EXPECT().GetChange(gomock.Any(), proto.MatcherEqual(&gerritpb.GetChangeRequest{
				Number:  cls[2].RevertOf,
				Options: []gerritpb.QueryOption{gerritpb.QueryOption_CURRENT_REVISION},
			})).Return(&gerritpb.ChangeInfo{
				CurrentRevision: "aa1def",
				Revisions: map[string]*gerritpb.RevisionInfo{
					"aa1def": {
						Created: timestamppb.Now(),
					},
				},
			}, nil)
			gerritMock.EXPECT().SetReview(gomock.Any(), proto.MatcherEqual(&gerritpb.SetReviewRequest{
				Number:     cls[2].Number,
				RevisionId: cls[2].CurrentRevision,
				Labels:     map[string]int32{"Bot-Commit": 1},
			})).Return(&gerritpb.ReviewResult{}, nil)

			// cls[3]: clean cherry-pick
			gerritMock.EXPECT().GetChange(gomock.Any(), proto.MatcherEqual(&gerritpb.GetChangeRequest{
				Number:  cls[3].CherryPickOfChange,
				Options: []gerritpb.QueryOption{gerritpb.QueryOption_CURRENT_REVISION},
			})).Return(&gerritpb.ChangeInfo{
				CurrentRevision: "456def",
				Revisions: map[string]*gerritpb.RevisionInfo{
					"456def": {
						Created: timestamppb.Now(),
					},
				},
			}, nil)
			gerritMock.EXPECT().GetMergeable(gomock.Any(), proto.MatcherEqual(&gerritpb.GetMergeableRequest{
				Number:     cls[3].Number,
				Project:    cls[3].Project,
				RevisionId: cls[3].CurrentRevision,
			})).Return(&gerritpb.MergeableInfo{
				Mergeable: true,
			}, nil)
			gerritMock.EXPECT().SetReview(gomock.Any(), proto.MatcherEqual(&gerritpb.SetReviewRequest{
				Number:     cls[3].Number,
				RevisionId: cls[3].CurrentRevision,
				Labels:     map[string]int32{"Bot-Commit": 1},
			})).Return(&gerritpb.ReviewResult{}, nil)

			So(EnqueueChangeReviewTask(ctx, host, cls[0]), ShouldBeNil)
			So(EnqueueChangeReviewTask(ctx, host, cls[0]), ShouldBeNil)
			So(EnqueueChangeReviewTask(ctx, host, cls[1]), ShouldBeNil)
			So(EnqueueChangeReviewTask(ctx, host, cls[1]), ShouldBeNil)
			So(EnqueueChangeReviewTask(ctx, host, cls[2]), ShouldBeNil)
			So(EnqueueChangeReviewTask(ctx, host, cls[3]), ShouldBeNil)
			sched.Run(ctx, tqtesting.StopWhenDrained())
			So(len(succeeded.Payloads()), ShouldEqual, 4)
			So(succeeded.Payloads(), ShouldContain, &taskspb.ChangeReviewTask{
				Host:           "host",
				Number:         12345,
				Revision:       "123abc",
				Repo:           "dummy",
				AutoSubmit:     true,
				RevisionsCount: 1,
			})
			So(succeeded.Payloads(), ShouldContain, &taskspb.ChangeReviewTask{
				Host:           "host",
				Number:         12345,
				Revision:       "456789",
				Repo:           "dummy",
				AutoSubmit:     false,
				RevisionsCount: 1,
			})
			So(succeeded.Payloads(), ShouldContain, &taskspb.ChangeReviewTask{
				Host:           "host",
				Number:         12315,
				Revision:       "112233",
				Repo:           "dummy",
				AutoSubmit:     false,
				RevertOf:       129380,
				RevisionsCount: 2,
			})
			So(succeeded.Payloads(), ShouldContain, &taskspb.ChangeReviewTask{
				Host:               "host",
				Number:             12387,
				Revision:           "111aaa",
				Repo:               "dummy",
				AutoSubmit:         false,
				CherryPickOfChange: 129380,
				RevisionsCount:     1,
			})
		})
	})
}
