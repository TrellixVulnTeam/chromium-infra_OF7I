// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"infra/appengine/crosskylabadmin/internal/app/config"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// TestIsRecoveryEnabledForLabstation tests that we correctly make
// a decision on whether to use recovery for labstations based on the config
// file.
func TestIsRecoveryEnabledForLabstation(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   *config.Paris
		out  bool
	}{
		{
			// If Paris is not enabled at all in the config, we should default
			// to always using the legacy flow.
			name: "default config without paris",
			in:   nil,
			out:  false,
		},
		{
			// If the recover labstation feature is enabled, every labstation should be opted in.
			name: "only enable labstation repair",
			in: &config.Paris{
				EnableLabstationRecovery: true,
			},
			out: true,
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
			actual := isRecoveryEnabledForLabstation(ctx)
			if diff := cmp.Diff(expected, actual); diff != "" {
				t.Errorf("unexpected diff (-want +got) in subtest %d: %s", i, diff)
			}
		})
	}
}
