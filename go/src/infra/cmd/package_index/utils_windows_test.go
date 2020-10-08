// +build windows

package main

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestConvertPathSlashesWindows(t *testing.T) {
	Convey("Convert path with forward slashes", t, func() {
		p := "\\test\\path\\"

		Convey("On Windows", func() {
			r := convertPathToForwardSlashes(p)

			Convey("The path should have forward slashes", func() {
				So(r, ShouldEqual, "/test/path/")
			})
		})
	})
}
