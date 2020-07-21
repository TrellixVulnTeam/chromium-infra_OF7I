// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"

	fleet "infra/unifiedfleet/api/v1/proto"
	chromeosLab "infra/unifiedfleet/api/v1/proto/chromeos/lab"
	. "infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/model/history"
	"infra/unifiedfleet/app/model/inventory"
	"infra/unifiedfleet/app/model/registration"
)

func mockDutMachineLSE(name string) *fleet.MachineLSE {
	dut := &chromeosLab.DeviceUnderTest{
		Hostname: name,
	}
	device := &fleet.ChromeOSDeviceLSE_Dut{
		Dut: dut,
	}
	deviceLse := &fleet.ChromeOSDeviceLSE{
		Device: device,
	}
	chromeosLse := &fleet.ChromeOSMachineLSE_DeviceLse{
		DeviceLse: deviceLse,
	}
	chromeOSMachineLse := &fleet.ChromeOSMachineLSE{
		ChromeosLse: chromeosLse,
	}
	lse := &fleet.MachineLSE_ChromeosMachineLse{
		ChromeosMachineLse: chromeOSMachineLse,
	}
	return &fleet.MachineLSE{
		Name:     name,
		Hostname: name,
		Lse:      lse,
	}
}

func mockLabstationMachineLSE(name string) *fleet.MachineLSE {
	labstation := &chromeosLab.Labstation{
		Hostname: name,
	}
	device := &fleet.ChromeOSDeviceLSE_Labstation{
		Labstation: labstation,
	}
	deviceLse := &fleet.ChromeOSDeviceLSE{
		Device: device,
	}
	chromeosLse := &fleet.ChromeOSMachineLSE_DeviceLse{
		DeviceLse: deviceLse,
	}
	chromeOSMachineLse := &fleet.ChromeOSMachineLSE{
		ChromeosLse: chromeosLse,
	}
	lse := &fleet.MachineLSE_ChromeosMachineLse{
		ChromeosMachineLse: chromeOSMachineLse,
	}
	return &fleet.MachineLSE{
		Name:     name,
		Hostname: name,
		Lse:      lse,
	}
}

func TestCreateMachineLSE(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("CreateMachineLSE", t, func() {
		Convey("Create new machineLSE with non existing machines", func() {
			machineLSE1 := &fleet.MachineLSE{
				Hostname: "machinelse-1",
				Machines: []string{"machine-1", "machine-2"},
			}
			resp, err := CreateMachineLSE(ctx, machineLSE1)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "There is no Machine")

			// No changes are recorded as the creation fails
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "machineLSEs/machinelse-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})
		Convey("Create new machineLSE with existing machines", func() {
			machine1 := &fleet.Machine{
				Name: "machine-1",
			}
			mresp, merr := registration.CreateMachine(ctx, machine1)
			So(merr, ShouldBeNil)
			So(mresp, ShouldResembleProto, machine1)

			machineLSE2 := &fleet.MachineLSE{
				Hostname: "machinelse-2",
				Machines: []string{"machine-1"},
			}
			resp, err := CreateMachineLSE(ctx, machineLSE2)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, machineLSE2)

			// No changes are recorded as the creation fails
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "machineLSEs/machinelse-2")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetEventLabel(), ShouldEqual, "machine_lse")
		})
	})
}

