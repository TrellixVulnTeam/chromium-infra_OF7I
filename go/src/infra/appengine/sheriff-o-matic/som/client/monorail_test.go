package client

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestMonorail(t *testing.T) {
	Convey("Test ParseMonorailIssueName", t, func() {
		Convey("valid issue name", func() {
			issueName := "projects/chromium/issues/123"
			projectID, bugID, err := ParseMonorailIssueName(issueName)
			So(projectID, ShouldEqual, "chromium")
			So(bugID, ShouldEqual, "123")
			So(err, ShouldBeNil)
		})
		Convey("invalid issue name", func() {
			issueName := "invalid"
			_, _, err := ParseMonorailIssueName(issueName)
			So(err, ShouldNotBeNil)
		})
	})
}
