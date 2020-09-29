// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package pubsub

import (
	"infra/appengine/cr-rev/backend/gitiles"
	"infra/appengine/cr-rev/config"
	"infra/appengine/cr-rev/models"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/luci/appengine/gaetesting"
	"go.chromium.org/luci/common/proto/git"
	gitilesProto "go.chromium.org/luci/common/proto/gitiles"
	"go.chromium.org/luci/gae/service/datastore"
)

func TestPubsubProcessor(t *testing.T) {
	ctx := gaetesting.TestingContext()
	ds := datastore.GetTestable(ctx)
	ds.Consistent(true)
	ds.AutoIndex(true)

	host := &config.Host{
		Name: "foo",
		Repos: []*config.Repository{
			{
				Name: "not-indexed",
				Indexing: &config.Repository_DoNotIndex{
					DoNotIndex: true,
				},
			},
			{
				Name:        "custom-refs",
				Refs:        []string{"refs/my-heads"},
				ExcludeRefs: []string{"refs/my-heads/exclude"},
			},
		},
	}
	p := Processor(host)
	Convey("invalid event name", t, func() {
		m := &SourceRepoEvent{
			Name: "invalid/name",
		}
		err := p(ctx, m)
		So(err, ShouldBeError)
	})

	Convey("skip not indexed", t, func() {
		m := &SourceRepoEvent{
			Name: "projects/foo/repos/not-indexed",
		}
		// We don't expect any gitiles calls, therefore we are not
		// setting gitiles fake to return anything.
		err := p(ctx, m)
		So(err, ShouldBeNil)
	})

	Convey("non update events", t, func() {
		m := &SourceRepoEvent{
			Name: "projects/foo/repos/bar",
			Event: &SourceRepoEvent_CreateRepoEvent_{
				CreateRepoEvent: &SourceRepoEvent_CreateRepoEvent{},
			},
		}
		// We don't expect any gitiles calls, therefore we are not
		// setting gitiles fake to return anything.
		err := p(ctx, m)
		So(err, ShouldBeNil)
	})

	Convey("valid events", t, func() {
		m := &SourceRepoEvent{
			Name: "projects/foo/repos/bar",
			Event: &SourceRepoEvent_RefUpdateEvent_{
				RefUpdateEvent: &SourceRepoEvent_RefUpdateEvent{
					RefUpdates: map[string]*SourceRepoEvent_RefUpdateEvent_RefUpdate{
						// indexed:
						"refs/heads/master": {
							RefName: "refs/heads/foo",
							OldId:   "000000000000000000000000000000000000000E",
							NewId:   "000000000000000000000000000000000000000F",
						},
						// not indexed:
						"refs/foo": {
							RefName: "refs/foo",
							OldId:   "0000000000000000000000000000000000000010",
							NewId:   "000000000000000000000000000000000000001F",
						},
					},
				},
			},
		}
		commits := []*git.Commit{
			{
				Id:      "000000000000000000000000000000000000000F",
				Parents: []string{"000000000000000000000000000000000000000E"},
			},
			{
				Id:      "000000000000000000000000000000000000000E",
				Parents: []string{"000000000000000000000000000000000000000D"},
			},
			{
				Id: "000000000000000000000000000000000000000D",
			},
			{
				Id: "000000000000000000000000000000000000001F",
			},
		}
		c := &gitilesProto.GitilesFake{}
		c.SetRepository("bar", nil, commits)
		ctx := gitiles.SetClient(ctx, c)
		err := p(ctx, m)
		So(err, ShouldBeNil)

		datastoreCommits := []*models.Commit{}
		q := datastore.NewQuery("Commit").Eq("Repository", "bar")
		datastore.GetAll(ctx, q, &datastoreCommits)
		So(len(datastoreCommits), ShouldEqual, 1)
		So(datastoreCommits[0].CommitHash, ShouldEqual, "000000000000000000000000000000000000000F")
	})

	Convey("respect include/exclude refs", t, func() {
		m := &SourceRepoEvent{
			Name: "projects/foo/repos/custom-refs",
			Event: &SourceRepoEvent_RefUpdateEvent_{
				RefUpdateEvent: &SourceRepoEvent_RefUpdateEvent{
					RefUpdates: map[string]*SourceRepoEvent_RefUpdateEvent_RefUpdate{
						// not-indexed:
						"refs/heads/master": {
							RefName: "refs/heads/master",
							OldId:   "000000000000000000000000000000000000000E",
							NewId:   "000000000000000000000000000000000000000F",
						},
						"refs/my-heads/excluded": {
							RefName: "refs/my-heads/exclude",
							OldId:   "000000000000000000000000000000000000000D",
							NewId:   "000000000000000000000000000000000000000E",
						},
						// indexed:
						"refs/my-heads/indexed": {
							RefName: "refs/my-heads/indexed",
							OldId:   "000000000000000000000000000000000000000C",
							NewId:   "000000000000000000000000000000000000000D",
						},
					},
				},
			},
		}
		commits := []*git.Commit{
			{
				Id:      "000000000000000000000000000000000000000F",
				Parents: []string{"000000000000000000000000000000000000000E"},
			},
			{
				Id:      "000000000000000000000000000000000000000E",
				Parents: []string{"000000000000000000000000000000000000000D"},
			},
			{
				Id:      "000000000000000000000000000000000000000D",
				Parents: []string{"000000000000000000000000000000000000000C"},
			},
			{
				Id: "000000000000000000000000000000000000000C",
			},
		}
		c := &gitilesProto.GitilesFake{}
		c.SetRepository("custom-refs", nil, commits)
		ctx := gitiles.SetClient(ctx, c)
		err := p(ctx, m)
		So(err, ShouldBeNil)

		datastoreCommits := []*models.Commit{}
		q := datastore.NewQuery("Commit").Eq("Repository", "custom-refs")
		datastore.GetAll(ctx, q, &datastoreCommits)
		So(len(datastoreCommits), ShouldEqual, 1)
		So(datastoreCommits[0].CommitHash, ShouldEqual, "000000000000000000000000000000000000000D")
	})

	Convey("ignore messages with no oldid", t, func() {
		m := &SourceRepoEvent{
			Name: "projects/foo/repos/oldid",
			Event: &SourceRepoEvent_RefUpdateEvent_{
				RefUpdateEvent: &SourceRepoEvent_RefUpdateEvent{
					RefUpdates: map[string]*SourceRepoEvent_RefUpdateEvent_RefUpdate{
						"refs/heads/master": {
							RefName: "refs/heads/master",
							OldId:   "",
							NewId:   "0000000000000000000000000000000000000001",
						},
					},
				},
			},
		}
		commits := []*git.Commit{
			{
				Id:      "0000000000000000000000000000000000000001",
				Parents: []string{"0000000000000000000000000000000000000000"},
			},
			{
				Id: "0000000000000000000000000000000000000000",
			},
		}
		c := &gitilesProto.GitilesFake{}
		c.SetRepository("oldid", nil, commits)
		ctx := gitiles.SetClient(ctx, c)
		err := p(ctx, m)
		So(err, ShouldBeError)

		datastoreCommits := []*models.Commit{}
		q := datastore.NewQuery("Commit").Eq("Repository", "oldid")
		datastore.GetAll(ctx, q, &datastoreCommits)
		So(len(datastoreCommits), ShouldEqual, 0)
	})
}
