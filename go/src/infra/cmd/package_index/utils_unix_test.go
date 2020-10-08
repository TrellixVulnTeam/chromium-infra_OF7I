// +build !windows

package main

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestConvertPathSlashesUnix(t *testing.T) {
	t.Parallel()
	Convey("Convert path with forward slashes", t, func() {
		p := "\\test\\path\\"

		Convey("On Unix", func() {
			Convey("The path should be the same", func() {
				So(convertPathToForwardSlashes(p), ShouldEqual, p)
			})
		})
	})
}
