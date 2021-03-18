// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"

	ufspb "infra/unifiedfleet/api/v1/models"
	chromeosLab "infra/unifiedfleet/api/v1/models/chromeos/lab"
	"infra/unifiedfleet/app/model/history"
	"infra/unifiedfleet/app/model/inventory"
	"infra/unifiedfleet/app/model/state"
	"infra/unifiedfleet/app/util"
)

func TestUpdateDutState(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	osCtx, _ := util.SetupDatastoreNamespace(ctx, util.OSNamespace)
	Convey("UpdateDutState", t, func() {
		Convey("Update dut state with non-existing host in dut state storage", func() {
			ds1 := mockDutState("update-dutstate-id1", "update-dutstate-hostname1")
			_, err := UpdateDutState(ctx, ds1)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Entity not found")
		})

		Convey("Update dut state - happy path with existing dut state", func() {
			ds1 := mockDutState("update-dutstate-id2", "update-dutstate-hostname2")
			ds1.Servo = chromeosLab.PeripheralState_WORKING
			ds1.Chameleon = chromeosLab.PeripheralState_WORKING
			ds1.StorageState = chromeosLab.HardwareState_HARDWARE_ACCEPTABLE

			// Use osCtx in setup
			_, err := inventory.CreateMachineLSE(osCtx, &ufspb.MachineLSE{
				Name:     "update-dutstate-hostname2",
				Hostname: "update-dutstate-hostname2",
				Lse: &ufspb.MachineLSE_ChromeBrowserMachineLse{
					ChromeBrowserMachineLse: &ufspb.ChromeBrowserMachineLSE{},
				},
			})
			So(err, ShouldBeNil)
			_, err = state.UpdateDutStates(osCtx, []*chromeosLab.DutState{ds1})
			So(err, ShouldBeNil)
			oldDS, err := state.GetDutState(osCtx, "update-dutstate-id2")
			So(err, ShouldBeNil)
			So(oldDS.GetServo(), ShouldEqual, chromeosLab.PeripheralState_WORKING)
			So(oldDS.GetChameleon(), ShouldEqual, chromeosLab.PeripheralState_WORKING)
			So(oldDS.GetStorageState(), ShouldEqual, chromeosLab.HardwareState_HARDWARE_ACCEPTABLE)

			// Use osCtx in testing, as in prod, ctx is forced to include namespace.
			ds2 := mockDutState("update-dutstate-id2", "update-dutstate-hostname2")
			ds2.Servo = chromeosLab.PeripheralState_BROKEN
			ds2.Chameleon = chromeosLab.PeripheralState_BROKEN
			ds2.StorageState = chromeosLab.HardwareState_HARDWARE_NEED_REPLACEMENT
			_, err = UpdateDutState(osCtx, ds2)
			So(err, ShouldBeNil)

			// Verify with osCtx
			newDS, err := state.GetDutState(osCtx, "update-dutstate-id2")
			So(err, ShouldBeNil)
			So(newDS.GetServo(), ShouldEqual, chromeosLab.PeripheralState_BROKEN)
			So(newDS.GetChameleon(), ShouldEqual, chromeosLab.PeripheralState_BROKEN)
			So(newDS.GetStorageState(), ShouldEqual, chromeosLab.HardwareState_HARDWARE_NEED_REPLACEMENT)
			// Verify changes
			changes, err := history.QueryChangesByPropertyName(osCtx, "name", "dutstates/update-dutstate-id2")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 3)
			So(changes[0].GetEventLabel(), ShouldEqual, "dut_state.servo")
			So(changes[0].GetOldValue(), ShouldEqual, chromeosLab.PeripheralState_WORKING.String())
			So(changes[0].GetNewValue(), ShouldEqual, chromeosLab.PeripheralState_BROKEN.String())
			So(changes[1].GetEventLabel(), ShouldEqual, "dut_state.chameleon")
			So(changes[1].GetOldValue(), ShouldEqual, chromeosLab.PeripheralState_WORKING.String())
			So(changes[1].GetNewValue(), ShouldEqual, chromeosLab.PeripheralState_BROKEN.String())
			So(changes[2].GetEventLabel(), ShouldEqual, "dut_state.storage_state")
			So(changes[2].GetOldValue(), ShouldEqual, chromeosLab.HardwareState_HARDWARE_ACCEPTABLE.String())
			So(changes[2].GetNewValue(), ShouldEqual, chromeosLab.HardwareState_HARDWARE_NEED_REPLACEMENT.String())
			msgs, err := history.QuerySnapshotMsgByPropertyName(osCtx, "resource_name", "dutstates/update-dutstate-id2")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			So(msgs[0].Delete, ShouldBeFalse)
		})
	})
}

