package main

import (
	"infra/appengine/cr-rev/common"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGitilesRedirect(t *testing.T) {
	commit := common.GitCommit{
		Host:       "foo.googlesource.com",
		Repository: "bar/baz",
		Sha1:       "1234567890123456789012345678901234567890",
	}
	redirect := &gitilesRedirect{}
	Convey("Commit redirect", t, func() {
		Convey("no path provided", func() {
			url, err := redirect.commit(commit, "")
			So(err, ShouldBeNil)
			So(
				url,
				ShouldEqual,
				"https://foo.googlesource.com/bar/baz/+/1234567890123456789012345678901234567890",
			)
		})

		Convey("path provided", func() {
			url, err := redirect.commit(commit, "README.md")
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
			commit2 := common.GitCommit{
				Host:       "foo.googlesource.com",
				Repository: "bar/baz",
				Sha1:       "0000000000000000000000000000000000000000",
			}
			url, err := redirect.diff(commit, commit2)
			So(err, ShouldBeNil)
			So(
				url,
				ShouldEqual,
				"https://foo.googlesource.com/bar/baz/+/1234567890123456789012345678901234567890...0000000000000000000000000000000000000000",
			)
		})

		Convey("different repositories", func() {
			commit2 := common.GitCommit{
				Host:       "foo.googlesource.com",
				Repository: "bar/baz/baq",
				Sha1:       "0000000000000000000000000000000000000000",
			}
			_, err := redirect.diff(commit, commit2)
			So(err, ShouldEqual, errNotIdenticalRepositories)
		})

		Convey("different host repositories", func() {
			commit2 := common.GitCommit{
				Host:       "bar.googlesource.com",
				Repository: "bar/baz",
				Sha1:       "0000000000000000000000000000000000000000",
			}
			_, err := redirect.diff(commit, commit2)
			So(err, ShouldEqual, errNotIdenticalRepositories)
		})
	})
}

func TestCodesearchRedirect(t *testing.T) {
	redirect := &codesearchRedirect{}
	Convey("Test commit redirect", t, func() {
		commit := common.GitCommit{
			Host:       "chromium.googlesource.com",
			Repository: "bar/baz",
			Sha1:       "1234567890123456789012345678901234567890",
		}
		Convey("no path provided", func() {
			url, err := redirect.commit(commit, "")
			So(err, ShouldBeNil)
			So(
				url,
				ShouldEqual,
				"https://source.chromium.org/chromium/bar/baz/+/1234567890123456789012345678901234567890",
			)
		})

		Convey("path provided", func() {
			url, err := redirect.commit(commit, "README.md")
			So(err, ShouldBeNil)
			So(
				url,
				ShouldEqual,
				"https://source.chromium.org/chromium/bar/baz/+/1234567890123456789012345678901234567890:README.md",
			)
		})
	})

	Convey("Diff redirect", t, func() {
		commit := common.GitCommit{
			Host:       "chromium.googlesource.com",
			Repository: "bar/baz",
			Sha1:       "1234567890123456789012345678901234567890",
		}
		Convey("identical repositories", func() {
			commit2 := common.GitCommit{
				Host:       "chromium.googlesource.com",
				Repository: "bar/baz",
				Sha1:       "0000000000000000000000000000000000000000",
			}
			url, err := redirect.diff(commit, commit2)
			So(err, ShouldBeNil)
			So(
				url,
				ShouldEqual,
				"https://source.chromium.org/chromium/bar/baz/+/1234567890123456789012345678901234567890...0000000000000000000000000000000000000000",
			)
		})

		Convey("different repositories", func() {
			commit2 := common.GitCommit{
				Host:       "chromium.googlesource.com",
				Repository: "bar/baz/baq",
				Sha1:       "0000000000000000000000000000000000000000",
			}
			_, err := redirect.diff(commit, commit2)
			So(err, ShouldEqual, errNotIdenticalRepositories)
		})

		Convey("different host repositories", func() {
			commit2 := common.GitCommit{
				Host:       "foo.googlesource.com",
				Repository: "bar/baz",
				Sha1:       "0000000000000000000000000000000000000000",
			}
			_, err := redirect.diff(commit, commit2)
			So(err, ShouldEqual, errNotIdenticalRepositories)
		})
	})

	Convey("Test not supported GoB hosts", t, func() {
		commit := common.GitCommit{
			Host:       "foo",
			Repository: "bar/baz",
		}
		Convey("no commit redirect", func() {
			_, err := redirect.commit(commit, "")
			So(err, ShouldEqual, errNotSupportedRepository)
		})
		Convey("no diff redirect", func() {
			commit2 := common.GitCommit{
				Host:       "foo",
				Repository: "bar/baz",
			}
			_, err := redirect.diff(commit, commit2)
			So(err, ShouldEqual, errNotSupportedRepository)
		})
	})
}
