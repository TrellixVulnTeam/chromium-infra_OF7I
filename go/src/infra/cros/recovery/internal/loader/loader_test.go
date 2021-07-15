package loader

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/types/known/durationpb"

	"infra/cros/recovery/internal/planpb"
)

var testCases = []struct {
	name string
	got  string
	exp  *planpb.Configuration
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
		&planpb.Configuration{
			Plans: map[string]*planpb.Plan{
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
					  "a1-full"
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
		&planpb.Configuration{
			Plans: map[string]*planpb.Plan{
				"full": {
					AllowFail: true,
					CriticalActions: []string{
						"a1-full",
					},
					Actions: map[string]*planpb.Action{
						"a1-full": {
							ExecName:               "a1",
							Conditions:             []string{"c1", "c2"},
							Dependencies:           []string{"d1"},
							ExecTimeout:            nil,
							ExecExtraArgs:          []string{},
							RecoveryActions:        []string{"r2"},
							AllowFailAfterRecovery: true,
							RunControl:             planpb.RunControl_RUN_ONCE,
						},
						"d1": {
							ExecName:               "d1-exec",
							Conditions:             []string{},
							Dependencies:           []string{"d2"},
							ExecTimeout:            nil,
							ExecExtraArgs:          []string{},
							RecoveryActions:        []string{"r1"},
							AllowFailAfterRecovery: false,
							RunControl:             planpb.RunControl_RERUN_AFTER_RECOVERY,
						},
						"d2": {
							ExecName:               "d2-exec",
							Conditions:             []string{},
							Dependencies:           []string{},
							ExecTimeout:            nil,
							ExecExtraArgs:          []string{},
							RecoveryActions:        []string{},
							AllowFailAfterRecovery: true,
							RunControl:             planpb.RunControl_RERUN_AFTER_RECOVERY,
						},
						"r2": {
							ExecName:               "r2-exec",
							Conditions:             []string{},
							Dependencies:           []string{"d2"},
							ExecTimeout:            &durationpb.Duration{Seconds: 1000},
							ExecExtraArgs:          []string{},
							RecoveryActions:        []string{},
							AllowFailAfterRecovery: false,
							RunControl:             planpb.RunControl_ALWAYS_RUN,
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
	for _, c := range testCases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			cr := strings.NewReader(c.got)
			config, err := LoadConfiguration(ctx, cr)
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
