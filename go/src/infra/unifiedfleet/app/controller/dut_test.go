package controller

import (
	"context"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/chromiumos/config/go/api"
	"go.chromium.org/chromiumos/config/go/payload"
	. "go.chromium.org/luci/common/testing/assertions"
	"google.golang.org/genproto/protobuf/field_mask"
	"google.golang.org/grpc/codes"

	ufspb "infra/unifiedfleet/api/v1/models"
	device "infra/unifiedfleet/api/v1/models/chromeos/device"
	chromeosLab "infra/unifiedfleet/api/v1/models/chromeos/lab"
	ufsmanufacturing "infra/unifiedfleet/api/v1/models/chromeos/manufacturing"
	"infra/unifiedfleet/app/config"
	"infra/unifiedfleet/app/external"
	"infra/unifiedfleet/app/model/configuration"
	"infra/unifiedfleet/app/model/history"
	"infra/unifiedfleet/app/model/inventory"
	"infra/unifiedfleet/app/model/registration"
	"infra/unifiedfleet/app/model/state"
)

func mockDUT(hostname, machine, servoHost, servoSerial, rpm, rpmOutlet string, servoPort int32, pools []string, dockerContainer string) *ufspb.MachineLSE {
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
										ServoHostname:       servoHost,
										ServoPort:           servoPort,
										ServoSerial:         servoSerial,
										DockerContainerName: dockerContainer,
									},
									Rpm: &chromeosLab.OSRPM{
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
	dut1 := mockDUT(dutName, dutMachine, labstationName, "serial-1", dutName+"-power-1", ".A1", int32(9999), []string{"DUT_POOL_QUOTA"}, "")
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
			dut1 := mockDUT("dut-1", "machine-10", "labstation-1", "serial-1", "dut-1-power-1", ".A1", int32(9999), []string{"DUT_POOL_QUOTA"}, "")
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
			dut1 := mockDUT("dut-2", "machine-21", "labstation-1", "serial-1", "dut-2-power-1", ".A1", int32(9999), []string{"DUT_POOL_QUOTA"}, "")
			_, err = CreateDUT(ctx, dut1)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "No device config")
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
			dut1 := mockDUT("dut-3", "machine-40", "labstation-3", "serial-2", "dut-3-power-1", ".A1", int32(9999), []string{"DUT_POOL_QUOTA"}, "")
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
			dut2 := mockDUT("dut-4", "machine-50", "labstation-3", "serial-3", "dut-4-power-1", ".A2", int32(9999), []string{"DUT_POOL_QUOTA"}, "")
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
			dut1 := mockDUT("dut-7", "machine-00", "labstation-5", "serial-1", "dut-7-power-3", ".A1", int32(9999), []string{"DUT_POOL_QUOTA"}, "")
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
			dut1 := mockDUT("dut-8", "machine-02", "labstation-6", "serial-1", "dut-8-power-3", ".A1", int32(9999), []string{"DUT_POOL_QUOTA"}, "")
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
			dut1 := mockDUT("dut-9", "machine-03", "labstation-7", "serial-1", "dut-9-power-3", ".A1", int32(9999), []string{"DUT_POOL_QUOTA"}, "")
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
		Convey("CreateDUT - RPM powerunit_name and powerunit_outlet conflict", func() {
			machine1 := &ufspb.Machine{
				Name: "machine-101",
				Device: &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: &ufspb.ChromeOSMachine{
						BuildTarget: "test",
						Model:       "test",
					},
				},
			}
			_, err := registration.CreateMachine(ctx, machine1)
			So(err, ShouldBeNil)
			machine2 := &ufspb.Machine{
				Name: "machine-102",
				Device: &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: &ufspb.ChromeOSMachine{
						BuildTarget: "test",
						Model:       "test",
					},
				},
			}
			_, err = registration.CreateMachine(ctx, machine2)
			So(err, ShouldBeNil)
			dut1 := mockDUT("dut-16", "machine-101", "", "", "dut-16-power-1", ".A1", 0, nil, "")
			_, err = inventory.CreateMachineLSE(ctx, dut1)
			So(err, ShouldBeNil)
			dut2 := mockDUT("dut-17", "machine-102", "", "", "dut-16-power-1", ".A1", 0, nil, "")
			_, err = CreateDUT(ctx, dut2)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "The rpm powerunit_name and powerunit_outlet is already in use by dut-16")
		})
		Convey("CreateDUT - Skip labstation check if docker container is given", func() {
			machine1 := &ufspb.Machine{
				Name: "machine-103",
				Device: &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: &ufspb.ChromeOSMachine{
						BuildTarget: "test",
						Model:       "test",
					},
				},
			}
			_, err := registration.CreateMachine(ctx, machine1)
			So(err, ShouldBeNil)
			dut1 := mockDUT("dut-18", "machine-103", "labstation-x", "serial-x", "dut-16-power-1", ".A1", 9988, nil, "docker-1")
			_, err = inventory.CreateMachineLSE(ctx, dut1)
			So(err, ShouldBeNil)
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
			dut1 := mockDUT("dut-1", "machine-10", "labstation-1", "serial-1", "dut-1-power-1", ".A1", int32(9999), []string{"DUT_POOL_QUOTA"}, "")
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
			dut1 := mockDUT("dut-1", "machine-20-fake", "labstation-1", "serial-1", "dut-1-power-1", ".A1", int32(9999), []string{"DUT_POOL_QUOTA"}, "")
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
			dut1 := mockDUT("dut-2", "machine-30", "labstation-2", "serial-1", "dut-2-power-1", ".A1", int32(9999), []string{"DUT_POOL_QUOTA"}, "")
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
			dut1 := mockDUT("dut-3-name", "", "", "", "", "", int32(0), nil, "")
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
			dut1 := mockDUT("dut-3-time", "machine-60-time", "labstation-3-time", "serial-1", "dut-3-time-power-1", ".A1", int32(9999), []string{"DUT_POOL_QUOTA"}, "")
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
			dut1 := mockDUT("dut-3-machine", "", "labstation-3-machine", "serial-1", "dut-3-machine-power-1", ".A1", int32(9999), []string{"DUT_POOL_QUOTA"}, "")
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
			dut1 := mockDUT("dut-3-hostname", "machine-60-hostname", "labstation-3-hostname", "dut-3-hostname-serial-1", "power-1", ".A1", int32(9999), []string{"DUT_POOL_QUOTA"}, "")
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
			dut1 := mockDUT("dut-3-pools", "machine-60-pools", "labstation-3-pools", "serial-1", "dut-3-pools-power-1", ".A1", int32(9999), []string{"DUT_POOL_CQ", "DUT_POOL_QUOTA"}, "")
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
			So(changes[1].NewValue, ShouldEqual, "[DUT_POOL_CQ DUT_POOL_QUOTA]")
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
			dut1 := mockDUT("dut-3-non-v3-host", "machine-60-non-v3-host", "", "", "", "", int32(9999), nil, "")
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
			dut1 := mockDUT("dut-3-non-v3-serial", "machine-60-non-v3-serial", "", "dut-3-non-v3-serial-serial-2", "", "", int32(0), nil, "")
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

		Convey("UpdateDUT - With valid servo serial mask", func() {
			// Create a DUT with labstation. Also creates servo with port: 9999 and serial: serial-1
			createValidDUTWithLabstation(ctx, "dut-3-serial", "machine-60-serial", "labstation-3-serial", "machine-50-serial")
			// Update with servo host mask and no servo host.
			dut1 := mockDUT("dut-3-serial", "machine-60-serial", "labstation-3-serial", "serial-2", "dut-3-serial-power-1", ".A1", int32(9999), []string{"DUT_POOL_QUOTA"}, "")
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
			So(changes, ShouldHaveLength, 3)
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-3-serial")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 2)
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/labstation-3-serial")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 3)
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
			dut1 := mockDUT("dut-3-port", "machine-60-port", "labstation-3-port", "serial-1", "dut-3-port-power-1", ".A1", int32(9988), []string{"DUT_POOL_QUOTA"}, "")
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
			So(changes, ShouldHaveLength, 3)
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-3-port")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 2)
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/labstation-3-port")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 3)
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
			dut1 := mockDUT("dut-3-out-of-range-port", "", "", "", "", "", int32(1111), nil, "")
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
			dut1 := mockDUT("dut-3-port-auto-assign", "machine-60-port-auto-assign", "labstation-3-port-auto-assign", "serial-1", "dut-3-port-auto-assign-power-1", ".A1", int32(9001), []string{"DUT_POOL_QUOTA"}, "")
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
			dut1 := mockDUT("dut-3-del", "machine-60-del", "", "", "dut-3-del-power-1", ".A1", int32(0), []string{"DUT_POOL_QUOTA"}, "")
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
			createValidDUTWithLabstation(ctx, "dut-3-del-servo", "machine-60-del-servo", "labstation-3-del-serv", "machine-50-del-servo")
			// Update with servo host mask and no servo host.
			dut1 := mockDUT("dut-3-del-servo", "machine-60-del-servo", "", "", "dut-3-del-servo-power-1", ".A1", int32(0), []string{"DUT_POOL_QUOTA"}, "")
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
			So(changes, ShouldHaveLength, 4)
			So(changes[1].OldValue, ShouldEqual, "labstation-3-del-serv")
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
			dut1 := mockDUT("dut-3-rpm-host", "machine-60-rpm-host", "labstation-3-rpm-host", "dut-3-rpm-host-serial-1", "", ".A1", int32(9999), []string{"DUT_POOL_QUOTA"}, "")
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
			dut1 := mockDUT("dut-3-rpm-outlet", "", "", "", "", "", int32(0), nil, "")
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
			dut1 := mockDUT("dut-3-rpm", "machine-60-rpm", "labstation-3-rpm", "serial-1", "dut-3-rpm-power-2", ".A2", int32(9999), []string{"DUT_POOL_QUOTA"}, "")
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
			So(changes[1].OldValue, ShouldEqual, "dut-3-rpm-power-1")
			So(changes[1].NewValue, ShouldEqual, "dut-3-rpm-power-2")
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
			dut1 := mockDUT("dut-3-rpm-del", "machine-60-rpm-del", "labstation-3-rpm-del", "serial-1", "", "", int32(9999), []string{"DUT_POOL_QUOTA"}, "")
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
			So(changes[1].OldValue, ShouldEqual, "dut-3-rpm-del-power-1")
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
			dut1 := mockDUT("dut-3", "machine-60", "labstation-3", "serial-1", "dut-3-power-1", ".A1", int32(9999), []string{"DUT_POOL_QUOTA"}, "")
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
			dut1 := mockDUT("dut-7", "machine-01", "labstation-5", "serial-2", "dut-7-power-1", ".A1", int32(9998), []string{"DUT_POOL_QUOTA"}, "")
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
			dut2 := mockDUT("dut-7", "machine-01", "labstation-5", "serial-2", "dut-7-power-1", ".A1", int32(9999), []string{"DUT_POOL_QUOTA"}, "")
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
			dut1 := mockDUT("dut-9", "machine-03", "labstation-6", "serial-2", "dut-9-power-1", ".A1", int32(9998), []string{"DUT_POOL_QUOTA"}, "")
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
			dut2 := mockDUT("dut-9", "machine-03", "labstation-6", "serial-1", "dut-9-power-1", ".A1", int32(9997), []string{"DUT_POOL_QUOTA"}, "")
			// Maskless update.
			_, err = UpdateDUT(ctx, dut2, nil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Servo serial serial-1 exists")
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
			dut2 := mockDUT("dut-11", "", "", "serial-2", "", "", int32(0), nil, "")
			resp, err := UpdateDUT(ctx, dut2, mockFieldMask("dut.servo.serial"))
			So(err, ShouldBeNil)
			dut2 = mockDUT("dut-11", "", "labstation-7", "", "", "", int32(9998), nil, "")
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
			dut2 := mockDUT("dut-12", "", "", "", "", "", int32(0), nil, "")
			_, err := UpdateDUT(ctx, dut2, mockFieldMask("dut.servo.hostname"))
			So(err, ShouldBeNil)
			// Verify that labstation-9 has no servos left on it.
			ls9, err := GetMachineLSE(ctx, "labstation-9")
			So(err, ShouldBeNil)
			So(ls9.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetServos(), ShouldHaveLength, 0)
			// Add the servo back.
			dut2 = mockDUT("dut-12", "", "labstation-9", "serial-2", "", "", int32(9901), nil, "")
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

		Convey("UpdateDUT - RPM powerunit_name and powerunit_outlet conflict", func() {
			machine1 := &ufspb.Machine{
				Name: "machine-101",
				Device: &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: &ufspb.ChromeOSMachine{
						BuildTarget: "test",
						Model:       "test",
					},
				},
			}
			_, err := registration.CreateMachine(ctx, machine1)
			So(err, ShouldBeNil)
			machine2 := &ufspb.Machine{
				Name: "machine-102",
				Device: &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: &ufspb.ChromeOSMachine{
						BuildTarget: "test",
						Model:       "test",
					},
				},
			}
			_, err = registration.CreateMachine(ctx, machine2)
			So(err, ShouldBeNil)
			dut1 := mockDUT("dut-16", "machine-101", "", "", "dut-16-power-1", ".A1", 0, nil, "")
			_, err = inventory.CreateMachineLSE(ctx, dut1)
			So(err, ShouldBeNil)
			dut2 := mockDUT("dut-17", "machine-102", "", "", "dut-17-power-1", ".A1", 0, nil, "")
			_, err = inventory.CreateMachineLSE(ctx, dut2)
			So(err, ShouldBeNil)
			// Update rpm powerunit_name to to dut-16-power-1 creating conflict with dut-16 rpm powerunit_name.
			dut3 := mockDUT("dut-17", "machine-102", "", "", "dut-16-power-1", ".A1", int32(0), []string{"DUT_POOL_QUOTA"}, "")
			_, err = UpdateDUT(ctx, dut3, nil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "The rpm powerunit_name and powerunit_outlet is already in use by dut-16")
		})

		Convey("UpdateDUT - RPM powerunit_name and powerunit_outlet conflict with multiple DUTs", func() {
			machine1 := &ufspb.Machine{
				Name: "machine-201",
				Device: &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: &ufspb.ChromeOSMachine{
						BuildTarget: "test",
						Model:       "test",
					},
				},
			}
			_, err := registration.CreateMachine(ctx, machine1)
			So(err, ShouldBeNil)
			machine2 := &ufspb.Machine{
				Name: "machine-202",
				Device: &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: &ufspb.ChromeOSMachine{
						BuildTarget: "test",
						Model:       "test",
					},
				},
			}
			_, err = registration.CreateMachine(ctx, machine2)
			So(err, ShouldBeNil)
			dut1 := mockDUT("dut-216", "machine-201", "", "", "dut-216-power-1", ".A1", 0, nil, "")
			_, err = inventory.CreateMachineLSE(ctx, dut1)
			So(err, ShouldBeNil)
			dut2 := mockDUT("dut-217", "machine-202", "", "", "dut-216-power-1", ".A1", 0, nil, "")
			_, err = inventory.CreateMachineLSE(ctx, dut2)
			So(err, ShouldBeNil)
			// Both dut-216 and dut-217 have same RPM info.
			// Update to dut-216 or dut-217 without change in rpm info will fail.
			dut3 := mockDUT("dut-216", "machine-201", "", "", "dut-216-power-1", ".A1", int32(0), []string{"DUT_POOL_QUOTA"}, "")
			_, err = UpdateDUT(ctx, dut3, nil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "The rpm powerunit_name and powerunit_outlet is already in use by dut-217")

			dut4 := mockDUT("dut-217", "machine-202", "", "", "dut-216-power-1", ".A1", int32(0), []string{"DUT_POOL_QUOTA"}, "")
			_, err = UpdateDUT(ctx, dut4, nil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "The rpm powerunit_name and powerunit_outlet is already in use by dut-216")
		})
		Convey("UpdateDUT - Add chameleon to DUT", func() {
			// Create a DUT with labstation.
			createValidDUTWithLabstation(ctx, "dut-13", "machine-13", "labstation-10", "machine-14")
			dut2, err := GetMachineLSE(ctx, "dut-13")
			So(err, ShouldBeNil)
			// Add chameleon to the DUT.
			dut2.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().Chameleon = &chromeosLab.Chameleon{
				ChameleonPeripherals: []chromeosLab.ChameleonType{chromeosLab.ChameleonType_CHAMELEON_TYPE_HDMI, chromeosLab.ChameleonType_CHAMELEON_TYPE_DP},
				AudioBoard:           true,
			}
			dut2.UpdateTime = nil
			// Update chameleon with correct paths.
			resp, err := UpdateDUT(ctx, dut2, mockFieldMask("dut.chameleon.type", "dut.chameleon.audioboard"))
			So(err, ShouldBeNil)
			resp.UpdateTime = nil
			// Validate the proto after update.
			So(dut2, ShouldResembleProto, resp)
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-13")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 3)
			// Chameleon type recorded.
			So(changes[1].OldValue, ShouldEqual, "[]")
			So(changes[1].NewValue, ShouldEqual, "[CHAMELEON_TYPE_HDMI CHAMELEON_TYPE_DP]")
			// Chameleon audioboard recorded.
			So(changes[2].OldValue, ShouldEqual, "false")
			So(changes[2].NewValue, ShouldEqual, "true")
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-13")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 2)
			s, err := state.GetStateRecord(ctx, "hosts/dut-13")
			So(err, ShouldBeNil)
			// State should be unchanged.
			So(s.GetState(), ShouldEqual, dut2.GetResourceState())
		})

		Convey("UpdateDUT - Remove chameleon from DUT", func() {
			// Create a DUT with labstation.
			createValidDUTWithLabstation(ctx, "dut-14", "machine-15", "labstation-11", "machine-16")
			dut2, err := GetMachineLSE(ctx, "dut-14")
			So(err, ShouldBeNil)
			// Add chameleon to the DUT.
			dut2.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().Chameleon = &chromeosLab.Chameleon{
				ChameleonPeripherals: []chromeosLab.ChameleonType{chromeosLab.ChameleonType_CHAMELEON_TYPE_HDMI, chromeosLab.ChameleonType_CHAMELEON_TYPE_DP},
				AudioBoard:           true,
			}
			dut2.UpdateTime = nil
			// Update the DUT with proper mask.
			resp, err := UpdateDUT(ctx, dut2, mockFieldMask("dut.chameleon.type", "dut.chameleon.audioboard"))
			So(err, ShouldBeNil)
			resp.UpdateTime = nil
			// Validate proto after update.
			So(dut2, ShouldResembleProto, resp)
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-14")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 3)
			So(changes[1].OldValue, ShouldEqual, "[]")
			So(changes[1].NewValue, ShouldEqual, "[CHAMELEON_TYPE_HDMI CHAMELEON_TYPE_DP]")
			So(changes[2].OldValue, ShouldEqual, "false")
			So(changes[2].NewValue, ShouldEqual, "true")
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-14")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 2)
			s, err := state.GetStateRecord(ctx, "hosts/dut-14")
			So(err, ShouldBeNil)
			// State should be unchanged.
			So(s.GetState(), ShouldEqual, dut2.GetResourceState())
			// Delete chameleon
			dut2.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().Chameleon = nil
			// UpdateDUT with at least one of the chameleon paths
			resp, err = UpdateDUT(ctx, dut2, mockFieldMask("dut.chameleon.type"))
			So(err, ShouldBeNil)
			resp.UpdateTime = nil
			// Validate proto after update.
			So(resp, ShouldResembleProto, dut2)
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-14")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 5)
			// Verify Chameleon types deleted
			So(changes[3].NewValue, ShouldEqual, "[]")
			So(changes[3].OldValue, ShouldEqual, "[CHAMELEON_TYPE_HDMI CHAMELEON_TYPE_DP]")
			// Verify audiobox reset
			So(changes[4].NewValue, ShouldEqual, "false")
			So(changes[4].OldValue, ShouldEqual, "true")
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-14")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 3)
			s, err = state.GetStateRecord(ctx, "hosts/dut-14")
			So(err, ShouldBeNil)
			// State should be unchanged.
			So(s.GetState(), ShouldEqual, dut2.GetResourceState())
			// Test the same thing but with empty chameleon struct. Expectation is that UFS ignores the empty struct given and sets chameleon to nil on delete.
			// Add chameleon to the DUT.
			dut2.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().Chameleon = &chromeosLab.Chameleon{
				ChameleonPeripherals: []chromeosLab.ChameleonType{chromeosLab.ChameleonType_CHAMELEON_TYPE_HDMI, chromeosLab.ChameleonType_CHAMELEON_TYPE_DP},
				AudioBoard:           true,
			}
			dut2.UpdateTime = nil
			// Update the DUT with proper mask.
			resp, err = UpdateDUT(ctx, dut2, mockFieldMask("dut.chameleon.type", "dut.chameleon.audioboard"))
			So(err, ShouldBeNil)
			resp.UpdateTime = nil
			// Validate proto after update.
			So(dut2, ShouldResembleProto, resp)
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-14")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 7)
			So(changes[5].OldValue, ShouldEqual, "[]")
			So(changes[5].NewValue, ShouldEqual, "[CHAMELEON_TYPE_HDMI CHAMELEON_TYPE_DP]")
			So(changes[6].OldValue, ShouldEqual, "false")
			So(changes[6].NewValue, ShouldEqual, "true")
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-14")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 4)
			s, err = state.GetStateRecord(ctx, "hosts/dut-14")
			So(err, ShouldBeNil)
			// State should be unchanged.
			So(s.GetState(), ShouldEqual, dut2.GetResourceState())
			// Delete chameleon
			dut2.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().Chameleon = &chromeosLab.Chameleon{}
			// UpdateDUT with at least one of the chameleon paths
			resp, err = UpdateDUT(ctx, dut2, mockFieldMask("dut.chameleon.type"))
			So(err, ShouldBeNil)
			resp.UpdateTime = nil
			// Chameleon should be assigned to nil and not empty struct.
			dut2.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().Chameleon = nil
			// Validate proto after update.
			So(resp, ShouldResembleProto, dut2)
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-14")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 9)
			// Verify Chameleon types deleted
			So(changes[7].NewValue, ShouldEqual, "[]")
			So(changes[7].OldValue, ShouldEqual, "[CHAMELEON_TYPE_HDMI CHAMELEON_TYPE_DP]")
			// Verify audiobox reset
			So(changes[8].NewValue, ShouldEqual, "false")
			So(changes[8].OldValue, ShouldEqual, "true")
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-14")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 5)
			s, err = state.GetStateRecord(ctx, "hosts/dut-14")
			So(err, ShouldBeNil)
			// State should be unchanged.
			So(s.GetState(), ShouldEqual, dut2.GetResourceState())

		})

		Convey("UpdateDUT - Add wifi to DUT", func() {
			// Create a DUT with labstation.
			createValidDUTWithLabstation(ctx, "dut-15", "machine-17", "labstation-12", "machine-18")
			dut2, err := GetMachineLSE(ctx, "dut-15")
			So(err, ShouldBeNil)
			// Add wifi setup to the DUT.
			dut2.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().Wifi = &chromeosLab.Wifi{
				AntennaConn: chromeosLab.Wifi_CONN_CONDUCTIVE,
				Router:      chromeosLab.Wifi_ROUTER_802_11AX,
				Wificell:    true,
			}
			dut2.UpdateTime = nil
			// Update DUT with proper paths.
			resp, err := UpdateDUT(ctx, dut2, mockFieldMask("dut.wifi.antennaconn", "dut.wifi.router", "dut.wifi.wificell"))
			So(err, ShouldBeNil)
			resp.UpdateTime = nil
			// Verify that the returned proto is updated.
			So(dut2, ShouldResembleProto, resp)
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-15")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 4)
			// Verify all changes recorded by the history.
			So(changes[1].OldValue, ShouldEqual, "CONN_UNKNOWN")
			So(changes[1].NewValue, ShouldEqual, "CONN_CONDUCTIVE")
			So(changes[2].OldValue, ShouldEqual, "false")
			So(changes[2].NewValue, ShouldEqual, "true")
			So(changes[3].OldValue, ShouldEqual, "ROUTER_UNSPECIFIED")
			So(changes[3].NewValue, ShouldEqual, "ROUTER_802_11AX")
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-15")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 2)
			s, err := state.GetStateRecord(ctx, "hosts/dut-15")
			So(err, ShouldBeNil)
			// State should be unchanged.
			So(s.GetState(), ShouldEqual, dut2.GetResourceState())
		})

		Convey("UpdateDUT - Remove wifi from DUT", func() {
			// Create a DUT with labstation.
			createValidDUTWithLabstation(ctx, "dut-160", "machine-19", "labstation-13", "machine-21")
			dut2, err := GetMachineLSE(ctx, "dut-160")
			So(err, ShouldBeNil)
			// Add wifi to the DUT.
			dut2.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().Wifi = &chromeosLab.Wifi{
				AntennaConn: chromeosLab.Wifi_CONN_CONDUCTIVE,
				Router:      chromeosLab.Wifi_ROUTER_802_11AX,
				Wificell:    true,
			}
			dut2.UpdateTime = nil
			// Update DUT with proper paths.
			resp, err := UpdateDUT(ctx, dut2, mockFieldMask("dut.wifi.antennaconn", "dut.wifi.router", "dut.wifi.wificell"))
			So(err, ShouldBeNil)
			resp.UpdateTime = nil
			// Verify the proto returned.
			So(dut2, ShouldResembleProto, resp)
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-160")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 4)
			// Verify that the changes were recorded by the history.
			So(changes[1].OldValue, ShouldEqual, "CONN_UNKNOWN")
			So(changes[1].NewValue, ShouldEqual, "CONN_CONDUCTIVE")
			So(changes[2].OldValue, ShouldEqual, "false")
			So(changes[2].NewValue, ShouldEqual, "true")
			So(changes[3].OldValue, ShouldEqual, "ROUTER_UNSPECIFIED")
			So(changes[3].NewValue, ShouldEqual, "ROUTER_802_11AX")
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-160")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 2)
			s, err := state.GetStateRecord(ctx, "hosts/dut-160")
			So(err, ShouldBeNil)
			// State should be unchanged.
			So(s.GetState(), ShouldEqual, dut2.GetResourceState())
			// Delete/reset wifi in DUT
			dut2.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().Wifi = nil
			// Update DUT with at least one of the wifi masks.
			resp, err = UpdateDUT(ctx, dut2, mockFieldMask("dut.wifi.antennaconn"))
			So(err, ShouldBeNil)
			resp.UpdateTime = nil
			So(resp, ShouldResembleProto, dut2)
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-160")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 7)
			// Verify that the changes were recorded by the history.
			So(changes[4].NewValue, ShouldEqual, "CONN_UNKNOWN")
			So(changes[4].OldValue, ShouldEqual, "CONN_CONDUCTIVE")
			So(changes[5].NewValue, ShouldEqual, "false")
			So(changes[5].OldValue, ShouldEqual, "true")
			So(changes[6].NewValue, ShouldEqual, "ROUTER_UNSPECIFIED")
			So(changes[6].OldValue, ShouldEqual, "ROUTER_802_11AX")
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-160")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 3)
			s, err = state.GetStateRecord(ctx, "hosts/dut-160")
			So(err, ShouldBeNil)
			// State should be unchanged.
			So(s.GetState(), ShouldEqual, dut2.GetResourceState())
		})

		Convey("UpdateDUT - Add carrier to DUT", func() {
			// Create a DUT with labstation.
			createValidDUTWithLabstation(ctx, "dut-170", "machine-22", "labstation-14", "machine-23")
			dut2, err := GetMachineLSE(ctx, "dut-170")
			So(err, ShouldBeNil)
			// Add carrier to the DUT.
			dut2.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().Carrier = "GenericCarrier"
			dut2.UpdateTime = nil
			// Update DUT with proper paths.
			resp, err := UpdateDUT(ctx, dut2, mockFieldMask("dut.carrier"))
			So(err, ShouldBeNil)
			resp.UpdateTime = nil
			// Verify that the returned proto is updated.
			So(dut2, ShouldResembleProto, resp)
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-170")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 2)
			// Verify all changes recorded by the history.
			So(changes[1].OldValue, ShouldEqual, "")
			So(changes[1].NewValue, ShouldEqual, "GenericCarrier")
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-170")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 2)
			s, err := state.GetStateRecord(ctx, "hosts/dut-170")
			So(err, ShouldBeNil)
			// State should be unchanged.
			So(s.GetState(), ShouldEqual, dut2.GetResourceState())
		})

		Convey("UpdateDUT - Remove carrier from DUT", func() {
			// Create a DUT with labstation.
			createValidDUTWithLabstation(ctx, "dut-18", "machine-24", "labstation-15", "machine-25")
			dut2, err := GetMachineLSE(ctx, "dut-18")
			So(err, ShouldBeNil)
			// Add carrier to the DUT.
			dut2.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().Carrier = "GenericCarrier"
			dut2.UpdateTime = nil
			// Update DUT with proper paths.
			resp, err := UpdateDUT(ctx, dut2, mockFieldMask("dut.carrier"))
			So(err, ShouldBeNil)
			resp.UpdateTime = nil
			// Verify that the returned proto is updated.
			So(dut2, ShouldResembleProto, resp)
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-18")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 2)
			// Verify all changes recorded by the history.
			So(changes[1].OldValue, ShouldEqual, "")
			So(changes[1].NewValue, ShouldEqual, "GenericCarrier")
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-18")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 2)
			s, err := state.GetStateRecord(ctx, "hosts/dut-18")
			So(err, ShouldBeNil)
			// State should be unchanged.
			So(s.GetState(), ShouldEqual, dut2.GetResourceState())
			// Delete/reset carrier in DUT
			dut2.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().Carrier = ""
			// Update DUT with carrier mask.
			resp, err = UpdateDUT(ctx, dut2, mockFieldMask("dut.carrier"))
			So(err, ShouldBeNil)
			resp.UpdateTime = nil
			So(resp, ShouldResembleProto, dut2)
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-18")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 3)
			// Verify that the changes were recorded by the history.
			So(changes[2].NewValue, ShouldEqual, "")
			So(changes[2].OldValue, ShouldEqual, "GenericCarrier")
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-18")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 3)
			s, err = state.GetStateRecord(ctx, "hosts/dut-18")
			So(err, ShouldBeNil)
			// State should be unchanged.
			So(s.GetState(), ShouldEqual, dut2.GetResourceState())
		})

		Convey("UpdateDUT - Set chaos on DUT", func() {
			// Create a DUT with labstation.
			createValidDUTWithLabstation(ctx, "dut-19", "machine-26", "labstation-16", "machine-27")
			dut2, err := GetMachineLSE(ctx, "dut-19")
			So(err, ShouldBeNil)
			// Add chaos to the DUT.
			dut2.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().Chaos = true
			dut2.UpdateTime = nil
			// Update DUT with proper paths.
			resp, err := UpdateDUT(ctx, dut2, mockFieldMask("dut.chaos"))
			So(err, ShouldBeNil)
			resp.UpdateTime = nil
			// Verify that the returned proto is updated.
			So(dut2, ShouldResembleProto, resp)
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-19")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 2)
			// Verify all changes recorded by the history.
			So(changes[1].OldValue, ShouldEqual, "false")
			So(changes[1].NewValue, ShouldEqual, "true")
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-19")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 2)
			s, err := state.GetStateRecord(ctx, "hosts/dut-19")
			So(err, ShouldBeNil)
			// State should be unchanged.
			So(s.GetState(), ShouldEqual, dut2.GetResourceState())
		})

		Convey("UpdateDUT - Reset chaos on DUT", func() {
			// Create a DUT with labstation.
			createValidDUTWithLabstation(ctx, "dut-20", "machine-28", "labstation-17", "machine-29")
			dut2, err := GetMachineLSE(ctx, "dut-20")
			So(err, ShouldBeNil)
			// Set chaos on the DUT.
			dut2.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().Chaos = true
			dut2.UpdateTime = nil
			// Update DUT with proper paths.
			resp, err := UpdateDUT(ctx, dut2, mockFieldMask("dut.chaos"))
			So(err, ShouldBeNil)
			resp.UpdateTime = nil
			// Verify that the returned proto is updated.
			So(dut2, ShouldResembleProto, resp)
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-20")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 2)
			// Verify all changes recorded by the history.
			So(changes[1].OldValue, ShouldEqual, "false")
			So(changes[1].NewValue, ShouldEqual, "true")
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-20")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 2)
			s, err := state.GetStateRecord(ctx, "hosts/dut-20")
			So(err, ShouldBeNil)
			// State should be unchanged.
			So(s.GetState(), ShouldEqual, dut2.GetResourceState())
			// reset chaos in DUT
			dut2.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().Chaos = false
			// Update DUT with chaos mask.
			resp, err = UpdateDUT(ctx, dut2, mockFieldMask("dut.chaos"))
			So(err, ShouldBeNil)
			resp.UpdateTime = nil
			So(resp, ShouldResembleProto, dut2)
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-20")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 3)
			// Verify that the changes were recorded by the history.
			So(changes[2].NewValue, ShouldEqual, "false")
			So(changes[2].OldValue, ShouldEqual, "true")
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-20")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 3)
			s, err = state.GetStateRecord(ctx, "hosts/dut-20")
			So(err, ShouldBeNil)
			// State should be unchanged.
			So(s.GetState(), ShouldEqual, dut2.GetResourceState())
		})

		Convey("UpdateDUT - Set usb smarthub on DUT", func() {
			// Create a DUT with labstation.
			createValidDUTWithLabstation(ctx, "dut-21", "machine-31", "labstation-18", "machine-32")
			dut2, err := GetMachineLSE(ctx, "dut-21")
			So(err, ShouldBeNil)
			// Set usb smarthub on DUT.
			dut2.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().SmartUsbhub = true
			dut2.UpdateTime = nil
			// Update DUT with proper paths.
			resp, err := UpdateDUT(ctx, dut2, mockFieldMask("dut.usb.smarthub"))
			So(err, ShouldBeNil)
			resp.UpdateTime = nil
			// Verify that the returned proto is updated.
			So(dut2, ShouldResembleProto, resp)
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-21")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 2)
			// Verify all changes recorded by the history.
			So(changes[1].OldValue, ShouldEqual, "false")
			So(changes[1].NewValue, ShouldEqual, "true")
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-21")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 2)
			s, err := state.GetStateRecord(ctx, "hosts/dut-21")
			So(err, ShouldBeNil)
			// State should be unchanged.
			So(s.GetState(), ShouldEqual, dut2.GetResourceState())
		})

		Convey("UpdateDUT - Reset usb smarthub on DUT", func() {
			// Create a DUT with labstation.
			createValidDUTWithLabstation(ctx, "dut-22", "machine-33", "labstation-19", "machine-34")
			dut2, err := GetMachineLSE(ctx, "dut-22")
			So(err, ShouldBeNil)
			// Set smart usb hub on the DUT.
			dut2.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().SmartUsbhub = true
			dut2.UpdateTime = nil
			// Update DUT with proper paths.
			resp, err := UpdateDUT(ctx, dut2, mockFieldMask("dut.usb.smarthub"))
			So(err, ShouldBeNil)
			resp.UpdateTime = nil
			// Verify that the returned proto is updated.
			So(dut2, ShouldResembleProto, resp)
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-22")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 2)
			// Verify all changes recorded by the history.
			So(changes[1].OldValue, ShouldEqual, "false")
			So(changes[1].NewValue, ShouldEqual, "true")
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-22")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 2)
			s, err := state.GetStateRecord(ctx, "hosts/dut-22")
			So(err, ShouldBeNil)
			// State should be unchanged.
			So(s.GetState(), ShouldEqual, dut2.GetResourceState())
			// reset usb smart hub on DUT
			dut2.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().SmartUsbhub = false
			// Update DUT with usb smart hub mask.
			resp, err = UpdateDUT(ctx, dut2, mockFieldMask("dut.usb.smarthub"))
			So(err, ShouldBeNil)
			resp.UpdateTime = nil
			So(resp, ShouldResembleProto, dut2)
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-22")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 3)
			// Verify that the changes were recorded by the history.
			So(changes[2].NewValue, ShouldEqual, "false")
			So(changes[2].OldValue, ShouldEqual, "true")
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-22")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 3)
			s, err = state.GetStateRecord(ctx, "hosts/dut-22")
			So(err, ShouldBeNil)
			// State should be unchanged.
			So(s.GetState(), ShouldEqual, dut2.GetResourceState())
		})

		Convey("UpdateDUT - Add camera to DUT", func() {
			// Create a DUT with labstation.
			createValidDUTWithLabstation(ctx, "dut-23", "machine-35", "labstation-20", "machine-36")
			dut2, err := GetMachineLSE(ctx, "dut-23")
			So(err, ShouldBeNil)
			// Add camera to DUT.
			dut2.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().ConnectedCamera = []*chromeosLab.Camera{
				{
					CameraType: chromeosLab.CameraType_CAMERA_HUDDLY,
				},
			}
			dut2.UpdateTime = nil
			// Update DUT with proper paths.
			resp, err := UpdateDUT(ctx, dut2, mockFieldMask("dut.camera.type"))
			So(err, ShouldBeNil)
			resp.UpdateTime = nil
			// Verify that the returned proto is updated.
			So(dut2, ShouldResembleProto, resp)
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-23")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 2)
			// Verify all changes recorded by the history.
			So(changes[1].OldValue, ShouldEqual, "[]")
			So(changes[1].NewValue, ShouldEqual, "[camera_type:CAMERA_HUDDLY]")
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-23")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 2)
			s, err := state.GetStateRecord(ctx, "hosts/dut-23")
			So(err, ShouldBeNil)
			// State should be unchanged.
			So(s.GetState(), ShouldEqual, dut2.GetResourceState())
		})

		Convey("UpdateDUT - Remove camera from DUT", func() {
			// Create a DUT with labstation.
			createValidDUTWithLabstation(ctx, "dut-24", "machine-37", "labstation-21", "machine-38")
			dut2, err := GetMachineLSE(ctx, "dut-24")
			So(err, ShouldBeNil)
			// Add camera to DUT.
			dut2.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().ConnectedCamera = []*chromeosLab.Camera{
				{
					CameraType: chromeosLab.CameraType_CAMERA_HUDDLY,
				},
			}
			dut2.UpdateTime = nil
			// Update DUT with proper paths.
			resp, err := UpdateDUT(ctx, dut2, mockFieldMask("dut.camera.type"))
			So(err, ShouldBeNil)
			resp.UpdateTime = nil
			// Verify that the returned proto is updated.
			So(dut2, ShouldResembleProto, resp)
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-24")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 2)
			// Verify all changes recorded by the history.
			So(changes[1].OldValue, ShouldEqual, "[]")
			So(changes[1].NewValue, ShouldEqual, "[camera_type:CAMERA_HUDDLY]")
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-24")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 2)
			s, err := state.GetStateRecord(ctx, "hosts/dut-24")
			So(err, ShouldBeNil)
			// State should be unchanged.
			So(s.GetState(), ShouldEqual, dut2.GetResourceState())
			// Remove camera from DUT.
			dut2.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().ConnectedCamera = nil
			// Update DUT with camera type mask.
			resp, err = UpdateDUT(ctx, dut2, mockFieldMask("dut.camera.type"))
			So(err, ShouldBeNil)
			resp.UpdateTime = nil
			So(resp, ShouldResembleProto, dut2)
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-24")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 3)
			// Verify that the changes were recorded by the history.
			So(changes[2].NewValue, ShouldEqual, "[]")
			So(changes[2].OldValue, ShouldEqual, "[camera_type:CAMERA_HUDDLY]")
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-24")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 3)
			s, err = state.GetStateRecord(ctx, "hosts/dut-24")
			So(err, ShouldBeNil)
			// State should be unchanged.
			So(s.GetState(), ShouldEqual, dut2.GetResourceState())
		})

		Convey("UpdateDUT - Add cable to DUT", func() {
			// Create a DUT with labstation.
			createValidDUTWithLabstation(ctx, "dut-25", "machine-39", "labstation-22", "machine-41")
			dut2, err := GetMachineLSE(ctx, "dut-25")
			So(err, ShouldBeNil)
			// Add cable to DUT.
			dut2.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().Cable = []*chromeosLab.Cable{
				{
					Type: chromeosLab.CableType_CABLE_USBAUDIO,
				},
				{
					Type: chromeosLab.CableType_CABLE_USBPRINTING,
				},
			}
			dut2.UpdateTime = nil
			// Update DUT with proper paths.
			resp, err := UpdateDUT(ctx, dut2, mockFieldMask("dut.cable.type"))
			So(err, ShouldBeNil)
			resp.UpdateTime = nil
			// Verify that the returned proto is updated.
			So(dut2, ShouldResembleProto, resp)
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-25")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 2)
			// Verify all changes recorded by the history.
			So(changes[1].OldValue, ShouldEqual, "[]")
			So(changes[1].NewValue, ShouldEqual, "[type:CABLE_USBAUDIO type:CABLE_USBPRINTING]")
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-25")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 2)
			s, err := state.GetStateRecord(ctx, "hosts/dut-25")
			So(err, ShouldBeNil)
			// State should be unchanged.
			So(s.GetState(), ShouldEqual, dut2.GetResourceState())
		})

		Convey("UpdateDUT - Remove cables from DUT", func() {
			// Create a DUT with labstation.
			createValidDUTWithLabstation(ctx, "dut-26", "machine-42", "labstation-23", "machine-43")
			dut2, err := GetMachineLSE(ctx, "dut-26")
			So(err, ShouldBeNil)
			// Add cable to DUT.
			dut2.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().Cable = []*chromeosLab.Cable{
				{
					Type: chromeosLab.CableType_CABLE_USBAUDIO,
				},
				{
					Type: chromeosLab.CableType_CABLE_USBPRINTING,
				},
			}
			dut2.UpdateTime = nil
			// Update DUT with proper paths.
			resp, err := UpdateDUT(ctx, dut2, mockFieldMask("dut.cable.type"))
			So(err, ShouldBeNil)
			resp.UpdateTime = nil
			// Verify that the returned proto is updated.
			So(dut2, ShouldResembleProto, resp)
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-26")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 2)
			// Verify all changes recorded by the history.
			So(changes[1].OldValue, ShouldEqual, "[]")
			So(changes[1].NewValue, ShouldEqual, "[type:CABLE_USBAUDIO type:CABLE_USBPRINTING]")
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-26")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 2)
			s, err := state.GetStateRecord(ctx, "hosts/dut-26")
			So(err, ShouldBeNil)
			// State should be unchanged.
			So(s.GetState(), ShouldEqual, dut2.GetResourceState())
			// Remove cables from DUT.
			dut2.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().Cable = nil
			// Update DUT with camera type mask.
			resp, err = UpdateDUT(ctx, dut2, mockFieldMask("dut.cable.type"))
			So(err, ShouldBeNil)
			resp.UpdateTime = nil
			So(resp, ShouldResembleProto, dut2)
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-26")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 3)
			// Verify that the changes were recorded by the history.
			So(changes[2].NewValue, ShouldEqual, "[]")
			So(changes[2].OldValue, ShouldEqual, "[type:CABLE_USBAUDIO type:CABLE_USBPRINTING]")
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-26")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 3)
			s, err = state.GetStateRecord(ctx, "hosts/dut-26")
			So(err, ShouldBeNil)
			// State should be unchanged.
			So(s.GetState(), ShouldEqual, dut2.GetResourceState())
		})

		Convey("UpdateDUT - Set touch mimo on DUT", func() {
			// Create a DUT with labstation.
			createValidDUTWithLabstation(ctx, "dut-27", "machine-44", "labstation-24", "machine-45")
			dut2, err := GetMachineLSE(ctx, "dut-27")
			So(err, ShouldBeNil)
			// Set touch mimo on DUT.
			dut2.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().Touch = &chromeosLab.Touch{
				Mimo: true,
			}
			dut2.UpdateTime = nil
			// Update DUT with proper paths.
			resp, err := UpdateDUT(ctx, dut2, mockFieldMask("dut.touch.mimo"))
			So(err, ShouldBeNil)
			resp.UpdateTime = nil
			// Verify that the returned proto is updated.
			So(dut2, ShouldResembleProto, resp)
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-27")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 2)
			// Verify all changes recorded by the history.
			So(changes[1].OldValue, ShouldEqual, "<nil>")
			So(changes[1].NewValue, ShouldEqual, "mimo:true")
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-27")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 2)
			s, err := state.GetStateRecord(ctx, "hosts/dut-27")
			So(err, ShouldBeNil)
			// State should be unchanged.
			So(s.GetState(), ShouldEqual, dut2.GetResourceState())
		})

		Convey("UpdateDUT - Reset touch mimo on DUT", func() {
			// Create a DUT with labstation.
			createValidDUTWithLabstation(ctx, "dut-28", "machine-46", "labstation-25", "machine-47")
			dut2, err := GetMachineLSE(ctx, "dut-28")
			So(err, ShouldBeNil)
			// Set touch mimo on DUT.
			dut2.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().Touch = &chromeosLab.Touch{
				Mimo: true,
			}
			dut2.UpdateTime = nil
			// Update DUT with proper paths.
			resp, err := UpdateDUT(ctx, dut2, mockFieldMask("dut.touch.mimo"))
			So(err, ShouldBeNil)
			resp.UpdateTime = nil
			// Verify that the returned proto is updated.
			So(dut2, ShouldResembleProto, resp)
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-28")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 2)
			// Verify all changes recorded by the history.
			So(changes[1].OldValue, ShouldEqual, "<nil>")
			So(changes[1].NewValue, ShouldEqual, "mimo:true")
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-28")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 2)
			s, err := state.GetStateRecord(ctx, "hosts/dut-28")
			So(err, ShouldBeNil)
			// State should be unchanged.
			So(s.GetState(), ShouldEqual, dut2.GetResourceState())
			// Remove cables from DUT.
			dut2.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().Touch = nil
			// Update DUT with camera type mask.
			resp, err = UpdateDUT(ctx, dut2, mockFieldMask("dut.touch.mimo"))
			So(err, ShouldBeNil)
			resp.UpdateTime = nil
			So(resp, ShouldResembleProto, dut2)
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-28")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 3)
			// Verify that the changes were recorded by the history.
			So(changes[2].NewValue, ShouldEqual, "<nil>")
			So(changes[2].OldValue, ShouldEqual, "mimo:true")
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-28")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 3)
			s, err = state.GetStateRecord(ctx, "hosts/dut-28")
			So(err, ShouldBeNil)
			// State should be unchanged.
			So(s.GetState(), ShouldEqual, dut2.GetResourceState())
		})

		Convey("UpdateDUT - Set camera box on DUT", func() {
			// Create a DUT with labstation.
			createValidDUTWithLabstation(ctx, "dut-29", "machine-48", "labstation-26", "machine-49")
			dut2, err := GetMachineLSE(ctx, "dut-29")
			So(err, ShouldBeNil)
			// Set camera box on DUT.
			dut2.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().Camerabox = true
			dut2.UpdateTime = nil
			// Update DUT with proper paths.
			resp, err := UpdateDUT(ctx, dut2, mockFieldMask("dut.camerabox"))
			So(err, ShouldBeNil)
			resp.UpdateTime = nil
			// Verify that the returned proto is updated.
			So(dut2, ShouldResembleProto, resp)
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-29")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 2)
			// Verify all changes recorded by the history.
			So(changes[1].OldValue, ShouldEqual, "false")
			So(changes[1].NewValue, ShouldEqual, "true")
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-29")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 2)
			s, err := state.GetStateRecord(ctx, "hosts/dut-29")
			So(err, ShouldBeNil)
			// State should be unchanged.
			So(s.GetState(), ShouldEqual, dut2.GetResourceState())
		})

		Convey("UpdateDUT - Reset camera box on DUT", func() {
			// Create a DUT with labstation.
			createValidDUTWithLabstation(ctx, "dut-31", "machine-51", "labstation-27", "machine-52")
			dut2, err := GetMachineLSE(ctx, "dut-31")
			So(err, ShouldBeNil)
			// Set camera box on DUT.
			dut2.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().Camerabox = true
			dut2.UpdateTime = nil
			// Update DUT with proper paths.
			resp, err := UpdateDUT(ctx, dut2, mockFieldMask("dut.camerabox"))
			So(err, ShouldBeNil)
			resp.UpdateTime = nil
			// Verify that the returned proto is updated.
			So(dut2, ShouldResembleProto, resp)
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-31")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 2)
			// Verify all changes recorded by the history.
			So(changes[1].OldValue, ShouldEqual, "false")
			So(changes[1].NewValue, ShouldEqual, "true")
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-31")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 2)
			s, err := state.GetStateRecord(ctx, "hosts/dut-31")
			So(err, ShouldBeNil)
			// State should be unchanged.
			So(s.GetState(), ShouldEqual, dut2.GetResourceState())
			// Remove cables from DUT.
			dut2.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().Camerabox = false
			// Update DUT with camera type mask.
			resp, err = UpdateDUT(ctx, dut2, mockFieldMask("dut.camerabox"))
			So(err, ShouldBeNil)
			resp.UpdateTime = nil
			So(resp, ShouldResembleProto, dut2)
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-31")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 3)
			// Verify that the changes were recorded by the history.
			So(changes[2].NewValue, ShouldEqual, "false")
			So(changes[2].OldValue, ShouldEqual, "true")
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-31")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 3)
			s, err = state.GetStateRecord(ctx, "hosts/dut-31")
			So(err, ShouldBeNil)
			// State should be unchanged.
			So(s.GetState(), ShouldEqual, dut2.GetResourceState())
		})

		Convey("UpdateDUT - Add audio to DUT", func() {
			// Create a DUT with labstation.
			createValidDUTWithLabstation(ctx, "dut-32", "machine-53", "labstation-28", "machine-54")
			dut2, err := GetMachineLSE(ctx, "dut-32")
			So(err, ShouldBeNil)
			// Add audio config to DUT
			dut2.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().Audio = &chromeosLab.Audio{
				AudioBox:   true,
				Atrus:      true,
				AudioCable: true,
			}
			dut2.UpdateTime = nil
			// Update DUT with proper paths.
			resp, err := UpdateDUT(ctx, dut2, mockFieldMask("dut.audio.box", "dut.audio.atrus", "dut.audio.cable"))
			So(err, ShouldBeNil)
			resp.UpdateTime = nil
			// Verify that the returned proto is updated.
			So(dut2, ShouldResembleProto, resp)
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-32")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 4)
			// Verify all changes recorded by the history.
			So(changes[1].OldValue, ShouldEqual, "false")
			So(changes[1].NewValue, ShouldEqual, "true")
			So(changes[2].OldValue, ShouldEqual, "false")
			So(changes[2].NewValue, ShouldEqual, "true")
			So(changes[3].OldValue, ShouldEqual, "false")
			So(changes[3].NewValue, ShouldEqual, "true")
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-32")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 2)
			s, err := state.GetStateRecord(ctx, "hosts/dut-32")
			So(err, ShouldBeNil)
			// State should be unchanged.
			So(s.GetState(), ShouldEqual, dut2.GetResourceState())
		})

		Convey("UpdateDUT - Delete audio on DUT", func() {
			// Create a DUT with labstation.
			createValidDUTWithLabstation(ctx, "dut-33", "machine-57", "labstation-29", "machine-56")
			dut2, err := GetMachineLSE(ctx, "dut-33")
			So(err, ShouldBeNil)
			// Add audio config to DUT
			dut2.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().Audio = &chromeosLab.Audio{
				AudioBox:   true,
				Atrus:      true,
				AudioCable: true,
			}
			dut2.UpdateTime = nil
			// Update DUT with proper paths.
			resp, err := UpdateDUT(ctx, dut2, mockFieldMask("dut.audio.box", "dut.audio.atrus", "dut.audio.cable"))
			So(err, ShouldBeNil)
			resp.UpdateTime = nil
			// Verify that the returned proto is updated.
			So(dut2, ShouldResembleProto, resp)
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-33")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 4)
			// Verify all changes recorded by the history.
			So(changes[1].OldValue, ShouldEqual, "false")
			So(changes[1].NewValue, ShouldEqual, "true")
			So(changes[2].OldValue, ShouldEqual, "false")
			So(changes[2].NewValue, ShouldEqual, "true")
			So(changes[3].OldValue, ShouldEqual, "false")
			So(changes[3].NewValue, ShouldEqual, "true")
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-33")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 2)
			s, err := state.GetStateRecord(ctx, "hosts/dut-33")
			So(err, ShouldBeNil)
			// State should be unchanged.
			So(s.GetState(), ShouldEqual, dut2.GetResourceState())
			// Remove audio from DUT.
			dut2.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().Audio = nil
			// Update DUT with correct mask. Note just one mask is enough to delete audio if Audio = nil
			resp, err = UpdateDUT(ctx, dut2, mockFieldMask("dut.audio.box"))
			So(err, ShouldBeNil)
			resp.UpdateTime = nil
			So(resp, ShouldResembleProto, dut2)
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-33")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 7)
			// Verify that the changes were recorded by the history.
			So(changes[4].NewValue, ShouldEqual, "false")
			So(changes[4].OldValue, ShouldEqual, "true")
			So(changes[5].NewValue, ShouldEqual, "false")
			So(changes[5].OldValue, ShouldEqual, "true")
			So(changes[6].NewValue, ShouldEqual, "false")
			So(changes[6].OldValue, ShouldEqual, "true")
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-33")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 3)
			s, err = state.GetStateRecord(ctx, "hosts/dut-33")
			So(err, ShouldBeNil)
			// State should be unchanged.
			So(s.GetState(), ShouldEqual, dut2.GetResourceState())
		})

		Convey("UpdateDUT - Add camerabox to DUT", func() {
			// Create a DUT with labstation.
			createValidDUTWithLabstation(ctx, "dut-34", "machine-58", "labstation-30", "machine-59")
			dut2, err := GetMachineLSE(ctx, "dut-34")
			So(err, ShouldBeNil)
			// Add camerabox config to DUT
			dut2.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().CameraboxInfo = &chromeosLab.Camerabox{
				Light:  chromeosLab.Camerabox_LIGHT_LED,
				Facing: chromeosLab.Camerabox_FACING_BACK,
			}
			dut2.UpdateTime = nil
			// Update DUT with proper paths.
			resp, err := UpdateDUT(ctx, dut2, mockFieldMask("dut.camerabox.facing", "dut.camerabox.light"))
			So(err, ShouldBeNil)
			resp.UpdateTime = nil
			// Verify that the returned proto is updated.
			So(dut2, ShouldResembleProto, resp)
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-34")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 3)
			// Verify all changes recorded by the history.
			So(changes[1].OldValue, ShouldEqual, "FACING_UNKNOWN")
			So(changes[1].NewValue, ShouldEqual, "FACING_BACK")
			So(changes[2].OldValue, ShouldEqual, "LIGHT_UNKNOWN")
			So(changes[2].NewValue, ShouldEqual, "LIGHT_LED")
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-34")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 2)
			s, err := state.GetStateRecord(ctx, "hosts/dut-34")
			So(err, ShouldBeNil)
			// State should be unchanged.
			So(s.GetState(), ShouldEqual, dut2.GetResourceState())
		})

		Convey("UpdateDUT - Delete camerabox on DUT", func() {
			// Create a DUT with labstation.
			createValidDUTWithLabstation(ctx, "dut-35", "machine-61", "labstation-31", "machine-62")
			dut2, err := GetMachineLSE(ctx, "dut-35")
			So(err, ShouldBeNil)
			// Add camerabox config to DUT
			dut2.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().CameraboxInfo = &chromeosLab.Camerabox{
				Light:  chromeosLab.Camerabox_LIGHT_NOLED,
				Facing: chromeosLab.Camerabox_FACING_FRONT,
			}
			dut2.UpdateTime = nil
			// Update DUT with proper paths.
			resp, err := UpdateDUT(ctx, dut2, mockFieldMask("dut.camerabox.facing", "dut.camerabox.light"))
			So(err, ShouldBeNil)
			resp.UpdateTime = nil
			// Verify that the returned proto is updated.
			So(dut2, ShouldResembleProto, resp)
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-35")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 3)
			// Verify all changes recorded by the history.
			So(changes[1].OldValue, ShouldEqual, "FACING_UNKNOWN")
			So(changes[1].NewValue, ShouldEqual, "FACING_FRONT")
			So(changes[2].OldValue, ShouldEqual, "LIGHT_UNKNOWN")
			So(changes[2].NewValue, ShouldEqual, "LIGHT_NOLED")
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-35")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 2)
			s, err := state.GetStateRecord(ctx, "hosts/dut-35")
			So(err, ShouldBeNil)
			// State should be unchanged.
			So(s.GetState(), ShouldEqual, dut2.GetResourceState())
			// Remove audio from DUT.
			dut2.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().CameraboxInfo = nil
			// Update DUT with correct mask. Note just one mask is enough to delete camerabox.
			resp, err = UpdateDUT(ctx, dut2, mockFieldMask("dut.camerabox.facing"))
			So(err, ShouldBeNil)
			resp.UpdateTime = nil
			So(resp, ShouldResembleProto, dut2)
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-35")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 5)
			// Verify that the changes were recorded by the history.
			So(changes[3].NewValue, ShouldEqual, "FACING_UNKNOWN")
			So(changes[3].OldValue, ShouldEqual, "FACING_FRONT")
			So(changes[4].NewValue, ShouldEqual, "LIGHT_UNKNOWN")
			So(changes[4].OldValue, ShouldEqual, "LIGHT_NOLED")
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-35")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 3)
			s, err = state.GetStateRecord(ctx, "hosts/dut-35")
			So(err, ShouldBeNil)
			// State should be unchanged.
			So(s.GetState(), ShouldEqual, dut2.GetResourceState())
		})
		Convey("UpdateDUT - Replace bad servo with good one", func() {
			// Create a DUT with labstation.
			createValidDUTWithLabstation(ctx, "dut-36", "machine-71", "labstation-32", "machine-72")
			dut2, err := GetMachineLSE(ctx, "dut-36")
			So(err, ShouldBeNil)
			dut2.UpdateTime = nil
			// Delete the labstation
			err = inventory.DeleteMachineLSE(ctx, "labstation-32")
			So(err, ShouldBeNil)
			// Create another valid labstation
			labstation1 := mockLabstation("labstation-33", "machine-72")
			_, err = CreateMachineLSE(ctx, labstation1, nil)
			So(err, ShouldBeNil)
			// Update servo on DUT to point to new labstation
			dut2.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().Servo = &chromeosLab.Servo{
				ServoHostname: "labstation-33",
				ServoPort:     int32(9990),
				ServoSerial:   "serial-2",
			}
			resp, err := UpdateDUT(ctx, dut2, mockFieldMask("dut.servo.hostname", "dut.servo.port", "dut.servo.serial"))
			So(err, ShouldBeNil)
			resp.UpdateTime = nil
			So(resp, ShouldResembleProto, dut2)
			// Check the servo changes were recorded.
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-36")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 4)
			// Verify that the changes were recorded by the history.
			So(changes[1].NewValue, ShouldEqual, "labstation-33")
			So(changes[1].OldValue, ShouldEqual, "labstation-32")
			So(changes[2].NewValue, ShouldEqual, "9990")
			So(changes[2].OldValue, ShouldEqual, "9999")
			So(changes[3].NewValue, ShouldEqual, "serial-2")
			So(changes[3].OldValue, ShouldEqual, "serial-1")
			// Two snapshots, one at registration another at update
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-36")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 2)
			s, err := state.GetStateRecord(ctx, "hosts/dut-36")
			So(err, ShouldBeNil)
			// State should be unchanged.
			So(s.GetState(), ShouldEqual, dut2.GetResourceState())
		})
		Convey("UpdateDUT - Replace servo on misconfigured labstation (serial conflict)", func() {
			// Create a DUT with labstation.
			createValidDUTWithLabstation(ctx, "dut-37", "machine-81", "labstation-34", "machine82")
			dut2, err := GetMachineLSE(ctx, "dut-37")
			So(err, ShouldBeNil)
			servo := dut2.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetServo()
			// Misconfigure labstation by deleting servo entry in DUT but not labstation
			dut2.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().Servo = nil
			_, err = inventory.BatchUpdateMachineLSEs(ctx, []*ufspb.MachineLSE{dut2})
			So(err, ShouldBeNil)
			// Add the same servo back (serial number conflict on labstation)
			servo.ServoPort = int32(9977)
			dut2.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().Servo = servo
			resp, err := UpdateDUT(ctx, dut2, mockFieldMask("dut.servo.hostname", "dut.servo.port", "dut.servo.serial"))
			So(err, ShouldBeNil)
			resp.UpdateTime = nil
			dut2.UpdateTime = nil
			So(resp, ShouldResembleProto, dut2)
			// Check the servo changes were recorded.
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-37")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 4)
			// Verify that the changes were recorded by the history.
			So(changes[1].NewValue, ShouldEqual, "labstation-34")
			So(changes[1].OldValue, ShouldEqual, "")
			So(changes[2].NewValue, ShouldEqual, "9977")
			So(changes[2].OldValue, ShouldEqual, "0")
			So(changes[3].NewValue, ShouldEqual, "serial-1")
			So(changes[3].OldValue, ShouldEqual, "")
			// Two snapshots, one at registration another at update
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-37")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 2)
			s, err := state.GetStateRecord(ctx, "hosts/dut-37")
			So(err, ShouldBeNil)
			// State should be unchanged.
			So(s.GetState(), ShouldEqual, dut2.GetResourceState())
		})
		Convey("UpdateDUT - Replace Labstation with docker container", func() {
			// Create a DUT with labstation.
			createValidDUTWithLabstation(ctx, "dut-38", "machine-82", "labstation-35", "machine83")
			dut2, err := GetMachineLSE(ctx, "dut-38")
			So(err, ShouldBeNil)
			servo := dut2.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetServo()
			servo.ServoHostname = "local_labstation"
			servo.DockerContainerName = "docker-1"
			dut2.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().Servo = servo
			resp, err := UpdateDUT(ctx, dut2, mockFieldMask("dut.servo.hostname", "dut.servo.docker_container"))
			So(err, ShouldBeNil)
			resp.UpdateTime = nil
			dut2.UpdateTime = nil
			So(resp, ShouldResembleProto, dut2)
			// Check the servo changes were recorded.
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-38")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 3)
			// Verify that the changes were recorded by the history.
			So(changes[1].OldValue, ShouldEqual, "labstation-35")
			So(changes[1].NewValue, ShouldEqual, "local_labstation")
			So(changes[2].NewValue, ShouldEqual, "docker-1")
			So(changes[2].OldValue, ShouldEqual, "")
			// Two snapshots, one at registration another at update
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-38")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 2)
			s, err := state.GetStateRecord(ctx, "hosts/dut-38")
			So(err, ShouldBeNil)
			// State should be unchanged.
			So(s.GetState(), ShouldEqual, dut2.GetResourceState())
			lab2, err := GetMachineLSE(ctx, "labstation-35")
			servos := lab2.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetServos()
			So(servos, ShouldHaveLength, 0)
		})
		Convey("UpdateDUT - Replace docker container with non-existent labstation", func() {
			machine1 := &ufspb.Machine{
				Name: "machine-104",
				Device: &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: &ufspb.ChromeOSMachine{
						BuildTarget: "test",
						Model:       "test",
					},
				},
			}
			_, err := registration.CreateMachine(ctx, machine1)
			So(err, ShouldBeNil)
			dut1 := mockDUT("dut-39", "machine-104", "labstation-x", "serial-x", "dut-39-power-1", ".A1", 9988, nil, "docker-1")
			_, err = inventory.CreateMachineLSE(ctx, dut1)
			So(err, ShouldBeNil)
			dut1.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetServo().DockerContainerName = ""
			_, err = UpdateDUT(ctx, dut1, mockFieldMask("dut.servo.docker_container"))
			So(err, ShouldNotBeNil)
		})
		Convey("UpdateDUT - Replace docker container with existing labstation", func() {
			machine1 := &ufspb.Machine{
				Name: "machine-105",
				Device: &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: &ufspb.ChromeOSMachine{
						BuildTarget: "test",
						Model:       "test",
					},
				},
			}
			_, err := registration.CreateMachine(ctx, machine1)
			So(err, ShouldBeNil)
			machine1 = &ufspb.Machine{
				Name: "machine-106",
				Device: &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: &ufspb.ChromeOSMachine{
						BuildTarget: "test",
						Model:       "test",
					},
				},
			}
			_, err = registration.CreateMachine(ctx, machine1)
			So(err, ShouldBeNil)
			dut1 := mockDUT("dut-40", "machine-105", "labstation-x", "serial-x", "dut-40-power-1", ".A1", 9988, nil, "docker-1")
			_, err = inventory.CreateMachineLSE(ctx, dut1)
			So(err, ShouldBeNil)
			// Create a labstation
			lab1 := mockLabstation("labstation-y", "machine-106")
			_, err = inventory.CreateMachineLSE(ctx, lab1)
			// Update the dut to use labstation instead of servod on docker
			dut1.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetServo().DockerContainerName = ""
			dut1.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetServo().ServoHostname = "labstation-y"
			resp, err := UpdateDUT(ctx, dut1, mockFieldMask("dut.servo.hostname", "dut.servo.docker_container"))
			So(err, ShouldBeNil)
			resp.UpdateTime = nil
			dut1.UpdateTime = nil
			So(resp, ShouldResembleProto, dut1)
			// Check the servo changes were recorded.
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-40")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 2)
			// Verify that the changes were recorded by the history.
			So(changes[0].OldValue, ShouldEqual, "labstation-x")
			So(changes[0].NewValue, ShouldEqual, "labstation-y")
			So(changes[1].OldValue, ShouldEqual, "docker-1")
			So(changes[1].NewValue, ShouldEqual, "")
			// One snapshot at registration
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-40")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			s, err := state.GetStateRecord(ctx, "hosts/dut-40")
			So(err, ShouldBeNil)
			// State should be unchanged.
			So(s.GetState(), ShouldEqual, dut1.GetResourceState())
			lab2, err := GetMachineLSE(ctx, "labstation-y")
			servos := lab2.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetServos()
			So(servos, ShouldHaveLength, 1)
		})
		Convey("UpdateDUT - Change servo docker container", func() {
			machine1 := &ufspb.Machine{
				Name: "machine-107",
				Device: &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: &ufspb.ChromeOSMachine{
						BuildTarget: "test",
						Model:       "test",
					},
				},
			}
			_, err := registration.CreateMachine(ctx, machine1)
			So(err, ShouldBeNil)
			dut1 := mockDUT("dut-41", "machine-107", "host-x", "serial-x", "dut-41-power-1", ".A1", 9988, nil, "docker-1")
			_, err = inventory.CreateMachineLSE(ctx, dut1)
			So(err, ShouldBeNil)
			// Update the docker container name
			dut1.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetServo().DockerContainerName = "docker-2"
			_, err = UpdateDUT(ctx, dut1, mockFieldMask("dut.servo.docker_container"))
			So(err, ShouldBeNil)
			// Check the servo changes were recorded.
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-41")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			// Verify that the changes were recorded by the history.
			So(changes[0].OldValue, ShouldEqual, "docker-1")
			So(changes[0].NewValue, ShouldEqual, "docker-2")
			// One snapshot at registration
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-41")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
		})
		Convey("UpdateDUT - Change servo type", func() {
			machine1 := &ufspb.Machine{
				Name: "machine-108",
				Device: &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: &ufspb.ChromeOSMachine{
						BuildTarget: "test",
						Model:       "test",
					},
				},
			}
			_, err := registration.CreateMachine(ctx, machine1)
			So(err, ShouldBeNil)
			dut1 := mockDUT("dut-42", "machine-108", "host-x", "serial-x", "dut-42-power-1", ".A1", 9988, nil, "docker-1")
			_, err = inventory.CreateMachineLSE(ctx, dut1)
			So(err, ShouldBeNil)
			// Update the servo setup type
			dut1.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetServo().ServoSetup = chromeosLab.ServoSetupType_SERVO_SETUP_DUAL_V4
			_, err = UpdateDUT(ctx, dut1, mockFieldMask("dut.servo.setup"))
			So(err, ShouldBeNil)
			// Check the servo changes were recorded.
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-42")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			// Verify that the changes were recorded by the history.
			So(changes[0].OldValue, ShouldEqual, "SERVO_SETUP_REGULAR")
			So(changes[0].NewValue, ShouldEqual, "SERVO_SETUP_DUAL_V4")
			// One snapshot at registration
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-42")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
		})
	})
}

