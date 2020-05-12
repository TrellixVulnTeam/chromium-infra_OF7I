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
	proto "infra/unifiedfleet/api/v1/proto"
	. "infra/unifiedfleet/app/model/datastore"
)

func mockNic(id string) *proto.Nic {
	return &proto.Nic{
		Name: id,
	}
}

func TestCreateNic(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	datastore.GetTestable(ctx).Consistent(true)
	nic1 := mockNic("Nic-1")
	nic2 := mockNic("")
	Convey("CreateNic", t, func() {
		Convey("Create new nic", func() {
			resp, err := CreateNic(ctx, nic1)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, nic1)
		})
		Convey("Create existing nic", func() {
			resp, err := CreateNic(ctx, nic1)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, AlreadyExists)
		})
		Convey("Create nic - invalid ID", func() {
			resp, err := CreateNic(ctx, nic2)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}

func TestUpdateNic(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	nic1 := mockNic("Nic-1")
	nic2 := mockNic("Nic-1")
	nic3 := mockNic("Nic-3")
	nic4 := mockNic("")
	Convey("UpdateNic", t, func() {
		Convey("Update existing nic", func() {
			resp, err := CreateNic(ctx, nic1)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, nic1)

			resp, err = UpdateNic(ctx, nic2)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, nic2)
		})
		Convey("Update non-existing nic", func() {
			resp, err := UpdateNic(ctx, nic3)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Update nic - invalid ID", func() {
			resp, err := UpdateNic(ctx, nic4)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}

func TestGetNic(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	nic1 := mockNic("Nic-1")
	Convey("GetNic", t, func() {
		Convey("Get nic by existing ID", func() {
			resp, err := CreateNic(ctx, nic1)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, nic1)
			resp, err = GetNic(ctx, "Nic-1")
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, nic1)
		})
		Convey("Get nic by non-existing ID", func() {
			resp, err := GetNic(ctx, "nic-2")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Get nic - invalid ID", func() {
			resp, err := GetNic(ctx, "")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}

func TestListNics(t *testing.T) {
	t.Parallel()
	Convey("ListNics", t, func() {
		ctx := gaetesting.TestingContextWithAppID("go-test")
		datastore.GetTestable(ctx).Consistent(true)
		nics := make([]*proto.Nic, 0, 4)
		for i := 0; i < 4; i++ {
			nic1 := mockNic(fmt.Sprintf("nic-%d", i))
			resp, err := CreateNic(ctx, nic1)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, nic1)
			nics = append(nics, resp)
		}
		Convey("List nics - page_token invalid", func() {
			resp, nextPageToken, err := ListNics(ctx, 5, "abc")
			So(resp, ShouldBeNil)
			So(nextPageToken, ShouldBeEmpty)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InvalidPageToken)
		})

		Convey("List nics - Full listing with no pagination", func() {
			resp, nextPageToken, err := ListNics(ctx, 4, "")
			So(resp, ShouldNotBeNil)
			So(nextPageToken, ShouldNotBeEmpty)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, nics)
		})

		Convey("List nics - listing with pagination", func() {
			resp, nextPageToken, err := ListNics(ctx, 3, "")
			So(resp, ShouldNotBeNil)
			So(nextPageToken, ShouldNotBeEmpty)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, nics[:3])

			resp, _, err = ListNics(ctx, 2, nextPageToken)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, nics[3:])
		})
	})
}

func TestDeleteNic(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	datastore.GetTestable(ctx).Consistent(true)
	nic1 := mockNic("nic-1")
	nic2 := mockNic("nic-2")
	Convey("DeleteNic", t, func() {
		Convey("Delete nic by existing ID with machine reference", func() {
			resp, cerr := CreateNic(ctx, nic1)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, nic1)

			chromeBrowserMachine1 := &proto.Machine{
				Name: "machine-1",
				Device: &proto.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &proto.ChromeBrowserMachine{
						Nic: "nic-1",
					},
				},
			}
			mresp, merr := CreateMachine(ctx, chromeBrowserMachine1)
			So(merr, ShouldBeNil)
			So(mresp, ShouldResembleProto, chromeBrowserMachine1)

			err := DeleteNic(ctx, "nic-1")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, CannotDelete)

			resp, cerr = GetNic(ctx, "nic-1")
			So(resp, ShouldNotBeNil)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, nic1)
		})
		Convey("Delete nic successfully by existing ID without references", func() {
			resp, cerr := CreateNic(ctx, nic2)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, nic2)

			err := DeleteNic(ctx, "nic-2")
			So(err, ShouldBeNil)

			resp, cerr = GetNic(ctx, "nic-2")
			So(resp, ShouldBeNil)
			So(cerr, ShouldNotBeNil)
			So(cerr.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Delete nic by non-existing ID", func() {
			err := DeleteNic(ctx, "nic-2")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Delete nic - invalid ID", func() {
			err := DeleteNic(ctx, "")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}
