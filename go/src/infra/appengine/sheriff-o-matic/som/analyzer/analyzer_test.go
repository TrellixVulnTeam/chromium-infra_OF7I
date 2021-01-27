// Copyright 2015 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package analyzer

import (
	"context"
	"testing"
)

func TestExcludeFailure(t *testing.T) {
	tests := []struct {
		name                        string
		cr                          *ConfigRules
		buildergroup, builder, step string
		want                        bool
	}{
		{
			name:         "empty config",
			buildergroup: "fake.buildergroup",
			builder:      "fake.builder",
			step:         "fake_step",
			cr:           &ConfigRules{},
			want:         false,
		},
		{
			name:         "specifically excluded builder",
			buildergroup: "fake.buildergroup",
			builder:      "fake.builder",
			step:         "fake_step",
			cr: &ConfigRules{
				BuilderGroupCfgs: map[string]BuilderGroupConfig{
					"fake.buildergroup": {
						ExcludedBuilders: []string{"fake.builder"},
					},
				},
			},
			want: true,
		},
		{
			name:         "specifically excluded buildergroup step",
			buildergroup: "fake.buildergroup",
			builder:      "fake.builder",
			step:         "fake_step",
			cr: &ConfigRules{
				IgnoredSteps: []string{"fake_step"},
			},
			want: true,
		},
		{
			name:         "specifically excluded builder step",
			buildergroup: "fake.buildergroup",
			builder:      "fake.builder",
			step:         "fake_step",
			cr: &ConfigRules{
				IgnoredSteps: []string{"fake_step"},
			},
			want: true,
		},
		{
			name:         "partial glob step excluded",
			buildergroup: "fake.buildergroup",
			builder:      "fake.builder",
			step:         "fake_step (experimental)",
			cr: &ConfigRules{
				IgnoredSteps: []string{"* (experimental)"},
			},
			want: true,
		},
	}

	ctx := context.Background()

	for _, test := range tests {
		got := test.cr.ExcludeFailure(ctx, test.buildergroup, test.builder, test.step)
		if got != test.want {
			t.Errorf("%s failed for config rules. Got: %+v, want: %+v", test.name, got, test.want)
		}
	}
}
