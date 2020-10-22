// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
	"google.golang.org/genproto/protobuf/field_mask"

	ufspb "infra/unifiedfleet/api/v1/models"
	"infra/unifiedfleet/app/model/configuration"
	. "infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/model/history"
	"infra/unifiedfleet/app/model/inventory"
	"infra/unifiedfleet/app/model/registration"
	"infra/unifiedfleet/app/model/state"
)

func TestRackRegistration(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("RackRegistration", t, func() {
		Convey("Create new rack", func() {
			rack := &ufspb.Rack{
				Name: "rack-1",
				Rack: &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{},
				},
			}
			resp, err := RackRegistration(ctx, rack)
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp.GetChromeBrowserRack().GetKvmObjects(), ShouldBeNil)
			So(resp.GetChromeBrowserRack().GetRpmObjects(), ShouldBeNil)
			So(resp.GetChromeBrowserRack().GetSwitchObjects(), ShouldBeNil)
			s, err := state.GetStateRecord(ctx, "racks/rack-1")
			So(err, ShouldBeNil)
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_REGISTERED)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "racks/rack-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetEventLabel(), ShouldEqual, "rack")
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "racks/rack-1")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			So(msgs[0].Delete, ShouldBeFalse)
		})

		Convey("Create new rack with nil browser/chromeos rack", func() {
			rack := &ufspb.Rack{
				Name: "rack-2",
				Location: &ufspb.Location{
					Zone: ufspb.Zone_ZONE_ATLANTA,
				},
			}
			resp, err := RackRegistration(ctx, rack)
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp.GetChromeBrowserRack(), ShouldNotBeNil)

			resp, _ = registration.GetRack(ctx, "rack-2")
			So(resp.GetChromeBrowserRack(), ShouldNotBeNil)
			s, err := state.GetStateRecord(ctx, "racks/rack-1")
			So(err, ShouldBeNil)
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_REGISTERED)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "racks/rack-2")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetEventLabel(), ShouldEqual, "rack")
		})

		Convey("Create new rack with nil browser/chromeos rack and no lcoation info", func() {
			rack := &ufspb.Rack{
				Name: "rack-3",
			}
			_, err := RackRegistration(ctx, rack)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "zone information in the location object cannot be empty/unspecified for a rack")

			// No changes are recorded as the creation fails
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "racks/rack-3")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})
	})
}

func TestUpdateRack(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("UpdateRack", t, func() {
		Convey("Update non-existing rack - error", func() {
			rack := &ufspb.Rack{
				Name: "rack-1",
			}
			_, err := UpdateRack(ctx, rack, nil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "There is no Rack with RackID rack-1 in the system.")
		})

		Convey("Update existing rack", func() {
			rack := &ufspb.Rack{
				Name: "rack-2",
				Rack: &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{},
				},
			}
			_, err := registration.CreateRack(ctx, rack)
			So(err, ShouldBeNil)

			rack = &ufspb.Rack{
				Name: "rack-2",
				Tags: []string{"tag-1"},
				Rack: &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{},
				},
			}
			resp, err := UpdateRack(ctx, rack, nil)
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp, ShouldResembleProto, rack)
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "racks/rack-2")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			So(msgs[0].Delete, ShouldBeFalse)
		})

		Convey("Update existing rack with nil rack/browser machine", func() {
			rack := &ufspb.Rack{
				Name: "rack-3",
				Rack: &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{},
				},
			}
			_, err := registration.CreateRack(ctx, rack)
			So(err, ShouldBeNil)

			rack = &ufspb.Rack{
				Name: "rack-3",
			}
			resp, err := UpdateRack(ctx, rack, nil)
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
		})

		Convey("Partial Update rack", func() {
			rack := &ufspb.Rack{
				Name:       "rack-4",
				CapacityRu: 55,
				Tags:       []string{"atl", "megarack"},
			}
			_, err := registration.CreateRack(ctx, rack)
			So(err, ShouldBeNil)

			rack1 := &ufspb.Rack{
				Name:       "rack-4",
				CapacityRu: 100,
			}
			resp, err := UpdateRack(ctx, rack1, &field_mask.FieldMask{Paths: []string{"capacity"}})
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp.GetCapacityRu(), ShouldEqual, 100)
			So(resp.GetTags(), ShouldResemble, []string{"atl", "megarack"})
		})
	})
}

