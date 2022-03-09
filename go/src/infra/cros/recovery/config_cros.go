// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package recovery

import (
	"infra/cros/recovery/internal/planpb"
	"io"

	"github.com/golang/protobuf/proto"
)

//copyPlay returns a deep copy of plan and sets allowFail on the copy.
func copyPlan(plan *planpb.Plan, allowFail bool) *planpb.Plan {
	p := proto.Clone(plan).(*planpb.Plan)
	p.AllowFail = allowFail
	return p
}

// CrosRepairConfig provides config for repair cros setup in the lab task.
func CrosRepairConfig() io.Reader {
	c := &planpb.Configuration{
		PlanNames: []string{
			PlanServo,
			PlanCrOS,
			PlanChameleon,
			PlanBluetoothPeer,
			PlanWifiRouter,
			PlanClosing,
		},
		Plans: map[string]*planpb.Plan{
			PlanServo:         copyPlan(servoRepairPlan, true),
			PlanCrOS:          copyPlan(crosRepairPlan, false),
			PlanChameleon:     copyPlan(chameleonPlan, true),
			PlanBluetoothPeer: copyPlan(btpeerRepairPlan, true),
			PlanWifiRouter:    copyPlan(wifiRouterRepairPlan, true),
			PlanClosing:       copyPlan(crosClosePlan, true),
		}}
	return mustCreateConfigJSON(c)
}

// CrosDeployConfig provides config for deploy cros setup in the lab task.
func CrosDeployConfig() io.Reader {
	c := &planpb.Configuration{
		PlanNames: []string{
			PlanServo,
			PlanCrOS,
			PlanChameleon,
			PlanBluetoothPeer,
			PlanWifiRouter,
			PlanClosing,
		},
		Plans: map[string]*planpb.Plan{
			PlanServo:         copyPlan(servoRepairPlan, false),
			PlanCrOS:          copyPlan(crosDeployPlan, false),
			PlanChameleon:     copyPlan(chameleonPlan, true),
			PlanBluetoothPeer: copyPlan(btpeerRepairPlan, true),
			PlanWifiRouter:    copyPlan(wifiRouterRepairPlan, true),
			PlanClosing:       copyPlan(crosClosePlan, true),
		},
	}
	return mustCreateConfigJSON(c)
}

// crosClosePlan provides plan to close cros repair/deploy tasks.
var crosClosePlan = &planpb.Plan{
	CriticalActions: []string{
		"Remove in-use flag on servo-host",
		"Remove request to reboot is servo is good",
	},
	Actions: map[string]*planpb.Action{
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
