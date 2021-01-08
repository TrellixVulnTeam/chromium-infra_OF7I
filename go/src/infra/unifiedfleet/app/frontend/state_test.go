// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
	"google.golang.org/genproto/googleapis/rpc/code"

	ufspb "infra/unifiedfleet/api/v1/models"
	chromeosLab "infra/unifiedfleet/api/v1/models/chromeos/lab"
	api "infra/unifiedfleet/api/v1/rpc"
	"infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/model/inventory"
	"infra/unifiedfleet/app/model/registration"
	"infra/unifiedfleet/app/model/state"
	"infra/unifiedfleet/app/util"
)

func TestImportStates(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	Convey("Import machine lses", t, func() {
		Convey("happy path", func() {
			req := &api.ImportStatesRequest{
				Source: &api.ImportStatesRequest_MachineDbSource{
					MachineDbSource: &api.MachineDBSource{
						Host: "fake_host",
					},
				},
			}
			res, err := tf.Fleet.ImportStates(ctx, req)
			So(err, ShouldBeNil)
			So(res.Code, ShouldEqual, code.Code_OK)

			states, _, err := state.ListStateRecords(ctx, 100, "", nil)
			So(err, ShouldBeNil)
			So(api.ParseResources(states, "ResourceName"), ShouldResemble, []string{"hosts/esx-8", "hosts/web", "machines/machine1", "machines/machine2", "machines/machine3", "vms/vm578-m4"})
			s, err := state.GetStateRecord(ctx, "machines/machine1")
			So(err, ShouldBeNil)
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_SERVING)
			s, err = state.GetStateRecord(ctx, "vms/vm578-m4")
			So(err, ShouldBeNil)
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_NEEDS_REPAIR)
		})
	})
}

func TestUpdateState(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	Convey("Update state", t, func() {
		Convey("happy path", func() {
			req := &api.UpdateStateRequest{
				State: &ufspb.StateRecord{
					ResourceName: "hosts/chromeos1-row2-rack3-host4",
					State:        ufspb.State_STATE_RESERVED,
				},
			}
			res, err := tf.Fleet.UpdateState(ctx, req)
			So(err, ShouldBeNil)
			s, err := state.GetStateRecord(ctx, "hosts/chromeos1-row2-rack3-host4")
			So(err, ShouldBeNil)
			So(res, ShouldResembleProto, s)
		})
		Convey("invalid resource prefix", func() {
			req := &api.UpdateStateRequest{
				State: &ufspb.StateRecord{
					ResourceName: "resources/chromeos1-row2-rack3-host4",
					State:        ufspb.State_STATE_RESERVED,
				},
			}
			_, err := tf.Fleet.UpdateState(ctx, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.ResourceFormat)
		})
		Convey("empty resource name", func() {
			req := &api.UpdateStateRequest{
				State: &ufspb.StateRecord{
					ResourceName: "",
					State:        ufspb.State_STATE_RESERVED,
				},
			}
			_, err := tf.Fleet.UpdateState(ctx, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.ResourceFormat)
		})
		Convey("invalid characters in resource name", func() {
			req := &api.UpdateStateRequest{
				State: &ufspb.StateRecord{
					ResourceName: "hosts/host1@_@",
					State:        ufspb.State_STATE_RESERVED,
				},
			}
			_, err := tf.Fleet.UpdateState(ctx, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.ResourceFormat)
		})
	})
}

