// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package config

// setAllowFail updates allowFail property and return plan.
func setAllowFail(p *Plan, allowFail bool) *Plan {
	p.AllowFail = allowFail
	return p
}

// CrosRepairConfig provides config for repair cros setup in the lab task.
func CrosRepairConfig() *Configuration {
	return &Configuration{
		PlanNames: []string{
			PlanServo,
			PlanCrOS,
			PlanChameleon,
			PlanBluetoothPeer,
			PlanWifiRouter,
			PlanClosing,
		},
		Plans: map[string]*Plan{
			PlanServo:         setAllowFail(servoRepairPlan(), true),
			PlanCrOS:          setAllowFail(crosRepairPlan(), false),
			PlanChameleon:     setAllowFail(chameleonPlan(), true),
			PlanBluetoothPeer: setAllowFail(btpeerRepairPlan(), true),
			PlanWifiRouter:    setAllowFail(wifiRouterRepairPlan(), true),
			PlanClosing:       setAllowFail(crosClosePlan(), true),
		}}
}

// CrosDeployConfig provides config for deploy cros setup in the lab task.
func CrosDeployConfig() *Configuration {
	return &Configuration{
		PlanNames: []string{
			PlanServo,
			PlanCrOS,
			PlanChameleon,
			PlanBluetoothPeer,
			PlanWifiRouter,
			PlanClosing,
		},
		Plans: map[string]*Plan{
			PlanServo:         setAllowFail(servoRepairPlan(), false),
			PlanCrOS:          setAllowFail(crosDeployPlan(), false),
			PlanChameleon:     setAllowFail(chameleonPlan(), true),
			PlanBluetoothPeer: setAllowFail(btpeerRepairPlan(), true),
			PlanWifiRouter:    setAllowFail(wifiRouterRepairPlan(), true),
			PlanClosing:       setAllowFail(crosClosePlan(), true),
		},
	}
}

// crosClosePlan provides plan to close cros repair/deploy tasks.
func crosClosePlan() *Plan {
	return &Plan{
		CriticalActions: []string{
			"Update peripheral wifi state",
			"Servo-host logs",
			"Remove in-use flag on servo-host",
			"Remove request to reboot is servo is good",
			"Update DUT state",
		},
		Actions: map[string]*Action{
			"servo_state_is_working": {
				Docs: []string{
					"check the servo's state is ServoStateWorking.",
				},
				ExecName:      "servo_match_state",
				ExecExtraArgs: []string{"state:WORKING"},
			},
			"Remove request to reboot is servo is good": {
				Conditions: []string{
					"is_not_flex_board",
					"dut_servo_host_present",
					"servo_state_is_working",
				},
				ExecName:               "cros_remove_reboot_request",
				AllowFailAfterRecovery: true,
			},
			"Remove in-use flag on servo-host": {
				Conditions: []string{
					"is_not_flex_board",
					"dut_servo_host_present",
				},
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
			"Servo-host logs": {
				Dependencies: []string{
					"Try copy messages from servo-host",
					"Try to collect servod logs",
				},
				ExecName: "sample_pass",
			},
			"Try to collect servod logs": {
				Docs: []string{
					"Try to collect all servod logs since latest start time.",
				},
				Conditions: []string{
					"dut_servo_host_present",
					"is_not_servo_v3",
				},
				ExecName:               "cros_collect_servod_logs",
				AllowFailAfterRecovery: true,
			},
			"Try copy messages from servo-host": {
				Docs: []string{
					"Try to collect /var/log/messages from servo-host.",
				},
				Conditions: []string{
					"dut_servo_host_present",
					"is_not_servo_v3",
				},
				ExecName: "cros_copy_file_to_log",
				ExecExtraArgs: []string{
					"filepath:/var/log/messages",
					"use_host_dir:true",
				},
				AllowFailAfterRecovery: true,
			},
			"is_not_servo_v3": {
				Conditions: []string{"is_servo_v3"},
				ExecName:   "sample_fail",
			},
			"Update peripheral wifi state": {
				Docs: []string{
					"Update peripheral wifi state based on wifi router states",
				},
				Conditions: []string{
					"wifi_router_host_present",
				},
				ExecName:               "update_peripheral_wifi_state",
				AllowFailAfterRecovery: true,
			},
			"Check failure count": {
				Docs: []string{
					"Check if the number of times the recovery task ",
					"has failed is greater than a threshold value or ",
					"not.",
				},
				ExecName: "metrics_check_task_failures",
				ExecExtraArgs: []string{
					"task_name:recovery",
					"repair_failed_count:49",
				},
			},
			"Update DUT state": {
				Docs: []string{
					"Set the DUT state to the value passed in the ",
					"extra args.",
				},
				Conditions: []string{
					"Check failure count",
				},
				ExecName: "dut_set_state",
				ExecExtraArgs: []string{
					"state:needs_manual_repair",
				},
				AllowFailAfterRecovery: true,
			},
		},
	}
}
