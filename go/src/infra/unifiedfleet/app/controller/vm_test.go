// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"context"
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
	"google.golang.org/genproto/protobuf/field_mask"

	ufspb "infra/unifiedfleet/api/v1/proto"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	"infra/unifiedfleet/app/model/configuration"
	. "infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/model/history"
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
			Zone: "fake_zone",
		})
		Convey("Create new VM", func() {
			vm1 := &ufspb.VM{
				Name:         "vm-create-1",
				MachineLseId: "create-host",
			}
			resp, err := CreateVM(ctx, vm1, nil)
			So(err, ShouldBeNil)
			So(resp.GetResourceState(), ShouldEqual, ufspb.State_STATE_REGISTERED)
			So(resp.GetMachineLseId(), ShouldEqual, "create-host")
			So(resp.GetZone(), ShouldEqual, "fake_zone")

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "vms/vm-create-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetEventLabel(), ShouldEqual, "vm")
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRegistration)
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "states/vms/vm-create-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetEventLabel(), ShouldEqual, "state_record.state")
			So(changes[0].GetOldValue(), ShouldEqual, ufspb.State_STATE_UNSPECIFIED.String())
			So(changes[0].GetNewValue(), ShouldEqual, ufspb.State_STATE_REGISTERED.String())
		})

		Convey("Create new VM with specifying vlan", func() {
			setupTestVlan(ctx)

			vm1 := &ufspb.VM{
				Name:         "vm-create-2",
				MachineLseId: "create-host",
			}
			resp, err := CreateVM(ctx, vm1, &ufsAPI.NetworkOption{
				Vlan: "vlan-1",
			})
			So(err, ShouldBeNil)
			So(resp.GetResourceState(), ShouldEqual, ufspb.State_STATE_DEPLOYING)
			So(resp.GetMachineLseId(), ShouldEqual, "create-host")
			dhcp, err := configuration.GetDHCPConfig(ctx, "vm-create-2")
			So(err, ShouldBeNil)
			ip, err := configuration.QueryIPByPropertyName(ctx, map[string]string{"ipv4_str": dhcp.GetIp()})
			So(err, ShouldBeNil)
			So(ip, ShouldHaveLength, 1)
			So(ip[0].GetOccupied(), ShouldBeTrue)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "vms/vm-create-2")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetEventLabel(), ShouldEqual, "vm")
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRegistration)
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "states/vms/vm-create-2")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetEventLabel(), ShouldEqual, "state_record.state")
			So(changes[0].GetOldValue(), ShouldEqual, ufspb.State_STATE_UNSPECIFIED.String())
			So(changes[0].GetNewValue(), ShouldEqual, ufspb.State_STATE_DEPLOYING.String())
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "dhcps/vm-create-2")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetEventLabel(), ShouldEqual, "dhcp_config.ip")
			So(changes[0].GetOldValue(), ShouldEqual, "")
			So(changes[0].GetNewValue(), ShouldEqual, dhcp.GetIp())
			changes, err = history.QueryChangesByPropertyName(ctx, "name", fmt.Sprintf("ips/%s", ip[0].GetId()))
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetEventLabel(), ShouldEqual, "ip.occupied")
			So(changes[0].GetOldValue(), ShouldEqual, "false")
			So(changes[0].GetNewValue(), ShouldEqual, "true")
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "vms/vm-create-2")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "states/vms/vm-create-2")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "dhcps/vm-create-2")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
		})

		Convey("Create new VM with specifying ip", func() {
			setupTestVlan(ctx)

			vm1 := &ufspb.VM{
				Name:         "vm-create-3",
				MachineLseId: "create-host",
			}
			resp, err := CreateVM(ctx, vm1, &ufsAPI.NetworkOption{
				Ip: "192.168.40.19",
			})
			So(err, ShouldBeNil)
			So(resp.GetResourceState(), ShouldEqual, ufspb.State_STATE_DEPLOYING)
			So(resp.GetMachineLseId(), ShouldEqual, "create-host")
			dhcp, err := configuration.GetDHCPConfig(ctx, "vm-create-3")
			So(err, ShouldBeNil)
			So(dhcp.GetIp(), ShouldEqual, "192.168.40.19")
			ip, err := configuration.QueryIPByPropertyName(ctx, map[string]string{"ipv4_str": "192.168.40.19"})
			So(err, ShouldBeNil)
			So(ip, ShouldHaveLength, 1)
			So(ip[0].GetOccupied(), ShouldBeTrue)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "vms/vm-create-3")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetEventLabel(), ShouldEqual, "vm")
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRegistration)
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "states/vms/vm-create-3")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetEventLabel(), ShouldEqual, "state_record.state")
			So(changes[0].GetOldValue(), ShouldEqual, ufspb.State_STATE_UNSPECIFIED.String())
			So(changes[0].GetNewValue(), ShouldEqual, ufspb.State_STATE_DEPLOYING.String())
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "dhcps/vm-create-3")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetEventLabel(), ShouldEqual, "dhcp_config.ip")
			So(changes[0].GetOldValue(), ShouldEqual, "")
			So(changes[0].GetNewValue(), ShouldEqual, "192.168.40.19")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", fmt.Sprintf("ips/%s", ip[0].GetId()))
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetEventLabel(), ShouldEqual, "ip.occupied")
			So(changes[0].GetOldValue(), ShouldEqual, "false")
			So(changes[0].GetNewValue(), ShouldEqual, "true")
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "vms/vm-create-3")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "states/vms/vm-create-3")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "dhcps/vm-create-3")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
		})
	})
}

