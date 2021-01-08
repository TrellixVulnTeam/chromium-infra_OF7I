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

func TestReviewCleanRevert(t *testing.T) {
	Convey("review benign file change", t, func() {
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
			Host:       "test-host",
			Number:     12345,
			Revision:   "123abc",
			Repo:       "dummy",
			AutoSubmit: false,
			RevertOf:   45678,
		}

		Convey("clean revert with no repo config is valid", func() {
			gerritMock.EXPECT().GetPureRevert(gomock.Any(), proto.MatcherEqual(&gerritpb.GetPureRevertRequest{
				Number:  t.Number,
				Project: t.Repo,
			})).Return(&gerritpb.PureRevertInfo{
				IsPureRevert: true,
			}, nil)
			gerritMock.EXPECT().GetChange(gomock.Any(), proto.MatcherEqual(&gerritpb.GetChangeRequest{
				Number:  t.RevertOf,
				Options: []gerritpb.QueryOption{gerritpb.QueryOption_CURRENT_REVISION},
			})).Return(&gerritpb.ChangeInfo{
				CurrentRevision: "456def",
				Revisions: map[string]*gerritpb.RevisionInfo{
					"456def": {
						Created: timestamppb.Now(),
					},
				},
			}, nil)
			msg, err := reviewCleanRevert(ctx, cfg, gerritMock, t)
			So(err, ShouldBeNil)
			So(msg, ShouldEqual, "")
		})
		Convey("clean revert with repo config is valid", func() {
			cfg.HostConfigs["test-host"].RepoConfigs["dummy"] = &config.RepoConfig{
				CleanRevertPattern: &config.CleanRevertPattern{
					TimeWindow:    "5m",
					ExcludedPaths: []string{"a/b/c.txt", "a/**/*.md"},
				},
			}
			gerritMock.EXPECT().GetPureRevert(gomock.Any(), proto.MatcherEqual(&gerritpb.GetPureRevertRequest{
				Number:  t.Number,
				Project: t.Repo,
			})).Return(&gerritpb.PureRevertInfo{
				IsPureRevert: true,
			}, nil)
			gerritMock.EXPECT().GetChange(gomock.Any(), proto.MatcherEqual(&gerritpb.GetChangeRequest{
				Number:  t.RevertOf,
				Options: []gerritpb.QueryOption{gerritpb.QueryOption_CURRENT_REVISION},
			})).Return(&gerritpb.ChangeInfo{
				CurrentRevision: "456def",
				Revisions: map[string]*gerritpb.RevisionInfo{
					"456def": {
						Created: timestamppb.New(time.Now().Add(-time.Minute)),
					},
				},
			}, nil)
			gerritMock.EXPECT().ListFiles(gomock.Any(), proto.MatcherEqual(&gerritpb.ListFilesRequest{
				Number:     t.Number,
				RevisionId: t.Revision,
			})).Return(&gerritpb.ListFilesResponse{
				Files: map[string]*gerritpb.FileInfo{
					"a/d/c.txt": nil,
					"a/valid.c": nil,
				},
			}, nil)
			msg, err := reviewCleanRevert(ctx, cfg, gerritMock, t)
			So(msg, ShouldEqual, "")
			So(err, ShouldBeNil)
		})
		Convey("invalid when gerrit GetPureRevert api returns false", func() {
			gerritMock.EXPECT().GetPureRevert(gomock.Any(), proto.MatcherEqual(&gerritpb.GetPureRevertRequest{
				Number:  t.Number,
				Project: t.Repo,
			})).Return(&gerritpb.PureRevertInfo{
				IsPureRevert: false,
			}, nil)
			msg, err := reviewCleanRevert(ctx, cfg, gerritMock, t)
			So(err, ShouldBeNil)
			So(msg, ShouldEqual, "Gerrit GetPureRevert API does not mark this CL as a pure revert.")
		})
		Convey("invalid when out of time window", func() {
			gerritMock.EXPECT().GetPureRevert(gomock.Any(), proto.MatcherEqual(&gerritpb.GetPureRevertRequest{
				Number:  t.Number,
				Project: t.Repo,
			})).Return(&gerritpb.PureRevertInfo{
				IsPureRevert: true,
			}, nil)
			Convey("global time window works", func() {
				gerritMock.EXPECT().GetChange(gomock.Any(), proto.MatcherEqual(&gerritpb.GetChangeRequest{
					Number:  t.RevertOf,
					Options: []gerritpb.QueryOption{gerritpb.QueryOption_CURRENT_REVISION},
				})).Return(&gerritpb.ChangeInfo{
					CurrentRevision: "456def",
					Revisions: map[string]*gerritpb.RevisionInfo{
						"456def": {
							Created: timestamppb.New(time.Now().Add(-8 * 24 * time.Hour)),
						},
					},
				}, nil)
				msg, err := reviewCleanRevert(ctx, cfg, gerritMock, t)
				So(err, ShouldBeNil)
				So(msg, ShouldEqual, "The change is not in the configured time window. Rubber Stamper is only allowed to review reverts within 7 day(s).")
			})
			Convey("host-level time window works", func() {
				cfg.HostConfigs["test-host"].CleanRevertTimeWindow = "5d"
				gerritMock.EXPECT().GetChange(gomock.Any(), proto.MatcherEqual(&gerritpb.GetChangeRequest{
					Number:  t.RevertOf,
					Options: []gerritpb.QueryOption{gerritpb.QueryOption_CURRENT_REVISION},
				})).Return(&gerritpb.ChangeInfo{
					CurrentRevision: "456def",
					Revisions: map[string]*gerritpb.RevisionInfo{
						"456def": {
							Created: timestamppb.New(time.Now().Add(-6 * 24 * time.Hour)),
						},
					},
				}, nil)
				msg, err := reviewCleanRevert(ctx, cfg, gerritMock, t)
				So(err, ShouldBeNil)
				So(msg, ShouldEqual, "The change is not in the configured time window. Rubber Stamper is only allowed to review reverts within 5 day(s).")
			})
			Convey("repo-level time window works", func() {
				cfg.HostConfigs["test-host"].CleanRevertTimeWindow = "5d"
				cfg.HostConfigs["test-host"].RepoConfigs["dummy"] = &config.RepoConfig{
					CleanRevertPattern: &config.CleanRevertPattern{
						TimeWindow: "5m",
					},
				}
				gerritMock.EXPECT().GetChange(gomock.Any(), proto.MatcherEqual(&gerritpb.GetChangeRequest{
					Number:  t.RevertOf,
					Options: []gerritpb.QueryOption{gerritpb.QueryOption_CURRENT_REVISION},
				})).Return(&gerritpb.ChangeInfo{
					CurrentRevision: "456def",
					Revisions: map[string]*gerritpb.RevisionInfo{
						"456def": {
							Created: timestamppb.New(time.Now().Add(-time.Hour)),
						},
					},
				}, nil)
				msg, err := reviewCleanRevert(ctx, cfg, gerritMock, t)
				So(err, ShouldBeNil)
				So(msg, ShouldEqual, "The change is not in the configured time window. Rubber Stamper is only allowed to review reverts within 5 minute(s).")
			})
		})
		Convey("invalid when contains excluded files", func() {
			cfg.HostConfigs["test-host"].RepoConfigs["dummy"] = &config.RepoConfig{
				CleanRevertPattern: &config.CleanRevertPattern{
					ExcludedPaths: []string{"a/b/c.txt", "a/**/*.md"},
				},
			}
			gerritMock.EXPECT().GetPureRevert(gomock.Any(), proto.MatcherEqual(&gerritpb.GetPureRevertRequest{
				Number:  t.Number,
				Project: t.Repo,
			})).Return(&gerritpb.PureRevertInfo{
				IsPureRevert: true,
			}, nil)
			gerritMock.EXPECT().GetChange(gomock.Any(), proto.MatcherEqual(&gerritpb.GetChangeRequest{
				Number:  t.RevertOf,
				Options: []gerritpb.QueryOption{gerritpb.QueryOption_CURRENT_REVISION},
			})).Return(&gerritpb.ChangeInfo{
				CurrentRevision: "456def",
				Revisions: map[string]*gerritpb.RevisionInfo{
					"456def": {
						Created: timestamppb.New(time.Now().Add(-2 * 24 * time.Hour)),
					},
				},
			}, nil)
			gerritMock.EXPECT().ListFiles(gomock.Any(), proto.MatcherEqual(&gerritpb.ListFilesRequest{
				Number:     t.Number,
				RevisionId: t.Revision,
			})).Return(&gerritpb.ListFilesResponse{
				Files: map[string]*gerritpb.FileInfo{
					"a/b/c.txt":  nil,
					"a/a/c/a.md": nil,
					"a/valid.c":  nil,
				},
			}, nil)
			msg, err := reviewCleanRevert(ctx, cfg, gerritMock, t)
			So(err, ShouldBeNil)
			So(msg, ShouldEqual, "The change contains the following files which require a human reviewer: a/a/c/a.md, a/b/c.txt")
		})
		Convey("returns error", func() {
			Convey("GetPureRevert API error", func() {
				gerritMock.EXPECT().GetPureRevert(gomock.Any(), proto.MatcherEqual(&gerritpb.GetPureRevertRequest{
					Number:  t.Number,
					Project: t.Repo,
				})).Return(nil, grpc.Errorf(codes.NotFound, "not found"))
				msg, err := reviewCleanRevert(ctx, cfg, gerritMock, t)
				So(msg, ShouldEqual, "")
				So(err, ShouldErrLike, "failed to call Gerrit GetPureRevert API")
			})
			Convey("time window config error", func() {
				cfg.HostConfigs["test-host"].CleanRevertTimeWindow = "1.2d"
				gerritMock.EXPECT().GetPureRevert(gomock.Any(), proto.MatcherEqual(&gerritpb.GetPureRevertRequest{
					Number:  t.Number,
					Project: t.Repo,
				})).Return(&gerritpb.PureRevertInfo{
					IsPureRevert: true,
				}, nil)
				msg, err := reviewCleanRevert(ctx, cfg, gerritMock, t)
				So(msg, ShouldEqual, "")
				So(err, ShouldErrLike, "invalid time_window config 1.2d")
			})
			Convey("ListFiles API error", func() {
				cfg.HostConfigs["test-host"].RepoConfigs["dummy"] = &config.RepoConfig{
					CleanRevertPattern: &config.CleanRevertPattern{
						ExcludedPaths: []string{"a/b/c.txt", "a/**/*.md"},
					},
				}
				gerritMock.EXPECT().GetPureRevert(gomock.Any(), proto.MatcherEqual(&gerritpb.GetPureRevertRequest{
					Number:  t.Number,
					Project: t.Repo,
				})).Return(&gerritpb.PureRevertInfo{
					IsPureRevert: true,
				}, nil)
				gerritMock.EXPECT().GetChange(gomock.Any(), proto.MatcherEqual(&gerritpb.GetChangeRequest{
					Number:  t.RevertOf,
					Options: []gerritpb.QueryOption{gerritpb.QueryOption_CURRENT_REVISION},
				})).Return(&gerritpb.ChangeInfo{
					CurrentRevision: "456def",
					Revisions: map[string]*gerritpb.RevisionInfo{
						"456def": {
							Created: timestamppb.New(time.Now().Add(-time.Minute)),
						},
					},
				}, nil)
				gerritMock.EXPECT().ListFiles(gomock.Any(), proto.MatcherEqual(&gerritpb.ListFilesRequest{
					Number:     t.Number,
					RevisionId: t.Revision,
				})).Return(nil, grpc.Errorf(codes.NotFound, "not found"))
				msg, err := reviewCleanRevert(ctx, cfg, gerritMock, t)
				So(msg, ShouldEqual, "")
				So(err, ShouldErrLike, "failed to call Gerrit ListFiles API")
			})
		})
	})
}
