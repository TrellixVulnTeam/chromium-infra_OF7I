package main

import (
	"infra/appengine/cr-rev/common"
	"testing"

	"github.com/smartystreets/goconvey/convey"
)

func TestGitilesRedirect(t *testing.T) {
	commit := common.GitCommit{
		Host:       "foo.googlesource.com",
		Repository: "bar/baz",
		Sha1:       "1234567890123456789012345678901234567890",
	}
	redirect := &gitilesRedirect{}
	convey.Convey("Commit redirect", t, func() {
		convey.Convey("no path provided", func() {
			url, err := redirect.commit(commit, "")
			convey.So(err, convey.ShouldBeNil)
			convey.So(
				url,
				convey.ShouldEqual,
				"https://foo.googlesource.com/bar/baz/+/1234567890123456789012345678901234567890",
			)
		})

		convey.Convey("path provided", func() {
			url, err := redirect.commit(commit, "README.md")
			convey.So(err, convey.ShouldBeNil)
			convey.So(
				url,
				convey.ShouldEqual,
				"https://foo.googlesource.com/bar/baz/+/1234567890123456789012345678901234567890/README.md",
			)
		})
	})

	convey.Convey("Diff redirect", t, func() {
		convey.Convey("identical repositories", func() {
			commit2 := common.GitCommit{
				Host:       "foo.googlesource.com",
				Repository: "bar/baz",
				Sha1:       "0000000000000000000000000000000000000000",
			}
			url, err := redirect.diff(commit, commit2)
			convey.So(err, convey.ShouldBeNil)
			convey.So(
				url,
				convey.ShouldEqual,
				"https://foo.googlesource.com/bar/baz/+/1234567890123456789012345678901234567890...0000000000000000000000000000000000000000",
			)
		})

		convey.Convey("different repositories", func() {
			commit2 := common.GitCommit{
				Host:       "foo.googlesource.com",
				Repository: "bar/baz/baq",
				Sha1:       "0000000000000000000000000000000000000000",
			}
			_, err := redirect.diff(commit, commit2)
			convey.So(
				err,
				convey.ShouldEqual,
				errNotIdenticalRepositories,
			)
		})

		convey.Convey("different host repositories", func() {
			commit2 := common.GitCommit{
				Host:       "bar.googlesource.com",
				Repository: "bar/baz",
				Sha1:       "0000000000000000000000000000000000000000",
			}
			_, err := redirect.diff(commit, commit2)
			convey.So(
				err,
				convey.ShouldEqual,
				errNotIdenticalRepositories,
			)
		})
	})
}

func TestCodesearchRedirect(t *testing.T) {
	redirect := &codesearchRedirect{}
	convey.Convey("Test commit redirect", t, func() {
		commit := common.GitCommit{
			Host:       "chromium.googlesource.com",
			Repository: "bar/baz",
			Sha1:       "1234567890123456789012345678901234567890",
		}
		convey.Convey("no path provided", func() {
			url, err := redirect.commit(commit, "")
			convey.So(err, convey.ShouldBeNil)
			convey.So(
				url,
				convey.ShouldEqual,
				"https://source.chromium.org/chromium/bar/baz/+/1234567890123456789012345678901234567890",
			)
		})

		convey.Convey("path provided", func() {
			url, err := redirect.commit(commit, "README.md")
			convey.So(err, convey.ShouldBeNil)
			convey.So(
				url,
				convey.ShouldEqual,
				"https://source.chromium.org/chromium/bar/baz/+/1234567890123456789012345678901234567890:README.md",
			)
		})
	})

	convey.Convey("Diff redirect", t, func() {
		commit := common.GitCommit{
			Host:       "chromium.googlesource.com",
			Repository: "bar/baz",
			Sha1:       "1234567890123456789012345678901234567890",
		}
		convey.Convey("identical repositories", func() {
			commit2 := common.GitCommit{
				Host:       "chromium.googlesource.com",
				Repository: "bar/baz",
				Sha1:       "0000000000000000000000000000000000000000",
			}
			url, err := redirect.diff(commit, commit2)
			convey.So(err, convey.ShouldBeNil)
			convey.So(
				url,
				convey.ShouldEqual,
				"https://source.chromium.org/chromium/bar/baz/+/1234567890123456789012345678901234567890...0000000000000000000000000000000000000000",
			)
		})

		convey.Convey("different repositories", func() {
			commit2 := common.GitCommit{
				Host:       "chromium.googlesource.com",
				Repository: "bar/baz/baq",
				Sha1:       "0000000000000000000000000000000000000000",
			}
			_, err := redirect.diff(commit, commit2)
			convey.So(err, convey.ShouldEqual, errNotIdenticalRepositories)
		})

		convey.Convey("different host repositories", func() {
			commit2 := common.GitCommit{
				Host:       "foo.googlesource.com",
				Repository: "bar/baz",
				Sha1:       "0000000000000000000000000000000000000000",
			}
			_, err := redirect.diff(commit, commit2)
			convey.So(
				err,
				convey.ShouldEqual,
				errNotIdenticalRepositories,
			)
		})
	})

	convey.Convey("Test not supported GoB hosts", t, func() {
		commit := common.GitCommit{
			Host:       "foo",
			Repository: "bar/baz",
		}
		convey.Convey("no commit redirect", func() {
			_, err := redirect.commit(commit, "")
			convey.So(
				err,
				convey.ShouldEqual,
				errNotSupportedRepository,
			)
		})
		convey.Convey("no diff redirect", func() {
			commit2 := common.GitCommit{
				Host:       "foo",
				Repository: "bar/baz",
			}
			_, err := redirect.diff(commit, commit2)
			convey.So(
				err,
				convey.ShouldEqual,
				errNotSupportedRepository,
			)
		})
	})
}
