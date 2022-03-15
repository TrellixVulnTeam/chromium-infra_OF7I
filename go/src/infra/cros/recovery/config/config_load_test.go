// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package config

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/types/known/durationpb"
)

var testLoadCases = []struct {
	name string
	got  string
	exp  *Configuration
}{
	{
		"simple",
		`{
			"plans": {
				"plan1": {},
				"plan2": {
					"allow_fail": true
				}
			}
		}`,
		&Configuration{
			Plans: map[string]*Plan{
				"plan1": {
					AllowFail:       false,
					CriticalActions: nil,
					Actions:         nil,
				},
				"plan2": {
					AllowFail:       true,
					CriticalActions: nil,
					Actions:         nil,
				},
			},
		},
	},
	{
		"full",
		`{
			"plans":{
				"full": {
					"critical_actions": [
					  "a1-full",
					  "missing_critical_action"
					],
					"actions": {
						"a1-full": {
							"exec_name": "a1",
							"allow_fail_after_recovery": true,
							"run_control": 2,
							"conditions": ["c1", "c2"],
							"dependencies": ["d1"],
							"recovery_actions":["r2"]
						},
						"d1": {
							"exec_name": "d1-exec",
							"dependencies": ["d2"],
							"recovery_actions":["r1"]
						},
						"d2": {
							"exec_name": "d2-exec",
							"allow_fail_after_recovery": true,
							"exec_extra_args":[]
						},
						"r2": {
							"exec_name": "r2-exec",
							"exec_timeout": {
								"seconds": 1000
							},
							"dependencies": ["d2"],
							"run_control":1
						}
					},
					"allow_fail": true
				}
			}
		}`,
		&Configuration{
			Plans: map[string]*Plan{
				"full": {
					AllowFail: true,
					CriticalActions: []string{
						"a1-full",
						"missing_critical_action",
					},
					Actions: map[string]*Action{
						"a1-full": {
							ExecName:               "a1",
							Conditions:             []string{"c1", "c2"},
							Dependencies:           []string{"d1"},
							RecoveryActions:        []string{"r2"},
							AllowFailAfterRecovery: true,
							RunControl:             RunControl_RUN_ONCE,
						},
						"c1": {
							ExecName: "c1",
						},
						"c2": {
							ExecName: "c2",
						},
						"d1": {
							ExecName:        "d1-exec",
							Dependencies:    []string{"d2"},
							ExecTimeout:     nil,
							RecoveryActions: []string{"r1"},
						},
						"d2": {
							ExecName:               "d2-exec",
							AllowFailAfterRecovery: true,
						},
						"r1": {
							ExecName: "r1",
						},
						"r2": {
							ExecName:     "r2-exec",
							Dependencies: []string{"d2"},
							ExecTimeout:  &durationpb.Duration{Seconds: 1000},
							RunControl:   RunControl_ALWAYS_RUN,
						},
						"missing_critical_action": {
							ExecName: "missing_critical_action",
						},
					},
				},
			},
		},
	},
}

func TestLoadConfiguration(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	for _, c := range testLoadCases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			execsExist := func(string) bool {
				return true
			}
			cr := strings.NewReader(c.got)
			config, err := Load(ctx, cr, execsExist)
			if err != nil {
				t.Errorf("unmarshal fail: %s", err)
			}
			loadedJson, _ := json.MarshalIndent(config, "", "\t")
			expectedJson, _ := json.MarshalIndent(c.exp, "", "\t")
			if diff := cmp.Diff(string(loadedJson), string(expectedJson)); diff != "" {
				t.Errorf("Receive diff: %v \ngot:\n %s", diff, loadedJson)
			}
		})
	}
}
