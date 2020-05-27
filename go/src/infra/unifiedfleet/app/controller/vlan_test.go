// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/appengine/gaetesting"
	. "go.chromium.org/luci/common/testing/assertions"
	proto "infra/unifiedfleet/api/v1/proto"
	"infra/unifiedfleet/app/model/configuration"
	. "infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/model/inventory"
)

func mockVlan(id string) *proto.Vlan {
	return &proto.Vlan{
		Name: id,
	}
}

func TestDeleteVlan(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	datastore.GetTestable(ctx).Consistent(true)
	vlan1 := mockVlan("vlan-1")
	vlan2 := mockVlan("vlan-2")
	Convey("DeleteVlan", t, func() {
		Convey("Delete vlan by existing ID with machinelse reference", func() {
			resp, cerr := configuration.CreateVlan(ctx, vlan1)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, vlan1)
			chromeosServerLse := &proto.ChromeOSServerLSE{
				SupportedRestrictedVlan: "vlan-1",
			}
			chromeoslse := &proto.ChromeOSMachineLSE_Server{
				Server: chromeosServerLse,
			}
			chromeosMachineLse := &proto.ChromeOSMachineLSE{
				ChromeosLse: chromeoslse,
			}
			lse := &proto.MachineLSE_ChromeosMachineLse{
				ChromeosMachineLse: chromeosMachineLse,
			}
			machineLSE1 := &proto.MachineLSE{
				Name: "machineLSE-1",
				Lse:  lse,
			}
			mresp, merr := inventory.CreateMachineLSE(ctx, machineLSE1)
			So(merr, ShouldBeNil)
			So(mresp, ShouldResembleProto, machineLSE1)

			err := DeleteVlan(ctx, "vlan-1")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, CannotDelete)

			resp, cerr = configuration.GetVlan(ctx, "vlan-1")
			So(resp, ShouldNotBeNil)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, vlan1)
		})
		Convey("Delete vlan successfully by existing ID without references", func() {
			resp, cerr := configuration.CreateVlan(ctx, vlan2)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, vlan2)

			err := DeleteVlan(ctx, "vlan-2")
			So(err, ShouldBeNil)

			resp, cerr = configuration.GetVlan(ctx, "vlan-2")
			So(resp, ShouldBeNil)
			So(cerr, ShouldNotBeNil)
			So(cerr.Error(), ShouldContainSubstring, NotFound)
		})
	})
}
