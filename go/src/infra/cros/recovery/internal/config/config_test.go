package config

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"infra/cros/recovery/internal/plan"
)

var loadCases = []struct {
	name string
	got  string
	exp  *plan.Plan
}{
	{
		"simple",
		`{"simple":{}}`,
		&plan.Plan{
			Name:      "simple",
			AllowFail: false,
			Verifiers: nil,
		},
	},
	{
		"create_default_action_if_not_provided",
		`{
			"create_default_action_if_not_provided": {
				"verifiers": [
				  "a1"
				],
				"allow_fail": true
			  }
		}`,
		&plan.Plan{
			Name:      "create_default_action_if_not_provided",
			AllowFail: true,
			Verifiers: []*plan.Action{
				{
					Name:         "a1",
					ExecName:     "a1",
					Dependencies: nil,
					Recoveries:   nil,
					AllowFail:    false,
					AllowCache:   true,
				},
			},
		},
	},
	{
		"simple_action",
		`{
			"simple_action": {
				"verifiers": [
				  "a1"
				],
				"actions": {
				  "a1": {}
				},
				"allow_fail": true
			  }
		}`,
		&plan.Plan{
			Name:      "simple_action",
			AllowFail: true,
			Verifiers: []*plan.Action{
				{
					Name:         "a1",
					ExecName:     "a1",
					Dependencies: nil,
					Recoveries:   nil,
					AllowFail:    false,
					AllowCache:   true,
				},
			},
		},
	},
	{
		"full",
		`{
			"full": {
				"verifiers": [
				  "a1-full"
				],
				"actions": {
					"a1-full": {
						"exec_name": "a1",
						"allow_fail": true,
						"allow_cache": "never",
						"dependencies": ["d1"],
						"recoveries":["r2"]
					},
					"d1": {
						"exec_name": "d1-exec",
						"dependencies": ["d2"],
						"recoveries":["r1"]
				    },
					"d2": {
						"exec_name": "d2-exec",
						"allow_fail": true
				    },
					"r2": {
						"exec_name": "r2-exec",
						"dependencies": ["d2"]
				    }
				},
				"allow_fail": true
			  }
		}`,
		&plan.Plan{
			Name:      "full",
			AllowFail: true,
			Verifiers: []*plan.Action{
				{
					Name:     "a1-full",
					ExecName: "a1",
					Dependencies: []*plan.Action{
						{
							Name:     "d1",
							ExecName: "d1-exec",
							Dependencies: []*plan.Action{
								{
									Name:       "d2",
									ExecName:   "d2-exec",
									AllowFail:  true,
									AllowCache: true,
								},
							},
							Recoveries: []*plan.Action{
								{
									Name:       "r1",
									ExecName:   "r1",
									AllowCache: true,
								},
							},
							AllowCache: true,
						},
					},
					Recoveries: []*plan.Action{
						{
							Name:     "r2",
							ExecName: "r2-exec",
							Dependencies: []*plan.Action{
								{
									Name:       "d2",
									ExecName:   "d2-exec",
									AllowFail:  true,
									AllowCache: true,
								},
							},
							AllowCache: true,
						},
					},
					AllowFail:  true,
					AllowCache: false,
				},
			},
		},
	},
}

func TestLoadPlans(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	for _, c := range loadCases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			cr := strings.NewReader(c.got)
			got, err := LoadPlans(ctx, []string{c.name}, cr)
			if err != nil {
				t.Errorf("unexpected fail: %s", err)
			} else if len(got) != 1 {
				t.Errorf("expected only 1 plan, got: %d", len(got))
			}
			if diff := cmp.Diff(got[0].Describe(), c.exp.Describe()); diff != "" {
				t.Errorf("Receive diff: %v \nGot:\n %s", diff, got[0].Describe())
			}
		})
	}
}

// Verify that default plan is correct json structure.
func TestDeafultPlans(t *testing.T) {
	t.Parallel()
	var m map[string]interface{}
	data := []byte(defaultPlans)
	if err := json.Unmarshal(data, &m); err != nil {
		t.Errorf("Expected json string. error: %s", err)
	}
	if len(m) != 3 {
		t.Errorf("Expected to have 3 default plans. got: %d", len(m))
	}
}
