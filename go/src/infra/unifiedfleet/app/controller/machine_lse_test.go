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
	chromeosLab "infra/unifiedfleet/api/v1/proto/chromeos/lab"
	"infra/unifiedfleet/app/model/history"
	"infra/unifiedfleet/app/model/inventory"
	"infra/unifiedfleet/app/model/registration"
)

func mockDutMachineLSE(name string) *ufspb.MachineLSE {
	dut := &chromeosLab.DeviceUnderTest{
		Hostname: name,
	}
	device := &ufspb.ChromeOSDeviceLSE_Dut{
		Dut: dut,
	}
	deviceLse := &ufspb.ChromeOSDeviceLSE{
		Device: device,
	}
	chromeosLse := &ufspb.ChromeOSMachineLSE_DeviceLse{
		DeviceLse: deviceLse,
	}
	chromeOSMachineLse := &ufspb.ChromeOSMachineLSE{
		ChromeosLse: chromeosLse,
	}
	lse := &ufspb.MachineLSE_ChromeosMachineLse{
		ChromeosMachineLse: chromeOSMachineLse,
	}
	return &ufspb.MachineLSE{
		Name:     name,
		Hostname: name,
		Lse:      lse,
	}
}

func mockLabstationMachineLSE(name string) *ufspb.MachineLSE {
	labstation := &chromeosLab.Labstation{
		Hostname: name,
	}
	device := &ufspb.ChromeOSDeviceLSE_Labstation{
		Labstation: labstation,
	}
	deviceLse := &ufspb.ChromeOSDeviceLSE{
		Device: device,
	}
	chromeosLse := &ufspb.ChromeOSMachineLSE_DeviceLse{
		DeviceLse: deviceLse,
	}
	chromeOSMachineLse := &ufspb.ChromeOSMachineLSE{
		ChromeosLse: chromeosLse,
	}
	lse := &ufspb.MachineLSE_ChromeosMachineLse{
		ChromeosMachineLse: chromeOSMachineLse,
	}
	return &ufspb.MachineLSE{
		Name:     name,
		Hostname: name,
		Lse:      lse,
	}
}

func TestCreateMachineLSE(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("CreateMachineLSE", t, func() {
		Convey("Create new machineLSE with already existing machinelse", func() {
			machine1 := &ufspb.Machine{
				Name: "machine-10",
			}
			_, merr := registration.CreateMachine(ctx, machine1)
			So(merr, ShouldBeNil)

			_, err := inventory.CreateMachineLSE(ctx, &ufspb.MachineLSE{
				Name: "machinelse-1",
			})
			So(err, ShouldBeNil)

			machineLSE1 := &ufspb.MachineLSE{
				Hostname: "machinelse-1",
			}
			resp, err := CreateMachineLSE(ctx, machineLSE1, []string{"machine-10"})
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "MachineLSE machinelse-1 already exists in the system.")

			// No changes are recorded as the creation fails
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "machineLSEs/machinelse-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})

		Convey("Create new machineLSE with non existing machines", func() {
			machineLSE1 := &ufspb.MachineLSE{
				Hostname: "machinelse-2",
			}
			resp, err := CreateMachineLSE(ctx, machineLSE1, []string{"machine-1", "machine-2"})
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "There is no Machine with MachineID machine-1 in the system.\n"+
				"There is no Machine with MachineID machine-2 in the system.")

			// No changes are recorded as the creation fails
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "machineLSEs/machinelse-2")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})

		Convey("Create new machineLSE with existing machines", func() {
			machine1 := &ufspb.Machine{
				Name: "machine-3",
			}
			_, err := registration.CreateMachine(ctx, machine1)
			So(err, ShouldBeNil)

			machineLSE2 := &ufspb.MachineLSE{
				Hostname: "machinelse-3",
			}
			resp, err := CreateMachineLSE(ctx, machineLSE2, []string{"machine-3"})
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, machineLSE2)

			// No changes are recorded as the creation fails
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "machineLSEs/machinelse-3")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].GetOldValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetNewValue(), ShouldEqual, LifeCycleRegistration)
			So(changes[0].GetEventLabel(), ShouldEqual, "machine_lse")
		})

		Convey("Create new machineLSE with a machine already attached to a different machinelse - error", func() {
			machine1 := &ufspb.Machine{
				Name: "machine-4",
			}
			_, err := registration.CreateMachine(ctx, machine1)
			So(err, ShouldBeNil)

			_, err = inventory.CreateMachineLSE(ctx, &ufspb.MachineLSE{
				Name:     "machinelse-4",
				Machines: []string{"machine-4"},
			})
			So(err, ShouldBeNil)

			machineLSE := &ufspb.MachineLSE{
				Hostname: "machinelse-5",
			}
			resp, err := CreateMachineLSE(ctx, machineLSE, []string{"machine-4"})
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Hosts referring the machine machine-4:\nmachinelse-4")

			// No changes are recorded as the creation fails
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "machineLSEs/machinelse-5")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})
	})
}

