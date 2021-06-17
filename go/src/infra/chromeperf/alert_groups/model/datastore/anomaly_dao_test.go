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

	"infra/chromeperf/alert_groups/model"

	"go.chromium.org/luci/gae/impl/memory"
	"go.chromium.org/luci/gae/service/datastore"

	. "github.com/smartystreets/goconvey/convey"
)

func TestAnomalyModel(t *testing.T) {
	t.Parallel()

	Convey("Given DAO objects and Anomaly structs that are stored in Datastore", t, func() {
		ctx := memory.Use(context.Background())
		var anomalyDao model.AnomalyDAO = AnomalyDAO{}

		sourceAnomaly1 := &model.Anomaly{
			ID: 1,
			Groups: []*datastore.Key{
				datastore.MakeKey(ctx, "AlertGroup", "uuid"),
			},
		}
		err := datastore.Put(ctx, sourceAnomaly1)
		So(err, ShouldBeNil)

		sourceAnomaly2 := &model.Anomaly{
			ID: 2,
		}
		err = datastore.Put(ctx, sourceAnomaly2)
		So(err, ShouldBeNil)

		Convey("Anomaly can retreived by key with DAO", func() {
			storedAnomaly1, err := anomalyDao.Get(ctx, sourceAnomaly1.ID)
			So(err, ShouldBeNil)
			So(storedAnomaly1, ShouldResemble, sourceAnomaly1)

			storedAnomaly2, err := anomalyDao.Get(ctx, sourceAnomaly2.ID)
			So(err, ShouldBeNil)
			So(storedAnomaly2, ShouldResemble, sourceAnomaly2)

			Convey("Anomaly can be updated with DAO", func() {
				storedAnomaly1.Groups = []*datastore.Key{}
				err = anomalyDao.Update(ctx, storedAnomaly1)
				So(err, ShouldBeNil)

				storedAnomaly2.Groups = append(storedAnomaly2.Groups, datastore.MakeKey(ctx, "AlertGroup", "uuid2"))
				err = anomalyDao.Update(ctx, storedAnomaly2)
				So(err, ShouldBeNil)

				Convey("Update is reflected in datastore", func() {
					datastoreAnomaly1, err := anomalyDao.Get(ctx, storedAnomaly1.ID)
					So(err, ShouldBeNil)
					So(len(datastoreAnomaly1.Groups), ShouldEqual, 0)

					datastoreAnomaly2, err := anomalyDao.Get(ctx, storedAnomaly2.ID)
					So(err, ShouldBeNil)
					So(len(datastoreAnomaly2.Groups), ShouldEqual, 1)
					So(datastoreAnomaly2.Groups[0], ShouldResemble, datastore.MakeKey(ctx, "AlertGroup", "uuid2"))
				})
			})
		})
	})
}
