// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package config

import (
	"google.golang.org/protobuf/types/known/durationpb"
)

func servoRepairPlan() *Plan {
	return &Plan{
		CriticalActions: []string{
			"Set state:MISSING_CONFIG",
			"Servo is know in the setup",
			"Set state:WRONG_CONFIG",
			"Servod port specified",
			"Servo serial is specified",
			"Initialize docker container",
			"Set state:NO_SSH",
			"Device is pingable",
			"Device is SSHable",
			"servo_v3_uptime",
			"servo_power_cycle_root_servo",
			"Set state:SERVO_HOST_ISSUE",
			"lock_labstation",
			"Set state:BROKEN",
			"has_enough_disk_space",
			"Cache latest servod start time",
			"Set state:NOT_CONNECTED",
			"servo_v4_root_present",
			"Set state:NEED_REPLACEMENT",
			"servo_v3_root_present",
			"servo_fw_need_update",
			"Set state:SERVO_HOST_ISSUE",
			"servo_host_servod_start",
			"Set state:DUT_NOT_CONNECTED",
			"servo_dut_detected",
			"Set state:COLD_RESET_PIN_ISSUE",
			"servo_cold_reset_pin",
			"Set state:WARM_RESET_PIN_ISSUE",
			"servo_warm_reset_pin",
			"Set state:SERVOD_ISSUE",
			"servod_servo_pd",
			"servo_cr50_checks",
			"Set state:DUT_NOT_CONNECTED",
			"dut_controller_missing_fault_off",
			"Set state:TOPOLOGY_ISSUE",
			"servo_topology",
			"Set state:SERVO_HOST_ISSUE",
			"servod_get_serialname",
			"Set state:SERVOD_ISSUE",
			"servo_servod_echo_host",
			"Set state:SERVOD_PROXY_ISSUE",
			"Initialize DUT part for servo",
			"Set state:CR50_CONSOLE_MISSING",
			"servo_cr50_console",
			"Set state:CCD_TESTLAB_ISSUE",
			"cr50_testlab",
			"Set state:SERVOD_PROXY_ISSUE",
			"servo_ec_check",
			"Set state:BROKEN",
			"servo_detect_usbkey",
			"update_servo_type_label",
			"Set state:WORKING",
		},
		Actions: map[string]*Action{
			"Servo is know in the setup": {
				Docs: []string{
					"Verify if setup data has any data related to servo-host which mean servo is present in setup.",
				},
				ExecName: "dut_servo_host_present",
			},
			"Servo serial is specified": {
				Docs: []string{
					"Check if root servo serial is present.",
				},
				Conditions: []string{"is_not_servo_v3"},
				ExecName:   "dut_servo_has_serial",
			},
			"Device is pingable": {
				Docs: []string{
					"Verify that device is reachable by ping.",
					"Limited to 15 seconds.",
				},
				ExecName: "cros_ping",
				ExecTimeout: &durationpb.Duration{
					Seconds: 15,
				},
				RunControl: RunControl_ALWAYS_RUN,
			},
			"Device is SSHable": {
				Docs: []string{
					"Verify that device is reachable by SSH.",
					"Limited to 15 seconds.",
				},
				ExecTimeout: &durationpb.Duration{Seconds: 15},
				ExecName:    "cros_ssh",
				RunControl:  RunControl_ALWAYS_RUN,
			},
			"Cache latest servod start time": {
				Docs: []string{
					"Cache servod start time based on previous runs.",
					"If we fail all logs will be collected",
				},
				Conditions: []string{
					"is_not_servo_v3",
				},
				ExecName:               "cros_register_servod_logs_start",
				AllowFailAfterRecovery: true,
			},
			"servo_host_servod_start": {
				Conditions: []string{"is_not_container"},
				RecoveryActions: []string{
					"servo_host_servod_stop",
					"servo_power_delivery_repair",
					"servo_fake_disconnect_dut_repair",
					"servo_servod_cc_toggle_repair",
				},
				ExecName:    "servo_host_servod_init",
				ExecTimeout: &durationpb.Duration{Seconds: 120},
			},
			"servo_host_servod_stop": {
				Docs:       []string{"Stop the servod."},
				RunControl: RunControl_ALWAYS_RUN,
			},
			"Initialize docker container": {
				Docs: []string{
					"Initiate docker to have access to the host.",
					"TODO: Need close docker host, and add to cros plan.",
				},
				ExecTimeout: &durationpb.Duration{Seconds: 360},
				Conditions:  []string{"is_container"},
				RecoveryActions: []string{
					"servo_host_servod_stop",
					"servo_power_delivery_repair",
					"servo_fake_disconnect_dut_repair",
					"servo_servod_cc_toggle_repair",
				},
				ExecName: "servo_host_servod_init",
			},
			"Servod port specified": {
				Docs: []string{
					"Verify that servod port is present in servo data.",
					"Port is not expected to be specified for servo_V3.",
				},
				Conditions: []string{"is_not_servo_v3"},
				ExecName:   "servo_servod_port_present",
			},
			"servo_v3_uptime": {
				ExecName:        "cros_validate_uptime",
				ExecExtraArgs:   []string{"max_duration:96h"},
				Conditions:      []string{"is_servo_v3"},
				RecoveryActions: []string{"reboot"},
			},
			"reboot": {ExecName: "sample_pass"},
			"is_labstation": {
				Docs:     []string{"Condition to check if the servohost is a labstation."},
				ExecName: "servo_host_is_labstation",
			},
			"is_container": {
				Docs:     []string{"Condition to check if servo uses servod container."},
				ExecName: "servo_uses_servod_container",
			},
			"is_servo_v3": {
				Docs: []string{"Condition to check if the servo is v3."},
			},
			"lock_labstation": {
				Docs:       []string{"create lock file is_in_use"},
				Conditions: []string{"is_labstation"},
				ExecName:   "cros_create_servo_in_use",
			},
			"has_enough_disk_space": {
				Docs:          []string{"check the stateful partition have enough disk space that is at least 0.5GB. The storage unit is in GB."},
				Conditions:    []string{"is_not_container"},
				ExecName:      "cros_has_enough_storage_space",
				ExecExtraArgs: []string{"/mnt/stateful_partition:0.5"},
				RecoveryActions: []string{
					"servo_servod_and_labstation_disk_cleanup",
					"cros_create_reboot_request",
					"servo_host_v3_reboot",
				},
			},
			"servo_servod_and_labstation_disk_cleanup": {
				Docs: []string{"Clean up the old servod files as well as labstation."},
				Dependencies: []string{
					"servo_labstation_disk_cleanup",
					"servod_old_logs_cleanup",
				},
				ExecName: "sample_pass",
			},
			"servod_old_logs_cleanup": {
				Docs:          []string{"Clean up the old servod files based on the max number of days given in the argument."},
				ExecName:      "servo_servod_old_logs_cleanup",
				ExecExtraArgs: []string{"max_days:5"},
			},
			"is_not_container": {
				Conditions: []string{"is_container"},
				ExecName:   "sample_fail",
			},
			"servo_topology": {
				Docs:       []string{"host.check_diskspace('/mnt/stateful_partition', 0.5)"},
				Conditions: []string{"is_not_servo_v3"},
				Dependencies: []string{
					"cros_ssh",
					"servo_topology_single_child",
					"servo_topology_dual_setup",
				},
				ExecName: "sample_pass",
			},
			"servo_topology_single_child": {
				Conditions: []string{"is_not_servo_v3"},
				ExecName:   "servo_topology_update",
				ExecExtraArgs: []string{
					"min_child:1",
					"persist_topology:true",
				},
				RecoveryActions: []string{
					"servo_host_servod_stop",
					"servo_power_cycle_repair",
					"servo_power_delivery_repair",
					"servo_fake_disconnect_dut_repair",
					"servo_servod_cc_toggle_repair",
					"servo_reboot_ec_on_dut",
					"reboot_dut_by_power_state:reset",
					"cros_create_reboot_request",
					"reflash_cr_50_fw_on_dut",
					"reset_ec_on_dut",
				},
			},
			"servo_topology_dual_setup": {
				Conditions: []string{
					"is_not_servo_v3",
					"is_dual_setup",
				},
				ExecName: "servo_topology_update",
				ExecExtraArgs: []string{
					"min_child:2",
					"persist_topology:true",
				},
				RecoveryActions: []string{
					"servo_host_servod_stop",
					"servo_power_cycle_repair",
					"servo_power_delivery_repair",
					"servo_fake_disconnect_dut_repair",
					"servo_servod_cc_toggle_repair",
					"servo_reboot_ec_on_dut",
					"reboot_dut_by_power_state:reset",
					"cros_create_reboot_request",
					"reflash_cr_50_fw_on_dut",
					"reset_ec_on_dut",
				},
			},
			"servo_v3_root_present": {
				Docs:            []string{"This remains to be implemented."},
				Dependencies:    []string{"cros_ssh"},
				Conditions:      []string{"is_servo_v3"},
				RecoveryActions: []string{"servo_host_v3_reboot"},
				ExecName:        "servo_v3_root_present",
			},
			"servo_v4_root_present": {
				Dependencies:    []string{"cros_ssh"},
				Conditions:      []string{"is_not_servo_v3"},
				ExecExtraArgs:   []string{"update_topology:true"},
				RecoveryActions: []string{"cros_create_reboot_request"},
				ExecName:        "servo_v4_root_present",
			},
			"servo_fw_need_update": {
				Conditions:      []string{"is_not_servo_v3"},
				ExecTimeout:     &durationpb.Duration{Seconds: 300},
				RecoveryActions: []string{"servo_fw_update"},
			},
			"servod_get_serialname": {
				Docs:     []string{"run command from xmlrpc"},
				ExecName: "servod_echo",
				RecoveryActions: []string{
					"servo_host_servod_stop",
					"servo_power_cycle_repair",
					"servo_power_delivery_repair",
					"servo_fake_disconnect_dut_repair",
					"servo_servod_cc_toggle_repair",
					"servo_reboot_ec_on_dut",
					"reboot_dut_by_power_state:reset",
					"reflash_cr_50_fw_on_dut",
					"reset_ec_on_dut",
				},
			},
			"servo_get_ppdut5_mv": {
				ExecExtraArgs: []string{"command:ppdut5_mv"},
				ExecName:      "servo_check_servod_control",
			},
			"servo_get_ppchg5_mv": {
				ExecExtraArgs: []string{"command:ppchg5_mv"},
				ExecName:      "servo_check_servod_control",
			},
			"servod_servo_pd": {
				Docs:         []string{"run command from xmlrpc"},
				Conditions:   []string{"is_servo_v4_type_c"},
				Dependencies: []string{"servo_get_ppdut5_mv", "servo_get_ppchg5_mv"},
				ExecExtraArgs: []string{
					"command:servo_pd_role",
					"expected_string_value:src",
				},
				RecoveryActions: []string{
					"servo_power_delivery_repair",
					"servo_fake_disconnect_dut_repair",
					"servo_servod_cc_toggle_repair",
					"servo_reboot_ec_on_dut",
					"reboot_dut_by_power_state:reset",
					"cros_create_reboot_request",
					"reflash_cr_50_fw_on_dut",
				},
				ExecName:               "servo_check_servod_control",
				AllowFailAfterRecovery: true,
			},
			"servo_cr50_checks": {
				Conditions: []string{"is_not_servo_v3"},
				Dependencies: []string{
					"Set state:SBU_LOW_VOLTAGE",
					"servo_cr50_low_sbu",
					"Set state:DUT_NOT_CONNECTED",
					"servo_cr50_enumerated",
				},
				ExecName: "sample_pass",
			},
			"servo_cr50_low_sbu": {
				Conditions: []string{
					"is_not_servo_v3",
					"is_servo_v4_type_c",
					"servo_is_sbu_voltage_issue",
				},
				RecoveryActions: []string{
					"servo_host_servod_stop",
					"servo_power_cycle_repair",
					"servo_power_delivery_repair",
					"servo_fake_disconnect_dut_repair",
					"servo_servod_cc_toggle_repair",
					"servo_reboot_ec_on_dut",
					"reboot_dut_by_power_state:reset",
					"reflash_cr_50_fw_on_dut",
					"reset_ec_on_dut",
				},
			},
			"servo_is_sbu_voltage_issue": {
				ExecExtraArgs: []string{
					"command:dut_sbu_voltage_float_fault",
					"expected_string_value:on",
				},
				RecoveryActions: []string{
					"servo_power_cycle_repair",
					"servo_power_delivery_repair",
					"servo_fake_disconnect_dut_repair",
					"servo_servod_cc_toggle_repair",
					"servo_reboot_ec_on_dut",
					"reboot_dut_by_power_state:reset",
					"reflash_cr_50_fw_on_dut",
					"reset_ec_on_dut",
				},
				ExecName: "servo_check_servod_control",
			},
			"servo_cr50_enumerated": {
				Docs: []string{"prev name servo_cr50_off"},
				Conditions: []string{
					"is_not_servo_v3",
					"is_servo_v4_type_c",
					"servo_is_sbu_voltage_issue",
				},
				RecoveryActions: []string{
					"servo_host_servod_stop",
					"servo_power_cycle_repair",
				},
				ExecName: "sample_fail",
			},
			"Main device is GSC chip": {
				Dependencies: []string{
					"is_servo_v4",
					"is_servo_type_ccd",
				},
				ExecName: "servo_main_device_is_gcs",
			},
			"servo_cr50_console": {
				Docs: []string{"Create new action to check that servotype has ccd_cr50, and set that as a condition for this action."},
				Conditions: []string{
					"is_not_servo_v3",
					"Main device is GSC chip",
				},
				Dependencies: []string{
					"Initialize DUT part for servo",
				},
				ExecExtraArgs: []string{
					"commands:cr50_ccd_level,cr50_testlab,cr50_ccd_state_flags",
					"any_one:true",
				},
				RecoveryActions: []string{
					"servo_host_servod_stop",
					"servo_power_cycle_repair",
					"servo_power_delivery_repair",
					"servo_fake_disconnect_dut_repair",
					"servo_servod_cc_toggle_repair",
					"servo_reboot_ec_on_dut",
					"reboot_dut_by_power_state:reset",
					"reflash_cr_50_fw_on_dut",
					"reset_ec_on_dut",
				},
				ExecName: "servod_can_read_all",
			},
			"cr50_testlab": {
				Conditions: []string{
					"is_not_servo_v3",
					"Main device is GSC chip",
				},
				ExecExtraArgs: []string{
					"command:cr50_testlab",
					"expected_string_value:on",
				},
				RecoveryActions: []string{
					"Open gsc testlab",
					"servo_power_delivery_repair",
					"servo_fake_disconnect_dut_repair",
					"servo_servod_cc_toggle_repair",
					"servo_reboot_ec_on_dut",
					"reboot_dut_by_power_state:reset",
					"reflash_cr_50_fw_on_dut",
					"reset_ec_on_dut",
				},
				ExecName: "servo_check_servod_control",
			},
			"Open gsc testlab": {
				Docs: []string{
					"If servo uses cr50/gsc to control the DUT, open testlab will allowed to work (cr50_reboot, cold_reset, warm_reset)",
				},
				Conditions: []string{
					"is_not_servo_v3",
					"Main device is GSC chip",
				},
				ExecExtraArgs: []string{
					"command:cr50_testlab",
					"string_value:open",
				},
				ExecName:               "servo_set",
				AllowFailAfterRecovery: true,
				RunControl:             RunControl_ALWAYS_RUN,
			},
			"Initialize DUT part for servo": {
				Conditions: []string{
					"is_not_servo_v3",
				},
				Dependencies: []string{
					"servod_set_main_device",
					"Open gsc testlab",
				},
				RecoveryActions: []string{
					"servo_power_delivery_repair",
					"servo_fake_disconnect_dut_repair",
					"servo_servod_cc_toggle_repair",
					"servo_reboot_ec_on_dut",
					"reboot_dut_by_power_state:reset",
					"reflash_cr_50_fw_on_dut",
					"reset_ec_on_dut",
				},
				ExecName: "init_dut_for_servo",
			},
			"pwr_button_supported_models": {
				Docs: []string{"power button check is not applicable for these models"},
				ExecExtraArgs: []string{
					"string_values:arkham,gale,mistral,storm,whirlwind",
					"invert_result:true",
				},
				ExecName: "dut_check_model",
			},
			"servo_pwr_button_pin": {
				Conditions:   []string{"pwr_button_supported_models"},
				Dependencies: []string{"dut_has_model_name"},
				ExecExtraArgs: []string{
					"command:pwr_button",
					"expected_string_value:release",
				},
				RecoveryActions: []string{
					"servo_host_servod_stop",
					"servo_power_delivery_repair",
					"servo_fake_disconnect_dut_repair",
					"servo_servod_cc_toggle_repair",
				},
				ExecName:               "servo_check_servod_control",
				AllowFailAfterRecovery: true,
			},
			"servo_dut_detected": {
				Conditions: []string{
					"is_not_servo_v3",
					"is_servo_v4_type_a",
				},
				RecoveryActions: []string{
					"servo_host_servod_stop",
					"servo_power_delivery_repair",
					"servo_fake_disconnect_dut_repair",
					"servo_servod_cc_toggle_repair",
					"servo_reboot_ec_on_dut",
					"reboot_dut_by_power_state:reset",
					"reset_ec_on_dut",
					"servo_micro_fw_update_repair",
					"reflash_cr_50_fw_on_dut",
				},
				ExecName: "servo_low_ppdut5",
			},
			"Servo type-a hub connected": {
				Docs: []string{
					"Verifier to check connection Servo type-a to DUT.",
					"Working only for labstation with servo_micro.",
				},
				Conditions: []string{
					"servo_host_is_labstation",
					"is_servo_micro",
					"dut_has_cros_ec",
					// Followed is condition to check if voltage is low means servo_micro is not connected.
					"DUT is UP by EC response",
				},
				ExecName: "servo_low_ppdut5",
				RecoveryActions: []string{
					"servo_host_servod_stop",
					"servo_power_delivery_repair",
					"servo_servod_cc_toggle_repair",
					"servo_reboot_ec_on_dut",
					"reboot_dut_by_power_state:reset",
					"reset_ec_on_dut",
				},
			},
			"DUT is UP by EC response": {
				Docs: []string{
					"Check if DUT is up.",
					"Verification based on EC response.",
				},
				Conditions: []string{
					"dut_has_cros_ec",
				},
				Dependencies: []string{
					"servo_servod_echo_host",
				},
				ExecName: "servo_check_servod_control",
				ExecExtraArgs: []string{
					"command:ec_system_powerstate",
					"expected_string_value:S0",
				},
			},
			"servo_ec_check": {
				Conditions: []string{
					"is_not_servo_v3",
					"dut_has_cros_ec",
				},
				Dependencies: []string{
					"Set state:EC_BROKEN",
					"servo_ec_console",
					"Set state:BAD_RIBBON_CABLE",
					"servo_pwr_button_pin",
					"Set state:LID_OPEN_FAILED",
					"servo_lid_open",
					"servo_battery_charging",
				},
				ExecName: "sample_pass",
			},
			"dut_has_cros_ec": {
				Docs: []string{
					"Verify if DUT has ChromeOS firmware for EC",
				},
				Dependencies: []string{
					"servo_servod_echo_host",
				},
				ExecExtraArgs: []string{
					"command:supports_cros_ec_communication",
					"expected_string_value:yes",
				},
				ExecName: "servo_check_servod_control",
			},
			"servo_ec_console": {
				Conditions: []string{
					"is_not_servo_v3",
					"dut_has_cros_ec",
				},
				ExecExtraArgs: []string{
					"commands:ec_system_powerstate,ec_board",
					"any_one:true",
				},
				RecoveryActions: []string{
					"servo_reboot_ec_on_dut",
					"reboot_dut_by_power_state:reset",
					"reflash_cr_50_fw_on_dut",
					"reset_ec_on_dut",
				},
				ExecName: "servod_can_read_all",
			},
			"battery_last_charge_readable": {
				ExecExtraArgs: []string{
					"command:battery_full_charge_mah",
				},
				ExecName: "servo_check_servod_control",
			},
			"servo_battery_charging": {
				Conditions: []string{
					"is_not_servo_v3",
					"dut_has_cros_ec",
					"battery_last_charge_readable",
				},
				AllowFailAfterRecovery: true,
			},
			"servo_detect_usbkey": {
				Docs: []string{
					"Will detect the path to USB Drive on servo-host.",
					"Verify that usb-key is responsive",
				},
				ExecTimeout:            &durationpb.Duration{Seconds: 120},
				AllowFailAfterRecovery: true,
			},
			"servo_audit_usbkey": {
				Docs:                   []string{"This action will detect whether or not the USB drive is in working condition."},
				AllowFailAfterRecovery: true,
				Dependencies:           []string{"servo_detect_usbkey"},
				ExecTimeout:            &durationpb.Duration{Seconds: 7300},
			},
			"is_servo_v4": {
				Docs: []string{"This action will detect whether or not the attached servo device is of type V4."},
			},
			"is_servo_v4_type_c": {
				Docs:       []string{"This action will detect whether or not the attached servo V4 device is connect to DUT using Type-C connection."},
				Conditions: []string{"is_servo_v4"},
				ExecExtraArgs: []string{
					"command:root.dut_connection_type",
					"expected_string_value:type-c",
				},
				ExecName: "servo_check_servod_control",
			},
			"servo_lid_open": {
				RecoveryActions: []string{
					"servo_reboot_ec_on_dut",
					"reboot_dut_by_power_state:reset",
					"reflash_cr_50_fw_on_dut",
					"reset_ec_on_dut",
				},
				ExecName: "servod_lidopen",
			},
			"is_not_servo_v3": {
				Conditions: []string{"is_servo_v3"},
				ExecName:   "sample_fail",
			},
			"is_servo_v4_type_a": {
				Docs:       []string{"This action will detect whether or not the attached servo V4 device is connect to DUT using Type-A connection."},
				Conditions: []string{"is_servo_v4"},
				ExecExtraArgs: []string{
					"command:root.dut_connection_type",
					"expected_string_value:type-a",
				},
				ExecName: "servo_check_servod_control",
			},
			"is_dual_setup": {
				Docs:     []string{"Check whether the servo device has dual setup. This check only applies to the devices that have the dual setup configured on them."},
				ExecName: "is_dual_setup_configured",
			},
			"is_not_dual_setup": {
				Conditions: []string{"is_dual_setup"},
				ExecName:   "sample_fail",
			},
			"servod_set_main_device": {
				Conditions: []string{
					"servo_has_active_dut_controller",
				},
				ExecName: "servod_set_main_device",
				RecoveryActions: []string{
					"servo_host_servod_stop",
				},
			},
			"servo_fw_update": {
				Docs: []string{
					"Try to update in  normal ways 3 times",
					"if fail allow run force update",
				},
				Conditions: []string{"is_not_servo_v3"},
				Dependencies: []string{
					"Set state:SERVO_UPDATER_ISSUE",
					"servo_host_servod_stop",
				},
				ExecExtraArgs: []string{
					"try_attempt_count:3",
					"try_force_update_after_fail:true",
				},
				ExecTimeout: &durationpb.Duration{Seconds: 600},
				ExecName:    "servo_update_servo_firmware",
			},
			"servo_micro_fw_update_repair": {
				Docs: []string{
					"Try to update servo micro firmware",
				},
				Conditions: []string{
					"servo_host_is_labstation",
					"is_servo_micro",
					"is_time_to_update_servo_micro_fw",
				},
				Dependencies: []string{"servo_micro_fw_update"},
				ExecName:     "servo_host_servod_stop",
			},
			"servo_micro_fw_update": {
				Docs: []string{
					"Try to update servo micro firmware",
				},
				Conditions: []string{
					"servo_host_is_labstation",
					"is_servo_micro",
					"is_time_to_update_servo_micro_fw",
				},
				ExecExtraArgs: []string{
					"force_update:true",
					"ignore_version:true",
					"servo_board:servo_micro",
				},
				ExecTimeout: &durationpb.Duration{Seconds: 180},
				ExecName:    "servo_update_servo_firmware",
			},
			"is_time_to_update_servo_micro_fw": {
				Docs: []string{
					"Verify that it is time when we can try to re-flash fw on servo micro.",
					"Re-flashing limited to once per once per 2 weeks to avoid over-flashing the servo device.",
				},
				Conditions: []string{
					"last_time_servo_micro_fw_update_within_2_weeks",
				},
				ExecName: "sample_fail",
			},
			"last_time_servo_micro_fw_update_within_2_weeks": {
				Docs: []string{
					"Confirm that servo micro fw update action has occurred in the past 2 weeks. (336 hours)",
				},
				ExecExtraArgs: []string{
					"metrics_kind:servo_firmware_update_servo_micro",
					"time_frame_hours:336",
				},
				ExecName: "metrics_found_at_last_time",
			},
			"servo_warm_reset_supported": {
				ExecExtraArgs: []string{"command:warm_reset"},
				ExecName:      "servo_check_servod_control",
			},
			"servo_warm_reset_pin_for_servo_v3": {
				Conditions: []string{
					"is_servo_v3",
					"servo_warm_reset_supported",
				},
				ExecExtraArgs: []string{
					"command:warm_reset",
					"expected_string_value:off",
				},
				RecoveryActions: []string{
					"servo_host_servod_stop",
					"servo_power_delivery_repair",
					"servo_fake_disconnect_dut_repair",
					"servo_servod_cc_toggle_repair",
					"servo_reboot_ec_on_dut",
					"reboot_dut_by_power_state:reset",
					"reflash_cr_50_fw_on_dut",
					"reset_ec_on_dut",
				},
				ExecName: "servo_check_servod_control",
			},
			"servo_warm_reset_pin_for_servo_micro": {
				Conditions: []string{
					"is_servo_micro",
					"servo_warm_reset_supported",
				},
				ExecExtraArgs: []string{
					"command:warm_reset",
					"expected_string_value:off",
				},
				RecoveryActions: []string{
					"servo_host_servod_stop",
					"servo_power_delivery_repair",
					"servo_servod_cc_toggle_repair",
					"servo_reboot_ec_on_dut",
					"reboot_dut_by_power_state:reset",
					"reflash_cr_50_fw_on_dut",
					"reset_ec_on_dut",
				},
				ExecName: "servo_check_servod_control",
			},
			"servo_warm_reset_pin": {
				Docs: []string{"We need to check for warm reset only for servo micro and V3."},
				Dependencies: []string{
					"servo_warm_reset_pin_for_servo_v3",
					"servo_warm_reset_pin_for_servo_micro",
				},
				AllowFailAfterRecovery: true,
				ExecName:               "sample_pass",
			},
			"servo_cold_reset_pin": {
				Conditions: []string{
					"is_servo_v3",
					"is_servo_v4_type_a",
				},
				ExecExtraArgs: []string{
					"command:cold_reset",
					"expected_string_value:off",
				},
				RecoveryActions: []string{
					"servo_host_servod_stop",
					"servo_power_delivery_repair",
					"servo_fake_disconnect_dut_repair",
					"servo_servod_cc_toggle_repair",
					"servo_reboot_ec_on_dut",
					"reboot_dut_by_power_state:reset",
					"reflash_cr_50_fw_on_dut",
					"reset_ec_on_dut",
				},
				ExecName: "servo_check_servod_control",
			},
			"servo_servod_echo_host": {
				Docs:        []string{"Uses a servod control to check whether the servod daemon is responsive."},
				ExecTimeout: &durationpb.Duration{Seconds: 30},
				RecoveryActions: []string{
					"servo_host_servod_stop",
					"servo_power_delivery_repair",
					"servo_fake_disconnect_dut_repair",
					"servo_servod_cc_toggle_repair",
					"servo_reboot_ec_on_dut",
					"reboot_dut_by_power_state:reset",
					"reflash_cr_50_fw_on_dut",
					"reset_ec_on_dut",
				},
			},
			"update_servo_type_label": {
				Docs:     []string{"Update the servo type label for the DUT info."},
				ExecName: "servo_update_servo_type_label",
			},
			"dut_controller_missing_fault_off": {
				ExecExtraArgs: []string{"command:dut_controller_missing_fault",
					"expected_string_value:off",
				},
				Conditions: []string{"is_not_servo_v3"},
				RecoveryActions: []string{
					"servo_power_delivery_repair",
					"servo_fake_disconnect_dut_repair",
					"servo_servod_cc_toggle_repair",
					"servo_reboot_ec_on_dut",
					"reboot_dut_by_power_state:reset",
					"reflash_cr_50_fw_on_dut",
					"reset_ec_on_dut",
				},
				ExecName: "servo_check_servod_control",
			},
			"servo_has_active_dut_controller": {
				ExecExtraArgs: []string{"command:active_dut_controller"},
				RecoveryActions: []string{
					"servo_power_delivery_repair",
					"servo_fake_disconnect_dut_repair",
					"servo_servod_cc_toggle_repair",
					"servo_reboot_ec_on_dut",
					"reboot_dut_by_power_state:reset",
					"reflash_cr_50_fw_on_dut",
					"reset_ec_on_dut",
				},
				ExecName: "servo_check_servod_control",
			},
			"servod_restart_dut": {
				ExecName: "sample_pass",
			},
			"servo_power_cycle_repair": {
				Docs: []string{
					"Toggle the servod command servo_pd_role only once. And then stop the servod afterwards.",
					"TODO: Add dependency for servo initialize.",
				},
				Dependencies: []string{"servo_pd_toggle_once"},
				ExecTimeout:  &durationpb.Duration{Seconds: 120},
				RunControl:   RunControl_ALWAYS_RUN,
				ExecName:     "servo_host_servod_stop",
			},
			"servo_pd_toggle_once": {
				Docs: []string{
					"Toggle the servod command servo_pd_role only once.",
				},
				ExecExtraArgs: []string{
					"toggle_times:1",
					"wait_in_retry:5",
					"wait_before_retry:1",
				},
				RunControl: RunControl_ALWAYS_RUN,
				ExecName:   "servo_servod_toggle_pd_role",
			},
			"servo_power_delivery_repair": {
				Docs:         []string{"Toggle the servod command servo_pd_role 5 times. And then stop the servod afterwards. TODO: Add dependency for servo initialize."},
				Dependencies: []string{"servo_pd_toggle_five_times"},
				ExecTimeout:  &durationpb.Duration{Seconds: 600},
				RunControl:   RunControl_ALWAYS_RUN,
				ExecName:     "servo_host_servod_stop",
			},
			"servo_pd_toggle_five_times": {
				Docs: []string{"Toggle the servod command servo_pd_role 5 times."},
				ExecExtraArgs: []string{
					"toggle_times:5",
					"wait_in_retry:5",
					"wait_before_retry:1",
				},
				RunControl: RunControl_ALWAYS_RUN,
				ExecName:   "servo_servod_toggle_pd_role",
			},
			"Set state:MISSING_CONFIG": {
				ExecExtraArgs: []string{"state:MISSING_CONFIG"},
				ExecName:      "servo_set_servo_state",
				RunControl:    RunControl_ALWAYS_RUN,
			},
			"Set state:WRONG_CONFIG": {
				ExecExtraArgs: []string{"state:WRONG_CONFIG"},
				ExecName:      "servo_set_servo_state",
				RunControl:    RunControl_ALWAYS_RUN,
			},
			"Set state:NO_SSH": {
				ExecExtraArgs: []string{"state:NO_SSH"},
				ExecName:      "servo_set_servo_state",
				RunControl:    RunControl_ALWAYS_RUN,
			},
			"Set state:BROKEN": {
				ExecExtraArgs: []string{"state:BROKEN"},
				ExecName:      "servo_set_servo_state",
				RunControl:    RunControl_ALWAYS_RUN,
			},
			"Set state:NOT_CONNECTED": {
				ExecExtraArgs: []string{"state:NOT_CONNECTED"},
				ExecName:      "servo_set_servo_state",
				RunControl:    RunControl_ALWAYS_RUN,
			},
			"Set state:NEED_REPLACEMENT": {
				ExecExtraArgs: []string{"state:NEED_REPLACEMENT"},
				ExecName:      "servo_set_servo_state",
				RunControl:    RunControl_ALWAYS_RUN,
			},
			"Set state:TOPOLOGY_ISSUE": {
				ExecExtraArgs: []string{"state:TOPOLOGY_ISSUE"},
				ExecName:      "servo_set_servo_state",
				RunControl:    RunControl_ALWAYS_RUN,
			},
			"Set state:SERVO_UPDATER_ISSUE": {
				ExecExtraArgs: []string{"state:SERVO_UPDATER_ISSUE"},
				ExecName:      "servo_set_servo_state",
				RunControl:    RunControl_ALWAYS_RUN,
			},
			"Set state:SERVOD_ISSUE": {
				ExecExtraArgs: []string{"state:SERVOD_ISSUE"},
				ExecName:      "servo_set_servo_state",
				RunControl:    RunControl_ALWAYS_RUN,
			},
			"Set state:SERVO_HOST_ISSUE": {
				ExecExtraArgs: []string{"state:SERVO_HOST_ISSUE"},
				ExecName:      "servo_set_servo_state",
				RunControl:    RunControl_ALWAYS_RUN,
			},
			"Set state:SERVOD_PROXY_ISSUE": {
				ExecExtraArgs: []string{"state:SERVOD_PROXY_ISSUE"},
				ExecName:      "servo_set_servo_state",
				RunControl:    RunControl_ALWAYS_RUN,
			},
			"Set state:COLD_RESET_PIN_ISSUE": {
				ExecExtraArgs: []string{"state:COLD_RESET_PIN_ISSUE"},
				ExecName:      "servo_set_servo_state",
				RunControl:    RunControl_ALWAYS_RUN,
			},
			"Set state:WARM_RESET_PIN_ISSUE": {
				ExecExtraArgs: []string{"state:WARM_RESET_PIN_ISSUE"},
				ExecName:      "servo_set_servo_state",
				RunControl:    RunControl_ALWAYS_RUN,
			},
			"Set state:SBU_LOW_VOLTAGE": {
				ExecExtraArgs: []string{"state:SBU_LOW_VOLTAGE"},
				ExecName:      "servo_set_servo_state",
				RunControl:    RunControl_ALWAYS_RUN,
			},
			"Set state:DUT_NOT_CONNECTED": {
				ExecExtraArgs: []string{"state:DUT_NOT_CONNECTED"},
				ExecName:      "servo_set_servo_state",
				RunControl:    RunControl_ALWAYS_RUN,
			},
			"Set state:CR50_CONSOLE_MISSING": {
				ExecExtraArgs: []string{"state:CR50_CONSOLE_MISSING"},
				ExecName:      "servo_set_servo_state",
				RunControl:    RunControl_ALWAYS_RUN,
			},
			"Set state:EC_BROKEN": {
				ExecExtraArgs: []string{"state:EC_BROKEN"},
				ExecName:      "servo_set_servo_state",
				RunControl:    RunControl_ALWAYS_RUN,
			},
			"Set state:BAD_RIBBON_CABLE": {
				ExecExtraArgs: []string{"state:BAD_RIBBON_CABLE"},
				ExecName:      "servo_set_servo_state",
				RunControl:    RunControl_ALWAYS_RUN,
			},
			"Set state:LID_OPEN_FAILED": {
				ExecExtraArgs: []string{"state:LID_OPEN_FAILED"},
				ExecName:      "servo_set_servo_state",
				RunControl:    RunControl_ALWAYS_RUN,
			},
			"Set state:CCD_TESTLAB_ISSUE": {
				ExecExtraArgs: []string{"state:CCD_TESTLAB_ISSUE"},
				ExecName:      "servo_set_servo_state",
				RunControl:    RunControl_ALWAYS_RUN,
			},
			"Set state:WORKING": {
				ExecExtraArgs: []string{"state:WORKING"},
				ExecName:      "servo_set_servo_state",
				RunControl:    RunControl_ALWAYS_RUN,
			},
			"servo_fake_disconnect_dut_repair": {
				Docs:         []string{"Try to repair servod by mimic reconnection of servo."},
				Dependencies: []string{"servo_fake_disconnect_dut"},
				ExecTimeout:  &durationpb.Duration{Seconds: 600},
				RunControl:   RunControl_ALWAYS_RUN,
				ExecName:     "servo_host_servod_stop",
			},
			"servo_fake_disconnect_dut": {
				Conditions: []string{"is_servo_type_ccd"},
				ExecExtraArgs: []string{
					"delay_in_ms:100",
					"timeout_in_ms:2000",
				},
			},
			"servo_servod_cc_toggle_repair": {
				Docs:         []string{"Try to repair servod by toggling cc."},
				Dependencies: []string{"servo_servod_cc_toggle"},
				ExecTimeout:  &durationpb.Duration{Seconds: 600},
				RunControl:   RunControl_ALWAYS_RUN,
				ExecName:     "servo_host_servod_stop",
			},
			"servo_servod_cc_toggle": {
				Conditions: []string{"is_servo_type_ccd"},
				ExecExtraArgs: []string{
					"cc_off_timeout:10",
					"cc_on_timeout:30",
				},
				RunControl: RunControl_ALWAYS_RUN,
			},
			"servo_reboot_ec_on_dut": {
				Docs: []string{
					"Try to reboot EC on DUT using servod command.",
					"It reboots just the embedded controllers on the DUT.",
				},
				ExecExtraArgs: []string{
					"wait_timeout:1",
					"value:reboot",
				},
				ExecName:   "servo_set_ec_uart_cmd",
				RunControl: RunControl_ALWAYS_RUN,
			},
			"reboot_dut_by_power_state:reset": {
				Docs:          []string{"Try to reboot DUT by resetting power state command on servod."},
				ExecExtraArgs: []string{"wait_timeout:1"},
				RunControl:    RunControl_ALWAYS_RUN,
				ExecName:      "servo_power_state_reset",
			},
			"cros_create_reboot_request": {
				Docs:       []string{"Try to create reboot flag file request."},
				Conditions: []string{"cros_ssh"},
				RunControl: RunControl_ALWAYS_RUN,
			},
			"reflash_cr_50_fw_on_dut": {
				Docs: []string{"Try to reflash cr50 firmware and reboot AP from DUT side to wake it up."},
				Conditions: []string{
					"is_servo_type_ccd",
					"cros_is_time_to_reflash_cr50_fw",
				},
				Dependencies: []string{
					"cros_reflash_cr50_fw",
				},
				RunControl: RunControl_ALWAYS_RUN,
				ExecName:   "servo_host_servod_stop",
			},
			"cros_reflash_cr50_fw": {
				Docs: []string{
					"Try to reflash cr50 firmware and reboot AP from DUT side to wake it up.",
					"Reboot after the fw flash is successful.",
				},
				Dependencies: []string{"cros_ssh_dut"},
				ExecName:     "cros_reflash_cr50_fw",
				ExecExtraArgs: []string{
					"flash_timeout:120",
					"wait_timeout:30",
				},
				ExecTimeout: &durationpb.Duration{Seconds: 150},
			},
			"cros_is_time_to_reflash_cr50_fw": {
				Docs: []string{
					"Verify that it is time when we can try to re-flash fw on cr50 (H1).",
					"Re-flashing limited to once per once per day to avoid over-flashing the device.",
				},
				Conditions: []string{
					"cros_last_time_cr50_reflash_within_24hr",
				},
				ExecName: "sample_fail",
			},
			"cros_last_time_cr50_reflash_within_24hr": {
				Docs: []string{
					"Confirm that no cr50 reflash action has occurred in the past 24 hours.",
				},
				ExecExtraArgs: []string{
					"metrics_kind:cr50_flash",
					"time_frame_hours:24",
				},
				ExecName: "metrics_found_at_last_time",
			},
			"reset_ec_on_dut": {
				Docs:         []string{"Try to reset EC from DUT side to wake CR50 up. And then restart the servod."},
				Conditions:   []string{"is_servo_type_ccd"},
				Dependencies: []string{"cros_reset_ec"},
				RunControl:   RunControl_ALWAYS_RUN,
				ExecName:     "servo_host_servod_stop",
			},
			"cros_reset_ec": {
				Docs:          []string{"Try to wake up the device as it will trigger recovering ec, cr50, and other fw."},
				Dependencies:  []string{"cros_ssh_dut"},
				ExecExtraArgs: []string{"wait_timeout:30"},
				RunControl:    RunControl_ALWAYS_RUN,
			},
			"servo_power_cycle_root_servo": {
				Docs:       []string{"Try to reset(power-cycle) the servo via smart usbhub."},
				Conditions: []string{"servo_host_is_labstation"},
				ExecExtraArgs: []string{
					"reset_timeout:60",
					"wait_timeout:20",
				},
				ExecTimeout:            &durationpb.Duration{Seconds: 120},
				RunControl:             RunControl_RUN_ONCE,
				AllowFailAfterRecovery: true,
			},
			"servo_host_v3_reboot": {
				Docs:          []string{"Try to reboot servo host v3."},
				Conditions:    []string{"is_servo_v3"},
				ExecTimeout:   &durationpb.Duration{Seconds: 300},
				ExecExtraArgs: []string{"reboot_timeout:10"},
				RunControl:    RunControl_RUN_ONCE,
			},
		},
	}
}
