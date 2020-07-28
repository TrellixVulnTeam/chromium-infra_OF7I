// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"

	ufspb "infra/unifiedfleet/api/v1/proto"
	"infra/unifiedfleet/app/model/configuration"
	. "infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/model/history"
	"infra/unifiedfleet/app/model/registration"
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
			}
			_, err := registration.CreateKVM(ctx, kvm1)

			resp, err := CreateKVM(ctx, kvm1, "rack-5")
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
			}
			resp, err := CreateKVM(ctx, kvm2, "rack-1")
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
			}
			resp, err := CreateKVM(ctx, kvm2, "rack-1")
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, kvm2)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "kvms/kvm-2")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetEventLabel(), ShouldEqual, "kvm")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "racks/rack-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, "[]")
			So(changes[0].GetNewValue(), ShouldEqual, "[kvm-2]")
			So(changes[0].GetEventLabel(), ShouldEqual, "rack.chrome_browser_rack.kvms")
		})

		Convey("Create new kvm with existing rack with kvms", func() {
			rack := &ufspb.Rack{
				Name: "rack-10",
				Rack: &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{
						Kvms: []string{"kvm-5"},
					},
				},
			}
			_, err := registration.CreateRack(ctx, rack)
			So(err, ShouldBeNil)

			kvm1 := &ufspb.KVM{
				Name: "kvm-20",
			}
			resp, err := CreateKVM(ctx, kvm1, "rack-10")
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, kvm1)

			mresp, err := registration.GetRack(ctx, "rack-10")
			So(err, ShouldBeNil)
			So(mresp.GetChromeBrowserRack().GetKvms(), ShouldResemble, []string{"kvm-5", "kvm-20"})

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "kvms/kvm-20")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetEventLabel(), ShouldEqual, "kvm")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "racks/rack-10")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, "[kvm-5]")
			So(changes[0].GetNewValue(), ShouldEqual, "[kvm-5 kvm-20]")
			So(changes[0].GetEventLabel(), ShouldEqual, "rack.chrome_browser_rack.kvms")
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
			}
			resp, err := UpdateKVM(ctx, kvm1, "rack-1")
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
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{
						Kvms: []string{"kvm-3"},
					},
				},
			}
			_, err := registration.CreateRack(ctx, rack3)
			So(err, ShouldBeNil)

			rack4 := &ufspb.Rack{
				Name: "rack-4",
				Rack: &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{
						Kvms: []string{"kvm-4"},
					},
				},
			}
			_, err = registration.CreateRack(ctx, rack4)
			So(err, ShouldBeNil)

			kvm3 := &ufspb.KVM{
				Name: "kvm-3",
			}
			_, err = registration.CreateKVM(ctx, kvm3)
			So(err, ShouldBeNil)

			resp, err := UpdateKVM(ctx, kvm3, "rack-4")
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp, ShouldResembleProto, kvm3)

			mresp, err := registration.GetRack(ctx, "rack-3")
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(mresp.GetChromeBrowserRack().GetKvms(), ShouldBeNil)

			mresp, err = registration.GetRack(ctx, "rack-4")
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(mresp.GetChromeBrowserRack().GetKvms(), ShouldResemble, []string{"kvm-4", "kvm-3"})

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "kvms/kvm-3")
			So(err, ShouldBeNil)
			// Nothing is changed for kvm-3
			So(changes, ShouldHaveLength, 0)
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "racks/rack-4")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, "[kvm-4]")
			So(changes[0].GetNewValue(), ShouldEqual, "[kvm-4 kvm-3]")
			So(changes[0].GetEventLabel(), ShouldEqual, "rack.chrome_browser_rack.kvms")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "racks/rack-3")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, "[kvm-3]")
			So(changes[0].GetNewValue(), ShouldEqual, "[]")
			So(changes[0].GetEventLabel(), ShouldEqual, "rack.chrome_browser_rack.kvms")
		})

		Convey("Update kvm with same rack", func() {
			rack := &ufspb.Rack{
				Name: "rack-5",
				Rack: &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{
						Kvms: []string{"kvm-5"},
					},
				},
			}
			_, err := registration.CreateRack(ctx, rack)
			So(err, ShouldBeNil)

			kvm1 := &ufspb.KVM{
				Name: "kvm-5",
			}
			_, err = registration.CreateKVM(ctx, kvm1)
			So(err, ShouldBeNil)

			resp, err := UpdateKVM(ctx, kvm1, "rack-5")
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp, ShouldResembleProto, kvm1)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "kvms/kvm-5")
			So(err, ShouldBeNil)
			// Nothing is changed for kvm-5
			So(changes, ShouldHaveLength, 0)
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "racks/rack-5")
			So(err, ShouldBeNil)
			// Nothing is changed for rack-5
			So(changes, ShouldHaveLength, 0)
		})

		Convey("Update kvm with non existing rack", func() {
			kvm1 := &ufspb.KVM{
				Name: "kvm-6",
			}
			_, err := registration.CreateKVM(ctx, kvm1)
			So(err, ShouldBeNil)

			resp, err := UpdateKVM(ctx, kvm1, "rack-6")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "There is no Rack with RackID rack-6 in the system.")

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "kvms/kvm-6")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
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
			rack := &ufspb.Rack{
				Name: "rack-5",
				Rack: &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{
						Kvms: []string{"kvm-1"},
					},
				},
			}
			_, err := registration.CreateRack(ctx, rack)
			So(err, ShouldBeNil)

			KVM1 := &ufspb.KVM{
				Name: "KVM-1",
			}
			_, err = registration.CreateKVM(ctx, KVM1)
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
			rack := &ufspb.Rack{
				Name: "rack-6",
				Rack: &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{
						Kvms: []string{"kvm-2"},
					},
				},
			}
			_, err := registration.CreateRack(ctx, rack)
			So(err, ShouldBeNil)

			kvm2 := mockKVM("kvm-2")
			_, err = registration.CreateKVM(ctx, kvm2)
			So(err, ShouldBeNil)

			err = DeleteKVM(ctx, "kvm-2")
			So(err, ShouldBeNil)

			resp, err := registration.GetKVM(ctx, "kvm-2")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)

			rresp, err := registration.GetRack(ctx, "rack-6")
			So(rresp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(rresp.GetChromeBrowserRack().GetKvms(), ShouldBeNil)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "kvms/kvm-2")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRetire)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRetire)
			So(changes[0].GetEventLabel(), ShouldEqual, "kvm")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "racks/rack-6")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, "[kvm-2]")
			So(changes[0].GetNewValue(), ShouldEqual, "[]")
			So(changes[0].GetEventLabel(), ShouldEqual, "rack.chrome_browser_rack.kvms")
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
