// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"

	ufspb "infra/unifiedfleet/api/v1/proto"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	"infra/unifiedfleet/app/model/configuration"
	. "infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/model/inventory"
	"infra/unifiedfleet/app/model/state"
	"infra/unifiedfleet/app/util"
)

func TestCreateVM(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("CreateVM", t, func() {
		inventory.CreateMachineLSE(ctx, &ufspb.MachineLSE{
			Name: "create-host",
			Lab:  "fake_lab",
		})
		Convey("Create new VM", func() {
			vm1 := &ufspb.VM{
				Name: "vm-create-1",
			}
			resp, err := CreateVM(ctx, vm1, "create-host", nil)
			So(err, ShouldBeNil)
			So(resp.GetState(), ShouldEqual, "STATE_DEPLOYED_PRE_SERVING")
			So(resp.GetMachineLseId(), ShouldEqual, "create-host")
			So(resp.GetLab(), ShouldEqual, "fake_lab")
		})

		Convey("Create new VM with specifying vlan", func() {
			setupTestVlan(ctx)

			vm1 := &ufspb.VM{
				Name: "vm-create-2",
			}
			resp, err := CreateVM(ctx, vm1, "create-host", &ufsAPI.NetworkOption{
				Vlan: "vlan-1",
			})
			So(err, ShouldBeNil)
			So(resp.GetState(), ShouldEqual, "STATE_DEPLOYED_PRE_SERVING")
			So(resp.GetMachineLseId(), ShouldEqual, "create-host")
			dhcp, err := configuration.GetDHCPConfig(ctx, "vm-create-2")
			So(err, ShouldBeNil)
			ip, err := configuration.QueryIPByPropertyName(ctx, map[string]string{"ipv4_str": dhcp.GetIp()})
			So(err, ShouldBeNil)
			So(ip, ShouldHaveLength, 1)
			So(ip[0].GetOccupied(), ShouldBeTrue)
		})

		Convey("Create new VM with specifying ip", func() {
			setupTestVlan(ctx)

			vm1 := &ufspb.VM{
				Name: "vm-create-3",
			}
			resp, err := CreateVM(ctx, vm1, "create-host", &ufsAPI.NetworkOption{
				Ip: "192.168.40.9",
			})
			So(err, ShouldBeNil)
			So(resp.GetState(), ShouldEqual, "STATE_DEPLOYED_PRE_SERVING")
			So(resp.GetMachineLseId(), ShouldEqual, "create-host")
			dhcp, err := configuration.GetDHCPConfig(ctx, "vm-create-3")
			So(err, ShouldBeNil)
			So(dhcp.GetIp(), ShouldEqual, "192.168.40.9")
			ip, err := configuration.QueryIPByPropertyName(ctx, map[string]string{"ipv4_str": "192.168.40.9"})
			So(err, ShouldBeNil)
			So(ip, ShouldHaveLength, 1)
			So(ip[0].GetOccupied(), ShouldBeTrue)
		})
	})
}

func TestUpdateVM(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("UpdateVM", t, func() {
		inventory.CreateMachineLSE(ctx, &ufspb.MachineLSE{
			Name: "update-host",
			Lab:  "fake_lab",
		})
		Convey("Update non-existing VM", func() {
			vm1 := &ufspb.VM{
				Name: "vm-update-1",
			}
			resp, err := UpdateVM(ctx, vm1, "create-host", nil, ufspb.State_STATE_UNSPECIFIED)
			So(err, ShouldNotBeNil)
			So(resp, ShouldBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})

		Convey("Update VM - happy path with vlan", func() {
			setupTestVlan(ctx)

			vm1 := &ufspb.VM{
				Name: "vm-update-2",
			}
			_, err := CreateVM(ctx, vm1, "update-host", nil)
			resp, err := UpdateVM(ctx, vm1, "update-host", &ufsAPI.NetworkOption{
				Vlan: "vlan-1",
			}, ufspb.State_STATE_UNSPECIFIED)
			So(err, ShouldBeNil)
			So(resp.GetState(), ShouldEqual, "STATE_DEPLOYED_PRE_SERVING")
			So(resp.GetMachineLseId(), ShouldEqual, "update-host")
			s, err := state.GetStateRecord(ctx, "vms/vm-update-2")
			So(err, ShouldBeNil)
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_DEPLOYED_PRE_SERVING)
			dhcp, err := configuration.GetDHCPConfig(ctx, "vm-update-2")
			So(err, ShouldBeNil)
			ip, err := configuration.QueryIPByPropertyName(ctx, map[string]string{"ipv4_str": dhcp.GetIp()})
			So(err, ShouldBeNil)
			So(ip, ShouldHaveLength, 1)
			So(ip[0].GetOccupied(), ShouldBeTrue)
		})

		Convey("Update VM - happy path with ip deletion", func() {
			setupTestVlan(ctx)
			vm1 := &ufspb.VM{
				Name: "vm-update-3",
			}
			_, err := CreateVM(ctx, vm1, "update-host", nil)
			So(err, ShouldBeNil)

			_, err = UpdateVM(ctx, vm1, "update-host", &ufsAPI.NetworkOption{
				Ip: "192.168.40.9",
			}, ufspb.State_STATE_UNSPECIFIED)
			So(err, ShouldBeNil)

			resp, err := UpdateVM(ctx, vm1, "update-host", &ufsAPI.NetworkOption{
				Delete: true,
			}, ufspb.State_STATE_UNSPECIFIED)
			So(err, ShouldBeNil)
			So(resp.GetState(), ShouldEqual, "STATE_DEPLOYED_PRE_SERVING")
			So(resp.GetMachineLseId(), ShouldEqual, "update-host")
			_, err = configuration.GetDHCPConfig(ctx, "vm-update-3")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
			ip, err := configuration.QueryIPByPropertyName(ctx, map[string]string{"ipv4_str": "192.168.40.9"})
			So(err, ShouldBeNil)
			So(ip, ShouldHaveLength, 1)
			So(ip[0].GetOccupied(), ShouldBeFalse)
		})

		Convey("Update VM - happy path with state updating", func() {
			setupTestVlan(ctx)

			vm1 := &ufspb.VM{
				Name: "vm-update-4",
			}
			_, err := CreateVM(ctx, vm1, "update-host", nil)
			resp, err := UpdateVM(ctx, vm1, "update-host", nil, ufspb.State_STATE_NEEDS_REPAIR)
			So(err, ShouldBeNil)
			So(resp.GetState(), ShouldEqual, "STATE_NEEDS_REPAIR")
			So(resp.GetMachineLseId(), ShouldEqual, "update-host")
			s, err := state.GetStateRecord(ctx, "vms/vm-update-4")
			So(err, ShouldBeNil)
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_NEEDS_REPAIR)
		})
	})
}

