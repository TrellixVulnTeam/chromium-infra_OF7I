package controller

import (
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
	"google.golang.org/genproto/protobuf/field_mask"

	ufspb "infra/unifiedfleet/api/v1/models"
	device "infra/unifiedfleet/api/v1/models/chromeos/device"
	chromeosLab "infra/unifiedfleet/api/v1/models/chromeos/lab"
	manufacturing "infra/unifiedfleet/api/v1/models/chromeos/manufacturing"
	"infra/unifiedfleet/app/external"
	"infra/unifiedfleet/app/model/history"
	"infra/unifiedfleet/app/model/inventory"
	"infra/unifiedfleet/app/model/registration"
	"infra/unifiedfleet/app/model/state"
)

func mockDUT(hostname, machine, servoHost, servoSerial, rpm, rpmOutlet string, servoPort int32, pools []string) *ufspb.MachineLSE {
	return &ufspb.MachineLSE{
		Name:     hostname,
		Hostname: hostname,
		Machines: []string{machine},
		Lse: &ufspb.MachineLSE_ChromeosMachineLse{
			ChromeosMachineLse: &ufspb.ChromeOSMachineLSE{
				ChromeosLse: &ufspb.ChromeOSMachineLSE_DeviceLse{
					DeviceLse: &ufspb.ChromeOSDeviceLSE{
						Device: &ufspb.ChromeOSDeviceLSE_Dut{
							Dut: &chromeosLab.DeviceUnderTest{
								Hostname: hostname,
								Peripherals: &chromeosLab.Peripherals{
									Servo: &chromeosLab.Servo{
										ServoHostname: servoHost,
										ServoPort:     servoPort,
										ServoSerial:   servoSerial,
									},
									Rpm: &chromeosLab.RPM{
										PowerunitName:   rpm,
										PowerunitOutlet: rpmOutlet,
									},
								},
								Pools: pools,
							},
						},
					},
				},
			},
		},
		// Output only fields. Set defaults for use in comparision.
		Zone:          "ZONE_CHROMEOS6",
		ResourceState: ufspb.State_STATE_REGISTERED,
	}
}

func mockLabstation(hostname, machine string) *ufspb.MachineLSE {
	return &ufspb.MachineLSE{
		Name:     hostname,
		Hostname: hostname,
		Machines: []string{machine},
		Lse: &ufspb.MachineLSE_ChromeosMachineLse{
			ChromeosMachineLse: &ufspb.ChromeOSMachineLSE{
				ChromeosLse: &ufspb.ChromeOSMachineLSE_DeviceLse{
					DeviceLse: &ufspb.ChromeOSDeviceLSE{
						Device: &ufspb.ChromeOSDeviceLSE_Labstation{
							Labstation: &chromeosLab.Labstation{
								Hostname: hostname,
							},
						},
					},
				},
			},
		},
	}
}

func mockFieldMask(opts ...string) *field_mask.FieldMask {
	return &field_mask.FieldMask{
		Paths: opts,
	}
}

func createValidDUTWithLabstation(ctx context.Context, dutName, dutMachine, labstationName, labstationMachine string) {
	machine1 := &ufspb.Machine{
		Name: labstationMachine,
		Location: &ufspb.Location{
			Zone: ufspb.Zone_ZONE_CHROMEOS6,
		},
		Device: &ufspb.Machine_ChromeosMachine{
			ChromeosMachine: &ufspb.ChromeOSMachine{
				BuildTarget: "test",
				Model:       "test",
			},
		},
	}
	machine2 := &ufspb.Machine{
		Name: dutMachine,
		Location: &ufspb.Location{
			Zone: ufspb.Zone_ZONE_CHROMEOS6,
		},
		Device: &ufspb.Machine_ChromeosMachine{
			ChromeosMachine: &ufspb.ChromeOSMachine{
				BuildTarget: "test",
				Model:       "test",
			},
		},
	}
	_, err := registration.CreateMachine(ctx, machine1)
	So(err, ShouldBeNil)
	_, err = registration.CreateMachine(ctx, machine2)
	So(err, ShouldBeNil)
	labstation1 := mockLabstation(labstationName, labstationMachine)
	_, err = CreateLabstation(ctx, labstation1)
	So(err, ShouldBeNil)
	dut1 := mockDUT(dutName, dutMachine, labstationName, "serial-1", "power-1", ".A1", int32(9999), []string{"DUT_POOL_QUOTA"})
	_, err = CreateDUT(ctx, dut1)
	So(err, ShouldBeNil)
	changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/"+dutName)
	So(err, ShouldBeNil)
	So(changes, ShouldHaveLength, 1)
	msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/"+dutName)
	So(err, ShouldBeNil)
	So(msgs, ShouldHaveLength, 1)
	So(msgs[0].Delete, ShouldBeFalse)
}

