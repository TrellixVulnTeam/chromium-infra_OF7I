// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package models

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/gae/service/datastore"
)

func TestRepoID(t *testing.T) {
	Convey("ToProperty", t, func() {
		Convey("nothing set", func() {
			rID := &RepoID{}
			_, err := rID.ToProperty()
			So(err, ShouldBeError)
		})
		Convey("host only", func() {
			rID := &RepoID{
				Host: "foo",
			}
			_, err := rID.ToProperty()
			So(err, ShouldBeError)
		})
		Convey("repository only", func() {
			rID := &RepoID{
				Repository: "foo",
			}
			_, err := rID.ToProperty()
			So(err, ShouldBeError)
		})
		Convey("minimal name", func() {
			rID := &RepoID{
				Host:       "f",
				Repository: "b",
			}
			ret, err := rID.ToProperty()
			So(err, ShouldBeNil)
			So(ret.Value(), ShouldEqual, "f/b")
		})
		Convey("with subrepos", func() {
			rID := &RepoID{
				Host:       "foo.example.org",
				Repository: "bar/baz/qux",
			}
			ret, err := rID.ToProperty()
			So(err, ShouldBeNil)
			So(ret.Value(), ShouldEqual, "foo.example.org/bar/baz/qux")
		})
	})

	Convey("FromProperty", t, func() {
		p := datastore.Property{}
		Convey("nothing set", func() {
			p.SetValue("", true)
			rID := &RepoID{}
			err := rID.FromProperty(p)

			So(err, ShouldBeError)
			So(rID.Host, ShouldBeZeroValue)
			So(rID.Repository, ShouldBeZeroValue)
		})
		Convey("host only", func() {
			p.SetValue("foo", true)
			rID := &RepoID{}
			err := rID.FromProperty(p)

			So(err, ShouldBeError)
			So(rID.Host, ShouldBeZeroValue)
			So(rID.Repository, ShouldBeZeroValue)
		})
		Convey("host only with slash", func() {
			p.SetValue("foo/", true)
			rID := &RepoID{}
			err := rID.FromProperty(p)

			So(err, ShouldBeError)
			So(rID.Host, ShouldBeZeroValue)
			So(rID.Repository, ShouldBeZeroValue)
		})
		Convey("repository only", func() {
			p.SetValue("/foo", true)
			rID := &RepoID{}
			err := rID.FromProperty(p)

			So(err, ShouldBeError)
			So(rID.Host, ShouldBeZeroValue)
			So(rID.Repository, ShouldBeZeroValue)
		})
		Convey("minimal valid name", func() {
			p.SetValue("f/b", true)
			rID := &RepoID{}
			err := rID.FromProperty(p)

			So(err, ShouldBeNil)
			So(rID.Host, ShouldEqual, "f")
			So(rID.Repository, ShouldEqual, "b")
		})
		Convey("with subrepos", func() {
			p.SetValue("foo.example.org/bar/baz/qux", true)
			rID := &RepoID{}
			err := rID.FromProperty(p)

			So(err, ShouldBeNil)
			So(rID.Host, ShouldEqual, "foo.example.org")
			So(rID.Repository, ShouldEqual, "bar/baz/qux")
		})
		Convey("invalid type", func() {
			p.SetValue(42, true)
			rID := &RepoID{}
			err := rID.FromProperty(p)

			So(err, ShouldBeError)
			So(rID.Host, ShouldBeZeroValue)
			So(rID.Repository, ShouldBeZeroValue)
		})
	})
}

func TestRepository(t *testing.T) {
	r := &Repository{}
	Convey("indexing", t, func() {
		currentTime := time.Now()
		Convey("not started", func() {
			So(
				r.IsFullScanStalled(currentTime),
				ShouldBeFalse,
			)
		})

		Convey("started", func() {
			r.FullScanLeaseStartTime = currentTime
			So(
				r.IsFullScanStalled(currentTime),
				ShouldBeFalse,
			)
		})

		Convey("expired", func() {
			r.FullScanLeaseStartTime = currentTime.Add(
				(RepositoryStaleIndexingDuration + time.Second) * -1)
			So(
				r.IsFullScanStalled(currentTime),
				ShouldBeFalse,
			)
		})

		Convey("completed", func() {
			r.SetIndexingCompleted(currentTime)
			So(
				r.IsFullScanStalled(currentTime),
				ShouldBeFalse,
			)
		})
	})

}
