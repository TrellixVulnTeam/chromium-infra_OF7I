package repoimport

import (
	"context"
	"infra/appengine/cr-rev/common"
	"infra/appengine/cr-rev/models"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/luci/appengine/gaetesting"
	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/clock/testclock"
	"go.chromium.org/luci/gae/service/datastore"
)

func TestLeaser(t *testing.T) {
	repo := common.GitRepository{
		Host: "foo",
		Name: "bar",
	}

	doc := &models.Repository{
		ID: models.RepoID{
			Host:       "foo",
			Repository: "bar",
		},
	}

	prepareEnvironment := func() (context.Context, *leaser) {
		ctx := gaetesting.TestingContext()
		ds := datastore.GetTestable(ctx)
		ds.Consistent(true)
		ds.AutoIndex(true)

		testclock := testclock.New(time.Now())
		ctx = clock.Set(ctx, testclock)

		return ctx, newLeaser(repo)
	}
	Convey("Lease", t, func() {
		Convey("non existing lock", func() {
			ctx, leaser := prepareEnvironment()
			err := leaser.refreshLease(ctx)
			So(err, ShouldBeError)
		})

		Convey("valid renew", func() {
			ctx, leaser := prepareEnvironment()
			err := leaser.acquireLease(ctx)
			So(err, ShouldBeNil)
			err = leaser.refreshLease(ctx)
			So(err, ShouldBeNil)
		})

		Convey("lock claimed by another process", func() {
			ctx, leaser := prepareEnvironment()
			err := leaser.acquireLease(ctx)
			So(err, ShouldBeNil)

			// Override lock
			datastore.Put(ctx, doc)
			err = leaser.refreshLease(ctx)
			So(err, ShouldBeError)
		})
	})
}
