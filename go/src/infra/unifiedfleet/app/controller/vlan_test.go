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
	"google.golang.org/genproto/protobuf/field_mask"

	ufspb "infra/unifiedfleet/api/v1/proto"
	"infra/unifiedfleet/app/model/configuration"
	. "infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/model/history"
	"infra/unifiedfleet/app/model/inventory"
	"infra/unifiedfleet/app/model/state"
)

func mockVlan(id, cidr string) *ufspb.Vlan {
	return &ufspb.Vlan{
		Name:        id,
		VlanAddress: cidr,
		State:       ufspb.State_STATE_UNSPECIFIED.String(),
	}
}

func TestCreateVlan(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	configuration.BatchUpdateVlans(ctx, []*ufspb.Vlan{mockVlan("create-vlan-0", "2.2.2.2/22")})
	Convey("CreateVlan", t, func() {
		Convey("Create vlan - with existing vlan", func() {
			resp, err := CreateVlan(ctx, mockVlan("create-vlan-0", ""))
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "already exists in the system")
		})

		Convey("Create vlan - with duplicated cidr block", func() {
			resp, err := CreateVlan(ctx, mockVlan("create-vlan-1", "2.2.2.2/22"))
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "cidr block 2.2.2.2/22 is already occupied by ")
		})

		Convey("Create vlan - invalid cidr block", func() {
			resp, err := CreateVlan(ctx, mockVlan("create-vlan-1", "a.b.c.d/ab"))
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "invalid CIDR block")
		})

		Convey("Create vlan - happy path", func() {
			resp, err := CreateVlan(ctx, mockVlan("create-vlan-1", "192.168.100.0/30"))
			So(err, ShouldBeNil)
			So(resp.GetName(), ShouldEqual, "create-vlan-1")
			So(resp.GetVlanAddress(), ShouldEqual, "192.168.100.0/30")
			So(resp.GetCapacityIp(), ShouldEqual, 4)
			So(resp.GetState(), ShouldEqual, ufspb.State_STATE_SERVING.String())

			s, err := state.GetStateRecord(ctx, "vlans/create-vlan-1")
			So(err, ShouldBeNil)
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_SERVING)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "vlans/create-vlan-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetEventLabel(), ShouldEqual, "vlan")
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "vlans/create-vlan-1")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			So(msgs[0].Delete, ShouldBeFalse)
		})
	})
}

