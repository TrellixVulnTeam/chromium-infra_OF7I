// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package registration

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/appengine/gaetesting"
	. "go.chromium.org/luci/common/testing/assertions"

	ufspb "infra/unifiedfleet/api/v1/proto"
	. "infra/unifiedfleet/app/model/datastore"
)

func mockRack(id string, rackCapactiy int32) *ufspb.Rack {
	return &ufspb.Rack{
		Name:       id,
		CapacityRu: rackCapactiy,
	}
}

func TestCreateRack(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	datastore.GetTestable(ctx).Consistent(true)
	rack1 := mockRack("Rack-1", 5)
	rack2 := mockRack("", 10)
	Convey("CreateRack", t, func() {
		Convey("Create new rack", func() {
			resp, err := CreateRack(ctx, rack1)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, rack1)
		})
		Convey("Create existing rack", func() {
			resp, err := CreateRack(ctx, rack1)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, AlreadyExists)
		})
		Convey("Create rack - invalid ID", func() {
			resp, err := CreateRack(ctx, rack2)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}

func TestUpdateRack(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	rack1 := mockRack("Rack-1", 5)
	rack2 := mockRack("Rack-1", 10)
	rack3 := mockRack("Rack-3", 15)
	rack4 := mockRack("", 20)
	Convey("UpdateRack", t, func() {
		Convey("Update existing rack", func() {
			resp, err := CreateRack(ctx, rack1)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, rack1)

			resp, err = UpdateRack(ctx, rack2)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, rack2)
		})
		Convey("Update non-existing rack", func() {
			resp, err := UpdateRack(ctx, rack3)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Update rack - invalid ID", func() {
			resp, err := UpdateRack(ctx, rack4)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}

func TestGetRack(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	rack1 := mockRack("Rack-1", 5)
	Convey("GetRack", t, func() {
		Convey("Get rack by existing ID", func() {
			resp, err := CreateRack(ctx, rack1)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, rack1)
			resp, err = GetRack(ctx, "Rack-1")
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, rack1)
		})
		Convey("Get rack by non-existing ID", func() {
			resp, err := GetRack(ctx, "rack-2")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Get rack - invalid ID", func() {
			resp, err := GetRack(ctx, "")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}

func TestListRacks(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	datastore.GetTestable(ctx).Consistent(true)
	racks := make([]*ufspb.Rack, 0, 4)
	for i := 0; i < 4; i++ {
		rack1 := mockRack(fmt.Sprintf("rack-%d", i), 5)
		resp, _ := CreateRack(ctx, rack1)
		racks = append(racks, resp)
	}
	Convey("ListRacks", t, func() {
		Convey("List racks - page_token invalid", func() {
			resp, nextPageToken, err := ListRacks(ctx, 5, "abc", nil, false)
			So(resp, ShouldBeNil)
			So(nextPageToken, ShouldBeEmpty)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InvalidPageToken)
		})

		Convey("List racks - Full listing with no pagination", func() {
			resp, nextPageToken, err := ListRacks(ctx, 4, "", nil, false)
			So(resp, ShouldNotBeNil)
			So(nextPageToken, ShouldNotBeEmpty)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, racks)
		})

		Convey("List racks - listing with pagination", func() {
			resp, nextPageToken, err := ListRacks(ctx, 3, "", nil, false)
			So(resp, ShouldNotBeNil)
			So(nextPageToken, ShouldNotBeEmpty)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, racks[:3])

			resp, _, err = ListRacks(ctx, 2, nextPageToken, nil, false)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, racks[3:])
		})
	})
}

func TestDeleteRack(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	rack1 := mockRack("rack-1", 5)
	Convey("DeleteRack", t, func() {
		Convey("Delete rack by existing ID", func() {
			resp, cerr := CreateRack(ctx, rack1)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, rack1)
			err := DeleteRack(ctx, "rack-1")
			So(err, ShouldBeNil)
			res, err := GetRack(ctx, "rack-1")
			So(res, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Delete rack by non-existing ID", func() {
			err := DeleteRack(ctx, "rack-2")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Delete rack - invalid ID", func() {
			err := DeleteRack(ctx, "")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}

func TestBatchUpdateRacks(t *testing.T) {
	t.Parallel()
	Convey("BatchUpdateRacks", t, func() {
		ctx := gaetesting.TestingContextWithAppID("go-test")
		datastore.GetTestable(ctx).Consistent(true)
		Racks := make([]*ufspb.Rack, 0, 4)
		for i := 0; i < 4; i++ {
			Rack1 := mockRack(fmt.Sprintf("Rack-%d", i), 10)
			resp, err := CreateRack(ctx, Rack1)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, Rack1)
			Racks = append(Racks, resp)
		}
		Convey("BatchUpdate all Racks", func() {
			resp, err := BatchUpdateRacks(ctx, Racks)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, Racks)
		})
		Convey("BatchUpdate existing and invalid Racks", func() {
			Rack5 := mockRack("", 10)
			Racks = append(Racks, Rack5)
			resp, err := BatchUpdateRacks(ctx, Racks)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}

func TestQueryRackByPropertyName(t *testing.T) {
	t.Parallel()
	Convey("QueryRackByPropertyName", t, func() {
		ctx := gaetesting.TestingContextWithAppID("go-test")
		datastore.GetTestable(ctx).Consistent(true)
		dummyRack := &ufspb.Rack{
			Name: "Rack-1",
		}
		Rack1 := &ufspb.Rack{
			Name: "Rack-1",
			Rack: &ufspb.Rack_ChromeBrowserRack{
				ChromeBrowserRack: &ufspb.ChromeBrowserRack{
					Kvms: []string{"KVM-1", "KVM-2"},
				},
			},
		}
		resp, cerr := CreateRack(ctx, Rack1)
		So(cerr, ShouldBeNil)
		So(resp, ShouldResembleProto, Rack1)

		Racks := make([]*ufspb.Rack, 0, 1)
		Racks = append(Racks, Rack1)

		dummyRacks := make([]*ufspb.Rack, 0, 1)
		dummyRacks = append(dummyRacks, dummyRack)
		Convey("Query By existing Rack", func() {
			resp, err := QueryRackByPropertyName(ctx, "kvm_ids", "KVM-1", false)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, Racks)
		})
		Convey("Query By non-existing Rack", func() {
			resp, err := QueryRackByPropertyName(ctx, "kvm_ids", "KVM-5", false)
			So(err, ShouldBeNil)
			So(resp, ShouldBeNil)
		})
		Convey("Query By existing RackPrototype keysonly", func() {
			resp, err := QueryRackByPropertyName(ctx, "kvm_ids", "KVM-1", true)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, dummyRacks)
		})
		Convey("Query By non-existing RackPrototype", func() {
			resp, err := QueryRackByPropertyName(ctx, "kvm_ids", "KVM-5", true)
			So(err, ShouldBeNil)
			So(resp, ShouldBeNil)
		})
	})
}
