// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package registration

/*
TODO(eshwarn) : Remove comment when kvm create/get is added
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

func mockKVM(id, chromePlatform string) *proto.KVM {
	return &proto.KVM{
		Name:           id,
		ChromePlatform: chromePlatform,
	}
}

func TestBatchUpdateKVMs(t *testing.T) {
	t.Parallel()
	Convey("BatchUpdateKVMs", t, func() {
		ctx := gaetesting.TestingContextWithAppID("go-test")
		datastore.GetTestable(ctx).Consistent(true)
		kvms := make([]*proto.KVM, 0, 4)
		for i := 0; i < 4; i++ {
			kvm1 := mockKVM(fmt.Sprintf("kvm-%d", i), "chromePlatform-1")
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
			KVM5 := mockKVM("", "chromeoslab", "samus")
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
			Name: "kvm-1",
		}
		kvm1 := mockKVM("kvm-1", "chromePlatform-1")
		resp, cerr := CreateKVM(ctx, kvm1)
		So(cerr, ShouldBeNil)
		So(resp, ShouldResembleProto, kvm1)

		kvms := make([]*proto.KVM, 0, 1)
		kvms = append(kvms, dummyKVM)

		kvms1 := make([]*proto.KVM, 0, 1)
		kvms1 = append(kvms1, kvm1)
		Convey("Query By existing ChromePlatform keysonly", func() {
			resp, err := QueryKVMByPropertyName(ctx, "chrome_platform_id", "chromePlatform-1", true)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, kvms)
		})
		Convey("Query By non-existing ChromePlatform", func() {
			resp, err := QueryKVMByPropertyName(ctx, "chrome_platform_id", "chromePlatform-2", true)
			So(err, ShouldBeNil)
			So(resp, ShouldBeNil)
		})
		Convey("Query By existing ChromePlatform", func() {
			resp, err := QueryKVMByPropertyName(ctx, "chrome_platform_id", "chromePlatform-1", false)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, kvms1)
		})
	})
}*/
