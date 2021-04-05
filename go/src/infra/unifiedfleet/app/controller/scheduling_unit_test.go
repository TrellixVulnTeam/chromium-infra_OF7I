// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"

	ufspb "infra/unifiedfleet/api/v1/models"
	"infra/unifiedfleet/app/model/history"
	"infra/unifiedfleet/app/model/inventory"
)

func mockSchedulingUnit(name string) *ufspb.SchedulingUnit {
	return &ufspb.SchedulingUnit{
		Name: name,
	}
}

func TestCreateSchedulingUnit(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("CreateSchedulingUnit", t, func() {
		Convey("Create new SchedulingUnit - happy path", func() {
			su := mockSchedulingUnit("su-1")
			resp, err := CreateSchedulingUnit(ctx, su)
			So(resp, ShouldNotBeNil)
			So(resp, ShouldResembleProto, su)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "schedulingunits/su-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetEventLabel(), ShouldEqual, "schedulingunit")

			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "schedulingunits/su-1")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			So(msgs[0].Delete, ShouldBeFalse)
		})

		Convey("Create new SchedulingUnit - already existing", func() {
			su1 := mockSchedulingUnit("su-2")
			inventory.CreateSchedulingUnit(ctx, su1)

			su2 := mockSchedulingUnit("su-2")
			_, err := CreateSchedulingUnit(ctx, su2)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "already exists")

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "schedulingunits/su-2")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)

			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "schedulingunits/su-2")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 0)
		})

		Convey("Create new SchedulingUnit - DUT non-existing", func() {
			_, err := inventory.CreateMachineLSE(ctx, &ufspb.MachineLSE{
				Name: "dut-1",
			})
			So(err, ShouldBeNil)

			su2 := mockSchedulingUnit("su-3")
			su2.MachineLSEs = []string{"dut-1", "dut-2"}
			_, err = CreateSchedulingUnit(ctx, su2)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "There is no MachineLSE with MachineLSEID dut-2 in the system.")

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "schedulingunits/su-3")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)

			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "schedulingunits/su-3")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 0)
		})

		Convey("Create new SchedulingUnit - DUT already associated", func() {
			_, err := inventory.CreateMachineLSE(ctx, &ufspb.MachineLSE{
				Name: "dut-2",
			})
			So(err, ShouldBeNil)

			su1 := mockSchedulingUnit("su-4")
			su1.MachineLSEs = []string{"dut-2"}
			inventory.CreateSchedulingUnit(ctx, su1)

			su2 := mockSchedulingUnit("su-5")
			su2.MachineLSEs = []string{"dut-2"}
			_, err = CreateSchedulingUnit(ctx, su2)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "already associated")

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "schedulingunits/su-5")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)

			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "schedulingunits/su-5")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 0)
		})
	})
}