func TestDeleteRack(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("DeleteRack", t, func() {
		Convey("Delete rack by existing ID with rackLSE reference", func() {
			rack1 := &ufspb.Rack{
				Name: "rack-3",
			}
			_, err := registration.CreateRack(ctx, rack1)
			So(err, ShouldBeNil)

			rackLSE1 := &ufspb.RackLSE{
				Name:  "racklse-1",
				Racks: []string{"rack-3"},
			}
			_, err = inventory.CreateRackLSE(ctx, rackLSE1)
			So(err, ShouldBeNil)

			err = DeleteRack(ctx, "rack-3")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, CannotDelete)

			resp, err := registration.GetRack(ctx, "rack-3")
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, rack1)

			// No changes are recorded as the deletion fails
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "racks/rack-3")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})

		Convey("Delete rack by existing ID without references", func() {
			rack2 := &ufspb.Rack{
				Name: "rack-4",
			}
			_, err := registration.CreateRack(ctx, rack2)
			So(err, ShouldBeNil)

			err = DeleteRack(ctx, "rack-4")
			So(err, ShouldBeNil)

			resp, err := registration.GetRack(ctx, "rack-4")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "racks/rack-4")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRetire)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRetire)
			So(changes[0].GetEventLabel(), ShouldEqual, "rack")
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "racks/rack-4")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			So(msgs[0].Delete, ShouldBeTrue)
		})

		Convey("Delete rack with rpms - happy path", func() {
			rpm := &ufspb.RPM{
				Name: "rpm-6",
				Rack: "rack-6",
			}
			_, err := registration.CreateRPM(ctx, rpm)
			So(err, ShouldBeNil)

			rack := &ufspb.Rack{
				Name: "rack-6",
				Rack: &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{},
				},
			}
			_, err = registration.CreateRack(ctx, rack)
			So(err, ShouldBeNil)

			_, err = state.BatchUpdateStates(ctx, []*ufspb.StateRecord{
				{
					ResourceName: "rpms/rpm-6",
					State:        ufspb.State_STATE_SERVING,
				},
			})
			So(err, ShouldBeNil)
			_, err = configuration.BatchUpdateDHCPs(ctx, []*ufspb.DHCPConfig{
				{
					Hostname: "rpm-6",
					Ip:       "1.2.3.6",
				},
			})
			So(err, ShouldBeNil)
			_, err = configuration.BatchUpdateIPs(ctx, []*ufspb.IP{
				{
					Id:       "ip2",
					Occupied: true,
					Ipv4Str:  "1.2.3.6",
					Vlan:     "fake_vlan",
					Ipv4:     uint32(100),
				},
			})
			So(err, ShouldBeNil)

			err = DeleteRack(ctx, "rack-6")
			So(err, ShouldBeNil)

			_, err = registration.GetRack(ctx, "rack-6")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)

			_, err = registration.GetRPM(ctx, "rpm-6")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
			_, err = configuration.GetDHCPConfig(ctx, "kvm-6")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
			resIPs, err := configuration.QueryIPByPropertyName(ctx, map[string]string{"ipv4_str": "1.2.3.6"})
			So(err, ShouldBeNil)
			So(resIPs, ShouldHaveLength, 1)
			So(resIPs[0].Occupied, ShouldBeFalse)

			_, err = state.GetStateRecord(ctx, "racks/rack-6")
			So(err.Error(), ShouldContainSubstring, NotFound)
			_, err = state.GetStateRecord(ctx, "rpms/rpm-6")
			So(err.Error(), ShouldContainSubstring, NotFound)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "racks/rack-6")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRetire)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRetire)
			So(changes[0].GetEventLabel(), ShouldEqual, "rack")
		})

		Convey("Delete rack with switches, kvms - happy path", func() {
			kvm := &ufspb.KVM{
				Name: "kvm-5",
				Rack: "rack-5",
			}
			_, err := registration.CreateKVM(ctx, kvm)
			So(err, ShouldBeNil)

			switch5 := &ufspb.Switch{
				Name: "switch-5",
				Rack: "rack-5",
			}
			_, err = registration.CreateSwitch(ctx, switch5)
			So(err, ShouldBeNil)

			rack := &ufspb.Rack{
				Name: "rack-5",
				Rack: &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{},
				},
			}
			_, err = registration.CreateRack(ctx, rack)
			So(err, ShouldBeNil)

			_, err = state.BatchUpdateStates(ctx, []*ufspb.StateRecord{
				{
					ResourceName: "racks/rack-5",
					State:        ufspb.State_STATE_SERVING,
				},
				{
					ResourceName: "switches/switch-5",
					State:        ufspb.State_STATE_SERVING,
				},
				{
					ResourceName: "kvms/kvm-5",
					State:        ufspb.State_STATE_SERVING,
				},
			})
			So(err, ShouldBeNil)
			_, err = configuration.BatchUpdateDHCPs(ctx, []*ufspb.DHCPConfig{
				{
					Hostname: "kvm-5",
					Ip:       "1.2.3.4",
				},
			})
			So(err, ShouldBeNil)
			_, err = configuration.BatchUpdateIPs(ctx, []*ufspb.IP{
				{
					Id:       "ip1",
					Occupied: true,
					Ipv4Str:  "1.2.3.4",
					Vlan:     "fake_vlan",
					Ipv4:     uint32(100),
				},
			})
			So(err, ShouldBeNil)

			err = DeleteRack(ctx, "rack-5")
			So(err, ShouldBeNil)

			_, err = registration.GetRack(ctx, "rack-5")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)

			_, err = registration.GetKVM(ctx, "kvm-5")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
			_, err = configuration.GetDHCPConfig(ctx, "kvm-5")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
			resIPs, err := configuration.QueryIPByPropertyName(ctx, map[string]string{"ipv4_str": "1.2.3.4"})
			So(err, ShouldBeNil)
			So(resIPs, ShouldHaveLength, 1)
			So(resIPs[0].Occupied, ShouldBeFalse)

			_, err = registration.GetSwitch(ctx, "switch-5")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)

			_, err = state.GetStateRecord(ctx, "racks/rack-5")
			So(err.Error(), ShouldContainSubstring, NotFound)
			_, err = state.GetStateRecord(ctx, "switches/switch-5")
			So(err.Error(), ShouldContainSubstring, NotFound)
			_, err = state.GetStateRecord(ctx, "kvms/kvm-5")
			So(err.Error(), ShouldContainSubstring, NotFound)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "racks/rack-5")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRetire)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRetire)
			So(changes[0].GetEventLabel(), ShouldEqual, "rack")
		})
	})
}

