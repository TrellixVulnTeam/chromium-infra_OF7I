// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"testing"

	"golang.org/x/net/context"

	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/proto"
	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/chromiumos/infra/proto/go/device"
	"go.chromium.org/chromiumos/infra/proto/go/lab"
	"go.chromium.org/chromiumos/infra/proto/go/manufacturing"
	"go.chromium.org/luci/appengine/gaetesting"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/server/auth"
	"go.chromium.org/luci/server/auth/authtest"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	api "infra/appengine/cros/lab_inventory/api/v1"
	"infra/appengine/cros/lab_inventory/app/config"
	"infra/libs/cros/lab_inventory/datastore"
	"infra/libs/cros/lab_inventory/deviceconfig"
	"infra/libs/cros/lab_inventory/hwid"
	"infra/libs/fleet/protos"
)

type testFixture struct {
	T *testing.T
	C context.Context

	Inventory          *InventoryServerImpl
	DecoratedInventory *api.DecoratedInventory
}

func newTestFixtureWithContext(ctx context.Context, t *testing.T) (testFixture, func()) {
	tf := testFixture{T: t, C: ctx}
	mc := gomock.NewController(t)

	tf.Inventory = &InventoryServerImpl{}
	tf.DecoratedInventory = &api.DecoratedInventory{
		Service: tf.Inventory,
		Prelude: checkAccess,
	}

	validate := func() {
		mc.Finish()
	}
	return tf, validate
}

func testingContext() context.Context {
	c := gaetesting.TestingContextWithAppID("dev~infra-lab-inventory")
	c = config.Use(c, &config.Config{
		Readers: &config.LuciAuthGroup{
			Value: "fake_group",
		},
	})
	return c
}

func TestACL(t *testing.T) {
	t.Parallel()

	Convey("Get Chrome OS devices with ACL check", t, func() {
		ctx := testingContext()
		tf, validate := newTestFixtureWithContext(ctx, t)
		defer validate()

		req := &api.GetCrosDevicesRequest{}
		Convey("Unknown user", func() {
			_, err := tf.DecoratedInventory.GetCrosDevices(tf.C, req)
			So(err, ShouldNotBeNil)
			So(status.Code(err), ShouldEqual, codes.Internal)
		})
		Convey("Non authorized user", func() {
			ctx := auth.WithState(tf.C, &authtest.FakeState{
				Identity:       "user:abc@def.com",
				IdentityGroups: []string{"abc"},
			})
			_, err := tf.DecoratedInventory.GetCrosDevices(ctx, req)
			So(err, ShouldNotBeNil)
			So(status.Code(err), ShouldEqual, codes.PermissionDenied)
		})
		Convey("Happy path", func() {
			ctx := auth.WithState(tf.C, &authtest.FakeState{
				Identity:       "user:abc@def.com",
				IdentityGroups: []string{"fake_group"},
			})
			_, err := tf.DecoratedInventory.GetCrosDevices(ctx, req)
			// Get invalid argument error since we pass an empty request.
			So(status.Code(err), ShouldEqual, codes.InvalidArgument)
		})
	})
}

