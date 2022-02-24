// Copyright 2022 The LUCI Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package inventory

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"go.chromium.org/luci/gae/service/datastore"
	"google.golang.org/protobuf/testing/protocmp"

	fleet "infra/appengine/crosskylabadmin/api/fleet/v1"
	dsinventory "infra/appengine/crosskylabadmin/internal/app/frontend/internal/datastore/inventory"
	dssv "infra/appengine/crosskylabadmin/internal/app/frontend/internal/datastore/stableversion"
	"infra/appengine/crosskylabadmin/internal/app/frontend/internal/datastore/stableversion/satlab"
	"infra/libs/skylab/inventory"
)

// TestGetStableVersionRPCForSatlabDeviceUsingBoardAndModel tests the happy path of extracting a satlab stable version using its board and model.
func TestGetStableVersionRPCForSatlabDeviceUsingBoardAndModel(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	datastore.GetTestable(ctx).Consistent(true)
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()

	board := "b"
	model := "m"

	modelBoardID := satlab.MakeSatlabStableVersionID("", board, model)

	_, err := satlab.GetSatlabStableVersionEntryByRawID(ctx, modelBoardID)
	if !datastore.IsErrNoSuchEntity(err) {
		t.Errorf("error should be no such entity: %s", err)
	}

	satlab.PutSatlabStableVersionEntry(ctx, &satlab.SatlabStableVersionEntry{
		ID:      modelBoardID,
		OS:      "o",
		FW:      "fw",
		FWImage: "i",
	})

	expected := &satlab.SatlabStableVersionEntry{
		ID:      modelBoardID,
		OS:      "o",
		FW:      "fw",
		FWImage: "i",
	}
	actual, err := satlab.GetSatlabStableVersionEntryByRawID(ctx, modelBoardID)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if diff := cmp.Diff(expected, actual, cmpopts.IgnoreUnexported(satlab.SatlabStableVersionEntry{})); diff != "" {
		t.Errorf("unexpected diff (-want +got): %s", err)
	}

	resp, err := tf.Inventory.GetStableVersion(ctx, &fleet.GetStableVersionRequest{
		BuildTarget:              "b",
		Model:                    "m",
		SatlabInformationalQuery: true,
	})
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	expectedResponse := &fleet.GetStableVersionResponse{
		CrosVersion:     "o",
		FirmwareVersion: "fw",
		FaftVersion:     "i",
		Reason:          `looked up satlab device using id "b|m"`,
	}
	actualResponse := resp
	if diff := cmp.Diff(expectedResponse, actualResponse, protocmp.Transform()); diff != "" {
		t.Errorf("unexpected diff (-want +got): %s", diff)
	}
}

// TestGetStableVersionRPCForSatlabDeviceUsingHostname tests the happy path of extracting a satlab stable version using its hostname.
func TestGetStableVersionRPCForSatlabDeviceUsingHostname(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	datastore.GetTestable(ctx).Consistent(true)
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()

	hostname := "satlab-h1"

	id := satlab.MakeSatlabStableVersionID(hostname, "", "")

	_, err := satlab.GetSatlabStableVersionEntryByRawID(ctx, id)
	if !datastore.IsErrNoSuchEntity(err) {
		t.Errorf("error should be no such entity: %s", err)
	}

	satlab.PutSatlabStableVersionEntry(ctx, &satlab.SatlabStableVersionEntry{
		ID:      id,
		OS:      "o",
		FW:      "fw",
		FWImage: "i",
	})

	expected := &satlab.SatlabStableVersionEntry{
		ID:      id,
		OS:      "o",
		FW:      "fw",
		FWImage: "i",
	}
	actual, err := satlab.GetSatlabStableVersionEntryByRawID(ctx, id)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if diff := cmp.Diff(expected, actual, cmpopts.IgnoreUnexported(satlab.SatlabStableVersionEntry{})); diff != "" {
		t.Errorf("unexpected diff (-want +got): %s", err)
	}

	resp, err := tf.Inventory.GetStableVersion(ctx, &fleet.GetStableVersionRequest{
		Hostname: hostname,
	})
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	expectedResponse := &fleet.GetStableVersionResponse{
		CrosVersion:     "o",
		FirmwareVersion: "fw",
		FaftVersion:     "i",
		Reason:          `looked up satlab device using id "satlab-h1"`,
	}
	actualResponse := resp
	if diff := cmp.Diff(expectedResponse, actualResponse, protocmp.Transform()); diff != "" {
		t.Errorf("unexpected diff (-want +got): %s", diff)
	}
}

// TestGetStableVersionRPCForSatlabDeviceUsingBoardAndModelFallback tests the fallback  path of extracting a satlab stable version using its board and model.
func TestGetStableVersionRPCForSatlabDeviceUsingBoardAndModelFallback(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	datastore.GetTestable(ctx).Consistent(true)
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()

	duts := []*inventory.DeviceUnderTest{
		{
			Common: &inventory.CommonDeviceSpecs{
				Attributes: []*inventory.KeyValue{},
				Id:         strptr("id"),
				Hostname:   strptr("h"),
				Labels: &inventory.SchedulableLabels{
					Model: strptr("m"),
					Board: strptr("b"),
				},
			},
		},
	}

	if err := dsinventory.UpdateDUTs(ctx, duts); err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	if err := dssv.PutSingleCrosStableVersion(ctx, "b", "m", "c"); err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if err := dssv.PutSingleFirmwareStableVersion(ctx, "b", "m", "f"); err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if err := dssv.PutSingleFaftStableVersion(ctx, "b", "m", "i"); err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	resp, err := tf.Inventory.GetStableVersion(ctx, &fleet.GetStableVersionRequest{
		BuildTarget:              "b",
		Model:                    "m",
		SatlabInformationalQuery: true,
	})
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	expectedResponse := &fleet.GetStableVersionResponse{
		CrosVersion:     "c",
		FirmwareVersion: "f",
		FaftVersion:     "i",
		Reason:          `wanted satlab, falling back to board "b" and model "m"`,
	}
	actualResponse := resp
	if diff := cmp.Diff(expectedResponse, actualResponse, protocmp.Transform()); diff != "" {
		t.Errorf("unexpected diff (-want +got): %s", diff)
	}
}