func TestCreateDUT(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	ctx = external.WithTestingContext(ctx)
	Convey("CreateDUT", t, func() {
		Convey("CreateDUT - With non-existent Labstation", func() {
			machine1 := &ufspb.Machine{
				Name: "machine-10",
				Device: &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: &ufspb.ChromeOSMachine{
						BuildTarget: "test",
						Model:       "test",
					},
				},
			}
			_, err := registration.CreateMachine(ctx, machine1)
			So(err, ShouldBeNil)
			dut1 := mockDUT("dut-1", "machine-10", "labstation-1", "serial-1", "power-1", ".A1", int32(9999), []string{"DUT_POOL_QUOTA"})
			_, err = CreateDUT(ctx, dut1)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "labstation-1 not found in the system")
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-1")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 0)
		})
		Convey("CreateDUT - With non-existent device config", func() {
			machine1 := &ufspb.Machine{
				Name: "machine-20",
				Device: &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: &ufspb.ChromeOSMachine{
						BuildTarget: "test",
						Model:       "test",
					},
				},
			}
			machine2 := &ufspb.Machine{
				Name: "machine-21",
				Device: &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: &ufspb.ChromeOSMachine{
						BuildTarget: "buildtest",
						Model:       "modeltest",
					},
				},
			}
			_, err := registration.CreateMachine(ctx, machine1)
			So(err, ShouldBeNil)
			_, err = registration.CreateMachine(ctx, machine2)
			So(err, ShouldBeNil)
			labstation1 := mockLabstation("labstation-1", "machine-20")
			_, err = CreateLabstation(ctx, labstation1)
			So(err, ShouldBeNil)
			dut1 := mockDUT("dut-2", "machine-21", "labstation-1", "serial-1", "power-1", ".A1", int32(9999), []string{"DUT_POOL_QUOTA"})
			_, err = CreateDUT(ctx, dut1)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Device config doesn't exist")
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-2")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-2")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 0)
		})
		Convey("CreateDUT - With port conflict on labstation", func() {
			machine1 := &ufspb.Machine{
				Name: "machine-30",
				Device: &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: &ufspb.ChromeOSMachine{
						BuildTarget: "test",
						Model:       "test",
					},
				},
			}
			machine2 := &ufspb.Machine{
				Name: "machine-40",
				Device: &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: &ufspb.ChromeOSMachine{
						BuildTarget: "test",
						Model:       "test",
					},
				},
			}
			machine3 := &ufspb.Machine{
				Name: "machine-50",
				Device: &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: &ufspb.ChromeOSMachine{
						BuildTarget: "test",
						Model:       "test",
					},
				},
			}
			_, err := registration.CreateMachine(ctx, machine1)
			So(err, ShouldBeNil)
			_, err = registration.CreateMachine(ctx, machine2)
			So(err, ShouldBeNil)
			_, err = registration.CreateMachine(ctx, machine3)
			So(err, ShouldBeNil)
			labstation1 := mockLabstation("labstation-3", "machine-30")
			_, err = CreateLabstation(ctx, labstation1)
			So(err, ShouldBeNil)
			dut1 := mockDUT("dut-3", "machine-40", "labstation-3", "serial-2", "power-1", ".A1", int32(9999), []string{"DUT_POOL_QUOTA"})
			_, err = CreateDUT(ctx, dut1)
			So(err, ShouldBeNil)
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-3")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-3")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			So(msgs[0].Delete, ShouldBeFalse)
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "hosts/labstation-3")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			dut2 := mockDUT("dut-4", "machine-50", "labstation-3", "serial-3", "power-1", ".A2", int32(9999), []string{"DUT_POOL_QUOTA"})
			_, err = CreateDUT(ctx, dut2)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Port: 9999 in labstation-3 is already in use by dut-3")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-4")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-4")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 0)
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "hosts/labstation-3")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
		})
		Convey("CreateDUT - Happy Path", func() {
			machine1 := &ufspb.Machine{
				Name: "machine-90",
				Device: &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: &ufspb.ChromeOSMachine{
						BuildTarget: "test",
						Model:       "test",
					},
				},
			}
			machine2 := &ufspb.Machine{
				Name: "machine-00",
				Device: &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: &ufspb.ChromeOSMachine{
						BuildTarget: "test",
						Model:       "test",
					},
				},
			}
			_, merr := registration.CreateMachine(ctx, machine1)
			So(merr, ShouldBeNil)
			_, merr = registration.CreateMachine(ctx, machine2)
			So(merr, ShouldBeNil)
			labstation1 := mockLabstation("labstation-5", "machine-90")
			_, err := CreateLabstation(ctx, labstation1)
			So(err, ShouldBeNil)
			dut1 := mockDUT("dut-7", "machine-00", "labstation-5", "serial-1", "power-3", ".A1", int32(9999), []string{"DUT_POOL_QUOTA"})
			_, err = CreateDUT(ctx, dut1)
			So(err, ShouldBeNil)
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-7")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-7")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			So(msgs[0].Delete, ShouldBeFalse)
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "hosts/labstation-5")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
		})
		Convey("CreateDUT - Existing DUT", func() {
			machine1 := &ufspb.Machine{
				Name: "machine-01",
				Device: &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: &ufspb.ChromeOSMachine{
						BuildTarget: "test",
						Model:       "test",
					},
				},
			}
			machine2 := &ufspb.Machine{
				Name: "machine-02",
				Device: &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: &ufspb.ChromeOSMachine{
						BuildTarget: "test",
						Model:       "test",
					},
				},
			}
			_, merr := registration.CreateMachine(ctx, machine1)
			So(merr, ShouldBeNil)
			_, merr = registration.CreateMachine(ctx, machine2)
			So(merr, ShouldBeNil)
			labstation1 := mockLabstation("labstation-6", "machine-01")
			_, err := CreateLabstation(ctx, labstation1)
			So(err, ShouldBeNil)
			dut1 := mockDUT("dut-8", "machine-02", "labstation-6", "serial-1", "power-3", ".A1", int32(9999), []string{"DUT_POOL_QUOTA"})
			_, err = CreateDUT(ctx, dut1)
			So(err, ShouldBeNil)
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-8")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-8")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			So(msgs[0].Delete, ShouldBeFalse)
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "hosts/labstation-6")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			_, err = CreateDUT(ctx, dut1)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "MachineLSE dut-8 already exists in the system")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-8")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-8")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			So(msgs[0].Delete, ShouldBeFalse)
		})
		Convey("CreateDUT - Existing machine", func() {
			machine1 := &ufspb.Machine{
				Name: "machine-03",
				Device: &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: &ufspb.ChromeOSMachine{
						BuildTarget: "test",
						Model:       "test",
					},
				},
			}
			_, merr := registration.CreateMachine(ctx, machine1)
			So(merr, ShouldBeNil)
			labstation1 := mockLabstation("labstation-7", "machine-03")
			_, err := CreateLabstation(ctx, labstation1)
			So(err, ShouldBeNil)
			dut1 := mockDUT("dut-9", "machine-03", "labstation-7", "serial-1", "power-3", ".A1", int32(9999), []string{"DUT_POOL_QUOTA"})
			_, err = CreateDUT(ctx, dut1)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Host dut-9 cannot be created because there are other hosts which are referring this machine machine-03")
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-9")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-9")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 0)
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "hosts/labstation-7")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
		})
	})
}

