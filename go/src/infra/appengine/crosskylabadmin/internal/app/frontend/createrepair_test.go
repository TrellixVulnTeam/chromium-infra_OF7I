// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"context"
	"math"
	"math/rand"
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

// TestRouteRepairTaskImplDUT tests that non-labstation DUTs that would qualify for the Paris flow
// are still blocked.
func TestRouteRepairTaskImplDUT(t *testing.T) {
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
			name: "good DUT is still blocked",
			in: &config.RolloutConfig{
				Enable:          true,
				OptinAllDuts:    true,
				RolloutPermille: 1000,
			},
			randFloat: 0.5,
			pools:     []string{"pool"},
			out:       legacy,
			reason:    notALabstation,
		},
	}

	for i, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			expected := tt.out
			expectedReason := reasonMessageMap[tt.reason]
			if expectedReason == "" {
				t.Errorf("expected reason should be valid reason")
			}
			actual, r := routeRepairTaskImpl(
				ctx,
				tt.in,
				&dutRoutingInfo{
					labstation: false,
					pools:      tt.pools,
				},
				tt.randFloat,
			)
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

// TestRouteRepairTaskImplLabstation tests that we correctly make
// a decision on whether to use recovery for labstations based on the config
// file.
func TestRouteRepairTaskImplLabstation(t *testing.T) {
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
			name: "do not use labstation",
			in: &config.RolloutConfig{
				Enable:       true,
				OptinAllDuts: true,
			},
			randFloat: 0.5,
			pools:     []string{"some pool"},
			out:       legacy,
			reason:    scoreTooHigh,
		},
		{
			name: "do use labstation",
			in: &config.RolloutConfig{
				Enable:          true,
				OptinAllDuts:    true,
				RolloutPermille: 1000,
			},
			randFloat: 0.5,
			pools:     []string{"some pool"},
			out:       paris,
			reason:    scoreBelowThreshold,
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
				RolloutPermille: 501,
				OptinAllDuts:    true,
			},
			pools:     []string{"some-pool"},
			randFloat: 0.5,
			out:       paris,
			reason:    scoreBelowThreshold,
		},
		{
			name: "use permille even when all labstations are opted in",
			in: &config.RolloutConfig{
				Enable:          true,
				RolloutPermille: 499,
				OptinAllDuts:    true,
			},
			pools:     []string{"some-pool"},
			randFloat: 0.5,
			out:       legacy,
			reason:    scoreTooHigh,
		},
		{
			name: "use labstation sometimes - good",
			in: &config.RolloutConfig{
				Enable:          true,
				RolloutPermille: 501,
				OptinAllDuts:    false,
			},
			pools:     []string{"some-pool"},
			randFloat: 0.5,
			out:       paris,
			reason:    scoreBelowThreshold,
		},
		{
			name: "use labstation sometimes - near miss",
			in: &config.RolloutConfig{
				Enable:          true,
				RolloutPermille: 499,
			},
			pools:     []string{"some-pool"},
			randFloat: 0.5,
			out:       legacy,
			reason:    scoreTooHigh,
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
			reason:    scoreBelowThreshold,
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
		{
			name: "ignore UFS error",
			in: &config.RolloutConfig{
				Enable:          true,
				RolloutPermille: 500,
				OptinAllDuts:    true,
				UfsErrorPolicy:  "lax",
			},
			randFloat: 0.5,
			out:       paris,
			reason:    scoreBelowThreshold,
		},
		{
			name: "don't ignore UFS error if we're above the threshold",
			in: &config.RolloutConfig{
				Enable:          true,
				RolloutPermille: 498,
				OptinAllDuts:    true,
				UfsErrorPolicy:  "lax",
			},
			randFloat: 0.5,
			out:       legacy,
			reason:    scoreTooHigh,
		},
	}

	for i, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			expected := tt.out
			expectedReason := reasonMessageMap[tt.reason]
			if expectedReason == "" {
				t.Errorf("expected reason should be valid reason")
			}
			actual, r := routeRepairTaskImpl(
				ctx,
				tt.in,
				&dutRoutingInfo{
					labstation: true,
					pools:      tt.pools,
				},
				tt.randFloat,
			)
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

// TestRouteRepairTask tests the RouteRepairTask function, which delegates most of the decision logic to
// routeLabstationRepairTask in a few simple cases.
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
					Enable:          true,
					OptinAllDuts:    true,
					RolloutPermille: 1000,
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

// TestRouteRepairTaskProbability tests that the probability that a labstation is sent to one path vs
// another is reasonsable. See b:216499840 for details.
func TestRouteRepairTaskProbability(t *testing.T) {
	// t.Parallel -- This test is sensitive to the state of the random number generator.
	//               Do not run it in parallel with anything else.

	const samples = 1000 * 1000

	// Make this test deterministic by configuring the RNG with a specific seed.
	// Save a random number from before we set the seed so that we can re-seed the RNG
	// when the test exits.
	seedForLater := int64(rand.Uint64())
	rand.Seed(1)
	defer rand.Seed(seedForLater)

	ctx := context.Background()

	rolloutCfg := &config.RolloutConfig{
		Enable:          true,
		OptinAllDuts:    true,
		RolloutPermille: 1,
	}

	tally := 0

	for i := 0; i < samples; i++ {
		dest, reason := routeRepairTaskImpl(
			ctx,
			rolloutCfg,
			&dutRoutingInfo{
				labstation: true,
				pools:      []string{"pool1"},
			},
			rand.Float64(),
		)
		switch reason {
		case scoreBelowThreshold:
			// do nothing
		case scoreTooHigh:
			// do nothing
		default:
			t.Errorf("unexpected reason: %q", reasonMessageMap[reason])
		}
		if dest == paris {
			tally++
		}
	}

	// The tolerance here is extremely wide compared to the standard deviation, which is sqrt{p(1-p)/n}, with n being
	// the number of samples we are averaging together.
	//
	// However, this test is mostly interested in the case where the interpretation of rolloutPermille is backwards,
	// so a wide tolerance is acceptable.
	expected := 0.001 * samples
	tol := 3 * expected
	dist := math.Abs(float64(tally) - expected)

	if dist > tol {
		t.Errorf("difference %f is too high", dist)
	}
}
