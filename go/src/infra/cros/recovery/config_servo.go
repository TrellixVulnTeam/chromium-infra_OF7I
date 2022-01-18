// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package recovery

const servoRepairPlanBody = `
"critical_actions": [
	"servo_host_info",
	"servod_info",
	"init_docker_host",
	"cros_ssh",
	"servo_v3_uptime",
	"lock_labstation",
	"has_enough_disk_space",
	"servo_root_check",
	"servo_topology",
	"servo_fw_need_update",
	"servo_host_servod_start",
	"servo_servod_echo_host",
	"servod_get_serialname",
	"servo_connection_pins",
	"servo_dut_detected",
	"servod_servo_pd",
	"servo_cr50_checks",
	"dut_controller_missing_fault_off",
	"servo_cr50_console",
	"servo_ec_check",
	"servod_set_main_device",
	"init_dut_for_servo",
	"servo_testlab_enabled",
	"servo_detect_usbkey",
	"update_servo_type_label"
],
"actions": {
	"servo_host_servod_start": {
		"exec_timeout": {
			"seconds": 120
		},
		"exec_name": "servo_host_servod_init"
	},
	"servo_host_servod_init": {
		"exec_timeout": {
			"seconds": 120
		},
		"recovery_actions": [
			"servo_host_servod_restart"
		]
	},
	"servo_host_info": {
		"exec_name":"dut_has_name"
	},
	"servod_info": {
		"docs" : ["Verify that servo port is available, and servo serial is readable (only for non-servo-v3)."],
		"exec_name":"sample_pass"
	},
	"init_docker_host": {
		"docs": [
			"Only to create docker. Need close docker host, and add to cros plan."
		],
		"exec_name":"sample_pass"
	},
	"servo_v3_uptime": {
		"exec_name":"cros_validate_uptime",
		"exec_extra_args": [
			"max_duration:96h"
		],
		"conditions": [
			"is_servo_v3"
		],
		"recovery_actions": [
			"reboot"
		]
	},
	"reboot": {
		"exec_name":"sample_pass"
	},
	"is_labstation": {
		"docs": [
			"Condition to check if the servohost is a labstation."
		],
		"exec_name":"servo_host_is_labstation"
	},
	"is_container": {
		"docs": [
			"Condition to check if servo uses servod container."
		],
		"exec_name":"servo_uses_servod_container"
	},
	"is_servo_v3": {
		"docs": [
			"Condition to check if the servo is v3."
		],
		"exec_name":"servo_is_v3"
	},
	"lock_labstation": {
		"docs": [
			"create lock file is_in_use"
		],
		"conditions": [
			"is_labstation"
		],
		"exec_name":"cros_create_servo_in_use"
	},
	"has_enough_disk_space": {
		"docs":[
			"check the stateful partition have enough disk space that is at least 0.5GB. The storage unit is in GB."
		],
		"conditions": [
			"is_not_container"
		],
		"exec_name":"cros_has_enough_storage_space",
		"exec_extra_args":[
			"/mnt/stateful_partition:0.5"
		],
		"recovery_actions": [
			"servo_servod_and_labstation_disk_cleanup"
		]
	},
	"servo_servod_and_labstation_disk_cleanup":{
		"docs":[
			"Clean up the old servod files as well as labstation."
		],
		"dependencies": [
			"servo_labstation_disk_cleanup",
			"servod_old_logs_cleanup"
		],
		"exec_name":"sample_pass"
	},
	"servod_old_logs_cleanup":{
		"docs":[
			"Clean up the old servod files based on the max number of days given in the argument."
		],
		"exec_name":"servo_servod_old_logs_cleanup",
		"exec_extra_args":[
			"max_days:5"
		]
	},
	"is_not_container": {
		"conditions": [
			"is_container"
		],
		"exec_name":"sample_fail"
	},
	"servo_root_check": {
		"dependencies": [
			"cros_ssh",
			"servo_v3_root_present",
			"servo_v4_root_present"
		],
		"exec_name":"sample_pass"
	},
	"servo_topology": {
		"docs": [
			"host.check_diskspace('/mnt/stateful_partition', 0.5)"
		],
		"conditions": [
			"is_not_servo_v3"
		],
		"dependencies": [
			"cros_ssh",
			"servo_topology_single_child",
			"servo_topology_dual_setup"
		],
		"recovery_actions": [
			"servo_host_servod_restart"
		],
		"exec_name":"sample_pass"
	},
	"servo_topology_single_child": {
		"conditions": [
			"is_not_servo_v3"
		],
		"exec_name":"servo_topology_update",
		"exec_extra_args": [
			"min_child:1",
			"persist_topology:true"
		]
	},
	"servo_topology_dual_setup": {
		"conditions": [
			"is_not_servo_v3",
			"is_dual_setup"
		],
		"exec_name":"servo_topology_update",
		"exec_extra_args": [
			"min_child:2",
			"persist_topology:true"
		],
		"recovery_actions": [
			"servod_restart_dut"
		],
		"allow_fail_after_recovery": true
	},
	"servo_v3_root_present": {
		"docs": [
			"This remains to be implemented."
		],
		"dependencies": [
			"cros_ssh"
		],
		"conditions": [
			"is_servo_v3"
		],
		"exec_name":"sample_pass"
	},
	"servo_v4_root_present": {
		"dependencies": [
			"cros_ssh"
		],
		"conditions": [
			"is_not_servo_v3"
		]
	},
	"servo_fw_need_update": {
		"conditions": [
			"is_not_servo_v3"
		],
		"recovery_actions": [
			"servo_fw_update"
		]
	},
	"servod_get_serialname": {
		"docs":[
			"run command from xmlrpc"
		],
		"exec_name":"servod_echo"
	},
	"servo_get_ppdut5_mv": {
		"exec_extra_args": [
			"command:ppdut5_mv"
		],
		"exec_name":"servo_check_servod_control"
	},
	"servo_get_ppchg5_mv": {
		"exec_extra_args": [
			"command:ppchg5_mv"
		],
		"exec_name":"servo_check_servod_control"
	},
	"servod_servo_pd": {
		"docs":[
			"run command from xmlrpc"
		],
		"conditions": [
			"servo_v4_type_c"
		],
		"dependencies": [
			"servo_get_ppdut5_mv",
			"servo_get_ppchg5_mv"
		],
		"exec_extra_args": [
			"command:servo_pd_role",
			"expected_string_value:snk"
		],
		"exec_name":"servo_check_servod_control",
		"allow_fail_after_recovery": true
	},
	"servo_cr50_checks": {
		"conditions": [
			"is_not_servo_v3"
		],
		"dependencies": [
			"servo_cr50_low_sbu",
			"servo_cr50_enumerated"
		],
		"exec_name":"sample_pass"
	},
	"servo_cr50_low_sbu": {
		"conditions": [
			"is_not_servo_v3",
			"servo_v4_type_c",
			"servo_is_sbu_voltage_issue"
		],
		"recovery_actions": [
			"servo_host_servod_restart"
		]
	},
	"servo_is_sbu_voltage_issue": {
		"exec_extra_args": [
			"command:dut_sbu_voltage_float_fault",
			"expected_string_value:on"
		],
		"exec_name":"servo_check_servod_control"
	},
	"servo_cr50_enumerated": {
		"docs": [
			"prev name servo_cr50_off"
		],
		"conditions": [
			"is_not_servo_v3",
			"servo_v4_type_c",
			"servo_is_sbu_voltage_issue"
		],
		"recovery_actions": [
			"servo_host_servod_restart"
		],
		"exec_name":"sample_fail"
	},
	"servo_cr50_console": {
		"docs": [
			"Create new action to check that servotype has ccd_cr50, and set that as a condition for this action."
		],
		"conditions": [
			"is_not_servo_v3"
		],
		"dependencies": [
			"init_dut_for_servo"
		],
		"exec_extra_args": [
			"commands:cr50_ccd_level,cr50_testlab,cr50_ccd_state_flags",
			"any_one:true"
		],
		"recovery_actions": [
			"servo_host_servod_restart"
		],
		"exec_name":"servod_can_read_all"
	},
	"cr50_testlab": {
		"exec_extra_args": [
			"command:cr50_testlab",
			"expected_string_value:on"
		],
		"exec_name":"servo_check_servod_control"
	},
	"init_dut_for_servo": {
		"conditions": [
			"is_not_servo_v3"
		],
		"dependencies": [
			"servod_set_main_device"
		]
	},
	"servo_connection_pins": {
		"conditions": [
			"is_servo_v3",
			"servo_v4_type_a"
		],
		"dependencies": [
			"servo_cold_reset_pin",
			"servo_warm_reset_pin"
		],
		"exec_name":"sample_pass"
	},
	"pwr_button_supported_models": {
		"docs": [
			"power button check is not applicable for these models"
		],
		"exec_extra_args": [
			"string_values:arkham,gale,mistral,storm,whirlwind",
			"invert_result:true"
		],
		"exec_name":"dut_check_model"
	},
	"servo_pwr_button_pin": {
		"conditions": [
			"pwr_button_supported_models"
		],
		"dependencies": [
			"dut_has_model_name"
		],
		"exec_extra_args": [
			"command:pwr_button",
			"expected_string_value:release"
		],
		"recovery_actions": [
			"servo_host_servod_restart"
		],
		"exec_name":"servo_check_servod_control",
		"allow_fail_after_recovery": true
	},
	"servo_dut_detected": {
		"conditions": [
			"is_not_servo_v3",
			"servo_v4_type_a"
		],
		"recovery_actions": [
			"servo_host_servod_restart"
		],
		"exec_name":"servo_low_ppdut5"
	},
	"servo_ec_check": {
		"conditions": [
			"is_not_servo_v3",
			"dut_has_cros_ec"
		],
		"dependencies": [
			"servo_ec_console",
			"servo_pwr_button_pin",
			"servo_lid_open",
			"servo_battery_charging"
		],
		"exec_name":"sample_pass"
	},
	"dut_has_cros_ec": {
		"exec_extra_args": [
			"command:supports_cros_ec_communication",
			"expected_string_value:yes"
		],
		"exec_name":"servo_check_servod_control"
	},
	"servo_ec_console": {
		"conditions": [
			"is_not_servo_v3",
			"dut_has_cros_ec"
		],
		"exec_extra_args": [
			"commands:ec_system_powerstate,ec_board",
			"any_one:true"
		],
		"recovery_actions": [
			"servo_host_servod_restart"
		],
		"exec_name":"servod_can_read_all"
	},
	"battery_last_charge_readable": {
		"exec_extra_args": [
			"command:battery_full_charge_mah"
		],
		"exec_name":"servo_check_servod_control"
	},
	"servo_battery_charging": {
		"conditions": [
			"is_not_servo_v3",
			"dut_has_cros_ec",
			"battery_last_charge_readable"
		],
		"allow_fail_after_recovery": true
	},
	"servo_testlab_enabled": {
		"conditions": [
			"is_not_servo_v3",
			"servo_v4_type_c"
		],
		"exec_name":"sample_pass"
	},
	"servo_detect_usbkey": {
		"docs": [
			"Will detect the path to USB Drive on servo-host.",
			"Verify that usb-key is responsive"
		],
		"allow_fail_after_recovery": true
	},
	"servo_audit_usbkey": {
		"docs": ["This action will detect whether or not the USB drive is in working condition."],
		"allow_fail_after_recovery": true,
		"dependencies" : ["servo_detect_usbkey"],
		"exec_timeout": {
			"seconds":7300
		}
	},
	"servo_v4_type_c": {
		"exec_name":"sample_pass"
	},
	"servo_lid_open": {
		"recovery_actions": [
			"servo_host_servod_restart"
		],
		"exec_name":"sample_pass"
	},
	"is_not_servo_v3": {
		"conditions":[
			"servo_is_v3"
		],
		"exec_name":"sample_fail"
	},
	"servo_v4_type_a": {
		"exec_name":"sample_pass"
	},
	"is_dual_setup": {
		"exec_name":"sample_pass"
	},
	"is_not_dual_setup": {
		"exec_name":"sample_pass"
	},
	"servod_set_main_device": {
		"conditions" : [
			"servo_has_active_dut_controller"
		],
		"recovery_actions": [
			"servo_host_servod_restart"
		]
	},
	"servo_fw_update": {
		"docs":[
			"Try to update in  normal ways 3 times",
			"if fail allow run force update"
		],
		"exec_extra_args": [
			"try:3",
			"use_force"
		],
		"exec_name":"sample_pass"
	},
	"servo_warm_reset_supported": {
		"exec_extra_args": [
			"command:warm_reset"
		],
		"exec_name":"servo_check_servod_control"
	},
	"servo_warm_reset_pin": {
		"docs": [
			"TODO: If the type C has warm reset, we can drop this condition."
		],
		"conditions": [
			"servo_warm_reset_supported"
		],
		"exec_extra_args": [
			"command:warm_reset",
			"expected_string_value:off"
		],
		"recovery_actions": [
			"servo_host_servod_restart"
		],
		"exec_name":"servo_check_servod_control"
	},
	"servo_cold_reset_pin": {
		"exec_extra_args": [
			"command:cold_reset",
			"expected_string_value:off"
		],
		"recovery_actions": [
			"servo_host_servod_restart"
		],
		"exec_name":"servo_check_servod_control"
	},
	"servo_servod_echo_host": {
		"docs": ["Uses a servod control to check whether the servod daemon is responsive."],
		"exec_timeout": {
			"seconds": 30
		},
		"recovery_actions": [
			"servo_host_servod_restart"
		]
	},
	"update_servo_type_label":{
		"docs":[
			"Update the servo type label for the DUT info."
		],
		"exec_name":"servo_update_servo_type_label"
	},
	"dut_controller_missing_fault_off":{
		"exec_extra_args": [
			"command:dut_controller_missing_fault",
			"expected_string_value:off"
		],
		"exec_name":"servo_check_servod_control"
	},
	"servo_host_servod_restart": {
		"exec_timeout": {
			"seconds": 120
		},
		"run_control": 2
	},
	"servo_has_active_dut_controller": {
		"exec_extra_args": [
			"command:active_dut_controller"
		],
		"exec_name":"servo_check_servod_control"
	},
	"servod_restart_dut": {
		"exec_name":"sample_pass"
	}
}`
