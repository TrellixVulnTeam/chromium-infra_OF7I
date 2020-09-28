// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"fmt"
	"testing"

	"github.com/golang/protobuf/proto"
	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/luci/appengine/gaetesting"
	. "go.chromium.org/luci/common/testing/assertions"
	"go.chromium.org/luci/gae/service/datastore"
	"google.golang.org/genproto/protobuf/field_mask"

	ufspb "infra/unifiedfleet/api/v1/proto"
	"infra/unifiedfleet/app/model/configuration"
	. "infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/model/history"
	"infra/unifiedfleet/app/model/inventory"
	"infra/unifiedfleet/app/model/state"
	"infra/unifiedfleet/app/util"
)

func mockVlan(id, cidr string) *ufspb.Vlan {
	return &ufspb.Vlan{
		Name:          id,
		VlanAddress:   cidr,
		ResourceState: ufspb.State_STATE_UNSPECIFIED,
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
			vlan1 := mockVlan("create-vlan-1", "192.168.100.0/27")
			vlan1.Zones = []ufspb.Zone{ufspb.Zone_ZONE_MTV97, ufspb.Zone_ZONE_MTV96}
			resp, err := CreateVlan(ctx, vlan1)
			So(err, ShouldBeNil)
			So(resp.GetName(), ShouldEqual, "create-vlan-1")
			So(resp.GetVlanAddress(), ShouldEqual, "192.168.100.0/27")
			So(resp.GetCapacityIp(), ShouldEqual, 32)
			So(resp.GetResourceState(), ShouldEqual, ufspb.State_STATE_SERVING)
			So(resp.GetFreeStartIpv4Str(), ShouldEqual, "192.168.100.11")
			So(resp.GetFreeEndIpv4Str(), ShouldEqual, "192.168.100.30")
			So(resp.GetZones(), ShouldHaveLength, 2)

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
	configuration.BatchUpdateVlans(ctx, []*ufspb.Vlan{mockVlan("update-vlan-0", "4.4.4.4/27")})
	Convey("UpdateVlan", t, func() {
		Convey("Update vlan - with existing vlan", func() {
			resp, err := UpdateVlan(ctx, mockVlan("update-vlan-non-exist", ""), nil)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "no Vlan with VlanID update-vlan-non-exist")
		})

		Convey("Update vlan - partial update invalid fields", func() {
			resp, err := UpdateVlan(ctx, mockVlan("update-vlan-0", "2.2.2.2/22"), &field_mask.FieldMask{Paths: []string{"cidr_block"}})
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "cidr_block cannot be updated")
		})

		Convey("Update vlan - fully update cidr_block won't work", func() {
			resp, err := UpdateVlan(ctx, mockVlan("update-vlan-0", "2.2.2.2/22"), nil)
			So(err, ShouldBeNil)
			So(resp.GetVlanAddress(), ShouldEqual, "4.4.4.4/27")
		})

		Convey("Update vlan - fully update description & state", func() {
			vlan2 := mockVlan("update-vlan-0", "4.4.4.4/30")
			vlan2.ResourceState = ufspb.State_STATE_SERVING
			vlan2.Description = "test fully update"
			resp, err := UpdateVlan(ctx, vlan2, nil)
			So(err, ShouldBeNil)
			So(resp.GetDescription(), ShouldEqual, "test fully update")
			So(resp.GetVlanAddress(), ShouldEqual, "4.4.4.4/27")
			So(resp.GetResourceState(), ShouldEqual, ufspb.State_STATE_SERVING)

			vlan, err := configuration.GetVlan(ctx, "update-vlan-0")
			So(err, ShouldBeNil)
			So(vlan.GetDescription(), ShouldEqual, "test fully update")
			So(vlan.GetVlanAddress(), ShouldEqual, "4.4.4.4/27")
			So(vlan.GetResourceState(), ShouldEqual, ufspb.State_STATE_SERVING)
			s, err := state.GetStateRecord(ctx, "vlans/update-vlan-0")
			So(err, ShouldBeNil)
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_SERVING)
		})

		Convey("Update vlan - partial update description", func() {
			configuration.BatchUpdateVlans(ctx, []*ufspb.Vlan{mockVlan("update-vlan-1", "5.5.5.5/27")})
			vlan2 := mockVlan("update-vlan-1", "2.2.2.2/27")
			vlan2.Description = "test partial update"
			resp, err := UpdateVlan(ctx, vlan2, &field_mask.FieldMask{Paths: []string{"description"}})
			So(err, ShouldBeNil)
			So(resp.GetDescription(), ShouldEqual, "test partial update")
			So(resp.GetVlanAddress(), ShouldEqual, "5.5.5.5/27")
			So(resp.GetResourceState(), ShouldEqual, ufspb.State_STATE_UNSPECIFIED)

			vlan, err := configuration.GetVlan(ctx, "update-vlan-1")
			So(err, ShouldBeNil)
			So(vlan.GetDescription(), ShouldEqual, "test partial update")
			So(vlan.GetVlanAddress(), ShouldEqual, "5.5.5.5/27")
			So(vlan.GetResourceState(), ShouldEqual, ufspb.State_STATE_UNSPECIFIED)
		})

		Convey("Update vlan - partial update state", func() {
			vlan1 := mockVlan("update-vlan-2", "5.5.5.5/27")
			vlan1.Description = "before update"
			configuration.BatchUpdateVlans(ctx, []*ufspb.Vlan{vlan1})
			vlan2 := mockVlan("update-vlan-2", "2.2.2.2/27")
			vlan2.Description = "after update"
			vlan2.ResourceState = ufspb.State_STATE_SERVING
			resp, err := UpdateVlan(ctx, vlan2, &field_mask.FieldMask{Paths: []string{"resourceState"}})
			So(err, ShouldBeNil)
			So(resp.GetDescription(), ShouldEqual, "before update")
			So(resp.GetVlanAddress(), ShouldEqual, "5.5.5.5/27")
			So(resp.GetResourceState(), ShouldEqual, ufspb.State_STATE_SERVING)

			vlan, err := configuration.GetVlan(ctx, "update-vlan-2")
			So(err, ShouldBeNil)
			So(vlan.GetDescription(), ShouldEqual, "before update")
			So(vlan.GetVlanAddress(), ShouldEqual, "5.5.5.5/27")
			So(vlan.GetResourceState(), ShouldEqual, ufspb.State_STATE_SERVING)

			s, err := state.GetStateRecord(ctx, "vlans/update-vlan-2")
			So(err, ShouldBeNil)
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_SERVING)
		})

		Convey("Update vlan - partial update reserved_ips - invalid ips", func() {
			vlan1 := mockVlan("update-vlan-3", "6.6.6.0/27")
			vlan1.Description = "before update"
			configuration.BatchUpdateVlans(ctx, []*ufspb.Vlan{vlan1})
			vlan2 := proto.Clone(vlan1).(*ufspb.Vlan)
			vlan2.ReservedIps = []string{"6.6.6.48"}
			_, err := UpdateVlan(ctx, vlan2, &field_mask.FieldMask{Paths: []string{"reserved_ips"}})
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "doesn't belong to vlan")
		})

		Convey("Update vlan - partial update reserved_ips - happy path", func() {
			vlan1 := mockVlan("update-vlan-4", "6.6.6.0/27")
			vlan1.Description = "before update"
			configuration.BatchUpdateVlans(ctx, []*ufspb.Vlan{vlan1})
			ips, _, _, _, err := util.ParseVlan("update-vlan-4", "6.6.6.0/27")
			So(err, ShouldBeNil)
			_, err = configuration.BatchUpdateIPs(ctx, ips)
			So(err, ShouldBeNil)

			vlan2 := proto.Clone(vlan1).(*ufspb.Vlan)
			vlan2.ReservedIps = []string{"6.6.6.14"}
			res, err := UpdateVlan(ctx, vlan2, &field_mask.FieldMask{Paths: []string{"reserved_ips"}})
			So(err, ShouldBeNil)
			So(res.GetReservedIps(), ShouldResemble, []string{"6.6.6.14"})
			resIPs, err := configuration.QueryIPByPropertyName(ctx, map[string]string{"ipv4_str": "6.6.6.14"})
			So(err, ShouldBeNil)
			So(resIPs, ShouldHaveLength, 0)
		})

		Convey("Update vlan - partial update reserved_ips - happy path with existing old reserved_ips", func() {
			vlan1 := mockVlan("update-vlan-5", "6.6.5.0/27")
			vlan1.ReservedIps = []string{"6.6.5.13"}
			configuration.BatchUpdateVlans(ctx, []*ufspb.Vlan{vlan1})
			ips, _, _, _, err := util.ParseVlan("update-vlan-5", "6.6.5.0/27")
			So(err, ShouldBeNil)
			_, err = configuration.BatchUpdateIPs(ctx, ips)
			So(err, ShouldBeNil)

			vlan2 := proto.Clone(vlan1).(*ufspb.Vlan)
			vlan2.ReservedIps = nil
			res, err := UpdateVlan(ctx, vlan2, &field_mask.FieldMask{Paths: []string{"reserved_ips"}})
			So(err, ShouldBeNil)
			So(res.GetReservedIps(), ShouldHaveLength, 0)
			resIPs, err := configuration.QueryIPByPropertyName(ctx, map[string]string{"ipv4_str": "6.6.5.13"})
			So(err, ShouldBeNil)
			So(resIPs, ShouldHaveLength, 1)
		})

		Convey("Update vlan - partial update zones - happy path", func() {
			vlan1 := mockVlan("update-vlan-6", "6.6.6.0/27")
			vlan1.Description = "before update"
			vlan1.Zones = []ufspb.Zone{ufspb.Zone_ZONE_ATL97}
			configuration.BatchUpdateVlans(ctx, []*ufspb.Vlan{vlan1})

			vlan2 := proto.Clone(vlan1).(*ufspb.Vlan)
			vlan2.Zones = []ufspb.Zone{ufspb.Zone_ZONE_ATL97, ufspb.Zone_ZONE_MTV96}
			res, err := UpdateVlan(ctx, vlan2, &field_mask.FieldMask{Paths: []string{"zones"}})
			So(err, ShouldBeNil)
			So(res.GetZones(), ShouldHaveLength, 2)
			So(res.GetZones(), ShouldContain, ufspb.Zone_ZONE_ATL97)
			So(res.GetZones(), ShouldContain, ufspb.Zone_ZONE_MTV96)

			vlan3 := mockVlan("update-vlan-6", "")
			res, err = UpdateVlan(ctx, vlan3, &field_mask.FieldMask{Paths: []string{"zones"}})
			So(err, ShouldBeNil)
			So(res.GetZones(), ShouldHaveLength, 0)
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
		vlan1.Zones = []ufspb.Zone{ufspb.Zone_ZONE_MTV96}
		if i == 0 {
			vlan1.Zones = append(vlan1.Zones, ufspb.Zone_ZONE_IAD97)
		}
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

		Convey("ListVlans - list by zones - happy path", func() {
			resp, _, err := ListVlans(ctx, 5, "", "zone = iad97", false)
			So(err, ShouldBeNil)
			So(resp, ShouldHaveLength, 1)
			So(resp[0].GetName(), ShouldEqual, "vlan-0")

			resp, _, err = ListVlans(ctx, 5, "", "zone = mtv96", false)
			So(err, ShouldBeNil)
			So(resp, ShouldHaveLength, 4)
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
			machineLSE1 := &ufspb.MachineLSE{
				Name: "machineLSE-1",
				Vlan: "vlan-1",
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

func TestBatchGetVlans(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("BatchGetVlans", t, func() {
		Convey("Batch get vlans - happy path", func() {
			entities := make([]*ufspb.Vlan, 4)
			for i := 0; i < 4; i++ {
				entities[i] = &ufspb.Vlan{
					Name: fmt.Sprintf("vlan-batchGet-%d", i),
				}
			}
			_, err := configuration.BatchUpdateVlans(ctx, entities)
			So(err, ShouldBeNil)
			resp, err := configuration.BatchGetVlans(ctx, []string{"vlan-batchGet-0", "vlan-batchGet-1", "vlan-batchGet-2", "vlan-batchGet-3"})
			So(err, ShouldBeNil)
			So(resp, ShouldHaveLength, 4)
			So(resp, ShouldResembleProto, entities)
		})
		Convey("Batch get vlans  - missing id", func() {
			resp, err := configuration.BatchGetVlans(ctx, []string{"vlan-batchGet-non-existing"})
			So(err, ShouldNotBeNil)
			So(resp, ShouldBeNil)
			So(err.Error(), ShouldContainSubstring, "vlan-batchGet-non-existing")
		})
		Convey("Batch get vlans  - empty input", func() {
			resp, err := configuration.BatchGetVlans(ctx, nil)
			So(err, ShouldBeNil)
			So(resp, ShouldHaveLength, 0)

			input := make([]string, 0)
			resp, err = configuration.BatchGetVlans(ctx, input)
			So(err, ShouldBeNil)
			So(resp, ShouldHaveLength, 0)
		})
	})
}
