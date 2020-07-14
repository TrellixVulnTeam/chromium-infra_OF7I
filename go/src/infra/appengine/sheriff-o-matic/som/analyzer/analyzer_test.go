// Copyright 2015 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package analyzer

import (
	"net/url"
	"testing"

	"golang.org/x/net/context"
)

func urlParse(s string, t *testing.T) *url.URL {
	p, err := url.Parse(s)
	if err != nil {
		t.Errorf("failed to parse %s: %s", s, err)
	}
	return p
}

func TestExcludeFailure(t *testing.T) {
	tests := []struct {
		name                  string
		cr                    *ConfigRules
		master, builder, step string
		want                  bool
	}{
		{
			name:    "empty config",
			master:  "fake.master",
			builder: "fake.builder",
			step:    "fake_step",
			cr:      &ConfigRules{},
			want:    false,
		},
		{
			name:    "specifically excluded builder",
			master:  "fake.master",
			builder: "fake.builder",
			step:    "fake_step",
			cr: &ConfigRules{
				MasterCfgs: map[string]MasterConfig{
					"fake.master": {
						ExcludedBuilders: []string{"fake.builder"},
					},
				},
			},
			want: true,
		},
		{
			name:    "specifically excluded master step",
			master:  "fake.master",
			builder: "fake.builder",
			step:    "fake_step",
			cr: &ConfigRules{
				IgnoredSteps: []string{"fake_step"},
			},
			want: true,
		},
		{
			name:    "specifically excluded builder step",
			master:  "fake.master",
			builder: "fake.builder",
			step:    "fake_step",
			cr: &ConfigRules{
				IgnoredSteps: []string{"fake_step"},
			},
			want: true,
		},
		{
			name:    "partial glob step excluded",
			master:  "fake.master",
			builder: "fake.builder",
			step:    "fake_step (experimental)",
			cr: &ConfigRules{
				IgnoredSteps: []string{"* (experimental)"},
			},
			want: true,
		},
	}

	ctx := context.Background()

	for _, test := range tests {
		got := test.cr.ExcludeFailure(ctx, test.master, test.builder, test.step)
		if got != test.want {
			t.Errorf("%s failed for config rules. Got: %+v, want: %+v", test.name, got, test.want)
		}
	}
}
