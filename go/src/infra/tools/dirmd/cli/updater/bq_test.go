// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package updater

import (
	"context"
	"sort"
	"sync"
	"testing"

	"google.golang.org/protobuf/types/known/timestamppb"

	"go.chromium.org/luci/common/bq"
	"go.chromium.org/luci/common/clock/testclock"

	"infra/tools/dirmd"
	dirmdpb "infra/tools/dirmd/proto"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

type mockInserter struct {
	insertedMessages []*bq.Row
	mu               sync.Mutex
}

func (i *mockInserter) Put(ctx context.Context, src interface{}) error {
	messages := src.([]*bq.Row)
	i.mu.Lock()
	i.insertedMessages = append(i.insertedMessages, messages...)
	i.mu.Unlock()
	return nil
}

func TestBqExport(t *testing.T) {
	t.Parallel()

	Convey("BqExport", t, func() {
		Convey("generateSchema", func() {
			_, err := generateSchema()
			So(err, ShouldBeNil)
		})
		mapping := &dirmd.Mapping{
			Dirs: map[string]*dirmdpb.Metadata{
				".": {
					TeamEmail: "chromium-review@chromium.org",
					Os:        dirmdpb.OS_LINUX,
				},
				"a": {
					TeamEmail: "team-email@chromium.org",
					Os:        dirmdpb.OS_LINUX,
					Monorail: &dirmdpb.Monorail{
						Project:   "chromium",
						Component: "Some>Component",
					},
				},
				"a/b": {
					TeamEmail: "team-email@chromium.org",
					Os:        dirmdpb.OS_LINUX,
					Monorail: &dirmdpb.Monorail{
						Project:   "chromium",
						Component: "Some>Component",
					},
					Wpt: &dirmdpb.WPT{
						Notify: dirmdpb.Trinary_YES,
					},
				},
				"v8/a/b": {
					TeamEmail: "team-email@chromium.org",
					Os:        dirmdpb.OS_LINUX,
					Monorail: &dirmdpb.Monorail{
						Project:   "chromium",
						Component: "Some>Component",
					},
				},
			},
			Repos: map[string]*dirmdpb.Repo{
				".": {
					Mixins: map[string]*dirmdpb.Metadata{
						"//mixin1": {
							Monorail: &dirmdpb.Monorail{
								Project: "chromium",
							},
						},
						"//mixin2": {
							Buganizer: &dirmdpb.Buganizer{ComponentId: 54},
						},
					},
				},
				"v8": {
					Mixins: map[string]*dirmdpb.Metadata{
						"//mixin1": {
							Monorail: &dirmdpb.Monorail{
								Project: "v8",
							},
						},
					},
				},
			},
		}
		Convey("success", func() {
			ctx, _ := testclock.UseTime(context.Background(), testclock.TestRecentTimeUTC)
			i := &mockInserter{}
			commit := &GitCommit{
				Host:     "host",
				Project:  chromiumProject,
				Ref:      "ref",
				Revision: "revision",
			}
			So(writeToBQ(ctx, i, mapping, commit), ShouldBeNil)

			pt := timestamppb.New(testclock.TestRecentTimeUTC)
			expected := []*dirmdpb.DirBQRow{
				{
					Source: &dirmdpb.Source{
						GitHost:  commit.Host,
						RootRepo: commit.Project,
						SubRepo:  "",
						Ref:      commit.Ref,
						Revision: commit.Revision,
					},
					Dir:                  ".",
					TeamEmail:            "chromium-review@chromium.org",
					Os:                   dirmdpb.OS_LINUX,
					TeamSpecificMetadata: &dirmdpb.TeamSpecific{},
					PartitionTime:        pt,
				},
				{
					Source: &dirmdpb.Source{
						GitHost:  commit.Host,
						RootRepo: commit.Project,
						SubRepo:  "",
						Ref:      commit.Ref,
						Revision: commit.Revision,
					},
					Dir: "a",
					Monorail: &dirmdpb.Monorail{
						Project:   "chromium",
						Component: "Some>Component",
					},
					TeamEmail:            "team-email@chromium.org",
					Os:                   dirmdpb.OS_LINUX,
					TeamSpecificMetadata: &dirmdpb.TeamSpecific{},
					PartitionTime:        pt,
				},
				{
					Source: &dirmdpb.Source{
						GitHost:  commit.Host,
						RootRepo: commit.Project,
						SubRepo:  "",
						Ref:      commit.Ref,
						Revision: commit.Revision,
					},
					Dir: "a/b",
					Monorail: &dirmdpb.Monorail{
						Project:   "chromium",
						Component: "Some>Component",
					},
					TeamEmail: "team-email@chromium.org",
					Os:        dirmdpb.OS_LINUX,
					TeamSpecificMetadata: &dirmdpb.TeamSpecific{
						Wpt: &dirmdpb.WPT{
							Notify: dirmdpb.Trinary_YES,
						},
					},
					PartitionTime: pt,
				},
				{
					Source: &dirmdpb.Source{
						GitHost:  commit.Host,
						RootRepo: commit.Project,
						SubRepo:  "v8",
						Ref:      commit.Ref,
						Revision: commit.Revision,
					},
					Dir: "v8/a/b",
					Monorail: &dirmdpb.Monorail{
						Project:   "chromium",
						Component: "Some>Component",
					},
					TeamEmail:            "team-email@chromium.org",
					Os:                   dirmdpb.OS_LINUX,
					TeamSpecificMetadata: &dirmdpb.TeamSpecific{},
					PartitionTime:        pt,
				},
			}
			i.mu.Lock()
			defer i.mu.Unlock()
			actual := make([]*dirmdpb.DirBQRow, len(i.insertedMessages))
			for n, m := range i.insertedMessages {
				actual[n] = m.Message.(*dirmdpb.DirBQRow)
			}
			sort.Slice(actual, func(i, j int) bool {
				return actual[i].Dir < actual[j].Dir
			})
			So(actual, ShouldResembleProto, expected)
		})
	})
}
