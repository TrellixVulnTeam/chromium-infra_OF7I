// Copyright 2021 The Chromium Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package datastore_dao

import (
	"context"
	"testing"
	"time"

	"infra/chromeperf/alert_groups/model"

	"go.chromium.org/luci/gae/impl/memory"
	"go.chromium.org/luci/gae/service/datastore"

	. "github.com/smartystreets/goconvey/convey"
)

func TestAlertGroupModel(t *testing.T) {
	t.Parallel()

	Convey("Given DAO objects and AlertGroup structs that are stored in Datastore", t, func() {
		ctx := memory.Use(context.Background())
		var alertGroupDao model.AlertGroupDAO = AlertGroupDAO{}

		now := time.Now().Round(time.Microsecond).UTC()
		sourceGroup1 := &model.AlertGroup{
			ID:   "uuid",
			Name: "group1",
			Revision: model.RevisionRange{
				Repository: "test",
				Start:      0,
				End:        1,
			},
			Created: now,
			Updated: now,
			Anomalies: []*datastore.Key{
				datastore.MakeKey(ctx, "Anomaly", 1),
				datastore.MakeKey(ctx, "Anomaly", 2),
			},
		}
		err := datastore.Put(ctx, sourceGroup1)
		So(err, ShouldBeNil)

		now = time.Now().Round(time.Microsecond).UTC()
		sourceGroup2 := &model.AlertGroup{
			ID:      "uuid2",
			Name:    "group2",
			Created: now,
			Updated: now,
		}
		err = datastore.Put(ctx, sourceGroup2)
		So(err, ShouldBeNil)

		Convey("Alert group can retreived by key with DAO", func() {
			storedGroup1, err := alertGroupDao.Get(ctx, sourceGroup1.ID)
			So(err, ShouldBeNil)
			So(storedGroup1, ShouldResemble, sourceGroup1)

			storedGroup2, err := alertGroupDao.Get(ctx, sourceGroup2.ID)
			So(err, ShouldBeNil)
			So(storedGroup2, ShouldResemble, sourceGroup2)

			Convey("Alert group can be updated with DAO", func() {
				storedGroup1.Anomalies = storedGroup1.Anomalies[:len(storedGroup1.Anomalies)-1]
				err = alertGroupDao.Update(ctx, storedGroup1)
				So(err, ShouldBeNil)

				storedGroup2.Anomalies = append(storedGroup2.Anomalies, datastore.MakeKey(ctx, "Anomaly", 2))
				err = alertGroupDao.Update(ctx, storedGroup2)
				So(err, ShouldBeNil)

				Convey("Update is reflected in datastore", func() {
					datastoreGroup1, err := alertGroupDao.Get(ctx, storedGroup1.ID)
					So(err, ShouldBeNil)
					So(len(datastoreGroup1.Anomalies), ShouldEqual, 1)
					So(datastoreGroup1.Anomalies[0], ShouldResemble, datastore.MakeKey(ctx, "Anomaly", 1))
					So(datastoreGroup1.Created.Before(datastoreGroup1.Updated), ShouldBeTrue)

					datastoreGroup2, err := alertGroupDao.Get(ctx, storedGroup2.ID)
					So(err, ShouldBeNil)
					So(len(datastoreGroup2.Anomalies), ShouldEqual, 1)
					So(datastoreGroup2.Anomalies[0], ShouldResemble, datastore.MakeKey(ctx, "Anomaly", 2))
					So(datastoreGroup2.Created.Before(datastoreGroup2.Updated), ShouldBeTrue)
				})
			})

			Convey("Alert group can be deleted with DAO", func() {
				err = alertGroupDao.Delete(ctx, storedGroup1)
				So(err, ShouldBeNil)

				deletedGroup, err := alertGroupDao.Get(ctx, storedGroup1.ID)
				So(err, ShouldNotBeNil)
				So(deletedGroup, ShouldBeNil)

				notDeletedGroup, err := alertGroupDao.Get(ctx, storedGroup2.ID)
				So(err, ShouldBeNil)
				So(notDeletedGroup, ShouldNotBeNil)
			})
		})
	})
}
