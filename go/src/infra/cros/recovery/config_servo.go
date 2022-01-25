// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package recovery

const servoRepairPlanBody = `
"critical_actions": [
	"set_state_missing_config",
	"servo_host_info",
	"servo_servod_port_present",
	"servod_info",
	"set_state_wrong_config",
	"servo_has_serial",
	"init_docker_host",
	"set_state_no_ssh",
	"cros_ssh",
	"cros_ping",
	"servo_v3_uptime",
	"set_state_servo_host_issue",
	"lock_labstation",
	"set_state_broken",
	"has_enough_disk_space",
	"set_state_not_connected",
	"servo_root_check",
	"set_state_topology_issue",
	"servo_topology",
	"set_state_servo_updater_issue",
	"servo_fw_need_update",
	"set_state_servod_issue",
	"servo_host_servod_start",
	"set_state_servo_host_issue",
	"servod_get_serialname",
	"set_state_servod_proxy_issue",
	"servo_servod_echo_host",
	"set_state_cold_reset_pin_issue",
	"servo_cold_reset_pin",
	"set_state_warm_reset_pin_issue",
	"servo_warm_reset_pin",
	"set_state_dut_not_connected",
	"servo_dut_detected",
	"set_state_servod_issue",
	"servod_servo_pd",
	"set_state_cr50_not_enumerated",
	"servo_cr50_checks",
	"set_state_servod_dut_controller_missing",
	"dut_controller_missing_fault_off",
	"set_state_cr50_console_missing",
	"servo_cr50_console",
	"set_state_ec_broken",
	"servo_ec_check",
	"set_state_servod_issue",
	"servod_set_main_device",
	"init_dut_for_servo",
	"set_state_ccd_testlab_issue",
	"servo_testlab_enabled",
	"set_state_broken",
	"servo_detect_usbkey",
	"update_servo_type_label",
	"set_state_working"
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
			"servo_host_servod_restart",
			"servo_power_delivery_repair"
		]
	},
	"servo_host_info": {
		"exec_name":"dut_has_name"
	},
	"servod_info": {
		"docs" : ["Verify that servo port is available, and servo serial is readable.",
					"This is applicable only if the servo version is not V3."],
		"conditions": [
			"is_not_servo_v3"
		],
		"dependencies": [
			"servo_servod_port_present",
			"servod_get_serialname"
		],
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
			"servo_host_servod_restart",
			"servo_power_cycle_repair",
			"servo_power_delivery_repair"
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
		"exec_timeout": {
			"seconds": 300
		},
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
			"is_servo_v4_type_c"
		],
		"dependencies": [
			"servo_get_ppdut5_mv",
			"servo_get_ppchg5_mv"
		],
		"exec_extra_args": [
			"command:servo_pd_role",
			"expected_string_value:snk"
		],
		"recovery_actions": [
			"servo_power_delivery_repair"
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
			"is_servo_v4_type_c",
			"servo_is_sbu_voltage_issue"
		],
		"recovery_actions": [
			"servo_host_servod_restart",
			"servo_power_cycle_repair",
			"servo_power_delivery_repair"
		]
	},
	"servo_is_sbu_voltage_issue": {
		"exec_extra_args": [
			"command:dut_sbu_voltage_float_fault",
			"expected_string_value:on"
		],
		"recovery_actions": [
			"servo_power_cycle_repair",
			"servo_power_delivery_repair"
		],
		"exec_name":"servo_check_servod_control"
	},
	"servo_cr50_enumerated": {
		"docs": [
			"prev name servo_cr50_off"
		],
		"conditions": [
			"is_not_servo_v3",
			"is_servo_v4_type_c",
			"servo_is_sbu_voltage_issue"
		],
		"recovery_actions": [
			"servo_host_servod_restart",
			"servo_power_cycle_repair"
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
			"servo_host_servod_restart",
			"servo_power_cycle_repair",
			"servo_power_delivery_repair"
		],
		"exec_name":"servod_can_read_all"
	},
	"cr50_testlab": {
		"exec_extra_args": [
			"command:cr50_testlab",
			"expected_string_value:on"
		],
		"recovery_actions": [
			"servo_power_delivery_repair"
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
			"servo_host_servod_restart",
			"servo_power_delivery_repair"
		],
		"exec_name":"servo_check_servod_control",
		"allow_fail_after_recovery": true
	},
	"servo_dut_detected": {
		"conditions": [
			"is_not_servo_v3",
			"is_servo_v4_type_a"
		],
		"recovery_actions": [
			"servo_host_servod_restart",
			"servo_power_delivery_repair"
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
			"is_servo_v4_type_c"
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
	"is_servo_v4": {
		"docs": ["This action will detect whether or not the attached servo device is of type V4."]
	},
	"is_servo_v4_type_c": {
		"docs": ["This action will detect whether or not the attached servo V4 device is connect to DUT using Type-C connection."],
		"conditions": [
			"is_servo_v4"
		],
		"exec_extra_args": [
			"command:root.dut_connection_type",
			"expected_string_value:type-c"
		],
		"exec_name":"servo_check_servod_control"
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
	"is_servo_v4_type_a": {
		"docs": ["This action will detect whether or not the attached servo V4 device is connect to DUT using Type-A connection."],
		"conditions": [
			"is_servo_v4"
		],
		"exec_extra_args": [
			"command:root.dut_connection_type",
			"expected_string_value:type-a"
		],
		"exec_name":"servo_check_servod_control"
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
		"conditions": [
			"is_not_servo_v3"
		],
		"exec_extra_args": [
			"try_attempt_count:3",
			"try_force_update_after_fail:true", 
			"force_update:false", 
			"ignore_version:false"
		],
		"exec_timeout": {
			"seconds": 600
		},
		"exec_name":"servo_update_servo_firmware"
	},
	"servo_warm_reset_supported": {
		"exec_extra_args": [
			"command:warm_reset"
		],
		"exec_name":"servo_check_servod_control"
	},
	"servo_warm_reset_pin_for_servo_v3": {
		"conditions": [
			"is_servo_v3",
			"servo_warm_reset_supported"
		],
		"exec_extra_args": [
			"command:warm_reset",
			"expected_string_value:off"
		],
		"recovery_actions": [
			"servo_host_servod_restart",
			"servo_power_delivery_repair"
		],
		"exec_name":"servo_check_servod_control"
	},
	"servo_warm_reset_pin_for_servo_micro": {
		"conditions": [
			"is_servo_micro",
			"servo_warm_reset_supported"
		],
		"exec_extra_args": [
			"command:warm_reset",
			"expected_string_value:off"
		],
		"recovery_actions": [
			"servo_host_servod_restart",
			"servo_power_delivery_repair"
		],
		"exec_name":"servo_check_servod_control"
	},
	"servo_warm_reset_pin": {
		"docs": [
			"We need to check for warm reset only for servo micro and V3."
		],
		"dependencies": [
			"servo_warm_reset_pin_for_servo_v3",
			"servo_warm_reset_pin_for_servo_micro"
		],
		"exec_name":"sample_pass"
	},
	"servo_cold_reset_pin": {
		"conditions": [
			"is_servo_v3",
			"is_servo_v4_type_a"
		],
		"exec_extra_args": [
			"command:cold_reset",
			"expected_string_value:off"
		],
		"recovery_actions": [
			"servo_host_servod_restart",
			"servo_power_delivery_repair"
		],
		"exec_name":"servo_check_servod_control"
	},
	"servo_servod_echo_host": {
		"docs": ["Uses a servod control to check whether the servod daemon is responsive."],
		"exec_timeout": {
			"seconds": 30
		},
		"recovery_actions": [
			"servo_host_servod_restart",
			"servo_power_delivery_repair"
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
		"recovery_actions": [
			"servo_power_delivery_repair"
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
		"recovery_actions": [
			"servo_power_delivery_repair"
		],
		"exec_name":"servo_check_servod_control"
	},
	"servod_restart_dut": {
		"exec_name":"sample_pass"
	},
	"servo_power_cycle_repair":{
		"docs":[
			"Toggle the servod command servo_pd_role only once. And then stop the servod afterwards. TODO: Add dependency for servo initialize."
		],
		"dependencies": [
			"servo_pd_toggle_once"
		],
		"exec_timeout": {
			"seconds": 120
		},
		"run_control": 1,
		"exec_name":"servo_host_servod_stop"
	},
	"servo_pd_toggle_once":{
		"docs":[
			"Toggle the servod command servo_pd_role only once."
		],
		"exec_extra_args": [
			"toggle_times:1",
			"wait_in_retry:5",
			"wait_before_retry:1"
		],
		"run_control": 1,
		"exec_name":"servo_servod_toggle_pd_role"
	},
	"servo_power_delivery_repair":{
		"docs":[
			"Toggle the servod command servo_pd_role 5 times. And then stop the servod afterwards. TODO: Add dependency for servo initialize."
		],
		"dependencies": [
			"servo_pd_toggle_five_times"
		],
		"exec_timeout": {
			"seconds": 600
		},
		"run_control": 1,
		"exec_name":"servo_host_servod_stop"
	},
	"servo_pd_toggle_five_times":{
		"docs":[
			"Toggle the servod command servo_pd_role 5 times."
		],
		"exec_extra_args": [
			"toggle_times:5",
			"wait_in_retry:5",
			"wait_before_retry:1"
		],
		"run_control": 1,
		"exec_name":"servo_servod_toggle_pd_role"
	},
	"set_state_missing_config": {
		"exec_extra_args": [
			"state:MISSING_CONFIG"
		],
		"exec_name": "servo_set_servo_state"
	},
	"set_state_wrong_config": {
		"exec_extra_args": [
			"state:WRONG_CONFIG"
		],
		"exec_name": "servo_set_servo_state"
	},
	"servo_has_serial": {
		"conditions": [
			"is_not_servo_v3"
		],
		"exec_name": "servo_has_serial"
	},
	"set_state_no_ssh": {
		"exec_extra_args": [
			"state:NO_SSH"
		],
		"exec_name": "servo_set_servo_state"
	},
	"set_state_broken": {
		"exec_extra_args": [
			"state:BROKEN"
		],
		"exec_name": "servo_set_servo_state"
	},
	"set_state_not_connected": {
		"exec_extra_args": [
			"state:NOT_CONNECTED"
		],
		"exec_name": "servo_set_servo_state"
	},
	"set_state_topology_issue": {
		"exec_extra_args": [
			"state:TOPOLOGY_ISSUE"
		],
		"exec_name": "servo_set_servo_state"
	},
	"set_state_servo_updater_issue": {
		"exec_extra_args": [
			"state:SERVO_UPDATER_ISSUE"
		],
		"exec_name": "servo_set_servo_state"
	},
	"set_state_servod_issue": {
		"exec_extra_args": [
			"state:SERVOD_ISSUE"
		],
		"exec_name": "servo_set_servo_state"
	},
	"set_state_servo_host_issue": {
		"exec_extra_args": [
			"state:SERVOD_ISSUE"
		],
		"exec_name": "servo_set_servo_state"
	},
	"set_state_servod_proxy_issue": {
		"exec_extra_args": [
			"state:SERVOD_PROXY_ISSUE"
		],
		"exec_name": "servo_set_servo_state"
	},
	"set_state_cold_reset_pin_issue": {
		"exec_extra_args": [
			"state:COLD_RESET_PIN_ISSUE"
		],
		"exec_name": "servo_set_servo_state"
	},
	"set_state_warm_reset_pin_issue": {
		"exec_extra_args": [
			"state:WARM_RESET_PIN_ISSUE"
		],
		"exec_name": "servo_set_servo_state"
	},
	"set_state_dut_not_connected": {
		"exec_extra_args": [
			"state:DUT_NOT_CONNECTED"
		],
		"exec_name": "servo_set_servo_state"
	},
	"set_state_cr50_not_enumerated": {
		"exec_extra_args": [
			"state:DUT_NOT_CONNECTED"
		],
		"exec_name": "servo_set_servo_state"
	},
	"set_state_servod_dut_controller_missing": {
		"exec_extra_args": [
			"state:DUT_NOT_CONNECTED"
		],
		"exec_name": "servo_set_servo_state"
	},
	"set_state_cr50_console_missing": {
		"exec_extra_args": [
			"state:CR50_CONSOLE_MISSING"
		],
		"exec_name": "servo_set_servo_state"
	},
	"set_state_ec_broken": {
		"exec_extra_args": [
			"state:EC_BROKEN"
		],
		"exec_name": "servo_set_servo_state"
	},
	"set_state_ccd_testlab_issue": {
		"exec_extra_args": [
			"state:CCD_TESTLAB_ISSUE"
		],
		"exec_name": "servo_set_servo_state"
	},
	"set_state_working": {
		"exec_extra_args": [
			"state:WORKING"
		],
		"exec_name": "servo_set_servo_state"
	}
}`
