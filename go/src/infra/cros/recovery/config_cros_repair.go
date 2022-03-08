// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package recovery

import (
	"fmt"
	"strings"
)

// List of critical actions for repair of the ChromeOS.
var crosRepairPlanCriticalActionList = []string{
	"dut_state_repair_failed",
	"cros_ssh",
	"internal_storage",
	"last_provision_successful",
	"device_system_info",
	"has_python",
	"device_enrollment",
	"power_info",
	"tpm_info",
	"tools_checks",
	"hardware_audit",
	"firmware_check",
	"stop_start_ui",
	"rw_vpd",
	"servo_keyboard",
	"servo_mac_address",
	"cros_match_job_repo_url_version_to_inventory",
	"cros_provisioning_labels_repair",
	"device_labels",
	"Collect dmesg logs from DUT",
}

// List of actions configs for repair of the ChromeOS.
// These are all the actions that are available; in order to be run, an action
// must be reachable from a critical action.
const crosRepairPlanActions = `
"Device is pingable": {
	"dependencies":[
		"dut_has_name",
		"dut_has_board_name",
		"dut_has_model_name"
	],
	"exec_name": "cros_ping",
	"exec_timeout": {
		"seconds":15
	}
},
"cros_ssh":{
	"dependencies":[
		"dut_has_name",
		"dut_has_board_name",
		"dut_has_model_name",
		"Device is pingable"
	],
	"recovery_actions": [
		"cros_servo_power_reset_repair",
		"Trigger kernel panic to reset the whole board and try ssh to DUT",
		"cros_servo_cr50_reboot_repair"
	]
},
"internal_storage":{
	"dependencies":[
		"cros_storage_writing",
		"cros_storage_file_system",
		"cros_storage_space_check",
		"cros_audit_storage_smart"
	],
	"exec_name":"sample_pass"
},
"device_system_info":{
	"conditions":[
		"is_not_flex_board"
	],
	"dependencies":[
		"cros_default_boot",
		"cros_boot_in_normal_mode",
		"cros_hwid_info",
		"cros_serial_number_info",
		"cros_tpm_fwver_match",
		"cros_tpm_kernver_match"
	],
	"exec_name":"sample_pass"
},
"has_python":{
	"docs":[
		"Verify that device has python on it.",
		"The Reven boards does not have python. TBD"
	],
	"conditions":[
		"is_not_flex_board"
	],
	"dependencies":[
		"cros_storage_writing"
	],
	"exec_name":"cros_has_python_interpreter_working",
	"recovery_actions": [
		"Quick provision OS"
	]
},
"last_provision_successful":{
	"dependencies":[
		"cros_storage_writing"
	],
	"exec_name":"cros_is_last_provision_successful"
},
"device_enrollment":{
	"dependencies":[
		"cros_storage_writing"
	],
	"recovery_actions": [
		"tpm_enrollment_cleanup_and_reboot"
	],
	"exec_name":"cros_is_enrollment_in_clean_state"
},
"tpm_enrollment_cleanup_and_reboot":{
	"docs":[
		"Cleanup the enrollment state."
	],
	"dependencies":[
		"cros_ssh"
	],
	"exec_extra_args": [
		"repair_timeout:120",
		"clear_tpm_owner_timeout:60",
		"file_deletion_timeout:120",
		"reboot_timeout:10",
		"tpm_timeout:150"
	],
	"exec_timeout": {
		"seconds":600
	},
	"exec_name":"cros_enrollment_cleanup"
},
"power_info":{
	"docs": [
		"Check for the AC power, and battery charging capability."
	],
	"conditions":[
		"is_not_flex_board",
		"cros_is_not_virtual_machine"
	],
	"dependencies":[
		"cros_storage_writing",
		"cros_is_ac_power_connected",
		"battery_is_good"
	],
	"recovery_actions": [
		"Power cycle DUT by RPM and wait",
		"cros_servo_power_reset_repair",
		"cros_servo_cr50_reboot_repair"
	],
	"exec_name":"sample_pass"
},
"tpm_info":{
	"conditions":[
		"is_not_flex_board",
		"cros_is_not_virtual_machine",
		"cros_is_tpm_present"
	],
	"exec_name":"cros_is_tpm_in_good_status"
},
"tools_checks":{
	"dependencies":[
		"cros_gsctool"
	],
	"exec_name":"sample_pass"
},
"hardware_audit":{
	"dependencies":[
		"wifi_audit",
		"bluetooth_audit"
	],
	"exec_name":"sample_pass"
},
"firmware_check":{
	"conditions":[
		"is_not_flex_board"
	],
	"dependencies":[
		"cros_storage_writing",
		"cros_is_firmware_in_good_state",
		"cros_rw_firmware_stable_verion"
	],
	"exec_name":"sample_pass"
},
"stop_start_ui":{
	"docs": [
		"Check the command 'stop ui' won't crash the DUT."
	],
	"exec_timeout": {
		"seconds": 45
	},
	"recovery_actions": [
		"cros_servo_power_reset_repair",
		"cros_servo_cr50_reboot_repair"
	],
	"exec_name":"cros_stop_start_ui"
},
"rw_vpd":{
	"docs":[
		"Verify that keys: 'should_send_rlz_ping', 'gbind_attribute', 'ubind_attribute' are present in vpd RW_VPD partition."
	],
	"conditions":[
		"is_not_flex_board"
	],
	"exec_name":"cros_are_required_rw_vpd_keys_present",
	"allow_fail_after_recovery": true
},
"servo_keyboard":{
	"conditions":[
		"dut_servo_host_present",
		"servo_state_is_working",
		"is_servo_keyboard_image_tool_present"
	],
	"dependencies":[
		"servo_init_usb_keyboard",
		"lufa_keyboard_found"
	],
	"exec_name":"cros_run_shell_command",
	"exec_extra_args":[
		"lsusb -vv -d 03eb:2042 |grep \"Remote Wakeup\""
	],
	"allow_fail_after_recovery": true
},
"servo_mac_address":{
	"conditions":[
		"dut_servo_host_present",
		"is_not_servo_v3",
		"servod_control_exist_for_mac_address"
	],
	"exec_name":"servo_audit_nic_mac_address",
	"allow_fail_after_recovery": true
},
"is_not_servo_v3": {
	"conditions":[
		"is_servo_v3"
	],
	"exec_name":"sample_fail"
},
"servod_control_exist_for_mac_address":{
	"exec_name":"servo_check_servod_control",
	"exec_extra_args":[
		"command:macaddr"
	]
},
"servo_init_usb_keyboard":{
	"docs":[
		"set servo's 'init_usb_keyboard' command to 'on' value."
	],
	"dependencies":[
		"dut_servo_host_present"
	],
	"exec_name":"servo_set",
	"exec_extra_args":[
		"command:init_usb_keyboard",
		"string_value:on"
	]
},
"is_servo_keyboard_image_tool_present":{
	"docs":[
		"check if the servo keyboard image specified by the name of dfu-programmer can be found in DUT cli."
	],
	"dependencies":[
		"dut_servo_host_present"
	],
	"exec_name":"cros_is_tool_present",
	"exec_extra_args":[
		"tools:dfu-programmer"
	]
},
"lufa_keyboard_found":{
	"docs":[
		"check if the lufa keyboard can be found by finding the match of the model information of it."
	],
	"exec_name":"cros_run_shell_command",
	"exec_extra_args":[
		"lsusb -d 03eb:2042 |grep \"LUFA Keyboard Demo\""
	]
},
"servo_state_is_working":{
	"docs":[
		"check the servo's state is WORKING."
	],
	"dependencies":[
		"dut_servo_host_present"
	],
	"exec_name":"servo_match_state",
	"exec_extra_args":[
		"state:WORKING"
	]
},
"servo_state_is_not_working":{
	"docs":[
		"check the servo's state is not WORKING."
	],
	"conditions":[
		"servo_state_is_working"
	],
	"exec_name":"sample_fail"
},
"cros_rw_firmware_stable_verion":{
	"dependencies":[
		"cros_storage_writing",
		"cros_is_on_rw_firmware_stable_verion",
		"cros_is_rw_firmware_stable_version_available"
	],
	"exec_name":"sample_pass"
},
"cros_gsctool":{
	"exec_name":"sample_pass"
},
"battery_is_good":{
	"docs":[
		"Check battery on the DUT is normal and update battery hardware state accordingly."
	],
	"conditions":[
		"cros_is_battery_expected",
		"cros_is_not_virtual_machine",
		"Battery is expected on device",
		"Battery is present on device"
	],
	"dependencies":[
		"cros_storage_writing",
		"cros_is_battery_chargable_or_good_level"
	],
	"exec_name":"cros_audit_battery"
},
"Battery is expected on device":{
	"docs":[
		"Verifies that device is expected to have battery based on DUT info."
	],
	"exec_name":"dut_has_battery"
},
"Battery is present on device":{
	"docs":[
		"Verifies if battery present is reported as present in power supply info."
	],
	"exec_name":"cros_is_battery_present"
},
"wifi_audit":{
	"docs":[
		"Check wifi on the DUT is normal and update wifi hardware state accordingly."
	],
	"dependencies":[
		"cros_ssh"
	],
	"exec_name":"cros_audit_wifi",
	"allow_fail_after_recovery": true
},
"bluetooth_audit":{
	"docs":[
		"Check bluetooth on the DUT is normal and update bluetooth hardware state accordingly."
	],
	"dependencies":[
		"cros_ssh"
	],
	"exec_name":"cros_audit_bluetooth",
	"allow_fail_after_recovery": true
},
"cros_tpm_fwver_match":{
	"dependencies":[
		"cros_storage_writing"
	],
	"conditions":[
		"is_not_flex_board"
	],
	"exec_name":"cros_match_dev_tpm_firmware_version"
},
"cros_tpm_kernver_match":{
	"dependencies":[
		"cros_storage_writing"
	],
	"conditions":[
		"is_not_flex_board"
	],
	"exec_name":"cros_match_dev_tpm_kernel_version"
},
"cros_default_boot":{
	"dependencies":[
		"cros_storage_writing"
	],
	"conditions":[
		"is_not_flex_board"
	],
	"exec_name":"cros_is_default_boot_from_disk"
},
"cros_boot_in_normal_mode":{
	"conditions":[
		"is_not_flex_board"
	],
	"dependencies":[
		"cros_storage_writing"
	],
	"recovery_actions": [
		"cros_switch_to_secure_mode_and_reboot"
	],
	"exec_name":"cros_is_not_in_dev_mode"
},
"cros_hwid_info":{
	"conditions":[
		"is_not_flex_board",
		"dut_has_hwid_info"
	],
	"dependencies":[
		"cros_storage_writing"
	],
	"exec_name":"cros_match_hwid_to_inventory"
},
"cros_serial_number_info":{
	"conditions":[
		"is_not_flex_board",
		"dut_has_serial_number_info"
	],
	"dependencies":[
		"cros_storage_writing"
	],
	"exec_name":"cros_match_serial_number_inventory"
},
"dut_has_hwid_info":{
	"exec_name":"sample_pass"
},
"dut_has_serial_number_info":{
	"exec_name":"sample_pass"
},
"cros_storage_writing":{
	"dependencies":[
		"cros_ssh"
	],
	"recovery_actions": [
		"cros_switch_to_secure_mode_and_reboot"
	],
	"exec_name":"cros_is_file_system_writable"
},
"cros_storage_file_system":{
	"dependencies":[
		"cros_ssh"
	],
	"exec_name":"cros_has_critical_kernel_error"
},
"cros_storage_space_check":{
	"dependencies":[
		"cros_stateful_partition_has_enough_inodes",
		"cros_stateful_partition_has_enough_storage_space",
		"cros_encrypted_stateful_partition_has_enough_storage_space"
	],
	"exec_name":"sample_pass"
},
"cros_stateful_partition_has_enough_inodes":{
	"docs":[
		"check the stateful partition path has enough inodes"
	],
	"exec_name":"cros_has_enough_inodes",
	"exec_extra_args":[
		"/mnt/stateful_partition:100"
	]
},
"cros_stateful_partition_has_enough_storage_space":{
	"docs":[
		"check the stateful partition have enough disk space. The storage unit is in GB."
	],
	"exec_name":"cros_has_enough_storage_space",
	"exec_extra_args":[
		"/mnt/stateful_partition:0.7"
	]
},
"cros_encrypted_stateful_partition_has_enough_storage_space":{
	"docs":[
		"check the encrypted stateful partition have enough disk space. The storage unit is in GB."
	],
	"exec_name":"cros_has_enough_storage_space",
	"exec_extra_args":[
		"/mnt/stateful_partition/encrypted:0.1"
	]
},
"device_labels":{
	"dependencies":[
		"device_sku",
		"cr50_labels",
		"audio_loop_back_label"
	 ],
	 "exec_name":"sample_pass"
},
"audio_loop_back_label":{
	"docs":[
		"Update the audio_loop_back label on the cros Device."
	],
	"conditions":[
		"dut_audio_loop_back_state_not_working"
	],
	"exec_name":"cros_update_audio_loopback_state_label",
	"allow_fail_after_recovery": true
},
"dut_audio_loop_back_state_not_working":{
	"docs":[
		"Confirm that the DUT's audio loopback state is in not working state"
	],
	"conditions":[
		"cros_is_audio_loopback_state_working"
	],
	"exec_name":"sample_fail"
},
"cr50_labels":{
	"docs":[
		"Update the cr50 label on the cros Device."
	],
	"conditions":[
		"cros_is_cr50_firmware_exist"
	],
	"dependencies":[
		"cros_update_cr50_label",
		"cros_update_cr50_key_id_label"
	 ],
	"exec_name":"sample_pass",
	"allow_fail_after_recovery": true
},
"cros_is_cr50_firmware_exist":{
	"docs":[
		"Checks if the cr 50 firmware exists on the DUT by running the gsctool version command."
	],
	"exec_name":"cros_run_shell_command",
	"exec_extra_args":[
		"gsctool -a -f"
	]
},
"device_sku":{
	"docs":[
		"Update the device_sku label from the device if not present in inventory data."
	],
	"conditions":[
		"dut_does_not_have_device_sku"
	],
	"exec_name":"cros_update_device_sku",
	"allow_fail_after_recovery": true
},
"dut_does_not_have_device_sku":{
	"docs":[
		"Confirm that the DUT itself does not have device_sku label."
	],
	"conditions":[
		"dut_has_device_sku"
	],
	"exec_name":"sample_fail"
},
"Servo has USB-key with require image":{
	"docs":[
		"USB-drive contains stable image on it."
	],
	"conditions":[
		"dut_servo_host_present",
		"servo_state_is_working"
	],
	"exec_name":"servo_usbkey_has_stable_image",
	"exec_timeout": {
		"seconds":120
	},
	"recovery_actions":[
		"Download stable image to USB-key"
	]
},
"Download stable image to USB-key":{
	"docs":[
		"Download lab stable image on servo USB-key",
		"Download the image can take longer if labstation download parallel a few images."
	],
	"dependencies":[
		"dut_servo_host_present",
		"servo_state_is_working"
	],
	"exec_name":"servo_download_image_to_usb",
	"exec_timeout": {
		"seconds":3000
	}
},
"cros_match_cros_version_to_inventory":{
	"docs":[
		"Verify that cros-version match version on the host."
	],
	"dependencies":[
		"cros_ssh"
	 ],
	"recovery_actions": [
		"cros_provisioning_labels_repair"
	]
},
"cros_match_job_repo_url_version_to_inventory":{
	"docs":[
		"Verify that job_repo_url matches the version on the host."
	],
	"dependencies":[
		"cros_ssh"
	 ],
	"recovery_actions": [
		"cros_provisioning_labels_repair"
	]
},
"cros_provisioning_labels_repair":{
	"docs":[
		"Cleanup the labels and job-repo-url."
	],
	"dependencies":[
		"cros_update_provision_os_version",
		"cros_update_job_repo_url"
	 ],
	"exec_name":"sample_pass"
},
"cros_switch_to_secure_mode_and_reboot":{
	"docs":[
		"This repair action utilizes the dependent actions to set the",
		" GBB flags and disable booting into dev-mode. Then it reboots",
		" the DUT."
	],
	"dependencies":[
		"cros_set_gbb_flags",
		"cros_switch_to_secure_mode",
		"cros_reboot"
	],
	"exec_name":"sample_pass"
},
"cros_set_gbb_flags":{
	"docs":[
		"This action sets the GBB flags."
	],
	"exec_timeout": {
		"seconds":3600
	},
	"allow_fail_after_recovery": true
},
"cros_switch_to_secure_mode":{
	"docs":[
		"This action disables booting into dev-mode."
	],
	"exec_timeout": {
		"seconds":3600
	},
	"allow_fail_after_recovery": true
},
"is_not_flex_board": {
	"docs": [
		"Verify that device is belong Reven models"
	],
	"exec_extra_args": [
		"string_values:reven",
		"invert_result:true"
	],
	"exec_name":"dut_check_board"
},
"Quick provision OS":{
	"docs":[
		"Install stable OS on the device."
	],
	"conditions": [
		"has_stable_version_cros_image"
	],
	"dependencies":[
		"cros_ssh"
	],
	"exec_name": "cros_provision",
	"exec_timeout": {
		"seconds": 3600
	}
},
"cros_servo_power_reset_repair":{
	"docs":[
		"This repair action will use servod command to reset power_state on the DUT.",
		"TODO: (blocked by: b/221083688) Collect logs from a successfully repaired DUT."
	],
	"exec_timeout": {
		"seconds":200
	},
	"conditions":[
		"servod_echo"
	],
	"dependencies":[
		"servo_power_state_reset",
		"wait_device_to_boot_after_reset"
	],
	"exec_name":"sample_pass"
},
"wait_device_to_boot_after_reset":{
	"docs":[
		"Try to wait device to be sshable after the device being rebooted."
	],
	"exec_timeout": {
		"seconds":150
	},
	"exec_name":"cros_ssh"
},
"Wait DUT to be pingable after reset":{
	"docs":[
		"Wait DUT to be pingable after some action on it."
	],
	"exec_timeout": {
		"seconds":150
	},
	"exec_name":"cros_ping"
},
"Trigger kernel panic to reset the whole board and try ssh to DUT":{
	"docs":[
		"This repair action repairs a Chrome device by sending a system request to the kernel.",
		"TODO: (blocked by: b/221083688) Collect logs from a successfully repaired DUT."
	],
	"conditions":[
		"servod_echo"
	],
	"dependencies":[
		"Trigger kernel panic by servod",
		"wait_device_to_boot_after_reset"
	],
	"exec_name":"sample_pass"
},
"Trigger kernel panic by servod":{
	"docs":[
		"This repair action repairs a Chrome device by sending a system request to the kernel."
	],
	"conditions":[
		"servod_echo"
	],
	"exec_extra_args":[
		"count:3",
		"retry_interval:2"
	],
	"exec_name":"servo_trigger_kernel_panic"
},
"cros_servo_cr50_reboot_repair":{
	"docs":[
		"Repair a Chrome Device by resetting cr50 by servo.",
		"Then, using servo to initialize dut again.",
		"TODO: (blocked by: b/221083688) Collect logs from a successfully repaired DUT."
	],
	"conditions":[
		"dut_servo_host_present",
		"servod_echo",
		"servo_host_is_labstation",
		"servod_has_control_cr50_reboot"
	],
	"dependencies":[
		"servo_power_state_cr50_reset",
		"sleep_1_second",
		"init_dut_for_servo",
		"wait_device_to_boot_after_reset"
	],
	"exec_name": "sample_pass"
},
"servod_has_control_cr50_reboot":{
	"docs":[
		"Checks whether the servod has the command control: cr50_reboot."
	],
	"exec_extra_args": [
		"command:cr50_reboot"
	],
	"exec_name":"servo_check_servod_control"
},
"servo_power_state_cr50_reset":{
	"docs":[
		"Repair a ChromeOS Device by resetting cr50 by servo."
	],
	"exec_extra_args": [
		"command:power_state",
		"string_value:cr50_reset"
	],
	"exec_name":"servo_set",
	"allow_fail_after_recovery": true
},
"sleep_1_second":{
	"exec_extra_args": [
		"sleep:1"
	],
	"exec_name":"sample_sleep"
},
"Read BIOS from DUT by servo":{
	"docs":[
		"Read GBB flags from the DUT by servo.",
		"Set 40 minutes as some FW BIOS is too big and take time to flash it."
	],
	"dependencies":[
		"dut_servo_host_present",
		"servo_state_is_working"
	],
	"exec_name":"cros_read_gbb_by_servo",
	"exec_extra_args":[
		"remove_file:false"
	],
	"exec_timeout": {
		"seconds":2400
	}
},
"Switch DUT to dev mode":{
	"docs":[
		"Force to set GBB flags to 0x18 to boot in DEV mode and enable to boot from USB-drive.",
		"Reboot and wait for device to be back."
	],
	"dependencies":[
		"Set GBB flags to 0x18 by servo",
		"Wait DUT to be pingable after reset"
	],
	"exec_name":"sample_pass"
},
"Set GBB flags to 0x18 by servo":{
	"docs":[
		"Force to set GBB flags to 0x18 to boot in DEV mode and enable to boot from USB-drive.",
		"Set 40 minutes as some FW BIOS is too big and take time to flash it."
	],
	"dependencies":[
		"Read BIOS from DUT by servo"
	],
	"exec_name":"cros_set_gbb_by_servo",
	"exec_extra_args":[
		"gbb_flags:0x18"
	],
	"exec_timeout": {
		"seconds":2400
	}
},
"Power cycle DUT by RPM and wait":{
	"docs":[
		"Perfrom RPM cycle and wait to device to boot back."
	],
	"conditions": [
		"has_rpm_info"
	],
	"dependencies":[
		"Power cycle DUT by RPM",
		"Wait DUT to be pingable after reset"
	],
	"exec_name":"sample_pass",
	"run_control": 1
},
"Is not in audio box":{
	"docs":[
		"Verify that setup is not audio box"
	],
	"dependencies":[
		"dut_is_in_audio_box"
	],
	"exec_name":"sample_fail"
},
"Power cycle DUT by RPM":{
	"docs":[
		"Power cycle the DUT by RPM outlet."
	],
	"conditions": [
		"has_rpm_info"
	],
	"exec_name":"rpm_power_cycle",
	"run_control": 1
},
"Collect dmesg logs from DUT": {
	"docs": [
		"Collect the entire output of dmesg"
	],
	"conditions": [],
	"dependencies": [],
	"exec_name": "cros_dmesg",
	"recovery_actions": [],
	"allow_fail_after_recovery": true
}
`

// Represents the Chrome OS repair plan for DUT.
var crosRepairPlanBody = `
"critical_actions": [` + joinCriticalList(crosRepairPlanCriticalActionList) + `],
"actions": {` + crosRepairPlanActions + `}
`

// joinCriticalList joins the list to make critical actions list for the plan.
//
// The list will be part of json so each item need to wrap by double quotes.
func joinCriticalList(al []string) string {
	var qa []string
	for _, a := range al {
		qa = append(qa, fmt.Sprintf("%q", a))
	}
	return strings.Join(qa, ",")
}
