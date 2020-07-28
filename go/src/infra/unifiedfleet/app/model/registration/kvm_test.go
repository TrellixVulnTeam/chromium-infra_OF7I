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

func mockKVM(id string) *ufspb.KVM {
	return &ufspb.KVM{
		Name: id,
	}
}

func TestCreateKVM(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	datastore.GetTestable(ctx).Consistent(true)
	KVM1 := mockKVM("KVM-1")
	KVM2 := mockKVM("")
	Convey("CreateKVM", t, func() {
		Convey("Create new KVM", func() {
			resp, err := CreateKVM(ctx, KVM1)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, KVM1)
		})
		Convey("Create existing KVM", func() {
			resp, err := CreateKVM(ctx, KVM1)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, AlreadyExists)
		})
		Convey("Create KVM - invalid ID", func() {
			resp, err := CreateKVM(ctx, KVM2)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}

func TestUpdateKVM(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	KVM1 := mockKVM("KVM-1")
	KVM2 := mockKVM("KVM-1")
	KVM3 := mockKVM("KVM-3")
	KVM4 := mockKVM("")
	Convey("UpdateKVM", t, func() {
		Convey("Update existing KVM", func() {
			resp, err := CreateKVM(ctx, KVM1)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, KVM1)

			resp, err = UpdateKVM(ctx, KVM2)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, KVM2)
		})
		Convey("Update non-existing KVM", func() {
			resp, err := UpdateKVM(ctx, KVM3)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Update KVM - invalid ID", func() {
			resp, err := UpdateKVM(ctx, KVM4)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}

func TestGetKVM(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	KVM1 := mockKVM("KVM-1")
	Convey("GetKVM", t, func() {
		Convey("Get KVM by existing ID", func() {
			resp, err := CreateKVM(ctx, KVM1)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, KVM1)
			resp, err = GetKVM(ctx, "KVM-1")
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, KVM1)
		})
		Convey("Get KVM by non-existing ID", func() {
			resp, err := GetKVM(ctx, "KVM-2")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Get KVM - invalid ID", func() {
			resp, err := GetKVM(ctx, "")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}

func TestListKVMs(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	datastore.GetTestable(ctx).Consistent(true)
	KVMs := make([]*ufspb.KVM, 0, 4)
	for i := 0; i < 4; i++ {
		KVM1 := mockKVM(fmt.Sprintf("KVM-%d", i))
		resp, _ := CreateKVM(ctx, KVM1)
		KVMs = append(KVMs, resp)
	}
	Convey("ListKVMs", t, func() {
		Convey("List KVMs - page_token invalid", func() {
			resp, nextPageToken, err := ListKVMs(ctx, 5, "abc", nil, false)
			So(resp, ShouldBeNil)
			So(nextPageToken, ShouldBeEmpty)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InvalidPageToken)
		})

		Convey("List KVMs - Full listing with no pagination", func() {
			resp, nextPageToken, err := ListKVMs(ctx, 4, "", nil, false)
			So(resp, ShouldNotBeNil)
			So(nextPageToken, ShouldNotBeEmpty)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, KVMs)
		})

		Convey("List KVMs - listing with pagination", func() {
			resp, nextPageToken, err := ListKVMs(ctx, 3, "", nil, false)
			So(resp, ShouldNotBeNil)
			So(nextPageToken, ShouldNotBeEmpty)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, KVMs[:3])

			resp, _, err = ListKVMs(ctx, 2, nextPageToken, nil, false)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, KVMs[3:])
		})
	})
}

func TestDeleteKVM(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	datastore.GetTestable(ctx).Consistent(true)
	KVM4 := mockKVM("KVM-4")
	Convey("DeleteKVM", t, func() {
		Convey("Delete KVM successfully by existing ID", func() {
			resp, cerr := CreateKVM(ctx, KVM4)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, KVM4)

			err := DeleteKVM(ctx, "KVM-4")
			So(err, ShouldBeNil)

			resp, cerr = GetKVM(ctx, "KVM-4")
			So(resp, ShouldBeNil)
			So(cerr, ShouldNotBeNil)
			So(cerr.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Delete KVM by non-existing ID", func() {
			err := DeleteKVM(ctx, "KVM-6")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Delete KVM - invalid ID", func() {
			err := DeleteKVM(ctx, "")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}

func TestBatchUpdateKVMs(t *testing.T) {
	t.Parallel()
	Convey("BatchUpdateKVMs", t, func() {
		ctx := gaetesting.TestingContextWithAppID("go-test")
		datastore.GetTestable(ctx).Consistent(true)
		kvms := make([]*ufspb.KVM, 0, 4)
		for i := 0; i < 4; i++ {
			kvm1 := mockKVM(fmt.Sprintf("kvm-%d", i))
			kvm1.ChromePlatform = "chromePlatform-1"
			resp, err := CreateKVM(ctx, kvm1)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, kvm1)
			kvms = append(kvms, resp)
		}
		Convey("BatchUpdate all kvms", func() {
			resp, err := BatchUpdateKVMs(ctx, kvms)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, kvms)
		})
		Convey("BatchUpdate existing and non-existing kvms", func() {
			KVM5 := mockKVM("")
			kvms = append(kvms, KVM5)
			resp, err := BatchUpdateKVMs(ctx, kvms)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}

func TestQueryKVMByPropertyName(t *testing.T) {
	t.Parallel()
	Convey("QueryKVMByPropertyName", t, func() {
		ctx := gaetesting.TestingContextWithAppID("go-test")
		datastore.GetTestable(ctx).Consistent(true)
		dummyKVM := &ufspb.KVM{
			Name: "kvm-15",
		}
		kvm1 := mockKVM("kvm-15")
		kvm1.ChromePlatform = "chromePlatform-1"
		resp, cerr := CreateKVM(ctx, kvm1)
		So(cerr, ShouldBeNil)
		So(resp, ShouldResembleProto, kvm1)

		kvms := make([]*ufspb.KVM, 0, 1)
		kvms = append(kvms, kvm1)

		kvms1 := make([]*ufspb.KVM, 0, 1)
		kvms1 = append(kvms1, dummyKVM)
		Convey("Query By existing ChromePlatform keysonly", func() {
			resp, err := QueryKVMByPropertyName(ctx, "chrome_platform_id", "chromePlatform-1", true)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, kvms1)
		})
		Convey("Query By non-existing ChromePlatform", func() {
			resp, err := QueryKVMByPropertyName(ctx, "chrome_platform_id", "chromePlatform-2", true)
			So(err, ShouldBeNil)
			So(resp, ShouldBeNil)
		})
		Convey("Query By existing ChromePlatform", func() {
			resp, err := QueryKVMByPropertyName(ctx, "chrome_platform_id", "chromePlatform-1", false)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, kvms)
		})
	})
}