func TestCreateMachineLSEDUT(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	labstationMachinelse := mockLabstationMachineLSE("BlueLabstation-0")
	inventory.CreateMachineLSE(ctx, labstationMachinelse)

	servo1 := &chromeosLab.Servo{
		ServoHostname: "BlueLabstation-0",
		ServoPort:     22,
	}
	peripherals1 := &chromeosLab.Peripherals{
		Servo: servo1,
	}
	dutMachinelse1 := mockDutMachineLSE("DUTMachineLSE-1")
	dutMachinelse1.GetChromeosMachineLse().GetDeviceLse().GetDut().Peripherals = peripherals1
	CreateMachineLSE(ctx, dutMachinelse1)

	servo2 := &chromeosLab.Servo{
		ServoHostname: "BlueLabstation-0",
		ServoPort:     33,
	}
	peripherals2 := &chromeosLab.Peripherals{
		Servo: servo2,
	}
	dutMachinelse2 := mockDutMachineLSE("DUTMachineLSE-2")
	dutMachinelse2.GetChromeosMachineLse().GetDeviceLse().GetDut().Peripherals = peripherals2
	CreateMachineLSE(ctx, dutMachinelse2)

	labstationMachinelse, _ = inventory.GetMachineLSE(ctx, "BlueLabstation-0")
	servos := labstationMachinelse.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetServos()
	Convey("CreateMachineLSE for a DUT", t, func() {
		Convey("Create machineLSE DUT without Servo Info", func() {
			dutMachinelse := mockDutMachineLSE("DUTMachineLSE-10")
			resp, err := CreateMachineLSE(ctx, dutMachinelse)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, dutMachinelse)
		})

		Convey("Create existing machineLSE DUT", func() {
			dutMachinelse := mockDutMachineLSE("DUTMachineLSE-11")
			inventory.CreateMachineLSE(ctx, dutMachinelse)

			resp, err := CreateMachineLSE(ctx, dutMachinelse)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, AlreadyExists)
		})

		Convey("Create machineLSE DUT with distinct Servo Info and existing Labstation", func() {
			servo := &chromeosLab.Servo{
				ServoHostname: "BlueLabstation-0",
				ServoPort:     55,
			}
			peripherals := &chromeosLab.Peripherals{
				Servo: servo,
			}
			dutMachinelse := mockDutMachineLSE("DUTMachineLSE-12")
			dutMachinelse.GetChromeosMachineLse().GetDeviceLse().GetDut().Peripherals = peripherals
			resp, err := CreateMachineLSE(ctx, dutMachinelse)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, dutMachinelse)

			servos = append(servos, servo)
			resp, _ = inventory.GetMachineLSE(ctx, "BlueLabstation-0")
			So(resp.GetChromeosMachineLse().GetDeviceLse().GetLabstation().Servos,
				ShouldResembleProto, servos)
		})

		Convey("Create machineLSE DUT with non-existing Labstation", func() {
			servo := &chromeosLab.Servo{
				ServoHostname: "BlueLabstation-99",
				ServoPort:     12,
			}
			peripherals := &chromeosLab.Peripherals{
				Servo: servo,
			}
			dutMachinelse := mockDutMachineLSE("DUTmachinelse-13")
			dutMachinelse.GetChromeosMachineLse().GetDeviceLse().GetDut().Peripherals = peripherals
			resp, err := CreateMachineLSE(ctx, dutMachinelse)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "not found in the system")
		})

		Convey("Create machineLSE with already assigned ServoHostname and ServoPort", func() {
			dutMachinelse := mockDutMachineLSE("DUTmachinelse-13")
			dutMachinelse.GetChromeosMachineLse().GetDeviceLse().GetDut().Peripherals = peripherals2
			resp, err := CreateMachineLSE(ctx, dutMachinelse)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "already in use")
		})
	})
}

func TestCreateMachineLSELabstation(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("CreateMachineLSE for a Labstation", t, func() {
		Convey("Create machineLSE Labstation with Servo Info", func() {
			labstationMachinelse1 := mockLabstationMachineLSE("RedLabstation-0")
			servo := &chromeosLab.Servo{
				ServoHostname: "RedLabstation-0",
				ServoPort:     22,
			}
			labstationMachinelse1.GetChromeosMachineLse().GetDeviceLse().GetLabstation().Servos = []*chromeosLab.Servo{servo}
			resp, err := CreateMachineLSE(ctx, labstationMachinelse1)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Servos are not allowed to be added")

			// No changes are recorded as the creation fails
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "machineLSEs/RedLabstation-0")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})
		Convey("Create machineLSE Labstation without Servo Info", func() {
			labstationMachinelse1 := mockLabstationMachineLSE("RedLabstation-1")
			resp, err := CreateMachineLSE(ctx, labstationMachinelse1)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, labstationMachinelse1)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "machineLSEs/RedLabstation-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetEventLabel(), ShouldEqual, "machine_lse")
		})
	})
}

