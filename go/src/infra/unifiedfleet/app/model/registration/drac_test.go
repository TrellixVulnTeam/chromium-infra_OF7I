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

func mockDrac(id string) *ufspb.Drac {
	return &ufspb.Drac{
		Name: id,
	}
}

func TestCreateDrac(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	datastore.GetTestable(ctx).Consistent(true)
	drac1 := mockDrac("Drac-1")
	drac2 := mockDrac("")
	Convey("CreateDrac", t, func() {
		Convey("Create new drac", func() {
			resp, err := CreateDrac(ctx, drac1)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, drac1)
		})
		Convey("Create existing drac", func() {
			resp, err := CreateDrac(ctx, drac1)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, AlreadyExists)
		})
		Convey("Create drac - invalid ID", func() {
			resp, err := CreateDrac(ctx, drac2)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}

func TestUpdateDrac(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	drac1 := mockDrac("Drac-1")
	drac2 := mockDrac("Drac-1")
	drac3 := mockDrac("Drac-3")
	drac4 := mockDrac("")
	Convey("UpdateDrac", t, func() {
		Convey("Update existing drac", func() {
			resp, err := CreateDrac(ctx, drac1)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, drac1)

			resp, err = UpdateDrac(ctx, drac2)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, drac2)
		})
		Convey("Update non-existing drac", func() {
			resp, err := UpdateDrac(ctx, drac3)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Update drac - invalid ID", func() {
			resp, err := UpdateDrac(ctx, drac4)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}

func TestGetDrac(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	drac1 := mockDrac("Drac-1")
	Convey("GetDrac", t, func() {
		Convey("Get drac by existing ID", func() {
			resp, err := CreateDrac(ctx, drac1)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, drac1)
			resp, err = GetDrac(ctx, "Drac-1")
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, drac1)
		})
		Convey("Get drac by non-existing ID", func() {
			resp, err := GetDrac(ctx, "drac-2")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Get drac - invalid ID", func() {
			resp, err := GetDrac(ctx, "")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}

func TestListDracs(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	datastore.GetTestable(ctx).Consistent(true)
	dracs := make([]*ufspb.Drac, 0, 4)
	for i := 0; i < 4; i++ {
		drac1 := mockDrac(fmt.Sprintf("drac-%d", i))
		resp, _ := CreateDrac(ctx, drac1)
		dracs = append(dracs, resp)
	}
	Convey("ListDracs", t, func() {
		Convey("List dracs - page_token invalid", func() {
			resp, nextPageToken, err := ListDracs(ctx, 5, "abc", nil, false)
			So(resp, ShouldBeNil)
			So(nextPageToken, ShouldBeEmpty)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InvalidPageToken)
		})

		Convey("List dracs - Full listing with no pagination", func() {
			resp, nextPageToken, err := ListDracs(ctx, 4, "", nil, false)
			So(resp, ShouldNotBeNil)
			So(nextPageToken, ShouldNotBeEmpty)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, dracs)
		})

		Convey("List dracs - listing with pagination", func() {
			resp, nextPageToken, err := ListDracs(ctx, 3, "", nil, false)
			So(resp, ShouldNotBeNil)
			So(nextPageToken, ShouldNotBeEmpty)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, dracs[:3])

			resp, _, err = ListDracs(ctx, 2, nextPageToken, nil, false)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, dracs[3:])
		})
	})
}

func TestDeleteDrac(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	datastore.GetTestable(ctx).Consistent(true)
	drac2 := mockDrac("drac-2")
	Convey("DeleteDrac", t, func() {
		Convey("Delete drac successfully by existing ID", func() {
			resp, cerr := CreateDrac(ctx, drac2)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, drac2)

			err := DeleteDrac(ctx, "drac-2")
			So(err, ShouldBeNil)

			resp, cerr = GetDrac(ctx, "drac-2")
			So(resp, ShouldBeNil)
			So(cerr, ShouldNotBeNil)
			So(cerr.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Delete drac by non-existing ID", func() {
			err := DeleteDrac(ctx, "drac-3")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Delete drac - invalid ID", func() {
			err := DeleteDrac(ctx, "")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}

func TestBatchUpdateDracs(t *testing.T) {
	t.Parallel()
	Convey("BatchUpdateDracs", t, func() {
		ctx := gaetesting.TestingContextWithAppID("go-test")
		datastore.GetTestable(ctx).Consistent(true)
		dracs := make([]*ufspb.Drac, 0, 4)
		for i := 0; i < 4; i++ {
			drac1 := mockDrac(fmt.Sprintf("drac-%d", i))
			resp, err := CreateDrac(ctx, drac1)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, drac1)
			dracs = append(dracs, resp)
		}
		Convey("BatchUpdate all dracs", func() {
			resp, err := BatchUpdateDracs(ctx, dracs)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, dracs)
		})
		Convey("BatchUpdate existing and non-existing dracs", func() {
			Drac5 := mockDrac("")
			dracs = append(dracs, Drac5)
			resp, err := BatchUpdateDracs(ctx, dracs)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}

func TestQueryDracByPropertyName(t *testing.T) {
	t.Parallel()
	Convey("QueryDracByPropertyName", t, func() {
		ctx := gaetesting.TestingContextWithAppID("go-test")
		datastore.GetTestable(ctx).Consistent(true)
		dummyDrac := &ufspb.Drac{
			Name: "drac-15",
		}
		drac1 := &ufspb.Drac{
			Name: "drac-15",
			SwitchInterface: &ufspb.SwitchInterface{
				Switch: "switch-1",
			},
		}
		resp, cerr := CreateDrac(ctx, drac1)
		So(cerr, ShouldBeNil)
		So(resp, ShouldResembleProto, drac1)

		dracs := make([]*ufspb.Drac, 0, 1)
		dracs = append(dracs, drac1)

		dracs1 := make([]*ufspb.Drac, 0, 1)
		dracs1 = append(dracs1, dummyDrac)
		Convey("Query By existing Switch keysonly", func() {
			resp, err := QueryDracByPropertyName(ctx, "switch_id", "switch-1", true)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, dracs1)
		})
		Convey("Query By non-existing Switch", func() {
			resp, err := QueryDracByPropertyName(ctx, "switch_id", "switch-2", true)
			So(err, ShouldBeNil)
			So(resp, ShouldBeNil)
		})
		Convey("Query By existing Switch", func() {
			resp, err := QueryDracByPropertyName(ctx, "switch_id", "switch-1", false)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, dracs)
		})
	})
}