func TestReplaceRack(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("ReplaceRacks", t, func() {
		Convey("Repalce an old Rack with new rack with RackLSE reference", func() {
			oldRack1 := &ufspb.Rack{
				Name: "rack-4",
			}
			_, err := registration.CreateRack(ctx, oldRack1)
			So(err, ShouldBeNil)

			rackLSE1 := &ufspb.RackLSE{
				Name:  "racklse-1",
				Racks: []string{"rack-0", "rack-50", "rack-4", "rack-7"},
			}
			_, err = inventory.CreateRackLSE(ctx, rackLSE1)
			So(err, ShouldBeNil)

			newRack2 := &ufspb.Rack{
				Name: "rack-100",
			}
			resp, err := ReplaceRack(ctx, oldRack1, newRack2)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, newRack2)

			rlse, err := inventory.GetRackLSE(ctx, "racklse-1")
			So(rlse, ShouldNotBeNil)
			So(rlse.GetRacks(), ShouldResemble, []string{"rack-0", "rack-50", "rack-100", "rack-7"})

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "racks/rack-4")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRetire)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRetire)
			So(changes[0].GetEventLabel(), ShouldEqual, "rack")

			changes, err = history.QueryChangesByPropertyName(ctx, "name", "racks/rack-100")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetEventLabel(), ShouldEqual, "rack")
		})

		Convey("Repalce an old Rack with already existing rack", func() {
			existingRack1 := &ufspb.Rack{
				Name: "rack-105",
			}
			_, err := registration.CreateRack(ctx, existingRack1)
			So(err, ShouldBeNil)

			oldRack1 := &ufspb.Rack{
				Name: "rack-5",
			}
			_, err = registration.CreateRack(ctx, oldRack1)
			So(err, ShouldBeNil)

			newRack2 := &ufspb.Rack{
				Name: "rack-105",
			}
			rresp, rerr := ReplaceRack(ctx, oldRack1, newRack2)
			So(rerr, ShouldNotBeNil)
			So(rresp, ShouldBeNil)
			So(rerr.Error(), ShouldContainSubstring, AlreadyExists)

			// No change are recorded as the replacement fails
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "racks/rack-5")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "racks/rack-105")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})
	})
}

