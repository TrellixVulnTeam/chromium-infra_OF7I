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
	"infra/unifiedfleet/app/model/registration"
	"infra/unifiedfleet/app/model/state"
)

func mockKVM(id string) *ufspb.KVM {
	return &ufspb.KVM{
		Name: id,
	}
}

func TestCreateKVM(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	rack1 := &ufspb.Rack{
		Name: "rack-1",
		Rack: &ufspb.Rack_ChromeBrowserRack{
			ChromeBrowserRack: &ufspb.ChromeBrowserRack{},
		},
	}
	registration.CreateRack(ctx, rack1)
	Convey("CreateKVM", t, func() {
		Convey("Create new kvm with already existing kvm - error", func() {
			kvm1 := &ufspb.KVM{
				Name: "kvm-1",
				Rack: "rack-5",
			}
			_, err := registration.CreateKVM(ctx, kvm1)

			resp, err := CreateKVM(ctx, kvm1)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "KVM kvm-1 already exists in the system")

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "kvms/kvm-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})

		Convey("Create new kvm with non existing chromePlatform", func() {
			kvm2 := &ufspb.KVM{
				Name:           "kvm-2",
				ChromePlatform: "chromePlatform-1",
				Rack:           "rack-1",
			}
			resp, err := CreateKVM(ctx, kvm2)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "There is no ChromePlatform with ChromePlatformID chromePlatform-1 in the system")

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "kvms/kvm-2")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})

		Convey("Create new kvm with existing resources", func() {
			chromePlatform2 := &ufspb.ChromePlatform{
				Name: "chromePlatform-2",
			}
			_, err := configuration.CreateChromePlatform(ctx, chromePlatform2)
			So(err, ShouldBeNil)

			kvm2 := &ufspb.KVM{
				Name:           "kvm-2",
				ChromePlatform: "chromePlatform-2",
				Rack:           "rack-1",
			}
			resp, err := CreateKVM(ctx, kvm2)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, kvm2)
			s, err := state.GetStateRecord(ctx, "kvms/kvm-2")
			So(err, ShouldBeNil)
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_REGISTERED)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "kvms/kvm-2")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetEventLabel(), ShouldEqual, "kvm")
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "kvms/kvm-2")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			So(msgs[0].Delete, ShouldBeFalse)
		})

		Convey("Create KVM - duplicated mac address", func() {
			kvm := &ufspb.KVM{
				Name:       "kvm-2-mac",
				MacAddress: "kvm-2-address",
			}
			_, err := registration.CreateKVM(ctx, kvm)
			kvm2 := &ufspb.KVM{
				Name:       "kvm-2-mac2",
				MacAddress: "kvm-2-address",
				Rack:       "rack-1",
			}
			_, err = CreateKVM(ctx, kvm2)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "mac_address kvm-2-address is already occupied")
		})

		Convey("Create new kvm with existing rack", func() {
			rack := &ufspb.Rack{
				Name: "rack-10",
				Rack: &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{},
				},
			}
			_, err := registration.CreateRack(ctx, rack)
			So(err, ShouldBeNil)

			kvm1 := &ufspb.KVM{
				Name: "kvm-20",
				Rack: "rack-10",
			}
			resp, err := CreateKVM(ctx, kvm1)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, kvm1)

			s, err := state.GetStateRecord(ctx, "kvms/kvm-20")
			So(err, ShouldBeNil)
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_REGISTERED)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "kvms/kvm-20")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetEventLabel(), ShouldEqual, "kvm")
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "kvms/kvm-20")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			So(msgs[0].Delete, ShouldBeFalse)
		})
	})
}