func TestAddCrosDevices(t *testing.T) {
	t.Parallel()
	dut1 := lab.ChromeOSDevice{
		Id: &lab.ChromeOSDeviceID{},
		Device: &lab.ChromeOSDevice_Dut{
			Dut: &lab.DeviceUnderTest{Hostname: "dut1"},
		},
	}
	dut2 := lab.ChromeOSDevice{
		Id: &lab.ChromeOSDeviceID{},
		Device: &lab.ChromeOSDevice_Dut{
			Dut: &lab.DeviceUnderTest{Hostname: "dut2"},
		},
	}
	labstation1 := lab.ChromeOSDevice{
		Id: &lab.ChromeOSDeviceID{},
		Device: &lab.ChromeOSDevice_Labstation{
			Labstation: &lab.Labstation{Hostname: "labstation1"},
		},
	}
	Convey("Add Chrome OS devices", t, func() {
		ctx := testingContext()
		tf, validate := newTestFixtureWithContext(ctx, t)
		defer validate()
		Convey("Add new devices", func() {
			for _, d := range []*lab.ChromeOSDevice{&dut1, &labstation1} {
				req := &api.AddCrosDevicesRequest{
					Devices: []*lab.ChromeOSDevice{d},
				}
				resp, err := tf.Inventory.AddCrosDevices(tf.C, req)
				So(err, ShouldBeNil)
				So(resp.PassedDevices, ShouldHaveLength, 1)
			}
		})
		Convey("Fail the input validation check", func() {
			req := &api.AddCrosDevicesRequest{
				Devices: []*lab.ChromeOSDevice{&dut1, &dut1},
			}
			resp, err := tf.Inventory.AddCrosDevices(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
		})
		Convey("Add existing devices", func() {
			req1 := &api.AddCrosDevicesRequest{
				Devices: []*lab.ChromeOSDevice{&dut1},
			}
			req2 := &api.AddCrosDevicesRequest{
				Devices: []*lab.ChromeOSDevice{&dut1, &dut2},
			}
			resp, err := tf.Inventory.AddCrosDevices(tf.C, req1)
			So(err, ShouldBeNil)
			So(resp.PassedDevices, ShouldHaveLength, 1)
			So(resp.FailedDevices, ShouldHaveLength, 0)
			So(resp.PassedDevices[0].Hostname, ShouldEqual, "dut1")

			resp, err = tf.Inventory.AddCrosDevices(tf.C, req2)
			So(err, ShouldBeNil)
			So(resp.FailedDevices, ShouldHaveLength, 1)
			So(resp.PassedDevices, ShouldHaveLength, 1)

			So(resp.FailedDevices[0].Hostname, ShouldEqual, "dut1")
			So(resp.PassedDevices[0].Hostname, ShouldEqual, "dut2")
		})
	})
}

func TestDeleteCrosDevices(t *testing.T) {
	t.Parallel()
	dut1 := lab.ChromeOSDevice{
		Id: &lab.ChromeOSDeviceID{},
		Device: &lab.ChromeOSDevice_Dut{
			Dut: &lab.DeviceUnderTest{
				Hostname: "dut1",
			},
		},
	}
	labstation1 := lab.ChromeOSDevice{
		Id: &lab.ChromeOSDeviceID{
			Value: "ASSET_ID_123",
		},
		Device: &lab.ChromeOSDevice_Labstation{
			Labstation: &lab.Labstation{
				Hostname: "labstation1",
			},
		},
	}
	devID1 := api.DeviceID{
		Id: &api.DeviceID_ChromeosDeviceId{ChromeosDeviceId: "ASSET_ID_123"},
	}
	devID2 := api.DeviceID{
		Id: &api.DeviceID_Hostname{Hostname: "dut1"},
	}
	devIDNonExisting := api.DeviceID{
		Id: &api.DeviceID_Hostname{Hostname: "ghost"},
	}
	Convey("Delete Chrome OS devices", t, func() {
		ctx := testingContext()
		tf, validate := newTestFixtureWithContext(ctx, t)
		defer validate()
		Convey("Happy path", func() {
			for _, d := range []*lab.ChromeOSDevice{&dut1, &labstation1} {
				req := &api.AddCrosDevicesRequest{
					Devices: []*lab.ChromeOSDevice{d},
				}
				resp, err := tf.Inventory.AddCrosDevices(tf.C, req)
				So(err, ShouldBeNil)
				So(resp.PassedDevices, ShouldHaveLength, 1)
			}

			reqDelete := &api.DeleteCrosDevicesRequest{
				Ids: []*api.DeviceID{&devID1, &devID2},
			}
			rsp, err := tf.Inventory.DeleteCrosDevices(tf.C, reqDelete)
			So(err, ShouldBeNil)
			So(rsp.RemovedDevices, ShouldHaveLength, 2)

			removedDeviceNames := make([]string, 2)
			for i, r := range rsp.RemovedDevices {
				removedDeviceNames[i] = r.Hostname
			}
			So("dut1", ShouldBeIn, removedDeviceNames)
			// "labstation1" won't be in the removed device names
			// since it was removed by its id.
			So("", ShouldBeIn, removedDeviceNames)

			So(rsp.FailedDevices, ShouldHaveLength, 0)
		})

		Convey("Bad request: duplicated ID", func() {
			req := &api.DeleteCrosDevicesRequest{Ids: []*api.DeviceID{&devID1, &devID1}}
			rsp, err := tf.Inventory.DeleteCrosDevices(tf.C, req)
			So(rsp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Duplicated id found")
		})

		Convey("Delete non existing device", func() {
			req := &api.DeleteCrosDevicesRequest{Ids: []*api.DeviceID{&devIDNonExisting}}
			rsp, err := tf.Inventory.DeleteCrosDevices(tf.C, req)
			// Remove nonexisting devices is regarded as a good operation.
			So(rsp.RemovedDevices, ShouldHaveLength, 1)
			So(err, ShouldBeNil)
		})
	})
}

func TestGetCrosDevices(t *testing.T) {
	t.Parallel()
	dut1 := lab.ChromeOSDevice{
		Id: &lab.ChromeOSDeviceID{},
		Device: &lab.ChromeOSDevice_Dut{
			Dut: &lab.DeviceUnderTest{Hostname: "dut1"},
		},
	}
	labstation1 := lab.ChromeOSDevice{
		Id: &lab.ChromeOSDeviceID{Value: "ASSET_ID_123"},
		Device: &lab.ChromeOSDevice_Labstation{
			Labstation: &lab.Labstation{Hostname: "labstation1"},
		},
	}
	devID1 := api.DeviceID{
		Id: &api.DeviceID_ChromeosDeviceId{ChromeosDeviceId: "ASSET_ID_123"},
	}
	devID2 := api.DeviceID{
		Id: &api.DeviceID_Hostname{Hostname: "dut1"},
	}
	devIDNonExisting := api.DeviceID{
		Id: &api.DeviceID_Hostname{Hostname: "ghost"},
	}

	Convey("Get Chrome OS devices", t, func() {
		ctx := testingContext()
		tf, validate := newTestFixtureWithContext(ctx, t)
		defer validate()

		for _, d := range []*lab.ChromeOSDevice{&dut1, &labstation1} {
			req := &api.AddCrosDevicesRequest{
				Devices: []*lab.ChromeOSDevice{d},
			}
			resp, err := tf.Inventory.AddCrosDevices(tf.C, req)
			So(err, ShouldBeNil)
			So(resp.PassedDevices, ShouldHaveLength, 1)
		}

		getHwidDataFunc = func(ctx context.Context, hwidstr string, secret string) (*hwid.Data, error) {
			return &hwid.Data{Sku: "sku", Variant: "variant"}, nil
		}
		getDeviceConfigFunc = func(ctx context.Context, ids []*device.ConfigId) ([]proto.Message, error) {
			fakeCfgs := make([]proto.Message, len(ids))
			for i := range fakeCfgs {
				fakeCfgs[i] = &device.Config{}
			}
			return fakeCfgs, nil
		}
		getManufacturingConfigFunc = func(ctx context.Context, ids []*manufacturing.ConfigID) ([]proto.Message, error) {
			fakeCfgs := make([]proto.Message, len(ids))
			for i := range fakeCfgs {
				fakeCfgs[i] = &manufacturing.Config{}
			}
			return fakeCfgs, nil
		}

		Convey("Happy path", func() {
			reqGet := &api.GetCrosDevicesRequest{
				Ids: []*api.DeviceID{&devID1, &devID2},
			}
			rsp, err := tf.Inventory.GetCrosDevices(tf.C, reqGet)
			So(err, ShouldBeNil)
			So(rsp.FailedDevices, ShouldBeEmpty)
			So(rsp.Data, ShouldHaveLength, 2)
		})

		Convey("Happy path with real device config", func() {
			realDutID1 := api.DeviceID{
				Id: &api.DeviceID_Hostname{Hostname: "real_dut1"},
			}
			realDutID2 := api.DeviceID{
				Id: &api.DeviceID_Hostname{Hostname: "real_dut2"},
			}
			realDut1 := lab.ChromeOSDevice{
				Id: &lab.ChromeOSDeviceID{},
				DeviceConfigId: &device.ConfigId{
					PlatformId: &device.PlatformId{
						Value: "falco_li",
					},
					ModelId: &device.ModelId{
						Value: "falco_li",
					},
				},
				Device: &lab.ChromeOSDevice_Dut{
					Dut: &lab.DeviceUnderTest{Hostname: "real_dut1"},
				},
			}
			realDut2 := lab.ChromeOSDevice{
				Id: &lab.ChromeOSDeviceID{},
				DeviceConfigId: &device.ConfigId{
					PlatformId: &device.PlatformId{
						Value: "hana",
					},
					ModelId: &device.ModelId{
						Value: "hana",
					},
				},
				Device: &lab.ChromeOSDevice_Dut{
					Dut: &lab.DeviceUnderTest{Hostname: "real_dut2"},
				},
			}
			_, err := datastore.AddDevices(ctx, []*lab.ChromeOSDevice{&realDut1, &realDut2}, false)
			So(err, ShouldBeNil)
			reqGet := &api.GetCrosDevicesRequest{
				Ids: []*api.DeviceID{&realDutID1, &realDutID2},
			}

			getDeviceConfigFunc = func(ctx context.Context, ids []*device.ConfigId) ([]proto.Message, error) {
				m := map[string]*device.Config{
					"slippy.falco.": {
						GpuFamily: "real_gpu",
					},
				}
				fakeCfgs := make([]proto.Message, 0)
				for _, id := range ids {
					if dc, ok := m[deviceconfig.GetDeviceConfigIDStr(id)]; ok {
						fakeCfgs = append(fakeCfgs, dc)
					} else {
						fakeCfgs = append(fakeCfgs, &device.Config{})
					}
				}
				return fakeCfgs, nil
			}
			rsp, err := tf.Inventory.GetCrosDevices(tf.C, reqGet)
			So(err, ShouldBeNil)
			So(rsp.Data, ShouldHaveLength, 2)
			resultM := make(map[string]string, 0)
			for _, d := range rsp.Data {
				resultM[d.GetLabConfig().GetDut().GetHostname()] = d.GetDeviceConfig().GetGpuFamily()
			}
			So(resultM["real_dut1"], ShouldEqual, "real_gpu")
			So(resultM["real_dut2"], ShouldEqual, "")
		})

		Convey("Bad hwid server", func() {
			getHwidDataFunc = hwid.GetHwidData
			reqGet := &api.GetCrosDevicesRequest{
				Ids: []*api.DeviceID{&devID1, &devID2},
			}
			rsp, err := tf.Inventory.GetCrosDevices(tf.C, reqGet)
			So(err, ShouldBeNil)
			So(rsp.Data, ShouldHaveLength, 2)
			So(rsp.FailedDevices, ShouldHaveLength, 0)
			So(rsp.Data[0].LabConfig, ShouldNotBeNil)
			So(rsp.Data[0].HwidData, ShouldBeNil)
			So(rsp.Data[1].LabConfig, ShouldNotBeNil)
			So(rsp.Data[1].HwidData, ShouldBeNil)
		})

		Convey("Failed to get device config", func() {
			getHwidDataFunc = func(ctx context.Context, hwidstr string, secret string) (*hwid.Data, error) {
				return &hwid.Data{Sku: "sku", Variant: "variant"}, nil
			}
			getDeviceConfigFunc = func(ctx context.Context, ids []*device.ConfigId) ([]proto.Message, error) {
				errs := make([]error, len(ids))
				for i := range ids {
					errs[i] = errors.New("get device config error")
				}
				return make([]proto.Message, len(ids)), errors.NewMultiError(errs...)
			}
			reqGet := &api.GetCrosDevicesRequest{
				Ids: []*api.DeviceID{&devID1, &devID2},
			}
			rsp, err := tf.Inventory.GetCrosDevices(tf.C, reqGet)
			So(err, ShouldBeNil)
			So(rsp.Data, ShouldHaveLength, 2)
			So(rsp.FailedDevices, ShouldHaveLength, 0)
			So(rsp.Data[0].LabConfig, ShouldNotBeNil)
			So(rsp.Data[0].DeviceConfig, ShouldBeNil)
			So(rsp.Data[1].LabConfig, ShouldNotBeNil)
			So(rsp.Data[1].DeviceConfig, ShouldBeNil)
		})

		Convey("Get non existing device", func() {
			reqGet := &api.GetCrosDevicesRequest{
				Ids: []*api.DeviceID{&devID1, &devID2, &devIDNonExisting},
			}
			rsp, err := tf.Inventory.GetCrosDevices(tf.C, reqGet)
			So(err, ShouldBeNil)
			So(rsp.FailedDevices, ShouldHaveLength, 1)
			So(rsp.FailedDevices[0].ErrorMsg, ShouldContainSubstring, "No such host")
			So(rsp.Data, ShouldHaveLength, 2)
		})
	})
}

func TestUpdateCrosDevicesSetup(t *testing.T) {
	t.Parallel()
	dut1 := lab.ChromeOSDevice{
		Id: &lab.ChromeOSDeviceID{Value: "UUID:01"},
		Device: &lab.ChromeOSDevice_Dut{
			Dut: &lab.DeviceUnderTest{
				Hostname:    "dut1",
				Peripherals: &lab.Peripherals{},
			},
		},
	}
	labstation1 := lab.ChromeOSDevice{
		Id: &lab.ChromeOSDeviceID{Value: "UUID:02"},
		Device: &lab.ChromeOSDevice_Labstation{
			Labstation: &lab.Labstation{Hostname: "labstation1"},
		},
	}
	Convey("Update Chrome OS devices setup", t, func() {
		ctx := testingContext()
		tf, validate := newTestFixtureWithContext(ctx, t)
		defer validate()

		for _, d := range []*lab.ChromeOSDevice{&labstation1, &dut1} {
			req := &api.AddCrosDevicesRequest{
				Devices: []*lab.ChromeOSDevice{d},
			}
			resp, err := tf.Inventory.AddCrosDevices(tf.C, req)
			So(err, ShouldBeNil)
			So(resp.PassedDevices, ShouldHaveLength, 1)
		}

		Convey("Happy path", func() {
			servo := lab.Servo{
				ServoHostname: "labstation1",
				ServoPort:     1234,
				ServoSerial:   "SN0001",
				ServoType:     "v3",
			}
			dut1.GetDut().GetPeripherals().Servo = &servo
			labstation1.GetLabstation().Servos = []*lab.Servo{&servo}

			req := &api.UpdateCrosDevicesSetupRequest{Devices: []*lab.ChromeOSDevice{&dut1, &labstation1}}
			resp, err := tf.Inventory.UpdateCrosDevicesSetup(tf.C, req)

			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp.UpdatedDevices, ShouldHaveLength, 2)

		})

		Convey("Update non-existing devices", func() {
			ghost := lab.ChromeOSDevice{
				Id: &lab.ChromeOSDeviceID{Value: "UUID:ghost"},
				Device: &lab.ChromeOSDevice_Dut{
					Dut: &lab.DeviceUnderTest{Hostname: "dut1"},
				},
			}
			req := &api.UpdateCrosDevicesSetupRequest{Devices: []*lab.ChromeOSDevice{&ghost}}
			resp, err := tf.Inventory.UpdateCrosDevicesSetup(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp.FailedDevices, ShouldHaveLength, 1)
			So(resp.FailedDevices[0].ErrorMsg, ShouldContainSubstring, "no such entity")
		})
	})
}

func TestUpdateDutsStatus(t *testing.T) {
	t.Parallel()
	dut1 := lab.ChromeOSDevice{
		Id: &lab.ChromeOSDeviceID{Value: "UUID:01"},
		Device: &lab.ChromeOSDevice_Dut{
			Dut: &lab.DeviceUnderTest{
				Hostname:    "dut1",
				Peripherals: &lab.Peripherals{},
			},
		},
	}
	labstation1 := lab.ChromeOSDevice{
		Id: &lab.ChromeOSDeviceID{Value: "UUID:02"},
		Device: &lab.ChromeOSDevice_Labstation{
			Labstation: &lab.Labstation{Hostname: "labstation1"},
		},
	}
	Convey("Update Dut status", t, func() {
		ctx := testingContext()
		tf, validate := newTestFixtureWithContext(ctx, t)
		defer validate()

		for _, d := range []*lab.ChromeOSDevice{&dut1, &labstation1} {
			req := &api.AddCrosDevicesRequest{
				Devices: []*lab.ChromeOSDevice{d},
			}
			resp, err := tf.Inventory.AddCrosDevices(tf.C, req)
			So(err, ShouldBeNil)
			So(resp.PassedDevices, ShouldHaveLength, 1)
		}

		Convey("Happy path", func() {
			req := &api.UpdateDutsStatusRequest{
				States: []*lab.DutState{
					{
						Id: &lab.ChromeOSDeviceID{Value: "UUID:01"},
					},
				}}
			resp, err := tf.Inventory.UpdateDutsStatus(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp.UpdatedDevices, ShouldHaveLength, 1)
			So(resp.UpdatedDevices[0].Id, ShouldEqual, "UUID:01")
		})

		Convey("Happy path with dut meta", func() {
			req := &api.UpdateDutsStatusRequest{
				States: []*lab.DutState{
					{
						Id: &lab.ChromeOSDeviceID{Value: "UUID:01"},
					},
				},
				DutMetas: []*api.DutMeta{
					{
						ChromeosDeviceId: "UUID:01",
						SerialNumber:     "serial2",
						HwID:             "hwid2",
					},
				},
			}
			resp, err := tf.Inventory.UpdateDutsStatus(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp.UpdatedDevices, ShouldHaveLength, 1)
			So(resp.UpdatedDevices[0].Id, ShouldEqual, "UUID:01")

			r := datastore.GetDevicesByIds(ctx, []string{"UUID:01"})
			So(r, ShouldHaveLength, 1)
			var p lab.ChromeOSDevice
			r[0].Entity.GetCrosDeviceProto(&p)
			So(p.GetSerialNumber(), ShouldEqual, "serial2")
			So(p.GetManufacturingId().GetValue(), ShouldEqual, "hwid2")
		})

		Convey("Cannot update a labstation", func() {
			req := &api.UpdateDutsStatusRequest{
				States: []*lab.DutState{
					{
						Id: &lab.ChromeOSDeviceID{Value: "UUID:02"},
					},
				}}
			resp, err := tf.Inventory.UpdateDutsStatus(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp.FailedDevices, ShouldHaveLength, 1)
			So(resp.FailedDevices[0].ErrorMsg, ShouldContainSubstring, "labstation")
		})
	})
}
func mockAsset(id, lab string) *fleet.ChopsAsset {
	return &fleet.ChopsAsset{
		Id: id,
		Location: &fleet.Location{
			Lab:      lab,
			Row:      "Phobos-3",
			Rack:     "Deimos-0",
			Shelf:    "Olympus-Mons",
			Position: "Amazonis-Planitia",
		},
	}
}

func assertLocationEqual(a, b *fleet.Location) {
	So(a.GetLab(), ShouldEqual, b.GetLab())
	So(a.GetRow(), ShouldEqual, b.GetRow())
	So(a.GetRack(), ShouldEqual, b.GetRack())
	So(a.GetShelf(), ShouldEqual, b.GetShelf())
	So(a.GetPosition(), ShouldEqual, b.GetPosition())
}

func TestAddAsset(t *testing.T) {
	t.Parallel()

	asset1 := mockAsset("4675636B596F75", "lab1")
	asset2 := mockAsset("4675636B596F76", "lab2")
	asset3 := mockAsset("", "")

	Convey("AddAsset", t, func() {
		ctx := testingContext()
		tf, validate := newTestFixtureWithContext(ctx, t)
		defer validate()

		Convey("Add asset with tag", func() {
			req := &api.AssetList{
				Asset: []*fleet.ChopsAsset{asset1},
			}
			resp, err := tf.Inventory.AddAssets(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp.Passed, ShouldHaveLength, 1)
			So(resp.Failed, ShouldHaveLength, 0)
			So(resp.Passed[0].Asset.GetId(), ShouldEqual, asset1.GetId())
		})
		Convey("Add asset without tag", func() {
			req := &api.AssetList{
				Asset: []*fleet.ChopsAsset{asset3},
			}
			resp, err := tf.Inventory.AddAssets(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp.Failed, ShouldHaveLength, 1)
			So(resp.Passed, ShouldHaveLength, 0)
		})
		Convey("Add existing asset", func() {
			req1 := &api.AssetList{
				Asset: []*fleet.ChopsAsset{asset1},
			}
			resp, err := tf.Inventory.AddAssets(tf.C, req1)
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp.Failed, ShouldHaveLength, 0)
			So(resp.Passed, ShouldHaveLength, 1)
			So(resp.Passed[0].Asset.GetId(), ShouldEqual, asset1.GetId())
			req2 := &api.AssetList{
				Asset: []*fleet.ChopsAsset{asset1, asset2},
			}
			resp, err = tf.Inventory.AddAssets(tf.C, req2)
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp.Failed, ShouldHaveLength, 1)
			So(resp.Passed, ShouldHaveLength, 1)
			So(resp.Passed[0].Asset.GetId(), ShouldEqual, asset2.GetId())
			So(resp.Failed[0].Asset.GetId(), ShouldEqual, asset1.GetId())
		})
	})
}