func TestGetChromeOSDeviceData(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	ctx = external.WithTestingContext(ctx)
	ctx = useTestingCfg(ctx)

	machine := &ufspb.Machine{
		Name: "machine-1",
		Device: &ufspb.Machine_ChromeosMachine{
			ChromeosMachine: &ufspb.ChromeOSMachine{
				ReferenceBoard: "test",
				BuildTarget:    "test",
				Model:          "test",
				Hwid:           "test",
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

	mfgCfg := &ufsmanufacturing.ManufacturingConfig{
		ManufacturingId: &ufsmanufacturing.ConfigID{Value: "test"},
	}

	hwidMockData := &ufspb.HwidData{Sku: "test-sku", Variant: "test-variant"}

	cfgBundle := &payload.ConfigBundle{
		DesignList: []*api.Design{
			{
				Id: &api.DesignId{
					Value: "test",
				},
				ProgramId: &api.ProgramId{
					Value: "test",
				},
			},
		},
	}
	cbBytes, _ := proto.Marshal(cfgBundle)
	UpdateConfigBundle(ctx, cbBytes, nil, true)

	attr1 := mockDutAttribute("attr_1", "design_list.id.value")
	configuration.UpdateDutAttribute(ctx, attr1)
	attr2 := mockDutAttribute("attr_2", "design_list.program_id.value")
	configuration.UpdateDutAttribute(ctx, attr2)

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
			So(resp.GetHwidData(), ShouldResembleProto, hwidMockData)
			So(resp.GetSchedulableLabels(), ShouldBeNil)
			So(resp.GetRespectAutomatedSchedulableLabels(), ShouldBeFalse)
			So(resp.GetDutV1().GetCommon().GetLabels().GetStability(), ShouldBeTrue)
		})

		Convey("GetChromeOSDevicedata - id happy path; inv v2 hwid api", func() {
			cfgLst := &config.Config{
				UseCachedHwidManufacturingConfig: false,
			}
			ctx2 := config.Use(ctx, cfgLst)
			invV2HwidData := &ufspb.HwidData{Sku: "test", Variant: "test"}

			resp, err := GetChromeOSDeviceData(ctx2, "machine-1", "")
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp.GetLabConfig(), ShouldResembleProto, dutMachinelse)
			So(resp.GetMachine(), ShouldResembleProto, machine)
			So(resp.GetDutState(), ShouldResembleProto, dutState)
			So(resp.GetDeviceConfig(), ShouldResembleProto, devCfg)
			So(resp.GetManufacturingConfig(), ShouldResembleProto, mfgCfg)
			So(resp.GetHwidData(), ShouldResembleProto, invV2HwidData)
			So(resp.GetSchedulableLabels(), ShouldBeNil)
			So(resp.GetRespectAutomatedSchedulableLabels(), ShouldBeFalse)
			So(resp.GetDutV1().GetCommon().GetLabels().GetStability(), ShouldBeTrue)
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
			So(resp.GetHwidData(), ShouldResembleProto, hwidMockData)
			So(resp.GetSchedulableLabels(), ShouldBeNil)
			So(resp.GetRespectAutomatedSchedulableLabels(), ShouldBeFalse)
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
			So(resp.GetSchedulableLabels(), ShouldBeNil)
			So(resp.GetRespectAutomatedSchedulableLabels(), ShouldBeFalse)
		})

		Convey("GetChromeOSDevicedata - data not found", func() {
			// DutState, DeviceConfig, ManufacturingConfig, HwidData,
			// SchedulableLabels all do not exist for "machine-3"
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
			So(resp.GetSchedulableLabels(), ShouldBeNil)
			So(resp.GetRespectAutomatedSchedulableLabels(), ShouldBeFalse)
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
			So(resp.GetSchedulableLabels(), ShouldBeNil)
			So(resp.GetRespectAutomatedSchedulableLabels(), ShouldBeFalse)
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
			So(resp.GetSchedulableLabels(), ShouldBeNil)
			So(resp.GetRespectAutomatedSchedulableLabels(), ShouldBeFalse)
		})

		Convey("GetChromeOSDevicedata - machinelse not found Error", func() {
			resp, err := GetChromeOSDeviceData(ctx, "machine-6", "")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "NotFound")
		})

		Convey("GetChromeOSDevicedata - happy path; hwid out of date and server fails", func() {
			// Mock server fails with test-no-server so use expired data found in datastore
			// instead.
			hwidMockDataExp := &ufspb.HwidData{Sku: "test-sku-no-server", Variant: "test-variant-no-server"}
			expiredTime := time.Now().Add(-2 * time.Hour).UTC()
			expiredLabels := &ufspb.DutLabel{
				PossibleLabels: []string{
					"test-possible-1",
					"test-possible-2",
				},
				Labels: []*ufspb.DutLabel_Label{
					{
						Name:  "test-label-1",
						Value: "test-value-1",
					},
					{
						Name:  "Sku",
						Value: "test-sku-no-server",
					},
					{
						Name:  "variant",
						Value: "test-variant-no-server",
					},
				},
			}
			fakeUpdateHwidData(ctx, expiredLabels, "test-no-server", expiredTime)

			machineExp := &ufspb.Machine{
				Name: "machine-using-exp",
				Device: &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: &ufspb.ChromeOSMachine{
						ReferenceBoard: "test",
						BuildTarget:    "test",
						Model:          "test",
						Hwid:           "test-no-server",
					},
				},
			}
			registration.CreateMachine(ctx, machineExp)

			dutMachinelseExp := mockDutMachineLSE("lse-using-exp")
			dutMachinelseExp.Machines = []string{"machine-using-exp"}
			inventory.CreateMachineLSE(ctx, dutMachinelseExp)

			dutStateExp := mockDutState("machine-using-exp", "lse-using-exp")
			UpdateDutState(ctx, dutStateExp)

			resp, err := GetChromeOSDeviceData(ctx, "machine-using-exp", "")
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp.GetLabConfig(), ShouldResembleProto, dutMachinelseExp)
			So(resp.GetMachine(), ShouldResembleProto, machineExp)
			So(resp.GetDutState(), ShouldResembleProto, dutStateExp)
			So(resp.GetDeviceConfig(), ShouldResembleProto, devCfg)
			So(resp.GetManufacturingConfig(), ShouldResembleProto, mfgCfg)
			So(resp.GetHwidData(), ShouldResembleProto, hwidMockDataExp)
			So(resp.GetSchedulableLabels(), ShouldBeNil)
			So(resp.GetRespectAutomatedSchedulableLabels(), ShouldBeFalse)
			So(resp.GetDutV1().GetCommon().GetLabels().GetStability(), ShouldBeTrue)
		})

		Convey("GetChromeOSDevicedata - happy path; hwid out of date and new cache", func() {
			// Datastore data is expired. Query hwid server, use values, and update cache.
			expiredTime := time.Now().Add(-2 * time.Hour).UTC()
			expiredLabels := &ufspb.DutLabel{
				PossibleLabels: []string{
					"test-possible-1",
					"test-possible-2",
				},
				Labels: []*ufspb.DutLabel_Label{
					{
						Name:  "test-label-1",
						Value: "test-value-1",
					},
					{
						Name:  "Sku",
						Value: "test-sku-exp",
					},
					{
						Name:  "variant",
						Value: "test-variant-exp",
					},
				},
			}
			hwid := "test"
			fakeUpdateHwidData(ctx, expiredLabels, hwid, expiredTime)

			machineHwid := &ufspb.Machine{
				Name: "machine-using-hwid-server",
				Device: &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: &ufspb.ChromeOSMachine{
						ReferenceBoard: "test",
						BuildTarget:    "test",
						Model:          "test",
						Hwid:           "test",
					},
				},
			}
			registration.CreateMachine(ctx, machineHwid)

			dutMachinelseHwid := mockDutMachineLSE("lse-using-hwid-server")
			dutMachinelseHwid.Machines = []string{"machine-using-hwid-server"}
			inventory.CreateMachineLSE(ctx, dutMachinelseHwid)

			dutStateHwid := mockDutState("machine-using-hwid-server", "lse-using-hwid-server")
			UpdateDutState(ctx, dutStateHwid)

			hwidEnt, err := configuration.GetHwidData(ctx, hwid)
			So(err, ShouldBeNil)
			So(hwidEnt, ShouldNotBeNil)
			So(hwidEnt.Updated, ShouldHappenWithin, time.Millisecond, expiredTime)

			resp, err := GetChromeOSDeviceData(ctx, "machine-using-hwid-server", "")
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp.GetLabConfig(), ShouldResembleProto, dutMachinelseHwid)
			So(resp.GetMachine(), ShouldResembleProto, machineHwid)
			So(resp.GetDutState(), ShouldResembleProto, dutStateHwid)
			So(resp.GetDeviceConfig(), ShouldResembleProto, devCfg)
			So(resp.GetManufacturingConfig(), ShouldResembleProto, mfgCfg)
			So(resp.GetHwidData(), ShouldResembleProto, hwidMockData)
			So(resp.GetSchedulableLabels(), ShouldBeNil)
			So(resp.GetRespectAutomatedSchedulableLabels(), ShouldBeFalse)
			So(resp.GetDutV1().GetCommon().GetLabels().GetStability(), ShouldBeTrue)

			hwidEnt, err = configuration.GetHwidData(ctx, hwid)
			So(err, ShouldBeNil)
			So(hwidEnt, ShouldNotBeNil)
			So(hwidEnt.Updated, ShouldHappenWithin, time.Second, time.Now().UTC())
		})

		Convey("GetChromeOSDevicedata - throttle hwid server traffic and no data in datastore", func() {
			// Try to get data from datastore but no data. Throttle traffic to hwid
			// server. HwidData should be nil. "test-no-cached-hwid-data" returns
			// valid fake, but HwidClient should not be called due to throttle.
			cfgLst := &config.Config{
				HwidServiceTrafficRatio:          0,
				UseCachedHwidManufacturingConfig: true,
			}
			trafficCtx := config.Use(ctx, cfgLst)

			machineThrottle := &ufspb.Machine{
				Name: "machine-throttle-hwid",
				Device: &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: &ufspb.ChromeOSMachine{
						ReferenceBoard: "test",
						BuildTarget:    "test",
						Model:          "test",
						Hwid:           "test-no-cached-hwid-data",
					},
				},
			}
			registration.CreateMachine(ctx, machineThrottle)

			dutMachinelseThrottle := mockDutMachineLSE("lse-throttle-hwid")
			dutMachinelseThrottle.Machines = []string{"machine-throttle-hwid"}
			inventory.CreateMachineLSE(ctx, dutMachinelseThrottle)

			dutStateThrottle := mockDutState("machine-throttle-hwid", "lse-throttle-hwid")
			UpdateDutState(ctx, dutStateThrottle)

			resp, err := GetChromeOSDeviceData(trafficCtx, "machine-throttle-hwid", "")
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp.GetLabConfig(), ShouldResembleProto, dutMachinelseThrottle)
			So(resp.GetMachine(), ShouldResembleProto, machineThrottle)
			So(resp.GetDutState(), ShouldResembleProto, dutStateThrottle)
			So(resp.GetDeviceConfig(), ShouldResembleProto, devCfg)
			So(resp.GetManufacturingConfig(), ShouldBeNil)
			So(resp.GetHwidData(), ShouldBeNil)
			So(resp.GetSchedulableLabels(), ShouldBeNil)
			So(resp.GetRespectAutomatedSchedulableLabels(), ShouldBeFalse)
			So(resp.GetDutV1().GetCommon().GetLabels().GetStability(), ShouldBeTrue)
		})

		Convey("GetChromeOSDevicedata - normal hwid server traffic and no data in datastore", func() {
			// Try to get data from datastore but no data. No throttle traffic to
			// hwid server. HwidData should return fake data using
			// "test-no-cached-hwid-data".
			machineNoThrottle := &ufspb.Machine{
				Name: "machine-no-throttle-hwid",
				Device: &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: &ufspb.ChromeOSMachine{
						ReferenceBoard: "test",
						BuildTarget:    "test",
						Model:          "test",
						Hwid:           "test-no-cached-hwid-data",
					},
				},
			}
			registration.CreateMachine(ctx, machineNoThrottle)

			dutMachinelseNoThrottle := mockDutMachineLSE("lse-no-throttle-hwid")
			dutMachinelseNoThrottle.Machines = []string{"machine-no-throttle-hwid"}
			inventory.CreateMachineLSE(ctx, dutMachinelseNoThrottle)

			dutStateNoThrottle := mockDutState("machine-no-throttle-hwid", "lse-no-throttle-hwid")
			UpdateDutState(ctx, dutStateNoThrottle)

			resp, err := GetChromeOSDeviceData(ctx, "machine-no-throttle-hwid", "")
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp.GetLabConfig(), ShouldResembleProto, dutMachinelseNoThrottle)
			So(resp.GetMachine(), ShouldResembleProto, machineNoThrottle)
			So(resp.GetDutState(), ShouldResembleProto, dutStateNoThrottle)
			So(resp.GetDeviceConfig(), ShouldResembleProto, devCfg)
			So(resp.GetManufacturingConfig(), ShouldBeNil)
			So(resp.GetHwidData(), ShouldResembleProto, hwidMockData)
			So(resp.GetSchedulableLabels(), ShouldBeNil)
			So(resp.GetRespectAutomatedSchedulableLabels(), ShouldBeFalse)
			So(resp.GetDutV1().GetCommon().GetLabels().GetStability(), ShouldBeTrue)
		})
	})
}

