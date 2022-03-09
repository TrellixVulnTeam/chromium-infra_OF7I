// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package recovery

import (
	"bytes"
	"context"
	"encoding/json"
	"infra/cros/recovery/internal/loader"
	"infra/cros/recovery/internal/planpb"
	"io"
	"testing"

	"github.com/golang/protobuf/proto"
)

// verifyConfig verifies that configuration can be parsed and contains all execs present in library.
// Fail if:
// 1) Cannot parse by loader,
// 2) Missing dependency, condition or recovery action used in the actions,
// 3) Used unknown exec function.
func verifyConfig(name string, t *testing.T, c io.Reader) {
	ctx := context.Background()
	p, err := loader.LoadConfiguration(ctx, c)
	if err != nil {
		t.Errorf("%q expected to pass but failed with error: %s", name, err)
	}
	if p == nil {
		t.Errorf("%q default config is empty", name)
	}
}

// TestLabstationRepairConfig verifies the labstation repair configuration.
func TestLabstationRepairConfig(t *testing.T) {
	t.Parallel()
	verifyConfig("labstation-repair", t, LabstationRepairConfig())
}

// TestLabstationDeployConfig verifies the labstation deploy configuration.
func TestLabstationDeployConfig(t *testing.T) {
	t.Parallel()
	verifyConfig("labstation-deploy", t, LabstationDeployConfig())
}

// TestCrosRepairConfig verifies the cros repair configuration.
func TestCrosRepairConfig(t *testing.T) {
	t.Parallel()
	verifyConfig("dut-repair", t, CrosRepairConfig())
}

// TestCrosDeployConfig verifies the cros deploy configuration.
func TestCrosDeployConfig(t *testing.T) {
	t.Parallel()
	verifyConfig("dut-deploy", t, CrosDeployConfig())
}

var generateCases = []struct {
	name string
	in   []*planpb.Plan
	want string
}{
	{
		"empty",
		nil,
		"",
	},
	{
		"single plan",
		[]*planpb.Plan{
			{
				Name:            "p1",
				CriticalActions: []string{},
				Actions:         make(map[string]*planpb.Action),
				AllowFail:       false,
			},
		},
		`{"plan_names":["p1"],"plans": {"p1":{"critical_actions":[], "actions":{}, "allow_fail":false}}}`,
	},
	{
		"full",
		[]*planpb.Plan{
			{
				Name:            "p1",
				CriticalActions: []string{"c1", "c2"},
				Actions: map[string]*planpb.Action{
					"c1": {
						Dependencies: []string{"c2"},
						ExecName:     "sample_pass",
					},
					"c2": {ExecName: "sample_fail"},
				},
				AllowFail: false,
			},
			{
				Name:            "p2",
				CriticalActions: []string{"c3", "c4"},
				Actions: map[string]*planpb.Action{
					"c3": {ExecName: "sample_pass"},
					"c4": {ExecName: "sample_fail"},
				},
				AllowFail: true,
			},
		},
		`{"plan_names":["p1","p2"],"plans": {"p1":{"critical_actions":["c1","c2"], "actions":{"c1":{"dependencies":["c2"],"exec_name":"sample_pass"},"c2":{"exec_name":"sample_fail"}}, "allow_fail":false},"p2":{"critical_actions":["c3","c4"], "actions":{"c3":{"exec_name":"sample_pass"},"c4":{"exec_name":"sample_fail"}}, "allow_fail":true}}}`,
	},
}

func TestCreateConfiguration(t *testing.T) {
	t.Parallel()
	for i, c := range generateCases {
		cs := c
		t.Run(cs.name, func(t *testing.T) {
			ctx := context.Background()
			got, err := createConfigurationJSON(cs.in)
			if err != nil {
				t.Fatalf("createConfigurationJSON(%d): %v", i, err)
			}

			if cs.want == "" {
				if string(got) != "" {
					t.Errorf("createConfigurationJSON(%d): %q, want empty", i, got)
				}
				// Remaining tests are not relevant on empty result.
				return
			}

			var gotProto, wantProto planpb.Plan
			// Convert both to protos and compare. This test now is more
			// useful to show this refactoring did not break anything.
			if err := json.Unmarshal(got, &gotProto); err != nil {
				t.Fatalf("Failed to unmarshal got bytes: %v", err)
			}
			if err := json.Unmarshal([]byte(cs.want), &wantProto); err != nil {
				t.Fatalf("Failed to unmarshal want bytes: %v", err)
			}

			if !proto.Equal(&wantProto, &gotProto) {
				t.Errorf("createConfiguration(%d): %v, want %v", i, got, cs.want)
			}
			p, err := loader.LoadConfiguration(ctx, bytes.NewReader(got))
			if err != nil {
				t.Errorf("loader.LoadConfiguration failed: %v", err)
			}
			if p == nil {
				t.Error("Default config is empty")
			}
		})
	}
}
