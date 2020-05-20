// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"testing"

	proto "infra/unifiedfleet/api/v1/proto"
	"infra/unifiedfleet/app/model/configuration"
	. "infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/model/inventory"
	"infra/unifiedfleet/app/model/registration"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func mockKVM(id string) *proto.KVM {
	return &proto.KVM{
		Name: id,
	}
}

func TestCreateKVM(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("CreateKVMs", t, func() {
		Convey("Create new kvm with non existing chromePlatform", func() {
			kvm1 := &proto.KVM{
				Name:           "kvm-1",
				ChromePlatform: "chromePlatform-1",
			}
			resp, err := CreateKVM(ctx, kvm1)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, CannotCreate)
		})

		Convey("Create new kvm with existing resources", func() {
			chromePlatform2 := &proto.ChromePlatform{
				Name: "chromePlatform-2",
			}
			presp, err := configuration.CreateChromePlatform(ctx, chromePlatform2)
			So(err, ShouldBeNil)
			So(presp, ShouldResembleProto, chromePlatform2)

			kvm2 := &proto.KVM{
				Name:           "kvm-2",
				ChromePlatform: "chromePlatform-2",
			}
			resp, err := CreateKVM(ctx, kvm2)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, kvm2)
		})
	})
}

func TestDeleteKVM(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
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
			mresp, merr := registration.CreateMachine(ctx, chromeBrowserMachine1)
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
			mresp, merr := registration.CreateRack(ctx, chromeBrowserRack1)
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
	})
}