func TestGetState(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	Convey("Get state", t, func() {
		Convey("happy path", func() {
			s := &ufspb.StateRecord{
				ResourceName: "hosts/chromeos1-row2-rack3-host4",
				State:        ufspb.State_STATE_RESERVED,
			}
			_, err := state.UpdateStateRecord(ctx, s)
			So(err, ShouldBeNil)
			req := &api.GetStateRequest{
				ResourceName: "hosts/chromeos1-row2-rack3-host4",
			}
			res, err := tf.Fleet.GetState(ctx, req)
			So(err, ShouldBeNil)
			So(res, ShouldResembleProto, s)
		})
		Convey("valid resource name, but not found", func() {
			res, err := tf.Fleet.GetState(ctx, &api.GetStateRequest{
				ResourceName: "hosts/chromeos-fakehost",
			})
			So(err, ShouldNotBeNil)
			So(res, ShouldBeNil)
			So(err.Error(), ShouldContainSubstring, datastore.NotFound)
		})
		Convey("invalid resource prefix", func() {
			req := &api.GetStateRequest{
				ResourceName: "resources/chromeos1-row2-rack3-host4",
			}
			_, err := tf.Fleet.GetState(ctx, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.ResourceFormat)
		})
		Convey("empty resource name", func() {
			req := &api.GetStateRequest{
				ResourceName: "",
			}
			_, err := tf.Fleet.GetState(ctx, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.ResourceFormat)
		})
		Convey("invalid characters in resource name", func() {
			req := &api.GetStateRequest{
				ResourceName: "hosts/host1@_@",
			}
			_, err := tf.Fleet.GetState(ctx, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.ResourceFormat)
		})
	})
}