func TestDeleteVM(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("DeleteVM", t, func() {
		inventory.CreateMachineLSE(ctx, &ufspb.MachineLSE{
			Name: "delete-host",
			Lab:  "fake_lab",
		})
		Convey("Delete non-existing VM", func() {
			err := DeleteVM(ctx, "vm-delete-1")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Delete VM - happy path", func() {
			setupTestVlan(ctx)
			vm1 := &ufspb.VM{
				Name: "vm-delete-1",
			}
			_, err := CreateVM(ctx, vm1, "delete-host", &ufsAPI.NetworkOption{
				Ip: "192.168.40.7",
			})
			So(err, ShouldBeNil)

			// Before
			s, err := state.GetStateRecord(ctx, "vms/vm-delete-1")
			So(err, ShouldBeNil)
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_DEPLOYED_PRE_SERVING)
			dhcp, err := configuration.GetDHCPConfig(ctx, "vm-delete-1")
			So(err, ShouldBeNil)
			So(dhcp.GetIp(), ShouldEqual, "192.168.40.7")
			ip, err := configuration.QueryIPByPropertyName(ctx, map[string]string{"ipv4_str": "192.168.40.7"})
			So(err, ShouldBeNil)
			So(ip, ShouldHaveLength, 1)
			So(ip[0].GetOccupied(), ShouldBeTrue)

			// After
			err = DeleteVM(ctx, "vm-delete-1")
			So(err, ShouldBeNil)
			_, err = state.GetStateRecord(ctx, "vms/vm-delete-1")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
			_, err = configuration.GetDHCPConfig(ctx, "vm-delete-1")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
			ip, err = configuration.QueryIPByPropertyName(ctx, map[string]string{"ipv4_str": "192.168.40.7"})
			So(err, ShouldBeNil)
			So(ip, ShouldHaveLength, 1)
			So(ip[0].GetOccupied(), ShouldBeFalse)
		})
	})
}

func TestListVMs(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	vms := []*ufspb.VM{
		{
			Name: "vm-list-1",
			OsVersion: &ufspb.OSVersion{
				Value: "os-1",
			},
			Vlan:  "vlan-1",
			State: ufspb.State_STATE_SERVING.String(),
		},
		{
			Name: "vm-list-2",
			OsVersion: &ufspb.OSVersion{
				Value: "os-1",
			},
			Vlan:  "vlan-2",
			State: ufspb.State_STATE_SERVING.String(),
		},
		{
			Name: "vm-list-3",
			OsVersion: &ufspb.OSVersion{
				Value: "os-2",
			},
			Vlan:  "vlan-1",
			State: ufspb.State_STATE_SERVING.String(),
		},
		{
			Name: "vm-list-4",
			OsVersion: &ufspb.OSVersion{
				Value: "os-2",
			},
			Lab:   "fake_lab",
			Vlan:  "vlan-2",
			State: ufspb.State_STATE_DEPLOYED_TESTING.String(),
		},
	}
	Convey("ListVMs", t, func() {
		_, err := inventory.BatchUpdateVMs(ctx, vms)
		So(err, ShouldBeNil)
		Convey("List VMs - filter invalid - error", func() {
			_, _, err := ListVMs(ctx, 5, "", "invalid=mx-1", false)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Invalid field name invalid")
		})

		Convey("List VMs - filter vlan - happy path with filter", func() {
			resp, _, _ := ListVMs(ctx, 5, "", "vlan=vlan-1", false)
			So(resp, ShouldNotBeNil)
			So(resp, ShouldHaveLength, 2)
			So(ufsAPI.ParseResources(resp, "Name"), ShouldResemble, []string{"vm-list-1", "vm-list-3"})
		})

		Convey("List VMs - Full listing - happy path", func() {
			resp, _, _ := ListVMs(ctx, 5, "", "", false)
			So(resp, ShouldNotBeNil)
			So(resp, ShouldResembleProto, vms)
		})
		Convey("List VMs - multiple filters", func() {
			resp, _, err := ListVMs(ctx, 5, "", "vlan=vlan-2 & state=testing & lab=fake_lab", false)
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp, ShouldHaveLength, 1)
			So(resp[0].GetName(), ShouldEqual, "vm-list-4")
		})
	})
}

func setupTestVlan(ctx context.Context) {
	vlan := &ufspb.Vlan{
		Name:        "vlan-1",
		VlanAddress: "192.168.40.0/22",
	}
	configuration.CreateVlan(ctx, vlan)
	ips, _, _ := util.ParseVlan(vlan.GetName(), vlan.GetVlanAddress())
	// Only import the first 20 as one single transaction cannot import all.
	configuration.ImportIPs(ctx, ips[0:20])
}
