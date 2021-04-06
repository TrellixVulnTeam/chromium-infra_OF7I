// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package inventory

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/luci/appengine/gaetesting"
	. "go.chromium.org/luci/common/testing/assertions"
	"go.chromium.org/luci/gae/service/datastore"

	ufspb "infra/unifiedfleet/api/v1/models"
	. "infra/unifiedfleet/app/model/datastore"
)

func mockSchedulingUnit(name string) *ufspb.SchedulingUnit {
	return &ufspb.SchedulingUnit{
		Name: name,
	}
}

func TestCreateSchedulingUnit(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	datastore.GetTestable(ctx).Consistent(true)
	Convey("CreateSchedulingUnit", t, func() {
		Convey("Create new SchedulingUnit", func() {
			su := mockSchedulingUnit("SU-X")
			resp, err := CreateSchedulingUnit(ctx, su)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, su)
		})
		Convey("Create existing SchedulingUnit", func() {
			su1 := mockSchedulingUnit("SU-Y")
			CreateSchedulingUnit(ctx, su1)

			resp, err := CreateSchedulingUnit(ctx, su1)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, AlreadyExists)
		})
	})
}

func TestBatchUpdateSchedulingUnits(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	datastore.GetTestable(ctx).Consistent(true)
	Convey("BatchUpdateSchedulingUnits", t, func() {
		Convey("Create new SchedulingUnit", func() {
			su := mockSchedulingUnit("SU-A")
			resp, err := BatchUpdateSchedulingUnits(ctx, []*ufspb.SchedulingUnit{su})
			So(err, ShouldBeNil)
			So(resp[0], ShouldResembleProto, su)
		})
	})
}

func TestQuerySchedulingUnitByPropertyNames(t *testing.T) {
	t.Parallel()
	keyOnlySchedulingUnit1 := &ufspb.SchedulingUnit{
		Name: "SchedulingUnit-1",
	}
	keyOnlySchedulingUnit2 := &ufspb.SchedulingUnit{
		Name: "SchedulingUnit-2",
	}
	keysOnlySchedulingUnits := []*ufspb.SchedulingUnit{keyOnlySchedulingUnit1, keyOnlySchedulingUnit2}
	schedulingUnit1 := &ufspb.SchedulingUnit{
		Name:        "SchedulingUnit-1",
		MachineLSEs: []string{"dut-1"},
		Pools:       []string{"pool-1", "pool-3"},
		Tags:        []string{"tags-3"},
	}
	schedulingUnit2 := &ufspb.SchedulingUnit{
		Name:        "SchedulingUnit-2",
		MachineLSEs: []string{"dut-2"},
		Pools:       []string{"pool-2", "pool-3"},
		Tags:        []string{"tags-3"},
	}
	schedulingUnits := []*ufspb.SchedulingUnit{schedulingUnit1, schedulingUnit2}
	Convey("QuerySchedulingUnitByPropertyNames", t, func() {
		ctx := gaetesting.TestingContextWithAppID("go-test")
		datastore.GetTestable(ctx).Consistent(true)
		_, err := BatchUpdateSchedulingUnits(ctx, schedulingUnits)
		So(err, ShouldBeNil)

		Convey("Query By existing MachineLSE", func() {
			resp, err := QuerySchedulingUnitByPropertyNames(ctx, map[string]string{"machinelses": "dut-1"}, false)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, []*ufspb.SchedulingUnit{schedulingUnit1})
		})
		Convey("Query By non-existing MachineLSE", func() {
			resp, err := QuerySchedulingUnitByPropertyNames(ctx, map[string]string{"machinelses": "dut-4"}, false)
			So(err, ShouldBeNil)
			So(resp, ShouldBeNil)
		})
		Convey("Query By existing pools and tags", func() {
			resp, err := QuerySchedulingUnitByPropertyNames(ctx, map[string]string{"pools": "pool-3", "tags": "tags-3"}, false)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, schedulingUnits)
		})
		Convey("Query By existing pools and MachineLSEs", func() {
			resp, err := QuerySchedulingUnitByPropertyNames(ctx, map[string]string{"pools": "pool-3", "machinelses": "dut-2"}, false)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, []*ufspb.SchedulingUnit{schedulingUnit2})
		})
		Convey("Query By existing pools and tags by keysonly", func() {
			resp, err := QuerySchedulingUnitByPropertyNames(ctx, map[string]string{"pools": "pool-3", "tags": "tags-3"}, true)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, keysOnlySchedulingUnits)
		})
	})
}

func TestGetSchedulingUnit(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	su1 := mockSchedulingUnit("su-1")
	Convey("GetSchedulingUnit", t, func() {
		Convey("Get SchedulingUnit by existing name/ID", func() {
			resp, err := CreateSchedulingUnit(ctx, su1)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, su1)
			resp, err = GetSchedulingUnit(ctx, "su-1")
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, su1)
		})
		Convey("Get SchedulingUnit by non-existing name/ID", func() {
			resp, err := GetSchedulingUnit(ctx, "su-2")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Get SchedulingUnit - invalid name/ID", func() {
			resp, err := GetSchedulingUnit(ctx, "")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}
