// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package config

import (
	"google.golang.org/protobuf/types/known/durationpb"
)

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
					"Update inventory info",
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
					"Update inventory info": {
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
