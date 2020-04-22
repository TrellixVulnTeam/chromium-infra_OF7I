// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	timestamp "github.com/golang/protobuf/ptypes/timestamp"
	"github.com/google/go-cmp/cmp"
	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/appengine/gaetesting"
	"go.chromium.org/luci/common/clock/testclock"
	rpb "infra/appengine/rotation-proxy/proto"
)

var person1 = &rpb.OncallPerson{Email: "person1@google.com"}
var person2 = &rpb.OncallPerson{Email: "person2@google.com"}
var person3 = &rpb.OncallPerson{Email: "person3@google.com"}
var person4 = &rpb.OncallPerson{Email: "person4@google.com"}
var person5 = &rpb.OncallPerson{Email: "person5@google.com"}
var oncallsShift1 = []*rpb.OncallPerson{person1, person2}
var startTimeShift1 = &timestamp.Timestamp{Seconds: 111, Nanos: 0}
var endTimeShift1 = &timestamp.Timestamp{Seconds: 222, Nanos: 0}
var oncallsShift2 = []*rpb.OncallPerson{person3, person4}
var startTimeShift2 = &timestamp.Timestamp{Seconds: 333, Nanos: 0}
var endTimeShift2 = &timestamp.Timestamp{Seconds: 555, Nanos: 0}
var oncallsShift3 = []*rpb.OncallPerson{person5}
var startTimeShift3 = &timestamp.Timestamp{Seconds: 777, Nanos: 0}
var endTimeShift3 = &timestamp.Timestamp{Seconds: 777, Nanos: 0}

var rotation1 = &rpb.Rotation{
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

func TestBatchUpdateRotations(t *testing.T) {
	ctx := gaetesting.TestingContext()

	server := &RotationProxyServer{}

	Convey("batch update rotations new rotation", t, func() {
		// TODO(nqmtuan): Figure out how can we set datastore to deal with
		// multiple entity groups in testing.
		// Currently, it is complaining about enabling XG=true, which should not
		// be required, since we are using firestore in datastore mode.
		request := &rpb.BatchUpdateRotationsRequest{
			Requests: []*rpb.UpdateRotationRequest{
				{Rotation: rotation1},
			},
		}
		response, err := server.BatchUpdateRotations(ctx, request)
		datastore.GetTestable(ctx).CatchupIndexes()

		So(err, ShouldBeNil)

		// Checking response
		rotations := response.Rotations
		So(len(rotations), ShouldEqual, 1)
		So(rotations[0], ShouldEqual, rotation1)

		// Checking data in datastore
		q := datastore.NewQuery("Rotation")
		dsRotations := []*Rotation{}
		err = datastore.GetAll(ctx, q, &dsRotations)
		So(err, ShouldBeNil)
		So(len(dsRotations), ShouldEqual, 1)
		diff := cmp.Diff(rotation1, &dsRotations[0].Proto, cmp.Comparer(proto.Equal))
		So(diff, ShouldEqual, "")
	})

	Convey("batch update rotations should delete previous shifts", t, func() {
		rotation1Updated := &rpb.Rotation{
			Name: "rotation1",
			Shifts: []*rpb.Shift{
				{
					Oncalls:   oncallsShift3,
					StartTime: startTimeShift3,
					EndTime:   endTimeShift3,
				},
			},
		}
		request := &rpb.BatchUpdateRotationsRequest{
			Requests: []*rpb.UpdateRotationRequest{
				{Rotation: rotation1Updated},
			},
		}
		_, err := server.BatchUpdateRotations(ctx, request)
		datastore.GetTestable(ctx).CatchupIndexes()
		So(err, ShouldBeNil)
		q := datastore.NewQuery("Rotation")
		dsRotations := []*Rotation{}
		err = datastore.GetAll(ctx, q, &dsRotations)
		So(err, ShouldBeNil)
		So(len(dsRotations), ShouldEqual, 1)
		diff := cmp.Diff(rotation1Updated, &dsRotations[0].Proto, cmp.Comparer(proto.Equal))
		So(diff, ShouldEqual, "")
	})
}

func TestBatchGetRotations(t *testing.T) {
	ctx := gaetesting.TestingContext()
	server := &RotationProxyServer{}
	Convey("batch get rotations", t, func() {
		var rotation = &rpb.Rotation{
			Name: "rotation",
			Shifts: []*rpb.Shift{
				{
					Oncalls:   oncallsShift3,
					StartTime: startTimeShift3,
				},
				{
					Oncalls:   oncallsShift2,
					StartTime: startTimeShift2,
					EndTime:   endTimeShift2,
				},
				{
					Oncalls:   oncallsShift1,
					StartTime: startTimeShift1,
					EndTime:   endTimeShift1,
				},
			},
		}
		updateRequest := &rpb.BatchUpdateRotationsRequest{
			Requests: []*rpb.UpdateRotationRequest{
				{Rotation: rotation},
			},
		}
		_, err := server.BatchUpdateRotations(ctx, updateRequest)
		datastore.GetTestable(ctx).CatchupIndexes()
		So(err, ShouldBeNil)

		getRequest := &rpb.BatchGetRotationsRequest{
			Names: []string{"rotation"},
		}

		// Mock clock
		ctx, _ = testclock.UseTime(ctx, time.Unix(444, 0))

		response, err := server.BatchGetRotations(ctx, getRequest)
		So(err, ShouldBeNil)
		So(len(response.Rotations), ShouldEqual, 1)
		expected := &rpb.Rotation{
			Name: "rotation",
			Shifts: []*rpb.Shift{
				{
					Oncalls:   oncallsShift2,
					StartTime: startTimeShift2,
					EndTime:   endTimeShift2,
				},
				{
					Oncalls:   oncallsShift3,
					StartTime: startTimeShift3,
				},
			},
		}
		diff := cmp.Diff(expected, response.Rotations[0], cmp.Comparer(proto.Equal))
		So(diff, ShouldEqual, "")
	})
}
