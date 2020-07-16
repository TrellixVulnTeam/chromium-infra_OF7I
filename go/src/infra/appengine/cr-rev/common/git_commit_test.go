package common

import (
	"testing"

	"github.com/smartystreets/goconvey/convey"
)

func TestGitCommit(t *testing.T) {
	convey.Convey("ID generation", t, func() {
		commit := &GitCommit{
			Host:       "foo",
			Repository: "bar",
			Sha1:       "0000000000000000000000000000000000000000",
		}
		convey.So(
			commit.ID(),
			convey.ShouldEqual,
			"foo-bar-0000000000000000000000000000000000000000",
		)
	})

	convey.Convey("extract Footers", t, func() {
		convey.Convey("nothing to extract", func() {
			commit := &GitCommit{
				CommitMessage: "foo\n\nbar",
			}
			convey.So(commit.GetFooters("foo"), convey.ShouldBeEmpty)
			convey.So(commit.GetFooters("bar"), convey.ShouldBeEmpty)
			convey.So(commit.GetFooters("baz"), convey.ShouldBeEmpty)
		})

		convey.Convey("empty value", func() {
			commit := &GitCommit{
				CommitMessage: "foo\n\nbar:",
			}
			convey.So(commit.GetFooters("foo"), convey.ShouldBeEmpty)
			convey.So(
				commit.GetFooters("bar"),
				convey.ShouldResemble,
				[]string{""},
			)
		})

		convey.Convey("multiple values", func() {
			commit := &GitCommit{
				CommitMessage: "foo\n\nBar: 42\nBar: 43",
			}
			convey.So(
				commit.GetFooters("Bar"),
				convey.ShouldResemble,
				[]string{"42", "43"},
			)
		})

		convey.Convey("quoted values not extracted", func() {
			commit := &GitCommit{
				CommitMessage: "foo\n\n> Bar: 42\nBar: 43",
			}
			convey.So(
				commit.GetFooters("Bar"),
				convey.ShouldResemble,
				[]string{"43"},
			)
		})
	})

	convey.Convey("position footer", t, func() {
		convey.Convey("no number", func() {
			commit := &GitCommit{
				CommitMessage: "foo",
			}
			_, err := commit.GetPositionNumber()
			convey.So(
				err, convey.ShouldEqual, errNoPositionFooter)
		})

		convey.Convey("gitnumberer syntax", func() {
			commit := &GitCommit{
				CommitMessage: "foo\n\nCr-Commit-Position: refs/heads/master@{#42}",
			}
			position, err := commit.GetPositionNumber()
			convey.So(err, convey.ShouldBeNil)
			convey.So(position.Number, convey.ShouldEqual, 42)
			convey.So(
				position.Name, convey.ShouldEqual,
				"refs/heads/master")
		})

		convey.Convey("svn-id syntax", func() {
			commit := &GitCommit{
				CommitMessage: "foo\n\ngit-svn-id: svn://svn.chromium.org/chrome/trunk/src@42 00000000-0000-0000-0000-000000000000",
			}
			position, err := commit.GetPositionNumber()
			convey.So(err, convey.ShouldBeNil)
			convey.So(position.Number, convey.ShouldEqual, 42)
			convey.So(
				position.Name, convey.ShouldEqual,
				"svn://svn.chromium.org/chrome/trunk/src")
		})

		convey.Convey("with quoted text position", func() {
			commit := &GitCommit{
				CommitMessage: "foo\n\n>Cr-Commit-Position: refs/heads/master@{#42}",
			}
			_, err := commit.GetPositionNumber()
			convey.So(
				err, convey.ShouldEqual, errNoPositionFooter)
		})

		convey.Convey("gitnumberer syntax with quoted text", func() {
			commit := &GitCommit{
				CommitMessage: "foo\n\n>Cr-Commit-Position: refs/heads/foo@{#42}\nCr-Commit-Position: refs/heads/master@{#43}",
			}
			position, err := commit.GetPositionNumber()
			convey.So(err, convey.ShouldBeNil)
			convey.So(position.Number, convey.ShouldEqual, 43)
			convey.So(
				position.Name, convey.ShouldEqual,
				"refs/heads/master")
		})

		convey.Convey("gitnumberer syntax with quoted text2", func() {
			commit := &GitCommit{
				CommitMessage: "foo\n\nCr-Commit-Position: refs/heads/foo@{#42}\n>Cr-Commit-Position: refs/heads/master@{#43}",
			}
			position, err := commit.GetPositionNumber()
			convey.So(err, convey.ShouldBeNil)
			convey.So(position.Number, convey.ShouldEqual, 42)
			convey.So(
				position.Name, convey.ShouldEqual,
				"refs/heads/foo")
		})

		convey.Convey("multiuple gitnumberer", func() {
			commit := &GitCommit{
				CommitMessage: "foo\n\nCr-Commit-Position: refs/heads/foo@{#42}\nCr-Commit-Position: refs/heads/master@{#43}",
			}
			position, err := commit.GetPositionNumber()
			convey.So(err, convey.ShouldBeNil)
			convey.So(position.Number, convey.ShouldEqual, 43)
		})

		convey.Convey("invalid format", func() {
			commit := &GitCommit{
				CommitMessage: "foo\n\nCr-Commit-Position: foo",
			}
			_, err := commit.GetPositionNumber()
			convey.So(
				err, convey.ShouldEqual,
				errInvalidPositionFooter)
		})
	})
}
