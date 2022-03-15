// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package recovery

import (
	"github.com/golang/protobuf/proto"

	"infra/cros/recovery/config"
)

//copyPlay returns a deep copy of plan and sets allowFail on the copy.
func copyPlan(plan *config.Plan, allowFail bool) *config.Plan {
	p := proto.Clone(plan).(*config.Plan)
	p.AllowFail = allowFail
	return p
}

// CrosRepairConfig provides config for repair cros setup in the lab task.
func CrosRepairConfig() *config.Configuration {
	return &config.Configuration{
		PlanNames: []string{
			config.PlanServo,
			config.PlanCrOS,
			config.PlanChameleon,
			config.PlanBluetoothPeer,
			config.PlanWifiRouter,
			config.PlanClosing,
		},
		Plans: map[string]*config.Plan{
			config.PlanServo:         copyPlan(servoRepairPlan, true),
			config.PlanCrOS:          copyPlan(crosRepairPlan, false),
			config.PlanChameleon:     copyPlan(chameleonPlan, true),
			config.PlanBluetoothPeer: copyPlan(btpeerRepairPlan, true),
			config.PlanWifiRouter:    copyPlan(wifiRouterRepairPlan, true),
			config.PlanClosing:       copyPlan(crosClosePlan, true),
		}}
}

// CrosDeployConfig provides config for deploy cros setup in the lab task.
func CrosDeployConfig() *config.Configuration {
	return &config.Configuration{
		PlanNames: []string{
			config.PlanServo,
			config.PlanCrOS,
			config.PlanChameleon,
			config.PlanBluetoothPeer,
			config.PlanWifiRouter,
			config.PlanClosing,
		},
		Plans: map[string]*config.Plan{
			config.PlanServo:         copyPlan(servoRepairPlan, false),
			config.PlanCrOS:          copyPlan(crosDeployPlan, false),
			config.PlanChameleon:     copyPlan(chameleonPlan, true),
			config.PlanBluetoothPeer: copyPlan(btpeerRepairPlan, true),
			config.PlanWifiRouter:    copyPlan(wifiRouterRepairPlan, true),
			config.PlanClosing:       copyPlan(crosClosePlan, true),
		},
	}
}

// crosClosePlan provides plan to close cros repair/deploy tasks.
var crosClosePlan = &config.Plan{
	CriticalActions: []string{
		"Remove in-use flag on servo-host",
		"Remove request to reboot is servo is good",
	},
	Actions: map[string]*config.Action{
		"servo_state_is_working": {
			Docs:          []string{"check the servo's state is ServoStateWorking."},
			ExecName:      "servo_match_state",
			ExecExtraArgs: []string{"state:WORKING"},
		},
		"Remove request to reboot is servo is good": {
			Conditions: []string{
				"is_not_flex_board",
				"servo_state_is_working",
			},
			ExecName:               "cros_remove_reboot_request",
			AllowFailAfterRecovery: true,
		},
		"Remove in-use flag on servo-host": {
			Conditions:             []string{"is_not_flex_board"},
			ExecName:               "cros_remove_servo_in_use",
			AllowFailAfterRecovery: true,
		},
		"is_not_flex_board": {
			Docs: []string{"Verify that device is belong Reven models"},
			ExecExtraArgs: []string{
				"string_values:x1c",
				"invert_result:true",
			},
			ExecName: "dut_check_model",
		},
	},
}
