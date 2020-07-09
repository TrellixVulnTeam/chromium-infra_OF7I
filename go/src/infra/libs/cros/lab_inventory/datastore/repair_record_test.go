package datastore

import (
	"testing"

	timestamp "github.com/golang/protobuf/ptypes/timestamp"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/luci/appengine/gaetesting"

	inv "infra/appengine/cros/lab_inventory/api/v1"
)

func mockDeviceManualRepairRecord(hostname string, assetTag string) *inv.DeviceManualRepairRecord {
	return &inv.DeviceManualRepairRecord{
		Hostname:                        hostname,
		AssetTag:                        assetTag,
		RepairTargetType:                inv.DeviceManualRepairRecord_TYPE_DUT,
		RepairState:                     inv.DeviceManualRepairRecord_STATE_INVALID,
		BuganizerBugUrl:                 "https://b/12345678",
		ChromiumBugUrl:                  "https://crbug.com/12345678",
		DutRepairFailureDescription:     "Mock DUT repair failure description.",
		DutVerifierFailureDescription:   "Mock DUT verifier failure description.",
		ServoRepairFailureDescription:   "Mock Servo repair failure description.",
		ServoVerifierFailureDescription: "Mock Servo verifier failure description.",
		Diagnosis:                       "Mock diagnosis.",
		RepairProcedure:                 "Mock repair procedure.",
		ManualRepairActions: []inv.DeviceManualRepairRecord_ManualRepairAction{
			inv.DeviceManualRepairRecord_ACTION_FIX_SERVO,
			inv.DeviceManualRepairRecord_ACTION_FIX_YOSHI_CABLE,
			inv.DeviceManualRepairRecord_ACTION_VISUAL_INSPECTION,
			inv.DeviceManualRepairRecord_ACTION_REIMAGE_DUT,
		},
		TimeTaken:     15,
		CreatedTime:   &timestamp.Timestamp{Seconds: 111, Nanos: 0},
		UpdatedTime:   &timestamp.Timestamp{Seconds: 222, Nanos: 0},
		CompletedTime: &timestamp.Timestamp{Seconds: 222, Nanos: 0},
	}
}

func mockUpdatedRecord(hostname string, assetTag string) *inv.DeviceManualRepairRecord {
	return &inv.DeviceManualRepairRecord{
		Hostname:                        hostname,
		AssetTag:                        assetTag,
		RepairTargetType:                inv.DeviceManualRepairRecord_TYPE_DUT,
		RepairState:                     inv.DeviceManualRepairRecord_STATE_NOT_STARTED,
		BuganizerBugUrl:                 "https://b/12345678",
		ChromiumBugUrl:                  "https://crbug.com/12345678",
		DutRepairFailureDescription:     "Mock DUT repair failure description.",
		DutVerifierFailureDescription:   "Mock DUT verifier failure description.",
		ServoRepairFailureDescription:   "Mock Servo repair failure description.",
		ServoVerifierFailureDescription: "Mock Servo verifier failure description.",
		Diagnosis:                       "Mock diagnosis.",
		RepairProcedure:                 "Mock repair procedure.",
		ManualRepairActions: []inv.DeviceManualRepairRecord_ManualRepairAction{
			inv.DeviceManualRepairRecord_ACTION_FIX_SERVO,
			inv.DeviceManualRepairRecord_ACTION_FIX_YOSHI_CABLE,
			inv.DeviceManualRepairRecord_ACTION_VISUAL_INSPECTION,
			inv.DeviceManualRepairRecord_ACTION_REIMAGE_DUT,
		},
		TimeTaken:     30,
		CreatedTime:   &timestamp.Timestamp{Seconds: 111, Nanos: 0},
		UpdatedTime:   &timestamp.Timestamp{Seconds: 222, Nanos: 0},
		CompletedTime: &timestamp.Timestamp{Seconds: 222, Nanos: 0},
	}
}