func TestUpdateMachineLSEDUT(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	labstationMachinelse := mockLabstationMachineLSE("BlueLabstation-10")
	inventory.CreateMachineLSE(ctx, labstationMachinelse)

	servo1 := &chromeosLab.Servo{
		ServoHostname: "BlueLabstation-10",
		ServoPort:     21,
	}
	peripherals1 := &chromeosLab.Peripherals{
		Servo: servo1,
	}
	dutMachinelse1 := mockDutMachineLSE("DUTMachineLSE-21")
	dutMachinelse1.GetChromeosMachineLse().GetDeviceLse().GetDut().Peripherals = peripherals1
	CreateMachineLSE(ctx, dutMachinelse1)

	servo2 := &chromeosLab.Servo{
		ServoHostname: "BlueLabstation-10",
		ServoPort:     22,
	}
	peripherals2 := &chromeosLab.Peripherals{
		Servo: servo2,
	}
	dutMachinelse2 := mockDutMachineLSE("DUTMachineLSE-22")
	dutMachinelse2.GetChromeosMachineLse().GetDeviceLse().GetDut().Peripherals = peripherals2
	CreateMachineLSE(ctx, dutMachinelse2)

	labstationMachinelse, _ = inventory.GetMachineLSE(ctx, "BlueLabstation-10")
	servos := labstationMachinelse.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetServos()
	Convey("UpdateMachineLSE for a DUT", t, func() {
		Convey("Update non-existing machineLSE DUT", func() {
			dutMachinelse := mockDutMachineLSE("DUTMachineLSE-23")
			resp, err := UpdateMachineLSE(ctx, dutMachinelse)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})

		Convey("Update machineLSE DUT with same ServerPort and same ServoHostname", func() {
			servo3 := &chromeosLab.Servo{
				ServoHostname: "BlueLabstation-10",
				ServoPort:     21,
			}
			peripherals3 := &chromeosLab.Peripherals{
				Servo: servo3,
			}
			dutMachinelse3 := mockDutMachineLSE("DUTMachineLSE-21")
			dutMachinelse3.GetChromeosMachineLse().GetDeviceLse().GetDut().Peripherals = peripherals3
			resp, err := UpdateMachineLSE(ctx, dutMachinelse3)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, dutMachinelse3)

			resp, _ = inventory.GetMachineLSE(ctx, "BlueLabstation-10")
			So(resp.GetChromeosMachineLse().GetDeviceLse().GetLabstation().Servos,
				ShouldResembleProto, servos)
		})

		Convey("Update machineLSE DUT with different ServerPort and same ServoHostname", func() {
			servo3 := &chromeosLab.Servo{
				ServoHostname: "BlueLabstation-10",
				ServoPort:     358,
			}
			peripherals3 := &chromeosLab.Peripherals{
				Servo: servo3,
			}
			dutMachinelse3 := mockDutMachineLSE("DUTMachineLSE-22")
			dutMachinelse3.GetChromeosMachineLse().GetDeviceLse().GetDut().Peripherals = peripherals3
			resp, err := UpdateMachineLSE(ctx, dutMachinelse3)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, dutMachinelse3)

			dummyServos := []*chromeosLab.Servo{servo1, servo3}

			resp, _ = inventory.GetMachineLSE(ctx, "BlueLabstation-10")
			So(resp.GetChromeosMachineLse().GetDeviceLse().GetLabstation().Servos,
				ShouldResembleProto, dummyServos)
		})

		Convey("Update machineLSE DUT with different ServoHostname", func() {
			labstationMachinelse1 := mockLabstationMachineLSE("BlueLabstation-17")
			resp, err := inventory.CreateMachineLSE(ctx, labstationMachinelse1)

			labstationMachinelse2 := mockLabstationMachineLSE("BlueLabstation-18")
			resp, err = inventory.CreateMachineLSE(ctx, labstationMachinelse2)

			servo1 := &chromeosLab.Servo{
				ServoHostname: "BlueLabstation-17",
				ServoPort:     17,
			}
			peripherals1 := &chromeosLab.Peripherals{
				Servo: servo1,
			}
			dutMachinelse1 := mockDutMachineLSE("DUTMachineLSE-17")
			dutMachinelse1.GetChromeosMachineLse().GetDeviceLse().GetDut().Peripherals = peripherals1
			resp, err = CreateMachineLSE(ctx, dutMachinelse1)

			servo2 := &chromeosLab.Servo{
				ServoHostname: "BlueLabstation-18",
				ServoPort:     18,
			}
			peripherals2 := &chromeosLab.Peripherals{
				Servo: servo2,
			}
			dutMachinelse2 := mockDutMachineLSE("DUTMachineLSE-17")
			dutMachinelse2.GetChromeosMachineLse().GetDeviceLse().GetDut().Peripherals = peripherals2
			resp, err = UpdateMachineLSE(ctx, dutMachinelse2)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, dutMachinelse2)

			resp, _ = inventory.GetMachineLSE(ctx, "BlueLabstation-17")
			servos := resp.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetServos()
			So(servos, ShouldBeEmpty)

			resp, _ = inventory.GetMachineLSE(ctx, "BlueLabstation-18")
			servos = resp.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetServos()
			So(servo2, ShouldResembleProto, servos[0])
		})
	})
}

