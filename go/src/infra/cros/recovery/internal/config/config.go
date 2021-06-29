// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package config provides struts to holds and read configs.
package config

import (
	"infra/cros/recovery/internal/plan"
)

// Please ignore errors in the CL.

// TODO(otabek@): Implement logic to load config from file.
// TODO(otabek@): Implement default plans.
// TODO(otabek@): Implement verification of actions loop.

// LoadPlans loads plans.
// Note: Only for local testing. Correct logic will be derived by next CL.
func LoadPlans(planNames []string) ([]*plan.Plan, error) {
	var plans []*plan.Plan
	for _, name := range planNames {
		plans = append(plans, createPlan(name))
	}
	return plans, nil
}

// Only for local testing.
// Will be removed in the next CL.
func createPlan(name string) *plan.Plan {
	actions := make(map[string]*plan.Action)
	actions["servo_host_ping"] = &plan.Action{
		Name:       "servo_host_ping",
		ExecName:   "servo_host_ping",
		AllowCache: true,
	}
	actions["servo_host_ssh"] = &plan.Action{
		Name:     "servo_host_ssh",
		ExecName: "servo_host_ssh",
		Dependencies: []*plan.Action{
			actions["servo_host_ping"],
		},
		AllowCache: true,
	}
	actions["always_pass"] = &plan.Action{
		Name:       "always_pass",
		ExecName:   "sample_pass",
		AllowCache: true,
	}
	actions["always_fail"] = &plan.Action{
		Name:     "always_fail",
		ExecName: "sample_fail",
		Dependencies: []*plan.Action{
			actions["always_pass"],
		},
		AllowFail:  true,
		AllowCache: true,
	}
	actions["always_pass2"] = &plan.Action{
		Name:     "always_pass2",
		ExecName: "sample_pass",
		Dependencies: []*plan.Action{
			actions["always_pass"],
			actions["always_fail"],
		},
		AllowCache: true,
	}
	actions["dut_ping"] = &plan.Action{
		Name:       "dut_ping",
		ExecName:   "dut_ping",
		AllowCache: false,
	}
	actions["dut_ssh"] = &plan.Action{
		Name:     "dut_ssh",
		ExecName: "dut_ssh",
		Dependencies: []*plan.Action{
			actions["dut_ping"],
		},
		AllowCache: true,
	}
	return &plan.Plan{
		Name:      name,
		AllowFail: false,
		Verifiers: []*plan.Action{
			actions["always_pass2"],
			actions["dut_ping"],
			actions["dut_ping"],
			actions["dut_ssh"],
			actions["always_fail"],
		},
	}
}
