package datastore

import (
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes"

	timestamp "github.com/golang/protobuf/ptypes/timestamp"
	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/luci/appengine/gaetesting"
	"go.chromium.org/luci/gae/service/datastore"

	invlibs "infra/cros/lab_inventory/protos"
)

func mockDeviceManualRepairRecord(hostname string, assetTag string, createdTime int64) *invlibs.DeviceManualRepairRecord {
	return &invlibs.DeviceManualRepairRecord{
		Hostname:                        hostname,
		AssetTag:                        assetTag,
		RepairTargetType:                invlibs.DeviceManualRepairRecord_TYPE_DUT,
		RepairState:                     invlibs.DeviceManualRepairRecord_STATE_NOT_STARTED,
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
		UpdatedTime:   &timestamp.Timestamp{Seconds: createdTime, Nanos: 0},
		CompletedTime: &timestamp.Timestamp{Seconds: 222, Nanos: 0},
	}
}

func mockUpdatedRecord(hostname string, assetTag string, createdTime int64) *invlibs.DeviceManualRepairRecord {
	return &invlibs.DeviceManualRepairRecord{
		Hostname:                        hostname,
		AssetTag:                        assetTag,
		RepairTargetType:                invlibs.DeviceManualRepairRecord_TYPE_DUT,
		RepairState:                     invlibs.DeviceManualRepairRecord_STATE_COMPLETED,
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
		TimeTaken:     30,
		CreatedTime:   &timestamp.Timestamp{Seconds: createdTime, Nanos: 0},
		UpdatedTime:   &timestamp.Timestamp{Seconds: 333, Nanos: 0},
		CompletedTime: &timestamp.Timestamp{Seconds: 333, Nanos: 0},
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

	rec1ID, _ := GenerateRepairRecordID(record1.Hostname, record1.AssetTag, ptypes.TimestampString(record1.CreatedTime))
	rec2ID, _ := GenerateRepairRecordID(record2.Hostname, record2.AssetTag, ptypes.TimestampString(record2.CreatedTime))
	ids1 := []string{rec1ID, rec2ID}

	rec3ID, _ := GenerateRepairRecordID(record3.Hostname, record3.AssetTag, ptypes.TimestampString(record3.CreatedTime))
	rec4ID, _ := GenerateRepairRecordID(record4.Hostname, record4.AssetTag, ptypes.TimestampString(record4.CreatedTime))
	ids2 := []string{rec3ID, rec4ID}
	Convey("Add device manual repair record to datastore", t, func() {
		Convey("Add multiple device manual repair records to datastore", func() {
			records := []*invlibs.DeviceManualRepairRecord{record1, record2}
			res, err := AddDeviceManualRepairRecords(ctx, records)
			So(err, ShouldBeNil)
			So(res, ShouldHaveLength, 2)

			// Put and Get order should be the same as the order in which the records
			// were passed in as arguments.
			for i, r := range res {
				So(r.Err, ShouldBeNil)
				So(r.Entity.Hostname, ShouldEqual, records[i].GetHostname())
				So(r.Entity.AssetTag, ShouldEqual, records[i].GetAssetTag())
				So(r.Entity.RepairState, ShouldEqual, "STATE_NOT_STARTED")

				updatedTime, _ := ptypes.Timestamp(records[i].GetUpdatedTime())
				So(r.Entity.UpdatedTime, ShouldEqual, updatedTime)
			}

			res = GetDeviceManualRepairRecords(ctx, ids1)
			So(res, ShouldHaveLength, 2)
			for i, r := range res {
				So(r.Err, ShouldBeNil)
				So(r.Entity.Hostname, ShouldEqual, records[i].GetHostname())
				So(r.Entity.AssetTag, ShouldEqual, records[i].GetAssetTag())
				So(r.Entity.RepairState, ShouldEqual, "STATE_NOT_STARTED")

				updatedTime, _ := ptypes.Timestamp(records[i].GetUpdatedTime())
				So(r.Entity.UpdatedTime, ShouldEqual, updatedTime)
			}
		})
		Convey("Add existing record to datastore", func() {
			req := []*invlibs.DeviceManualRepairRecord{record3}
			res, err := AddDeviceManualRepairRecords(ctx, req)
			So(err, ShouldBeNil)
			So(res, ShouldHaveLength, 1)
			So(res[0].Err, ShouldBeNil)

			// Verify adding existing record.
			req = []*invlibs.DeviceManualRepairRecord{record3, record4}
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
			for i, r := range res {
				So(r.Err, ShouldBeNil)
				So(r.Entity.Hostname, ShouldEqual, req[i].GetHostname())
				So(r.Entity.AssetTag, ShouldEqual, req[i].GetAssetTag())
				So(r.Entity.RepairState, ShouldEqual, "STATE_NOT_STARTED")

				updatedTime, _ := ptypes.Timestamp(req[i].GetUpdatedTime())
				So(r.Entity.UpdatedTime, ShouldResemble, updatedTime)
			}
		})
		Convey("Add record without hostname to datastore", func() {
			req := []*invlibs.DeviceManualRepairRecord{record5}
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
	rec1ID, _ := GenerateRepairRecordID(record1.Hostname, record1.AssetTag, ptypes.TimestampString(record1.CreatedTime))
	rec2ID, _ := GenerateRepairRecordID(record2.Hostname, record2.AssetTag, ptypes.TimestampString(record2.CreatedTime))
	ids1 := []string{rec1ID, rec2ID}
	Convey("Get device manual repair record from datastore", t, func() {
		Convey("Get non-existent device manual repair record from datastore", func() {
			records := []*invlibs.DeviceManualRepairRecord{record1}
			res, err := AddDeviceManualRepairRecords(ctx, records)
			So(err, ShouldBeNil)
			So(res, ShouldHaveLength, 1)

			res = GetDeviceManualRepairRecords(ctx, ids1)
			So(res, ShouldHaveLength, 2)
			So(res[0].Err, ShouldBeNil)
			So(res[1].Err, ShouldNotBeNil)
			So(res[1].Err.Error(), ShouldContainSubstring, "datastore: no such entity")

			updatedTime, _ := ptypes.Timestamp(record1.CreatedTime)
			So(res[0].Entity.UpdatedTime, ShouldResemble, updatedTime)
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
	records := []*invlibs.DeviceManualRepairRecord{record1, record2, record3}

	// Set up records in datastore and test
	AddDeviceManualRepairRecords(ctx, records)

	Convey("Get device manual repair record from datastore by property name", t, func() {
		Convey("Get repair record by Hostname", func() {
			// Query should return record1, record2, record3
			res, err := GetRepairRecordByPropertyName(ctx, map[string]string{"hostname": "chromeos-getByProp-aa"}, -1, 0, []string{})
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
		Convey("Get repair record by Hostname limit to 1", func() {
			// Query should return record1
			res, err := GetRepairRecordByPropertyName(ctx, map[string]string{"hostname": "chromeos-getByProp-aa"}, 1, 0, []string{})
			So(res, ShouldHaveLength, 1)
			So(err, ShouldBeNil)
			r := res[0]
			So(r.Err, ShouldBeNil)
			So(r.Entity.Hostname, ShouldEqual, "chromeos-getByProp-aa")
			So(r.Record.GetHostname(), ShouldEqual, "chromeos-getByProp-aa")
			So([]string{"STATE_NOT_STARTED", "STATE_COMPLETED"}, ShouldContain, r.Entity.RepairState)
			So([]string{"STATE_NOT_STARTED", "STATE_COMPLETED"}, ShouldContain, r.Record.GetRepairState().String())
			So([]string{"getByProp-111", "getByProp-222"}, ShouldContain, r.Entity.AssetTag)
			So([]string{"getByProp-111", "getByProp-222"}, ShouldContain, r.Record.GetAssetTag())
		})
		Convey("Get repair record by AssetTag", func() {
			// Query should return record1, record2
			res, err := GetRepairRecordByPropertyName(ctx, map[string]string{"asset_tag": "getByProp-111"}, -1, 0, []string{})
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
			res, err := GetRepairRecordByPropertyName(ctx, map[string]string{"repair_state": "STATE_COMPLETED"}, -1, 0, []string{})
			So(res, ShouldHaveLength, 1)
			So(err, ShouldBeNil)
			So(res[0].Err, ShouldBeNil)
			So(res[0].Entity.Hostname, ShouldEqual, "chromeos-getByProp-aa")
			So(res[0].Record.GetHostname(), ShouldEqual, "chromeos-getByProp-aa")
			So(res[0].Entity.AssetTag, ShouldEqual, "getByProp-111")
			So(res[0].Record.GetAssetTag(), ShouldEqual, "getByProp-111")
			So(res[0].Entity.RepairState, ShouldEqual, "STATE_COMPLETED")
			So(res[0].Record.GetRepairState(), ShouldEqual, invlibs.DeviceManualRepairRecord_STATE_COMPLETED)
		})
		Convey("Get repair record by multiple properties", func() {
			// Query should return record1 and record2
			res, err := GetRepairRecordByPropertyName(ctx,
				map[string]string{
					"hostname":  "chromeos-getByProp-aa",
					"asset_tag": "getByProp-111",
				}, -1, 0, []string{})
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
		Convey("Get repair record by multiple properties with offset", func() {
			// Query should return record2
			res, err := GetRepairRecordByPropertyName(ctx,
				map[string]string{
					"hostname":  "chromeos-getByProp-aa",
					"asset_tag": "getByProp-111",
				}, -1, 1, []string{})
			So(res, ShouldHaveLength, 1)
			So(err, ShouldBeNil)
			So(res[0].Err, ShouldBeNil)
			So(res[0].Entity.Hostname, ShouldEqual, "chromeos-getByProp-aa")
			So(res[0].Record.GetHostname(), ShouldEqual, "chromeos-getByProp-aa")
			So(res[0].Entity.AssetTag, ShouldEqual, "getByProp-111")
			So(res[0].Record.GetAssetTag(), ShouldEqual, "getByProp-111")
			So(res[0].Entity.RepairState, ShouldEqual, "STATE_COMPLETED")
			So(res[0].Record.GetRepairState().String(), ShouldEqual, "STATE_COMPLETED")
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
			rec1ID, _ := GenerateRepairRecordID(record1.Hostname, record1.AssetTag, ptypes.TimestampString(record1.CreatedTime))
			req := []*invlibs.DeviceManualRepairRecord{record1}
			res, err := AddDeviceManualRepairRecords(ctx, req)
			So(err, ShouldBeNil)
			So(res, ShouldNotBeNil)
			So(res, ShouldHaveLength, 1)
			So(res[0].Err, ShouldBeNil)

			res = GetDeviceManualRepairRecords(ctx, []string{rec1ID})
			So(res, ShouldHaveLength, 1)
			So(res[0].Err, ShouldBeNil)
			So(res[0].Entity.RepairState, ShouldEqual, "STATE_NOT_STARTED")

			updatedTime1, _ := ptypes.Timestamp(record1.CreatedTime)
			So(res[0].Entity.UpdatedTime, ShouldResemble, updatedTime1)

			// Update and check
			reqUpdate := map[string]*invlibs.DeviceManualRepairRecord{rec1ID: record1Update}
			res, err = UpdateDeviceManualRepairRecords(ctx, reqUpdate)
			So(err, ShouldBeNil)
			So(res, ShouldHaveLength, 1)
			So(res[0].Err, ShouldBeNil)

			res = GetDeviceManualRepairRecords(ctx, []string{rec1ID})
			So(res, ShouldHaveLength, 1)
			So(res[0].Err, ShouldBeNil)
			So(res[0].Entity.RepairState, ShouldEqual, "STATE_COMPLETED")

			updatedTime1, _ = ptypes.Timestamp(&timestamp.Timestamp{Seconds: 333, Nanos: 0})
			So(res[0].Entity.UpdatedTime, ShouldResemble, updatedTime1)
		})
		Convey("Update non-existent record in datastore", func() {
			rec2ID, _ := GenerateRepairRecordID(record2.Hostname, record2.AssetTag, ptypes.TimestampString(record2.CreatedTime))
			reqUpdate := map[string]*invlibs.DeviceManualRepairRecord{rec2ID: record2}
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
			rec3ID, _ := GenerateRepairRecordID(record3.Hostname, record3.AssetTag, ptypes.TimestampString(record3.CreatedTime))
			rec4ID, _ := GenerateRepairRecordID(record4.Hostname, record4.AssetTag, ptypes.TimestampString(record4.CreatedTime))
			req := []*invlibs.DeviceManualRepairRecord{record3, record4}
			res, err := AddDeviceManualRepairRecords(ctx, req)
			So(err, ShouldBeNil)
			So(res, ShouldNotBeNil)
			So(res, ShouldHaveLength, 2)
			So(res[0].Err, ShouldBeNil)
			So(res[1].Err, ShouldBeNil)

			reqUpdate := map[string]*invlibs.DeviceManualRepairRecord{
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
			So(res[0].Entity.RepairState, ShouldEqual, "STATE_COMPLETED")
			So(res[1].Entity.RepairState, ShouldEqual, "STATE_NOT_STARTED")

			updatedTime3, _ := ptypes.Timestamp(&timestamp.Timestamp{Seconds: 333, Nanos: 0})
			So(res[0].Entity.UpdatedTime, ShouldResemble, updatedTime3)

			updatedTime4, _ := ptypes.Timestamp(record4.CreatedTime)
			So(res[1].Entity.UpdatedTime, ShouldResemble, updatedTime4)
		})
		Convey("Update record without ID to datastore", func() {
			rec5ID, _ := GenerateRepairRecordID(record5.Hostname, record5.AssetTag, ptypes.TimestampString(record5.CreatedTime))
			reqUpdate := map[string]*invlibs.DeviceManualRepairRecord{rec5ID: record5}
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

func TestManualRepairIndexes(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	datastore.GetTestable(ctx).Consistent(true)

	record1 := mockUpdatedRecord("chromeos-indexTest-aa", "indexTest-111", 1)
	record2 := mockUpdatedRecord("chromeos-indexTest-bb", "indexTest-222", 1)
	record3 := mockDeviceManualRepairRecord("chromeos-indexTest-cc", "indexTest-222", 1)
	record4 := mockUpdatedRecord("chromeos-indexTest-dd", "indexTest-444", 1)

	records := []*invlibs.DeviceManualRepairRecord{record1, record2, record3, record4}
	_, _ = AddDeviceManualRepairRecords(ctx, records)

	Convey("Query device manual repair record from datastore using indexes", t, func() {
		Convey("Query by repair_state", func() {
			q := datastore.NewQuery(DeviceManualRepairRecordEntityKind).
				Eq("repair_state", invlibs.DeviceManualRepairRecord_STATE_COMPLETED.String())

			var entities []*DeviceManualRepairRecordEntity
			err := datastore.GetAll(ctx, q, &entities)

			So(err, ShouldBeNil)
			So(entities, ShouldHaveLength, 3)
		})
		Convey("Query by updated_time", func() {
			rec4Update := mockUpdatedRecord("chromeos-indexTest-dd", "indexTest-444", 1)
			rec4Update.UpdatedTime, _ = ptypes.TimestampProto(time.Unix(555, 0).UTC())

			rec4ID, _ := GenerateRepairRecordID(rec4Update.Hostname, rec4Update.AssetTag, ptypes.TimestampString(rec4Update.CreatedTime))
			reqUpdate := map[string]*invlibs.DeviceManualRepairRecord{rec4ID: rec4Update}
			_, err := UpdateDeviceManualRepairRecords(ctx, reqUpdate)
			So(err, ShouldBeNil)

			q := datastore.NewQuery(DeviceManualRepairRecordEntityKind).
				Gte("updated_time", time.Unix(500, 0).UTC())

			var entities []*DeviceManualRepairRecordEntity
			err = datastore.GetAll(ctx, q, &entities)
			So(err, ShouldBeNil)
			So(entities, ShouldHaveLength, 1)
			So(entities[0].ID, ShouldEqual, rec4ID)
		})
	})
}