func TestUpdateMachineLSELabstation(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("UpdateMachineLSE for a Labstation", t, func() {
		Convey("Update machineLSE Labstation with Servo Info", func() {
			labstationMachinelse1 := mockLabstationMachineLSE("RedLabstation-10")
			inventory.CreateMachineLSE(ctx, labstationMachinelse1)

			labstationMachinelse2 := mockLabstationMachineLSE("RedLabstation-10")
			servo := &chromeosLab.Servo{
				ServoHostname: "RedLabstation-10",
				ServoPort:     22,
			}
			labstationMachinelse2.GetChromeosMachineLse().GetDeviceLse().GetLabstation().Servos = []*chromeosLab.Servo{servo}
			resp, err := UpdateMachineLSE(ctx, labstationMachinelse2)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Servos are not allowed to be updated")

			// No changes are recorded as the updating fails
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "machineLSEs/RedLabstation-10")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})

		Convey("Update machineLSE Labstation without Servo Info", func() {
			labstationMachinelse1 := mockLabstationMachineLSE("RedLabstation-11")
			inventory.CreateMachineLSE(ctx, labstationMachinelse1)

			labstationMachinelse2 := mockLabstationMachineLSE("RedLabstation-11")
			resp, err := UpdateMachineLSE(ctx, labstationMachinelse2)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, labstationMachinelse2)

			// No changes happened in this update
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "machineLSEs/RedLabstation-11")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})
	})
}