func TestValidateDeviceconfig(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	ctx = external.WithTestingContext(ctx)
	Convey("ValidateDeviceconfig", t, func() {
		Convey("TestValidateDeviceconfig - With sku", func() {
			machine1 := &ufspb.Machine{
				Name: "machine-10",
				Device: &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: &ufspb.ChromeOSMachine{
						BuildTarget: "test",
						Model:       "test",
						Sku:         "test",
					},
				},
			}
			err := validateDeviceConfig(ctx, machine1)
			So(err, ShouldBeNil)
		})
		Convey("TestValidateDeviceconfig - Without sku", func() {
			machine1 := &ufspb.Machine{
				Name: "machine-10",
				Device: &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: &ufspb.ChromeOSMachine{
						BuildTarget: "test",
						Model:       "test",
					},
				},
			}
			err := validateDeviceConfig(ctx, machine1)
			So(err, ShouldBeNil)
		})
		Convey("TestValidateDeviceconfig - non exisitent", func() {
			machine1 := &ufspb.Machine{
				Name: "machine-11",
				Device: &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: &ufspb.ChromeOSMachine{
						BuildTarget: "test11",
						Model:       "test11",
						Sku:         "test11",
					},
				},
			}
			err := validateDeviceConfig(ctx, machine1)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "No device config")
		})
	})
}