func TestUpdateAsset(t *testing.T) {
	t.Parallel()

	asset1 := mockAsset("4675636B596F75", "lab1")
	asset2 := mockAsset("4675636B596F76", "lab2")
	asset3 := mockAsset("4675636B596F76", "lab3")
	asset4 := mockAsset("", "")

	Convey("UpdateAsset", t, func() {
		ctx := testingContext()
		tf, validate := newTestFixtureWithContext(ctx, t)
		defer validate()

		Convey("Update non-existing asset", func() {
			req := &api.AssetList{
				Asset: []*fleet.ChopsAsset{asset1},
			}
			resp, err := tf.Inventory.UpdateAssets(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp.Passed, ShouldHaveLength, 0)
			So(resp.Failed, ShouldHaveLength, 1)
			So(resp.Failed[0].Asset.GetId(), ShouldEqual, asset1.GetId())
		})
		Convey("Update asset without tag", func() {
			req := &api.AssetList{
				Asset: []*fleet.ChopsAsset{asset4},
			}
			resp, err := tf.Inventory.UpdateAssets(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp.Failed, ShouldHaveLength, 1)
			So(resp.Passed, ShouldHaveLength, 0)
		})
		Convey("Update existing asset", func() {
			req1 := &api.AssetList{
				Asset: []*fleet.ChopsAsset{asset2},
			}
			resp, err := tf.Inventory.AddAssets(tf.C, req1)
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp.Failed, ShouldHaveLength, 0)
			So(resp.Passed, ShouldHaveLength, 1)
			So(resp.Passed[0].Asset.GetId(), ShouldEqual, asset2.GetId())
			req2 := &api.AssetList{
				Asset: []*fleet.ChopsAsset{asset1, asset3},
			}
			resp, err = tf.Inventory.UpdateAssets(tf.C, req2)
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp.Failed, ShouldHaveLength, 1)
			So(resp.Passed, ShouldHaveLength, 1)
			So(resp.Failed[0].Asset.GetId(), ShouldEqual, asset1.GetId())
			So(resp.Passed[0].Asset.GetId(), ShouldEqual, asset3.GetId())

			req := &api.AssetIDList{Id: []string{asset2.GetId()}}
			resp, err = tf.Inventory.GetAssets(tf.C, req)
			So(err, ShouldBeNil)
			So(resp.Passed[0].Asset.GetLocation().GetLab(), ShouldEqual, asset3.GetLocation().GetLab())
		})
	})
}