func TestUpdateVM(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("UpdateVM", t, func() {
		inventory.CreateMachineLSE(ctx, &ufspb.MachineLSE{
			Name: "update-host",
			Zone: "fake_zone",
		})
		Convey("Update non-existing VM", func() {
			vm1 := &ufspb.VM{
				Name:         "vm-update-1",
				MachineLseId: "create-host",
			}
			resp, err := UpdateVM(ctx, vm1, nil)
			So(err, ShouldNotBeNil)
			So(resp, ShouldBeNil)
			So(err.Error(), ShouldContainSubstring, "There is no ChromeVM with ChromeVMID vm-update-1 in the system")

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "vms/vm-update-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})

		Convey("Update VM - happy path with vlan", func() {
			setupTestVlan(ctx)

			vm1 := &ufspb.VM{
				Name:         "vm-update-2",
				MachineLseId: "update-host",
			}
			_, err := CreateVM(ctx, vm1, nil)
			resp, err := UpdateVMHost(ctx, vm1.Name, &ufsAPI.NetworkOption{
				Vlan: "vlan-1",
			})
			So(err, ShouldBeNil)
			So(resp.GetResourceState(), ShouldEqual, ufspb.State_STATE_DEPLOYING)
			s, err := state.GetStateRecord(ctx, "vms/vm-update-2")
			So(err, ShouldBeNil)
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_DEPLOYING)
			dhcp, err := configuration.GetDHCPConfig(ctx, "vm-update-2")
			So(err, ShouldBeNil)
			ips, err := configuration.QueryIPByPropertyName(ctx, map[string]string{"ipv4_str": dhcp.GetIp()})
			So(err, ShouldBeNil)
			So(ips, ShouldHaveLength, 1)
			So(ips[0].GetOccupied(), ShouldBeTrue)

			// Come from CreateVM+UpdateVMHost
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "vms/vm-update-2")
			So(err, ShouldBeNil)
			// VM created & vlan, ip changes
			So(changes, ShouldHaveLength, 4)
			So(changes[0].GetEventLabel(), ShouldEqual, "vm")
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[1].GetEventLabel(), ShouldEqual, "vm.vlan")
			So(changes[1].GetOldValue(), ShouldEqual, "")
			So(changes[1].GetNewValue(), ShouldEqual, "vlan-1")
			So(changes[2].GetEventLabel(), ShouldEqual, "vm.ip")
			So(changes[2].GetOldValue(), ShouldEqual, "")
			So(changes[2].GetNewValue(), ShouldEqual, "192.168.40.11")
			So(changes[3].GetEventLabel(), ShouldEqual, "vm.resource_state")
			So(changes[3].GetOldValue(), ShouldEqual, "STATE_REGISTERED")
			So(changes[3].GetNewValue(), ShouldEqual, "STATE_DEPLOYING")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "states/vms/vm-update-2")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 2)
			So(changes[0].GetEventLabel(), ShouldEqual, "state_record.state")
			So(changes[0].GetOldValue(), ShouldEqual, ufspb.State_STATE_UNSPECIFIED.String())
			So(changes[0].GetNewValue(), ShouldEqual, ufspb.State_STATE_REGISTERED.String())
			So(changes[1].GetEventLabel(), ShouldEqual, "state_record.state")
			So(changes[1].GetOldValue(), ShouldEqual, ufspb.State_STATE_REGISTERED.String())
			So(changes[1].GetNewValue(), ShouldEqual, ufspb.State_STATE_DEPLOYING.String())
			// Come from UpdateVM
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "dhcps/vm-update-2")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetEventLabel(), ShouldEqual, "dhcp_config.ip")
			So(changes[0].GetOldValue(), ShouldEqual, "")
			So(changes[0].GetNewValue(), ShouldEqual, dhcp.GetIp())
			changes, err = history.QueryChangesByPropertyName(ctx, "name", fmt.Sprintf("ips/%s", ips[0].GetId()))
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetEventLabel(), ShouldEqual, "ip.occupied")
			So(changes[0].GetOldValue(), ShouldEqual, "false")
			So(changes[0].GetNewValue(), ShouldEqual, "true")
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "vms/vm-update-2")
			So(err, ShouldBeNil)
			// 1 come from CreateVM
			So(msgs, ShouldHaveLength, 2)
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "states/vms/vm-update-2")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 2)
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "dhcps/vm-update-2")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
		})

		Convey("Update VM - happy path with ip specification & deletion", func() {
			setupTestVlan(ctx)
			vm1 := &ufspb.VM{
				Name:         "vm-update-3",
				MachineLseId: "update-host",
			}
			_, err := CreateVM(ctx, vm1, nil)
			So(err, ShouldBeNil)

			_, err = UpdateVMHost(ctx, vm1.Name, &ufsAPI.NetworkOption{
				Ip: "192.168.40.19",
			})
			So(err, ShouldBeNil)

			err = DeleteVMHost(ctx, vm1.Name)
			So(err, ShouldBeNil)
			_, err = configuration.GetDHCPConfig(ctx, "vm-update-3")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
			ips, err := configuration.QueryIPByPropertyName(ctx, map[string]string{"ipv4_str": "192.168.40.19"})
			So(err, ShouldBeNil)
			So(ips, ShouldHaveLength, 1)
			So(ips[0].GetOccupied(), ShouldBeFalse)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "vms/vm-update-3")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 7)
			So(changes[0].GetEventLabel(), ShouldEqual, "vm")
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRegistration)
			// vlan & ip info are changed
			So(changes[1].GetEventLabel(), ShouldEqual, "vm.vlan")
			So(changes[1].GetOldValue(), ShouldEqual, "")
			So(changes[1].GetNewValue(), ShouldEqual, "vlan-1")
			So(changes[2].GetEventLabel(), ShouldEqual, "vm.ip")
			So(changes[2].GetOldValue(), ShouldEqual, "")
			So(changes[2].GetNewValue(), ShouldEqual, "192.168.40.19")
			So(changes[3].GetEventLabel(), ShouldEqual, "vm.resource_state")
			So(changes[3].GetOldValue(), ShouldEqual, "STATE_REGISTERED")
			So(changes[3].GetNewValue(), ShouldEqual, "STATE_DEPLOYING")
			// From deleting vm's ip
			So(changes[4].GetEventLabel(), ShouldEqual, "vm.vlan")
			So(changes[4].GetOldValue(), ShouldEqual, "vlan-1")
			So(changes[4].GetNewValue(), ShouldEqual, "")
			So(changes[5].GetEventLabel(), ShouldEqual, "vm.ip")
			So(changes[5].GetOldValue(), ShouldEqual, "192.168.40.19")
			So(changes[5].GetNewValue(), ShouldEqual, "")
			So(changes[6].GetEventLabel(), ShouldEqual, "vm.resource_state")
			So(changes[6].GetOldValue(), ShouldEqual, "STATE_DEPLOYING")
			So(changes[6].GetNewValue(), ShouldEqual, "STATE_REGISTERED")
			// log dhcp changes
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "dhcps/vm-update-3")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 2)
			So(changes[0].GetEventLabel(), ShouldEqual, "dhcp_config.ip")
			So(changes[0].GetOldValue(), ShouldEqual, "")
			So(changes[0].GetNewValue(), ShouldEqual, "192.168.40.19")
			So(changes[1].GetEventLabel(), ShouldEqual, "dhcp_config.ip")
			So(changes[1].GetOldValue(), ShouldEqual, "192.168.40.19")
			So(changes[1].GetNewValue(), ShouldEqual, "")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", fmt.Sprintf("ips/%s", ips[0].GetId()))
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 2)
			So(changes[0].GetEventLabel(), ShouldEqual, "ip.occupied")
			So(changes[0].GetOldValue(), ShouldEqual, "false")
			So(changes[0].GetNewValue(), ShouldEqual, "true")
			So(changes[1].GetEventLabel(), ShouldEqual, "ip.occupied")
			So(changes[1].GetOldValue(), ShouldEqual, "true")
			So(changes[1].GetNewValue(), ShouldEqual, "false")
			// snapshots
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "vms/vm-update-3")
			So(err, ShouldBeNil)
			// 1 create, 1 UpdateVMHost, 1 DeleteVMHost
			So(msgs, ShouldHaveLength, 3)
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "states/vms/vm-update-3")
			So(err, ShouldBeNil)
			// 1 create, 1 UpdateVMHost, 1 DeleteVMHost
			So(msgs, ShouldHaveLength, 3)
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "dhcps/vm-update-3")
			So(err, ShouldBeNil)
			// 2 host update
			So(msgs, ShouldHaveLength, 2)
			So(msgs[1].Delete, ShouldBeTrue)
		})

		Convey("Update VM - happy path with state updating", func() {
			setupTestVlan(ctx)

			vm1 := &ufspb.VM{
				Name:         "vm-update-4",
				MachineLseId: "update-host",
			}
			_, err := CreateVM(ctx, vm1, nil)
			vm1.ResourceState = ufspb.State_STATE_NEEDS_REPAIR
			resp, err := UpdateVM(ctx, vm1, nil)
			So(err, ShouldBeNil)
			So(resp.GetResourceState(), ShouldEqual, ufspb.State_STATE_NEEDS_REPAIR)
			So(resp.GetMachineLseId(), ShouldEqual, "update-host")
			s, err := state.GetStateRecord(ctx, "vms/vm-update-4")
			So(err, ShouldBeNil)
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_NEEDS_REPAIR)

			// Come from CreateVM
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "states/vms/vm-update-4")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 2)
			So(changes[0].GetEventLabel(), ShouldEqual, "state_record.state")
			So(changes[0].GetOldValue(), ShouldEqual, ufspb.State_STATE_UNSPECIFIED.String())
			So(changes[0].GetNewValue(), ShouldEqual, ufspb.State_STATE_REGISTERED.String())
			// Come from UpdateVM
			So(changes[1].GetEventLabel(), ShouldEqual, "state_record.state")
			So(changes[1].GetOldValue(), ShouldEqual, ufspb.State_STATE_REGISTERED.String())
			So(changes[1].GetNewValue(), ShouldEqual, ufspb.State_STATE_NEEDS_REPAIR.String())
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "dhcps/vm-update-4")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
			// snapshots
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "vms/vm-update-4")
			So(err, ShouldBeNil)
			// 1 create, 1 update
			So(msgs, ShouldHaveLength, 2)
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "states/vms/vm-update-4")
			So(err, ShouldBeNil)
			// 1 create, 1 update
			So(msgs, ShouldHaveLength, 2)
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "dhcps/vm-update-4")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 0)
		})

		Convey("Partial Update vm", func() {
			vm := &ufspb.VM{
				Name: "vm-7",
				OsVersion: &ufspb.OSVersion{
					Value: "windows",
				},
				Tags:         []string{"tag-1"},
				MachineLseId: "update-host",
			}
			_, err := CreateVM(ctx, vm, nil)
			So(err, ShouldBeNil)

			vm1 := &ufspb.VM{
				Name: "vm-7",
				Tags: []string{"tag-2"},
			}
			resp, err := UpdateVM(ctx, vm1, &field_mask.FieldMask{Paths: []string{"tags"}})
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp.GetTags(), ShouldResemble, []string{"tag-1", "tag-2"})
			So(resp.GetOsVersion().GetValue(), ShouldEqual, "windows")
		})
	})
}

