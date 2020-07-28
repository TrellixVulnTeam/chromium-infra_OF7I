// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package configuration

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

func mockVlan(id string) *ufspb.Vlan {
	return &ufspb.Vlan{
		Name: id,
	}
}

func TestCreateVlan(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	datastore.GetTestable(ctx).Consistent(true)
	vlan1 := mockVlan("Vlan-1")
	vlan2 := mockVlan("")
	Convey("CreateVlan", t, func() {
		Convey("Create new vlan", func() {
			resp, err := CreateVlan(ctx, vlan1)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, vlan1)
		})
		Convey("Create existing vlan", func() {
			resp, err := CreateVlan(ctx, vlan1)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, AlreadyExists)
		})
		Convey("Create vlan - invalid ID", func() {
			resp, err := CreateVlan(ctx, vlan2)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}

func TestUpdateVlan(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	vlan1 := mockVlan("Vlan-1")
	vlan2 := mockVlan("Vlan-1")
	vlan3 := mockVlan("Vlan-3")
	vlan4 := mockVlan("")
	Convey("UpdateVlan", t, func() {
		Convey("Update existing vlan", func() {
			resp, err := CreateVlan(ctx, vlan1)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, vlan1)

			resp, err = UpdateVlan(ctx, vlan2)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, vlan2)
		})
		Convey("Update non-existing vlan", func() {
			resp, err := UpdateVlan(ctx, vlan3)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Update vlan - invalid ID", func() {
			resp, err := UpdateVlan(ctx, vlan4)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}

func TestGetVlan(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	vlan1 := mockVlan("Vlan-1")
	Convey("GetVlan", t, func() {
		Convey("Get vlan by existing ID", func() {
			resp, err := CreateVlan(ctx, vlan1)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, vlan1)
			resp, err = GetVlan(ctx, "Vlan-1")
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, vlan1)
		})
		Convey("Get vlan by non-existing ID", func() {
			resp, err := GetVlan(ctx, "vlan-2")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Get vlan - invalid ID", func() {
			resp, err := GetVlan(ctx, "")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}

func TestListVlans(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	datastore.GetTestable(ctx).Consistent(true)
	vlans := make([]*ufspb.Vlan, 0, 4)
	for i := 0; i < 4; i++ {
		vlan1 := mockVlan(fmt.Sprintf("vlan-%d", i))
		resp, _ := CreateVlan(ctx, vlan1)
		vlans = append(vlans, resp)
	}
	Convey("ListVlans", t, func() {
		Convey("List vlans - page_token invalid", func() {
			resp, nextPageToken, err := ListVlans(ctx, 5, "abc", nil, false)
			So(resp, ShouldBeNil)
			So(nextPageToken, ShouldBeEmpty)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InvalidPageToken)
		})

		Convey("List vlans - Full listing with no pagination", func() {
			resp, nextPageToken, err := ListVlans(ctx, 4, "", nil, false)
			So(resp, ShouldNotBeNil)
			So(nextPageToken, ShouldNotBeEmpty)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, vlans)
		})

		Convey("List vlans - listing with pagination", func() {
			resp, nextPageToken, err := ListVlans(ctx, 3, "", nil, false)
			So(resp, ShouldNotBeNil)
			So(nextPageToken, ShouldNotBeEmpty)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, vlans[:3])

			resp, _, err = ListVlans(ctx, 2, nextPageToken, nil, false)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, vlans[3:])
		})
	})
}

func TestDeleteVlan(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	datastore.GetTestable(ctx).Consistent(true)
	vlan2 := mockVlan("vlan-2")
	Convey("DeleteVlan", t, func() {
		Convey("Delete vlan successfully by existing ID", func() {
			resp, cerr := CreateVlan(ctx, vlan2)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, vlan2)

			err := DeleteVlan(ctx, "vlan-2")
			So(err, ShouldBeNil)

			resp, cerr = GetVlan(ctx, "vlan-2")
			So(resp, ShouldBeNil)
			So(cerr, ShouldNotBeNil)
			So(cerr.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Delete vlan by non-existing ID", func() {
			err := DeleteVlan(ctx, "vlan-2")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Delete vlan - invalid ID", func() {
			err := DeleteVlan(ctx, "")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}
