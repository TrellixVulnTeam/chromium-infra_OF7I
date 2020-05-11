package handler

import (
	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/luci/appengine/gaetesting"
	"testing"
)

func TestMigrateToUngroupedAlerts(t *testing.T) {
	Convey("test migrate to ungrouped alerts should not run if autogrouping is off", t, func() {
		c := gaetesting.TestingContext()
		err := migrateToUngroupedAlerts(c, false)
		So(err, ShouldNotBeNil)
	})

	Convey("test migrate to ungrouped alerts", t, func() {
		c := gaetesting.TestingContext()
		err := migrateToUngroupedAlerts(c, true)
		So(err, ShouldBeNil)
	})
}
