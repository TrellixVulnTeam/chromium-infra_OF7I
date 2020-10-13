// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"encoding/csv"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/datastore"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
	"go.chromium.org/luci/common/bq"
	"go.chromium.org/luci/common/logging"

	apibq "infra/appengine/cros/lab_inventory/api/bigquery"
	ds "infra/libs/cros/lab_inventory/datastore"
	invprotos "infra/libs/cros/lab_inventory/protos"
)

var project string = "cros-lab-inventory-dev"

func main() {
	// Open the file
	csvfile, err := os.Open("plx_manual_repair_20201005113210.csv")
	if err != nil {
		log.Fatalln("Couldn't open the csv file", err)
	}

	ctx := context.Background()
	dsClient, err := datastore.NewClient(ctx, project)

	// Parse the file
	r := csv.NewReader(csvfile)
	var records []*invprotos.DeviceManualRepairRecord

	// Skip headers
	record, err := r.Read()

	// Iterate through the records
	for {
		// Read each record from csv
		record, err = r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		records = append(records, createRecordFromCsvRow(ctx, dsClient, record))
	}

	err = dumpManualRepairRecordsToBQ(records)
	if err != nil {
		log.Fatal(err)
	}
}

// createRecordFromCsvRow takes a csv row with the listed columns and parses it
// into a DeviceManualRepairRecord. All fields are accounted for unless
// specified otherwise.
//
// Notes:
// - Free text fields are left blank if blank.
// - Asset tags that are blank will be imputed with "null" string.
// - Any record that has a 'completed' date should be marked as fixed.
// - Any record that is marked as fixed but null 'completed' date gets marked
// as completed on 10/9/2020 6pm UTC.
//
// Original PLX Table exported CSV
// Column number - Column name
// 0 - date (omitted)
// 1 - hostname
// 2 - repair_type
// 3 - dut_asset_tag (lots of empty records)
// 4 - servo_v3 (omitted)
// 5 - servo_type_a (omitted)
// 6 - servo_type_c (omitted)
// 7 - labstation_type (omitted)
// 8 - model (omitted)
// 9 - phase (omitted)
// 10 - buganizer_bug
// 11 - diagnosis
// 12 - fix_procedure
// 13 - read_log
// 14 - fix_labstation -> Labstation->power-cycle
// 15 - fix_reset_servo -> Servo board->power-cycle
// 16 - fix_yoshi_cable_servo_micro -> Yoshi Cable->Replaced Yoshi Cable
// 17 - visual_inspection (omitted)
// 18 - check_fix_power_for_dut -> Charger->Other
// 19 - troubleshoot_dut -> DUT->Other
// 20 - reimage_reflash_dut -> DUT->Reimage
// 21 - host_location_updated
// 22 - fixed
// 23 - total_repair_time
// 24 - dut_started_repairing (omitted; use timestamp)
// 25 - dut_finished_repairing (omitted; use timestamp)
// 26 - dut_finished_repairing_timestamp
// 27 - dut_started_repairing_timestamp
// 28 - claimed_by
func createRecordFromCsvRow(ctx context.Context, dsClient *datastore.Client, row []string) *invprotos.DeviceManualRepairRecord {
	record := &invprotos.DeviceManualRepairRecord{}

	hostname := row[1]
	log.Println("Creating record for host " + hostname)

	record.Hostname = hostname
	record.RepairTargetType = mapRepairType(row[2])

	if len(row[3]) > 0 && row[3] != "null" {
		record.AssetTag = row[3]
	} else {
		record.AssetTag = getCurrentAssetTag(ctx, dsClient, hostname)
	}

	record.BuganizerBugUrl = processStandardStringField(row[10])
	record.Diagnosis = processStandardStringField(row[11])
	record.RepairProcedure = processStandardStringField(row[12])

	// Repair actions. See mapping in comments above.
	record.LabstationRepairActions = getLabstationRepairActions(row)
	record.ServoRepairActions = getServoRepairActions(row)
	record.YoshiRepairActions = getYoshiRepairActions(row)

	var additionalComments string
	record.ChargerRepairActions, additionalComments = getChargerRepairActions(row, additionalComments)
	record.DutRepairActions, additionalComments = getDutRepairActions(row, additionalComments)
	record.AdditionalComments = additionalComments

	// If record fixed field is not boolean, mark as unfixed by default.
	issueFixed, err := strconv.ParseBool(row[22])
	if err != nil {
		issueFixed = false
	}

	// Include LDAP only when field is not null. Otherwise, blank.
	var userLdap string
	if len(row[28]) > 0 && row[28] != "null" {
		userLdap = row[28] + "@google.com"
	}
	record.UserLdap = userLdap

	// Set default time taken to 0 if not parseable.
	timeTaken64, err := strconv.ParseInt(row[23], 10, 64)
	if err != nil {
		timeTaken64 = 0
	}
	record.TimeTaken = int32(timeTaken64)

	var createdTimestamp int64
	var createdSeconds int64 = 0
	createdTimestamp, err = strconv.ParseInt(row[27], 10, 64)
	if err == nil {
		createdSeconds = createdTimestamp / 1000000
		record.CreatedTime = &timestamp.Timestamp{Seconds: createdSeconds, Nanos: 0}
		record.UpdatedTime = &timestamp.Timestamp{Seconds: createdSeconds, Nanos: 0}
	} else {
		log.Println("Record has no dut_started_repairing_timestamp")
	}

	// Set CompletedTime based on parsed data. Set CompletedTime to 10/9/2020 6pm
	// UTC if issue fixed but not completed time available from data.
	var completedTimestamp int64
	var completedSeconds int64 = 0
	completedTimestamp, err = strconv.ParseInt(row[26], 10, 64)
	if err == nil {
		completedSeconds = completedTimestamp / 1000000
		record.CompletedTime = &timestamp.Timestamp{Seconds: completedSeconds, Nanos: 0}
		record.UpdatedTime = &timestamp.Timestamp{Seconds: completedSeconds, Nanos: 0}
		issueFixed = true
	} else if issueFixed {
		completedTimestamp = 1602266400
		completedSeconds = completedTimestamp / 1000000
		record.CompletedTime = &timestamp.Timestamp{Seconds: completedSeconds, Nanos: 0}
		record.UpdatedTime = &timestamp.Timestamp{Seconds: completedSeconds, Nanos: 0}
	}

	record.IssueFixed = issueFixed
	if issueFixed {
		record.RepairState = invprotos.DeviceManualRepairRecord_STATE_COMPLETED
	} else {
		record.RepairState = invprotos.DeviceManualRepairRecord_STATE_IN_PROGRESS
	}

	return record
}

