// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"fmt"
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

func TestGetSchedulingUnit(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	su, _ := inventory.CreateSchedulingUnit(ctx, &ufspb.SchedulingUnit{
		Name: "su-1",
	})
	Convey("GetSchedulingUnit", t, func() {
		Convey("Get SchedulingUnit by existing ID - happy path", func() {
			resp, _ := GetSchedulingUnit(ctx, "su-1")
			So(resp, ShouldNotBeNil)
			So(resp, ShouldResembleProto, su)
		})

		Convey("Get SchedulingUnit by non-existing ID", func() {
			_, err := GetSchedulingUnit(ctx, "su-2")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
	})
}

func TestDeleteSchedulingUnit(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	inventory.CreateSchedulingUnit(ctx, &ufspb.SchedulingUnit{
		Name: "su-1",
	})
	Convey("DeleteSchedulingUnit", t, func() {
		Convey("Delete SchedulingUnit by existing ID - happy path", func() {
			err := DeleteSchedulingUnit(ctx, "su-1")
			So(err, ShouldBeNil)

			res, err := inventory.GetSchedulingUnit(ctx, "su-1")
			So(res, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "schedulingunits/su-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRetire)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRetire)
			So(changes[0].GetEventLabel(), ShouldEqual, "schedulingunit")

			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "schedulingunits/su-1")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			So(msgs[0].Delete, ShouldBeTrue)
		})

		Convey("Delete SchedulingUnit by non-existing ID", func() {
			err := DeleteSchedulingUnit(ctx, "su-2")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
	})
}

func TestListSchedulingUnits(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	schedulingUnitsWithPools := make([]*ufspb.SchedulingUnit, 0, 2)
	schedulingUnits := make([]*ufspb.SchedulingUnit, 0, 4)
	for i := 0; i < 4; i++ {
		su := mockSchedulingUnit(fmt.Sprintf("su-%d", i))
		if i%2 == 0 {
			su.Pools = []string{"DUT_QUOTA"}
		}
		resp, _ := inventory.CreateSchedulingUnit(ctx, su)
		if i%2 == 0 {
			schedulingUnitsWithPools = append(schedulingUnitsWithPools, resp)
		}
		schedulingUnits = append(schedulingUnits, resp)
	}
	Convey("ListSchedulingUnits", t, func() {
		Convey("List SchedulingUnits - filter invalid - error", func() {
			_, _, err := ListSchedulingUnits(ctx, 5, "", "invalid=mx-1", false)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Invalid field name invalid")
		})

		Convey("List SchedulingUnits - filter switch - happy path", func() {
			resp, _, _ := ListSchedulingUnits(ctx, 5, "", "pools=DUT_QUOTA", false)
			So(resp, ShouldNotBeNil)
			So(resp, ShouldResembleProto, schedulingUnitsWithPools)
		})

		Convey("ListSchedulingUnits - Full listing - happy path", func() {
			resp, _, _ := ListSchedulingUnits(ctx, 5, "", "", false)
			So(resp, ShouldNotBeNil)
			So(resp, ShouldResembleProto, schedulingUnits)
		})
	})
}