func TestGetDutState(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	osCtx, _ := util.SetupDatastoreNamespace(ctx, util.OSNamespace)
	Convey("GetDutState", t, func() {
		Convey("Get dut state by id with non-existing host in dut state storage", func() {
			_, err := GetDutState(ctx, "id1", "")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Entity not found")
		})

		Convey("Get dut state by hostname with non-existing host in dut state storage", func() {
			_, err := GetDutState(ctx, "", "hostname1")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Dut State not found for hostname1.")
		})

		Convey("Get dut state by id - happy path with existing dut state", func() {
			ds1 := mockDutState("update-dutstate-id2", "update-dutstate-hostname2")
			ds1.Servo = chromeosLab.PeripheralState_WORKING
			ds1.Chameleon = chromeosLab.PeripheralState_WORKING
			ds1.StorageState = chromeosLab.HardwareState_HARDWARE_ACCEPTABLE

			_, err := state.UpdateDutStates(osCtx, []*chromeosLab.DutState{ds1})
			So(err, ShouldBeNil)

			oldDS, err := GetDutState(osCtx, "update-dutstate-id2", "")
			So(err, ShouldBeNil)
			So(oldDS.GetServo(), ShouldEqual, chromeosLab.PeripheralState_WORKING)
			So(oldDS.GetChameleon(), ShouldEqual, chromeosLab.PeripheralState_WORKING)
			So(oldDS.GetStorageState(), ShouldEqual, chromeosLab.HardwareState_HARDWARE_ACCEPTABLE)
		})

		Convey("Get dut state by hostname - happy path with existing dut state", func() {
			ds1 := mockDutState("update-dutstate-id3", "update-dutstate-hostname3")
			ds1.Servo = chromeosLab.PeripheralState_WORKING
			ds1.Chameleon = chromeosLab.PeripheralState_WORKING
			ds1.StorageState = chromeosLab.HardwareState_HARDWARE_ACCEPTABLE

			_, err := state.UpdateDutStates(osCtx, []*chromeosLab.DutState{ds1})
			So(err, ShouldBeNil)

			oldDS, err := GetDutState(osCtx, "", "update-dutstate-hostname3")
			So(err, ShouldBeNil)
			So(oldDS.GetServo(), ShouldEqual, chromeosLab.PeripheralState_WORKING)
			So(oldDS.GetChameleon(), ShouldEqual, chromeosLab.PeripheralState_WORKING)
			So(oldDS.GetStorageState(), ShouldEqual, chromeosLab.HardwareState_HARDWARE_ACCEPTABLE)
		})
	})
}

func TestListDutStates(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	dutStates := make([]*chromeosLab.DutState, 0, 4)
	for i := 0; i < 4; i++ {
		cs := mockDutState(fmt.Sprintf("cs-machine-%d", i), fmt.Sprintf("cs-dut-%d", i))
		dutStates = append(dutStates, cs)
	}
	dutStates, _ = state.UpdateDutStates(ctx, dutStates)
	Convey("ListDutStates", t, func() {
		Convey("ListDutStates - Full listing - happy path", func() {
			resp, _, _ := ListDutStates(ctx, 5, "", "", false)
			So(resp, ShouldNotBeNil)
			So(resp, ShouldResembleProto, dutStates)
		})
	})
}

func mockDutState(id, hostname string) *chromeosLab.DutState {
	return &chromeosLab.DutState{
		Id: &chromeosLab.ChromeOSDeviceID{
			Value: id,
		},
		Hostname: hostname,
	}
}