func dumpManualRepairRecordsToBQ(records []*invprotos.DeviceManualRepairRecord) error {
	ctx := context.Background()
	logging.Infof(ctx, "Start to dump manual repair records to bigquery")
	dataset := "inventory"
	table := "manual_repair_records"

	client, err := bigquery.NewClient(ctx, project)
	if err != nil {
		return err
	}
	up := bq.NewUploader(ctx, client, dataset, table)
	up.SkipInvalidRows = true
	up.IgnoreUnknownValues = true

	logging.Debugf(ctx, "Preparing manual repair records for BigQuery")
	msgs := make([]proto.Message, len(records))
	for i, r := range records {
		id, err := ds.GenerateRepairRecordID(r.Hostname, r.AssetTag, ptypes.TimestampString(r.CreatedTime))
		if err != nil {
			panic(err)
		}

		msgs[i] = &apibq.DeviceManualRepairRecordRow{
			RepairRecordId: id,
			RepairRecord:   r,
		}
	}

	logging.Debugf(ctx, "Uploading %d PLX records of manual repair records", len(msgs))
	if err := up.Put(ctx, msgs...); err != nil {
		return err
	}

	return nil
}

// mapRepairType maps the repair type string from the PLX table to the proto enum.
func mapRepairType(rt string) invprotos.DeviceManualRepairRecord_RepairTargetType {
	lowerRt := strings.ToLower(rt)

	switch lowerRt {
	case "labstation":
		return invprotos.DeviceManualRepairRecord_TYPE_LABSTATION
	case "servo":
		return invprotos.DeviceManualRepairRecord_TYPE_SERVO
	default:
		return invprotos.DeviceManualRepairRecord_TYPE_DUT
	}
}