func TestGetAsset(t *testing.T) {
	t.Parallel()

	asset1 := mockAsset("4675636B596F75", "lab1")

	Convey("GetAsset", t, func() {
		ctx := testingContext()
		tf, validate := newTestFixtureWithContext(ctx, t)
		defer validate()
		req := &api.AssetList{
			Asset: []*fleet.ChopsAsset{asset1},
		}
		resp, err := tf.Inventory.AddAssets(tf.C, req)
		So(err, ShouldBeNil)
		So(resp.Passed, ShouldHaveLength, 1)
		So(resp.Passed[0].ErrorMsg, ShouldEqual, "")
		So(resp.Passed[0].Asset.GetId(), ShouldEqual, asset1.GetId())
		assertLocationEqual(resp.Passed[0].Asset.GetLocation(), asset1.Location)
		Convey("Get Asset from ID", func() {
			req := &api.AssetIDList{Id: []string{asset1.GetId()}}
			resp, err = tf.Inventory.GetAssets(tf.C, req)
			So(err, ShouldBeNil)
			So(resp.Passed, ShouldHaveLength, 1)
			So(resp.Passed[0].ErrorMsg, ShouldEqual, "")
			So(resp.Passed[0].Asset.GetId(), ShouldEqual, asset1.GetId())
			assertLocationEqual(resp.Passed[0].Asset.GetLocation(), asset1.Location)
		})
		Convey("Get Asset from non existent ID", func() {
			req := &api.AssetIDList{Id: []string{
				"4675636B596F76",
			}}
			resp, err = tf.Inventory.GetAssets(tf.C, req)
			So(err, ShouldBeNil)
			So(resp.Failed, ShouldHaveLength, 1)
			So(resp.Failed[0].ErrorMsg, ShouldNotEqual, "")
			So(resp.Failed[0].Asset.GetId(), ShouldEqual, "4675636B596F76")
		})
	})
}

