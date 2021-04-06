// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	timestamp "github.com/golang/protobuf/ptypes/timestamp"
	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/chromiumos/infra/proto/go/device"
	"go.chromium.org/chromiumos/infra/proto/go/lab"
	"go.chromium.org/luci/appengine/gaetesting"
	. "go.chromium.org/luci/common/testing/assertions"
	ds "go.chromium.org/luci/gae/service/datastore"
	"go.chromium.org/luci/server/auth"
	"go.chromium.org/luci/server/auth/authtest"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	api "infra/appengine/cros/lab_inventory/api/v1"
	"infra/appengine/cros/lab_inventory/app/config"
	"infra/cros/lab_inventory/datastore"
	invlibs "infra/cros/lab_inventory/protos"
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

type devcfgEntity struct {
	_kind     string `gae:"$kind,DevConfig"`
	ID        string `gae:"$id"`
	DevConfig []byte `gae:",noindex"`
	Updated   time.Time
}

func TestDeviceConfigsExists(t *testing.T) {
	t.Parallel()

	Convey("Test exists device config in datastore", t, func() {
		ctx := testingContext()
		tf, validate := newTestFixtureWithContext(ctx, t)
		defer validate()
		err := ds.Put(ctx, []devcfgEntity{
			{ID: "kunimitsu.lars.variant1"},
			{ID: "sarien.arcada.variant2"},
			{
				ID:        "platform.model.variant3",
				DevConfig: []byte("bad data"),
			},
		})
		So(err, ShouldBeNil)

		Convey("Happy path", func() {
			resp, err := tf.Inventory.DeviceConfigsExists(ctx, &api.DeviceConfigsExistsRequest{
				ConfigIds: []*device.ConfigId{
					{
						PlatformId: &device.PlatformId{Value: "lars"},
						ModelId:    &device.ModelId{Value: "lars"},
						VariantId:  &device.VariantId{Value: "variant1"},
					},
					{
						PlatformId: &device.PlatformId{Value: "arcada"},
						ModelId:    &device.ModelId{Value: "arcada"},
						VariantId:  &device.VariantId{Value: "variant2"},
					},
				},
			})
			So(err, ShouldBeNil)
			So(resp.Exists[0], ShouldBeTrue)
			So(resp.Exists[1], ShouldBeTrue)
		})

		Convey("check for nonexisting data", func() {
			resp, err := tf.Inventory.DeviceConfigsExists(ctx, &api.DeviceConfigsExistsRequest{
				ConfigIds: []*device.ConfigId{
					{
						PlatformId: &device.PlatformId{Value: "platform"},
						ModelId:    &device.ModelId{Value: "model"},
						VariantId:  &device.VariantId{Value: "variant-nonexisting"},
					},
				},
			})
			So(err, ShouldBeNil)
			So(resp.Exists[0], ShouldBeFalse)
		})

		Convey("check for existing and nonexisting data", func() {
			resp, err := tf.Inventory.DeviceConfigsExists(ctx, &api.DeviceConfigsExistsRequest{
				ConfigIds: []*device.ConfigId{
					{
						PlatformId: &device.PlatformId{Value: "platform"},
						ModelId:    &device.ModelId{Value: "model"},
						VariantId:  &device.VariantId{Value: "variant-nonexisting"},
					},
					{
						PlatformId: &device.PlatformId{Value: "arcada"},
						ModelId:    &device.ModelId{Value: "arcada"},
						VariantId:  &device.VariantId{Value: "variant2"},
					},
				},
			})
			So(err, ShouldBeNil)
			So(resp.Exists[0], ShouldBeFalse)
			So(resp.Exists[1], ShouldBeTrue)
		})
	})
}

func mockDeviceManualRepairRecord(hostname string, assetTag string, createdTime int64, completed bool) *invlibs.DeviceManualRepairRecord {
	var state invlibs.DeviceManualRepairRecord_RepairState
	var updatedTime timestamp.Timestamp
	var completedTime timestamp.Timestamp
	if completed {
		state = invlibs.DeviceManualRepairRecord_STATE_COMPLETED
		updatedTime = timestamp.Timestamp{Seconds: 444, Nanos: 0}
		completedTime = timestamp.Timestamp{Seconds: 444, Nanos: 0}
	} else {
		state = invlibs.DeviceManualRepairRecord_STATE_IN_PROGRESS
		updatedTime = timestamp.Timestamp{Seconds: 222, Nanos: 0}
		completedTime = timestamp.Timestamp{Seconds: 444, Nanos: 0}
	}

	return &invlibs.DeviceManualRepairRecord{
		Hostname:                        hostname,
		AssetTag:                        assetTag,
		RepairTargetType:                invlibs.DeviceManualRepairRecord_TYPE_DUT,
		RepairState:                     state,
		BuganizerBugUrl:                 "https://b/12345678",
		ChromiumBugUrl:                  "https://crbug.com/12345678",
		DutRepairFailureDescription:     "Mock DUT repair failure description.",
		DutVerifierFailureDescription:   "Mock DUT verifier failure description.",
		ServoRepairFailureDescription:   "Mock Servo repair failure description.",
		ServoVerifierFailureDescription: "Mock Servo verifier failure description.",
		Diagnosis:                       "Mock diagnosis.",
		RepairProcedure:                 "Mock repair procedure.",
		LabstationRepairActions: []invlibs.LabstationRepairAction{
			invlibs.LabstationRepairAction_LABSTATION_POWER_CYCLE,
			invlibs.LabstationRepairAction_LABSTATION_REIMAGE,
			invlibs.LabstationRepairAction_LABSTATION_UPDATE_CONFIG,
			invlibs.LabstationRepairAction_LABSTATION_REPLACE,
		},
		IssueFixed:    true,
		UserLdap:      "testing-account",
		TimeTaken:     15,
		CreatedTime:   &timestamp.Timestamp{Seconds: createdTime, Nanos: 0},
		UpdatedTime:   &updatedTime,
		CompletedTime: &completedTime,
	}
}

