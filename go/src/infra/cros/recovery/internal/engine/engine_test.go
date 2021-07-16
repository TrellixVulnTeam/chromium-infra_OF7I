package engine

import (
	"context"
	"testing"

	"infra/cros/recovery/internal/planpb"
)

// TODO(otabek@) Add cases with verification the cache.

// Predefined exec functions.
const (
	exec_pass = "sample_pass"
	exec_fail = "sample_fail"
)

var planTestCases = []struct {
	name       string
	got        *planpb.Plan
	expSuccess bool
}{
	{
		"simple",
		&planpb.Plan{},
		true,
	},
	{
		"critical action fail",
		&planpb.Plan{
			CriticalActions: []string{
				"a1",
				"a2",
			},
			Actions: map[string]*planpb.Action{
				"a1": {
					ExecName: exec_pass,
				},
				"a2": {
					ExecName: exec_fail,
				},
			},
		},
		false,
	},
	{
		"allowed critical action fail",
		&planpb.Plan{
			AllowFail: true,
			CriticalActions: []string{
				"a1",
				"a2",
			},
			Actions: map[string]*planpb.Action{
				"a1": {
					ExecName: exec_pass,
				},
				"a2": {
					ExecName: exec_fail,
				},
			},
		},
		true,
	},
	{
		"skip fail action as not applicable",
		&planpb.Plan{
			CriticalActions: []string{
				"a1",
			},
			Actions: map[string]*planpb.Action{
				"a1": {
					ExecName:   exec_fail,
					Conditions: []string{"c1"},
				},
				"c1": {
					ExecName: exec_fail,
				},
			},
		},
		true,
	},
	{
		"skip fail dependency as not applicable",
		&planpb.Plan{
			CriticalActions: []string{
				"a1",
			},
			Actions: map[string]*planpb.Action{
				"a1": {
					ExecName:     exec_pass,
					Dependencies: []string{"d1"},
				},
				"d1": {
					ExecName:   exec_fail,
					Conditions: []string{"c1"},
				},
				"c1": {
					ExecName: exec_fail,
				},
			},
		},
		true,
	},
	{
		"fail action by dependencies",
		&planpb.Plan{
			CriticalActions: []string{
				"a1",
			},
			Actions: map[string]*planpb.Action{
				"a1": {
					ExecName:     exec_pass,
					Dependencies: []string{"d1"},
				},
				"d1": {
					ExecName: exec_fail,
				},
			},
		},
		false,
	},
	{
		"success run",
		&planpb.Plan{
			CriticalActions: []string{
				"a1",
			},
			Actions: map[string]*planpb.Action{
				"a1": {
					ExecName:     exec_pass,
					Conditions:   []string{"c1"},
					Dependencies: []string{"d1"},
				},
				"c1": {
					ExecName:     exec_pass,
					Dependencies: []string{"d2"},
				},
				"d1": {
					ExecName:     exec_pass,
					Dependencies: []string{"d2"},
				},
				"d2": {
					ExecName: exec_pass,
				},
			},
		},
		true,
	},
}

func TestRun(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	for _, c := range planTestCases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			err := Run(ctx, c.name, c.got, nil)
			if c.expSuccess {
				if err != nil {
					t.Errorf("Case %q fail but expected to pass. Received error: %s", c.name, err)
				}
			} else {
				if err == nil {
					t.Errorf("Case %q expected to fail but pass", c.name)
				}
			}
		})
	}
}

var recoveryTestCases = []struct {
	name         string
	got          map[string]*planpb.Action
	expStartOver bool
}{
	{
		"no recoveries",
		map[string]*planpb.Action{
			"a": {
				RecoveryActions: nil,
			},
		},
		false,
	},
	{
		"recoveries stopped on passed r2 and create request to start over",
		map[string]*planpb.Action{
			"a": {
				RecoveryActions: []string{"r1", "r2", "r3"},
			},
			"r1": {
				ExecName: exec_fail,
			},
			"r2": {
				ExecName: exec_pass,
			},
			"r3": {}, //Should not reached
		},
		true,
	},
	{
		"recoveries fail but the process still pass",
		map[string]*planpb.Action{
			"a": {
				RecoveryActions: []string{"r1", "r2", "r3"},
			},
			"r1": {
				ExecName: exec_fail,
			},
			"r2": {
				ExecName: exec_fail,
			},
			"r3": {
				ExecName: exec_fail,
			},
		},
		false,
	},
}

func TestRunRecovery(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	for _, c := range recoveryTestCases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			r := recoveryEngine{
				plan: &planpb.Plan{
					Actions: c.got,
				},
			}
			err := r.runRecoveries(ctx, "a")
			if c.expStartOver {
				if !startOverTag.In(err) {
					t.Errorf("Case %q expected to get request to start over. Received: %s", c.name, err)
				}
			} else {
				if err != nil {
					t.Errorf("Case %q expected to receive nil. Received error: %s", c.name, err)
				}
			}
		})
	}
}

var runExecTestCases = []struct {
	name           string
	canUseRecovery bool
	got            map[string]*planpb.Action
	expError       bool
	expStartOver   bool
}{
	{
		"do not start recovery flow if action passed",
		true,
		map[string]*planpb.Action{
			"a": {
				ExecName: exec_pass,
				// Will fail if reached any recovery actions.
				RecoveryActions: []string{"r11"},
			},
		},
		false,
		false,
	},
	{
		"do not start recovery flow if it is not allowed",
		false,
		map[string]*planpb.Action{
			"a": {
				ExecName: exec_fail,
				// Will fail if reached any recovery actions.
				RecoveryActions: []string{"r21"},
			},
		},
		true,
		false,
	},
	{
		"receive start over request after run successful recovery action",
		true,
		map[string]*planpb.Action{
			"a": {
				ExecName:        exec_fail,
				RecoveryActions: []string{"r31"},
			},
			"r31": {
				ExecName: exec_pass,
			},
		},
		true,
		true,
	},
	{
		"receive error after try recovery action",
		true,
		map[string]*planpb.Action{
			"a": {
				ExecName:        exec_fail,
				RecoveryActions: []string{"r41"},
			},
			"r41": {
				ExecName: exec_fail,
			},
		},
		true,
		false,
	},
}

func TestActionExec(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	for _, c := range runExecTestCases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			r := recoveryEngine{
				plan: &planpb.Plan{
					Actions: c.got,
				},
			}
			err := r.runActionExec(ctx, "a", c.canUseRecovery)
			if c.expError && c.expStartOver {
				if !startOverTag.In(err) {
					t.Errorf("Case %q expected to get request to start over. Received error: %s", c.name, err)
				}
			} else if c.expError {
				if err == nil {
					t.Errorf("Case %q expected to fail by passed", c.name)
				}
			} else {
				if err != nil {
					t.Errorf("Case %q expected to receive nil. Received error: %s", c.name, err)
				}
			}
		})
	}
}
