// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package recovery

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"infra/cros/recovery/internal/loader"
)

// verifyConfig verifies that configuration can be parsed and contains all execs present in library.
// Fail if:
// 1) Cannot parse by loader,
// 2) Missing dependency, condition or recovery action used in the actions,
// 3) Used unknown exec function.
func verifyConfig(t *testing.T, c io.Reader) {
	ctx := context.Background()
	p, err := loader.LoadConfiguration(ctx, c)
	if err != nil {
		t.Errorf("expected to pass by failed with error: %s", err)
	}
	if p == nil {
		t.Errorf("default config is empty")
	}
}

// TestLabstationRepairConfig verifies the labstation repair configuration.
func TestLabstationRepairConfig(t *testing.T) {
	t.Parallel()
	verifyConfig(t, LabstationRepairConfig())
}

// TestLabstationDeployConfig verifies the labstation deploy configuration.
func TestLabstationDeployConfig(t *testing.T) {
	t.Parallel()
	verifyConfig(t, LabstationDeployConfig())
}

// TestCrosRepairConfig verifies the cros repair configuration.
func TestCrosRepairConfig(t *testing.T) {
	t.Parallel()
	verifyConfig(t, CrosRepairConfig())
}

// TestCrosDeployConfig verifies the cros deploy configuration.
func TestCrosDeployConfig(t *testing.T) {
	t.Parallel()
	verifyConfig(t, CrosDeployConfig())
}

var generateCases = []struct {
	name string
	in   []configPlan
	out  string
}{
	{
		"empty",
		nil,
		"",
	},
	{
		"single plan",
		[]configPlan{
			{
				"p1",
				`"critical_actions":[], "actions":{}`,
				false,
			},
		},
		`{"plan_names":["p1"],"plans": {"p1":{"critical_actions":[], "actions":{}, "allow_fail":false}}}`,
	},
	{
		"full",
		[]configPlan{
			{
				"p1",
				`"critical_actions":["c1","c2"], "actions":{"c1":{"dependencies":["c2"],"exec_name":"sample_pass"},"c2":{"exec_name":"sample_fail"}}`,
				false,
			},
			{
				"p2",
				`"critical_actions":["c3","c4"], "actions":{"c3":{"exec_name":"sample_pass"},"c4":{"exec_name":"sample_fail"}}`,
				true,
			},
		},
		`{"plan_names":["p1","p2"],"plans": {"p1":{"critical_actions":["c1","c2"], "actions":{"c1":{"dependencies":["c2"],"exec_name":"sample_pass"},"c2":{"exec_name":"sample_fail"}}, "allow_fail":false},"p2":{"critical_actions":["c3","c4"], "actions":{"c3":{"exec_name":"sample_pass"},"c4":{"exec_name":"sample_fail"}}, "allow_fail":true}}}`,
	},
}

func TestCreateConfiguration(t *testing.T) {
	t.Parallel()
	for _, c := range generateCases {
		cs := c
		t.Run(cs.name, func(t *testing.T) {
			ctx := context.Background()
			c := createConfiguration(cs.in)
			if d := cmp.Diff(c, cs.out); d != "" {
				t.Errorf("diff: %v\nwant: %v", d, cs.out)
			}
			if cs.out != "" {
				p, err := loader.LoadConfiguration(ctx, strings.NewReader(c))
				if err != nil {
				}
				if err != nil {
					t.Errorf("expected to pass by failed with error: %s", err)
				}
				if p == nil {
					t.Errorf("default config is empty")
				}
			}
		})
	}
}
