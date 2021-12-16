// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package control

import (
	"context"
	"testing"
	"time"

	"go.chromium.org/luci/server/span"

	"infra/appengine/weetbix/internal/testutil"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func TestSpan(t *testing.T) {
	Convey(`With Spanner Test Database`, t, func() {
		ctx := testutil.SpannerTestContext(t)
		Convey(`Read`, func() {
			entriesToCreate := []*Entry{
				NewEntry(0).WithBuildID("buildbucket-instance/1").Build(),
				NewEntry(1).WithBuildID("buildbucket-instance/2").WithProject("otherproject").Build(),
				NewEntry(2).WithBuildID("buildbucket-instance/3").WithBuildResult(nil).Build(),
				NewEntry(3).WithBuildID("buildbucket-instance/4").WithPresubmitResult(nil).Build(),
			}
			_, err := SetEntriesForTesting(ctx, entriesToCreate)
			So(err, ShouldBeNil)

			Convey(`None exist`, func() {
				buildIDs := []string{"buildbucket-instance/5"}
				results, err := Read(span.Single(ctx), testProject, buildIDs)
				So(err, ShouldBeNil)
				So(len(results), ShouldEqual, 1)
				EntryShouldResemble(results[0], nil)
			})
			Convey(`Some exist`, func() {
				buildIDs := []string{"buildbucket-instance/3", "buildbucket-instance/4", "buildbucket-instance/2", "buildbucket-instance/1"}
				results, err := Read(span.Single(ctx), testProject, buildIDs)
				So(err, ShouldBeNil)
				So(len(results), ShouldEqual, 4)
				EntryShouldResemble(results[0], entriesToCreate[2])
				EntryShouldResemble(results[1], entriesToCreate[3])
				EntryShouldResemble(results[2], nil)
				EntryShouldResemble(results[3], entriesToCreate[0])
			})
		})
		Convey(`InsertOrUpdate`, func() {
			testInsertOrUpdate := func(e *Entry) (time.Time, error) {
				commitTime, err := span.ReadWriteTransaction(ctx, func(ctx context.Context) error {
					return InsertOrUpdate(ctx, e)
				})
				return commitTime.In(time.UTC), err
			}

			e := NewEntry(100).WithPresubmitResult(nil).Build()
			Convey(`Valid`, func() {
				// Create for first time.
				commitTime, err := testInsertOrUpdate(e)
				So(err, ShouldBeNil)
				e.LastUpdated = commitTime

				result, err := Read(span.Single(ctx), testProject, []string{e.BuildID})
				So(err, ShouldBeNil)
				So(len(result), ShouldEqual, 1)
				EntryShouldResemble(result[0], e)

				// Update.
				e2 := NewEntry(100).Build()
				commitTime, err = testInsertOrUpdate(e2)
				So(err, ShouldBeNil)
				e2.LastUpdated = commitTime

				result, err = Read(span.Single(ctx), testProject, []string{e.BuildID})
				So(err, ShouldBeNil)
				So(len(result), ShouldEqual, 1)
				EntryShouldResemble(result[0], e2)
			})
			Convey(`With invalid Project`, func() {
				Convey(`Missing`, func() {
					e.Project = ""
					_, err := testInsertOrUpdate(e)
					So(err, ShouldErrLike, "project must be valid")
				})
				Convey(`Invalid`, func() {
					e.Project = "!"
					_, err := testInsertOrUpdate(e)
					So(err, ShouldErrLike, "project must be valid")
				})
			})
			Convey(`With missing Build ID`, func() {
				e.BuildID = ""
				_, err := testInsertOrUpdate(e)
				So(err, ShouldErrLike, "build ID must be specified")
			})
			Convey(`With invalid Build Result`, func() {
				Convey(`Missing host`, func() {
					e.BuildResult.Host = ""
					_, err := testInsertOrUpdate(e)
					So(err, ShouldErrLike, "host must be specified")
				})
				Convey(`Missing id`, func() {
					e.BuildResult.Id = 0
					_, err := testInsertOrUpdate(e)
					So(err, ShouldErrLike, "id must be specified")
				})
				Convey(`Missing creation time`, func() {
					e.BuildResult.CreationTime = nil
					_, err := testInsertOrUpdate(e)
					So(err, ShouldErrLike, "build result: creation time must be specified")
				})
			})
			Convey(`With invalid Presubmit Result`, func() {
				e = NewEntry(100).Build()
				Convey(`Missing Presubmit run ID`, func() {
					e.PresubmitResult.PresubmitRunId = nil
					_, err := testInsertOrUpdate(e)
					So(err, ShouldErrLike, "presubmit run ID must be specified")
				})
				Convey(`Invalid Presubmit run ID host`, func() {
					e.PresubmitResult.PresubmitRunId.System = "!"
					_, err := testInsertOrUpdate(e)
					So(err, ShouldErrLike, "presubmit run system must be 'luci-cv'")
				})
				Convey(`Missing Presubmit run ID system-specific ID`, func() {
					e.PresubmitResult.PresubmitRunId.Id = ""
					_, err := testInsertOrUpdate(e)
					So(err, ShouldErrLike, "presubmit run system-specific ID must be specified")
				})
				Convey(`Missing creation time`, func() {
					e.PresubmitResult.CreationTime = nil
					_, err := testInsertOrUpdate(e)
					So(err, ShouldErrLike, "presubmit result: creation time must be specified")
				})
			})
		})
	})
}

func EntryShouldResemble(actual *Entry, expected *Entry) {
	if expected == nil {
		So(actual, ShouldBeNil)
		return
	}
	// Check equality of non-proto fields.
	a := *actual
	e := *expected
	a.BuildResult = nil
	a.PresubmitResult = nil
	e.BuildResult = nil
	e.PresubmitResult = nil
	So(e, ShouldResemble, a)

	// Check equality of proto fields.
	So(expected.BuildResult, ShouldResembleProto, actual.BuildResult)
	So(expected.PresubmitResult, ShouldResembleProto, actual.PresubmitResult)
}
