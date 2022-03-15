// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package reviewer

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/luci/common/proto"
	gerritpb "go.chromium.org/luci/common/proto/gerrit"
	. "go.chromium.org/luci/common/testing/assertions"
	"go.chromium.org/luci/gae/impl/memory"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/timestamppb"

	"infra/appengine/rubber-stamper/config"
	"infra/appengine/rubber-stamper/tasks/taskspb"
)

func TestReviewCleanCherryPick(t *testing.T) {
	Convey("review clean cherry pick", t, func() {
		ctx := memory.Use(context.Background())

		ctl := gomock.NewController(t)
		defer ctl.Finish()
		gerritMock := gerritpb.NewMockGerritClient(ctl)

		cfg := &config.Config{
			DefaultTimeWindow: "7d",
			HostConfigs: map[string]*config.HostConfig{
				"test-host": {
					RepoConfigs: map[string]*config.RepoConfig{},
				},
			},
		}

		t := &taskspb.ChangeReviewTask{
			Host:               "test-host",
			Number:             12345,
			Revision:           "123abc",
			Repo:               "dummy",
			AutoSubmit:         false,
			RevisionsCount:     1,
			CherryPickOfChange: 12121,
			Created:            timestamppb.New(time.Now().Add(-time.Minute)),
		}

		Convey("decline when the current revision made any file changes compared with the initial version", func() {
			t.RevisionsCount = 2
			gerritMock.EXPECT().ListFiles(gomock.Any(), proto.MatcherEqual(&gerritpb.ListFilesRequest{
				Number:     t.Number,
				RevisionId: t.Revision,
				Base:       "1",
			})).Return(&gerritpb.ListFilesResponse{
				Files: map[string]*gerritpb.FileInfo{
					"/COMMIT_MSG": nil,
					"no.txt":      nil,
				},
			}, nil)

			msg, err := reviewCleanCherryPick(ctx, cfg, gerritMock, t)
			So(err, ShouldBeNil)
			So(msg, ShouldEqual, "The current revision changed the following files compared with the initial revision: no.txt.")
		})
		Convey("decline when out of configured time window", func() {
			Convey("global time window works", func() {
				gerritMock.EXPECT().GetChange(gomock.Any(), proto.MatcherEqual(&gerritpb.GetChangeRequest{
					Number:  t.CherryPickOfChange,
					Options: []gerritpb.QueryOption{gerritpb.QueryOption_CURRENT_REVISION},
				})).Return(&gerritpb.ChangeInfo{
					Status:          gerritpb.ChangeStatus_MERGED,
					CurrentRevision: "456def",
					Revisions: map[string]*gerritpb.RevisionInfo{
						"456def": {
							Created: timestamppb.New(time.Now().Add(-8 * 24 * time.Hour)),
						},
						"789aaa": {
							Created: timestamppb.New(time.Now().Add(-9 * 24 * time.Hour)),
						},
					},
				}, nil)
				msg, err := reviewCleanCherryPick(ctx, cfg, gerritMock, t)
				So(err, ShouldBeNil)
				So(msg, ShouldEqual, "The change is not in the configured time window. Rubber Stamper is only allowed to review cherry-picks within 7 day(s).")
			})
			Convey("host-level time window works", func() {
				cfg.HostConfigs["test-host"].CleanCherryPickTimeWindow = "5d"
				gerritMock.EXPECT().GetChange(gomock.Any(), proto.MatcherEqual(&gerritpb.GetChangeRequest{
					Number:  t.CherryPickOfChange,
					Options: []gerritpb.QueryOption{gerritpb.QueryOption_CURRENT_REVISION},
				})).Return(&gerritpb.ChangeInfo{
					Status:          gerritpb.ChangeStatus_MERGED,
					CurrentRevision: "456def",
					Revisions: map[string]*gerritpb.RevisionInfo{
						"456def": {
							Created: timestamppb.New(time.Now().Add(-6 * 24 * time.Hour)),
						},
						"789aaa": {
							Created: timestamppb.New(time.Now().Add(-9 * 24 * time.Hour)),
						},
					},
				}, nil)
				msg, err := reviewCleanCherryPick(ctx, cfg, gerritMock, t)
				So(err, ShouldBeNil)
				So(msg, ShouldEqual, "The change is not in the configured time window. Rubber Stamper is only allowed to review cherry-picks within 5 day(s).")
			})
			Convey("repo-level time window works", func() {
				cfg.HostConfigs["test-host"].CleanCherryPickTimeWindow = "5d"
				cfg.HostConfigs["test-host"].RepoConfigs["dummy"] = &config.RepoConfig{
					CleanCherryPickPattern: &config.CleanCherryPickPattern{
						TimeWindow: "58m",
					},
				}
				gerritMock.EXPECT().GetChange(gomock.Any(), proto.MatcherEqual(&gerritpb.GetChangeRequest{
					Number:  t.CherryPickOfChange,
					Options: []gerritpb.QueryOption{gerritpb.QueryOption_CURRENT_REVISION},
				})).Return(&gerritpb.ChangeInfo{
					Status:          gerritpb.ChangeStatus_MERGED,
					CurrentRevision: "456def",
					Revisions: map[string]*gerritpb.RevisionInfo{
						"456def": {
							Created: timestamppb.New(time.Now().Add(-time.Hour)),
						},
						"789aaa": {
							Created: timestamppb.New(time.Now().Add(-9 * 24 * time.Hour)),
						},
					},
				}, nil)
				msg, err := reviewCleanCherryPick(ctx, cfg, gerritMock, t)
				So(err, ShouldBeNil)
				So(msg, ShouldEqual, "The change is not in the configured time window. Rubber Stamper is only allowed to review cherry-picks within 58 minute(s).")
			})
		})
		Convey("decline when the change wasn't cherry-picked after the original CL has been merged.", func() {
			Convey("decline when the original CL hasn't been merged", func() {
				gerritMock.EXPECT().GetChange(gomock.Any(), proto.MatcherEqual(&gerritpb.GetChangeRequest{
					Number:  t.CherryPickOfChange,
					Options: []gerritpb.QueryOption{gerritpb.QueryOption_CURRENT_REVISION},
				})).Return(&gerritpb.ChangeInfo{
					Status:          gerritpb.ChangeStatus_NEW,
					CurrentRevision: "456def",
					Revisions: map[string]*gerritpb.RevisionInfo{
						"456def": {
							Created: timestamppb.New(time.Now().Add(-24 * time.Hour)),
						},
					},
				}, nil)
				msg, err := reviewCleanCherryPick(ctx, cfg, gerritMock, t)
				So(err, ShouldBeNil)
				So(msg, ShouldEqual, "The change is not cherry-picked after the original CL has been merged.")
			})
			Convey("decline when cherry-picked before the original CL has been merged", func() {
				gerritMock.EXPECT().GetChange(gomock.Any(), proto.MatcherEqual(&gerritpb.GetChangeRequest{
					Number:  t.CherryPickOfChange,
					Options: []gerritpb.QueryOption{gerritpb.QueryOption_CURRENT_REVISION},
				})).Return(&gerritpb.ChangeInfo{
					Status:          gerritpb.ChangeStatus_MERGED,
					CurrentRevision: "456def",
					Revisions: map[string]*gerritpb.RevisionInfo{
						"456def": {
							Created: timestamppb.New(time.Now().Add(-24 * time.Hour)),
						},
					},
				}, nil)
				t.Created = timestamppb.New(time.Now().Add(-24*time.Hour - time.Minute))
				msg, err := reviewCleanCherryPick(ctx, cfg, gerritMock, t)
				So(err, ShouldBeNil)
				So(msg, ShouldEqual, "The change is not cherry-picked after the original CL has been merged.")
			})
		})
		Convey("decline when alters any excluded file", func() {
			cfg.HostConfigs["test-host"].RepoConfigs["dummy"] = &config.RepoConfig{
				CleanCherryPickPattern: &config.CleanCherryPickPattern{
					ExcludedPaths: []string{"p/q/**", "**.c"},
				},
			}
			gerritMock.EXPECT().GetChange(gomock.Any(), proto.MatcherEqual(&gerritpb.GetChangeRequest{
				Number:  t.CherryPickOfChange,
				Options: []gerritpb.QueryOption{gerritpb.QueryOption_CURRENT_REVISION},
			})).Return(&gerritpb.ChangeInfo{
				Status:          gerritpb.ChangeStatus_MERGED,
				CurrentRevision: "456def",
				Revisions: map[string]*gerritpb.RevisionInfo{
					"456def": {
						Created: timestamppb.New(time.Now().Add(-5 * 24 * time.Hour)),
					},
					"789aaa": {
						Created: timestamppb.New(time.Now().Add(-9 * 24 * time.Hour)),
					},
				},
			}, nil)
			gerritMock.EXPECT().ListFiles(gomock.Any(), proto.MatcherEqual(&gerritpb.ListFilesRequest{
				Number:     t.Number,
				RevisionId: t.Revision,
			})).Return(&gerritpb.ListFilesResponse{
				Files: map[string]*gerritpb.FileInfo{
					"p/q/o/0.txt": nil,
					"valid.md":    nil,
					"a/invalid.c": nil,
				},
			}, nil)
			msg, err := reviewCleanCherryPick(ctx, cfg, gerritMock, t)
			So(err, ShouldBeNil)
			So(msg, ShouldEqual, "The change contains the following files which require a human reviewer: a/invalid.c, p/q/o/0.txt.")
		})
		Convey("decline when not mergeable", func() {
			gerritMock.EXPECT().GetChange(gomock.Any(), proto.MatcherEqual(&gerritpb.GetChangeRequest{
				Number:  t.CherryPickOfChange,
				Options: []gerritpb.QueryOption{gerritpb.QueryOption_CURRENT_REVISION},
			})).Return(&gerritpb.ChangeInfo{
				Status:          gerritpb.ChangeStatus_MERGED,
				CurrentRevision: "456def",
				Revisions: map[string]*gerritpb.RevisionInfo{
					"456def": {
						Created: timestamppb.New(time.Now().Add(-5 * 24 * time.Hour)),
					},
					"789aaa": {
						Created: timestamppb.New(time.Now().Add(-9 * 24 * time.Hour)),
					},
				},
			}, nil)
			gerritMock.EXPECT().GetMergeable(gomock.Any(), proto.MatcherEqual(&gerritpb.GetMergeableRequest{
				Number:     t.Number,
				Project:    t.Repo,
				RevisionId: t.Revision,
			})).Return(&gerritpb.MergeableInfo{
				Mergeable: false,
			}, nil)
			msg, err := reviewCleanCherryPick(ctx, cfg, gerritMock, t)
			So(err, ShouldBeNil)
			So(msg, ShouldEqual, "The change is not mergeable.")
		})
		Convey("return error works", func() {
			Convey("Gerrit GetChange API returns error", func() {
				gerritMock.EXPECT().GetChange(gomock.Any(), proto.MatcherEqual(&gerritpb.GetChangeRequest{
					Number:  t.CherryPickOfChange,
					Options: []gerritpb.QueryOption{gerritpb.QueryOption_CURRENT_REVISION},
				})).Return(nil, grpc.Errorf(codes.NotFound, "not found"))
				msg, err := reviewCleanCherryPick(ctx, cfg, gerritMock, t)
				So(msg, ShouldEqual, "")
				So(err, ShouldErrLike, "gerrit GetChange rpc call failed with error")
			})
			Convey("Gerrit ListFiles API returns error", func() {
				cfg.HostConfigs["test-host"].RepoConfigs["dummy"] = &config.RepoConfig{
					CleanCherryPickPattern: &config.CleanCherryPickPattern{
						ExcludedPaths: []string{"p/q/**", "**.c"},
					},
				}
				gerritMock.EXPECT().GetChange(gomock.Any(), proto.MatcherEqual(&gerritpb.GetChangeRequest{
					Number:  t.CherryPickOfChange,
					Options: []gerritpb.QueryOption{gerritpb.QueryOption_CURRENT_REVISION},
				})).Return(&gerritpb.ChangeInfo{
					Status:          gerritpb.ChangeStatus_MERGED,
					CurrentRevision: "456def",
					Revisions: map[string]*gerritpb.RevisionInfo{
						"456def": {
							Created: timestamppb.New(time.Now().Add(-5 * 24 * time.Hour)),
						},
						"789aaa": {
							Created: timestamppb.New(time.Now().Add(-9 * 24 * time.Hour)),
						},
					},
				}, nil)
				gerritMock.EXPECT().ListFiles(gomock.Any(), proto.MatcherEqual(&gerritpb.ListFilesRequest{
					Number:     t.Number,
					RevisionId: t.Revision,
				})).Return(nil, grpc.Errorf(codes.NotFound, "not found"))
				msg, err := reviewCleanCherryPick(ctx, cfg, gerritMock, t)
				So(msg, ShouldEqual, "")
				So(err, ShouldErrLike, "gerrit ListFiles rpc call failed with error")
			})
			Convey("Gerrit GetMergeable API returns error", func() {
				gerritMock.EXPECT().GetChange(gomock.Any(), proto.MatcherEqual(&gerritpb.GetChangeRequest{
					Number:  t.CherryPickOfChange,
					Options: []gerritpb.QueryOption{gerritpb.QueryOption_CURRENT_REVISION},
				})).Return(&gerritpb.ChangeInfo{
					Status:          gerritpb.ChangeStatus_MERGED,
					CurrentRevision: "456def",
					Revisions: map[string]*gerritpb.RevisionInfo{
						"456def": {
							Created: timestamppb.New(time.Now().Add(-5 * 24 * time.Hour)),
						},
						"789aaa": {
							Created: timestamppb.New(time.Now().Add(-9 * 24 * time.Hour)),
						},
					},
				}, nil)
				gerritMock.EXPECT().GetMergeable(gomock.Any(), proto.MatcherEqual(&gerritpb.GetMergeableRequest{
					Number:     t.Number,
					Project:    t.Repo,
					RevisionId: t.Revision,
				})).Return(nil, grpc.Errorf(codes.NotFound, "not found"))
				msg, err := reviewCleanCherryPick(ctx, cfg, gerritMock, t)
				So(msg, ShouldEqual, "")
				So(err, ShouldErrLike, "gerrit GetMergeable rpc call failed with error")
			})
			Convey("time window config error", func() {
				cfg.HostConfigs["test-host"].CleanCherryPickTimeWindow = "112-1d"
				gerritMock.EXPECT().GetChange(gomock.Any(), proto.MatcherEqual(&gerritpb.GetChangeRequest{
					Number:  t.CherryPickOfChange,
					Options: []gerritpb.QueryOption{gerritpb.QueryOption_CURRENT_REVISION},
				})).Return(&gerritpb.ChangeInfo{
					Status:          gerritpb.ChangeStatus_MERGED,
					CurrentRevision: "456def",
					Revisions: map[string]*gerritpb.RevisionInfo{
						"456def": {
							Created: timestamppb.New(time.Now().Add(-5 * 24 * time.Hour)),
						},
						"789aaa": {
							Created: timestamppb.New(time.Now().Add(-9 * 24 * time.Hour)),
						},
					},
				}, nil)
				msg, err := reviewCleanCherryPick(ctx, cfg, gerritMock, t)
				So(msg, ShouldEqual, "")
				So(err, ShouldErrLike, "invalid time_window config 112-1d")
			})
		})
	})
}

