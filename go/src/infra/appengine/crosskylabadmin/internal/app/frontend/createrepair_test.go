// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"infra/appengine/crosskylabadmin/internal/app/config"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// TestRouteRepairTask tests that we correctly make
// a decision on whether to use recovery for labstations based on the config
// file.
func TestRouteRepairTask(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name          string
		in            *config.Paris
		botID         string
		expectedState string
		randFloat     float64
		out           string
		hasErr        bool
	}{
		{
			name:          "default config",
			in:            nil,
			botID:         "foo-labstation1",
			expectedState: "ready",
			randFloat:     0.5,
			out:           "",
			hasErr:        false,
		},
		{
			name: "use labstation",
			in: &config.Paris{
				EnableLabstationRecovery: true,
				OptinAllLabstations:      true,
			},
			botID:         "foo-labstation1",
			expectedState: "ready",
			randFloat:     0.5,
			out:           "paris",
			hasErr:        false,
		},
		{
			name: "use labstation -- default threshold of zero is not okay",
			in: &config.Paris{
				EnableLabstationRecovery: true,
				OptinAllLabstations:      false,
			},
			botID:         "foo-labstation1",
			expectedState: "ready",
			randFloat:     0,
			out:           "",
			hasErr:        true,
		},
		{
			name: "use labstation sometimes - good",
			in: &config.Paris{
				EnableLabstationRecovery:   true,
				LabstationRecoveryPermille: 499,
			},
			botID:         "foo-labstation1",
			expectedState: "ready",
			randFloat:     0.5,
			out:           "paris",
			hasErr:        false,
		},
		{
			name: "use labstation sometimes - near miss",
			in: &config.Paris{
				EnableLabstationRecovery:   true,
				LabstationRecoveryPermille: 501,
			},
			botID:         "foo-labstation1",
			expectedState: "ready",
			randFloat:     0.5,
			out:           "",
			hasErr:        false,
		},
	}

	for i, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Use the default testing context. It has a basic config
			// with no Paris subconfig.
			ctx := testingContext()
			// Attach our newfangled Paris subconfig into the context.
			cfg := config.Get(ctx)
			cfg.Paris = tt.in
			ctx = config.Use(ctx, cfg)
			expected := tt.out
			actual, err := RouteRepairTask(ctx, tt.botID, tt.expectedState, tt.randFloat)
			if diff := cmp.Diff(expected, actual); diff != "" {
				t.Errorf("unexpected diff (-want +got) in subtest %d: %s", i, diff)
			}
			if tt.hasErr {
				if err == nil {
					t.Errorf("expected error but didn't get one")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %s", err)
				}
			}
		})
	}
}