func TestCreateMachineLSEDUT(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	servo1 := &chromeosLab.Servo{
		ServoHostname: "BlueLabstation-0",
		ServoPort:     22,
	}
	servo2 := &chromeosLab.Servo{
		ServoHostname: "BlueLabstation-0",
		ServoPort:     33,
	}

	labstationMachinelse := mockLabstationMachineLSE("BlueLabstation-0")
	labstationMachinelse.GetChromeosMachineLse().GetDeviceLse().GetLabstation().Servos = []*chromeosLab.Servo{servo1, servo2}
	inventory.CreateMachineLSE(ctx, labstationMachinelse)

	peripherals1 := &chromeosLab.Peripherals{
		Servo: servo1,
	}
	dutMachinelse1 := mockDutMachineLSE("DUTMachineLSE-1")
	dutMachinelse1.GetChromeosMachineLse().GetDeviceLse().GetDut().Peripherals = peripherals1
	inventory.CreateMachineLSE(ctx, dutMachinelse1)

	peripherals2 := &chromeosLab.Peripherals{
		Servo: servo2,
	}
	dutMachinelse2 := mockDutMachineLSE("DUTMachineLSE-2")
	dutMachinelse2.GetChromeosMachineLse().GetDeviceLse().GetDut().Peripherals = peripherals2
	inventory.CreateMachineLSE(ctx, dutMachinelse2)

	labstationMachinelse, _ = inventory.GetMachineLSE(ctx, "BlueLabstation-0")
	servos := labstationMachinelse.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetServos()
	Convey("CreateMachineLSE for a DUT", t, func() {
		Convey("Create machineLSE DUT without Servo Info", func() {
			machine := &ufspb.Machine{
				Name: "machine-32",
			}
			_, err := registration.CreateMachine(ctx, machine)
			So(err, ShouldBeNil)

			dutMachinelse := mockDutMachineLSE("DUTMachineLSE-10")
			resp, err := CreateMachineLSE(ctx, dutMachinelse, []string{"machine-32"})
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, dutMachinelse)
		})

		Convey("Create existing machineLSE DUT", func() {
			dutMachinelse := mockDutMachineLSE("DUTMachineLSE-11")
			_, err := inventory.CreateMachineLSE(ctx, dutMachinelse)
			So(err, ShouldBeNil)

			machine := &ufspb.Machine{
				Name: "machine-33",
			}
			_, err = registration.CreateMachine(ctx, machine)
			So(err, ShouldBeNil)

			resp, err := CreateMachineLSE(ctx, dutMachinelse, []string{"machine-33"})
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "MachineLSE DUTMachineLSE-11 already exists in the system.")
		})

		Convey("Create machineLSE DUT with distinct Servo Info and existing Labstation", func() {
			machine := &ufspb.Machine{
				Name: "machine-34",
			}
			_, err := registration.CreateMachine(ctx, machine)
			So(err, ShouldBeNil)

			servo := &chromeosLab.Servo{
				ServoHostname: "BlueLabstation-0",
				ServoPort:     55,
			}
			peripherals := &chromeosLab.Peripherals{
				Servo: servo,
			}
			dutMachinelse := mockDutMachineLSE("DUTMachineLSE-12")
			dutMachinelse.GetChromeosMachineLse().GetDeviceLse().GetDut().Peripherals = peripherals
			resp, err := CreateMachineLSE(ctx, dutMachinelse, []string{"machine-34"})
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, dutMachinelse)

			servos = append(servos, servo)
			resp, _ = inventory.GetMachineLSE(ctx, "BlueLabstation-0")
			So(resp.GetChromeosMachineLse().GetDeviceLse().GetLabstation().Servos,
				ShouldResembleProto, servos)
		})

		Convey("Create machineLSE DUT with non-existing Labstation", func() {
			machine := &ufspb.Machine{
				Name: "machine-35",
			}
			_, err := registration.CreateMachine(ctx, machine)
			So(err, ShouldBeNil)

			servo := &chromeosLab.Servo{
				ServoHostname: "BlueLabstation-99",
				ServoPort:     12,
			}
			peripherals := &chromeosLab.Peripherals{
				Servo: servo,
			}
			dutMachinelse := mockDutMachineLSE("DUTmachinelse-13")
			dutMachinelse.GetChromeosMachineLse().GetDeviceLse().GetDut().Peripherals = peripherals
			resp, err := CreateMachineLSE(ctx, dutMachinelse, []string{"machine-35"})
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "not found in the system")
		})

		Convey("Create machineLSE with already assigned ServoHostname and ServoPort", func() {
			machine := &ufspb.Machine{
				Name: "machine-36",
			}
			_, err := registration.CreateMachine(ctx, machine)
			So(err, ShouldBeNil)

			dutMachinelse := mockDutMachineLSE("DUTmachinelse-13")
			dutMachinelse.GetChromeosMachineLse().GetDeviceLse().GetDut().Peripherals = peripherals2
			resp, err := CreateMachineLSE(ctx, dutMachinelse, []string{"machine-36"})
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "already in use")
		})
	})
}

