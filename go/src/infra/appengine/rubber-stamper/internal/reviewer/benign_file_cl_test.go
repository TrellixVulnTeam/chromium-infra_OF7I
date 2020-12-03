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
			BenignFilePattern: &config.BenignFilePattern{
				FileExtensionMap: map[string]*config.Paths{
					"": {
						Paths: []string{"a/x", "a/q/y"},
					},
					".txt": {
						Paths: []string{"a/b.txt", "a/c.txt", "a/e/*", "a/f*"},
					},
				},
			},
		}

		t := &taskspb.ChangeReviewTask{
			Host:     "test-host",
			Number:   12345,
			Revision: "123abc",
		}
		Convey("valid file", func() {
			gerritMock.EXPECT().ListFiles(gomock.Any(), proto.MatcherEqual(&gerritpb.ListFilesRequest{
				Number:     t.Number,
				RevisionId: t.Revision,
			})).Return(&gerritpb.ListFilesResponse{
				Files: map[string]*gerritpb.FileInfo{
					"a/b.txt":   nil,
					"a/c.txt":   nil,
					"a/q/y":     nil,
					"a/e/a.txt": nil,
					"a/fz.txt":  nil,
				},
			}, nil)

			invalidFiles, err := reviewBenignFileChange(ctx, hostCfg, gerritMock, t)
			So(err, ShouldBeNil)
			So(len(invalidFiles), ShouldEqual, 0)
		})
		Convey("missing config", func() {
			hostCfg.BenignFilePattern = nil
			gerritMock.EXPECT().ListFiles(gomock.Any(), proto.MatcherEqual(&gerritpb.ListFilesRequest{
				Number:     t.Number,
				RevisionId: t.Revision,
			})).Return(&gerritpb.ListFilesResponse{
				Files: map[string]*gerritpb.FileInfo{
					"a/b.txt": nil,
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
					"a/b.md": nil,
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
	})
}
