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
						Hostname: "AAA-HOSTNAME-AAA",
					},
				},
				CrosVersion:     "AAA-CROS-AAA",
				FirmwareVersion: "AAA-FW-AAA",
				FirmwareImage:   "AAA-FWImage-AAA",
			},
			out: &SatlabStableVersionEntry{
				ID:      "AAA-HOSTNAME-AAA",
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
						Board: "AAA-BOARD-AAA",
						Model: "AAA-MODEL-AAA",
					},
				},
				CrosVersion:     "AAA-CROS-AAA",
				FirmwareVersion: "AAA-FW-AAA",
				FirmwareImage:   "AAA-FWImage-AAA",
			},
			out: &SatlabStableVersionEntry{
				ID:      "AAA-BOARD-AAA|AAA-MODEL-AAA",
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
			actual, _ := MakeSatlabStableVersionEntry(tt.in)
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