func mockServo(servoHost string) *lab.Servo {
	return &lab.Servo{
		ServoHostname: servoHost,
		ServoPort:     8888,
		ServoSerial:   "SERVO1",
		ServoType:     "v3",
	}
}

func mockDut(hostname, id, servoHost string) *lab.ChromeOSDevice {
	return &lab.ChromeOSDevice{
		Id: &lab.ChromeOSDeviceID{
			Value: id,
		},
		Device: &lab.ChromeOSDevice_Dut{
			Dut: &lab.DeviceUnderTest{
				Hostname: hostname,
				Peripherals: &lab.Peripherals{
					Servo:       mockServo(servoHost),
					SmartUsbhub: false,
				},
			},
		},
	}
}

func mockLabstation(hostname, id string) *lab.ChromeOSDevice {
	return &lab.ChromeOSDevice{
		Id: &lab.ChromeOSDeviceID{
			Value: id,
		},
		Device: &lab.ChromeOSDevice_Labstation{
			Labstation: &lab.Labstation{
				Hostname: hostname,
			},
		},
	}
}

func TestGetDeviceManualRepairRecord(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()

	ds.GetTestable(ctx).Consistent(true)

	record1 := mockDeviceManualRepairRecord("chromeos-getRecords-aa", "getRecords-111", 1, false)
	record2 := mockDeviceManualRepairRecord("chromeos-getRecords-bb", "getRecords-222", 1, false)
	record3 := mockDeviceManualRepairRecord("chromeos-getRecords-bb", "getRecords-333", 1, false)
	records := []*invlibs.DeviceManualRepairRecord{record1, record2, record3}

	// Set up records in datastore
	datastore.AddDeviceManualRepairRecords(ctx, records)

	Convey("Test get device manual repair records", t, func() {
		Convey("Get record using single hostname", func() {
			req := &api.GetDeviceManualRepairRecordRequest{
				Hostname: "chromeos-getRecords-aa",
			}
			resp, err := tf.Inventory.GetDeviceManualRepairRecord(tf.C, req)
			So(err, ShouldBeNil)
			So(resp.DeviceRepairRecord, ShouldNotBeNil)
		})
		Convey("Get first record when hostname has multiple active records", func() {
			req := &api.GetDeviceManualRepairRecordRequest{
				Hostname: "chromeos-getRecords-bb",
			}
			resp, err := tf.Inventory.GetDeviceManualRepairRecord(tf.C, req)
			So(resp.DeviceRepairRecord, ShouldNotBeNil)
			So(resp.DeviceRepairRecord.GetAssetTag(), ShouldEqual, "getRecords-222")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "More than one active record found")
		})
		Convey("Get record using non-existent hostname", func() {
			req := &api.GetDeviceManualRepairRecordRequest{
				Hostname: "chromeos-getRecords-cc",
			}
			resp, err := tf.Inventory.GetDeviceManualRepairRecord(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "No record found")
		})
		Convey("Get record using empty hostname", func() {
			req := &api.GetDeviceManualRepairRecordRequest{
				Hostname: "",
			}
			resp, err := tf.Inventory.GetDeviceManualRepairRecord(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "No record found")
		})
	})
}

