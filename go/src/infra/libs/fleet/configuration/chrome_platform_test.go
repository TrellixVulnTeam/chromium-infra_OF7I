// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package configuration

import (
	fleet "infra/unifiedfleet/api/v1/proto"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/appengine/gaetesting"
)

func mockChromePlatform(id string) *fleet.ChromePlatform {
	return &fleet.ChromePlatform{
		Name: id,
	}
}
func TestInsertChromePlatforms(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	datastore.GetTestable(ctx).Consistent(true)

	Convey("Add chrome platform to datastore", t, func() {
		Convey("Add 2 new platforms", func() {
			toAdd := []*fleet.ChromePlatform{
				mockChromePlatform("platform1"),
				mockChromePlatform("platform2"),
			}
			dsResp, err := InsertChromePlatforms(ctx, toAdd)
			So(err, ShouldBeNil)
			So(dsResp.Passed(), ShouldHaveLength, len(toAdd))
			So(dsResp.Failed(), ShouldHaveLength, 0)
			want := []string{
				toAdd[0].GetName(),
				toAdd[1].GetName(),
			}
			gets, err := GetAllChromePlatforms(ctx)
			So(err, ShouldBeNil)
			So(gets, ShouldHaveLength, 2)
			passed := gets.Passed()
			got := []string{
				passed[0].Data.(*fleet.ChromePlatform).Name,
				passed[1].Data.(*fleet.ChromePlatform).Name,
			}
			So(got, ShouldResemble, want)
		})

		Convey("Add existing platforms", func() {
			toAdd := []*fleet.ChromePlatform{
				mockChromePlatform("platform1"),
				mockChromePlatform("platform2"),
			}
			_, err := InsertChromePlatforms(ctx, toAdd)
			So(err, ShouldBeNil)

			toAddDuplicated := []*fleet.ChromePlatform{
				mockChromePlatform("platform1"),
				mockChromePlatform("platform3"),
			}
			dsResp, err := InsertChromePlatforms(ctx, toAddDuplicated)
			So(err, ShouldBeNil)
			So(dsResp.Passed(), ShouldHaveLength, 1)
			So(dsResp.Failed(), ShouldHaveLength, 1)
			So(dsResp.Passed()[0].Data.(*fleet.ChromePlatform).Name, ShouldEqual, "platform3")
			So(dsResp.Failed()[0].Data.(*fleet.ChromePlatform).Name, ShouldEqual, "platform1")

			want := []string{"platform1", "platform2", "platform3"}
			gets, err := GetAllChromePlatforms(ctx)
			So(err, ShouldBeNil)
			So(gets, ShouldHaveLength, 3)
			passed := gets.Passed()
			got := []string{
				passed[0].Data.(*fleet.ChromePlatform).Name,
				passed[1].Data.(*fleet.ChromePlatform).Name,
				passed[2].Data.(*fleet.ChromePlatform).Name,
			}
			So(got, ShouldResemble, want)
		})
	})
}