func TestUpdateMachineLSE(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("UpdateMachineLSE for a browser machine", t, func() {
		Convey("Update machineLSE with vms", func() {
			machine1 := &fleet.Machine{
				Name: "machine-update1",
			}
			machineLSE1 := &fleet.MachineLSE{
				Name:     "machinelse-update1",
				Hostname: "machinelse-update1",
				Machines: []string{"machine-update1"},
				Lse: &fleet.MachineLSE_ChromeBrowserMachineLse{
					ChromeBrowserMachineLse: &fleet.ChromeBrowserMachineLSE{
						Vms: []*fleet.VM{
							{
								Name:       "vm1",
								MacAddress: "old_mac_address",
							},
						},
					},
				},
			}
			_, err := registration.CreateMachine(ctx, machine1)
			So(err, ShouldBeNil)
			_, err = inventory.CreateMachineLSE(ctx, machineLSE1)
			So(err, ShouldBeNil)
			machineLSE1.GetChromeBrowserMachineLse().Vms = []*fleet.VM{
				{
					Name:       "vm1",
					MacAddress: "new_mac_address",
				},
				{
					Name:       "vm2",
					MacAddress: "vm2_mac_address",
				},
			}
			machineLSE1.GetChromeBrowserMachineLse().OsVersion = &fleet.OSVersion{
				Value: "new_os",
			}
			m, err := UpdateMachineLSE(ctx, machineLSE1)
			So(err, ShouldBeNil)
			So(m.GetChromeBrowserMachineLse().GetVms(), ShouldHaveLength, 2)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "machineLSEs/machinelse-update1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetEventLabel(), ShouldEqual, "machine_lse.chrome_browser_machine_lse.os_version")
			So(changes[0].OldValue, ShouldEqual, "<nil>")
			So(changes[0].NewValue, ShouldEqual, "value:\"new_os\"")

			changes, err = history.QueryChangesByPropertyName(ctx, "name", "vms/vm2")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetEventLabel(), ShouldEqual, "vm")

			changes, err = history.QueryChangesByPropertyName(ctx, "name", "vms/vm1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetEventLabel(), ShouldEqual, "vm.mac_address")
			So(changes[0].GetOldValue(), ShouldEqual, "old_mac_address")
			So(changes[0].GetNewValue(), ShouldEqual, "new_mac_address")
		})

		Convey("Update machineLSE Labstation without Servo Info", func() {
			labstationMachinelse1 := mockLabstationMachineLSE("RedLabstation-11")
			inventory.CreateMachineLSE(ctx, labstationMachinelse1)

			labstationMachinelse2 := mockLabstationMachineLSE("RedLabstation-11")
			resp, err := UpdateMachineLSE(ctx, labstationMachinelse2)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, labstationMachinelse2)

			// No changes happened in this update
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "machineLSEs/RedLabstation-11")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})
	})
}

func TestDeleteMachineLSEDUT(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("DeleteMachineLSE for a DUT", t, func() {
		Convey("Delete machineLSE DUT with Servo Info", func() {
			labstationMachinelse := mockLabstationMachineLSE("RedLabstation-92")
			inventory.CreateMachineLSE(ctx, labstationMachinelse)

			servo := &chromeosLab.Servo{
				ServoHostname: "RedLabstation-92",
				ServoPort:     92,
			}
			peripherals := &chromeosLab.Peripherals{
				Servo: servo,
			}
			dutMachinelse := mockDutMachineLSE("DUTMachineLse-92")
			dutMachinelse.GetChromeosMachineLse().GetDeviceLse().GetDut().Peripherals = peripherals
			inventory.CreateMachineLSE(ctx, dutMachinelse)

			err := DeleteMachineLSE(ctx, "DUTMachineLse-92")
			So(err, ShouldBeNil)

			resp, _ := inventory.GetMachineLSE(ctx, "RedLabstation-92")
			So(resp.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetServos(), ShouldBeEmpty)
		})
	})
}

func TestDeleteMachineLSELabstation(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("DeleteMachineLSE for a Labstation", t, func() {
		Convey("Delete machineLSE Labstation with Servo Info", func() {
			labstationMachinelse := mockLabstationMachineLSE("RedLabstation-90")
			inventory.CreateMachineLSE(ctx, labstationMachinelse)

			servo := &chromeosLab.Servo{
				ServoHostname: "RedLabstation-90",
				ServoPort:     90,
			}
			peripherals := &chromeosLab.Peripherals{
				Servo: servo,
			}
			dutMachinelse := mockDutMachineLSE("DUTMachineLSE-90")
			dutMachinelse.GetChromeosMachineLse().GetDeviceLse().GetDut().Peripherals = peripherals
			CreateMachineLSE(ctx, dutMachinelse)

			err := DeleteMachineLSE(ctx, "RedLabstation-90")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "cannot be deleted")

			// No changes are recorded as the deletion fails
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "machineLSEs/RedLabstation-90")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})
		Convey("Delete machineLSE Labstation without Servo Info", func() {
			labstationMachinelse := mockLabstationMachineLSE("RedLabstation-100")
			inventory.CreateMachineLSE(ctx, labstationMachinelse)

			err := DeleteMachineLSE(ctx, "RedLabstation-100")
			So(err, ShouldBeNil)

			changes, err := history.QueryChangesByPropertyName(ctx, "name", "machineLSEs/RedLabstation-100")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRetire)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRetire)
			So(changes[0].GetEventLabel(), ShouldEqual, "machine_lse")
		})
	})
}
