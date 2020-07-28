// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/appengine/gaetesting"
	. "go.chromium.org/luci/common/testing/assertions"

	ufspb "infra/unifiedfleet/api/v1/proto"
	"infra/unifiedfleet/app/model/configuration"
	. "infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/model/inventory"
)

func mockVlan(id string) *ufspb.Vlan {
	return &ufspb.Vlan{
		Name: id,
	}
}

func TestListVlans(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	vlans := make([]*ufspb.Vlan, 0, 4)
	for i := 0; i < 4; i++ {
		vlan1 := mockVlan("")
		vlan1.Name = fmt.Sprintf("vlan-%d", i)
		resp, _ := configuration.CreateVlan(ctx, vlan1)
		vlans = append(vlans, resp)
	}
	Convey("ListVlans", t, func() {
		Convey("List Vlans - filter invalid", func() {
			_, _, err := ListVlans(ctx, 5, "", "machine=mx-1", false)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Failed to read filter for listing vlans")
		})

		Convey("ListVlans - Full listing - happy path", func() {
			resp, _, _ := ListVlans(ctx, 5, "", "", false)
			So(resp, ShouldNotBeNil)
			So(resp, ShouldResembleProto, vlans)
		})
	})
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
			chromeosServerLse := &ufspb.ChromeOSServerLSE{
				SupportedRestrictedVlan: "vlan-1",
			}
			chromeoslse := &ufspb.ChromeOSMachineLSE_ServerLse{
				ServerLse: chromeosServerLse,
			}
			chromeosMachineLse := &ufspb.ChromeOSMachineLSE{
				ChromeosLse: chromeoslse,
			}
			lse := &ufspb.MachineLSE_ChromeosMachineLse{
				ChromeosMachineLse: chromeosMachineLse,
			}
			machineLSE1 := &ufspb.MachineLSE{
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
