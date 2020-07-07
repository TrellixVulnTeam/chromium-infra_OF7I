// Copyright 2015 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package analyzer

import (
	"net/url"
	"testing"

	"golang.org/x/net/context"

	"infra/monitoring/messages"
)

func urlParse(s string, t *testing.T) *url.URL {
	p, err := url.Parse(s)
	if err != nil {
		t.Errorf("failed to parse %s: %s", s, err)
	}
	return p
}

func newTestAnalyzer(minBuilds, maxBuilds int) *Analyzer {
	a := New(minBuilds, maxBuilds)
	a.CrBug = nil
	a.FindIt = nil

	return a
}

func TestExcludeFailure(t *testing.T) {
	tests := []struct {
		name                        string
		gk                          messages.GatekeeperConfig
		gkt                         map[string][]messages.TreeMasterConfig
		master, builder, step, tree string
		want                        bool
	}{
		{
			name:    "empty config",
			tree:    "test_tree1",
			master:  "fake.master",
			builder: "fake.builder",
			step:    "fake_step",
			want:    false,
		},
		{
			name:    "specifically excluded builder",
			tree:    "test_tree2",
			master:  "fake.master",
			builder: "fake.builder",
			step:    "fake_step",
			gk: messages.GatekeeperConfig{Masters: map[string][]messages.MasterConfig{
				"https://ci.chromium.org/p/fake.master": {{
					ExcludedBuilders: []string{"fake.builder"},
				}},
			}},
			want: true,
		},
		{
			name:    "specifically excluded master step",
			tree:    "test_tree3",
			master:  "fake.master",
			builder: "fake.builder",
			step:    "fake_step",
			gk: messages.GatekeeperConfig{Masters: map[string][]messages.MasterConfig{
				"https://ci.chromium.org/p/fake.master": {{
					ExcludedSteps: []string{"fake_step"},
				}},
			}},
			want: true,
		},
		{
			name:    "specifically excluded builder step",
			tree:    "test_tree4",
			master:  "fake.master",
			builder: "fake.builder",
			step:    "fake_step",
			gk: messages.GatekeeperConfig{Masters: map[string][]messages.MasterConfig{
				"https://ci.chromium.org/p/fake.master": {{
					Builders: map[string]messages.BuilderConfig{
						"fake.builder": {
							ExcludedSteps: []string{"fake_step"},
						},
					}},
				}},
			},
			want: true,
		},
		{
			name:    "wildcard builder excluded",
			tree:    "test_tree5",
			master:  "fake.master",
			builder: "fake.builder",
			step:    "fake_step",
			gk: messages.GatekeeperConfig{Masters: map[string][]messages.MasterConfig{
				"https://ci.chromium.org/p/fake.master": {{
					ExcludedBuilders: []string{"*"},
				}},
			}},
			want: true,
		},
		{
			name:    "config should exclude builder (tree config)",
			tree:    "test_tree6",
			master:  "fake.master",
			builder: "fake.builder",
			step:    "fake_step",
			gk: messages.GatekeeperConfig{Masters: map[string][]messages.MasterConfig{
				"https://ci.chromium.org/p/fake.master": {{
					Builders: map[string]messages.BuilderConfig{
						"*": {},
					},
				}},
			}},
			gkt: map[string][]messages.TreeMasterConfig{
				"test_tree": {
					messages.TreeMasterConfig{
						Masters: map[messages.MasterLocation][]string{
							{URL: *urlParse(
								"https://ci.chromium.org/p/fake.master", t)}: {"other.builder"},
						},
					},
				},
			},
			want: true,
		},
		{
			name:    "config shouldn't exclude builder (tree config)",
			tree:    "test_tree7",
			master:  "fake.master",
			builder: "fake.builder",
			step:    "fake_step",
			gk: messages.GatekeeperConfig{Masters: map[string][]messages.MasterConfig{
				"https://ci.chromium.org/p/fake.master": {{
					Builders: map[string]messages.BuilderConfig{
						"*": {},
					},
				}},
			}},
			gkt: map[string][]messages.TreeMasterConfig{
				"test_tree7": {
					messages.TreeMasterConfig{
						Masters: map[messages.MasterLocation][]string{
							{URL: *urlParse(
								"https://ci.chromium.org/p/fake.master", t)}: {"fake.builder"},
						},
					},
				},
			},
			want: false,
		},
		{
			name:    "config shouldn't exclude builder (tree config glob)",
			tree:    "test_tree8",
			master:  "fake.master",
			builder: "fake.builder",
			step:    "fake_step",
			gk: messages.GatekeeperConfig{Masters: map[string][]messages.MasterConfig{
				"https://ci.chromium.org/p/fake.master": {{
					Builders: map[string]messages.BuilderConfig{
						"*": {},
					},
				}},
			}},
			gkt: map[string][]messages.TreeMasterConfig{
				"test_tree8": {
					messages.TreeMasterConfig{
						Masters: map[messages.MasterLocation][]string{
							{URL: *urlParse(
								"https://ci.chromium.org/p/fake.master", t)}: {"*"},
						},
					},
				},
			},
			want: false,
		},
		{
			name:    "partial glob step excluded",
			tree:    "test_tree9",
			master:  "fake.master",
			builder: "fake.builder",
			step:    "fake_step (experimental)",
			gk: messages.GatekeeperConfig{Masters: map[string][]messages.MasterConfig{
				"https://ci.chromium.org/p/fake.master": {{
					Builders: map[string]messages.BuilderConfig{
						"*": {},
					},
					ExcludedSteps: []string{
						"* (experimental)",
					},
				}},
			}},
			gkt: map[string][]messages.TreeMasterConfig{
				"test_tree9": {
					messages.TreeMasterConfig{
						Masters: map[messages.MasterLocation][]string{
							{URL: *urlParse(
								"https://ci.chromium.org/p/fake.master", t)}: {"*"},
						},
					},
				},
			},
			want: true,
		},
		{
			name:    "partial glob step excluded by builder category",
			tree:    "test_tree10",
			master:  "fake.master",
			builder: "fake.builder",
			step:    "fake_step (experimental)",
			gk: messages.GatekeeperConfig{
				Masters: map[string][]messages.MasterConfig{
					"https://ci.chromium.org/p/fake.master": {{
						Builders: map[string]messages.BuilderConfig{
							"fake.builder": {
								Categories: []string{
									"experimental_tests",
								},
							},
						},
					}},
				},
				Categories: map[string]messages.CategoryConfig{
					"experimental_tests": {
						ExcludedSteps: []string{
							"* (experimental)",
						},
					},
				},
			},
			want: true,
		},
		{
			name:    "partial glob step excluded by master category",
			tree:    "test_tree11",
			master:  "fake.master",
			builder: "fake.builder",
			step:    "fake_step (experimental)",
			gk: messages.GatekeeperConfig{
				Masters: map[string][]messages.MasterConfig{
					"https://ci.chromium.org/p/fake.master": {{
						Builders: map[string]messages.BuilderConfig{
							"*": {},
						},
						Categories: []string{
							"experimental_tests",
						},
					}},
				},
				Categories: map[string]messages.CategoryConfig{
					"experimental_tests": {
						ExcludedSteps: []string{
							"* (experimental)",
						},
					},
				},
			},
			want: true,
		},
	}

	ctx := context.Background()

	a := newTestAnalyzer(0, 10)
	for _, test := range tests {
		a.Gatekeeper = NewGatekeeperRules(ctx, []*messages.GatekeeperConfig{&test.gk}, test.gkt)
		got := a.Gatekeeper.ExcludeFailure(ctx, test.tree, &messages.MasterLocation{URL: *urlParse("https://ci.chromium.org/p/"+test.master, t)}, test.builder, test.step)
		if got != test.want {
			t.Errorf("%s failed. Got: %+v, want: %+v", test.name, got, test.want)
		}
	}
}
