package loader

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/types/known/durationpb"

	"infra/cros/recovery/config"
)

var testCases = []struct {
	name string
	got  string
	exp  *config.Configuration
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
		&config.Configuration{
			Plans: map[string]*config.Plan{
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
		&config.Configuration{
			Plans: map[string]*config.Plan{
				"full": {
					AllowFail: true,
					CriticalActions: []string{
						"a1-full",
						"missing_critical_action",
					},
					Actions: map[string]*config.Action{
						"a1-full": {
							ExecName:               "a1",
							Conditions:             []string{"c1", "c2"},
							Dependencies:           []string{"d1"},
							RecoveryActions:        []string{"r2"},
							AllowFailAfterRecovery: true,
							RunControl:             config.RunControl_RUN_ONCE,
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
							RunControl:   config.RunControl_ALWAYS_RUN,
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
	for _, c := range testCases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			// t.Parallel() Test cannot be parallel because it modifies a global variable.
			oldExecsExist := execsExist
			execsExist = func(string) bool {
				return true
			}
			defer func() {
				execsExist = oldExecsExist
			}()
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

var cycleTestCases = []struct {
	testName     string
	in           *config.Plan
	errorActions []string
}{
	{
		"A_dependency -> A",
		&config.Plan{
			Actions: map[string]*config.Action{
				"A": {Dependencies: []string{"A"}},
			},
		},
		[]string{"A"},
	},
	{
		"A_dependency -> B_condition -> A",
		&config.Plan{
			Actions: map[string]*config.Action{
				"A": {Dependencies: []string{"B"}},
				"B": {Conditions: []string{"A"}},
			},
		},
		[]string{"A", "B"},
	},
	{
		"A_dependency -> B_condition -> C_recovery -> A",
		&config.Plan{
			Actions: map[string]*config.Action{
				"A": {Dependencies: []string{"B"}},
				"B": {Conditions: []string{"C"}},
				"C": {RecoveryActions: []string{"A"}},
			},
		},
		[]string{"A", "B", "C"},
	},
	{
		"A_dependency -> B_dependency -> C_dependency -> A",
		&config.Plan{
			Actions: map[string]*config.Action{
				"A": {Dependencies: []string{"B"}},
				"B": {Dependencies: []string{"C"}},
				"C": {Dependencies: []string{"A"}},
			},
		},
		[]string{"A", "B", "C"},
	},
	{
		"C_dependency -> B_condition -> A_recovery -> C",
		&config.Plan{
			Actions: map[string]*config.Action{
				"A": {RecoveryActions: []string{"C"}},
				"B": {Conditions: []string{"A"}},
				"C": {Dependencies: []string{"B"}},
			},
		},
		[]string{"A", "B", "C"},
	},
	{
		"A_dependency -> B_condition -> C_recovery -> D_recovery -> E_dependency -> F_condition -> B",
		&config.Plan{
			Actions: map[string]*config.Action{
				"A": {Dependencies: []string{"B"}},
				"B": {Conditions: []string{"C"}},
				"C": {RecoveryActions: []string{"D"}},
				"D": {RecoveryActions: []string{"E"}},
				"E": {Dependencies: []string{"F"}},
				"F": {Conditions: []string{"B"}},
			},
		},
		[]string{"B", "C", "D", "E", "F"},
	},
	{
		"A_dependency -> B_condition -> C_recovery -> D_recovery -> E_dependency -> F_condition -> B",
		&config.Plan{
			Actions: map[string]*config.Action{
				"A": {Dependencies: []string{"B"}},
				"B": {Conditions: []string{"C"}},
				"C": {RecoveryActions: []string{"D"}},
				"D": {RecoveryActions: []string{"E"}},
				"E": {Dependencies: []string{"F"}},
				"F": {Conditions: []string{"B"}},
			},
		},
		[]string{},
	},
	{
		"A_dependency -> B_condition -> C_recovery -> D_recovery -> E_dependency -> F; A_dependency -> E_dependency -> F; C_condition -> F",
		&config.Plan{
			Actions: map[string]*config.Action{
				"A": {Dependencies: []string{"B", "E"}},
				"B": {Conditions: []string{"C"}},
				"C": {RecoveryActions: []string{"D", "F"}},
				"D": {RecoveryActions: []string{"E"}},
				"E": {Dependencies: []string{"F"}},
				"F": {},
			},
		},
		[]string{},
	},
	// Test Case: Cycle in actions, but not reachable by critical actions.
	{
		"A_dependency -> B_condition -> C_recovery; D_recovery -> E_dependency -> F_recovery -> D",
		&config.Plan{
			Actions: map[string]*config.Action{
				"A": {Dependencies: []string{"B"}},
				"B": {Conditions: []string{"C"}},
				"C": {},
				"D": {RecoveryActions: []string{"E"}},
				"E": {Dependencies: []string{"F"}},
				"F": {RecoveryActions: []string{"D"}},
			},
		},
		[]string{},
	},
}

func TestVerifyPlanAcyclic(t *testing.T) {
	t.Parallel()
	for _, tt := range cycleTestCases {
		tt := tt
		t.Run(tt.testName, func(t *testing.T) {
			t.Parallel()
			// Assume "A" as the critical action.
			tt.in.CriticalActions = []string{"A"}
			if err := verifyPlanAcyclic(tt.in); err != nil {
				errMessage := err.Error()
				raiseError := false
				for _, eachAction := range tt.errorActions {
					if !strings.Contains(errMessage, eachAction) {
						raiseError = true
					}
				}
				if raiseError {
					t.Errorf("got %q, want %q", errMessage, tt.errorActions)
				}
			} else if len(tt.errorActions) != 0 {
				t.Errorf("got nil, want %q", tt.errorActions)
			}
		})
	}
}

var createMissingActionsCases = []struct {
	name      string
	inPlan    *config.Plan
	inActions []string
	outPlan   *config.Plan
}{
	{
		"init Actions and set action if missed in actions map",
		&config.Plan{
			Actions: nil,
		},
		[]string{"a"},
		&config.Plan{
			Actions: map[string]*config.Action{
				"a": {},
			},
		},
	},
	{
		"do not replace if action is present in the plan",
		&config.Plan{
			Actions: map[string]*config.Action{
				"a": {Dependencies: []string{"F"}},
			},
		},
		[]string{"a"},
		&config.Plan{
			Actions: map[string]*config.Action{
				"a": {Dependencies: []string{"F"}},
			},
		},
	},
}

func TestCreateMissingActions(t *testing.T) {
	t.Parallel()
	for _, tt := range createMissingActionsCases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			createMissingActions(tt.inPlan, tt.inActions)
			if !reflect.DeepEqual(tt.inPlan, tt.outPlan) {
				t.Errorf("case %q: did not updated in expecteds struct", tt.name)
			}
		})
	}
}