func TestReviewBypassFileCheck(t *testing.T) {
	Convey("bypass file check", t, func() {
		ctx := context.Background()

		fr := &config.CleanCherryPickPattern_FileCheckBypassRule{
			IncludedPaths: []string{"dir_a/dir_b/**/*.json"},
			Hashtag:       "Example_Hashtag",
			AllowedOwners: []string{"userA@example.com", "userB@example.com"},
		}

		invalidFiles := []string{"dir_a/dir_b/test.json", "dir_a/dir_b/dir_c/ok.json"}
		hashtags := []string{"Random", "Example_Hashtag"}
		owner := "userA@example.com"

		Convey("approve", func() {
			So(bypassFileCheck(ctx, invalidFiles, hashtags, owner, fr), ShouldEqual, true)
		})
		Convey("decline when config is incomplete", func() {
			Convey("nil config", func() {
				fr = nil
				So(bypassFileCheck(ctx, invalidFiles, hashtags, owner, fr), ShouldEqual, false)
			})
			Convey("no includedPath", func() {
				fr.IncludedPaths = nil
				So(bypassFileCheck(ctx, invalidFiles, hashtags, owner, fr), ShouldEqual, false)
			})
			Convey("no hashtag", func() {
				fr.Hashtag = ""
				So(bypassFileCheck(ctx, invalidFiles, hashtags, owner, fr), ShouldEqual, false)
			})
			Convey("no allowedOwners", func() {
				fr.AllowedOwners = nil
				So(bypassFileCheck(ctx, invalidFiles, hashtags, owner, fr), ShouldEqual, false)
			})
		})
		Convey("decline when files are not included", func() {
			invalidFiles = append(invalidFiles, "dir_c/ok.json")
			So(bypassFileCheck(ctx, invalidFiles, hashtags, owner, fr), ShouldEqual, false)
		})
		Convey("decline when no hashtag matches", func() {
			hashtags = []string{"Random1", "Random2"}
			So(bypassFileCheck(ctx, invalidFiles, hashtags, owner, fr), ShouldEqual, false)
		})
		Convey("decline when owner is not allowed", func() {
			owner = "userC@example.com"
			So(bypassFileCheck(ctx, invalidFiles, hashtags, owner, fr), ShouldEqual, false)
		})
	})
}
