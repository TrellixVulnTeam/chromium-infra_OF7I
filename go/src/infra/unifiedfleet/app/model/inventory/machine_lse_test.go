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

func mockMachineLSE(id string) *ufspb.MachineLSE {
	return &ufspb.MachineLSE{
		Name: id,
	}
}

func TestCreateMachineLSE(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	datastore.GetTestable(ctx).Consistent(true)
	machineLSE1 := mockMachineLSE("machineLSE-1")
	machineLSE2 := mockMachineLSE("")
	Convey("CreateMachineLSE", t, func() {
		Convey("Create new machineLSE", func() {
			resp, err := CreateMachineLSE(ctx, machineLSE1)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, machineLSE1)
		})
		Convey("Create existing machineLSE", func() {
			resp, err := CreateMachineLSE(ctx, machineLSE1)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, AlreadyExists)
		})
		Convey("Create machineLSE - invalid ID", func() {
			resp, err := CreateMachineLSE(ctx, machineLSE2)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}

func TestUpdateMachineLSE(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	machineLSE1 := mockMachineLSE("machineLSE-1")
	machineLSE2 := mockMachineLSE("machineLSE-1")
	machineLSE2.Hostname = "Linux Server"
	machineLSE3 := mockMachineLSE("machineLSE-3")
	machineLSE4 := mockMachineLSE("")
	Convey("UpdateMachineLSE", t, func() {
		Convey("Update existing machineLSE", func() {
			resp, err := CreateMachineLSE(ctx, machineLSE1)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, machineLSE1)

			resp, err = UpdateMachineLSE(ctx, machineLSE2)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, machineLSE2)
		})
		Convey("Update non-existing machineLSE", func() {
			resp, err := UpdateMachineLSE(ctx, machineLSE3)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Update machineLSE - invalid ID", func() {
			resp, err := UpdateMachineLSE(ctx, machineLSE4)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}

func TestGetMachineLSE(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	machineLSE1 := mockMachineLSE("machineLSE-1")
	Convey("GetMachineLSE", t, func() {
		Convey("Get machineLSE by existing ID", func() {
			resp, err := CreateMachineLSE(ctx, machineLSE1)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, machineLSE1)
			resp, err = GetMachineLSE(ctx, "machineLSE-1")
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, machineLSE1)
		})
		Convey("Get machineLSE by non-existing ID", func() {
			resp, err := GetMachineLSE(ctx, "machineLSE-2")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Get machineLSE - invalid ID", func() {
			resp, err := GetMachineLSE(ctx, "")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}

func TestListMachineLSEs(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	datastore.GetTestable(ctx).Consistent(true)
	machineLSEs := make([]*ufspb.MachineLSE, 0, 4)
	for i := 0; i < 4; i++ {
		machineLSE1 := mockMachineLSE(fmt.Sprintf("machineLSE-%d", i))
		resp, _ := CreateMachineLSE(ctx, machineLSE1)
		machineLSEs = append(machineLSEs, resp)
	}
	Convey("ListMachineLSEs", t, func() {
		Convey("List machineLSEs - page_token invalid", func() {
			resp, nextPageToken, err := ListMachineLSEs(ctx, 5, 5, "abc", nil, false, nil)
			So(resp, ShouldBeNil)
			So(nextPageToken, ShouldBeEmpty)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InvalidPageToken)
		})

		Convey("List machineLSEs - Full listing with no pagination", func() {
			resp, nextPageToken, err := ListMachineLSEs(ctx, 4, 4, "", nil, false, nil)
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(nextPageToken, ShouldNotBeEmpty)
			So(resp, ShouldResembleProto, machineLSEs)
		})

		Convey("List machineLSEs - listing with pagination", func() {
			resp, nextPageToken, err := ListMachineLSEs(ctx, 3, 3, "", nil, false, nil)
			So(resp, ShouldNotBeNil)
			So(nextPageToken, ShouldNotBeEmpty)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, machineLSEs[:3])

			resp, _, err = ListMachineLSEs(ctx, 2, 2, nextPageToken, nil, false, nil)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, machineLSEs[3:])
		})
	})
}

func TestDeleteMachineLSE(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	machineLSE1 := mockMachineLSE("machineLSE-1")
	Convey("DeleteMachineLSE", t, func() {
		Convey("Delete machineLSE by existing ID", func() {
			resp, cerr := CreateMachineLSE(ctx, machineLSE1)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, machineLSE1)
			err := DeleteMachineLSE(ctx, "machineLSE-1")
			So(err, ShouldBeNil)
			res, err := GetMachineLSE(ctx, "machineLSE-1")
			So(res, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Delete machineLSE by non-existing ID", func() {
			err := DeleteMachineLSE(ctx, "machineLSE-2")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Delete machineLSE - invalid ID", func() {
			err := DeleteMachineLSE(ctx, "")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}

func TestBatchUpdateMachineLSEs(t *testing.T) {
	t.Parallel()
	Convey("BatchUpdateMachineLSEs", t, func() {
		ctx := gaetesting.TestingContextWithAppID("go-test")
		datastore.GetTestable(ctx).Consistent(true)
		machineLSEs := make([]*ufspb.MachineLSE, 0, 4)
		for i := 0; i < 4; i++ {
			machineLSE1 := mockMachineLSE(fmt.Sprintf("machineLSE-%d", i))
			resp, err := CreateMachineLSE(ctx, machineLSE1)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, machineLSE1)
			machineLSEs = append(machineLSEs, resp)
		}
		Convey("BatchUpdate all machineLSEs", func() {
			resp, err := BatchUpdateMachineLSEs(ctx, machineLSEs)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, machineLSEs)
		})
		Convey("BatchUpdate existing and invalid machineLSEs", func() {
			machineLSE5 := mockMachineLSE("")
			machineLSEs = append(machineLSEs, machineLSE5)
			resp, err := BatchUpdateMachineLSEs(ctx, machineLSEs)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}

func TestQueryMachineLSEByPropertyName(t *testing.T) {
	t.Parallel()
	Convey("QueryMachineLSEByPropertyName", t, func() {
		ctx := gaetesting.TestingContextWithAppID("go-test")
		datastore.GetTestable(ctx).Consistent(true)
		dummymachineLSE := &ufspb.MachineLSE{
			Name: "machineLSE-1",
		}
		machineLSE1 := &ufspb.MachineLSE{
			Name:                "machineLSE-1",
			Machines:            []string{"machine-1", "machine-2"},
			MachineLsePrototype: "machineLsePrototype-1",
		}
		resp, cerr := CreateMachineLSE(ctx, machineLSE1)
		So(cerr, ShouldBeNil)
		So(resp, ShouldResembleProto, machineLSE1)

		machineLSEs := make([]*ufspb.MachineLSE, 0, 1)
		machineLSEs = append(machineLSEs, machineLSE1)

		dummymachineLSEs := make([]*ufspb.MachineLSE, 0, 1)
		dummymachineLSEs = append(dummymachineLSEs, dummymachineLSE)
		Convey("Query By existing Machine", func() {
			resp, err := QueryMachineLSEByPropertyName(ctx, "machine_ids", "machine-1", false)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, machineLSEs)
		})
		Convey("Query By non-existing Machine", func() {
			resp, err := QueryMachineLSEByPropertyName(ctx, "machine_ids", "machine-5", false)
			So(err, ShouldBeNil)
			So(resp, ShouldBeNil)
		})
		Convey("Query By existing MachineLsePrototype keysonly", func() {
			resp, err := QueryMachineLSEByPropertyName(ctx, "machinelse_prototype_id", "machineLsePrototype-1", true)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, dummymachineLSEs)
		})
		Convey("Query By non-existing MachineLsePrototype", func() {
			resp, err := QueryMachineLSEByPropertyName(ctx, "machinelse_prototype_id", "machineLsePrototype-2", true)
			So(err, ShouldBeNil)
			So(resp, ShouldBeNil)
		})
	})
}
