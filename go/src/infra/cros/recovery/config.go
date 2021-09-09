// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package recovery

import (
	"io"
	"strings"
)

// Default cobfiguration with all planes supported by engine.
// WHen you change or add new plan please be sure that is working
// and predictable.
// TODO(otabek@): Add plan for labstation.
// TODO(vkjoshi@): Add plans for Servo and DUT.
const defaultConfig = `
{
	"plans":{
		"labstation_repair":{
			"critical_actions": [
				"dut_state_repair_failed",
				"has_dut_name",
				"has_dut_board_name",
				"has_dut_model_name",
				"cros_ssh",
				"cros_match_stable_os_version_to_device",
				"update_provision_info",
				"booted_from_right_kernel",
				"reboot_by_request",
				"dut_state_ready"
			],
			"actions": {
				"cros_match_stable_os_version_to_device":{
					"conditions": [
						"has_stable_version_cros_image"
					],
					"recovery_actions":[
						"cros_provision_no_reboot"
					]
				},
				"cros_provision_no_reboot":{
					"docs":[
						"This part is not ready and waiting to deliver cros-provision CLI"
					],
					"conditions": [
						"has_stable_version_cros_image"
					],
					"exec_name": "sample_fail"
				},
				"update_provision_info":{
					"exec_name": "cros_update_provision_os_version"
				},
				"labstation_langid_check":{
					"docs":[
						"This part is not ready.",
						"The action and will validate present of lang_id issue"
					],
					"exec_name": "sample_pass",
					"allow_fail_after_recovery": true
				},
				"cros_stop_powerd":{
					"exec_name":"cros_run_shell_command",
					"exec_extra_args":[
						"stop",
						"powerd"
					],
					"allow_fail_after_recovery": true
				},
				"cros_clean_tmp_owner_request":{
					"docs":[
						"In some cases, the update flow puts the TPM into a state such that it fails verification.",
						"We don't know why. However, this call papers over the problem by clearing the TPM during the reboot.",
						"We ignore failures from 'crossystem'.",
						"Although failure here is unexpected, and could signal a bug, the point of the exercise is to paper over problems."
					],
					"allow_fail_after_recovery": true
				},
				"labstation_uptime_6_hours":{
					"exec_name":"cros_validate_uptime",
					"exec_extra_args":[
						"min_duration:6h"
					]
				},
				"remove_reboot_requests":{
					"exec_name":"cros_remove_reboot_request",
					"allow_fail_after_recovery": true
				},
				"reboot_by_request":{
					"docs":[
						"Some DUTs can request reboot labstation if they has issue with servo-nic or other issues with servo-host."
					],
					"exec_name": "cros_remove_reboot_request",
					"conditions": [
						"cros_has_reboot_request",
						"cros_has_no_servo_in_use",
						"labstation_uptime_6_hours"
					],
					"dependencies":[
						"labstation_reboot",
						"rpm_power_cycle"
					]
				},
				"booted_from_right_kernel":{
					"docs":[
						"Verified if kernel has update and waiting for update.",
						"Kernel can wait for reboot as provisioning is not doing reboot by default for labstations."
					],
					"exec_name": "cros_has_kernel_priority_change",
					"conditions": [
						"remove_reboot_requests",
						"cros_has_no_servo_in_use"
					],
					"recovery_actions":[
						"labstation_reboot",
						"rpm_power_cycle"
					]
				},
				"cros_ssh":{
					"recovery_actions":[
						"rpm_power_cycle"
					]
				},
				"labstation_reboot":{
					"docs":[
						"Before reboot we need stop powerd service. Assumption if trigger power the labstation will not boot in normal mode."
					],
					"dependencies":[
						"cros_stop_powerd",
						"cros_clean_tmp_owner_request",
						"cros_has_noreboot_request",
						"remove_reboot_requests"
					],
					"exec_name": "cros_reboot",
					"exec_timeout": {
						"seconds": 120
					},
					"run_control": 1
				},
				"rpm_power_cycle":{
					"docs":[
						"Action is always runnable."
					],
					"conditions": [
						"has_rpm_info"
					],
					"run_control": 1
				}
			}
		},
		"servo_repair":{
			"critical_actions": [
				"cros_ssh",
				"servo_host_servod_restart",
				"servod_echo",
				"servo_detect_usbkey"
			],
			"actions": {
				"servo_host_servod_restart":{
					"exec_timeout": {
						"seconds": 120
					}
				},
				"servo_detect_usbkey": {
					"docs": ["Will detect the path to USB Drive on servo-host."],
					"allow_fail_after_recovery": true
				}
			},
			"allow_fail": true
		},
		"chameleon_repair":{
			"critical_actions": [
				"sample_pass"
			],
			"actions": {},
			"allow_fail": true
		},
		"bluetooth_peer_repair":{
			"critical_actions": [
				"sample_pass"
			],
			"actions": {},
			"allow_fail": true
		},
		"cros_repair":{
			"critical_actions": [
				"sample_pass"
			],
			"actions": {}
		},
		"labstation_deploy":{
			"critical_actions": [
				"sample_pass"
			],
			"actions": {}
		},
		"cros_deploy":{
			"critical_actions": [
				"sample_pass"
			],
			"actions": {}
		}
	}
}
 `

// DefaultConfig provides default config for recovery engine.
func DefaultConfig() io.Reader {
	return strings.NewReader(defaultConfig)
}