func TestUpdateKVM(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("UpdateKVM", t, func() {
		Convey("Update kvm with non-existing kvm", func() {
			rack1 := &ufspb.Rack{
				Name: "rack-1",
			}
			_, err := registration.CreateRack(ctx, rack1)
			So(err, ShouldBeNil)

			kvm1 := &ufspb.KVM{
				Name: "kvm-1",
				Rack: "rack-1",
			}
			resp, err := UpdateKVM(ctx, kvm1, nil)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "There is no KVM with KVMID kvm-1 in the system")

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "kvms/kvm-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})

		Convey("Update kvm with new rack", func() {
			rack3 := &ufspb.Rack{
				Name: "rack-3",
				Rack: &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{},
				},
			}
			_, err := registration.CreateRack(ctx, rack3)
			So(err, ShouldBeNil)

			rack4 := &ufspb.Rack{
				Name: "rack-4",
				Rack: &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{},
				},
			}
			_, err = registration.CreateRack(ctx, rack4)
			So(err, ShouldBeNil)

			kvm3 := &ufspb.KVM{
				Name: "kvm-3",
				Rack: "rack-3",
			}
			_, err = registration.CreateKVM(ctx, kvm3)
			So(err, ShouldBeNil)

			kvm3.Rack = "rack-4"
			resp, err := UpdateKVM(ctx, kvm3, nil)
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp, ShouldResembleProto, kvm3)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "kvms/kvm-3")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetEventLabel(), ShouldEqual, "kvm.rack")
			So(changes[0].GetOldValue(), ShouldEqual, "rack-3")
			So(changes[0].GetNewValue(), ShouldEqual, "rack-4")
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "kvms/kvm-3")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			So(msgs[0].Delete, ShouldBeFalse)
		})

		Convey("Update kvm with same rack", func() {
			rack := &ufspb.Rack{
				Name: "rack-5",
				Rack: &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{},
				},
			}
			_, err := registration.CreateRack(ctx, rack)
			So(err, ShouldBeNil)

			kvm1 := &ufspb.KVM{
				Name:       "kvm-5",
				Rack:       "rack-5",
				MacAddress: "kvm-10-address",
			}
			_, err = registration.CreateKVM(ctx, kvm1)
			So(err, ShouldBeNil)

			resp, err := UpdateKVM(ctx, kvm1, nil)
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp, ShouldResembleProto, kvm1)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "kvms/kvm-5")
			So(err, ShouldBeNil)
			// Nothing is changed for kvm-5
			So(changes, ShouldHaveLength, 0)
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "kvms/kvm-5")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			So(msgs[0].Delete, ShouldBeFalse)
		})

		Convey("Update kvm with non existing rack", func() {
			kvm1 := &ufspb.KVM{
				Name: "kvm-6",
			}
			_, err := registration.CreateKVM(ctx, kvm1)
			So(err, ShouldBeNil)

			kvm1.Rack = "rack-6"
			resp, err := UpdateKVM(ctx, kvm1, nil)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "There is no Rack with RackID rack-6 in the system.")

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "kvms/kvm-6")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})

		Convey("Partial Update kvm", func() {
			kvm := &ufspb.KVM{
				Name:           "kvm-7",
				ChromePlatform: "chromePlatform-7",
				Tags:           []string{"testkvm"},
			}
			_, err := registration.CreateKVM(ctx, kvm)
			So(err, ShouldBeNil)

			chromePlatform := &ufspb.ChromePlatform{
				Name: "chromePlatform-8",
			}
			_, err = configuration.CreateChromePlatform(ctx, chromePlatform)
			So(err, ShouldBeNil)

			kvm1 := &ufspb.KVM{
				Name:           "kvm-7",
				MacAddress:     "efgh",
				ChromePlatform: "chromePlatform-8",
			}
			resp, err := UpdateKVM(ctx, kvm1, &field_mask.FieldMask{Paths: []string{"platform", "macAddress"}})
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp.GetChromePlatform(), ShouldResemble, "chromePlatform-8")
			So(resp.GetMacAddress(), ShouldResemble, "efgh")
			So(resp.GetTags(), ShouldResemble, []string{"testkvm"})
		})

		Convey("Partial Update kvm mac address - duplicated mac address", func() {
			kvm := &ufspb.KVM{
				Name:       "kvm-8",
				MacAddress: "kvm-8-address",
			}
			_, err := registration.CreateKVM(ctx, kvm)
			So(err, ShouldBeNil)
			kvm2 := &ufspb.KVM{
				Name:       "kvm-8.2",
				MacAddress: "kvm-8.2-address",
			}
			_, err = registration.CreateKVM(ctx, kvm2)
			So(err, ShouldBeNil)

			kvm1 := &ufspb.KVM{
				Name:       "kvm-8",
				MacAddress: "kvm-8.2-address",
			}
			_, err = UpdateKVM(ctx, kvm1, &field_mask.FieldMask{Paths: []string{"macAddress"}})
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "mac_address kvm-8.2-address is already occupied")
		})

		Convey("Update kvm mac address - duplicated mac address", func() {
			kvm := &ufspb.KVM{
				Name:       "kvm-9",
				MacAddress: "kvm-9-address",
			}
			_, err := registration.CreateKVM(ctx, kvm)
			So(err, ShouldBeNil)
			kvm2 := &ufspb.KVM{
				Name:       "kvm-9.2",
				MacAddress: "kvm-9.2-address",
			}
			_, err = registration.CreateKVM(ctx, kvm2)
			So(err, ShouldBeNil)

			kvm1 := &ufspb.KVM{
				Name:       "kvm-9",
				MacAddress: "kvm-9.2-address",
			}
			_, err = UpdateKVM(ctx, kvm1, nil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "mac_address kvm-9.2-address is already occupied")
		})

	})
}

