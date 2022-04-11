// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package config

import (
	"google.golang.org/protobuf/types/known/durationpb"
)

// LabstationRepairConfig provides config for repair labstation task.
func LabstationRepairConfig() *Configuration {
	return &Configuration{
		PlanNames: []string{PlanCrOS},
		Plans: map[string]*Plan{
			PlanCrOS: {
				AllowFail: false,
				CriticalActions: []string{
					"dut_state_repair_failed",
					"check_host_info",
					"cros_ping",
					"cros_ssh",
					"cros_is_on_stable_version",
					"Update provisioned info",
					"booted_from_right_kernel",
					"reboot_by_request",
					"dut_state_ready",
				},
				Actions: map[string]*Action{
					"dut_state_repair_failed": {
						RunControl: RunControl_RUN_ONCE,
					},
					"check_host_info": {
						Docs:     []string{"Check basic info for deployment."},
						ExecName: "sample_pass",
						Dependencies: []string{
							"dut_has_name",
							"dut_has_board_name",
							"dut_has_model_name",
						},
					},
					"cros_is_on_stable_version": {
						Conditions: []string{
							"has_stable_version_cros_image",
							"cros_kernel_priority_has_not_changed",
							"not_exempted_pool",
						},
						RecoveryActions: []string{
							"install_stable_os",
						},
					},
					"install_stable_os": {
						Docs: []string{"Install stable OS on the device."},
						Conditions: []string{
							"has_stable_version_cros_image",
							"cros_kernel_priority_has_not_changed",
						},
						ExecName: "cros_provision",
						ExecExtraArgs: []string{
							"no_reboot",
						},
						ExecTimeout: &durationpb.Duration{Seconds: 3600},
					},
					"not_exempted_pool": {
						Docs: []string{
							"There are some labstations we don't want they receive auto-update, e.g. labstations that used for image qualification purpose",
						},
						ExecName: "dut_not_in_pool",
						ExecExtraArgs: []string{
							"servo_verification",
							"labstation_tryjob",
							"labstation_canary",
							"labstation_phone_station",
						},
					},
					"Update provisioned info": {
						Docs: []string{
							"Read and update cros-provision label.",
						},
						ExecName: "cros_update_provision_os_version",
					},
					"labstation_langid_check": {
						Docs: []string{
							"This part is not ready.",
							"The action and will validate present of lang_id issue",
						},
						ExecName:               "sample_pass",
						AllowFailAfterRecovery: true,
					},
					"cros_stop_powerd": {
						ExecName: "cros_run_shell_command",
						ExecExtraArgs: []string{
							"stop",
							"powerd",
						},
						AllowFailAfterRecovery: true,
					},
					"cros_clean_tmp_owner_request": {
						Docs: []string{
							"In some cases, the update flow puts the TPM into a state such that it fails verification.",
							"We don't know why. However, this call papers over the problem by clearing the TPM during the reboot.",
							"We ignore failures from 'crossystem'.",
							"Although failure here is unexpected, and could signal a bug, the point of the exercise is to paper over problems.",
						},
						AllowFailAfterRecovery: true,
					},
					"labstation_uptime_6_hours": {
						ExecName: "cros_validate_uptime",
						ExecExtraArgs: []string{
							"min_duration:6h",
						},
					},
					"remove_reboot_requests": {
						ExecName:               "cros_remove_all_reboot_request",
						AllowFailAfterRecovery: true,
					},
					"reboot_by_request": {
						Docs: []string{
							"Some DUTs can request reboot labstation if they has issue with servo-nic or other issues with servo-host.",
						},
						ExecName: "cros_remove_all_reboot_request",
						Conditions: []string{
							"cros_has_reboot_request",
							"cros_has_no_servo_in_use",
							"labstation_uptime_6_hours",
						},
						Dependencies: []string{
							"labstation_reboot",
							"rpm_power_cycle",
						},
					},
					"booted_from_right_kernel": {
						Docs: []string{
							"Verified if kernel has update and waiting for update.",
							"Kernel can wait for reboot as provisioning is not doing reboot by default for labstations.",
						},
						ExecName: "cros_kernel_priority_has_not_changed",
						Conditions: []string{
							"remove_reboot_requests",
							"cros_has_no_servo_in_use",
						},
						RecoveryActions: []string{
							"labstation_reboot",
							"rpm_power_cycle",
						},
					},
					"cros_ping": {
						Docs: []string{
							"This verifier checks whether a given host is reachable over ping. ",
							"This should happen as soon as the network driver gets loaded and the network becomes operational.",
						},
						RecoveryActions: []string{
							"rpm_power_cycle",
						},
						ExecTimeout: &durationpb.Duration{Seconds: 300},
					},
					"cros_ssh": {
						Docs: []string{
							"This verifier checks whether the host is accessible over ssh.",
						},
						RecoveryActions: []string{
							"rpm_power_cycle",
						},
						ExecTimeout: &durationpb.Duration{Seconds: 120},
					},
					"labstation_reboot": {
						Docs: []string{
							"Before reboot we need stop powerd service. Assumption if trigger power the labstation will not boot in normal mode.",
						},
						Dependencies: []string{
							"cros_stop_powerd",
							"cros_clean_tmp_owner_request",
							"cros_allowed_reboot",
							"remove_reboot_requests",
						},
						ExecName:    "cros_reboot",
						ExecTimeout: &durationpb.Duration{Seconds: 120},
						RunControl:  RunControl_ALWAYS_RUN,
					},
					"rpm_power_cycle": {
						Docs: []string{
							"Action is always runnable.",
						},
						Conditions: []string{
							"has_rpm_info",
						},
						RunControl: RunControl_ALWAYS_RUN,
					},
				},
			},
		},
	}
}