func TestAddRecord(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	record1 := mockDeviceManualRepairRecord("chromeosxx-rowxx-rackxx-hostxx", "xxxxxxxx")
	record2 := mockDeviceManualRepairRecord("chromeosyy-rowyy-rackyy-hostyy", "yyyyyyyy")
	record3 := mockDeviceManualRepairRecord("chromeoszz-rowzz-rackzz-hostzz", "zzzzzzzz")
	record4 := mockDeviceManualRepairRecord("chromeosaa-rowaa-rackaa-hostaa", "aaaaaaaa")
	record5 := mockDeviceManualRepairRecord("", "")

	rec1ID, _ := generateRepairRecordID(record1.Hostname, record1.AssetTag, record1.CreatedTime.String())
	rec2ID, _ := generateRepairRecordID(record2.Hostname, record2.AssetTag, record2.CreatedTime.String())
	ids1 := []string{rec1ID, rec2ID}

	rec3ID, _ := generateRepairRecordID(record3.Hostname, record3.AssetTag, record3.CreatedTime.String())
	rec4ID, _ := generateRepairRecordID(record4.Hostname, record4.AssetTag, record4.CreatedTime.String())
	ids2 := []string{rec3ID, rec4ID}
	Convey("Add device manual repair record to datastore", t, func() {
		Convey("Add multiple device manual repair records to datastore", func() {
			records := []*inv.DeviceManualRepairRecord{record1, record2}
			res, err := AddDeviceManualRepairRecords(ctx, records)
			So(err, ShouldBeNil)
			So(res, ShouldHaveLength, 2)
			for i, r := range records {
				So(res[i].Err, ShouldBeNil)
				So(res[i].Entity.Hostname, ShouldEqual, r.GetHostname())
				So(res[i].Entity.AssetTag, ShouldEqual, r.GetAssetTag())
				So(res[i].Entity.RepairState, ShouldEqual, r.GetRepairState().String())
			}

			res = GetDeviceManualRepairRecords(ctx, ids1)
			So(res, ShouldHaveLength, 2)
			for i, r := range records {
				So(res[i].Err, ShouldBeNil)
				So(res[i].Entity.Hostname, ShouldEqual, r.GetHostname())
				So(res[i].Entity.AssetTag, ShouldEqual, r.GetAssetTag())
				So(res[i].Entity.RepairState, ShouldEqual, r.GetRepairState().String())
			}
		})
		Convey("Add existing record to datastore", func() {
			req := []*inv.DeviceManualRepairRecord{record3}
			res, err := AddDeviceManualRepairRecords(ctx, req)
			So(err, ShouldBeNil)
			So(res, ShouldHaveLength, 1)
			So(res[0].Err, ShouldBeNil)

			// Verify adding existing record.
			req = []*inv.DeviceManualRepairRecord{record3, record4}
			res, err = AddDeviceManualRepairRecords(ctx, req)
			So(err, ShouldBeNil)
			So(res, ShouldNotBeNil)
			So(res, ShouldHaveLength, 2)
			So(res[0].Err, ShouldNotBeNil)
			So(res[0].Err.Error(), ShouldContainSubstring, "Record exists in the datastore")
			So(res[1].Err, ShouldBeNil)
			So(res[1].Entity.Hostname, ShouldEqual, record4.GetHostname())

			// Check both records are in datastore.
			res = GetDeviceManualRepairRecords(ctx, ids2)
			So(res, ShouldHaveLength, 2)
			for i, r := range req {
				So(res[i].Err, ShouldBeNil)
				So(res[i].Entity.Hostname, ShouldEqual, r.GetHostname())
				So(res[i].Entity.AssetTag, ShouldEqual, r.GetAssetTag())
				So(res[i].Entity.RepairState, ShouldEqual, r.GetRepairState().String())
			}
		})
		Convey("Add record without hostname to datastore", func() {
			req := []*inv.DeviceManualRepairRecord{record5}
			res, err := AddDeviceManualRepairRecords(ctx, req)
			So(err, ShouldBeNil)
			So(res, ShouldHaveLength, 1)
			So(res[0].Err, ShouldNotBeNil)
			So(res[0].Err.Error(), ShouldContainSubstring, "Hostname cannot be empty")
		})
	})
}

func TestGetRecord(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	record1 := mockDeviceManualRepairRecord("chromeosyy-rowyy-rackyy-hostyy", "yyyyyyyy")
	record2 := mockDeviceManualRepairRecord("chromeoszz-rowzz-rackzz-hostzz", "12345678")
	rec1ID, _ := generateRepairRecordID(record1.Hostname, record1.AssetTag, record1.CreatedTime.String())
	rec2ID, _ := generateRepairRecordID(record2.Hostname, record2.AssetTag, record2.CreatedTime.String())
	ids1 := []string{rec1ID, rec2ID}
	Convey("Get device manual repair record from datastore", t, func() {
		Convey("Get non-existent device manual repair record from datastore", func() {
			records := []*inv.DeviceManualRepairRecord{record1}
			res, err := AddDeviceManualRepairRecords(ctx, records)
			So(err, ShouldBeNil)
			So(res, ShouldHaveLength, 1)

			res = GetDeviceManualRepairRecords(ctx, ids1)
			So(res, ShouldHaveLength, 2)
			So(res[0].Err, ShouldBeNil)
			So(res[1].Err, ShouldNotBeNil)
			So(res[1].Err.Error(), ShouldContainSubstring, "datastore: no such entity")
		})
		Convey("Get record with empty id", func() {
			res := GetDeviceManualRepairRecords(ctx, []string{""})
			So(res, ShouldHaveLength, 1)
			So(res[0].Err, ShouldNotBeNil)
			So(res[0].Err.Error(), ShouldContainSubstring, "datastore: invalid key")
		})
	})
}

