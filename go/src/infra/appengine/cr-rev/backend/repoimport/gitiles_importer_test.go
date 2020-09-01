// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package repoimport

import (
	"context"
	"infra/appengine/cr-rev/backend/gitiles"
	"infra/appengine/cr-rev/common"
	"infra/appengine/cr-rev/models"
	"strconv"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/luci/appengine/gaetesting"
	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/clock/testclock"
	"go.chromium.org/luci/common/proto/git"
	gitilesProto "go.chromium.org/luci/common/proto/gitiles"
	"go.chromium.org/luci/gae/service/datastore"
)

func TestGitilesImporter(t *testing.T) {
	repo := common.GitRepository{
		Host: "foo",
		Name: "bar",
	}

	doc := &models.Repository{
		ID: models.RepoID{
			Host:       "foo",
			Repository: "bar",
		},
	}

	prepareEnvironment := func() (context.Context, *gitilesProto.GitilesFake, importer) {
		ctx := gaetesting.TestingContext()
		ds := datastore.GetTestable(ctx)
		ds.Consistent(true)
		ds.AutoIndex(true)

		testclock := testclock.New(time.Now())
		ctx = clock.Set(ctx, testclock)

		client := &gitilesProto.GitilesFake{}
		ctx = gitiles.SetClient(ctx, client)

		imp := newGitilesImporter(ctx, repo)
		return ctx, client, imp
	}

	assertCommitDocuments := func(ctx context.Context, expected int) []*models.Commit {
		dsCommits := []*models.Commit{}
		q := datastore.NewQuery("Commit")
		datastore.GetAll(ctx, q, &dsCommits)
		So(len(dsCommits), ShouldEqual, expected)
		return dsCommits
	}

	Convey("non existing repository", t, func() {
		ctx, _, importer := prepareEnvironment()
		err := importer.Run(ctx)
		So(err, ShouldBeError)
		So(err.Error(), ShouldEqual, "Repository not found")
		// Datastore should not have lock anymore, and should unset last run
		datastore.Get(ctx, doc)
		So(doc.FullScanLeaseStartTime, ShouldEqual, time.Time{})
		So(doc.FullScanLastRun, ShouldEqual, time.Time{})
	})

	Convey("existing repository", t, func() {
		Convey("empty repository", func() {
			ctx, client, importer := prepareEnvironment()

			client.SetRepository("bar", map[string]string{}, []*git.Commit{})
			err := importer.Run(ctx)
			So(err, ShouldBeNil)

			// Datastore should not have lock anymore, and last run should be set
			datastore.Get(ctx, doc)
			So(doc.FullScanLeaseStartTime, ShouldEqual, time.Time{})
			So(doc.FullScanLastRun, ShouldEqual, clock.Get(ctx).Now().UTC().Round(time.Millisecond))
		})

		Convey("empty default branch", func() {
			ctx, client, importer := prepareEnvironment()

			refs := map[string]string{
				"main": "",
			}
			commits := []*git.Commit{}
			client.SetRepository("bar", refs, commits)
			err := importer.Run(ctx)
			So(err, ShouldBeNil)
			// Datastore should not have lock anymore, and last run should be set
			datastore.Get(ctx, doc)
			So(doc.FullScanLeaseStartTime, ShouldEqual, time.Time{})
			So(doc.FullScanLastRun, ShouldEqual, clock.Get(ctx).Now().UTC().Round(time.Millisecond))
			assertCommitDocuments(ctx, 0)
		})

		Convey("one commit, two branches", func() {
			ctx, client, importer := prepareEnvironment()

			refs := map[string]string{
				"main":    "0000000000000000000000000000000000000000",
				"release": "0000000000000000000000000000000000000000",
			}
			commits := []*git.Commit{
				{
					Id: "0000000000000000000000000000000000000000",
					Message: `Commit message

Bug: 123
Change-Id: Ifoo
Cr-Commit-Position: refs/heads/main@{#1}`,
				},
			}
			client.SetRepository("bar", refs, commits)
			err := importer.Run(ctx)
			So(err, ShouldBeNil)

			datastore.Get(ctx, doc)
			So(doc.FullScanLeaseStartTime, ShouldEqual, time.Time{})
			So(doc.FullScanLastRun, ShouldEqual, clock.Get(ctx).Now().UTC().Round(time.Millisecond))
			docs := assertCommitDocuments(ctx, 1)
			So(docs[0].PositionRef, ShouldEqual, "refs/heads/main")
			So(docs[0].PositionNumber, ShouldEqual, 1)
		})

		Convey("diverged branches", func() {
			ctx, client, importer := prepareEnvironment()

			refs := map[string]string{
				"main":    "5",
				"release": "2",
			}
			commits := make([]*git.Commit, 5)
			for i := 0; i < 5; i++ {
				commits[i] = &git.Commit{
					Id: strconv.Itoa(i + 1),
				}
				if i > 0 {
					commits[i].Parents = []string{commits[i-1].GetId()}
				}
			}
			client.SetRepository("bar", refs, commits)
			err := importer.Run(ctx)
			So(err, ShouldBeNil)

			datastore.Get(ctx, doc)
			So(doc.FullScanLeaseStartTime, ShouldEqual, time.Time{})
			So(doc.FullScanLastRun, ShouldEqual, clock.Get(ctx).Now().UTC().Round(time.Millisecond))
			assertCommitDocuments(ctx, 5)
		})

		Convey("Log commit caching", func() {
			ctx, client, importer := prepareEnvironment()

			// Require two Log pages for each branch
			refs := map[string]string{
				"main":    strconv.Itoa(gitilesLogPageSize + 2),
				"release": strconv.Itoa(gitilesLogPageSize + 1),
			}
			commits := make([]*git.Commit, gitilesLogPageSize+2)
			for i := 0; i < gitilesLogPageSize+2; i++ {
				commits[i] = &git.Commit{
					Id: strconv.Itoa(i + 1),
				}
				if i > 0 {
					commits[i].Parents = []string{commits[i-1].GetId()}
				}
			}
			client.SetRepository("bar", refs, commits)
			err := importer.Run(ctx)
			So(err, ShouldBeNil)

			datastore.Get(ctx, doc)
			So(doc.FullScanLeaseStartTime, ShouldEqual, time.Time{})
			So(doc.FullScanLastRun, ShouldEqual, clock.Get(ctx).Now().UTC().Round(time.Millisecond))
			assertCommitDocuments(ctx, gitilesLogPageSize+2)
			// We expect 4 calls.
			// One is for listing all refs and their revisions.
			// Very first indexed branch should make two calls to
			// Gitiles, and the last should make only one.
			So(len(client.GetCallLogs()), ShouldEqual, 4)
		})
	})
}
