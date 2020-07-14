package datastore

import (
	"testing"

	"github.com/golang/protobuf/ptypes"
	timestamp "github.com/golang/protobuf/ptypes/timestamp"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/appengine/gaetesting"

	inv "infra/appengine/cros/lab_inventory/api/v1"
)

func mockDeviceManualRepairRecord(hostname string, assetTag string, createdTime int64) *inv.DeviceManualRepairRecord {
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
		IssueFixed:    true,
		UserLdap:      "testing-account",
		TimeTaken:     15,
		CreatedTime:   &timestamp.Timestamp{Seconds: createdTime, Nanos: 0},
		UpdatedTime:   &timestamp.Timestamp{Seconds: 222, Nanos: 0},
		CompletedTime: &timestamp.Timestamp{Seconds: 222, Nanos: 0},
	}
}

func mockUpdatedRecord(hostname string, assetTag string, createdTime int64) *inv.DeviceManualRepairRecord {
	return &inv.DeviceManualRepairRecord{
		Hostname:                        hostname,
		AssetTag:                        assetTag,
		RepairTargetType:                inv.DeviceManualRepairRecord_TYPE_DUT,
		RepairState:                     inv.DeviceManualRepairRecord_STATE_COMPLETED,
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
		IssueFixed:    true,
		UserLdap:      "testing-account",
		TimeTaken:     30,
		CreatedTime:   &timestamp.Timestamp{Seconds: createdTime, Nanos: 0},
		UpdatedTime:   &timestamp.Timestamp{Seconds: 222, Nanos: 0},
		CompletedTime: &timestamp.Timestamp{Seconds: 222, Nanos: 0},
	}
}

