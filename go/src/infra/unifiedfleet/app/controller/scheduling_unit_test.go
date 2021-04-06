// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
	"google.golang.org/genproto/protobuf/field_mask"

	ufspb "infra/unifiedfleet/api/v1/models"
	. "infra/unifiedfleet/app/model/datastore"
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

func TestUpdateSchedulingUnit(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("UpdateSchedulingUnit", t, func() {
		Convey("Update SchedulingUnit for existing SchedulingUnit - happy path", func() {
			su1 := mockSchedulingUnit("su-1")
			su1.Tags = []string{"Dell"}
			inventory.CreateSchedulingUnit(ctx, su1)

			su2 := mockSchedulingUnit("su-1")
			su2.Tags = []string{"Apple"}
			resp, _ := UpdateSchedulingUnit(ctx, su2, nil)
			So(resp, ShouldNotBeNil)
			So(resp, ShouldResembleProto, su2)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "schedulingunits/su-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetEventLabel(), ShouldEqual, "schedulingunit.tags")
			So(changes[0].GetOldValue(), ShouldResemble, "[Dell]")
			So(changes[0].GetNewValue(), ShouldResemble, "[Apple]")

			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "schedulingunits/su-1")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			So(msgs[0].Delete, ShouldBeFalse)
		})

		Convey("Update SchedulingUnit for non-existing SchedulingUnit", func() {
			su := mockSchedulingUnit("su-2")
			resp, err := UpdateSchedulingUnit(ctx, su, nil)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "schedulingunits/su-2")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})

		Convey("UpdateSchedulingUnit - DUT non-existing", func() {
			_, err := inventory.CreateMachineLSE(ctx, &ufspb.MachineLSE{
				Name: "dut-3",
			})
			So(err, ShouldBeNil)

			su := mockSchedulingUnit("su-3")
			su.MachineLSEs = []string{"dut-3"}
			_, err = inventory.CreateSchedulingUnit(ctx, su)
			So(err, ShouldBeNil)

			su1 := mockSchedulingUnit("su-3")
			su1.MachineLSEs = []string{"dut-3", "dut-4"}
			_, err = UpdateSchedulingUnit(ctx, su1, nil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "There is no MachineLSE with MachineLSEID dut-4 in the system.")

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "schedulingunits/su-3")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)

			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "schedulingunits/su-3")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 0)
		})

		Convey("UpdateSchedulingUnit - DUT already associated", func() {
			_, err := inventory.CreateMachineLSE(ctx, &ufspb.MachineLSE{
				Name: "dut-4",
			})
			So(err, ShouldBeNil)

			su1 := mockSchedulingUnit("su-4")
			su1.MachineLSEs = []string{"dut-4"}
			inventory.CreateSchedulingUnit(ctx, su1)

			su2 := mockSchedulingUnit("su-5")
			_, err = inventory.CreateSchedulingUnit(ctx, su2)
			So(err, ShouldBeNil)

			su3 := mockSchedulingUnit("su-5")
			su3.MachineLSEs = []string{"dut-4"}
			_, err = UpdateSchedulingUnit(ctx, su3, nil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "already associated")

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "schedulingunits/su-5")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)

			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "schedulingunits/su-5")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 0)
		})

		Convey("Update SchedulingUnit for existing SchedulingUnit - partial update(append) machinelses", func() {
			_, err := inventory.CreateMachineLSE(ctx, &ufspb.MachineLSE{
				Name: "dut-1",
			})
			So(err, ShouldBeNil)

			_, err = inventory.CreateMachineLSE(ctx, &ufspb.MachineLSE{
				Name: "dut-2",
			})
			So(err, ShouldBeNil)

			su1 := mockSchedulingUnit("su-7")
			su1.MachineLSEs = []string{"dut-1"}
			inventory.CreateSchedulingUnit(ctx, su1)

			su2 := mockSchedulingUnit("su-7")
			su2.MachineLSEs = []string{"dut-2"}
			resp, _ := UpdateSchedulingUnit(ctx, su2, &field_mask.FieldMask{Paths: []string{"machinelses"}})
			So(resp, ShouldNotBeNil)
			So(resp.GetName(), ShouldEqual, su2.GetName())
			So(resp.GetMachineLSEs(), ShouldResemble, []string{"dut-1", "dut-2"})

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "schedulingunits/su-7")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetEventLabel(), ShouldEqual, "schedulingunit.machinelses")
			So(changes[0].GetOldValue(), ShouldEqual, "[dut-1]")
			So(changes[0].GetNewValue(), ShouldEqual, "[dut-1 dut-2]")

			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "schedulingunits/su-7")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			So(msgs[0].Delete, ShouldBeFalse)
		})

		Convey("Update SchedulingUnit for existing SchedulingUnit - partial update(remove) machinelses", func() {
			_, err := inventory.CreateMachineLSE(ctx, &ufspb.MachineLSE{
				Name: "dut-6",
			})
			So(err, ShouldBeNil)

			su1 := mockSchedulingUnit("su-6")
			su1.MachineLSEs = []string{"dut-6"}
			inventory.CreateSchedulingUnit(ctx, su1)

			su2 := mockSchedulingUnit("su-6")
			su2.MachineLSEs = []string{"dut-6"}
			resp, err := UpdateSchedulingUnit(ctx, su2, &field_mask.FieldMask{Paths: []string{"machinelses.remove"}})
			So(err, ShouldBeNil)
			So(resp.GetName(), ShouldEqual, su2.GetName())
			So(resp.GetMachineLSEs(), ShouldResemble, []string{})

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "schedulingunits/su-6")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetEventLabel(), ShouldEqual, "schedulingunit.machinelses")
			So(changes[0].GetOldValue(), ShouldEqual, "[dut-6]")
			So(changes[0].GetNewValue(), ShouldEqual, "[]")

			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "schedulingunits/su-6")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			So(msgs[0].Delete, ShouldBeFalse)
		})
	})
}
