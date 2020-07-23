// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"

	ufspb "infra/unifiedfleet/api/v1/proto"
	"infra/unifiedfleet/app/model/history"
	"infra/unifiedfleet/app/model/registration"
)

func TestRackRegistration(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("TestRackRegistration", t, func() {
		Convey("Register rack with already existing rack, switches, kvms and rpms", func() {
			rack := &ufspb.Rack{
				Name: "rack-1",
				Rack: &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{
						Switches: []string{"switch-1"},
						Kvms:     []string{"kvm-1"},
						Rpms:     []string{"rpm-1"},
					},
				},
			}
			_, err := registration.CreateRack(ctx, rack)
			So(err, ShouldBeNil)

			kvm := &ufspb.KVM{
				Name: "kvm-1",
			}
			_, err = registration.CreateKVM(ctx, kvm)
			So(err, ShouldBeNil)

			rpm := &ufspb.RPM{
				Name: "rpm-1",
			}
			_, err = registration.CreateRPM(ctx, rpm)
			So(err, ShouldBeNil)

			switch1 := &ufspb.Switch{
				Name: "switch-1",
			}
			_, err = registration.CreateSwitch(ctx, switch1)
			So(err, ShouldBeNil)

			_, _, _, _, err = RackRegistration(ctx, rack, []*ufspb.Switch{switch1}, []*ufspb.KVM{kvm}, []*ufspb.RPM{rpm})
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring,
				"Rack rack-1 already exists in the system.\n"+
					"Switch switch-1 already exists in the system.\n"+
					"KVM kvm-1 already exists in the system.\n"+
					"RPM rpm-1 already exists in the system.\n")

			// No changes are recorded as the registration fails
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "racks/rack-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "switches/switch-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "rpms/rpm-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "kvms/kvm-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})

		Convey("Register rack with invalid KVM(referencing non existing resources chromeplatform)", func() {
			rack := &ufspb.Rack{
				Name: "rack-4",
				Rack: &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{},
				},
			}

			kvm := &ufspb.KVM{
				Name:           "kvm-4",
				ChromePlatform: "chromePlatform-4",
			}

			_, _, _, _, err := RackRegistration(ctx, rack, nil, []*ufspb.KVM{kvm}, nil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Cannot create rack rack-4:\n"+
				"There is no ChromePlatform with ChromePlatformID chromePlatform-4 in the system.")

			// No changes are recorded as the registration fails
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "racks/rack-4")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "kvms/kvm-4")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})

		Convey("Register browser rack happy path", func() {
			rack := &ufspb.Rack{
				Name: "rack-3",
				Rack: &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{},
				},
			}
			kvm := &ufspb.KVM{
				Name: "kvm-3",
			}
			rpm := &ufspb.RPM{
				Name: "rpm-3",
			}
			switch3 := &ufspb.Switch{
				Name: "switch-3",
			}

			r, s, k, rp, err := RackRegistration(ctx, rack, []*ufspb.Switch{switch3}, []*ufspb.KVM{kvm}, []*ufspb.RPM{rpm})
			So(err, ShouldBeNil)
			So(r, ShouldResembleProto, rack)
			So(s, ShouldResembleProto, []*ufspb.Switch{switch3})
			So(k, ShouldResembleProto, []*ufspb.KVM{kvm})
			So(rp, ShouldResembleProto, []*ufspb.RPM{rpm})

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "racks/rack-3")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetEventLabel(), ShouldEqual, "rack")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "switches/switch-3")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetEventLabel(), ShouldEqual, "switch")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "rpms/rpm-3")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetEventLabel(), ShouldEqual, "rpm")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "kvms/kvm-3")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetEventLabel(), ShouldEqual, "kvm")
		})
	})
}