func TestCreateMachineLSELabstation(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	machine := &ufspb.Machine{
		Name: "machine-4",
	}
	registration.CreateMachine(ctx, machine)
	Convey("CreateMachineLSE for a Labstation", t, func() {
		Convey("Create machineLSE Labstation with Servo Info", func() {
			labstationMachinelse1 := mockLabstationMachineLSE("RedLabstation-0")
			servo := &chromeosLab.Servo{
				ServoHostname: "RedLabstation-0",
				ServoPort:     22,
			}
			labstationMachinelse1.GetChromeosMachineLse().GetDeviceLse().GetLabstation().Servos = []*chromeosLab.Servo{servo}
			resp, err := CreateMachineLSE(ctx, labstationMachinelse1, []string{"machine-4"})
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
			resp, err := CreateMachineLSE(ctx, labstationMachinelse1, []string{"machine-4"})
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
	servo1 := &chromeosLab.Servo{
		ServoHostname: "BlueLabstation-10",
		ServoPort:     21,
	}
	servo2 := &chromeosLab.Servo{
		ServoHostname: "BlueLabstation-10",
		ServoPort:     22,
	}

	labstationMachinelse := mockLabstationMachineLSE("BlueLabstation-10")
	labstationMachinelse.GetChromeosMachineLse().GetDeviceLse().GetLabstation().Servos = []*chromeosLab.Servo{servo1, servo2}
	inventory.CreateMachineLSE(ctx, labstationMachinelse)

	peripherals1 := &chromeosLab.Peripherals{
		Servo: servo1,
	}
	dutMachinelse1 := mockDutMachineLSE("DUTMachineLSE-21")
	dutMachinelse1.GetChromeosMachineLse().GetDeviceLse().GetDut().Peripherals = peripherals1
	inventory.CreateMachineLSE(ctx, dutMachinelse1)

	peripherals2 := &chromeosLab.Peripherals{
		Servo: servo2,
	}
	dutMachinelse2 := mockDutMachineLSE("DUTMachineLSE-22")
	dutMachinelse2.GetChromeosMachineLse().GetDeviceLse().GetDut().Peripherals = peripherals2
	inventory.CreateMachineLSE(ctx, dutMachinelse2)

	labstationMachinelse, _ = inventory.GetMachineLSE(ctx, "BlueLabstation-10")
	servos := labstationMachinelse.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetServos()
	Convey("UpdateMachineLSE for a DUT", t, func() {
		Convey("Update non-existing machineLSE DUT", func() {
			dutMachinelse := mockDutMachineLSE("DUTMachineLSE-23")
			resp, err := UpdateMachineLSE(ctx, dutMachinelse, nil)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "There is no MachineLSE with MachineLSEID DUTMachineLSE-23 in the system.")
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
			resp, err := UpdateMachineLSE(ctx, dutMachinelse3, nil)
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
			resp, err := UpdateMachineLSE(ctx, dutMachinelse3, nil)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, dutMachinelse3)

			dummyServos := []*chromeosLab.Servo{servo1, servo3}

			resp, _ = inventory.GetMachineLSE(ctx, "BlueLabstation-10")
			So(resp.GetChromeosMachineLse().GetDeviceLse().GetLabstation().Servos,
				ShouldResembleProto, dummyServos)
		})

		Convey("Update machineLSE DUT with different ServoHostname", func() {
			servo1 := &chromeosLab.Servo{
				ServoHostname: "BlueLabstation-17",
				ServoPort:     17,
			}
			labstationMachinelse1 := mockLabstationMachineLSE("BlueLabstation-17")
			labstationMachinelse.GetChromeosMachineLse().GetDeviceLse().GetLabstation().Servos = []*chromeosLab.Servo{servo1}
			_, err := inventory.CreateMachineLSE(ctx, labstationMachinelse1)
			So(err, ShouldBeNil)

			servo2 := &chromeosLab.Servo{
				ServoHostname: "BlueLabstation-18",
				ServoPort:     18,
			}
			labstationMachinelse2 := mockLabstationMachineLSE("BlueLabstation-18")
			labstationMachinelse.GetChromeosMachineLse().GetDeviceLse().GetLabstation().Servos = []*chromeosLab.Servo{servo2}
			_, err = inventory.CreateMachineLSE(ctx, labstationMachinelse2)
			So(err, ShouldBeNil)

			peripherals1 := &chromeosLab.Peripherals{
				Servo: servo1,
			}
			dutMachinelse1 := mockDutMachineLSE("DUTMachineLSE-17")
			dutMachinelse1.GetChromeosMachineLse().GetDeviceLse().GetDut().Peripherals = peripherals1
			_, err = inventory.CreateMachineLSE(ctx, dutMachinelse1)
			So(err, ShouldBeNil)

			peripherals2 := &chromeosLab.Peripherals{
				Servo: servo2,
			}
			dutMachinelse2 := mockDutMachineLSE("DUTMachineLSE-17")
			dutMachinelse2.GetChromeosMachineLse().GetDeviceLse().GetDut().Peripherals = peripherals2
			resp, err := UpdateMachineLSE(ctx, dutMachinelse2, nil)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, dutMachinelse2)

			resp, _ = inventory.GetMachineLSE(ctx, "BlueLabstation-17")
			So(resp, ShouldNotBeNil)
			servos := resp.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetServos()
			So(servos, ShouldBeEmpty)

			resp, _ = inventory.GetMachineLSE(ctx, "BlueLabstation-18")
			So(resp, ShouldNotBeNil)
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
			resp, err := UpdateMachineLSE(ctx, labstationMachinelse2, nil)
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
			resp, err := UpdateMachineLSE(ctx, labstationMachinelse2, nil)
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
			machine1 := &ufspb.Machine{
				Name: "machine-update1",
			}
			machineLSE1 := &ufspb.MachineLSE{
				Name:     "machinelse-update1",
				Hostname: "machinelse-update1",
				Machines: []string{"machine-update1"},
				Lse: &ufspb.MachineLSE_ChromeBrowserMachineLse{
					ChromeBrowserMachineLse: &ufspb.ChromeBrowserMachineLSE{
						Vms: []*ufspb.VM{
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
			machineLSE1.GetChromeBrowserMachineLse().Vms = []*ufspb.VM{
				{
					Name:       "vm1",
					MacAddress: "new_mac_address",
				},
				{
					Name:       "vm2",
					MacAddress: "vm2_mac_address",
				},
			}
			machineLSE1.GetChromeBrowserMachineLse().OsVersion = &ufspb.OSVersion{
				Value: "new_os",
			}
			m, err := UpdateMachineLSE(ctx, machineLSE1, nil)
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
			_, err := inventory.CreateMachineLSE(ctx, labstationMachinelse1)
			So(err, ShouldBeNil)

			labstationMachinelse2 := mockLabstationMachineLSE("RedLabstation-11")
			resp, err := UpdateMachineLSE(ctx, labstationMachinelse2, nil)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, labstationMachinelse2)

			// No changes happened in this update
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "machineLSEs/RedLabstation-11")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
		})

		Convey("Update a machineLSE with a machine already attached to a different machinelse - error", func() {
			machine := &ufspb.Machine{
				Name: "machine-4",
			}
			_, err := registration.CreateMachine(ctx, machine)
			So(err, ShouldBeNil)

			_, err = inventory.CreateMachineLSE(ctx, &ufspb.MachineLSE{
				Name:     "machinelse-4",
				Machines: []string{"machine-4"},
			})
			So(err, ShouldBeNil)

			machine = &ufspb.Machine{
				Name: "machine-5",
			}
			_, err = registration.CreateMachine(ctx, machine)
			So(err, ShouldBeNil)

			_, err = inventory.CreateMachineLSE(ctx, &ufspb.MachineLSE{
				Name:     "machinelse-5",
				Machines: []string{"machine-5"},
			})
			So(err, ShouldBeNil)

			machineLSE := &ufspb.MachineLSE{
				Hostname: "machinelse-4",
			}
			resp, err := UpdateMachineLSE(ctx, machineLSE, []string{"machine-5"})
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "there is another host machinelse-5 which is referring this machine machine-5")
		})
	})
}

func TestDeleteMachineLSEDUT(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("DeleteMachineLSE for a DUT", t, func() {
		Convey("Delete machineLSE DUT with Servo Info", func() {
			labstationMachinelse := mockLabstationMachineLSE("RedLabstation-92")
			_, err := inventory.CreateMachineLSE(ctx, labstationMachinelse)
			So(err, ShouldBeNil)

			servo := &chromeosLab.Servo{
				ServoHostname: "RedLabstation-92",
				ServoPort:     92,
			}
			peripherals := &chromeosLab.Peripherals{
				Servo: servo,
			}
			dutMachinelse := mockDutMachineLSE("DUTMachineLse-92")
			dutMachinelse.GetChromeosMachineLse().GetDeviceLse().GetDut().Peripherals = peripherals
			_, err = inventory.CreateMachineLSE(ctx, dutMachinelse)
			So(err, ShouldBeNil)

			err = DeleteMachineLSE(ctx, "DUTMachineLse-92")
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
			servo := &chromeosLab.Servo{
				ServoHostname: "RedLabstation-90",
				ServoPort:     90,
			}
			labstationMachinelse := mockLabstationMachineLSE("RedLabstation-90")
			labstationMachinelse.GetChromeosMachineLse().GetDeviceLse().GetLabstation().Servos = []*chromeosLab.Servo{servo}
			_, err := inventory.CreateMachineLSE(ctx, labstationMachinelse)
			So(err, ShouldBeNil)

			peripherals := &chromeosLab.Peripherals{
				Servo: servo,
			}
			dutMachinelse := mockDutMachineLSE("DUTMachineLSE-90")
			dutMachinelse.GetChromeosMachineLse().GetDeviceLse().GetDut().Peripherals = peripherals
			_, err = inventory.CreateMachineLSE(ctx, dutMachinelse)
			So(err, ShouldBeNil)

			err = DeleteMachineLSE(ctx, "RedLabstation-90")
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

func TestListMachineLSEs(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	machineLSEsWithSwitch := make([]*ufspb.MachineLSE, 0, 2)
	machineLSEs := make([]*ufspb.MachineLSE, 0, 4)
	for i := 0; i < 4; i++ {
		machineLSE := mockDutMachineLSE(fmt.Sprintf("machineLSE-%d", i))
		if i%2 == 0 {
			machineLSE.GetChromeosMachineLse().GetDeviceLse().NetworkDeviceInterface = &ufspb.SwitchInterface{Switch: "switch-1"}
		}
		resp, _ := inventory.CreateMachineLSE(ctx, machineLSE)
		if i%2 == 0 {
			machineLSEsWithSwitch = append(machineLSEsWithSwitch, resp)
		}
		machineLSEs = append(machineLSEs, resp)
	}
	Convey("ListMachineLSEs", t, func() {
		Convey("List MachineLSEs - filter invalid - error", func() {
			_, _, err := ListMachineLSEs(ctx, 5, "", "invalid=mx-1", false)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Invalid field name invalid")
		})

		Convey("List MachineLSEs - filter switch - happy path with filter", func() {
			resp, _, _ := ListMachineLSEs(ctx, 5, "", "switch=switch-1", false)
			So(resp, ShouldNotBeNil)
			So(resp, ShouldResembleProto, machineLSEsWithSwitch)
		})

		Convey("ListMachineLSEs - Full listing - happy path", func() {
			resp, _, _ := ListMachineLSEs(ctx, 5, "", "", false)
			So(resp, ShouldNotBeNil)
			So(resp, ShouldResembleProto, machineLSEs)
		})
	})
}