func TestDeleteVM(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("DeleteVM", t, func() {
		inventory.CreateMachineLSE(ctx, &ufspb.MachineLSE{
			Name: "delete-host",
			Zone: "fake_zone",
		})
		Convey("Delete non-existing VM", func() {
			err := DeleteVM(ctx, "vm-delete-1")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "vms/vm-delete-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})
		Convey("Delete VM - happy path", func() {
			setupTestVlan(ctx)
			vm1 := &ufspb.VM{
				Name:         "vm-delete-1",
				MachineLseId: "delete-host",
			}
			_, err := CreateVM(ctx, vm1, &ufsAPI.NetworkOption{
				Ip: "192.168.40.17",
			})
			So(err, ShouldBeNil)

			// Before
			s, err := state.GetStateRecord(ctx, "vms/vm-delete-1")
			So(err, ShouldBeNil)
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_DEPLOYING)
			dhcp, err := configuration.GetDHCPConfig(ctx, "vm-delete-1")
			So(err, ShouldBeNil)
			So(dhcp.GetIp(), ShouldEqual, "192.168.40.17")
			ip, err := configuration.QueryIPByPropertyName(ctx, map[string]string{"ipv4_str": "192.168.40.17"})
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
			ips, err := configuration.QueryIPByPropertyName(ctx, map[string]string{"ipv4_str": "192.168.40.17"})
			So(err, ShouldBeNil)
			So(ips, ShouldHaveLength, 1)
			So(ips[0].GetOccupied(), ShouldBeFalse)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "vms/vm-delete-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 2)
			So(changes[1].GetOldValue(), ShouldEqual, LifeCycleRetire)
			So(changes[1].GetNewValue(), ShouldEqual, LifeCycleRetire)
			So(changes[1].GetEventLabel(), ShouldEqual, "vm")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "states/vms/vm-delete-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 2)
			So(changes[1].GetEventLabel(), ShouldEqual, "state_record.state")
			So(changes[1].GetOldValue(), ShouldEqual, ufspb.State_STATE_DEPLOYING.String())
			So(changes[1].GetNewValue(), ShouldEqual, ufspb.State_STATE_UNSPECIFIED.String())
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "dhcps/vm-delete-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 2)
			So(changes[1].GetEventLabel(), ShouldEqual, "dhcp_config.ip")
			So(changes[1].GetOldValue(), ShouldEqual, "192.168.40.17")
			So(changes[1].GetNewValue(), ShouldEqual, "")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", fmt.Sprintf("ips/%s", ips[0].GetId()))
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 2)
			So(changes[1].GetEventLabel(), ShouldEqual, "ip.occupied")
			So(changes[1].GetOldValue(), ShouldEqual, "true")
			So(changes[1].GetNewValue(), ShouldEqual, "false")
			// snapshots
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "vms/vm-delete-1")
			So(err, ShouldBeNil)
			// 1 create, 1 deletion
			So(msgs, ShouldHaveLength, 2)
			So(msgs[1].Delete, ShouldBeTrue)
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "states/vms/vm-delete-1")
			So(err, ShouldBeNil)
			// 1 create, 1 deletion
			So(msgs, ShouldHaveLength, 2)
			So(msgs[1].Delete, ShouldBeTrue)
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "dhcps/vm-delete-1")
			So(err, ShouldBeNil)
			// 1 create, 1 deletion
			So(msgs, ShouldHaveLength, 2)
			So(msgs[1].Delete, ShouldBeTrue)
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
			Vlan:          "vlan-1",
			ResourceState: ufspb.State_STATE_SERVING,
		},
		{
			Name: "vm-list-2",
			OsVersion: &ufspb.OSVersion{
				Value: "os-1",
			},
			Vlan:          "vlan-2",
			ResourceState: ufspb.State_STATE_SERVING,
		},
		{
			Name: "vm-list-3",
			OsVersion: &ufspb.OSVersion{
				Value: "os-2",
			},
			Vlan:          "vlan-1",
			ResourceState: ufspb.State_STATE_SERVING,
		},
		{
			Name: "vm-list-4",
			OsVersion: &ufspb.OSVersion{
				Value: "os-2",
			},
			Zone:          ufspb.Zone_ZONE_ATLANTA.String(),
			Vlan:          "vlan-2",
			ResourceState: ufspb.State_STATE_DEPLOYED_TESTING,
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
			resp, _, err := ListVMs(ctx, 5, "", "vlan=vlan-2 & state=deployed_testing & zone=atlanta", false)
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp, ShouldHaveLength, 1)
			So(resp[0].GetName(), ShouldEqual, "vm-list-4")
		})
	})
}
func TestBatchGetVMs(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("BatchGetVMs", t, func() {
		Convey("Batch get vms - happy path", func() {
			entities := make([]*ufspb.VM, 4)
			for i := 0; i < 4; i++ {
				entities[i] = &ufspb.VM{
					Name: fmt.Sprintf("vm-batchGet-%d", i),
				}
			}
			_, err := inventory.BatchUpdateVMs(ctx, entities)
			So(err, ShouldBeNil)
			resp, err := inventory.BatchGetVMs(ctx, []string{"vm-batchGet-0", "vm-batchGet-1", "vm-batchGet-2", "vm-batchGet-3"})
			So(err, ShouldBeNil)
			So(resp, ShouldHaveLength, 4)
			So(resp, ShouldResembleProto, entities)
		})
		Convey("Batch get vms  - missing id", func() {
			resp, err := inventory.BatchGetVMs(ctx, []string{"vm-batchGet-non-existing"})
			So(err, ShouldNotBeNil)
			So(resp, ShouldBeNil)
			So(err.Error(), ShouldContainSubstring, "vm-batchGet-non-existing")
		})
		Convey("Batch get vms  - empty input", func() {
			resp, err := inventory.BatchGetVMs(ctx, nil)
			So(err, ShouldBeNil)
			So(resp, ShouldHaveLength, 0)

			input := make([]string, 0)
			resp, err = inventory.BatchGetVMs(ctx, input)
			So(err, ShouldBeNil)
			So(resp, ShouldHaveLength, 0)
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