func TestUpdateVlan(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	configuration.BatchUpdateVlans(ctx, []*ufspb.Vlan{mockVlan("update-vlan-0", "4.4.4.4/30")})
	Convey("UpdateVlan", t, func() {
		Convey("Update vlan - with existing vlan", func() {
			resp, err := UpdateVlan(ctx, mockVlan("update-vlan-non-exist", ""), nil, ufspb.State_STATE_UNSPECIFIED)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "no Vlan with VlanID update-vlan-non-exist")
		})

		Convey("Update vlan - partial update invalid fields", func() {
			resp, err := UpdateVlan(ctx, mockVlan("update-vlan-0", "2.2.2.2/22"), &field_mask.FieldMask{Paths: []string{"cidr_block"}}, ufspb.State_STATE_UNSPECIFIED)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "cidr_block cannot be updated")
		})

		Convey("Update vlan - fully update cidr_block won't work", func() {
			resp, err := UpdateVlan(ctx, mockVlan("update-vlan-0", "2.2.2.2/22"), nil, ufspb.State_STATE_UNSPECIFIED)
			So(err, ShouldBeNil)
			So(resp.GetVlanAddress(), ShouldEqual, "4.4.4.4/30")
		})

		Convey("Update vlan - fully update description & state", func() {
			vlan2 := mockVlan("update-vlan-0", "4.4.4.4/30")
			vlan2.Description = "test fully update"
			resp, err := UpdateVlan(ctx, vlan2, nil, ufspb.State_STATE_SERVING)
			So(err, ShouldBeNil)
			So(resp.GetDescription(), ShouldEqual, "test fully update")
			So(resp.GetVlanAddress(), ShouldEqual, "4.4.4.4/30")
			So(resp.GetState(), ShouldEqual, ufspb.State_STATE_SERVING.String())

			vlan, err := configuration.GetVlan(ctx, "update-vlan-0")
			So(err, ShouldBeNil)
			So(vlan.GetDescription(), ShouldEqual, "test fully update")
			So(vlan.GetVlanAddress(), ShouldEqual, "4.4.4.4/30")
			So(vlan.GetState(), ShouldEqual, ufspb.State_STATE_SERVING.String())
			s, err := state.GetStateRecord(ctx, "vlans/update-vlan-0")
			So(err, ShouldBeNil)
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_SERVING)
		})

		Convey("Update vlan - partial update description", func() {
			configuration.BatchUpdateVlans(ctx, []*ufspb.Vlan{mockVlan("update-vlan-1", "5.5.5.5/30")})
			vlan2 := mockVlan("update-vlan-1", "2.2.2.2/30")
			vlan2.Description = "test partial update"
			resp, err := UpdateVlan(ctx, vlan2, &field_mask.FieldMask{Paths: []string{"description"}}, ufspb.State_STATE_UNSPECIFIED)
			So(err, ShouldBeNil)
			So(resp.GetDescription(), ShouldEqual, "test partial update")
			So(resp.GetVlanAddress(), ShouldEqual, "5.5.5.5/30")
			So(resp.GetState(), ShouldEqual, ufspb.State_STATE_UNSPECIFIED.String())

			vlan, err := configuration.GetVlan(ctx, "update-vlan-1")
			So(err, ShouldBeNil)
			So(vlan.GetDescription(), ShouldEqual, "test partial update")
			So(vlan.GetVlanAddress(), ShouldEqual, "5.5.5.5/30")
			So(vlan.GetState(), ShouldEqual, ufspb.State_STATE_UNSPECIFIED.String())
		})

		Convey("Update vlan - partial update state", func() {
			vlan1 := mockVlan("update-vlan-2", "5.5.5.5/30")
			vlan1.Description = "before update"
			configuration.BatchUpdateVlans(ctx, []*ufspb.Vlan{vlan1})
			vlan2 := mockVlan("update-vlan-2", "2.2.2.2/30")
			vlan2.Description = "after update"
			resp, err := UpdateVlan(ctx, vlan2, &field_mask.FieldMask{Paths: []string{"state"}}, ufspb.State_STATE_SERVING)
			So(err, ShouldBeNil)
			So(resp.GetDescription(), ShouldEqual, "before update")
			So(resp.GetVlanAddress(), ShouldEqual, "5.5.5.5/30")
			So(resp.GetState(), ShouldEqual, ufspb.State_STATE_SERVING.String())

			vlan, err := configuration.GetVlan(ctx, "update-vlan-2")
			So(err, ShouldBeNil)
			So(vlan.GetDescription(), ShouldEqual, "before update")
			So(vlan.GetVlanAddress(), ShouldEqual, "5.5.5.5/30")
			So(vlan.GetState(), ShouldEqual, ufspb.State_STATE_SERVING.String())

			s, err := state.GetStateRecord(ctx, "vlans/update-vlan-2")
			So(err, ShouldBeNil)
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_SERVING)
		})
	})
}

func TestListVlans(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	vlans := make([]*ufspb.Vlan, 0, 4)
	for i := 0; i < 4; i++ {
		vlan1 := mockVlan("", "")
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
	vlan1 := mockVlan("vlan-1", "")
	vlan2 := mockVlan("vlan-2", "")
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
			So(err.Error(), ShouldContainSubstring, "vlan vlan-1 is occupied by")

			resp, cerr = configuration.GetVlan(ctx, "vlan-1")
			So(resp, ShouldNotBeNil)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, vlan1)
		})
		Convey("Delete vlan successfully with large numbers of IPs", func() {
			vlan2.VlanAddress = "192.168.16.0/24"
			resp, err := CreateVlan(ctx, vlan2)
			So(err, ShouldBeNil)

			err = DeleteVlan(ctx, "vlan-2")
			So(err, ShouldBeNil)

			resp, err = configuration.GetVlan(ctx, "vlan-2")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
			respIps, err := configuration.QueryIPByPropertyName(ctx, map[string]string{"ipv4_str": "192.168.16.1"})
			So(err, ShouldBeNil)
			So(respIps, ShouldBeNil)
		})
	})
}