func TestUpdateDutState(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	osCtx, _ := util.SetupDatastoreNamespace(ctx, util.OSNamespace)
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	dutStateGood := &chromeosLab.DutState{
		Id:       &chromeosLab.ChromeOSDeviceID{Value: "UUID:01"},
		Hostname: "hostname-01",
	}
	Convey("Update dut state", t, func() {
		Convey("empty dut ID", func() {
			_, err := tf.Fleet.UpdateDutState(osCtx, &api.UpdateDutStateRequest{
				DutState: &chromeosLab.DutState{},
			})
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.EmptyID)
		})

		Convey("dut ID with all spaces", func() {
			_, err := tf.Fleet.UpdateDutState(osCtx, &api.UpdateDutStateRequest{
				DutState: &chromeosLab.DutState{
					Id: &chromeosLab.ChromeOSDeviceID{Value: "   "},
				},
			})
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.EmptyID)
		})

		Convey("empty hostname", func() {
			_, err := tf.Fleet.UpdateDutState(osCtx, &api.UpdateDutStateRequest{
				DutState: &chromeosLab.DutState{
					Id:       &chromeosLab.ChromeOSDeviceID{Value: "UUID:01"},
					Hostname: "   ",
				},
			})
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Hostname cannot be empty")
		})

		Convey("non-matched dut ID in lab meta", func() {
			_, err := tf.Fleet.UpdateDutState(osCtx, &api.UpdateDutStateRequest{
				DutState: dutStateGood,
				LabMeta: &ufspb.LabMeta{
					ChromeosDeviceId: "UUID:wrong",
				},
			})
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Mismatched dut ID")
		})

		Convey("non-matched dut hostname in lab meta", func() {
			_, err := tf.Fleet.UpdateDutState(osCtx, &api.UpdateDutStateRequest{
				DutState: dutStateGood,
				LabMeta: &ufspb.LabMeta{
					ChromeosDeviceId: "UUID:01",
					Hostname:         "hostname-wrong",
				},
			})
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Mismatched dut hostname")
		})

		Convey("non-matched dut ID in dut meta", func() {
			_, err := tf.Fleet.UpdateDutState(osCtx, &api.UpdateDutStateRequest{
				DutState: dutStateGood,
				DutMeta: &ufspb.DutMeta{
					ChromeosDeviceId: "UUID:wrong",
				},
			})
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Mismatched dut ID")
		})

		Convey("non-matched dut hostname in dut meta", func() {
			_, err := tf.Fleet.UpdateDutState(osCtx, &api.UpdateDutStateRequest{
				DutState: dutStateGood,
				DutMeta: &ufspb.DutMeta{
					ChromeosDeviceId: "UUID:01",
					Hostname:         "hostname-wrong",
				},
			})
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Mismatched dut hostname")
		})

		Convey("happy path with no data", func() {
			err := mockOSMachineAssetAndHost(ctx, "rpc-dutstate-id1", "rpc-dutstate-host1", "dut")
			So(err, ShouldBeNil)

			// Use osCtx as we will restrict ctx to include namespace in prod.
			_, err = tf.Fleet.UpdateDutState(osCtx, &api.UpdateDutStateRequest{
				DutState: &chromeosLab.DutState{
					Id:       &chromeosLab.ChromeOSDeviceID{Value: "rpc-dutstate-id1"},
					Hostname: "rpc-dutstate-host1",
				},
			})
			So(err, ShouldBeNil)

			m, err := registration.GetMachine(osCtx, "rpc-dutstate-id1")
			So(err, ShouldBeNil)
			So(m.GetSerialNumber(), ShouldEqual, "")
			So(m.GetChromeosMachine().GetSku(), ShouldEqual, "")
			a, err := registration.GetAsset(osCtx, "rpc-dutstate-id1")
			So(err, ShouldBeNil)
			So(a.GetInfo().GetSerialNumber(), ShouldEqual, "")
			So(a.GetInfo().GetSku(), ShouldEqual, "")
			lse, err := inventory.GetMachineLSE(osCtx, "rpc-dutstate-host1")
			So(err, ShouldBeNil)
			So(lse.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetServo().GetServoType(), ShouldBeEmpty)
			So(lse.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetServo().GetServoTopology(), ShouldBeNil)
			So(lse.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetSmartUsbhub(), ShouldBeFalse)
		})

		Convey("happy path with dut meta", func() {
			err := mockOSMachineAssetAndHost(ctx, "rpc-dutstate-id2", "rpc-dutstate-host2", "dut")
			So(err, ShouldBeNil)
			// Use osCtx as we will restrict ctx to include namespace in prod.
			_, err = tf.Fleet.UpdateDutState(osCtx, &api.UpdateDutStateRequest{
				DutState: &chromeosLab.DutState{
					Id:       &chromeosLab.ChromeOSDeviceID{Value: "rpc-dutstate-id2"},
					Hostname: "rpc-dutstate-host2",
				},
				DutMeta: &ufspb.DutMeta{
					ChromeosDeviceId: "rpc-dutstate-id2",
					Hostname:         "rpc-dutstate-host2",
					SerialNumber:     "real-serial",
					HwID:             "real-hwid",
					DeviceSku:        "real-sku",
				},
			})
			So(err, ShouldBeNil)

			m, err := registration.GetMachine(osCtx, "rpc-dutstate-id2")
			So(err, ShouldBeNil)
			So(m.GetSerialNumber(), ShouldEqual, "real-serial")
			So(m.GetChromeosMachine().GetSku(), ShouldEqual, "real-sku")
			So(m.GetChromeosMachine().GetHwid(), ShouldEqual, "real-hwid")
			a, err := registration.GetAsset(osCtx, "rpc-dutstate-id2")
			So(err, ShouldBeNil)
			So(a.GetInfo().GetSerialNumber(), ShouldEqual, "real-serial")
			So(a.GetInfo().GetSku(), ShouldEqual, "real-sku")
			So(a.GetInfo().GetHwid(), ShouldEqual, "real-hwid")
			lse, err := inventory.GetMachineLSE(osCtx, "rpc-dutstate-host2")
			So(err, ShouldBeNil)
			So(lse.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetServo().GetServoType(), ShouldBeEmpty)
			So(lse.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetServo().GetServoTopology(), ShouldBeNil)
			So(lse.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetSmartUsbhub(), ShouldBeFalse)
		})

		Convey("happy path with lab meta", func() {
			err := mockOSMachineAssetAndHost(ctx, "rpc-dutstate-id3", "rpc-dutstate-host3", "dut")
			So(err, ShouldBeNil)
			topology := &chromeosLab.ServoTopology{
				Main: &chromeosLab.ServoTopologyItem{
					Type:         "servo_v4",
					Serial:       "SomeSerial",
					SysfsProduct: "1-4.6.5",
				},
			}
			// Use osCtx as we will restrict ctx to include namespace in prod.
			_, err = tf.Fleet.UpdateDutState(osCtx, &api.UpdateDutStateRequest{
				DutState: &chromeosLab.DutState{
					Id:       &chromeosLab.ChromeOSDeviceID{Value: "rpc-dutstate-id3"},
					Hostname: "rpc-dutstate-host3",
				},
				LabMeta: &ufspb.LabMeta{
					ChromeosDeviceId: "rpc-dutstate-id3",
					Hostname:         "rpc-dutstate-host3",
					ServoType:        "servo_v4_with_ccd_cr50",
					ServoTopology:    topology,
					SmartUsbhub:      true,
				},
			})
			So(err, ShouldBeNil)

			m, err := registration.GetMachine(osCtx, "rpc-dutstate-id3")
			So(err, ShouldBeNil)
			So(m.GetSerialNumber(), ShouldEqual, "")
			So(m.GetChromeosMachine().GetSku(), ShouldEqual, "")
			a, err := registration.GetAsset(osCtx, "rpc-dutstate-id3")
			So(err, ShouldBeNil)
			So(a.GetInfo().GetSerialNumber(), ShouldEqual, "")
			So(a.GetInfo().GetSku(), ShouldEqual, "")
			lse, err := inventory.GetMachineLSE(osCtx, "rpc-dutstate-host3")
			So(err, ShouldBeNil)
			So(lse.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetServo().GetServoType(), ShouldEqual, "servo_v4_with_ccd_cr50")
			So(lse.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetServo().GetServoTopology(), ShouldResembleProto, topology)
			So(lse.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetSmartUsbhub(), ShouldBeTrue)
		})

		Convey("only dut meta update for labstation", func() {
			err := mockOSMachineAssetAndHost(ctx, "rpc-dutstate-id4", "rpc-dutstate-host4", "labstation")
			So(err, ShouldBeNil)
			// Use osCtx as we will restrict ctx to include namespace in prod.
			_, err = tf.Fleet.UpdateDutState(osCtx, &api.UpdateDutStateRequest{
				DutState: &chromeosLab.DutState{
					Id:       &chromeosLab.ChromeOSDeviceID{Value: "rpc-dutstate-id4"},
					Hostname: "rpc-dutstate-host4",
				},
				DutMeta: &ufspb.DutMeta{
					ChromeosDeviceId: "rpc-dutstate-id4",
					Hostname:         "rpc-dutstate-host4",
					SerialNumber:     "real-serial",
				},
				LabMeta: &ufspb.LabMeta{
					ChromeosDeviceId: "rpc-dutstate-id4",
					Hostname:         "rpc-dutstate-host4",
					ServoType:        "servo_v4_with_ccd_cr50",
					SmartUsbhub:      true,
				},
			})
			So(err, ShouldBeNil)

			m, err := registration.GetMachine(osCtx, "rpc-dutstate-id4")
			So(err, ShouldBeNil)
			So(m.GetSerialNumber(), ShouldEqual, "real-serial")
			a, err := registration.GetAsset(osCtx, "rpc-dutstate-id4")
			So(err, ShouldBeNil)
			So(a.GetInfo().GetSerialNumber(), ShouldEqual, "real-serial")
			lse, err := inventory.GetMachineLSE(osCtx, "rpc-dutstate-host4")
			So(err, ShouldBeNil)
			So(lse.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetServo().GetServoType(), ShouldBeEmpty)
			So(lse.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetServo().GetServoTopology(), ShouldBeNil)
			So(lse.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetSmartUsbhub(), ShouldBeFalse)
		})

		Convey("no update for chrome device", func() {
			err := mockOSMachineAssetAndHost(ctx, "rpc-dutstate-id5", "rpc-dutstate-host5", "browser")
			So(err, ShouldBeNil)
			// Use osCtx as we will restrict ctx to include namespace in prod.
			_, err = tf.Fleet.UpdateDutState(osCtx, &api.UpdateDutStateRequest{
				DutState: &chromeosLab.DutState{
					Id:       &chromeosLab.ChromeOSDeviceID{Value: "rpc-dutstate-id5"},
					Hostname: "rpc-dutstate-host5",
				},
				DutMeta: &ufspb.DutMeta{
					ChromeosDeviceId: "rpc-dutstate-id5",
					Hostname:         "rpc-dutstate-host5",
					SerialNumber:     "real-serial",
				},
				LabMeta: &ufspb.LabMeta{
					ChromeosDeviceId: "rpc-dutstate-id5",
					Hostname:         "rpc-dutstate-host5",
					ServoType:        "servo_v4_with_ccd_cr50",
					SmartUsbhub:      true,
				},
			})
			So(err, ShouldBeNil)

			m, err := registration.GetMachine(osCtx, "rpc-dutstate-id5")
			So(err, ShouldBeNil)
			So(m.GetSerialNumber(), ShouldEqual, "")
			_, err = registration.GetAsset(osCtx, "rpc-dutstate-id5")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "not found")
			lse, err := inventory.GetMachineLSE(osCtx, "rpc-dutstate-host5")
			So(err, ShouldBeNil)
			So(lse.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetServo().GetServoType(), ShouldBeEmpty)
			So(lse.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetServo().GetServoTopology(), ShouldBeNil)
			So(lse.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetSmartUsbhub(), ShouldBeFalse)
		})
	})
}