func getCurrentAssetTag(ctx context.Context, dsClient *datastore.Client, hostname string) string {
	q := datastore.NewQuery(ds.DeviceKind).Filter("Hostname =", hostname)

	var entities []*ds.DeviceEntity
	keys, err := dsClient.GetAll(ctx, q, &entities)
	if err != nil {
		log.Println("Failed to query from datastore: ", err)
		return "null"
	}

	if len(keys) == 0 {
		return "null"
	}

	r := keys[0]
	return r.Name
}

func processStandardStringField(fieldValue string) string {
	var field string
	if len(fieldValue) > 0 && fieldValue != "null" {
		field = fieldValue
	}
	return field
}

func getLabstationRepairActions(row []string) []invprotos.LabstationRepairAction {
	var labstationRepairActions []invprotos.LabstationRepairAction
	action, err := strconv.ParseBool(row[14])
	if err == nil && action {
		labstationRepairActions = append(labstationRepairActions, invprotos.LabstationRepairAction_LABSTATION_POWER_CYCLE)
	}
	return labstationRepairActions
}

func getServoRepairActions(row []string) []invprotos.ServoRepairAction {
	var servoRepairActions []invprotos.ServoRepairAction
	action, err := strconv.ParseBool(row[15])
	if err == nil && action {
		servoRepairActions = append(servoRepairActions, invprotos.ServoRepairAction_SERVO_POWER_CYCLE)
	}
	return servoRepairActions
}

func getYoshiRepairActions(row []string) []invprotos.YoshiRepairAction {
	var yoshiRepairActions []invprotos.YoshiRepairAction
	action, err := strconv.ParseBool(row[16])
	if err == nil && action {
		yoshiRepairActions = append(yoshiRepairActions, invprotos.YoshiRepairAction_YOSHI_REPLACE)
	}
	return yoshiRepairActions
}

func getChargerRepairActions(row []string, additionalComments string) ([]invprotos.ChargerRepairAction, string) {
	var chargerRepairActions []invprotos.ChargerRepairAction
	action, err := strconv.ParseBool(row[18])
	if err == nil && action {
		chargerRepairActions = append(chargerRepairActions, invprotos.ChargerRepairAction_CHARGER_OTHER)
		additionalComments += "check_fix_power_for_dut\n"
	}
	return chargerRepairActions, additionalComments
}

func getDutRepairActions(row []string, additionalComments string) ([]invprotos.DutRepairAction, string) {
	var dutRepairActions []invprotos.DutRepairAction
	visualInspection, err := strconv.ParseBool(row[17])
	troubleshootDut, err2 := strconv.ParseBool(row[19])
	if (err == nil && visualInspection) || (err2 == nil && troubleshootDut) {
		dutRepairActions = append(dutRepairActions, invprotos.DutRepairAction_DUT_OTHER)
	}

	if err == nil && visualInspection {
		additionalComments += "visual_inspection\n"
	}

	if err2 == nil && troubleshootDut {
		additionalComments += "troubleshoot_dut\n"
	}

	reimageDut, err := strconv.ParseBool(row[20])
	if err == nil && reimageDut {
		dutRepairActions = append(dutRepairActions, invprotos.DutRepairAction_DUT_REIMAGE_PROD)
	}
	return dutRepairActions, additionalComments
}