func TestRenameDUT(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	ctx = external.WithTestingContext(ctx)
	Convey("renameDUT", t, func() {
		Convey("renameDUT - Rename a dut in scheduling unit", func() {
			createValidDUTWithLabstation(ctx, "dut-1", "machine-1d", "labstation-1", "machine-1l")
			_, err := inventory.CreateSchedulingUnit(ctx, &ufspb.SchedulingUnit{
				Name:        "su-1",
				MachineLSEs: []string{"dut-1"},
			})
			So(err, ShouldBeNil)
			_, err = RenameMachineLSE(ctx, "dut-1", "dut-2")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, codes.FailedPrecondition.String())
		})
		Convey("renameDUT - Happy path", func() {
			createValidDUTWithLabstation(ctx, "dut-2", "machine-2d", "labstation-2", "machine-2l")
			_, err := RenameMachineLSE(ctx, "dut-2", "dut-3")
			So(err, ShouldBeNil)
			// Two snapshots, one at registration one at rename
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-2")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 2)
			// One snapshot at registration
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-3")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-2")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 3)
			// Verify all changes recorded by the history.
			So(changes[0].OldValue, ShouldEqual, "REGISTRATION")
			So(changes[0].NewValue, ShouldEqual, "REGISTRATION")
			So(changes[1].OldValue, ShouldEqual, "RENAME")
			So(changes[1].NewValue, ShouldEqual, "RENAME")
			So(changes[2].OldValue, ShouldEqual, "dut-2")
			So(changes[2].NewValue, ShouldEqual, "dut-3")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-3")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 2)
			// Verify all changes recorded by the history.
			So(changes[0].OldValue, ShouldEqual, "RENAME")
			So(changes[0].NewValue, ShouldEqual, "RENAME")
			So(changes[1].OldValue, ShouldEqual, "dut-2")
			So(changes[1].NewValue, ShouldEqual, "dut-3")
			// State record for old dut should be deleted
			_, err = state.GetStateRecord(ctx, "hosts/dut-2")
			So(err, ShouldNotBeNil)
			// State record for new dut should be same as old dut
			s, err := state.GetStateRecord(ctx, "hosts/dut-3")
			So(err, ShouldBeNil)
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_REGISTERED)
		})
	})
}

func TestCheckDutIdAndHostnameAreAssociated(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	ctx = external.WithTestingContext(ctx)
	Convey("Check DUT Id and Hostname", t, func() {
		Convey("renameDUT - Rename a dut in scheduling unit", func() {
			createValidDUTWithLabstation(ctx, "dut-1", "machine-20", "labstation-1", "machine-10")
			err := checkDutIdAndHostnameAreAssociated(ctx, "machine-20", "dut-1")
			So(err, ShouldBeNil)
			err = checkDutIdAndHostnameAreAssociated(ctx, "machine-20", "dut-2")
			So(err, ShouldNotBeNil)
		})
	})
}