func TestUpdateDUT(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	ctx = external.WithTestingContext(ctx)
	Convey("UpdateDUT", t, func() {

		Convey("UpdateDUT - With non-existent dut", func() {
			// Create a DUT with labstation. Also creates servo with port: 9999 and serial: serial-1
			dut1 := mockDUT("dut-1", "machine-10", "labstation-1", "serial-1", "power-1", ".A1", int32(9999), []string{"DUT_POOL_QUOTA"})
			// dut-1 doesn't exist. Should fail.
			_, err := UpdateDUT(ctx, dut1, nil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Failed to get existing MachineLSE")
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-1")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 0)
		})

		Convey("UpdateDUT - With non-existent machine", func() {
			// Create a DUT with labstation. Also creates servo with port: 9999 and serial: serial-1
			createValidDUTWithLabstation(ctx, "dut-1", "machine-20", "labstation-1", "machine-10")
			// Update DUT machine to a non existent one. This should fail.
			dut1 := mockDUT("dut-1", "machine-20-fake", "labstation-1", "serial-1", "power-1", ".A1", int32(9999), []string{"DUT_POOL_QUOTA"})
			_, err := UpdateDUT(ctx, dut1, nil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "There is no Machine with MachineID machine-20-fake in the system")
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-1")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			s, err := state.GetStateRecord(ctx, "hosts/dut-1")
			So(err, ShouldBeNil)
			// State should be set to registered. No change.
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_REGISTERED)
		})

		Convey("UpdateDUT - With existing deployed machine", func() {
			// Create a DUT with labstation. Also creates servo with port: 9999 and serial: serial-1
			createValidDUTWithLabstation(ctx, "dut-2", "machine-40", "labstation-2", "machine-30")
			// Update DUT machine to labstations machine (machine-30). Should fail.
			dut1 := mockDUT("dut-2", "machine-30", "labstation-2", "serial-1", "power-1", ".A1", int32(9999), []string{"DUT_POOL_QUOTA"})
			_, err := UpdateDUT(ctx, dut1, nil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Host dut-2 cannot be updated because there is another host labstation-2 which is referring this machine machine-30")
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-2")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-2")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			s, err := state.GetStateRecord(ctx, "hosts/dut-2")
			So(err, ShouldBeNil)
			// State should be set to registered. No change.
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_REGISTERED)
		})

		Convey("UpdateDUT - With invalid name mask", func() {
			createValidDUTWithLabstation(ctx, "dut-3-name", "machine-60-name", "labstation-3-name", "machine-50-name")
			// Update with name mask.
			dut1 := mockDUT("dut-3-name", "", "", "", "", "", int32(0), nil)
			_, err := UpdateDUT(ctx, dut1, mockFieldMask("name"))
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "name cannot be updated")
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-3-name")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-3-name")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			s, err := state.GetStateRecord(ctx, "hosts/dut-3-name")
			So(err, ShouldBeNil)
			// State should be set to registered. No change.
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_REGISTERED)
		})

		Convey("UpdateDUT - With invalid update_time mask", func() {
			// Create a DUT with labstation. Also creates servo with port: 9999 and serial: serial-1
			createValidDUTWithLabstation(ctx, "dut-3-time", "machine-60-time", "labstation-3-time", "machine-50-time")
			// Update with update_time mask.
			dut1 := mockDUT("dut-3-time", "machine-60-time", "labstation-3-time", "serial-1", "power-1", ".A1", int32(9999), []string{"DUT_POOL_QUOTA"})
			_, err := UpdateDUT(ctx, dut1, mockFieldMask("update-time"))
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "unsupported update mask path")
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-3-time")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-3-time")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			s, err := state.GetStateRecord(ctx, "hosts/dut-3-time")
			So(err, ShouldBeNil)
			// State should be set to registered. No change.
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_REGISTERED)
		})

		Convey("UpdateDUT - With invalid machine mask", func() {
			// Create a DUT with labstation. Also creates servo with port: 9999 and serial: serial-1
			createValidDUTWithLabstation(ctx, "dut-3-machine", "machine-60-machine", "labstation-3-machine", "machine-50-machine")
			// Update with machine mask and no machines.
			dut1 := mockDUT("dut-3-machine", "", "labstation-3-machine", "serial-1", "power-1", ".A1", int32(9999), []string{"DUT_POOL_QUOTA"})
			_, err := UpdateDUT(ctx, dut1, mockFieldMask("machines"))
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "machines field cannot be empty/nil")
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-3-machine")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-3-machine")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			s, err := state.GetStateRecord(ctx, "hosts/dut-3-machine")
			So(err, ShouldBeNil)
			// State should be set to registered. No change.
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_REGISTERED)
		})

		Convey("UpdateDUT - With invalid dut hostname mask", func() {
			// Create a DUT with labstation. Also creates servo with port: 9999 and serial: serial-1
			createValidDUTWithLabstation(ctx, "dut-3-hostname", "machine-60-hostname", "labstation-3-hostname", "machine-50-hostname")
			// Update with dut hostname mask.
			dut1 := mockDUT("dut-3-hostname", "machine-60-hostname", "labstation-3-hostname", "serial-1", "power-1", ".A1", int32(9999), []string{"DUT_POOL_QUOTA"})
			_, err := UpdateDUT(ctx, dut1, mockFieldMask("dut.hostname"))
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "hostname cannot be updated")
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-3-hostname")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-3-hostname")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			s, err := state.GetStateRecord(ctx, "hosts/dut-3-hostname")
			So(err, ShouldBeNil)
			// State should be set to registered. No change.
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_REGISTERED)
		})

		Convey("UpdateDUT - With valid dut pools mask", func() {
			// Create a DUT with labstation. Also creates servo with port: 9999 and serial: serial-1
			createValidDUTWithLabstation(ctx, "dut-3-pools", "machine-60-pools", "labstation-3-pools", "machine-50-pools")
			// Update with dut pools mask and valid pools.
			dut1 := mockDUT("dut-3-pools", "machine-60-pools", "labstation-3-pools", "serial-1", "power-1", ".A1", int32(9999), []string{"DUT_POOL_QUOTA", "DUT_POOL_CQ"})
			resp, err := UpdateDUT(ctx, dut1, mockFieldMask("dut.pools"))
			So(err, ShouldBeNil)
			// Clear update time to compare the protos
			resp.UpdateTime = nil
			So(resp, ShouldResembleProto, dut1)
			dut2, err := GetMachineLSE(ctx, "dut-3-pools")
			So(err, ShouldBeNil)
			// Clear update time to compare the protos
			dut2.UpdateTime = nil
			So(dut2, ShouldResembleProto, dut1)
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-3-pools")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 2)
			So(changes[1].OldValue, ShouldEqual, "[DUT_POOL_QUOTA]")
			So(changes[1].NewValue, ShouldEqual, "[DUT_POOL_QUOTA DUT_POOL_CQ]")
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-3-pools")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 2)
			So(msgs[0].Delete, ShouldBeFalse)
			So(msgs[1].Delete, ShouldBeFalse)
			s, err := state.GetStateRecord(ctx, "hosts/dut-3-pools")
			So(err, ShouldBeNil)
			// State should be set to registered. No change.
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_REGISTERED)
		})

		Convey("UpdateDUT - With invalid servo host mask (delete host and update port)", func() {
			// Create a DUT with labstation. Also creates servo with port: 9999 and serial: serial-1
			createValidDUTWithLabstation(ctx, "dut-3-non-v3-host", "machine-60-non-v3-host", "labstation-3-non-v3-host", "machine-50-non-v3-host")
			// Update with servo host mask and no servo host.
			dut1 := mockDUT("dut-3-non-v3-host", "machine-60-non-v3-host", "", "", "", "", int32(9999), nil)
			_, err := UpdateDUT(ctx, dut1, mockFieldMask("dut.servo.hostname", "dut.servo.port"))
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Cannot update servo port. Servo host is being reset.")
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-3-non-v3-host")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-3-non-v3-host")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			// Verify that labstation-3-non-v3-host wasn't changed after last update.
			ls9, err := GetMachineLSE(ctx, "labstation-3-non-v3-host")
			So(err, ShouldBeNil)
			// Verify that nothing was changed on labstation.
			So(ls9.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetServos(), ShouldResembleProto, []*chromeosLab.Servo{
				// Servo generated by createValidDUTWithLabstation.
				{
					ServoHostname: "labstation-3-non-v3-host",
					ServoPort:     int32(9999),
					ServoSerial:   "serial-1",
				},
			})
			s, err := state.GetStateRecord(ctx, "hosts/dut-3-non-v3-host")
			So(err, ShouldBeNil)
			// State should be set to registered. No change.
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_REGISTERED)
		})

		Convey("UpdateDUT - With invalid servo host mask (delete host and update serial)", func() {
			// Create a DUT with labstation. Also creates servo with port: 9999 and serial: serial-1
			createValidDUTWithLabstation(ctx, "dut-3-non-v3-serial", "machine-60-non-v3-serial", "labstation-3-non-v3-serial", "machine-50-non-v3-serial")
			// Update with servo host mask and no servo host.
			dut1 := mockDUT("dut-3-non-v3-serial", "machine-60-non-v3-serial", "", "serial-2", "", "", int32(0), nil)
			_, err := UpdateDUT(ctx, dut1, mockFieldMask("dut.servo.hostname", "dut.servo.serial"))
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Cannot update servo serial. Servo host is being reset")
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-3-non-v3-serial")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-3-non-v3-serial")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			// Verify that labstation-3-non-v3-serial wasn't changed after last update.
			ls9, err := GetMachineLSE(ctx, "labstation-3-non-v3-serial")
			So(err, ShouldBeNil)
			// Verify that nothing was changed on labstation.
			So(ls9.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetServos(), ShouldResembleProto, []*chromeosLab.Servo{
				// Servo generated by createValidDUTWithLabstation.
				{
					ServoHostname: "labstation-3-non-v3-serial",
					ServoPort:     int32(9999),
					ServoSerial:   "serial-1",
				},
			})
			s, err := state.GetStateRecord(ctx, "hosts/dut-3-non-v3-serial")
			So(err, ShouldBeNil)
			// State should be set to registered. No change.
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_REGISTERED)
		})

		Convey("UpdateDUT - Change servo V3 port and serial", func() {
			// Servo V3 enforces port 9999 and  doesn't store a serial. Therefore, this use case is essentially no-op.
			// Servo V3 labstations have hostnames with suffix '-servo'.

			// Create a DUT with labstation. Also creates servo with port: 9999 and serial: serial-1
			createValidDUTWithLabstation(ctx, "dut-3-servo", "machine-60-servo", "labstation-3-servo", "machine-50-servo")
			// Update with servo host mask and no servo host.
			dut1 := mockDUT("dut-3-servo", "machine-60-servo", "labstation-3-servo", "serial-2", "power-1", ".A1", int32(9988), []string{"DUT_POOL_QUOTA"})
			resp, err := UpdateDUT(ctx, dut1, mockFieldMask("dut.servo.port", "dut.servo.serial"))
			So(err, ShouldBeNil)
			// Clear update time to compare the protos.
			resp.UpdateTime = nil
			// ServoPort = 9999; ServoSerial = "" enforced for servo V3.
			dut1.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetServo().ServoPort = int32(9999)
			dut1.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetServo().ServoSerial = ""
			So(resp, ShouldResembleProto, dut1)
			dut2, err := GetMachineLSE(ctx, "dut-3-servo")
			So(err, ShouldBeNil)
			// Clear update time to compare the protos.
			dut2.UpdateTime = nil
			So(dut2, ShouldResembleProto, dut1)
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-3-servo")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "hosts/labstation-3-servo")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			// No change recorded in the labstation record.
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-3-servo")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 2)
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/labstation-3-servo")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 2)
			// Verify that labstation-3-servo hasn't changed.
			ls9, err := GetMachineLSE(ctx, "labstation-3-servo")
			So(err, ShouldBeNil)
			So(ls9.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetServos(), ShouldHaveLength, 1)
			// Verify that the servo was included in the new labstation.
			So(ls9.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetServos(), ShouldResembleProto, []*chromeosLab.Servo{
				{
					ServoHostname: "labstation-3-servo",
					ServoPort:     int32(9999),
				},
			})
			s, err := state.GetStateRecord(ctx, "hosts/dut-3-servo")
			So(err, ShouldBeNil)
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_REGISTERED)
		})

		Convey("UpdateDUT - With valid servo serial mask", func() {
			// Create a DUT with labstation. Also creates servo with port: 9999 and serial: serial-1
			createValidDUTWithLabstation(ctx, "dut-3-serial", "machine-60-serial", "labstation-3-serial", "machine-50-serial")
			// Update with servo host mask and no servo host.
			dut1 := mockDUT("dut-3-serial", "machine-60-serial", "labstation-3-serial", "serial-2", "power-1", ".A1", int32(9999), []string{"DUT_POOL_QUOTA"})
			resp, err := UpdateDUT(ctx, dut1, mockFieldMask("dut.servo.serial"))
			So(err, ShouldBeNil)
			// Clear update time to compare the protos
			resp.UpdateTime = nil
			So(resp, ShouldResembleProto, dut1)
			dut2, err := GetMachineLSE(ctx, "dut-3-serial")
			So(err, ShouldBeNil)
			// Clear update time to compare the protos
			dut2.UpdateTime = nil
			So(dut2, ShouldResembleProto, dut1)
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-3-serial")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 2)
			So(changes[1].OldValue, ShouldEqual, "serial-1")
			So(changes[1].NewValue, ShouldEqual, "serial-2")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "hosts/labstation-3-serial")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 2)
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-3-serial")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 2)
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/labstation-3-serial")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 2)
			// Verify that labstation-3-serial has updated servo.
			ls9, err := GetMachineLSE(ctx, "labstation-3-serial")
			So(err, ShouldBeNil)
			// Verify that the servo was included in the new labstation.
			So(ls9.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetServos(), ShouldResembleProto, []*chromeosLab.Servo{
				{
					ServoHostname: "labstation-3-serial",
					ServoPort:     int32(9999),
					ServoSerial:   "serial-2",
				},
			})
			s, err := state.GetStateRecord(ctx, "hosts/dut-3-serial")
			So(err, ShouldBeNil)
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_REGISTERED)
		})

		Convey("UpdateDUT - With valid servo port mask", func() {
			// Create a DUT with labstation. Also creates servo with port: 9999 and serial: serial-1
			createValidDUTWithLabstation(ctx, "dut-3-port", "machine-60-port", "labstation-3-port", "machine-50-port")
			// Update with servo port mask to port 9988.
			dut1 := mockDUT("dut-3-port", "machine-60-port", "labstation-3-port", "serial-1", "power-1", ".A1", int32(9988), []string{"DUT_POOL_QUOTA"})
			_, err := UpdateDUT(ctx, dut1, mockFieldMask("dut.servo.port"))
			So(err, ShouldBeNil)
			dut2, err := GetMachineLSE(ctx, "dut-3-port")
			So(err, ShouldBeNil)
			// Clear update time to compare the protos
			dut2.UpdateTime = nil
			So(dut2, ShouldResembleProto, dut1)
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-3-port")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 2)
			So(changes[1].OldValue, ShouldEqual, "9999")
			So(changes[1].NewValue, ShouldEqual, "9988")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "hosts/labstation-3-port")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 2)
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-3-port")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 2)
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/labstation-3-port")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 2)
			// Verify that labstation-3-port has updated servo.
			ls9, err := GetMachineLSE(ctx, "labstation-3-port")
			So(err, ShouldBeNil)
			// Verify that the moved servo was included in the new labstation.
			So(ls9.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetServos(), ShouldResembleProto, []*chromeosLab.Servo{
				// Servo generated by createValidDUTWithLabstation.
				{
					ServoHostname: "labstation-3-port",
					ServoPort:     int32(9988),
					ServoSerial:   "serial-1",
				},
			})
			s, err := state.GetStateRecord(ctx, "hosts/dut-3-port")
			So(err, ShouldBeNil)
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_REGISTERED)
		})

		Convey("UpdateDUT - With servo port out of range", func() {
			// Create a DUT with labstation. Also creates servo with port: 9999 and serial: serial-1
			createValidDUTWithLabstation(ctx, "dut-3-out-of-range-port", "machine-60-out-of-range-port", "labstation-3-out-of-range-port", "machine-50-out-of-range-port")
			// Update with servo host mask and no servo host.
			dut1 := mockDUT("dut-3-out-of-range-port", "", "", "", "", "", int32(1111), nil)
			_, err := UpdateDUT(ctx, dut1, mockFieldMask("dut.servo.port"))
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Servo port 1111 is invalid")
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-3-out-of-range-port")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-3-out-of-range-port")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			// Verify that labstation-3-out-of-range-port-servo has no servos.
			ls9, err := GetMachineLSE(ctx, "labstation-3-out-of-range-port")
			So(err, ShouldBeNil)
			// Verify that the servo port wasn't updated on labstation.
			So(ls9.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetServos(), ShouldHaveLength, 1)
			So(ls9.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetServos()[0].GetServoPort(), ShouldEqual, int32(9999))
			s, err := state.GetStateRecord(ctx, "hosts/dut-3-out-of-range-port")
			So(err, ShouldBeNil)
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_REGISTERED)
		})

		Convey("UpdateDUT - Remove servo port and auto assign", func() {
			// Create a DUT with labstation. Also creates servo with port: 9999 and serial: serial-1
			createValidDUTWithLabstation(ctx, "dut-3-port-auto-assign", "machine-60-port-auto-assign", "labstation-3-port-auto-assign", "machine-50-port-auto-assign")
			// Update with servo host mask and no servo host.
			dut1 := mockDUT("dut-3-port-auto-assign", "machine-60-port-auto-assign", "labstation-3-port-auto-assign", "serial-1", "power-1", ".A1", int32(9001), []string{"DUT_POOL_QUOTA"})
			// Change servo port to 9001.
			resp, err := UpdateDUT(ctx, dut1, mockFieldMask("dut.servo.port"))
			So(err, ShouldBeNil)
			// Clear update time to compare the protos
			resp.UpdateTime = nil
			So(resp, ShouldResembleProto, dut1)
			dut2, err := GetMachineLSE(ctx, "dut-3-port-auto-assign")
			So(err, ShouldBeNil)
			// Clear update time to compare the protos
			dut2.UpdateTime = nil
			So(dut2, ShouldResembleProto, dut1)
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-3-port-auto-assign")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 2)
			So(changes[1].OldValue, ShouldEqual, "9999")
			So(changes[1].NewValue, ShouldEqual, "9001")

			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-3-port-auto-assign")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 2)
			// Verify that labstation-3-port-auto-assign-servo has correct servo.
			ls9, err := GetMachineLSE(ctx, "labstation-3-port-auto-assign")
			So(err, ShouldBeNil)
			// Verify that the servo was updated in labstation.
			So(ls9.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetServos(), ShouldHaveLength, 1)
			So(ls9.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetServos()[0].GetServoPort(), ShouldEqual, int32(9001))
			s, err := state.GetStateRecord(ctx, "hosts/dut-3-port-auto-assign")
			So(err, ShouldBeNil)
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_REGISTERED)

			// No port given. Will get 9999 auto assigned to the servo.
			dut1.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetServo().ServoPort = int32(0)
			// Change servo port to 9999.
			resp, err = UpdateDUT(ctx, dut1, mockFieldMask("dut.servo.port"))
			So(err, ShouldBeNil)
			// Clear update time to compare the protos
			resp.UpdateTime = nil
			// Update servo port to 9999 as it was autoassigned.
			dut1.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetServo().ServoPort = int32(9999)
			So(resp, ShouldResembleProto, dut1)
			dut2, err = GetMachineLSE(ctx, "dut-3-port-auto-assign")
			So(err, ShouldBeNil)
			// Clear update time to compare the protos
			dut2.UpdateTime = nil
			So(dut2, ShouldResembleProto, dut1)
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-3-port-auto-assign")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 3)
			So(changes[2].OldValue, ShouldEqual, "9001")
			So(changes[2].NewValue, ShouldEqual, "9999")

			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-3-port-auto-assign")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 3)
			// Verify that labstation-3-port-auto-assign-servo has correct servo.
			ls9, err = GetMachineLSE(ctx, "labstation-3-port-auto-assign")
			So(err, ShouldBeNil)
			// Verify that the servo was updated in labstation.
			So(ls9.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetServos(), ShouldHaveLength, 1)
			So(ls9.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetServos()[0].GetServoPort(), ShouldEqual, int32(9999))
			s, err = state.GetStateRecord(ctx, "hosts/dut-3-port-auto-assign")
			So(err, ShouldBeNil)
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_REGISTERED)
		})

		Convey("UpdateDUT - With valid servo mask (delete servo)", func() {
			// Create a DUT with labstation. Also creates servo with port: 9999 and serial: serial-1
			createValidDUTWithLabstation(ctx, "dut-3-del", "machine-60-del", "labstation-3-del", "machine-50-del")
			// Update with servo host mask and no servo host.
			dut1 := mockDUT("dut-3-del", "machine-60-del", "", "", "power-1", ".A1", int32(0), []string{"DUT_POOL_QUOTA"})
			// Remove servo from DUT.
			dut1.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().Servo = nil
			resp, err := UpdateDUT(ctx, dut1, mockFieldMask("dut.servo.hostname"))
			So(err, ShouldBeNil)
			// Clear update time to compare the protos
			resp.UpdateTime = nil
			So(resp, ShouldResembleProto, dut1)
			dut2, err := GetMachineLSE(ctx, "dut-3-del")
			So(err, ShouldBeNil)
			// Clear update time to compare the protos
			dut2.UpdateTime = nil
			So(dut2, ShouldResembleProto, dut1)
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-3-del")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 4)
			So(changes[1].OldValue, ShouldEqual, "labstation-3-del")
			So(changes[1].NewValue, ShouldEqual, "")
			So(changes[2].OldValue, ShouldEqual, "9999")
			So(changes[2].NewValue, ShouldEqual, "0")
			So(changes[3].OldValue, ShouldEqual, "serial-1")
			So(changes[3].NewValue, ShouldEqual, "")

			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-3-del")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 2)
			// Verify that labstation-3-del-servo has no servos.
			ls9, err := GetMachineLSE(ctx, "labstation-3-del")
			So(err, ShouldBeNil)
			// Verify that the servo was deleted in labstation.
			So(ls9.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetServos(), ShouldHaveLength, 0)
			s, err := state.GetStateRecord(ctx, "hosts/dut-3-del")
			So(err, ShouldBeNil)
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_REGISTERED)
		})

		Convey("UpdateDUT - With valid servo mask (delete servo) [Servo V3]", func() {
			// Create a DUT with labstation. Also creates servo with port: 9999 and serial: serial-1
			createValidDUTWithLabstation(ctx, "dut-3-del-servo", "machine-60-del-servo", "labstation-3-del-servo", "machine-50-del-servo")
			// Update with servo host mask and no servo host.
			dut1 := mockDUT("dut-3-del-servo", "machine-60-del-servo", "", "", "power-1", ".A1", int32(0), []string{"DUT_POOL_QUOTA"})
			// Remove servo from DUT.
			dut1.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().Servo = nil
			resp, err := UpdateDUT(ctx, dut1, mockFieldMask("dut.servo.hostname"))
			So(err, ShouldBeNil)
			// Clear update time to compare the protos
			resp.UpdateTime = nil
			// Proto compare as dut1 doesn't contain servo and delete is successful.
			So(resp, ShouldResembleProto, dut1)
			dut2, err := GetMachineLSE(ctx, "dut-3-del-servo")
			So(err, ShouldBeNil)
			// Clear update time to compare the protos
			dut2.UpdateTime = nil
			So(dut2, ShouldResembleProto, dut1)
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-3-del-servo")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 3)
			So(changes[1].OldValue, ShouldEqual, "labstation-3-del-servo")
			So(changes[1].NewValue, ShouldEqual, "")
			So(changes[2].OldValue, ShouldEqual, "9999")
			So(changes[2].NewValue, ShouldEqual, "0")
			// No change in servo serial recorded for servo V3.
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-3-del-servo")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 2)
			s, err := state.GetStateRecord(ctx, "hosts/dut-3-del-servo")
			So(err, ShouldBeNil)
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_REGISTERED)
		})

		Convey("UpdateDUT - With invalid rpm host mask", func() {
			// Create a DUT with labstation. Also creates servo with port: 9999 and serial: serial-1
			createValidDUTWithLabstation(ctx, "dut-3-rpm-host", "machine-60-rpm-host", "labstation-3-rpm-host", "machine-50-rpm-host")
			// Update with rpm host mask and no rpm host.
			dut1 := mockDUT("dut-3-rpm-host", "machine-60-rpm-host", "labstation-3-rpm-host", "serial-1", "", ".A1", int32(9999), []string{"DUT_POOL_QUOTA"})
			_, err := UpdateDUT(ctx, dut1, mockFieldMask("dut.rpm.host", "dut.rpm.outlet"))
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Deleting rpm host deletes everything. Cannot update outlet.")
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-3-rpm-host")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-3-rpm-host")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			s, err := state.GetStateRecord(ctx, "hosts/dut-3-rpm-host")
			So(err, ShouldBeNil)
			// State should be set to registered.
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_REGISTERED)
		})

		Convey("UpdateDUT - With invalid rpm outlet mask", func() {
			// Create a DUT with labstation. Also creates servo with port: 9999 and serial: serial-1
			createValidDUTWithLabstation(ctx, "dut-3-rpm-outlet", "machine-60-rpm-outlet", "labstation-3-rpm-outlet", "machine-50-rpm-outlet")
			// Update with rpm outlet mask and no rpm outlet.
			dut1 := mockDUT("dut-3-rpm-outlet", "", "", "", "", "", int32(0), nil)
			_, err := UpdateDUT(ctx, dut1, mockFieldMask("dut.rpm.outlet"))
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Cannot remove rpm outlet. Please delete rpm")
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-3-rpm-outlet")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-3-rpm-outlet")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			s, err := state.GetStateRecord(ctx, "hosts/dut-3-rpm-outlet")
			So(err, ShouldBeNil)
			// State should be set to registered.
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_REGISTERED)
		})

		Convey("UpdateDUT - With valid rpm mask", func() {
			// Create a DUT with labstation. Also creates servo with port: 9999 and serial: serial-1
			createValidDUTWithLabstation(ctx, "dut-3-rpm", "machine-60-rpm", "labstation-3-rpm", "machine-50-rpm")
			dut1 := mockDUT("dut-3-rpm", "machine-60-rpm", "labstation-3-rpm", "serial-1", "power-2", ".A2", int32(9999), []string{"DUT_POOL_QUOTA"})
			resp, err := UpdateDUT(ctx, dut1, mockFieldMask("dut.rpm.outlet", "dut.rpm.host"))
			So(err, ShouldBeNil)
			// Remove update time to compare proto
			resp.UpdateTime = nil
			So(resp, ShouldResembleProto, dut1)
			dut2, err := GetMachineLSE(ctx, "dut-3-rpm")
			So(err, ShouldBeNil)
			dut2.UpdateTime = nil
			// Remove update time to compare proto
			So(dut2, ShouldResembleProto, dut1)
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-3-rpm")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 3)
			So(changes[1].OldValue, ShouldEqual, "power-1")
			So(changes[1].NewValue, ShouldEqual, "power-2")
			So(changes[2].OldValue, ShouldEqual, ".A1")
			So(changes[2].NewValue, ShouldEqual, ".A2")
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-3-rpm")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 2)
			s, err := state.GetStateRecord(ctx, "hosts/dut-3-rpm")
			So(err, ShouldBeNil)
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_REGISTERED)
		})

		Convey("UpdateDUT - With valid rpm mask to delete rpm", func() {
			// Create a DUT with labstation. Also creates servo with port: 9999 and serial: serial-1
			createValidDUTWithLabstation(ctx, "dut-3-rpm-del", "machine-60-rpm-del", "labstation-3-rpm-del", "machine-50-rpm-del")
			// Update with rpm host mask and no rpm.
			dut1 := mockDUT("dut-3-rpm-del", "machine-60-rpm-del", "labstation-3-rpm-del", "serial-1", "", "", int32(9999), []string{"DUT_POOL_QUOTA"})
			dut1.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().Rpm = nil
			resp, err := UpdateDUT(ctx, dut1, mockFieldMask("dut.rpm.host"))
			So(err, ShouldBeNil)
			// Remove update time to compare proto
			resp.UpdateTime = nil
			So(resp, ShouldResembleProto, dut1)
			dut2, err := GetMachineLSE(ctx, "dut-3-rpm-del")
			// Remove update time to compare proto
			dut2.UpdateTime = nil
			So(dut2, ShouldResembleProto, dut1)
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-3-rpm-del")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 3)
			So(changes[1].OldValue, ShouldEqual, "power-1")
			So(changes[1].NewValue, ShouldEqual, "")
			So(changes[2].OldValue, ShouldEqual, ".A1")
			So(changes[2].NewValue, ShouldEqual, "")
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-3-rpm-del")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 2)
			s, err := state.GetStateRecord(ctx, "hosts/dut-3-rpm-del")
			So(err, ShouldBeNil)
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_REGISTERED)
		})

		Convey("UpdateDUT - With invalid mask", func() {
			// Create a DUT with labstation. Also creates servo with port: 9999 and serial: serial-1
			createValidDUTWithLabstation(ctx, "dut-3", "machine-60", "labstation-3", "machine-50")
			// Update with invalid masks.
			dut1 := mockDUT("dut-3", "machine-60", "labstation-3", "serial-1", "power-1", ".A1", int32(9999), []string{"DUT_POOL_QUOTA"})
			_, err := UpdateDUT(ctx, dut1, mockFieldMask("invalid-mask-1", "invalid-mask-2"))
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "unsupported update mask path")
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-3")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-3")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			s, err := state.GetStateRecord(ctx, "hosts/dut-3")
			So(err, ShouldBeNil)
			// State should be set to registered.
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_REGISTERED)
		})

		Convey("UpdateDUT - Servo port conflict", func() {
			machine3 := &ufspb.Machine{
				Name: "machine-01",
				Device: &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: &ufspb.ChromeOSMachine{
						BuildTarget: "test",
						Model:       "test",
					},
				},
			}
			_, err := registration.CreateMachine(ctx, machine3)
			So(err, ShouldBeNil)
			// Create a DUT with labstation. Also creates servo with port: 9999 and serial: serial-1.
			createValidDUTWithLabstation(ctx, "dut-6", "machine-00", "labstation-5", "machine-02")
			dut1 := mockDUT("dut-7", "machine-01", "labstation-5", "serial-2", "power-1", ".A1", int32(9998), []string{"DUT_POOL_QUOTA"})
			_, err = CreateDUT(ctx, dut1)
			So(err, ShouldBeNil)
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-7")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-7")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			So(msgs[0].Delete, ShouldBeFalse)
			// Update port to 9999 creating conflict with dut-6 servo.
			dut2 := mockDUT("dut-7", "machine-01", "labstation-5", "serial-2", "power-1", ".A1", int32(9999), []string{"DUT_POOL_QUOTA"})
			// Maskless update.
			_, err = UpdateDUT(ctx, dut2, nil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Port: 9999 in labstation-5 is already in use by dut-6")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-7")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-7")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			// Verify that labstation-5 has 2 servos. And wasn't changed after last update.
			ls9, err := GetMachineLSE(ctx, "labstation-5")
			So(err, ShouldBeNil)
			So(ls9.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetServos(), ShouldResembleProto, []*chromeosLab.Servo{
				// Servo generated by createValidDUTWithLabstation.
				{
					ServoHostname: "labstation-5",
					ServoPort:     int32(9999),
					ServoSerial:   "serial-1",
				},
				// dut-9 servo should remain on port 9998.
				{
					ServoHostname: "labstation-5",
					ServoPort:     int32(9998),
					ServoSerial:   "serial-2",
				},
			})
			s, err := state.GetStateRecord(ctx, "hosts/dut-7")
			So(err, ShouldBeNil)
			// State should be set to registered.
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_REGISTERED)
			s, err = state.GetStateRecord(ctx, "hosts/dut-6")
			So(err, ShouldBeNil)
			// State should be set to registered.
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_REGISTERED)
		})

		Convey("UpdateDUT - Servo serial conflict", func() {
			machine3 := &ufspb.Machine{
				Name: "machine-03",
				Device: &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: &ufspb.ChromeOSMachine{
						BuildTarget: "test",
						Model:       "test",
					},
				},
			}
			_, err := registration.CreateMachine(ctx, machine3)
			So(err, ShouldBeNil)
			// Create a DUT with labstation. Also creates servo with port: 9999 and serial: serial-1
			createValidDUTWithLabstation(ctx, "dut-8", "machine-04", "labstation-6", "machine-05")
			dut1 := mockDUT("dut-9", "machine-03", "labstation-6", "serial-2", "power-1", ".A1", int32(9998), []string{"DUT_POOL_QUOTA"})
			_, err = CreateDUT(ctx, dut1)
			So(err, ShouldBeNil)
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-9")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-9")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			So(msgs[0].Delete, ShouldBeFalse)
			// Update dut-9 servo with servo serial of dut-8 (serial-1 created by createValidDUTWithLabstation).
			dut2 := mockDUT("dut-9", "machine-03", "labstation-6", "serial-1", "power-1", ".A1", int32(9997), []string{"DUT_POOL_QUOTA"})
			// Maskless update.
			_, err = UpdateDUT(ctx, dut2, nil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Servo serial serial-1 already exists on labstation-6")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-9")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-9")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			// Verify that labstation-6 has 2 servos. And wasn't changed after last update.
			ls9, err := GetMachineLSE(ctx, "labstation-6")
			So(err, ShouldBeNil)
			// Verify that nothing was changed on labstation.
			So(ls9.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetServos(), ShouldResembleProto, []*chromeosLab.Servo{
				// Servo generated by createValidDUTWithLabstation.
				{
					ServoHostname: "labstation-6",
					ServoPort:     int32(9999),
					ServoSerial:   "serial-1",
				},
				// dut-9 servo should remain serial-2.
				{
					ServoHostname: "labstation-6",
					ServoPort:     int32(9998),
					ServoSerial:   "serial-2",
				},
			})
			s, err := state.GetStateRecord(ctx, "hosts/dut-8")
			So(err, ShouldBeNil)
			// State should be set to registered.
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_REGISTERED)
			s, err = state.GetStateRecord(ctx, "hosts/dut-9")
			So(err, ShouldBeNil)
			// State should be set to registered.
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_REGISTERED)
		})

		Convey("UpdateDUT - Move servo to different labstation", func() {
			// Create a DUT with labstation. Also creates servo with port: 9999 and serial: serial-1
			createValidDUTWithLabstation(ctx, "dut-10", "machine-06", "labstation-7", "machine-07")
			// Create a DUT with labstation. Also creates servo with port: 9999 and serial: serial-1
			createValidDUTWithLabstation(ctx, "dut-11", "machine-08", "labstation-8", "machine-09")
			// Update the servo serial of the dut to avoid conflict with labstation-7.
			dut2 := mockDUT("dut-11", "", "", "serial-2", "", "", int32(0), nil)
			resp, err := UpdateDUT(ctx, dut2, mockFieldMask("dut.servo.serial"))
			So(err, ShouldBeNil)
			dut2 = mockDUT("dut-11", "", "labstation-7", "", "", "", int32(9998), nil)
			resp, err = UpdateDUT(ctx, dut2, mockFieldMask("dut.servo.hostname", "dut.servo.port"))
			So(err, ShouldBeNil)
			// Verify that labstation-8 has no servos left on it.
			ls8, err := GetMachineLSE(ctx, "labstation-8")
			So(err, ShouldBeNil)
			So(ls8.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetServos(), ShouldHaveLength, 0)
			// Verify that labstation-7 has 2 servos
			ls9, err := GetMachineLSE(ctx, "labstation-7")
			So(err, ShouldBeNil)
			// Verify that the moved servo was included in the new labstation.
			So(ls9.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetServos(), ShouldResembleProto, []*chromeosLab.Servo{
				// Servo generated by createValidDUTWithLabstation.
				{
					ServoHostname: "labstation-7",
					ServoPort:     int32(9999),
					ServoSerial:   "serial-1",
				},
				// The new servo can be obtained from last DUT update's result.
				resp.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetServo(),
			})
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-11")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 4)
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-11")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 3)
			s, err := state.GetStateRecord(ctx, "hosts/dut-10")
			So(err, ShouldBeNil)
			// State should be set to registered.
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_REGISTERED)
			s, err = state.GetStateRecord(ctx, "hosts/dut-11")
			So(err, ShouldBeNil)
			// State should remain same.
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_REGISTERED)
		})

		Convey("UpdateDUT - Delete and add servo", func() {
			// Create a DUT with labstation. Also creates servo with port: 9999 and serial: serial-1
			createValidDUTWithLabstation(ctx, "dut-12", "machine-11", "labstation-9", "machine-12")
			// Update the servo serial of the dut to avoid conflict with labstation-7.
			dut2 := mockDUT("dut-12", "", "", "", "", "", int32(0), nil)
			_, err := UpdateDUT(ctx, dut2, mockFieldMask("dut.servo.hostname"))
			So(err, ShouldBeNil)
			// Verify that labstation-9 has no servos left on it.
			ls9, err := GetMachineLSE(ctx, "labstation-9")
			So(err, ShouldBeNil)
			So(ls9.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetServos(), ShouldHaveLength, 0)
			// Add the servo back.
			dut2 = mockDUT("dut-12", "", "labstation-9", "serial-2", "", "", int32(9901), nil)
			_, err = UpdateDUT(ctx, dut2, mockFieldMask("dut.servo.hostname", "dut.servo.serial", "dut.servo.port"))
			So(err, ShouldBeNil)
			// Verify that labstation-9 has servo
			ls9, err = GetMachineLSE(ctx, "labstation-9")
			So(err, ShouldBeNil)
			// Verify that the moved servo was included in the new labstation.
			So(ls9.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetServos(), ShouldResembleProto, []*chromeosLab.Servo{
				// Servo generated by createValidDUTWithLabstation.
				{
					ServoHostname: "labstation-9",
					ServoPort:     int32(9901),
					ServoSerial:   "serial-2",
				},
			})
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-12")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 7)
			So(changes[1].OldValue, ShouldEqual, "labstation-9")
			So(changes[1].NewValue, ShouldEqual, "")
			So(changes[2].OldValue, ShouldEqual, "9999")
			So(changes[2].NewValue, ShouldEqual, "0")
			So(changes[3].OldValue, ShouldEqual, "serial-1")
			So(changes[3].NewValue, ShouldEqual, "")
			So(changes[4].NewValue, ShouldEqual, "labstation-9")
			So(changes[4].OldValue, ShouldEqual, "")
			So(changes[5].NewValue, ShouldEqual, "9901")
			So(changes[5].OldValue, ShouldEqual, "0")
			So(changes[6].NewValue, ShouldEqual, "serial-2")
			So(changes[6].OldValue, ShouldEqual, "")
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-12")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 3)
			s, err := state.GetStateRecord(ctx, "hosts/dut-12")
			So(err, ShouldBeNil)
			// State should be set to registered.
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_REGISTERED)
		})
	})
}