func TestCreateDeviceManualRepairRecord(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()

	ds.GetTestable(ctx).Consistent(true)

	// Empty datastore
	record1 := mockDeviceManualRepairRecord("chromeos-createRecords-aa", "", 1, false)
	record2 := mockDeviceManualRepairRecord("", "", 1, false)

	// Set up records in datastore
	Convey("Test add devices using an empty datastore", t, func() {
		Convey("Add single record", func() {
			propFilter := map[string]string{"hostname": record1.Hostname}
			req := &api.CreateDeviceManualRepairRecordRequest{DeviceRepairRecord: record1}
			rsp, err := tf.Inventory.CreateDeviceManualRepairRecord(tf.C, req)
			So(rsp.String(), ShouldEqual, "")
			So(err, ShouldBeNil)

			// Check added record
			getRes, err := datastore.GetRepairRecordByPropertyName(ctx, propFilter, -1, 0, []string{})
			So(getRes, ShouldHaveLength, 1)
			So(getRes[0].Record.GetHostname(), ShouldEqual, "chromeos-createRecords-aa")
			So(getRes[0].Record.GetAssetTag(), ShouldEqual, "n/a")
			So(getRes[0].Record.GetCreatedTime(), ShouldNotResemble, &timestamp.Timestamp{Seconds: 1, Nanos: 0})
		})
		Convey("Add single record without hostname", func() {
			req := &api.CreateDeviceManualRepairRecordRequest{DeviceRepairRecord: record2}
			rsp, err := tf.Inventory.CreateDeviceManualRepairRecord(tf.C, req)
			So(rsp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Hostname cannot be empty")

			// No record should be added
			propFilter := map[string]string{"hostname": record2.Hostname}
			getRes, err := datastore.GetRepairRecordByPropertyName(ctx, propFilter, -1, 0, []string{})
			So(getRes, ShouldHaveLength, 0)
		})
		Convey("Add single record to a host with an open record", func() {
			// Check existing record
			propFilter := map[string]string{"hostname": record1.Hostname}
			getRes, err := datastore.GetRepairRecordByPropertyName(ctx, propFilter, -1, 0, []string{})
			So(getRes, ShouldHaveLength, 1)
			So(getRes[0].Record.GetHostname(), ShouldEqual, "chromeos-createRecords-aa")

			req := &api.CreateDeviceManualRepairRecordRequest{DeviceRepairRecord: record1}
			rsp, err := tf.Inventory.CreateDeviceManualRepairRecord(tf.C, req)
			So(rsp, ShouldBeNil)
			So(err.Error(), ShouldContainSubstring, "A record already exists for host chromeos-createRecords-aa")
		})
	})

	// Datastore with DeviceEntity
	record3 := mockDeviceManualRepairRecord("chromeos-createRecords-bb", "", 1, false)
	record4 := mockDeviceManualRepairRecord("chromeos-createRecords-cc", "", 1, false)
	record5 := mockDeviceManualRepairRecord("", "", 1, false)
	record6 := mockDeviceManualRepairRecord("chromeos-createRecords-ee", "", 1, true)

	Convey("Test add devices using an non-empty datastore", t, func() {
		dut1 := mockDut("chromeos-createRecords-bb", "mockDutAssetTag-111", "labstation1")
		dut2 := mockDut("chromeos-createRecords-cc", "", "labstation1")
		dut3 := mockDut("chromeos-createRecords-ee", "mockDutAssetTag-222", "labstation1")
		labstation1 := mockLabstation("labstation1", "assetId-111")
		dut1.DeviceConfigId = &device.ConfigId{ModelId: &device.ModelId{Value: "model1"}}
		dut2.DeviceConfigId = &device.ConfigId{ModelId: &device.ModelId{Value: "model2"}}
		dut3.DeviceConfigId = &device.ConfigId{ModelId: &device.ModelId{Value: "model3"}}
		labstation1.DeviceConfigId = &device.ConfigId{
			ModelId: &device.ModelId{Value: "model5"},
		}
		devsToAdd := []*lab.ChromeOSDevice{dut1, dut2, dut3, labstation1}
		_, err := datastore.AddDevices(ctx, devsToAdd, false)
		if err != nil {
			t.Fatal(err)
		}
		Convey("Add single record", func() {
			propFilter := map[string]string{"hostname": record3.Hostname}
			req := &api.CreateDeviceManualRepairRecordRequest{DeviceRepairRecord: record3}
			rsp, err := tf.Inventory.CreateDeviceManualRepairRecord(tf.C, req)
			So(rsp.String(), ShouldEqual, "")
			So(err, ShouldBeNil)

			// Check added record
			getRes, err := datastore.GetRepairRecordByPropertyName(ctx, propFilter, -1, 0, []string{})
			So(getRes, ShouldHaveLength, 1)
			So(getRes[0].Record.GetHostname(), ShouldEqual, "chromeos-createRecords-bb")
			So(getRes[0].Record.GetAssetTag(), ShouldEqual, "mockDutAssetTag-111")
			So(getRes[0].Record.GetCreatedTime(), ShouldNotResemble, &timestamp.Timestamp{Seconds: 1, Nanos: 0})
		})
		Convey("Add single record using dut without asset tag", func() {
			propFilter := map[string]string{"hostname": record4.Hostname}
			req := &api.CreateDeviceManualRepairRecordRequest{DeviceRepairRecord: record4}
			rsp, err := tf.Inventory.CreateDeviceManualRepairRecord(tf.C, req)
			So(rsp.String(), ShouldEqual, "")
			So(err, ShouldBeNil)

			// Asset tag should be uuid generated for dut
			getRes, err := datastore.GetRepairRecordByPropertyName(ctx, propFilter, -1, 0, []string{})
			So(getRes, ShouldHaveLength, 1)
			So(getRes[0].Record.GetHostname(), ShouldEqual, "chromeos-createRecords-cc")
			So(getRes[0].Record.GetAssetTag(), ShouldNotEqual, "")
			So(getRes[0].Record.GetAssetTag(), ShouldNotEqual, "n/a")
			So(getRes[0].Record.GetCreatedTime(), ShouldNotResemble, &timestamp.Timestamp{Seconds: 1, Nanos: 0})
		})
		Convey("Add single record with no hostname", func() {
			req := &api.CreateDeviceManualRepairRecordRequest{DeviceRepairRecord: record5}
			rsp, err := tf.Inventory.CreateDeviceManualRepairRecord(tf.C, req)
			So(rsp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Hostname cannot be empty")
		})
		Convey("Add single record with completed repair state", func() {
			propFilter := map[string]string{"hostname": record6.Hostname}
			req := &api.CreateDeviceManualRepairRecordRequest{DeviceRepairRecord: record6}
			rsp, err := tf.Inventory.CreateDeviceManualRepairRecord(tf.C, req)
			So(rsp.String(), ShouldEqual, "")
			So(err, ShouldBeNil)

			// Completed time should be same as created
			getRes, err := datastore.GetRepairRecordByPropertyName(ctx, propFilter, -1, 0, []string{})
			So(getRes, ShouldHaveLength, 1)
			So(getRes[0].Record.GetHostname(), ShouldEqual, "chromeos-createRecords-ee")
			So(getRes[0].Record.GetAssetTag(), ShouldEqual, "mockDutAssetTag-222")
			So(getRes[0].Record.GetCreatedTime(), ShouldNotResemble, &timestamp.Timestamp{Seconds: 1, Nanos: 0})
			So(getRes[0].Record.GetCreatedTime(), ShouldResembleProto, getRes[0].Record.GetCompletedTime())
		})
	})
}

func TestUpdateDeviceManualRepairRecord(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()

	ds.GetTestable(ctx).Consistent(true)

	// Empty datastore
	record1 := mockDeviceManualRepairRecord("chromeos-updateRecords-aa", "updateRec-111", 1, false)
	record1Complete := mockDeviceManualRepairRecord("chromeos-updateRecords-aa", "updateRec-111", 1, true)
	record2 := mockDeviceManualRepairRecord("chromeos-updateRecords-bb", "updateRec-222", 1, false)
	record2Complete := mockDeviceManualRepairRecord("chromeos-updateRecords-bb", "updateRec-222", 1, true)
	record3 := mockDeviceManualRepairRecord("chromeos-updateRecords-cc", "updateRec-333", 1, false)
	record3Update := mockDeviceManualRepairRecord("chromeos-updateRecords-cc", "updateRec-333", 1, false)
	record4 := mockDeviceManualRepairRecord("chromeos-updateRecords-dd", "updateRec-444", 1, false)

	// Set up records in datastore
	records := []*invlibs.DeviceManualRepairRecord{record1, record2, record3}
	datastore.AddDeviceManualRepairRecords(ctx, records)

	Convey("Test update devices using an non-empty datastore", t, func() {
		Convey("Update single record with completed repair state", func() {
			propFilter := map[string]string{"hostname": record1.Hostname}
			getRes, err := datastore.GetRepairRecordByPropertyName(ctx, propFilter, -1, 0, []string{})
			req := &api.UpdateDeviceManualRepairRecordRequest{
				Id:                 getRes[0].Entity.ID,
				DeviceRepairRecord: record1Complete,
			}
			rsp, err := tf.Inventory.UpdateDeviceManualRepairRecord(tf.C, req)
			So(rsp.String(), ShouldEqual, "")
			So(err, ShouldBeNil)

			// Check updated record
			getRes, err = datastore.GetRepairRecordByPropertyName(ctx, propFilter, -1, 0, []string{})
			So(getRes, ShouldHaveLength, 1)
			So(getRes[0].Record.GetHostname(), ShouldEqual, "chromeos-updateRecords-aa")
			So(getRes[0].Record.GetRepairState(), ShouldEqual, invlibs.DeviceManualRepairRecord_STATE_COMPLETED)
			So(getRes[0].Record.GetUpdatedTime(), ShouldNotResemble, &timestamp.Timestamp{Seconds: 222, Nanos: 0})
			So(getRes[0].Record.GetUpdatedTime(), ShouldResembleProto, getRes[0].Record.GetCompletedTime())
		})
		Convey("Update single record with no id", func() {
			req := &api.UpdateDeviceManualRepairRecordRequest{
				Id:                 "",
				DeviceRepairRecord: record2Complete,
			}
			rsp, err := tf.Inventory.UpdateDeviceManualRepairRecord(tf.C, req)
			So(rsp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "ID cannot be empty")

			// Check updated record and make sure it is unchanged
			getRes, err := datastore.GetRepairRecordByPropertyName(ctx, map[string]string{"hostname": record2.Hostname}, -1, 0, []string{})
			So(getRes, ShouldHaveLength, 1)
			So(getRes[0].Record.GetHostname(), ShouldEqual, "chromeos-updateRecords-bb")
			So(getRes[0].Record.GetRepairState(), ShouldEqual, invlibs.DeviceManualRepairRecord_STATE_IN_PROGRESS)
			So(getRes[0].Record.GetUpdatedTime(), ShouldResembleProto, &timestamp.Timestamp{Seconds: 222, Nanos: 0})
		})
		Convey("Update single record", func() {
			propFilter := map[string]string{"hostname": record3.Hostname}
			getRes, err := datastore.GetRepairRecordByPropertyName(ctx, propFilter, -1, 0, []string{})
			record3Update.TimeTaken = 20
			req := &api.UpdateDeviceManualRepairRecordRequest{
				Id:                 getRes[0].Entity.ID,
				DeviceRepairRecord: record3Update,
			}
			rsp, err := tf.Inventory.UpdateDeviceManualRepairRecord(tf.C, req)
			So(rsp.String(), ShouldEqual, "")
			So(err, ShouldBeNil)

			// Check updated record and make sure fields are changed properly
			getRes, err = datastore.GetRepairRecordByPropertyName(ctx, propFilter, -1, 0, []string{})
			So(getRes, ShouldHaveLength, 1)
			So(getRes[0].Record.GetHostname(), ShouldEqual, "chromeos-updateRecords-cc")
			So(getRes[0].Record.GetRepairState(), ShouldEqual, invlibs.DeviceManualRepairRecord_STATE_IN_PROGRESS)
			So(getRes[0].Record.GetTimeTaken(), ShouldEqual, 20)
			So(getRes[0].Record.GetUpdatedTime(), ShouldNotResemble, &timestamp.Timestamp{Seconds: 222, Nanos: 0})
			So(getRes[0].Record.GetCompletedTime(), ShouldResembleProto, &timestamp.Timestamp{Seconds: 444, Nanos: 0})
		})
		Convey("Update single non-existent record", func() {
			propFilter := map[string]string{"hostname": record4.Hostname}
			getRes, err := datastore.GetRepairRecordByPropertyName(ctx, propFilter, -1, 0, []string{})
			So(getRes, ShouldHaveLength, 0)

			req := &api.UpdateDeviceManualRepairRecordRequest{
				Id:                 "test-id",
				DeviceRepairRecord: record4,
			}
			rsp, err := tf.Inventory.UpdateDeviceManualRepairRecord(tf.C, req)
			So(rsp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "No open record exists for host chromeos-updateRecords-dd")
		})
	})
}

func TestListManualRepairRecords(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()

	ds.GetTestable(ctx).AutoIndex(true)
	ds.GetTestable(ctx).Consistent(true)

	// Updated times should go in descending order of record1 > record2 = record3
	record1 := mockDeviceManualRepairRecord("chromeos-getRecords-aa", "getRecords-111", 1, true)
	record2 := mockDeviceManualRepairRecord("chromeos-getRecords-aa", "getRecords-111", 2, false)
	record3 := mockDeviceManualRepairRecord("chromeos-getRecords-aa", "getRecords-222", 3, false)
	records := []*invlibs.DeviceManualRepairRecord{record1, record2, record3}

	// Set up records in datastore
	datastore.AddDeviceManualRepairRecords(ctx, records)

	Convey("Test list device manual repair records", t, func() {
		Convey("List records using hostname and asset tag", func() {
			req := &api.ListManualRepairRecordsRequest{
				Hostname: "chromeos-getRecords-aa",
				AssetTag: "getRecords-111",
				Limit:    5,
			}
			resp, err := tf.Inventory.ListManualRepairRecords(tf.C, req)

			So(err, ShouldBeNil)
			So(resp.RepairRecords, ShouldNotBeNil)
			So(resp.RepairRecords, ShouldHaveLength, 2)
			So(resp.RepairRecords[0].GetHostname(), ShouldEqual, "chromeos-getRecords-aa")
			So(resp.RepairRecords[0].GetAssetTag(), ShouldEqual, "getRecords-111")
			So(resp.RepairRecords[0].GetRepairState(), ShouldEqual, invlibs.DeviceManualRepairRecord_STATE_IN_PROGRESS)
			So(resp.RepairRecords[1].GetHostname(), ShouldEqual, "chromeos-getRecords-aa")
			So(resp.RepairRecords[1].GetAssetTag(), ShouldEqual, "getRecords-111")
			So(resp.RepairRecords[1].GetRepairState(), ShouldEqual, invlibs.DeviceManualRepairRecord_STATE_COMPLETED)
		})
		Convey("List records using hostname and asset tag with offset", func() {
			req := &api.ListManualRepairRecordsRequest{
				Hostname: "chromeos-getRecords-aa",
				AssetTag: "getRecords-111",
				Limit:    1,
				Offset:   1,
			}
			resp, err := tf.Inventory.ListManualRepairRecords(tf.C, req)

			So(err, ShouldBeNil)
			So(resp.RepairRecords, ShouldNotBeNil)
			So(resp.RepairRecords, ShouldHaveLength, 1)
			So(resp.RepairRecords[0].GetHostname(), ShouldEqual, "chromeos-getRecords-aa")
			So(resp.RepairRecords[0].GetAssetTag(), ShouldEqual, "getRecords-111")
			So(resp.RepairRecords[0].GetRepairState(), ShouldEqual, invlibs.DeviceManualRepairRecord_STATE_COMPLETED)
		})
		Convey("List records using all filters", func() {
			req := &api.ListManualRepairRecordsRequest{
				Hostname:    "chromeos-getRecords-aa",
				AssetTag:    "getRecords-111",
				Limit:       5,
				UserLdap:    "testing-account",
				RepairState: "STATE_COMPLETED",
			}
			resp, err := tf.Inventory.ListManualRepairRecords(tf.C, req)

			So(err, ShouldBeNil)
			So(resp.RepairRecords, ShouldNotBeNil)
			So(resp.RepairRecords, ShouldHaveLength, 1)
			So(resp.RepairRecords[0].GetHostname(), ShouldEqual, "chromeos-getRecords-aa")
			So(resp.RepairRecords[0].GetAssetTag(), ShouldEqual, "getRecords-111")
			So(resp.RepairRecords[0].GetRepairState(), ShouldEqual, invlibs.DeviceManualRepairRecord_STATE_COMPLETED)
		})
		Convey("List records using hostname and asset tag with limit 1", func() {
			req := &api.ListManualRepairRecordsRequest{
				Hostname: "chromeos-getRecords-aa",
				AssetTag: "getRecords-111",
				Limit:    1,
			}
			resp, err := tf.Inventory.ListManualRepairRecords(tf.C, req)

			So(err, ShouldBeNil)
			So(resp.RepairRecords, ShouldNotBeNil)
			So(resp.RepairRecords, ShouldHaveLength, 1)
			So(resp.RepairRecords[0].GetHostname(), ShouldEqual, "chromeos-getRecords-aa")
			So(resp.RepairRecords[0].GetAssetTag(), ShouldEqual, "getRecords-111")
			So(resp.RepairRecords[0].GetRepairState(), ShouldEqual, invlibs.DeviceManualRepairRecord_STATE_IN_PROGRESS)
		})
		Convey("List records that do not exist", func() {
			req := &api.ListManualRepairRecordsRequest{
				Hostname: "chromeos-getRecords-bb",
				AssetTag: "getRecords-111",
				Limit:    5,
			}
			resp, err := tf.Inventory.ListManualRepairRecords(tf.C, req)

			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp.RepairRecords, ShouldHaveLength, 0)
		})
	})
}

func TestBatchGetManualRepairRecords(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()

	ds.GetTestable(ctx).Consistent(true)

	record1 := mockDeviceManualRepairRecord("chromeos-getRecords-xx", "getRecords-111", 1, false)
	record2 := mockDeviceManualRepairRecord("chromeos-getRecords-yy", "getRecords-222", 1, false)
	record3 := mockDeviceManualRepairRecord("chromeos-getRecords-zz", "getRecords-333", 1, false)
	record4 := mockDeviceManualRepairRecord("chromeos-getRecords-zz", "getRecords-444", 1, false)
	records := []*invlibs.DeviceManualRepairRecord{record1, record2, record3, record4}

	// Set up records in datastore
	datastore.AddDeviceManualRepairRecords(ctx, records)

	Convey("Test batch get manual repair records", t, func() {
		Convey("Get record using multiple hostnames", func() {
			req := &api.BatchGetManualRepairRecordsRequest{
				Hostnames: []string{
					"chromeos-getRecords-xx",
					"chromeos-getRecords-yy",
				},
			}
			resp, err := tf.Inventory.BatchGetManualRepairRecords(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp.RepairRecords, ShouldHaveLength, 2)
			So(resp.RepairRecords[0].ErrorMsg, ShouldBeEmpty)
			So(resp.RepairRecords[0].RepairRecord, ShouldNotBeNil)
			So(resp.RepairRecords[0].RepairRecord.Hostname, ShouldEqual, "chromeos-getRecords-xx")
			So(resp.RepairRecords[0].Hostname, ShouldEqual, "chromeos-getRecords-xx")
			So(resp.RepairRecords[1].ErrorMsg, ShouldBeEmpty)
			So(resp.RepairRecords[1].RepairRecord, ShouldNotBeNil)
			So(resp.RepairRecords[1].RepairRecord.Hostname, ShouldEqual, "chromeos-getRecords-yy")
			So(resp.RepairRecords[1].Hostname, ShouldEqual, "chromeos-getRecords-yy")
		})
		Convey("Get first record when hostname has multiple active records", func() {
			req := &api.BatchGetManualRepairRecordsRequest{
				Hostnames: []string{
					"chromeos-getRecords-zz",
				},
			}
			resp, err := tf.Inventory.BatchGetManualRepairRecords(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp.RepairRecords, ShouldHaveLength, 1)
			So(resp.RepairRecords[0].RepairRecord.Hostname, ShouldEqual, "chromeos-getRecords-zz")
			So(resp.RepairRecords[0].Hostname, ShouldEqual, "chromeos-getRecords-zz")
		})
		Convey("Get record using a non-existent hostname", func() {
			req := &api.BatchGetManualRepairRecordsRequest{
				Hostnames: []string{
					"chromeos-getRecords-xx",
					"chromeos-getRecords-cc",
				},
			}
			resp, err := tf.Inventory.BatchGetManualRepairRecords(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp.RepairRecords, ShouldHaveLength, 2)
			So(resp.RepairRecords[0].ErrorMsg, ShouldBeEmpty)
			So(resp.RepairRecords[0].RepairRecord, ShouldNotBeNil)
			So(resp.RepairRecords[0].RepairRecord.Hostname, ShouldEqual, "chromeos-getRecords-xx")
			So(resp.RepairRecords[0].Hostname, ShouldEqual, "chromeos-getRecords-xx")
			So(resp.RepairRecords[1].ErrorMsg, ShouldContainSubstring, "No record found")
			So(resp.RepairRecords[1].RepairRecord, ShouldBeNil)
			So(resp.RepairRecords[1].Hostname, ShouldEqual, "chromeos-getRecords-cc")
		})
	})
}

func TestBatchCreateManualRepairRecords(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()

	ds.GetTestable(ctx).Consistent(true)

	// Empty datastore
	record1 := mockDeviceManualRepairRecord("chromeos-createRecords-zz", "", 1, false)
	record2 := mockDeviceManualRepairRecord("chromeos-createRecords-yy", "", 1, false)
	record3 := mockDeviceManualRepairRecord("chromeos-createRecords-xx", "", 1, false)
	record4 := mockDeviceManualRepairRecord("chromeos-createRecords-ww", "", 1, false)
	record5 := mockDeviceManualRepairRecord("", "", 1, false)

	// Set up records in datastore
	Convey("Test add devices using an empty datastore", t, func() {
		Convey("Add single record", func() {
			createReq := &api.BatchCreateManualRepairRecordsRequest{
				RepairRecords: []*invlibs.DeviceManualRepairRecord{record1},
			}
			createRsp, err := tf.Inventory.BatchCreateManualRepairRecords(tf.C, createReq)
			So(err, ShouldBeNil)
			So(createRsp, ShouldNotBeNil)
			So(createRsp.RepairRecords, ShouldHaveLength, 1)
			So(createRsp.RepairRecords[0].ErrorMsg, ShouldBeEmpty)
			So(createRsp.RepairRecords[0].RepairRecord, ShouldNotBeNil)
			So(createRsp.RepairRecords[0].RepairRecord.Hostname, ShouldEqual, "chromeos-createRecords-zz")
			So(createRsp.RepairRecords[0].Hostname, ShouldEqual, "chromeos-createRecords-zz")

			// Check added record
			getReq := &api.BatchGetManualRepairRecordsRequest{
				Hostnames: []string{
					"chromeos-createRecords-zz",
				},
			}
			getRsp, err := tf.Inventory.BatchGetManualRepairRecords(tf.C, getReq)
			So(err, ShouldBeNil)
			So(getRsp, ShouldNotBeNil)
			So(getRsp.RepairRecords, ShouldHaveLength, 1)
			So(getRsp.RepairRecords[0].ErrorMsg, ShouldBeEmpty)
			So(getRsp.RepairRecords[0].RepairRecord, ShouldNotBeNil)
			So(getRsp.RepairRecords[0].RepairRecord.Hostname, ShouldEqual, "chromeos-createRecords-zz")
			So(getRsp.RepairRecords[0].Hostname, ShouldEqual, "chromeos-createRecords-zz")
		})
		Convey("Add multiple records", func() {
			createReq := &api.BatchCreateManualRepairRecordsRequest{
				RepairRecords: []*invlibs.DeviceManualRepairRecord{record2, record3},
			}
			createRsp, err := tf.Inventory.BatchCreateManualRepairRecords(tf.C, createReq)
			So(err, ShouldBeNil)
			So(createRsp, ShouldNotBeNil)
			So(createRsp.RepairRecords, ShouldHaveLength, 2)
			So(createRsp.RepairRecords[0].ErrorMsg, ShouldBeEmpty)
			So(createRsp.RepairRecords[0].RepairRecord, ShouldNotBeNil)
			So(createRsp.RepairRecords[0].RepairRecord.Hostname, ShouldEqual, "chromeos-createRecords-xx")
			So(createRsp.RepairRecords[0].Hostname, ShouldEqual, "chromeos-createRecords-xx")
			So(createRsp.RepairRecords[1].ErrorMsg, ShouldBeEmpty)
			So(createRsp.RepairRecords[1].RepairRecord, ShouldNotBeNil)
			So(createRsp.RepairRecords[1].RepairRecord.Hostname, ShouldEqual, "chromeos-createRecords-yy")
			So(createRsp.RepairRecords[1].Hostname, ShouldEqual, "chromeos-createRecords-yy")

			// Check added record
			getReq := &api.BatchGetManualRepairRecordsRequest{
				Hostnames: []string{
					"chromeos-createRecords-yy",
					"chromeos-createRecords-xx",
				},
			}
			getRsp, err := tf.Inventory.BatchGetManualRepairRecords(tf.C, getReq)
			So(err, ShouldBeNil)
			So(getRsp, ShouldNotBeNil)
			So(getRsp.RepairRecords, ShouldHaveLength, 2)
			So(getRsp.RepairRecords[0].ErrorMsg, ShouldBeEmpty)
			So(getRsp.RepairRecords[0].RepairRecord, ShouldNotBeNil)
			So(getRsp.RepairRecords[0].RepairRecord.Hostname, ShouldEqual, "chromeos-createRecords-yy")
			So(getRsp.RepairRecords[0].Hostname, ShouldEqual, "chromeos-createRecords-yy")
			So(getRsp.RepairRecords[1].ErrorMsg, ShouldBeEmpty)
			So(getRsp.RepairRecords[1].RepairRecord, ShouldNotBeNil)
			So(getRsp.RepairRecords[1].RepairRecord.Hostname, ShouldEqual, "chromeos-createRecords-xx")
			So(getRsp.RepairRecords[1].Hostname, ShouldEqual, "chromeos-createRecords-xx")
		})
		Convey("Add multiple records; one with an open record", func() {
			// Check existing record
			propFilter := map[string]string{"hostname": record1.Hostname}
			getRes, err := datastore.GetRepairRecordByPropertyName(ctx, propFilter, -1, 0, []string{})
			So(getRes, ShouldHaveLength, 1)
			So(getRes[0].Record.GetHostname(), ShouldEqual, "chromeos-createRecords-zz")

			createReq := &api.BatchCreateManualRepairRecordsRequest{
				RepairRecords: []*invlibs.DeviceManualRepairRecord{record1, record4},
			}
			createRsp, err := tf.Inventory.BatchCreateManualRepairRecords(tf.C, createReq)
			So(err, ShouldBeNil)
			So(createRsp, ShouldNotBeNil)
			So(createRsp.RepairRecords, ShouldHaveLength, 2)
			So(createRsp.RepairRecords[0].ErrorMsg, ShouldBeEmpty)
			So(createRsp.RepairRecords[0].RepairRecord, ShouldNotBeNil)
			So(createRsp.RepairRecords[0].RepairRecord.Hostname, ShouldEqual, "chromeos-createRecords-ww")
			So(createRsp.RepairRecords[0].Hostname, ShouldEqual, "chromeos-createRecords-ww")
			So(createRsp.RepairRecords[1].ErrorMsg, ShouldContainSubstring, "A record already exists for host chromeos-createRecords-zz")
			So(createRsp.RepairRecords[1].RepairRecord, ShouldBeNil)
			So(createRsp.RepairRecords[1].Hostname, ShouldEqual, "chromeos-createRecords-zz")
		})
		Convey("Add single record without hostname", func() {
			createReq := &api.BatchCreateManualRepairRecordsRequest{
				RepairRecords: []*invlibs.DeviceManualRepairRecord{record5},
			}
			createRsp, err := tf.Inventory.BatchCreateManualRepairRecords(tf.C, createReq)
			So(err, ShouldBeNil)
			So(createRsp, ShouldNotBeNil)
			So(createRsp.RepairRecords, ShouldHaveLength, 1)
			So(createRsp.RepairRecords[0].ErrorMsg, ShouldContainSubstring, "Hostname cannot be empty")
			So(createRsp.RepairRecords[0].Hostname, ShouldBeEmpty)

			// No record should be added
			propFilter := map[string]string{"hostname": record5.Hostname}
			getRes, err := datastore.GetRepairRecordByPropertyName(ctx, propFilter, -1, 0, []string{})
			So(getRes, ShouldHaveLength, 0)
		})
	})
}
