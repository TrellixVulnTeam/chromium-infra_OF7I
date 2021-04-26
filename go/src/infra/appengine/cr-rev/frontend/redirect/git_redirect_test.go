// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package redirect

import (
	"infra/appengine/cr-rev/models"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGitilesRedirect(t *testing.T) {
	commit := models.Commit{
		Host:       "foo",
		Repository: "bar/baz",
		CommitHash: "1234567890123456789012345678901234567890",
	}
	redirect := &gitilesRedirect{}
	Convey("Commit redirect", t, func() {
		Convey("no path provided", func() {
			url, err := redirect.Commit(commit, "")
			So(err, ShouldBeNil)
			So(
				url,
				ShouldEqual,
				"https://foo.googlesource.com/bar/baz/+/1234567890123456789012345678901234567890",
			)
		})

		Convey("path provided", func() {
			url, err := redirect.Commit(commit, "README.md")
			So(err, ShouldBeNil)
			So(
				url,
				ShouldEqual,
				"https://foo.googlesource.com/bar/baz/+/1234567890123456789012345678901234567890/README.md",
			)
		})
	})

	Convey("Diff redirect", t, func() {
		Convey("identical repositories", func() {
			commit2 := models.Commit{
				Host:       "foo",
				Repository: "bar/baz",
				CommitHash: "0000000000000000000000000000000000000000",
			}
			url, err := redirect.Diff(commit, commit2)
			So(err, ShouldBeNil)
			So(
				url,
				ShouldEqual,
				"https://foo.googlesource.com/bar/baz/+/1234567890123456789012345678901234567890..0000000000000000000000000000000000000000",
			)
		})

		Convey("different repositories", func() {
			commit2 := models.Commit{
				Host:       "foo",
				Repository: "bar/baz/baq",
				CommitHash: "0000000000000000000000000000000000000000",
			}
			_, err := redirect.Diff(commit, commit2)
			So(err, ShouldEqual, errNotIdenticalRepositories)
		})

		Convey("different host repositories", func() {
			commit2 := models.Commit{
				Host:       "bar",
				Repository: "bar/baz",
				CommitHash: "0000000000000000000000000000000000000000",
			}
			_, err := redirect.Diff(commit, commit2)
			So(err, ShouldEqual, errNotIdenticalRepositories)
		})
	})
}

func TestCodesearchRedirect(t *testing.T) {
	redirect := &codesearchRedirect{}
	Convey("Test commit redirect", t, func() {
		commit := models.Commit{
			Host:       "chromium",
			Repository: "bar/baz",
			CommitHash: "1234567890123456789012345678901234567890",
		}
		Convey("no path provided", func() {
			url, err := redirect.Commit(commit, "")
			So(err, ShouldBeNil)
			So(
				url,
				ShouldEqual,
				"https://source.chromium.org/chromium/bar/baz/+/1234567890123456789012345678901234567890",
			)
		})

		Convey("path provided", func() {
			url, err := redirect.Commit(commit, "README.md")
			So(err, ShouldBeNil)
			So(
				url,
				ShouldEqual,
				"https://source.chromium.org/chromium/bar/baz/+/1234567890123456789012345678901234567890:README.md",
			)
		})
	})

	Convey("Diff redirect", t, func() {
		commit := models.Commit{
			Host:       "chromium",
			Repository: "bar/baz",
			CommitHash: "1234567890123456789012345678901234567890",
		}
		Convey("identical repositories", func() {
			commit2 := models.Commit{
				Host:       "chromium",
				Repository: "bar/baz",
				CommitHash: "0000000000000000000000000000000000000000",
			}
			url, err := redirect.Diff(commit, commit2)
			So(err, ShouldBeNil)
			So(
				url,
				ShouldEqual,
				"https://source.chromium.org/chromium/bar/baz/+/1234567890123456789012345678901234567890...0000000000000000000000000000000000000000",
			)
		})

		Convey("different repositories", func() {
			commit2 := models.Commit{
				Host:       "chromium",
				Repository: "bar/baz/baq",
				CommitHash: "0000000000000000000000000000000000000000",
			}
			_, err := redirect.Diff(commit, commit2)
			So(err, ShouldEqual, errNotIdenticalRepositories)
		})

		Convey("different host repositories", func() {
			commit2 := models.Commit{
				Host:       "foo",
				Repository: "bar/baz",
				CommitHash: "0000000000000000000000000000000000000000",
			}
			_, err := redirect.Diff(commit, commit2)
			So(err, ShouldEqual, errNotIdenticalRepositories)
		})
	})

	Convey("Test not supported GoB hosts", t, func() {
		commit := models.Commit{
			Host:       "foo",
			Repository: "bar/baz",
		}
		Convey("no commit redirect", func() {
			_, err := redirect.Commit(commit, "")
			So(err, ShouldEqual, errNotSupportedRepository)
		})
		Convey("no diff redirect", func() {
			commit2 := models.Commit{
				Host:       "foo",
				Repository: "bar/baz",
			}
			_, err := redirect.Diff(commit, commit2)
			So(err, ShouldEqual, errNotSupportedRepository)
		})
	})
}
