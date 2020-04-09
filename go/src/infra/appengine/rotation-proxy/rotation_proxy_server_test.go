// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"testing"

	timestamp "github.com/golang/protobuf/ptypes/timestamp"
	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/appengine/gaetesting"
	ds "infra/appengine/rotation-proxy/datastore"
	rpb "infra/appengine/rotation-proxy/proto"
)

func TestBatchUpdateRotations(t *testing.T) {
	ctx := gaetesting.TestingContext()
	person1 := &rpb.OncallPerson{Email: "person1@google.com"}
	person2 := &rpb.OncallPerson{Email: "person2@google.com"}
	person3 := &rpb.OncallPerson{Email: "person3@google.com"}
	person4 := &rpb.OncallPerson{Email: "person4@google.com"}
	person5 := &rpb.OncallPerson{Email: "person5@google.com"}
	oncallsShift1 := []*rpb.OncallPerson{person1, person2}
	startTimeShift1 := &timestamp.Timestamp{Seconds: 111, Nanos: 2222}
	endTimeShift1 := &timestamp.Timestamp{Seconds: 333, Nanos: 4444}
	oncallsShift2 := []*rpb.OncallPerson{person3, person4}
	startTimeShift2 := &timestamp.Timestamp{Seconds: 333, Nanos: 4444}
	endTimeShift2 := &timestamp.Timestamp{Seconds: 555, Nanos: 6666}
	oncallsShift3 := []*rpb.OncallPerson{person5}
	startTimeShift3 := &timestamp.Timestamp{Seconds: 777, Nanos: 8888}
	endTimeShift3 := &timestamp.Timestamp{Seconds: 777, Nanos: 9999}
	rotation1 := &rpb.Rotation{
		Name: "rotation1",
		Shifts: []*rpb.Shift{
			{
				Oncalls:   oncallsShift1,
				StartTime: startTimeShift1,
				EndTime:   endTimeShift1,
			},
			{
				Oncalls:   oncallsShift2,
				StartTime: startTimeShift2,
				EndTime:   endTimeShift2,
			},
		},
	}
	rotation2 := &rpb.Rotation{
		Name: "rotation2",
		Shifts: []*rpb.Shift{
			{
				Oncalls:   oncallsShift3,
				StartTime: startTimeShift3,
				EndTime:   endTimeShift3,
			},
		},
	}
	server := &RotationProxyServer{}

	Convey("batch update rotations new rotation", t, func() {
		request := &rpb.BatchUpdateRotationsRequest{
			Requests: []*rpb.UpdateRotationRequest{
				{Rotation: rotation1},
				{Rotation: rotation2},
			},
		}
		response, err := server.BatchUpdateRotations(ctx, request)
		datastore.GetTestable(ctx).CatchupIndexes()

		So(err, ShouldBeNil)

		// Checking response
		rotations := response.Rotations
		So(len(rotations), ShouldEqual, 2)
		So(rotations[0], ShouldEqual, rotation1)
		So(rotations[1], ShouldEqual, rotation2)

		// Checking data in datastore
		q := datastore.NewQuery("Rotation")
		dsRotations := []*ds.Rotation{}
		err = datastore.GetAll(ctx, q, &dsRotations)
		So(err, ShouldBeNil)
		So(len(dsRotations), ShouldEqual, 2)
		So(dsRotations[0].Name, ShouldEqual, "rotation1")
		So(dsRotations[1].Name, ShouldEqual, "rotation2")

		q = datastore.NewQuery("Shift").Ancestor(datastore.MakeKey(ctx, "Rotation", "rotation1"))
		dsShifts := []*ds.Shift{}
		err = datastore.GetAll(ctx, q, &dsShifts)
		So(err, ShouldBeNil)
		So(len(dsShifts), ShouldEqual, 2)
		assertShiftsEqual(dsShifts[0], &ds.Shift{
			Oncalls:   []rpb.OncallPerson{*person1, *person2},
			StartTime: *startTimeShift1,
			EndTime:   *endTimeShift1,
		})
		assertShiftsEqual(dsShifts[1], &ds.Shift{
			Oncalls:   []rpb.OncallPerson{*person3, *person4},
			StartTime: *startTimeShift2,
			EndTime:   *endTimeShift2,
		})

		q = datastore.NewQuery("Shift").Ancestor(datastore.MakeKey(ctx, "Rotation", "rotation2"))
		dsShifts = []*ds.Shift{}
		err = datastore.GetAll(ctx, q, &dsShifts)
		So(err, ShouldBeNil)
		So(len(dsShifts), ShouldEqual, 1)
		assertShiftsEqual(dsShifts[0], &ds.Shift{
			Oncalls:   []rpb.OncallPerson{*person5},
			StartTime: *startTimeShift3,
			EndTime:   *endTimeShift3,
		})
	})

	Convey("batch update rotations should delete previous shifts", t, func() {
		request := &rpb.BatchUpdateRotationsRequest{
			Requests: []*rpb.UpdateRotationRequest{
				{Rotation: rotation2},
			},
		}
		_, err := server.BatchUpdateRotations(ctx, request)
		datastore.GetTestable(ctx).CatchupIndexes()
		So(err, ShouldBeNil)

		// Try to update rotation 2, make sure shifts are replace
		rotation2Updated := &rpb.Rotation{
			Name: "rotation2",
			Shifts: []*rpb.Shift{
				{
					Oncalls:   oncallsShift2,
					StartTime: startTimeShift2,
					EndTime:   endTimeShift2,
				},
			},
		}
		request2 := &rpb.BatchUpdateRotationsRequest{
			Requests: []*rpb.UpdateRotationRequest{
				{Rotation: rotation2Updated},
			},
		}
		_, err2 := server.BatchUpdateRotations(ctx, request2)
		datastore.GetTestable(ctx).CatchupIndexes()

		So(err2, ShouldBeNil)
		q := datastore.NewQuery("Shift").Ancestor(datastore.MakeKey(ctx, "Rotation", "rotation2"))
		dsShifts := []*ds.Shift{}
		err = datastore.GetAll(ctx, q, &dsShifts)
		So(err, ShouldBeNil)
		So(len(dsShifts), ShouldEqual, 1)

		assertShiftsEqual(dsShifts[0], &ds.Shift{
			Oncalls:   []rpb.OncallPerson{*person3, *person4},
			StartTime: *startTimeShift2,
			EndTime:   *endTimeShift2,
		})
	})
}

// Compare shift without ID
func assertShiftsEqual(shift1 *ds.Shift, shift2 *ds.Shift) {
	So(shift1.StartTime, ShouldResemble, shift2.StartTime)
	So(shift1.EndTime, ShouldResemble, shift2.EndTime)
	So(shift1.Oncalls, ShouldResemble, shift2.Oncalls)
}
