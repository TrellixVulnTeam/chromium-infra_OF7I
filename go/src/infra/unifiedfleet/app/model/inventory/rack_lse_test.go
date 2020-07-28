// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package inventory

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

func mockRackLSE(id string) *ufspb.RackLSE {
	return &ufspb.RackLSE{
		Name: id,
	}
}

func TestCreateRackLSE(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	datastore.GetTestable(ctx).Consistent(true)
	rackLSE1 := mockRackLSE("rackLSE-1")
	rackLSE2 := mockRackLSE("")
	Convey("CreateRackLSE", t, func() {
		Convey("Create new rackLSE", func() {
			resp, err := CreateRackLSE(ctx, rackLSE1)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, rackLSE1)
		})
		Convey("Create existing rackLSE", func() {
			resp, err := CreateRackLSE(ctx, rackLSE1)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, AlreadyExists)
		})
		Convey("Create rackLSE - invalid ID", func() {
			resp, err := CreateRackLSE(ctx, rackLSE2)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}

func TestUpdateRackLSE(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	rackLSE1 := mockRackLSE("rackLSE-1")
	rackLSE2 := mockRackLSE("rackLSE-1")
	rackLSE3 := mockRackLSE("rackLSE-3")
	rackLSE4 := mockRackLSE("")
	Convey("UpdateRackLSE", t, func() {
		Convey("Update existing rackLSE", func() {
			resp, err := CreateRackLSE(ctx, rackLSE1)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, rackLSE1)

			resp, err = UpdateRackLSE(ctx, rackLSE2)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, rackLSE2)
		})
		Convey("Update non-existing rackLSE", func() {
			resp, err := UpdateRackLSE(ctx, rackLSE3)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Update rackLSE - invalid ID", func() {
			resp, err := UpdateRackLSE(ctx, rackLSE4)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}

func TestGetRackLSE(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	rackLSE1 := mockRackLSE("rackLSE-1")
	Convey("GetRackLSE", t, func() {
		Convey("Get rackLSE by existing ID", func() {
			resp, err := CreateRackLSE(ctx, rackLSE1)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, rackLSE1)
			resp, err = GetRackLSE(ctx, "rackLSE-1")
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, rackLSE1)
		})
		Convey("Get rackLSE by non-existing ID", func() {
			resp, err := GetRackLSE(ctx, "rackLSE-2")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Get rackLSE - invalid ID", func() {
			resp, err := GetRackLSE(ctx, "")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}

func TestListRackLSEs(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	datastore.GetTestable(ctx).Consistent(true)
	rackLSEs := make([]*ufspb.RackLSE, 0, 4)
	for i := 0; i < 4; i++ {
		rackLSE1 := mockRackLSE(fmt.Sprintf("rackLSE-%d", i))
		resp, _ := CreateRackLSE(ctx, rackLSE1)
		rackLSEs = append(rackLSEs, resp)
	}
	Convey("ListRackLSEs", t, func() {
		Convey("List rackLSEs - page_token invalid", func() {
			resp, nextPageToken, err := ListRackLSEs(ctx, 5, "abc", nil, false)
			So(resp, ShouldBeNil)
			So(nextPageToken, ShouldBeEmpty)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InvalidPageToken)
		})

		Convey("List rackLSEs - Full listing with no pagination", func() {
			resp, nextPageToken, err := ListRackLSEs(ctx, 4, "", nil, false)
			So(resp, ShouldNotBeNil)
			So(nextPageToken, ShouldNotBeEmpty)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, rackLSEs)
		})

		Convey("List rackLSEs - listing with pagination", func() {
			resp, nextPageToken, err := ListRackLSEs(ctx, 3, "", nil, false)
			So(resp, ShouldNotBeNil)
			So(nextPageToken, ShouldNotBeEmpty)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, rackLSEs[:3])

			resp, _, err = ListRackLSEs(ctx, 2, nextPageToken, nil, false)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, rackLSEs[3:])
		})
	})
}

func TestDeleteRackLSE(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	rackLSE1 := mockRackLSE("rackLSE-1")
	Convey("DeleteRackLSE", t, func() {
		Convey("Delete rackLSE by existing ID", func() {
			resp, cerr := CreateRackLSE(ctx, rackLSE1)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, rackLSE1)
			err := DeleteRackLSE(ctx, "rackLSE-1")
			So(err, ShouldBeNil)
			res, err := GetRackLSE(ctx, "rackLSE-1")
			So(res, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Delete rackLSE by non-existing ID", func() {
			err := DeleteRackLSE(ctx, "rackLSE-2")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Delete rackLSE - invalid ID", func() {
			err := DeleteRackLSE(ctx, "")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}

func TestBatchUpdateRackLSEs(t *testing.T) {
	t.Parallel()
	Convey("BatchUpdateRackLSEs", t, func() {
		ctx := gaetesting.TestingContextWithAppID("go-test")
		datastore.GetTestable(ctx).Consistent(true)
		rackLSEs := make([]*ufspb.RackLSE, 0, 4)
		for i := 0; i < 4; i++ {
			rackLSE1 := mockRackLSE(fmt.Sprintf("rackLSE-%d", i))
			resp, err := CreateRackLSE(ctx, rackLSE1)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, rackLSE1)
			rackLSEs = append(rackLSEs, resp)
		}
		Convey("BatchUpdate all rackLSEs", func() {
			resp, err := BatchUpdateRackLSEs(ctx, rackLSEs)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, rackLSEs)
		})
		Convey("BatchUpdate existing and invalid rackLSEs", func() {
			rackLSE5 := mockRackLSE("")
			rackLSEs = append(rackLSEs, rackLSE5)
			resp, err := BatchUpdateRackLSEs(ctx, rackLSEs)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}

func TestQueryRackLSEByPropertyName(t *testing.T) {
	t.Parallel()
	Convey("QueryRackLSEByPropertyName", t, func() {
		ctx := gaetesting.TestingContextWithAppID("go-test")
		datastore.GetTestable(ctx).Consistent(true)
		dummyrackLSE := &ufspb.RackLSE{
			Name: "rackLSE-1",
		}
		rackLSE1 := &ufspb.RackLSE{
			Name:             "rackLSE-1",
			Racks:            []string{"rack-1", "rack-2"},
			RackLsePrototype: "rackLsePrototype-1",
		}
		resp, cerr := CreateRackLSE(ctx, rackLSE1)
		So(cerr, ShouldBeNil)
		So(resp, ShouldResembleProto, rackLSE1)

		rackLSEs := make([]*ufspb.RackLSE, 0, 1)
		rackLSEs = append(rackLSEs, rackLSE1)

		dummyrackLSEs := make([]*ufspb.RackLSE, 0, 1)
		dummyrackLSEs = append(dummyrackLSEs, dummyrackLSE)
		Convey("Query By existing Rack", func() {
			resp, err := QueryRackLSEByPropertyName(ctx, "rack_ids", "rack-1", false)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, rackLSEs)
		})
		Convey("Query By non-existing Rack", func() {
			resp, err := QueryRackLSEByPropertyName(ctx, "rack_ids", "rack-5", false)
			So(err, ShouldBeNil)
			So(resp, ShouldBeNil)
		})
		Convey("Query By existing RackLsePrototype keysonly", func() {
			resp, err := QueryRackLSEByPropertyName(ctx, "racklse_prototype_id", "rackLsePrototype-1", true)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, dummyrackLSEs)
		})
		Convey("Query By non-existing RackLsePrototype", func() {
			resp, err := QueryRackLSEByPropertyName(ctx, "racklse_prototype_id", "rackLsePrototype-2", true)
			So(err, ShouldBeNil)
			So(resp, ShouldBeNil)
		})
	})
}