func TestDeleteKVM(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("DeleteKVM", t, func() {
		Convey("Delete kvm by non-existing ID - error", func() {
			err := DeleteKVM(ctx, "kvm-10")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Delete failed - unable to delete kvm kvm-10")

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "kvms/kvm-10")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})

		Convey("Delete KVM by existing ID with machine reference", func() {
			KVM1 := &ufspb.KVM{
				Name: "KVM-1",
			}
			_, err := registration.CreateKVM(ctx, KVM1)
			So(err, ShouldBeNil)

			chromeBrowserMachine1 := &ufspb.Machine{
				Name: "machine-1",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{
						KvmInterface: &ufspb.KVMInterface{
							Kvm: "KVM-1",
						},
					},
				},
			}
			_, err = registration.CreateMachine(ctx, chromeBrowserMachine1)
			So(err, ShouldBeNil)

			err = DeleteKVM(ctx, "KVM-1")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Machines referring the KVM:")

			resp, err := registration.GetKVM(ctx, "KVM-1")
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, KVM1)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "kvms/KVM-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})

		Convey("Delete KVM successfully", func() {
			kvm2 := mockKVM("kvm-2")
			_, err := registration.CreateKVM(ctx, kvm2)
			So(err, ShouldBeNil)
			_, err = state.BatchUpdateStates(ctx, []*ufspb.StateRecord{
				{
					ResourceName: "kvms/kvm-2",
					State:        ufspb.State_STATE_SERVING,
				},
			})
			So(err, ShouldBeNil)

			err = DeleteKVM(ctx, "kvm-2")
			So(err, ShouldBeNil)

			resp, err := registration.GetKVM(ctx, "kvm-2")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)

			_, err = state.GetStateRecord(ctx, "kvms/kvm-2")
			So(err.Error(), ShouldContainSubstring, NotFound)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "kvms/kvm-2")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRetire)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRetire)
			So(changes[0].GetEventLabel(), ShouldEqual, "kvm")
		})

		Convey("Delete KVM successfully together with deleting ip", func() {
			kvm2 := mockKVM("kvm-ip2")
			_, err := registration.CreateKVM(ctx, kvm2)
			So(err, ShouldBeNil)
			_, err = state.BatchUpdateStates(ctx, []*ufspb.StateRecord{
				{
					ResourceName: "kvms/kvm-ip2",
					State:        ufspb.State_STATE_SERVING,
				},
			})
			So(err, ShouldBeNil)
			_, err = configuration.BatchUpdateDHCPs(ctx, []*ufspb.DHCPConfig{
				{
					Hostname: "kvm-ip2",
					Ip:       "1.2.3.4",
				},
			})
			So(err, ShouldBeNil)
			_, err = configuration.ImportIPs(ctx, []*ufspb.IP{
				{
					Id:       "vlan-1:123",
					Ipv4Str:  "1.2.3.4",
					Vlan:     "vlan-1",
					Occupied: true,
					Ipv4:     123,
				},
			})
			So(err, ShouldBeNil)

			err = DeleteKVM(ctx, "kvm-ip2")
			So(err, ShouldBeNil)
			ip, err := configuration.QueryIPByPropertyName(ctx, map[string]string{"ipv4_str": "1.2.3.4"})
			So(err, ShouldBeNil)
			So(ip, ShouldHaveLength, 1)
			So(ip[0].GetOccupied(), ShouldBeFalse)
			_, err = configuration.GetDHCPConfig(ctx, "kvm-ip2")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)

			resp, err := registration.GetKVM(ctx, "kvm-2")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)

			_, err = state.GetStateRecord(ctx, "kvms/kvm-2")
			So(err.Error(), ShouldContainSubstring, NotFound)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "kvms/kvm-ip2")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRetire)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRetire)
			So(changes[0].GetEventLabel(), ShouldEqual, "kvm")
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "kvms/kvm-ip2")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			So(msgs[0].Delete, ShouldBeTrue)
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "dhcps/kvm-ip2")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			So(msgs[0].Delete, ShouldBeTrue)
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "states/kvms/kvm-ip2")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			So(msgs[0].Delete, ShouldBeTrue)
		})
	})
}

