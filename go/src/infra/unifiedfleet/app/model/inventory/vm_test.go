// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package inventory

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/luci/appengine/gaetesting"
	. "go.chromium.org/luci/common/testing/assertions"
	"go.chromium.org/luci/gae/service/datastore"

	ufspb "infra/unifiedfleet/api/v1/proto"
	. "infra/unifiedfleet/app/model/datastore"
)

func mockVM(id string) *ufspb.VM {
	return &ufspb.VM{
		Name: id,
	}
}

func TestBatchUpdateVMs(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	vm1 := mockVM("vm-1")
	vm2 := mockVM("vm-2")
	vm3 := mockVM("")
	Convey("Batch Update VM", t, func() {
		Convey("BatchUpdate all vms", func() {
			resp, err := BatchUpdateVMs(ctx, []*ufspb.VM{vm1, vm2})
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, []*ufspb.VM{vm1, vm2})
		})
		Convey("BatchUpdate existing vms", func() {
			vm2.MacAddress = "123"
			_, err := BatchUpdateVMs(ctx, []*ufspb.VM{vm1, vm2})
			So(err, ShouldBeNil)
			vm, err := GetVM(ctx, "vm-2")
			So(err, ShouldBeNil)
			So(vm.GetMacAddress(), ShouldEqual, "123")
		})
		Convey("BatchUpdate invalid vms", func() {
			resp, err := BatchUpdateVMs(ctx, []*ufspb.VM{vm1, vm2, vm3})
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}

func TestGetVM(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	Convey("GetVM", t, func() {
		Convey("Get machineLSE by non-existing ID", func() {
			resp, err := GetMachineLSE(ctx, "empty")
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

func TestListVMs(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	datastore.GetTestable(ctx).Consistent(true)
	vms := make([]*ufspb.VM, 0, 4)
	for i := 0; i < 4; i++ {
		vm := mockVM(fmt.Sprintf("vm-%d", i))
		vms = append(vms, vm)
	}
	Convey("ListVMs", t, func() {
		_, err := BatchUpdateVMs(ctx, vms)
		So(err, ShouldBeNil)
		Convey("List vms - page_token invalid", func() {
			resp, nextPageToken, err := ListVMs(ctx, 5, 5, "abc", nil, false, nil)
			So(resp, ShouldBeNil)
			So(nextPageToken, ShouldBeEmpty)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InvalidPageToken)
		})

		Convey("List vms - Full listing with no pagination", func() {
			resp, nextPageToken, err := ListVMs(ctx, 4, 4, "", nil, false, nil)
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(nextPageToken, ShouldNotBeEmpty)
			So(resp, ShouldResembleProto, vms)
		})

		Convey("List vms - listing with pagination", func() {
			resp, nextPageToken, err := ListVMs(ctx, 3, 3, "", nil, false, nil)
			So(resp, ShouldNotBeNil)
			So(nextPageToken, ShouldNotBeEmpty)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, vms[:3])

			resp, _, err = ListVMs(ctx, 2, 2, nextPageToken, nil, false, nil)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, vms[3:])
		})
	})
}

func TestDeleteVMs(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	vm1 := mockVM("vm-delete1")
	Convey("DeleteVMs", t, func() {
		Convey("Delete VM by existing ID", func() {
			_, err := BatchUpdateVMs(ctx, []*ufspb.VM{vm1})
			So(err, ShouldBeNil)
			DeleteVMs(ctx, []string{"vm-delete1"})
			vm, err := GetVM(ctx, "vm-delete1")
			So(vm, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Delete vms by non-existing ID", func() {
			res := DeleteVMs(ctx, []string{"vm-delete2"})
			So(res.Failed(), ShouldHaveLength, 1)
		})
		Convey("Delete machineLSE - invalid ID", func() {
			res := DeleteVMs(ctx, []string{""})
			So(res.Failed(), ShouldHaveLength, 1)
		})
	})
}