func mockOSMachineAssetAndHost(ctx context.Context, id, hostname, deviceType string) error {
	osCtx, err := util.SetupDatastoreNamespace(ctx, util.OSNamespace)
	if err != nil {
		return err
	}
	var machineLSE1 *ufspb.MachineLSE
	var machine *ufspb.Machine
	var asset *ufspb.Asset
	switch deviceType {
	case "dut":
		machineLSE1 = &ufspb.MachineLSE{
			Name:     hostname,
			Hostname: hostname,
			Machines: []string{id},
			Lse: &ufspb.MachineLSE_ChromeosMachineLse{
				ChromeosMachineLse: &ufspb.ChromeOSMachineLSE{
					ChromeosLse: &ufspb.ChromeOSMachineLSE_DeviceLse{
						DeviceLse: &ufspb.ChromeOSDeviceLSE{
							Device: &ufspb.ChromeOSDeviceLSE_Dut{
								Dut: &chromeosLab.DeviceUnderTest{
									Hostname: hostname,
									Peripherals: &chromeosLab.Peripherals{
										Servo: &chromeosLab.Servo{},
									},
								},
							},
						},
					},
				},
			},
		}
		machine = &ufspb.Machine{
			Name: id,
			Device: &ufspb.Machine_ChromeosMachine{
				ChromeosMachine: &ufspb.ChromeOSMachine{},
			},
		}
		asset = &ufspb.Asset{
			Name: id,
			Info: &ufspb.AssetInfo{
				AssetTag: id,
			},
			Type:     ufspb.AssetType_DUT,
			Location: &ufspb.Location{},
		}
	case "labstation":
		machineLSE1 = &ufspb.MachineLSE{
			Name:     hostname,
			Hostname: hostname,
			Machines: []string{id},
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
		machine = &ufspb.Machine{
			Name: id,
			Device: &ufspb.Machine_ChromeosMachine{
				ChromeosMachine: &ufspb.ChromeOSMachine{},
			},
		}
		asset = &ufspb.Asset{
			Name: id,
			Info: &ufspb.AssetInfo{
				AssetTag: id,
			},
			Type:     ufspb.AssetType_LABSTATION,
			Location: &ufspb.Location{},
		}
	case "browser":
		machineLSE1 = &ufspb.MachineLSE{
			Name:     hostname,
			Hostname: hostname,
			Machines: []string{id},
			Lse: &ufspb.MachineLSE_ChromeBrowserMachineLse{
				ChromeBrowserMachineLse: &ufspb.ChromeBrowserMachineLSE{},
			},
		}
		machine = &ufspb.Machine{
			Name: id,
			Device: &ufspb.Machine_ChromeBrowserMachine{
				ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
			},
		}
	}

	if _, err := registration.CreateMachine(osCtx, machine); err != nil {
		return err
	}
	if asset != nil {
		if _, err := registration.CreateAsset(osCtx, asset); err != nil {
			return err
		}
	}
	if _, err := inventory.CreateMachineLSE(osCtx, machineLSE1); err != nil {
		return err
	}
	return nil
}