func TestUpdateRecord(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	record1 := mockDeviceManualRepairRecord("chromeosxx-rowxx-rackxx-hostxx", "xxxxxxxx")
	record1Update := mockUpdatedRecord("chromeosxx-rowxx-rackxx-hostxx", "xxxxxxxx")

	record2 := mockDeviceManualRepairRecord("chromeosyy-rowyy-rackyy-hostyy", "yyyyyyyy")

	record3 := mockDeviceManualRepairRecord("chromeoszz-rowzz-rackzz-hostzz", "zzzzzzzz")
	record3Update := mockUpdatedRecord("chromeoszz-rowzz-rackzz-hostzz", "zzzzzzzz")
	record4 := mockDeviceManualRepairRecord("chromeosaa-rowaa-rackaa-hostaa", "aaaaaaaa")

	record5 := mockDeviceManualRepairRecord("", "")
	Convey("Update record in datastore", t, func() {
		Convey("Update existing record to datastore", func() {
			rec1ID, _ := generateRepairRecordID(record1.Hostname, record1.AssetTag, record1.CreatedTime.String())
			req := []*inv.DeviceManualRepairRecord{record1}
			res, err := AddDeviceManualRepairRecords(ctx, req)
			So(err, ShouldBeNil)
			So(res, ShouldNotBeNil)
			So(res, ShouldHaveLength, 1)
			So(res[0].Err, ShouldBeNil)

			res = GetDeviceManualRepairRecords(ctx, []string{rec1ID})
			So(res, ShouldHaveLength, 1)
			So(res[0].Err, ShouldBeNil)
			So(res[0].Entity.RepairState, ShouldEqual, record1.GetRepairState().String())

			// Update and check
			reqUpdate := map[string]*inv.DeviceManualRepairRecord{rec1ID: record1Update}
			res2, err := UpdateDeviceManualRepairRecords(ctx, reqUpdate)
			So(err, ShouldBeNil)
			So(res2, ShouldHaveLength, 1)
			So(res2[0].Err, ShouldBeNil)

			res = GetDeviceManualRepairRecords(ctx, []string{rec1ID})
			So(res, ShouldHaveLength, 1)
			So(res[0].Err, ShouldBeNil)
			So(res[0].Entity.RepairState, ShouldEqual, record1Update.GetRepairState().String())
		})
		Convey("Update non-existent record in datastore", func() {
			rec2ID, _ := generateRepairRecordID(record2.Hostname, record2.AssetTag, record2.CreatedTime.String())
			reqUpdate := map[string]*inv.DeviceManualRepairRecord{rec2ID: record2}
			res, err := UpdateDeviceManualRepairRecords(ctx, reqUpdate)
			So(err, ShouldBeNil)
			So(res, ShouldHaveLength, 1)
			So(res[0].Err, ShouldNotBeNil)
			So(res[0].Err.Error(), ShouldContainSubstring, "datastore: no such entity")

			res = GetDeviceManualRepairRecords(ctx, []string{rec2ID})
			So(res, ShouldHaveLength, 1)
			So(res[0].Err, ShouldNotBeNil)
		})
		Convey("Update multiple records to datastore", func() {
			rec3ID, _ := generateRepairRecordID(record3.Hostname, record3.AssetTag, record3.CreatedTime.String())
			rec4ID, _ := generateRepairRecordID(record4.Hostname, record4.AssetTag, record4.CreatedTime.String())
			req := []*inv.DeviceManualRepairRecord{record3, record4}
			res, err := AddDeviceManualRepairRecords(ctx, req)
			So(err, ShouldBeNil)
			So(res, ShouldNotBeNil)
			So(res, ShouldHaveLength, 2)
			So(res[0].Err, ShouldBeNil)
			So(res[1].Err, ShouldBeNil)

			reqUpdate := map[string]*inv.DeviceManualRepairRecord{
				rec3ID: record3Update,
				rec4ID: record4,
			}
			res, err = UpdateDeviceManualRepairRecords(ctx, reqUpdate)
			So(err, ShouldBeNil)
			So(res, ShouldHaveLength, 2)
			So(res[0].Err, ShouldBeNil)
			So(res[1].Err, ShouldBeNil)

			res = GetDeviceManualRepairRecords(ctx, []string{rec3ID, rec4ID})
			So(res, ShouldHaveLength, 2)
			So(res[0].Err, ShouldBeNil)
			So(res[1].Err, ShouldBeNil)
			So(res[0].Entity.RepairState, ShouldEqual, record3Update.GetRepairState().String())
			So(res[1].Entity.RepairState, ShouldEqual, record4.GetRepairState().String())
		})
		Convey("Update record without ID to datastore", func() {
			rec5ID, _ := generateRepairRecordID(record5.Hostname, record5.AssetTag, record5.CreatedTime.String())
			reqUpdate := map[string]*inv.DeviceManualRepairRecord{rec5ID: record5}
			res, err := UpdateDeviceManualRepairRecords(ctx, reqUpdate)
			So(err, ShouldBeNil)
			So(res, ShouldHaveLength, 1)

			// Error should occur when trying to get old entity from datastore
			So(res[0].Err, ShouldNotBeNil)
			So(res[0].Err.Error(), ShouldContainSubstring, "datastore: no such entity")

			res = GetDeviceManualRepairRecords(ctx, []string{rec5ID})
			So(res, ShouldHaveLength, 1)
			So(res[0].Err, ShouldNotBeNil)
			So(res[0].Err.Error(), ShouldContainSubstring, "datastore: no such entity")
		})
	})
}