// LabstationDeployConfig provides config for deploy labstation task.
func LabstationDeployConfig() *Configuration {
	return &Configuration{
		PlanNames: []string{PlanCrOS},
		Plans: map[string]*Plan{
			PlanCrOS: {
				AllowFail: false,
				CriticalActions: []string{
					"dut_state_needs_deploy",
					"check_host_info",
					"cros_ping",
					"cros_ssh",
					"update_inv_info",
					"install_stable_os",
					"remove_reboot_requests",
					"Update provisioned info",
					"validate_rpm",
					"dut_state_ready",
				},
				Actions: map[string]*Action{
					"check_host_info": {
						Docs: []string{
							"Check basic info for deployment.",
						},
						ExecName: "sample_pass",
						Dependencies: []string{
							"dut_has_name",
							"dut_has_board_name",
							"dut_has_model_name",
						},
					},
					"update_inv_info": {
						Docs: []string{
							"Updating device info in inventory.",
						},
						ExecName: "sample_pass",
						Dependencies: []string{
							"cros_ssh",
							"cros_update_hwid_to_inventory",
							"cros_update_serial_number_inventory",
						},
					},
					"install_stable_os": {
						Docs: []string{
							"Install stable OS on the device.",
							"Labstation will be rebooted to make it ready for use.",
						},
						ExecName: "cros_provision",
						Conditions: []string{
							"has_stable_version_cros_image",
							"cros_not_on_stable_version",
						},
						ExecTimeout: &durationpb.Duration{
							Seconds: 3600,
						},
					},
					"remove_reboot_requests": {
						Docs: []string{
							"Remove reboot request flag files.",
						},
						ExecName:               "cros_remove_all_reboot_request",
						AllowFailAfterRecovery: true,
					},
					"Update provisioned info": {
						Docs: []string{
							"Read and update cros-provision label.",
						},
						ExecName: "cros_update_provision_os_version",
					},
					"validate_rpm": {
						Docs: []string{
							"Validate and update rpm_state.",
							"The execs is not ready yet.",
						},
						ExecName: "rpm_audit_without_battery",
						ExecTimeout: &durationpb.Duration{
							Seconds: 600,
						},
						Conditions: []string{
							"has_rpm_info",
						},
					},
				},
			},
		},
	}
}
