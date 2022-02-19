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

package satlab

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"go.chromium.org/luci/appengine/gaetesting"
	"go.chromium.org/luci/gae/service/datastore"

	fleet "infra/appengine/crosskylabadmin/api/fleet/v1"
)

// TestMakeSatlabStableVersionEntry tests that we correctly convert form
// an incoming request to a datastore entry.
func TestMakeSatlabStableVersionEntry(t *testing.T) {
	t.Parallel()

	data := []struct {
		name string
		in   *fleet.SetSatlabStableVersionRequest
		out  *SatlabStableVersionEntry
	}{
		{
			name: "empty",
			in:   nil,
			out:  nil,
		},
		{
			name: "with hostname",
			in: &fleet.SetSatlabStableVersionRequest{
				Strategy: &fleet.SetSatlabStableVersionRequest_SatlabHostnameStrategy{
					SatlabHostnameStrategy: &fleet.SatlabHostnameStrategy{
						Hostname: "aaa-hostname-aaa",
					},
				},
				CrosVersion:     "AAA-CROS-AAA",
				FirmwareVersion: "AAA-FW-AAA",
				FirmwareImage:   "AAA-FWImage-AAA",
			},
			out: &SatlabStableVersionEntry{
				ID:      "aaa-hostname-aaa",
				OS:      "AAA-CROS-AAA",
				FW:      "AAA-FW-AAA",
				FWImage: "AAA-FWImage-AAA",
			},
		},
		{
			name: "with model",
			in: &fleet.SetSatlabStableVersionRequest{
				Strategy: &fleet.SetSatlabStableVersionRequest_SatlabBoardAndModelStrategy{
					SatlabBoardAndModelStrategy: &fleet.SatlabBoardAndModelStrategy{
						Board: "aaa-board-aaa",
						Model: "aaa-model-aaa",
					},
				},
				CrosVersion:     "AAA-CROS-AAA",
				FirmwareVersion: "AAA-FW-AAA",
				FirmwareImage:   "AAA-FWImage-AAA",
			},
			out: &SatlabStableVersionEntry{
				ID:      "aaa-board-aaa|aaa-model-aaa",
				OS:      "AAA-CROS-AAA",
				FW:      "AAA-FW-AAA",
				FWImage: "AAA-FWImage-AAA",
			},
		},
	}

	for _, tt := range data {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := gaetesting.TestingContext()
			datastore.GetTestable(ctx).Consistent(true)

			expected := tt.out
			actual, _ := MakeSatlabStableVersionEntry(tt.in, false)
			// Ignore the base64-encoded original request field.
			// It's probably right.
			if actual != nil {
				actual.Base64Req = ""
			}

			if diff := cmp.Diff(expected, actual, cmpopts.IgnoreUnexported(SatlabStableVersionEntry{})); diff != "" {
				t.Errorf("unexpected diff: %s", diff)
			}
		})
	}
}

// TestGetSatlabStableVersionEntryByID tests putting a record into datastore and retrieving it by its ID.
func TestGetSatlabStableVersionEntryByID(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   *SatlabStableVersionEntry
		req  *fleet.GetStableVersionRequest
	}{
		{
			name: "hostname",
			in: &SatlabStableVersionEntry{
				ID:      "aaa-hostname-aaa",
				OS:      "AAA-CROS-AAA",
				FW:      "AAA-FW-AAA",
				FWImage: "AAA-FWImage-AAA",
			},
			req: &fleet.GetStableVersionRequest{
				Hostname: "aaa-hostname-aaa",
			},
		},
		{
			name: "hostname",
			in: &SatlabStableVersionEntry{
				ID:      "aaa-board-aaa|aaa-model-aaa",
				OS:      "AAA-CROS-AAA",
				FW:      "AAA-FW-AAA",
				FWImage: "AAA-FWImage-AAA",
			},
			req: &fleet.GetStableVersionRequest{
				BuildTarget: "AAA-BOARD-AAA",
				Model:       "AAA-MODEL-AAA",
			},
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := gaetesting.TestingContext()
			datastore.GetTestable(ctx).Consistent(true)

			expected := tt.in

			if err := PutSatlabStableVersionEntry(ctx, tt.in); err != nil {
				t.Errorf("unexpected error: %s", err)
			}
			actual, err := GetSatlabStableVersionEntryByID(ctx, tt.req)
			if err != nil {
				t.Errorf("unexpected error: %s", err)
			}

			if diff := cmp.Diff(expected, actual, cmpopts.IgnoreUnexported(SatlabStableVersionEntry{})); diff != "" {
				t.Errorf("unexpected diff: %s", diff)
			}
		})
	}
}

// TestDeleteSatlabStableVersionEntry tests deleting a satlab stable version entry.
func TestDeleteSatlabStableVersionEntry(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContext()
	datastore.GetTestable(ctx).Consistent(true)

	if err := PutSatlabStableVersionEntry(ctx, &SatlabStableVersionEntry{
		ID: "aaaa",
	}); err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	if err := DeleteSatlabStableVersionEntryByRawID(ctx, "aaaa"); err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	_, err := GetSatlabStableVersionEntryByRawID(ctx, "aaaa")
	if err == nil {
		t.Errorf("unexpected success: getting deleted entry should fail")
	}
	if !datastore.IsErrNoSuchEntity(err) {
		t.Errorf("unexpected error: %s", err)
	}
}