func TestGetChromeOSDevicedata(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	ctx = external.WithTestingContext(ctx)
	machine := &ufspb.Machine{
		Name: "machine-1",
		Device: &ufspb.Machine_ChromeosMachine{
			ChromeosMachine: &ufspb.ChromeOSMachine{
				BuildTarget: "test",
				Model:       "test",
				Hwid:        "test",
			},
		},
	}
	registration.CreateMachine(ctx, machine)

	dutMachinelse := mockDutMachineLSE("lse-1")
	dutMachinelse.Machines = []string{"machine-1"}
	inventory.CreateMachineLSE(ctx, dutMachinelse)

	dutState := mockDutState("machine-1", "lse-1")
	UpdateDutState(ctx, dutState)

	devCfg := &device.Config{
		Id: &device.ConfigId{
			PlatformId: &device.PlatformId{Value: "test"},
			ModelId:    &device.ModelId{Value: "test"},
		},
	}

	mfgCfg := &manufacturing.Config{
		ManufacturingId: &manufacturing.ConfigID{Value: "test"},
	}

	hwidData := &ufspb.HwidData{Sku: "test", Variant: "test"}

	Convey("TestGetChromeOSDevicedata", t, func() {
		Convey("GetChromeOSDevicedata - id happy path", func() {
			resp, err := GetChromeOSDeviceData(ctx, "machine-1", "")
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp.GetLabConfig(), ShouldResembleProto, dutMachinelse)
			So(resp.GetMachine(), ShouldResembleProto, machine)
			So(resp.GetDutState(), ShouldResembleProto, dutState)
			So(resp.GetDeviceConfig(), ShouldResembleProto, devCfg)
			So(resp.GetManufacturingConfig(), ShouldResembleProto, mfgCfg)
			So(resp.GetHwidData(), ShouldResembleProto, hwidData)
		})

		Convey("GetChromeOSDevicedata - hostname happy path", func() {
			resp, err := GetChromeOSDeviceData(ctx, "", "lse-1")
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp.GetLabConfig(), ShouldResembleProto, dutMachinelse)
			So(resp.GetMachine(), ShouldResembleProto, machine)
			So(resp.GetDutState(), ShouldResembleProto, dutState)
			So(resp.GetDeviceConfig(), ShouldResembleProto, devCfg)
			So(resp.GetManufacturingConfig(), ShouldResembleProto, mfgCfg)
			So(resp.GetHwidData(), ShouldResembleProto, hwidData)
		})

		Convey("GetChromeOSDevicedata - InvV2 errors", func() {
			machine := &ufspb.Machine{
				Name: "machine-2",
				Device: &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: &ufspb.ChromeOSMachine{
						BuildTarget: "test-err",
						Model:       "test-err",
						Hwid:        "test-err",
					},
				},
			}
			registration.CreateMachine(ctx, machine)

			dutMachinelse := mockDutMachineLSE("lse-2")
			dutMachinelse.Machines = []string{"machine-2"}
			inventory.CreateMachineLSE(ctx, dutMachinelse)

			dutState := mockDutState("machine-2", "lse-2")
			UpdateDutState(ctx, dutState)

			resp, err := GetChromeOSDeviceData(ctx, "", "lse-2")
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp.GetLabConfig(), ShouldResembleProto, dutMachinelse)
			So(resp.GetMachine(), ShouldResembleProto, machine)
			So(resp.GetDutState(), ShouldResembleProto, dutState)
			So(resp.GetDeviceConfig(), ShouldBeNil)
			So(resp.GetManufacturingConfig(), ShouldBeNil)
			So(resp.GetHwidData(), ShouldBeNil)
		})

		Convey("GetChromeOSDevicedata - dutState not found", func() {
			machine := &ufspb.Machine{
				Name: "machine-3",
				Device: &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: &ufspb.ChromeOSMachine{
						BuildTarget: "test-err",
						Model:       "test-err",
						Hwid:        "test-err",
					},
				},
			}
			registration.CreateMachine(ctx, machine)

			dutMachinelse := mockDutMachineLSE("lse-3")
			dutMachinelse.Machines = []string{"machine-3"}
			inventory.CreateMachineLSE(ctx, dutMachinelse)

			resp, err := GetChromeOSDeviceData(ctx, "", "lse-3")
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp.GetLabConfig(), ShouldResembleProto, dutMachinelse)
			So(resp.GetMachine(), ShouldResembleProto, machine)
			So(resp.GetDutState(), ShouldBeNil)
			So(resp.GetDeviceConfig(), ShouldBeNil)
			So(resp.GetManufacturingConfig(), ShouldBeNil)
			So(resp.GetHwidData(), ShouldBeNil)
		})

		Convey("GetChromeOSDevicedata - machine not found by hostname", func() {
			dutMachinelse := mockDutMachineLSE("lse-4")
			dutMachinelse.Machines = []string{"machine-4"}
			inventory.CreateMachineLSE(ctx, dutMachinelse)

			resp, err := GetChromeOSDeviceData(ctx, "", "lse-4")
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp.GetLabConfig(), ShouldResembleProto, dutMachinelse)
			So(resp.GetMachine(), ShouldBeNil)
			So(resp.GetDutState(), ShouldBeNil)
			So(resp.GetDeviceConfig(), ShouldBeNil)
			So(resp.GetManufacturingConfig(), ShouldBeNil)
			So(resp.GetHwidData(), ShouldBeNil)
		})

		Convey("GetChromeOSDevicedata - machine not found by id", func() {
			dutMachinelse := mockDutMachineLSE("lse-5")
			dutMachinelse.Machines = []string{"machine-5"}
			inventory.CreateMachineLSE(ctx, dutMachinelse)

			resp, err := GetChromeOSDeviceData(ctx, "machine-5", "")
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp.GetLabConfig(), ShouldResembleProto, dutMachinelse)
			So(resp.GetMachine(), ShouldBeNil)
			So(resp.GetDutState(), ShouldBeNil)
			So(resp.GetDeviceConfig(), ShouldBeNil)
			So(resp.GetManufacturingConfig(), ShouldBeNil)
			So(resp.GetHwidData(), ShouldBeNil)
		})

		Convey("GetChromeOSDevicedata - machinelse not found Error", func() {
			resp, err := GetChromeOSDeviceData(ctx, "machine-6", "")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "NotFound")
		})
	})
}
