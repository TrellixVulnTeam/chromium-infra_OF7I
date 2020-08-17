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
				Name: "vm-create-1",
			}
			resp, err := CreateVM(ctx, vm1, "create-host", nil)
			So(err, ShouldBeNil)
			So(resp.GetState(), ShouldEqual, "STATE_DEPLOYED_PRE_SERVING")
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
			So(changes[0].GetNewValue(), ShouldEqual, ufspb.State_STATE_DEPLOYED_PRE_SERVING.String())
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
			So(changes[0].GetNewValue(), ShouldEqual, ufspb.State_STATE_DEPLOYED_PRE_SERVING.String())
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
			So(changes[0].GetNewValue(), ShouldEqual, ufspb.State_STATE_DEPLOYED_PRE_SERVING.String())
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "dhcps/vm-create-3")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetEventLabel(), ShouldEqual, "dhcp_config.ip")
			So(changes[0].GetOldValue(), ShouldEqual, "")
			So(changes[0].GetNewValue(), ShouldEqual, "192.168.40.9")
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
				Name: "vm-update-1",
			}
			resp, err := UpdateVM(ctx, vm1, "create-host", ufspb.State_STATE_UNSPECIFIED, nil)
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
				Name: "vm-update-2",
			}
			_, err := CreateVM(ctx, vm1, "update-host", nil)
			resp, err := UpdateVMHost(ctx, vm1.Name, &ufsAPI.NetworkOption{
				Vlan: "vlan-1",
			})
			So(err, ShouldBeNil)
			So(resp.GetState(), ShouldEqual, "STATE_DEPLOYED_PRE_SERVING")
			s, err := state.GetStateRecord(ctx, "vms/vm-update-2")
			So(err, ShouldBeNil)
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_DEPLOYED_PRE_SERVING)
			dhcp, err := configuration.GetDHCPConfig(ctx, "vm-update-2")
			So(err, ShouldBeNil)
			ips, err := configuration.QueryIPByPropertyName(ctx, map[string]string{"ipv4_str": dhcp.GetIp()})
			So(err, ShouldBeNil)
			So(ips, ShouldHaveLength, 1)
			So(ips[0].GetOccupied(), ShouldBeTrue)

			// Come from CreateVM+UpdateVMHost
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "vms/vm-update-2")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 2)
			So(changes[0].GetEventLabel(), ShouldEqual, "vm")
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRegistration)
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "states/vms/vm-update-2")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetEventLabel(), ShouldEqual, "state_record.state")
			So(changes[0].GetOldValue(), ShouldEqual, ufspb.State_STATE_UNSPECIFIED.String())
			So(changes[0].GetNewValue(), ShouldEqual, ufspb.State_STATE_DEPLOYED_PRE_SERVING.String())
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
			So(msgs, ShouldHaveLength, 1)
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "dhcps/vm-update-2")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
		})

		Convey("Update VM - happy path with ip deletion", func() {
			setupTestVlan(ctx)
			vm1 := &ufspb.VM{
				Name: "vm-update-3",
			}
			_, err := CreateVM(ctx, vm1, "update-host", nil)
			So(err, ShouldBeNil)

			_, err = UpdateVMHost(ctx, vm1.Name, &ufsAPI.NetworkOption{
				Ip: "192.168.40.9",
			})
			So(err, ShouldBeNil)

			err = DeleteVMHost(ctx, vm1.Name)
			So(err, ShouldBeNil)
			_, err = configuration.GetDHCPConfig(ctx, "vm-update-3")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
			ips, err := configuration.QueryIPByPropertyName(ctx, map[string]string{"ipv4_str": "192.168.40.9"})
			So(err, ShouldBeNil)
			So(ips, ShouldHaveLength, 1)
			So(ips[0].GetOccupied(), ShouldBeFalse)

			// Come from UpdateVM
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "dhcps/vm-update-3")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 2)
			So(changes[0].GetEventLabel(), ShouldEqual, "dhcp_config.ip")
			So(changes[0].GetOldValue(), ShouldEqual, "")
			So(changes[0].GetNewValue(), ShouldEqual, "192.168.40.9")
			So(changes[1].GetEventLabel(), ShouldEqual, "dhcp_config.ip")
			So(changes[1].GetOldValue(), ShouldEqual, "192.168.40.9")
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
			// 1 create
			So(msgs, ShouldHaveLength, 1)
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "dhcps/vm-update-3")
			So(err, ShouldBeNil)
			// 2 host update
			So(msgs, ShouldHaveLength, 2)
			So(msgs[1].Delete, ShouldBeTrue)
		})

		Convey("Update VM - happy path with state updating", func() {
			setupTestVlan(ctx)

			vm1 := &ufspb.VM{
				Name: "vm-update-4",
			}
			_, err := CreateVM(ctx, vm1, "update-host", nil)
			resp, err := UpdateVM(ctx, vm1, "update-host", ufspb.State_STATE_NEEDS_REPAIR, nil)
			So(err, ShouldBeNil)
			So(resp.GetState(), ShouldEqual, "STATE_NEEDS_REPAIR")
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
			So(changes[0].GetNewValue(), ShouldEqual, ufspb.State_STATE_DEPLOYED_PRE_SERVING.String())
			// Come from UpdateVM
			So(changes[1].GetEventLabel(), ShouldEqual, "state_record.state")
			So(changes[1].GetOldValue(), ShouldEqual, ufspb.State_STATE_DEPLOYED_PRE_SERVING.String())
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
				Tags: []string{"tag-1"},
			}
			_, err := CreateVM(ctx, vm, "update-host", nil)
			So(err, ShouldBeNil)

			vm1 := &ufspb.VM{
				Name: "vm-7",
				Tags: []string{"tag-2"},
			}
			resp, err := UpdateVM(ctx, vm1, "", ufspb.State_STATE_UNSPECIFIED, &field_mask.FieldMask{Paths: []string{"tags"}})
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
			ips, err := configuration.QueryIPByPropertyName(ctx, map[string]string{"ipv4_str": "192.168.40.7"})
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
			So(changes[1].GetOldValue(), ShouldEqual, ufspb.State_STATE_DEPLOYED_PRE_SERVING.String())
			So(changes[1].GetNewValue(), ShouldEqual, ufspb.State_STATE_UNSPECIFIED.String())
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "dhcps/vm-delete-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 2)
			So(changes[1].GetEventLabel(), ShouldEqual, "dhcp_config.ip")
			So(changes[1].GetOldValue(), ShouldEqual, "192.168.40.7")
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
			Zone:  ufspb.Zone_ZONE_ATLANTA.String(),
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
			resp, _, err := ListVMs(ctx, 5, "", "vlan=vlan-2 & state=deployed_testing & zone=atlanta", false)
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
