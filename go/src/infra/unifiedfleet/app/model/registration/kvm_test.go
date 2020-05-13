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
	"infra/unifiedfleet/app/model/inventory"
)

func mockKVM(id string) *proto.KVM {
	return &proto.KVM{
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
	Convey("ListKVMs", t, func() {
		ctx := gaetesting.TestingContextWithAppID("go-test")
		datastore.GetTestable(ctx).Consistent(true)
		KVMs := make([]*proto.KVM, 0, 4)
		for i := 0; i < 4; i++ {
			KVM1 := mockKVM(fmt.Sprintf("KVM-%d", i))
			resp, err := CreateKVM(ctx, KVM1)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, KVM1)
			KVMs = append(KVMs, resp)
		}
		Convey("List KVMs - page_token invalid", func() {
			resp, nextPageToken, err := ListKVMs(ctx, 5, "abc")
			So(resp, ShouldBeNil)
			So(nextPageToken, ShouldBeEmpty)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InvalidPageToken)
		})

		Convey("List KVMs - Full listing with no pagination", func() {
			resp, nextPageToken, err := ListKVMs(ctx, 4, "")
			So(resp, ShouldNotBeNil)
			So(nextPageToken, ShouldNotBeEmpty)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, KVMs)
		})

		Convey("List KVMs - listing with pagination", func() {
			resp, nextPageToken, err := ListKVMs(ctx, 3, "")
			So(resp, ShouldNotBeNil)
			So(nextPageToken, ShouldNotBeEmpty)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, KVMs[:3])

			resp, _, err = ListKVMs(ctx, 2, nextPageToken)
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
	KVM1 := mockKVM("KVM-1")
	KVM2 := mockKVM("KVM-2")
	KVM3 := mockKVM("KVM-3")
	KVM4 := mockKVM("KVM-4")
	Convey("DeleteKVM", t, func() {
		Convey("Delete KVM by existing ID with machine reference", func() {
			resp, cerr := CreateKVM(ctx, KVM1)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, KVM1)

			chromeBrowserMachine1 := &proto.Machine{
				Name: "machine-1",
				Device: &proto.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &proto.ChromeBrowserMachine{
						KvmInterface: &proto.KVMInterface{
							Kvm: "KVM-1",
						},
					},
				},
			}
			mresp, merr := CreateMachine(ctx, chromeBrowserMachine1)
			So(merr, ShouldBeNil)
			So(mresp, ShouldResembleProto, chromeBrowserMachine1)

			err := DeleteKVM(ctx, "KVM-1")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, CannotDelete)

			resp, cerr = GetKVM(ctx, "KVM-1")
			So(resp, ShouldNotBeNil)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, KVM1)
		})
		Convey("Delete KVM by existing ID with rack reference", func() {
			resp, cerr := CreateKVM(ctx, KVM2)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, KVM2)

			chromeBrowserRack1 := &proto.Rack{
				Name: "rack-1",
				Rack: &proto.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &proto.ChromeBrowserRack{
						Kvms: []string{"KVM-2", "KVM-5"},
					},
				},
			}
			mresp, merr := CreateRack(ctx, chromeBrowserRack1)
			So(merr, ShouldBeNil)
			So(mresp, ShouldResembleProto, chromeBrowserRack1)

			err := DeleteKVM(ctx, "KVM-2")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, CannotDelete)

			resp, cerr = GetKVM(ctx, "KVM-2")
			So(resp, ShouldNotBeNil)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, KVM2)
		})
		Convey("Delete KVM by existing ID with racklse reference", func() {
			resp, cerr := CreateKVM(ctx, KVM3)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, KVM3)

			chromeOSRackLSE1 := &proto.RackLSE{
				Name: "racklse-1",
				Lse: &proto.RackLSE_ChromeosRackLse{
					ChromeosRackLse: &proto.ChromeOSRackLSE{
						Kvms: []string{"KVM-3", "KVM-5"},
					},
				},
			}
			mresp, merr := inventory.CreateRackLSE(ctx, chromeOSRackLSE1)
			So(merr, ShouldBeNil)
			So(mresp, ShouldResembleProto, chromeOSRackLSE1)

			err := DeleteKVM(ctx, "KVM-3")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, CannotDelete)

			resp, cerr = GetKVM(ctx, "KVM-3")
			So(resp, ShouldNotBeNil)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, KVM3)
		})
		Convey("Delete KVM successfully by existing ID without references", func() {
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
		kvms := make([]*proto.KVM, 0, 4)
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
		dummyKVM := &proto.KVM{
			Name: "kvm-15",
		}
		kvm1 := mockKVM("kvm-15")
		kvm1.ChromePlatform = "chromePlatform-1"
		resp, cerr := CreateKVM(ctx, kvm1)
		So(cerr, ShouldBeNil)
		So(resp, ShouldResembleProto, kvm1)

		kvms := make([]*proto.KVM, 0, 1)
		kvms = append(kvms, kvm1)

		kvms1 := make([]*proto.KVM, 0, 1)
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
