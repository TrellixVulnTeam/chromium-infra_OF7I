// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package recovery

const servoRepairPlanBody = `
"critical_actions": [
	"servo_host_info",
	"servod_info",
	"create_host",
	"cros_ssh",
	"servo_v3_uptime",
	"lock_labstation",
	"has_enough_disk_space",
	"servo_components_check",
	"servo_fw_need_update",
	"servo_host_servod_init",
	"servod_echo",
	"servod_get_serialname",
	"servod_servo_pd",
	"servod_read_servo_type",
	"servo_cr50_checks",
	"servo_connection_pins",
	"servo_ppdut5",
	"servod_set_main_device",
	"servo_ec_check",
	"servo_testlab_enabled",
	"servo_detect_usbkey",
	"servo_servod_echo_host"
],
"actions": {
	"servo_host_servod_restart": {
		"exec_timeout": {
			"seconds": 120
		}
	},
	"servo_host_servod_init": {
		"exec_timeout": {
			"seconds": 120
		}
	},
	"servo_host_info": {
		"dependencies" : [
			"has_dut_name"
		],
		"exec_name":"sample_pass"
	},
	"servod_info": {
		"exec_name":"sample_pass"
	},
	"create_host": {
		"docs": [
			"Only to create docker"
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
	"is_labstation": {
		"docs": [
			"condition to check is name contains labstation word"
		],
		"exec_name":"sample_pass"
	},
	"is_container": {
		"docs": [
			"condition to check is name contains labstation word"
		],
		"exec_name":"sample_pass"
	},
	"is_servo_v3": {
		"docs": [
			"Revers from labstation and docker"
		],
		"exec_name":"sample_pass"
	},
	"lock_labstation": {
		"docs": [
			"create lock file is_in_use"
		],
		"conditions": [
			"is_labstation"
		],
		"exec_name":"sample_pass"
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
		]
	},
	"is_not_container": {
		"conditions": [
			"is_servo_v3",
			"is_labstation"
		],
		"exec_name":"sample_fail"
	},
	"servo_components_check": {
		"dependencies": [
			"cros_ssh",
			"root_present_servo_v3",
			"servo_v4_root_present",
			"servo_topology"
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
	"root_present_servo_v3": {
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
		"exec_name":"sample_pass"
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
			"servo_type_c"
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
			"servo_cr50_enumerated",
			"servo_cr50_console"
		],
		"exec_name":"sample_pass"
	},
	"servo_cr50_low_sbu": {
		"conditions": [
			"is_not_servo_v3",
			"servo_type_c",
			"servo_is_sbu_voltage_issue"
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
			"servo_type_c",
			"servo_is_sbu_voltage_issue"
		],
		"exec_name":"sample_fail"
	},
	"servo_cr50_console": {
		"conditions": [
			"is_not_servo_v3"
		],
		"dependencies": [
			"servod_dut_controller"
		],
		"exec_extra_args": [
			"cr50_ccd_level",
			"cr50_testlab",
			"cr50_ccd_state_flags"
		],
		"exec_name":"sample_pass"
	},
	"cr50_testlab": {
		"exec_extra_args": [
			"command:cr50_testlab",
			"expected_string_value:on"
		],
		"exec_name":"servo_check_servod_control"
	},
	"servod_dut_controller": {
		"conditions": [
			"is_not_servo_v3"
		],
		"exec_name":"sample_pass"
	},
	"servo_connection_pins": {
		"conditions": [
			"is_not_servo_v3"
		],
		"dependencies": [
			"servo_cold_reset_pin",
			"servo_warm_reset_pin"
		],
		"exec_name":"sample_pass"
	},
	"servo_pwr_button_pin": {
		"exec_extra_args": [
			"except:board1,board2"
		],
		"exec_name":"sample_pass"
	},
	"servo_ppdut5": {
		"conditions": [
			"is_not_servo_v3",
			"servo_v4_type_a"
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
	"servo_ec_console": {
		"conditions": [
			"is_not_servo_v3",
			"dut_has_cros_ec"
		],
		"exec_name":"sample_pass"
	},
	"servo_battery_charging": {
		"conditions": [
			"is_not_servo_v3",
			"dut_has_cros_ec"
		],
		"allow_fail_after_recovery": true,
		"exec_name":"sample_pass"
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
	"servo_type_c": {
		"exec_name":"sample_pass"
	},
	"servo_lid_open": {
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
		"exec_name":"sample_pass"
	},
	"servo_v4_type_c": {
		"exec_name":"sample_pass"
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
		"conditions": [
			"servo_warm_reset_supported"
		],
		"exec_extra_args": [
			"command:warm_reset",
			"expected_string_value:off"
		],
		"exec_name":"servo_check_servod_control"
	},
	"dut_has_cros_ec": {
		"exec_name":"sample_pass"
	},
	"reboot":{
		"exec_name":"sample_pass"
	},
	"servo_cold_reset_pin": {
		"exec_extra_args": [
			"command:cold_reset",
			"expected_string_value:off"
		],
		"exec_name":"servo_check_servod_control"
	},
	"servod_read_servo_type": {
		"exec_name":"sample_pass"
	},
	"servod_restart_dut": {
		"exec_name":"sample_pass"
	},
	"servo_servod_echo_host": {
		"exec_timeout": {
			"seconds": 30
		}
	},
	"servod_echo": {
		"dependencies" : ["servo_servod_echo_host"]
	}
}`
