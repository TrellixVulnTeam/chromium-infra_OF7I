// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package reviewer

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/luci/common/proto"
	gerritpb "go.chromium.org/luci/common/proto/gerrit"
	"go.chromium.org/luci/gae/impl/memory"
	"google.golang.org/protobuf/types/known/timestamppb"

	"infra/appengine/rubber-stamper/config"
	"infra/appengine/rubber-stamper/internal/util"
	"infra/appengine/rubber-stamper/tasks/taskspb"
)

func TestReviewChange(t *testing.T) {
	Convey("review change", t, func() {
		cfg := &config.Config{
			DefaultTimeWindow: "7d",
			HostConfigs: map[string]*config.HostConfig{
				"test-host": {
					RepoConfigs: map[string]*config.RepoConfig{
						"dummy": {
							BenignFilePattern: &config.BenignFilePattern{
								Paths: []string{"a/x", "a/b.txt", "a/c.txt", "a/e/*.txt", "a/f*.txt"},
							},
						},
					},
				},
			},
		}
		ctx := memory.Use(context.Background())
		ctx, gerritMock, _ := util.SetupTestingContext(ctx, cfg, "srv-account@example.com", "test-host", t)

		Convey("BenignFileChange", func() {
			t := &taskspb.ChangeReviewTask{
				Host:       "test-host",
				Number:     12345,
				Revision:   "123abc",
				Repo:       "dummy",
				AutoSubmit: false,
			}
			Convey("valid BenignFileChange", func() {
				gerritMock.EXPECT().ListFiles(gomock.Any(), proto.MatcherEqual(&gerritpb.ListFilesRequest{
					Number:     t.Number,
					RevisionId: t.Revision,
				})).Return(&gerritpb.ListFilesResponse{
					Files: map[string]*gerritpb.FileInfo{
						"a/x": nil,
					},
				}, nil)
				gerritMock.EXPECT().SetReview(gomock.Any(), proto.MatcherEqual(&gerritpb.SetReviewRequest{
					Number:     t.Number,
					RevisionId: t.Revision,
					Labels:     map[string]int32{"Bot-Commit": 1},
				})).Return(&gerritpb.ReviewResult{}, nil)

				err := ReviewChange(ctx, t)
				So(err, ShouldBeNil)
			})
			Convey("valid BenignFileChange with Auto-Submit", func() {
				t.AutoSubmit = true
				gerritMock.EXPECT().ListFiles(gomock.Any(), proto.MatcherEqual(&gerritpb.ListFilesRequest{
					Number:     t.Number,
					RevisionId: t.Revision,
				})).Return(&gerritpb.ListFilesResponse{
					Files: map[string]*gerritpb.FileInfo{
						"a/x": nil,
					},
				}, nil)
				gerritMock.EXPECT().SetReview(gomock.Any(), proto.MatcherEqual(&gerritpb.SetReviewRequest{
					Number:     t.Number,
					RevisionId: t.Revision,
					Labels:     map[string]int32{"Bot-Commit": 1, "Commit-Queue": 2},
				})).Return(&gerritpb.ReviewResult{}, nil)

				err := ReviewChange(ctx, t)
				So(err, ShouldBeNil)
			})
			Convey("invalid BenignFileChange", func() {
				gerritMock.EXPECT().ListFiles(gomock.Any(), proto.MatcherEqual(&gerritpb.ListFilesRequest{
					Number:     t.Number,
					RevisionId: t.Revision,
				})).Return(&gerritpb.ListFilesResponse{
					Files: map[string]*gerritpb.FileInfo{
						"a/d.txt":     nil,
						"a/p":         nil,
						"a/e/p/p.txt": nil,
						"a/f/z.txt":   nil,
						"a/fz.txt":    nil,
					},
				}, nil)
				gerritMock.EXPECT().SetReview(gomock.Any(), proto.MatcherEqual(&gerritpb.SetReviewRequest{
					Number:     t.Number,
					RevisionId: t.Revision,
					Message:    "The change cannot be auto-reviewed. The following files do not match the benign file configuration: a/d.txt, a/e/p/p.txt, a/f/z.txt, a/p",
				})).Return(&gerritpb.ReviewResult{}, nil)
				gerritMock.EXPECT().DeleteReviewer(gomock.Any(), proto.MatcherEqual(&gerritpb.DeleteReviewerRequest{
					Number:    t.Number,
					AccountId: "srv-account@example.com",
				})).Return(nil, nil)

				err := ReviewChange(ctx, t)
				So(err, ShouldBeNil)
			})
		})
		Convey("CleanRevert", func() {
			t := &taskspb.ChangeReviewTask{
				Host:       "test-host",
				Number:     12345,
				Revision:   "123abc",
				Repo:       "dummy",
				AutoSubmit: false,
				RevertOf:   45678,
			}
			Convey("valid CleanRevert", func() {
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
				gerritMock.EXPECT().SetReview(gomock.Any(), proto.MatcherEqual(&gerritpb.SetReviewRequest{
					Number:     t.Number,
					RevisionId: t.Revision,
					Labels:     map[string]int32{"Bot-Commit": 1},
				})).Return(&gerritpb.ReviewResult{}, nil)

				err := ReviewChange(ctx, t)
				So(err, ShouldBeNil)
			})
			Convey("invalid CleanRevert but can pass the BenignFilePattern", func() {
				gerritMock.EXPECT().GetPureRevert(gomock.Any(), proto.MatcherEqual(&gerritpb.GetPureRevertRequest{
					Number:  t.Number,
					Project: t.Repo,
				})).Return(&gerritpb.PureRevertInfo{
					IsPureRevert: false,
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
				gerritMock.EXPECT().ListFiles(gomock.Any(), proto.MatcherEqual(&gerritpb.ListFilesRequest{
					Number:     t.Number,
					RevisionId: t.Revision,
				})).Return(&gerritpb.ListFilesResponse{
					Files: map[string]*gerritpb.FileInfo{
						"a/x": nil,
					},
				}, nil)
				gerritMock.EXPECT().SetReview(gomock.Any(), proto.MatcherEqual(&gerritpb.SetReviewRequest{
					Number:     t.Number,
					RevisionId: t.Revision,
					Labels:     map[string]int32{"Bot-Commit": 1},
				})).Return(&gerritpb.ReviewResult{}, nil)

				err := ReviewChange(ctx, t)
				So(err, ShouldBeNil)
			})
			Convey("invalid CleanRevert", func() {
				gerritMock.EXPECT().GetPureRevert(gomock.Any(), proto.MatcherEqual(&gerritpb.GetPureRevertRequest{
					Number:  t.Number,
					Project: t.Repo,
				})).Return(&gerritpb.PureRevertInfo{
					IsPureRevert: false,
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
				gerritMock.EXPECT().ListFiles(gomock.Any(), proto.MatcherEqual(&gerritpb.ListFilesRequest{
					Number:     t.Number,
					RevisionId: t.Revision,
				})).Return(&gerritpb.ListFilesResponse{
					Files: map[string]*gerritpb.FileInfo{
						"a/d.txt":     nil,
						"a/p":         nil,
						"a/e/p/p.txt": nil,
						"a/f/z.txt":   nil,
						"a/fz.txt":    nil,
					},
				}, nil)
				gerritMock.EXPECT().SetReview(gomock.Any(), proto.MatcherEqual(&gerritpb.SetReviewRequest{
					Number:     t.Number,
					RevisionId: t.Revision,
					Message:    "Gerrit GetPureRevert API does not mark this CL as a pure revert.",
				})).Return(&gerritpb.ReviewResult{}, nil)
				gerritMock.EXPECT().DeleteReviewer(gomock.Any(), proto.MatcherEqual(&gerritpb.DeleteReviewerRequest{
					Number:    t.Number,
					AccountId: "srv-account@example.com",
				})).Return(nil, nil)

				err := ReviewChange(ctx, t)
				So(err, ShouldBeNil)
			})
		})
	})
}
