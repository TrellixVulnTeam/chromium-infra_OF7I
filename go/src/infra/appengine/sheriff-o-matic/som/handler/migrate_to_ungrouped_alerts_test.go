package handler

import (
	"context"
	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/luci/appengine/gaetesting"
	"testing"
)

func fakepopulateAlertsNonGrouping(c context.Context) error {
	return nil
}

func TestMigrateToUngroupedAlerts(t *testing.T) {
	Convey("test migrate to ungrouped alerts should not run if autogrouping is off", t, func() {
		c := gaetesting.TestingContext()
		err := migrateToUngroupedAlerts(c, false, fakepopulateAlertsNonGrouping)
		So(err, ShouldNotBeNil)
	})

	Convey("test migrate to ungrouped alerts", t, func() {
		c := gaetesting.TestingContext()
		err := migrateToUngroupedAlerts(c, true, fakepopulateAlertsNonGrouping)
		So(err, ShouldBeNil)
	})
}
