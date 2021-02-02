// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmd

import (
	"testing"

	"go.chromium.org/chromiumos/infra/proto/go/test_platform"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/phosphorus"
)

func TestShouldRunTLSProvision(t *testing.T) {
	cases := []struct {
		Tag                          string
		DesiredProvisionableLabel    string
		ProvisionDutExperimentConfig *phosphorus.ProvisionDutExperiment
		Want                         bool
	}{
		{
			Tag:  "nil config",
			Want: false,
		},
		{
			Tag: "globally disabled",
			ProvisionDutExperimentConfig: &phosphorus.ProvisionDutExperiment{
				Enabled: false,
			},
			Want: false,
		},
		{
			Tag:                       "no allow or disallow list",
			DesiredProvisionableLabel: "octopus-release/R90-13749.0.0",
			ProvisionDutExperimentConfig: &phosphorus.ProvisionDutExperiment{
				Enabled: true,
			},
			Want: false,
		},
		{
			Tag:                       "included in allow_list",
			DesiredProvisionableLabel: "octopus-release/R90-13749.0.0",
			ProvisionDutExperimentConfig: &phosphorus.ProvisionDutExperiment{
				Enabled: true,
				CrosVersionSelector: &phosphorus.ProvisionDutExperiment_CrosVersionAllowList{
					CrosVersionAllowList: &phosphorus.ProvisionDutExperiment_CrosVersionSelector{
						Prefixes: []string{"octopus-release", "atlas-cq"},
					},
				},
			},
			Want: true,
		},
		{
			Tag:                       "not included in allow_list",
			DesiredProvisionableLabel: "octopus-release/R90-13749.0.0",
			ProvisionDutExperimentConfig: &phosphorus.ProvisionDutExperiment{
				Enabled: true,
				CrosVersionSelector: &phosphorus.ProvisionDutExperiment_CrosVersionAllowList{
					CrosVersionAllowList: &phosphorus.ProvisionDutExperiment_CrosVersionSelector{
						Prefixes: []string{"octopus-release/R87", "atlas-cq"},
					},
				},
			},
			Want: false,
		},
		{
			Tag:                       "included in disallow_list",
			DesiredProvisionableLabel: "octopus-release/R90-13749.0.0",
			ProvisionDutExperimentConfig: &phosphorus.ProvisionDutExperiment{
				Enabled: true,
				CrosVersionSelector: &phosphorus.ProvisionDutExperiment_CrosVersionDisallowList{
					CrosVersionDisallowList: &phosphorus.ProvisionDutExperiment_CrosVersionSelector{
						Prefixes: []string{"octopus-release", "atlas-cq"},
					},
				},
			},
			Want: false,
		},
		{
			Tag:                       "not included in disallow_list",
			DesiredProvisionableLabel: "octopus-release/R90-13749.0.0",
			ProvisionDutExperimentConfig: &phosphorus.ProvisionDutExperiment{
				Enabled: true,
				CrosVersionSelector: &phosphorus.ProvisionDutExperiment_CrosVersionDisallowList{
					CrosVersionDisallowList: &phosphorus.ProvisionDutExperiment_CrosVersionSelector{
						Prefixes: []string{"octopus-release/R91", "atlas-cq"},
					},
				},
			},
			Want: true,
		},
	}

	for _, c := range cases {
		t.Run(c.Tag, func(t *testing.T) {
			r := &phosphorus.PrejobRequest{
				SoftwareDependencies: []*test_platform.Request_Params_SoftwareDependency{
					{
						Dep: &test_platform.Request_Params_SoftwareDependency_ChromeosBuild{
							ChromeosBuild: c.DesiredProvisionableLabel,
						},
					},
				},
				Config: &phosphorus.Config{
					PrejobStep: &phosphorus.PrejobStep{
						ProvisionDutExperiment: c.ProvisionDutExperimentConfig,
					},
				},
			}
			if b := shouldProvisionChromeOSViaTLS(r); b != c.Want {
				t.Errorf("Incorrect response from shouldRunTLSProvision(%v): %t, want %t", r, b, c.Want)
			}
		})
	}
}