func TestListRacks(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	racksWithSwitch := make([]*ufspb.Rack, 0, 2)
	racks := make([]*ufspb.Rack, 0, 4)
	for i := 0; i < 4; i++ {
		rack := &ufspb.Rack{
			Name: fmt.Sprintf("rack-%d", i),
			Rack: &ufspb.Rack_ChromeBrowserRack{
				ChromeBrowserRack: &ufspb.ChromeBrowserRack{},
			},
		}
		if i%2 == 0 {
			rack.Tags = []string{"tag-12"}
		}
		resp, _ := registration.CreateRack(ctx, rack)
		if i%2 == 0 {
			racksWithSwitch = append(racksWithSwitch, resp)
		}
		racks = append(racks, resp)
	}
	Convey("ListRacks", t, func() {
		Convey("List Racks - filter invalid - error", func() {
			_, _, err := ListRacks(ctx, 5, "", "invalid=mx-1", false)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Invalid field name invalid")
		})

		Convey("List Racks - filter switch - happy path", func() {
			resp, _, _ := ListRacks(ctx, 5, "", "tag=tag-12", false)
			So(resp, ShouldNotBeNil)
			So(resp, ShouldResembleProto, racksWithSwitch)
		})

		Convey("ListRacks - Full listing - happy path", func() {
			resp, _, _ := ListRacks(ctx, 5, "", "", false)
			So(resp, ShouldNotBeNil)
			So(resp, ShouldResembleProto, racks)
		})
	})
}

func TestBatchGetRacks(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("BatchGetRacks", t, func() {
		Convey("Batch get racks - happy path", func() {
			entities := make([]*ufspb.Rack, 4)
			for i := 0; i < 4; i++ {
				entities[i] = &ufspb.Rack{
					Name: fmt.Sprintf("rack-batchGet-%d", i),
				}
			}
			_, err := registration.BatchUpdateRacks(ctx, entities)
			So(err, ShouldBeNil)
			resp, err := registration.BatchGetRacks(ctx, []string{"rack-batchGet-0", "rack-batchGet-1", "rack-batchGet-2", "rack-batchGet-3"})
			So(err, ShouldBeNil)
			So(resp, ShouldHaveLength, 4)
			So(resp, ShouldResembleProto, entities)
		})
		Convey("Batch get racks  - missing id", func() {
			resp, err := registration.BatchGetRacks(ctx, []string{"rack-batchGet-non-existing"})
			So(err, ShouldNotBeNil)
			So(resp, ShouldBeNil)
			So(err.Error(), ShouldContainSubstring, "rack-batchGet-non-existing")
		})
		Convey("Batch get racks  - empty input", func() {
			resp, err := registration.BatchGetRacks(ctx, nil)
			So(err, ShouldBeNil)
			So(resp, ShouldHaveLength, 0)

			input := make([]string, 0)
			resp, err = registration.BatchGetRacks(ctx, input)
			So(err, ShouldBeNil)
			So(resp, ShouldHaveLength, 0)
		})
	})
}