func TestAddRecord(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	record1 := mockDeviceManualRepairRecord("chromeos-addRec-aa", "addRec-111", 1)
	record2 := mockDeviceManualRepairRecord("chromeos-addRec-bb", "addRec-222", 1)
	record3 := mockDeviceManualRepairRecord("chromeos-addRec-cc", "addRec-333", 1)
	record4 := mockDeviceManualRepairRecord("chromeos-addRec-dd", "addRec-444", 1)
	record5 := mockDeviceManualRepairRecord("", "", 1)

	rec1ID, _ := generateRepairRecordID(record1.Hostname, record1.AssetTag, ptypes.TimestampString(record1.CreatedTime))
	rec2ID, _ := generateRepairRecordID(record2.Hostname, record2.AssetTag, ptypes.TimestampString(record2.CreatedTime))
	ids1 := []string{rec1ID, rec2ID}

	rec3ID, _ := generateRepairRecordID(record3.Hostname, record3.AssetTag, ptypes.TimestampString(record3.CreatedTime))
	rec4ID, _ := generateRepairRecordID(record4.Hostname, record4.AssetTag, ptypes.TimestampString(record4.CreatedTime))
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
	record1 := mockDeviceManualRepairRecord("chromeos-getRec-aa", "getRec-111", 1)
	record2 := mockDeviceManualRepairRecord("chromeos-getRec-bb", "getRec-222", 1)
	rec1ID, _ := generateRepairRecordID(record1.Hostname, record1.AssetTag, ptypes.TimestampString(record1.CreatedTime))
	rec2ID, _ := generateRepairRecordID(record2.Hostname, record2.AssetTag, ptypes.TimestampString(record2.CreatedTime))
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

func TestGetRecordByPropertyName(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	datastore.GetTestable(ctx).Consistent(true)

	record1 := mockDeviceManualRepairRecord("chromeos-getByProp-aa", "getByProp-111", 1)
	record2 := mockUpdatedRecord("chromeos-getByProp-aa", "getByProp-111", 2)
	record3 := mockDeviceManualRepairRecord("chromeos-getByProp-aa", "getByProp-222", 1)
	records := []*inv.DeviceManualRepairRecord{record1, record2, record3}

	// Set up records in datastore and test
	AddDeviceManualRepairRecords(ctx, records)

	Convey("Get device manual repair record from datastore by property name", t, func() {
		Convey("Get repair record by Hostname", func() {
			// Query should return record1, record2, record3
			res, err := GetRepairRecordByPropertyName(ctx, "hostname", "chromeos-getByProp-aa")
			So(res, ShouldHaveLength, 3)
			So(err, ShouldBeNil)
			for _, r := range res {
				So(r.Err, ShouldBeNil)
				So(r.Entity.Hostname, ShouldEqual, "chromeos-getByProp-aa")
				So(r.Record.GetHostname(), ShouldEqual, "chromeos-getByProp-aa")
				So([]string{"STATE_NOT_STARTED", "STATE_COMPLETED"}, ShouldContain, r.Entity.RepairState)
				So([]string{"STATE_NOT_STARTED", "STATE_COMPLETED"}, ShouldContain, r.Record.GetRepairState().String())
				So([]string{"getByProp-111", "getByProp-222"}, ShouldContain, r.Entity.AssetTag)
				So([]string{"getByProp-111", "getByProp-222"}, ShouldContain, r.Record.GetAssetTag())
			}
		})
		Convey("Get repair record by AssetTag", func() {
			// Query should return record1, record2
			res, err := GetRepairRecordByPropertyName(ctx, "asset_tag", "getByProp-111")
			So(res, ShouldHaveLength, 2)
			So(err, ShouldBeNil)
			for _, r := range res {
				So(r.Err, ShouldBeNil)
				So(r.Entity.Hostname, ShouldEqual, "chromeos-getByProp-aa")
				So(r.Record.GetHostname(), ShouldEqual, "chromeos-getByProp-aa")
				So(r.Entity.AssetTag, ShouldEqual, "getByProp-111")
				So(r.Record.GetAssetTag(), ShouldEqual, "getByProp-111")
				So([]string{"STATE_NOT_STARTED", "STATE_COMPLETED"}, ShouldContain, r.Entity.RepairState)
				So([]string{"STATE_NOT_STARTED", "STATE_COMPLETED"}, ShouldContain, r.Record.GetRepairState().String())
			}
		})
		Convey("Get repair record by RepairState", func() {
			// Query should return record2
			res, err := GetRepairRecordByPropertyName(ctx, "repair_state", "STATE_COMPLETED")
			So(res, ShouldHaveLength, 1)
			So(err, ShouldBeNil)
			So(res[0].Err, ShouldBeNil)
			So(res[0].Entity.Hostname, ShouldEqual, "chromeos-getByProp-aa")
			So(res[0].Record.GetHostname(), ShouldEqual, "chromeos-getByProp-aa")
			So(res[0].Entity.AssetTag, ShouldEqual, "getByProp-111")
			So(res[0].Record.GetAssetTag(), ShouldEqual, "getByProp-111")
			So(res[0].Entity.RepairState, ShouldEqual, "STATE_COMPLETED")
			So(res[0].Record.GetRepairState(), ShouldEqual, inv.DeviceManualRepairRecord_STATE_COMPLETED)
		})
	})
}

func TestUpdateRecord(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	record1 := mockDeviceManualRepairRecord("chromeos-updateRec-aa", "updateRec-111", 1)
	record1Update := mockUpdatedRecord("chromeos-updateRec-aa", "updateRec-111", 1)

	record2 := mockDeviceManualRepairRecord("chromeos-updateRec-bb", "updateRec-222", 1)

	record3 := mockDeviceManualRepairRecord("chromeos-updateRec-cc", "updateRec-333", 1)
	record3Update := mockUpdatedRecord("chromeos-updateRec-cc", "updateRec-333", 1)
	record4 := mockDeviceManualRepairRecord("chromeos-updateRec-dd", "updateRec-444", 1)

	record5 := mockDeviceManualRepairRecord("", "", 1)
	Convey("Update record in datastore", t, func() {
		Convey("Update existing record to datastore", func() {
			rec1ID, _ := generateRepairRecordID(record1.Hostname, record1.AssetTag, ptypes.TimestampString(record1.CreatedTime))
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
			rec2ID, _ := generateRepairRecordID(record2.Hostname, record2.AssetTag, ptypes.TimestampString(record2.CreatedTime))
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
			rec3ID, _ := generateRepairRecordID(record3.Hostname, record3.AssetTag, ptypes.TimestampString(record3.CreatedTime))
			rec4ID, _ := generateRepairRecordID(record4.Hostname, record4.AssetTag, ptypes.TimestampString(record4.CreatedTime))
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
			rec5ID, _ := generateRepairRecordID(record5.Hostname, record5.AssetTag, ptypes.TimestampString(record5.CreatedTime))
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
