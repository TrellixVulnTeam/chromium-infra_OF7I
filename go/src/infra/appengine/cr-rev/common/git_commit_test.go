package common

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGitCommit(t *testing.T) {
	Convey("ID generation", t, func() {
		commit := &GitCommit{
			Host:       "foo",
			Repository: "bar",
			Sha1:       "0000000000000000000000000000000000000000",
		}
		So(commit.ID(), ShouldEqual, "foo-bar-0000000000000000000000000000000000000000")
	})

	Convey("extract Footers", t, func() {
		Convey("nothing to extract", func() {
			commit := &GitCommit{
				CommitMessage: "foo\n\nbar",
			}
			So(commit.GetFooters("foo"), ShouldBeEmpty)
			So(commit.GetFooters("bar"), ShouldBeEmpty)
			So(commit.GetFooters("baz"), ShouldBeEmpty)
		})

		Convey("empty value", func() {
			commit := &GitCommit{
				CommitMessage: "foo\n\nbar:",
			}
			So(commit.GetFooters("foo"), ShouldBeEmpty)
			So(commit.GetFooters("bar"), ShouldResemble, []string{""})
		})

		Convey("multiple values", func() {
			commit := &GitCommit{
				CommitMessage: "foo\n\nBar: 42\nBar: 43",
			}
			So(commit.GetFooters("Bar"), ShouldResemble, []string{"42", "43"})
		})

		Convey("quoted values not extracted", func() {
			commit := &GitCommit{
				CommitMessage: "foo\n\n> Bar: 42\nBar: 43",
			}
			So(commit.GetFooters("Bar"), ShouldResemble, []string{"43"})
		})
	})

	Convey("position footer", t, func() {
		Convey("no number", func() {
			commit := &GitCommit{
				CommitMessage: "foo",
			}
			_, err := commit.GetPositionNumber()
			So(err, ShouldEqual, errNoPositionFooter)
		})

		Convey("gitnumberer syntax", func() {
			commit := &GitCommit{
				CommitMessage: "foo\n\nCr-Commit-Position: refs/heads/master@{#42}",
			}
			position, err := commit.GetPositionNumber()
			So(err, ShouldBeNil)
			So(position.Number, ShouldEqual, 42)
			So(position.Name, ShouldEqual, "refs/heads/master")
		})

		Convey("svn-id syntax", func() {
			commit := &GitCommit{
				CommitMessage: "foo\n\ngit-svn-id: svn://svn.chromium.org/chrome/trunk/src@42 00000000-0000-0000-0000-000000000000",
			}
			position, err := commit.GetPositionNumber()
			So(err, ShouldBeNil)
			So(position.Number, ShouldEqual, 42)
			So(position.Name, ShouldEqual, "svn://svn.chromium.org/chrome/trunk/src")
		})

		Convey("with quoted text position", func() {
			commit := &GitCommit{
				CommitMessage: "foo\n\n>Cr-Commit-Position: refs/heads/master@{#42}",
			}
			_, err := commit.GetPositionNumber()
			So(err, ShouldEqual, errNoPositionFooter)
		})

		Convey("gitnumberer syntax with quoted text", func() {
			commit := &GitCommit{
				CommitMessage: "foo\n\n>Cr-Commit-Position: refs/heads/foo@{#42}\nCr-Commit-Position: refs/heads/master@{#43}",
			}
			position, err := commit.GetPositionNumber()
			So(err, ShouldBeNil)
			So(position.Number, ShouldEqual, 43)
			So(position.Name, ShouldEqual, "refs/heads/master")
		})

		Convey("gitnumberer syntax with quoted text2", func() {
			commit := &GitCommit{
				CommitMessage: "foo\n\nCr-Commit-Position: refs/heads/foo@{#42}\n>Cr-Commit-Position: refs/heads/master@{#43}",
			}
			position, err := commit.GetPositionNumber()
			So(err, ShouldBeNil)
			So(position.Number, ShouldEqual, 42)
			So(position.Name, ShouldEqual, "refs/heads/foo")
		})

		Convey("multiuple gitnumberer", func() {
			commit := &GitCommit{
				CommitMessage: "foo\n\nCr-Commit-Position: refs/heads/foo@{#42}\nCr-Commit-Position: refs/heads/master@{#43}",
			}
			position, err := commit.GetPositionNumber()
			So(err, ShouldBeNil)
			So(position.Number, ShouldEqual, 43)
		})

		Convey("invalid format", func() {
			commit := &GitCommit{
				CommitMessage: "foo\n\nCr-Commit-Position: foo",
			}
			_, err := commit.GetPositionNumber()
			So(err, ShouldEqual, errInvalidPositionFooter)
		})
	})
}
