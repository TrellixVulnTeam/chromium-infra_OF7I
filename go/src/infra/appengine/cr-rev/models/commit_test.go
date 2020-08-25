package models

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestCommit(t *testing.T) {
	Convey("Compare repos", t, func() {
		Convey("Empty repo", func() {
			c1 := Commit{}
			c2 := Commit{}
			So(c1.SameRepoAs(c1), ShouldBeTrue)
			So(c1.SameRepoAs(c2), ShouldBeTrue)
		})
		Convey("Identical", func() {
			c1 := Commit{Host: "foo", Repository: "bar"}
			c2 := Commit{Host: "foo", Repository: "bar"}
			So(c1.SameRepoAs(c2), ShouldBeTrue)
		})
		Convey("Different hosts", func() {
			c1 := Commit{Host: "foo", Repository: "bar"}
			c2 := Commit{Host: "baz", Repository: "bar"}
			So(c1.SameRepoAs(c2), ShouldBeFalse)
		})
	})
}
