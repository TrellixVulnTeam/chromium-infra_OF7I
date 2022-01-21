// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"infra/appengine/crosskylabadmin/internal/app/config"
)

// TestIsDisjoint tests that isDisjoint(a, b) returns true if and only if
// the intersection of a and b (interpreted as sets) is ∅.
func TestIsDisjoint(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		a    []string
		b    []string
		out  bool
	}{
		{
			name: "nil nil is technically disjoint",
			a:    nil,
			b:    nil,
			out:  true,
		},
		{
			name: "[] nil is technically disjoint",
			a:    []string{},
			b:    nil,
			out:  true,
		},
		{
			name: "nil [] is technically disjoint",
			a:    nil,
			b:    []string{},
			out:  true,
		},
		{
			name: "[] [] is technically disjoint",
			a:    []string{},
			b:    []string{},
			out:  true,
		},
		{
			name: `["a"] [] is disjoint`,
			a:    []string{"a"},
			b:    []string{},
			out:  true,
		},
		{
			name: `[] ["a"] is disjoint`,
			a:    []string{"a"},
			b:    []string{},
			out:  true,
		},
		{
			name: `["a"] ["a"] is NOT disjoint`,
			a:    []string{"a"},
			b:    []string{"a"},
			out:  false,
		},
		{
			name: `["a"] ["b"] is disjoint`,
			a:    []string{"a"},
			b:    []string{"b"},
			out:  true,
		},
	}

	for i, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			expected := tt.out
			actual := isDisjoint(tt.a, tt.b)
			if diff := cmp.Diff(expected, actual); diff != "" {
				t.Errorf("unexpected diff (-want +got) in subtest %d: %s", i, diff)
			}
		})
	}

}

// TestRouteLabstationRepairTask tests that we correctly make
// a decision on whether to use recovery for labstations based on the config
// file.
func TestRouteLabstationRepairTask(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		in        *config.RolloutConfig
		pools     []string
		randFloat float64
		out       string
		reason    reason
	}{
		{
			name:      "default config",
			in:        nil,
			randFloat: 0.5,
			pools:     nil,
			out:       legacy,
			reason:    parisNotEnabled,
		},
		{
			name: "use labstation",
			in: &config.RolloutConfig{
				Enable:       true,
				OptinAllDuts: true,
			},
			randFloat: 0.5,
			pools:     []string{"some pool"},
			out:       paris,
			reason:    scoreExceedsThreshold,
		},
		{
			name: "no pool means UFS error",
			in: &config.RolloutConfig{
				Enable:       true,
				OptinAllDuts: true,
			},
			pools:     nil,
			randFloat: 1,
			out:       legacy,
			reason:    noPools,
		},
		{
			name: "use labstation -- default threshold of zero is not okay",
			in: &config.RolloutConfig{
				Enable:       true,
				OptinAllDuts: false,
			},
			pools:     []string{"some-pool"},
			randFloat: 0,
			out:       legacy,
			reason:    thresholdZero,
		},
		{
			name: "all labstations are opted in",
			in: &config.RolloutConfig{
				Enable:          true,
				RolloutPermille: 499,
				OptinAllDuts:    true,
			},
			pools:     []string{"some-pool"},
			randFloat: 0.5,
			out:       paris,
			reason:    scoreExceedsThreshold,
		},
		{
			name: "use permille even when all labstations are opted in",
			in: &config.RolloutConfig{
				Enable:          true,
				RolloutPermille: 501,
				OptinAllDuts:    true,
			},
			pools:     []string{"some-pool"},
			randFloat: 0.5,
			out:       legacy,
			reason:    scoreTooLow,
		},
		{
			name: "use labstation sometimes - good",
			in: &config.RolloutConfig{
				Enable:          true,
				RolloutPermille: 499,
				OptinAllDuts:    false,
			},
			pools:     []string{"some-pool"},
			randFloat: 0.5,
			out:       paris,
			reason:    scoreExceedsThreshold,
		},
		{
			name: "use labstation sometimes - near miss",
			in: &config.RolloutConfig{
				Enable:          true,
				RolloutPermille: 501,
			},
			pools:     []string{"some-pool"},
			randFloat: 0.5,
			out:       legacy,
			reason:    scoreTooLow,
		},
		{
			name: "good pool",
			in: &config.RolloutConfig{
				Enable:          true,
				RolloutPermille: 500,
				OptinAllDuts:    false,
				OptinDutPool:    []string{"paris"},
			},
			pools:     []string{"paris"},
			randFloat: 0.5,
			out:       paris,
			reason:    scoreExceedsThreshold,
		},
		{
			name: "bad pool",
			in: &config.RolloutConfig{
				Enable:          true,
				RolloutPermille: 500,
				OptinAllDuts:    false,
				OptinDutPool:    []string{"paris"},
			},
			pools:     []string{"NOT PARIS"},
			randFloat: 0.5,
			out:       legacy,
			reason:    wrongPool,
		},
	}

	for i, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			expected := tt.out
			expectedReason := reasonMessageMap[tt.reason]
			if expectedReason == "" {
				t.Errorf("expected reason should be valid reason")
			}
			actual, r := routeLabstationRepairTask(tt.in, tt.pools, tt.randFloat)
			actualReason := reasonMessageMap[r]
			if diff := cmp.Diff(expected, actual); diff != "" {
				t.Errorf("unexpected diff (-want +got) in subtest %d %q: %s", i, tt.name, diff)
			}
			if diff := cmp.Diff(expectedReason, actualReason); diff != "" {
				t.Errorf("unexpected diff (-want +got) in subtest %d %q: %s", i, tt.name, diff)
			}
		})
	}
}

func TestRouteRepairTask(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name          string
		in            *config.Paris
		botID         string
		expectedState string
		pools         []string
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
			out:           legacy,
			hasErr:        false,
		},
		{
			name: "paris labstation",
			in: &config.Paris{
				LabstationRepair: &config.RolloutConfig{
					Enable:       true,
					OptinAllDuts: true,
				},
			},
			botID:         "foo-labstation1",
			expectedState: "ready",
			pools:         []string{"some-pool"},
			randFloat:     1,
			out:           paris,
			hasErr:        false,
		},
		{
			name: "legacy labstation",
			in: &config.Paris{
				LabstationRepair: &config.RolloutConfig{
					Enable: false,
				},
			},
			botID:         "foo-labstation1",
			expectedState: "ready",
			pools:         nil,
			randFloat:     1,
			out:           legacy,
			hasErr:        false,
		},
	}

	for i, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := testingContext()
			cfg := config.Get(ctx)
			cfg.Paris = tt.in
			ctx = config.Use(ctx, cfg)
			expected := tt.out
			actual, err := RouteRepairTask(ctx, tt.botID, tt.expectedState, tt.pools, tt.randFloat)
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
