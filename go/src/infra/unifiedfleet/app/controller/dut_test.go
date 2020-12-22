package controller

import (
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"

	"google.golang.org/genproto/protobuf/field_mask"
	ufspb "infra/unifiedfleet/api/v1/models"
	lab "infra/unifiedfleet/api/v1/models/chromeos/lab"
	"infra/unifiedfleet/app/external"
	"infra/unifiedfleet/app/model/history"
	"infra/unifiedfleet/app/model/registration"
)

func mockDUT(hostname, machine, servoHost, servoSerial, rpm, rpmOutlet string, servoPort int32, pools []lab.DeviceUnderTest_DUTPool) *ufspb.MachineLSE {
	return &ufspb.MachineLSE{
		Name:     hostname,
		Hostname: hostname,
		Machines: []string{machine},
		Lse: &ufspb.MachineLSE_ChromeosMachineLse{
			ChromeosMachineLse: &ufspb.ChromeOSMachineLSE{
				ChromeosLse: &ufspb.ChromeOSMachineLSE_DeviceLse{
					DeviceLse: &ufspb.ChromeOSDeviceLSE{
						Device: &ufspb.ChromeOSDeviceLSE_Dut{
							Dut: &lab.DeviceUnderTest{
								Hostname: hostname,
								Peripherals: &lab.Peripherals{
									Servo: &lab.Servo{
										ServoHostname: servoHost,
										ServoPort:     servoPort,
										ServoSerial:   servoSerial,
									},
									Rpm: &lab.RPM{
										PowerunitName:   rpm,
										PowerunitOutlet: rpmOutlet,
									},
								},
								CriticalPools: pools,
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
							Labstation: &lab.Labstation{
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
	_, err = createLabstation(ctx, labstation1)
	So(err, ShouldBeNil)
	dut1 := mockDUT(dutName, dutMachine, labstationName, "serial-1", "power-1", ".A1", int32(9999), []lab.DeviceUnderTest_DUTPool{lab.DeviceUnderTest_DUT_POOL_QUOTA})
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
			dut1 := mockDUT("dut-1", "machine-10", "labstation-1", "serial-1", "power-1", ".A1", int32(9999), []lab.DeviceUnderTest_DUTPool{lab.DeviceUnderTest_DUT_POOL_QUOTA})
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
			_, err = createLabstation(ctx, labstation1)
			So(err, ShouldBeNil)
			dut1 := mockDUT("dut-2", "machine-21", "labstation-1", "serial-1", "power-1", ".A1", int32(9999), []lab.DeviceUnderTest_DUTPool{lab.DeviceUnderTest_DUT_POOL_QUOTA})
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
			_, err = createLabstation(ctx, labstation1)
			So(err, ShouldBeNil)
			dut1 := mockDUT("dut-3", "machine-40", "labstation-3", "serial-2", "power-1", ".A1", int32(9999), []lab.DeviceUnderTest_DUTPool{lab.DeviceUnderTest_DUT_POOL_QUOTA})
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
			dut2 := mockDUT("dut-4", "machine-50", "labstation-3", "serial-3", "power-1", ".A2", int32(9999), []lab.DeviceUnderTest_DUTPool{lab.DeviceUnderTest_DUT_POOL_QUOTA})
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
			_, err := createLabstation(ctx, labstation1)
			So(err, ShouldBeNil)
			dut1 := mockDUT("dut-7", "machine-00", "labstation-5", "serial-1", "power-3", ".A1", int32(9999), []lab.DeviceUnderTest_DUTPool{lab.DeviceUnderTest_DUT_POOL_QUOTA})
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
			_, err := createLabstation(ctx, labstation1)
			So(err, ShouldBeNil)
			dut1 := mockDUT("dut-8", "machine-02", "labstation-6", "serial-1", "power-3", ".A1", int32(9999), []lab.DeviceUnderTest_DUTPool{lab.DeviceUnderTest_DUT_POOL_QUOTA})
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
			_, err := createLabstation(ctx, labstation1)
			So(err, ShouldBeNil)
			dut1 := mockDUT("dut-9", "machine-03", "labstation-7", "serial-1", "power-3", ".A1", int32(9999), []lab.DeviceUnderTest_DUTPool{lab.DeviceUnderTest_DUT_POOL_QUOTA})
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
			dut1 := mockDUT("dut-1", "machine-10", "labstation-1", "serial-1", "power-1", ".A1", int32(9999), []lab.DeviceUnderTest_DUTPool{lab.DeviceUnderTest_DUT_POOL_QUOTA})
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
			createValidDUTWithLabstation(ctx, "dut-1", "machine-20", "labstation-1", "machine-10")
			// Update DUT machine to a non existent one
			dut1 := mockDUT("dut-1", "machine-20-fake", "labstation-1", "serial-1", "power-1", ".A1", int32(9999), []lab.DeviceUnderTest_DUTPool{lab.DeviceUnderTest_DUT_POOL_QUOTA})
			_, err := UpdateDUT(ctx, dut1, nil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "There is no Machine with MachineID machine-20-fake in the system")
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-1")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
		})

		Convey("UpdateDUT - With existing deployed machine", func() {
			createValidDUTWithLabstation(ctx, "dut-2", "machine-40", "labstation-2", "machine-30")
			// Update DUT machine to labstations machine.
			dut1 := mockDUT("dut-2", "machine-30", "labstation-2", "serial-1", "power-1", ".A1", int32(9999), []lab.DeviceUnderTest_DUTPool{lab.DeviceUnderTest_DUT_POOL_QUOTA})
			_, err := UpdateDUT(ctx, dut1, nil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Host dut-2 cannot be updated because there is another host labstation-2 which is referring this machine machine-30")
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-2")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-2")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
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
		})

		Convey("UpdateDUT - With invalid update_time mask", func() {
			createValidDUTWithLabstation(ctx, "dut-3-time", "machine-60-time", "labstation-3-time", "machine-50-time")
			// Update with update_time mask.
			dut1 := mockDUT("dut-3-time", "machine-60-time", "labstation-3-time", "serial-1", "power-1", ".A1", int32(9999), []lab.DeviceUnderTest_DUTPool{lab.DeviceUnderTest_DUT_POOL_QUOTA})
			_, err := UpdateDUT(ctx, dut1, mockFieldMask("update-time"))
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "unsupported update mask path")
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-3-time")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-3-time")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
		})

		Convey("UpdateDUT - With invalid machine mask", func() {
			createValidDUTWithLabstation(ctx, "dut-3-machine", "machine-60-machine", "labstation-3-machine", "machine-50-machine")
			// Update with machine mask and no machines.
			dut1 := mockDUT("dut-3-machine", "", "labstation-3-machine", "serial-1", "power-1", ".A1", int32(9999), []lab.DeviceUnderTest_DUTPool{lab.DeviceUnderTest_DUT_POOL_QUOTA})
			_, err := UpdateDUT(ctx, dut1, mockFieldMask("machines"))
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "machines field cannot be empty/nil")
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-3-machine")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-3-machine")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
		})

		Convey("UpdateDUT - With invalid dut hostname mask", func() {
			createValidDUTWithLabstation(ctx, "dut-3-hostname", "machine-60-hostname", "labstation-3-hostname", "machine-50-hostname")
			// Update with dut hostname mask.
			dut1 := mockDUT("dut-3-hostname", "machine-60-hostname", "labstation-3-hostname", "serial-1", "power-1", ".A1", int32(9999), []lab.DeviceUnderTest_DUTPool{lab.DeviceUnderTest_DUT_POOL_QUOTA})
			_, err := UpdateDUT(ctx, dut1, mockFieldMask("dut.hostname"))
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "hostname cannot be updated")
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-3-hostname")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-3-hostname")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
		})

		Convey("UpdateDUT - With valid dut pools mask", func() {
			createValidDUTWithLabstation(ctx, "dut-3-pools", "machine-60-pools", "labstation-3-pools", "machine-50-pools")
			// Update with dut pools mask and valid pools.
			dut1 := mockDUT("dut-3-pools", "machine-60-pools", "labstation-3-pools", "serial-1", "power-1", ".A1", int32(9999), []lab.DeviceUnderTest_DUTPool{lab.DeviceUnderTest_DUT_POOL_QUOTA, lab.DeviceUnderTest_DUT_POOL_CQ})
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
		})

		Convey("UpdateDUT - With invalid servo host mask (delete host and update port)", func() {
			createValidDUTWithLabstation(ctx, "dut-3-servo-host", "machine-60-servo-host", "labstation-3-servo-host", "machine-50-servo-host")
			// Update with servo host mask and no servo host.
			dut1 := mockDUT("dut-3-servo-host", "machine-60-servo-host", "", "", "", "", int32(9999), nil)
			_, err := UpdateDUT(ctx, dut1, mockFieldMask("dut.servo.hostname", "dut.servo.port"))
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Cannot update servo port. Servo host is being reset.")
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-3-servo-host")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-3-servo-host")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
		})

		Convey("UpdateDUT - With invalid servo host mask (delete host and update serial)", func() {
			createValidDUTWithLabstation(ctx, "dut-3-servo-serial", "machine-60-servo-serial", "labstation-3-servo-serial", "machine-50-servo-serial")
			// Update with servo host mask and no servo host.
			dut1 := mockDUT("dut-3-servo-serial", "machine-60-servo-serial", "", "serial-2", "", "", int32(0), nil)
			_, err := UpdateDUT(ctx, dut1, mockFieldMask("dut.servo.hostname", "dut.servo.serial"))
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Cannot update servo serial. Servo host is being reset")
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-3-servo-serial")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-3-servo-serial")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
		})

		Convey("UpdateDUT - With valid servo mask", func() {
			createValidDUTWithLabstation(ctx, "dut-3-servo", "machine-60-servo", "labstation-3-servo", "machine-50-servo")
			// Update with servo host mask and no servo host.
			dut1 := mockDUT("dut-3-servo", "machine-60-servo", "labstation-3-servo", "serial-2", "power-1", ".A1", int32(9998), []lab.DeviceUnderTest_DUTPool{lab.DeviceUnderTest_DUT_POOL_QUOTA})
			resp, err := UpdateDUT(ctx, dut1, mockFieldMask("dut.servo.port", "dut.servo.serial"))
			So(err, ShouldBeNil)
			// Clear update time to compare the protos
			resp.UpdateTime = nil
			So(resp, ShouldResembleProto, dut1)
			dut2, err := GetMachineLSE(ctx, "dut-3-servo")
			So(err, ShouldBeNil)
			// Clear update time to compare the protos
			dut2.UpdateTime = nil
			So(dut2, ShouldResembleProto, dut1)
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-3-servo")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 3)
			So(changes[1].OldValue, ShouldEqual, "9999")
			So(changes[1].NewValue, ShouldEqual, "9998")
			So(changes[2].OldValue, ShouldEqual, "serial-1")
			So(changes[2].NewValue, ShouldEqual, "serial-2")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "hosts/labstation-3-servo")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 2)
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-3-servo")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 2)
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/labstation-3-servo")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 2)
		})

		Convey("UpdateDUT - With valid servo mask (delete servo)", func() {
			createValidDUTWithLabstation(ctx, "dut-3-servo-del", "machine-60-servo-del", "labstation-3-servo-del", "machine-50-servo-del")
			// Update with servo host mask and no servo host.
			dut1 := mockDUT("dut-3-servo-del", "machine-60-servo-del", "", "", "power-1", ".A1", int32(0), []lab.DeviceUnderTest_DUTPool{lab.DeviceUnderTest_DUT_POOL_QUOTA})
			// Remove servo from DUT.
			dut1.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().Servo = nil
			resp, err := UpdateDUT(ctx, dut1, mockFieldMask("dut.servo.hostname"))
			So(err, ShouldBeNil)
			// Clear update time to compare the protos
			resp.UpdateTime = nil
			So(resp, ShouldResembleProto, dut1)
			dut2, err := GetMachineLSE(ctx, "dut-3-servo-del")
			So(err, ShouldBeNil)
			// Clear update time to compare the protos
			dut2.UpdateTime = nil
			So(dut2, ShouldResembleProto, dut1)
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-3-servo-del")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 4)
			So(changes[1].OldValue, ShouldEqual, "labstation-3-servo-del")
			So(changes[1].NewValue, ShouldEqual, "")
			So(changes[2].OldValue, ShouldEqual, "9999")
			So(changes[2].NewValue, ShouldEqual, "0")
			So(changes[3].OldValue, ShouldEqual, "serial-1")
			So(changes[3].NewValue, ShouldEqual, "")

			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-3-servo-del")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 2)
		})

		Convey("UpdateDUT - With invalid rpm host mask", func() {
			createValidDUTWithLabstation(ctx, "dut-3-rpm-host", "machine-60-rpm-host", "labstation-3-rpm-host", "machine-50-rpm-host")
			// Update with rpm host mask and no rpm host.
			dut1 := mockDUT("dut-3-rpm-host", "machine-60-rpm-host", "labstation-3-rpm-host", "serial-1", "", ".A1", int32(9999), []lab.DeviceUnderTest_DUTPool{lab.DeviceUnderTest_DUT_POOL_QUOTA})
			_, err := UpdateDUT(ctx, dut1, mockFieldMask("dut.rpm.host", "dut.rpm.outlet"))
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Deleting rpm host deletes everything. Cannot update outlet.")
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-3-rpm-host")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-3-rpm-host")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
		})

		Convey("UpdateDUT - With invalid rpm outlet mask", func() {
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
		})

		Convey("UpdateDUT - With valid rpm mask", func() {
			createValidDUTWithLabstation(ctx, "dut-3-rpm", "machine-60-rpm", "labstation-3-rpm", "machine-50-rpm")
			dut1 := mockDUT("dut-3-rpm", "machine-60-rpm", "labstation-3-rpm", "serial-1", "power-2", ".A2", int32(9999), []lab.DeviceUnderTest_DUTPool{lab.DeviceUnderTest_DUT_POOL_QUOTA})
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
		})

		Convey("UpdateDUT - With valid rpm mask to delete rpm", func() {
			createValidDUTWithLabstation(ctx, "dut-3-rpm-del", "machine-60-rpm-del", "labstation-3-rpm-del", "machine-50-rpm-del")
			// Update with rpm host mask and no rpm.
			dut1 := mockDUT("dut-3-rpm-del", "machine-60-rpm-del", "labstation-3-rpm-del", "serial-1", "", "", int32(9999), []lab.DeviceUnderTest_DUTPool{lab.DeviceUnderTest_DUT_POOL_QUOTA})
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
		})

		Convey("UpdateDUT - With invalid mask", func() {
			createValidDUTWithLabstation(ctx, "dut-3", "machine-60", "labstation-3", "machine-50")
			// Update with invalid masks.
			dut1 := mockDUT("dut-3", "machine-60", "labstation-3", "serial-1", "power-1", ".A1", int32(9999), []lab.DeviceUnderTest_DUTPool{lab.DeviceUnderTest_DUT_POOL_QUOTA})
			_, err := UpdateDUT(ctx, dut1, mockFieldMask("invalid-mask-1", "invalid-mask-2"))
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "unsupported update mask path")
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-3")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-3")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
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
			// Create a DUT with labstation.
			createValidDUTWithLabstation(ctx, "dut-6", "machine-00", "labstation-5", "machine-02")
			dut1 := mockDUT("dut-7", "machine-01", "labstation-5", "serial-2", "power-1", ".A1", int32(9998), []lab.DeviceUnderTest_DUTPool{lab.DeviceUnderTest_DUT_POOL_QUOTA})
			_, err = CreateDUT(ctx, dut1)
			So(err, ShouldBeNil)
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-7")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-7")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			So(msgs[0].Delete, ShouldBeFalse)
			// Update to dut-6 servo.
			dut2 := mockDUT("dut-7", "machine-01", "labstation-5", "serial-1", "power-1", ".A1", int32(9999), []lab.DeviceUnderTest_DUTPool{lab.DeviceUnderTest_DUT_POOL_QUOTA})
			_, err = UpdateDUT(ctx, dut2, nil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Port: 9999 in labstation-5 is already in use by dut-6")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-7")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-7")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
		})
	})
}
