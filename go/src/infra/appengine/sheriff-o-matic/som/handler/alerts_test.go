package handler

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/appengine/gaetesting"
	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/clock/testclock"
	"infra/appengine/sheriff-o-matic/som/model"
)

func TestFlushAlerts(t *testing.T) {
	Convey("test flush alerts", t, func() {
		c := gaetesting.TestingContext()
		cl := testclock.New(testclock.TestRecentTimeUTC)
		c = clock.Set(c, cl)
		now := clock.Get(c).Now()

		alertIdx := datastore.IndexDefinition{
			Kind: "AlertJSONNonGrouping",
			SortBy: []datastore.IndexColumn{
				{
					Property:   "Resolved",
					Descending: false,
				},
				{
					Property:   "ResolvedDate",
					Descending: false,
				},
			},
		}

		indexes := []*datastore.IndexDefinition{&alertIdx}
		datastore.GetTestable(c).AddIndexes(indexes...)

		alerts := []*model.AlertJSONNonGrouping{
			{
				ID:           "alert1",
				Resolved:     false,
				ResolvedDate: now.Add(-time.Hour * 24 * 8),
			},
			{
				ID:           "alert2",
				Resolved:     true,
				ResolvedDate: now.Add(-time.Hour * 24 * 8),
			},
			{
				ID:           "alert3",
				Resolved:     true,
				ResolvedDate: now.Add(-time.Hour * 24 * 6),
			},
		}

		So(datastore.Put(c, alerts), ShouldBeNil)
		datastore.GetTestable(c).CatchupIndexes()
		num, err := flushOldAlerts(c)
		So(err, ShouldBeNil)
		So(num, ShouldEqual, 1)

		q := datastore.NewQuery("AlertJSONNonGrouping")
		result := []*model.AlertJSONNonGrouping{}
		err = datastore.GetAll(c, q, &result)
		So(err, ShouldBeNil)
		So(len(result), ShouldEqual, 2)
		So(result[0].ID, ShouldEqual, "alert1")
		So(result[1].ID, ShouldEqual, "alert3")
	})
}