func TestListKVMs(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	kvmsWithChromeplatform := make([]*ufspb.KVM, 0, 2)
	kvms := make([]*ufspb.KVM, 0, 4)
	for i := 0; i < 4; i++ {
		kvm := mockKVM(fmt.Sprintf("kvm-%d", i))
		if i%2 == 0 {
			kvm.ChromePlatform = "chromeplatform-12"
		}
		resp, _ := registration.CreateKVM(ctx, kvm)
		if i%2 == 0 {
			kvmsWithChromeplatform = append(kvmsWithChromeplatform, resp)
		}
		kvms = append(kvms, resp)
	}
	Convey("ListKVMs", t, func() {
		Convey("List KVMs - filter invalid - error", func() {
			_, _, err := ListKVMs(ctx, 5, "", "invalid=mx-1", false)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Invalid field name invalid")
		})

		Convey("List KVMs - filter chromeplatform - happy path", func() {
			resp, _, _ := ListKVMs(ctx, 5, "", "platform=chromeplatform-12", false)
			So(resp, ShouldNotBeNil)
			So(resp, ShouldResembleProto, kvmsWithChromeplatform)
		})

		Convey("ListKVMs - Full listing - happy path", func() {
			resp, _, _ := ListKVMs(ctx, 5, "", "", false)
			So(resp, ShouldNotBeNil)
			So(resp, ShouldResembleProto, kvms)
		})
	})
}

func TestBatchGetKVMs(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("BatchGetKVMs", t, func() {
		Convey("Batch get kvms - happy path", func() {
			kvms := make([]*ufspb.KVM, 0, 4)
			for i := 0; i < 4; i++ {
				kvm := mockKVM(fmt.Sprintf("kvm-batchGet-%d", i))
				resp, err := registration.CreateKVM(ctx, kvm)
				So(err, ShouldBeNil)
				kvms = append(kvms, resp)
			}
			resp, err := registration.BatchGetKVM(ctx, []string{"kvm-batchGet-0", "kvm-batchGet-1", "kvm-batchGet-2", "kvm-batchGet-3"})
			So(err, ShouldBeNil)
			So(resp, ShouldHaveLength, 4)
			So(resp, ShouldResembleProto, kvms)
		})
		Convey("Batch get kvms - missing id", func() {
			resp, err := registration.BatchGetKVM(ctx, []string{"kvm-batchGet-non-existing"})
			So(err, ShouldNotBeNil)
			So(resp, ShouldBeNil)
			So(err.Error(), ShouldContainSubstring, "kvm-batchGet-non-existing")
		})
		Convey("Batch get kvms - empty input", func() {
			resp, err := registration.BatchGetKVM(ctx, nil)
			So(err, ShouldBeNil)
			So(resp, ShouldHaveLength, 0)

			input := make([]string, 0)
			resp, err = registration.BatchGetKVM(ctx, input)
			So(err, ShouldBeNil)
			So(resp, ShouldHaveLength, 0)
		})
	})
}
