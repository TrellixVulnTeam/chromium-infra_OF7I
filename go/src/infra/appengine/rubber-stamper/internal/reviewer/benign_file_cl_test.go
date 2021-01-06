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

	"infra/appengine/rubber-stamper/config"
	"infra/appengine/rubber-stamper/tasks/taskspb"
)

func TestReviewBenignFileChange(t *testing.T) {
	Convey("review benign file change", t, func() {
		ctx := memory.Use(context.Background())

		ctl := gomock.NewController(t)
		defer ctl.Finish()
		gerritMock := gerritpb.NewMockGerritClient(ctl)

		hostCfg := &config.HostConfig{
			RepoConfigs: map[string]*config.RepoConfig{
				"dummy": {
					BenignFilePattern: &config.BenignFilePattern{
						Paths: []string{
							"test/a/*",
							"test/b/c.txt",
							"test/c/**",
							"test/override/**",
							"!test/override/**",
							"test/override/a/**",
							"!test/override/a/b/*",
						},
						FileExtensionMap: map[string]*config.Paths{
							"": {
								Paths: []string{"a/x", "a/q/y"},
							},
							"*": {
								Paths: []string{"t/*"},
							},
							".txt": {
								Paths: []string{"a/b.txt", "a/c.txt", "a/e/*", "a/f*"},
							},
							".xtb": {
								Paths: []string{"**"},
							},
						},
					},
				},
			},
		}

		t := &taskspb.ChangeReviewTask{
			Host:       "test-host",
			Number:     12345,
			Revision:   "123abc",
			Repo:       "dummy",
			AutoSubmit: false,
		}

		Convey("all paths valid for a particular extension ", func() {
			gerritMock.EXPECT().ListFiles(gomock.Any(), proto.MatcherEqual(&gerritpb.ListFilesRequest{
				Number:     t.Number,
				RevisionId: t.Revision,
			})).Return(&gerritpb.ListFilesResponse{
				Files: map[string]*gerritpb.FileInfo{
					"/COMMIT_MSG": nil,
					"a/b.xtb":     nil,
					"d.xtb":       nil,
					"f/e/g.xtb":   nil,
					"t/o":         nil,
					"t/readme.md": nil,
				},
			}, nil)
			invalidFiles, err := reviewBenignFileChange(ctx, hostCfg, gerritMock, t)
			So(err, ShouldBeNil)
			So(len(invalidFiles), ShouldEqual, 0)
		})
		Convey("valid file", func() {
			gerritMock.EXPECT().ListFiles(gomock.Any(), proto.MatcherEqual(&gerritpb.ListFilesRequest{
				Number:     t.Number,
				RevisionId: t.Revision,
			})).Return(&gerritpb.ListFilesResponse{
				Files: map[string]*gerritpb.FileInfo{
					"/COMMIT_MSG": nil,
					"a/b.txt":     nil,
					"a/c.txt":     nil,
					"a/q/y":       nil,
					"a/e/a.txt":   nil,
					"a/fz.txt":    nil,
					"t/asd.txt":   nil,
					"t/q":         nil,
				},
			}, nil)

			invalidFiles, err := reviewBenignFileChange(ctx, hostCfg, gerritMock, t)
			So(err, ShouldBeNil)
			So(len(invalidFiles), ShouldEqual, 0)
		})
		Convey("missing config", func() {
			hostCfg.RepoConfigs["dummy"].BenignFilePattern = nil
			gerritMock.EXPECT().ListFiles(gomock.Any(), proto.MatcherEqual(&gerritpb.ListFilesRequest{
				Number:     t.Number,
				RevisionId: t.Revision,
			})).Return(&gerritpb.ListFilesResponse{
				Files: map[string]*gerritpb.FileInfo{
					"/COMMIT_MSG": nil,
					"a/b.txt":     nil,
				},
			}, nil)

			invalidFiles, err := reviewBenignFileChange(ctx, nil, gerritMock, t)
			So(err, ShouldBeNil)
			So(invalidFiles, ShouldResemble, []string{"a/b.txt"})
		})
		Convey("invalid file extension", func() {
			gerritMock.EXPECT().ListFiles(gomock.Any(), proto.MatcherEqual(&gerritpb.ListFilesRequest{
				Number:     t.Number,
				RevisionId: t.Revision,
			})).Return(&gerritpb.ListFilesResponse{
				Files: map[string]*gerritpb.FileInfo{
					"/COMMIT_MSG": nil,
					"a/b.md":      nil,
				},
			}, nil)

			invalidFiles, err := reviewBenignFileChange(ctx, hostCfg, gerritMock, t)
			So(err, ShouldBeNil)
			So(invalidFiles, ShouldResemble, []string{"a/b.md"})
		})
		Convey("invalid file", func() {
			gerritMock.EXPECT().ListFiles(gomock.Any(), proto.MatcherEqual(&gerritpb.ListFilesRequest{
				Number:     t.Number,
				RevisionId: t.Revision,
			})).Return(&gerritpb.ListFilesResponse{
				Files: map[string]*gerritpb.FileInfo{
					"/COMMIT_MSG": nil,
					"a/d.txt":     nil,
					"a/p":         nil,
					"a/e/p/p.txt": nil,
					"a/f/z.txt":   nil,
					"a/fz.txt":    nil,
				},
			}, nil)

			invalidFiles, err := reviewBenignFileChange(ctx, hostCfg, gerritMock, t)
			So(err, ShouldBeNil)
			So(invalidFiles, ShouldResemble, []string{"a/d.txt", "a/e/p/p.txt", "a/f/z.txt", "a/p"})
		})
		Convey("valid files with gitignore style patterns", func() {
			gerritMock.EXPECT().ListFiles(gomock.Any(), proto.MatcherEqual(&gerritpb.ListFilesRequest{
				Number:     t.Number,
				RevisionId: t.Revision,
			})).Return(&gerritpb.ListFilesResponse{
				Files: map[string]*gerritpb.FileInfo{
					"/COMMIT_MSG":   nil,
					"test/a/b.xtb":  nil,
					"test/b/c.txt":  nil,
					"test/c/pp.xtb": nil,
					"test/c/i/a.md": nil,
				},
			}, nil)
			invalidFiles, err := reviewBenignFileChange(ctx, hostCfg, gerritMock, t)
			So(err, ShouldBeNil)
			So(len(invalidFiles), ShouldEqual, 0)
		})
		Convey("gitigore style patterns' order matterns", func() {
			gerritMock.EXPECT().ListFiles(gomock.Any(), proto.MatcherEqual(&gerritpb.ListFilesRequest{
				Number:     t.Number,
				RevisionId: t.Revision,
			})).Return(&gerritpb.ListFilesResponse{
				Files: map[string]*gerritpb.FileInfo{
					"/COMMIT_MSG":             nil,
					"test/override/1.txt":     nil,
					"test/override/a/2.txt":   nil,
					"test/override/a/b/3.txt": nil,
					"test/override/a/c/4.txt": nil,
					"test/override/ab/5.txt":  nil,
				},
			}, nil)
			invalidFiles, err := reviewBenignFileChange(ctx, hostCfg, gerritMock, t)
			So(err, ShouldBeNil)
			So(invalidFiles, ShouldResemble, []string{"test/override/1.txt", "test/override/a/b/3.txt", "test/override/ab/5.txt"})
		})
	})
}
