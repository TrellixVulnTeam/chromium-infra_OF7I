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

		SkipConvey("Happy path", func() {
			reqGet := &api.GetCrosDevicesRequest{
				Ids: []*api.DeviceID{&devID1, &devID2},
			}
			rsp, err := tf.Inventory.GetCrosDevices(tf.C, reqGet)
			So(err, ShouldBeNil)
			So(rsp.FailedDevices, ShouldBeEmpty)
			So(rsp.Data, ShouldHaveLength, 2)
		})

		SkipConvey("Happy path with real device config", func() {
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

		SkipConvey("Bad hwid server", func() {
			getHwidDataFunc = hwid.GetHwidData
			reqGet := &api.GetCrosDevicesRequest{
				Ids: []*api.DeviceID{&devID1, &devID2},
			}
			rsp, err := tf.Inventory.GetCrosDevices(tf.C, reqGet)
			So(err, ShouldBeNil)
			So(rsp.Data, ShouldHaveLength, 0)
			So(rsp.FailedDevices, ShouldHaveLength, 2)
			So(rsp.FailedDevices[0].ErrorMsg, ShouldContainSubstring, "HWID server responsonse was not OK")
		})

		SkipConvey("Failed to get device config", func() {
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
			So(rsp.Data, ShouldHaveLength, 0)
			So(rsp.FailedDevices, ShouldHaveLength, 2)
			So(rsp.FailedDevices[0].ErrorMsg, ShouldEqual, "get device config error")
		})

		SkipConvey("Get non existing device", func() {
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