func TestDeleteAsset(t *testing.T) {
	t.Parallel()

	asset1 := mockAsset("4675636B596F75", "lab1")
	asset2 := mockAsset("4675636B596F76", "lab2")

	Convey("DeleteAsset", t, func() {
		ctx := testingContext()
		tf, validate := newTestFixtureWithContext(ctx, t)
		defer validate()
		Convey("Delete Asset from ID", func() {
			req := &api.AssetList{
				Asset: []*fleet.ChopsAsset{asset1},
			}
			resp, err := tf.Inventory.AddAssets(tf.C, req)
			So(err, ShouldBeNil)
			So(resp.Passed, ShouldHaveLength, 1)
			So(resp.Passed[0].Asset.GetId(), ShouldEqual, asset1.GetId())
			req1 := &api.AssetIDList{Id: []string{asset1.GetId()}}
			resp1, err := tf.Inventory.DeleteAssets(tf.C, req1)
			So(err, ShouldBeNil)
			So(resp1.Passed, ShouldHaveLength, 1)
			So(resp1.Passed[0].Id, ShouldEqual, asset1.GetId())
		})
		Convey("Delete non-existent Asset", func() {
			req := &api.AssetList{
				Asset: []*fleet.ChopsAsset{asset1},
			}
			resp, err := tf.Inventory.AddAssets(tf.C, req)
			So(err, ShouldBeNil)
			So(resp.Passed, ShouldHaveLength, 1)
			So(resp.Passed[0].Asset.GetId(), ShouldEqual, asset1.GetId())
			req1 := &api.AssetIDList{
				Id: []string{asset1.GetId(), asset2.GetId()},
			}
			resp2, err := tf.Inventory.DeleteAssets(tf.C, req1)
			So(err, ShouldBeNil)
			So(resp2.Failed, ShouldHaveLength, 1)
			So(resp2.Passed, ShouldHaveLength, 1)
			So(resp2.Failed[0].Id, ShouldEqual, asset2.GetId())
			So(resp2.Passed[0].Id, ShouldEqual, asset1.GetId())
		})
	})
}
