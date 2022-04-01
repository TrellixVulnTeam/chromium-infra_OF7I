// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package config

import (
	"google.golang.org/protobuf/types/known/durationpb"
)

func crosRepairPlan() *Plan {
	return &Plan{
		CriticalActions: []string{
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
			"Match provision labels",
			"dut_state_ready",
			"device_labels",
			"Collect dmesg logs from DUT",
		},
		Actions: crosRepairActions(),
	}
}

func crosRepairActions() map[string]*Action {
	return map[string]*Action{
		"Device is pingable": {
			Dependencies: []string{
				"dut_has_name",
				"dut_has_board_name",
				"dut_has_model_name",
			},
			RecoveryActions: []string{
				"Restore AC detection by EC console",
				"Update FW from fw-image by servo",
			},
			ExecName: "cros_ping",
			ExecTimeout: &durationpb.Duration{
				Seconds: 15,
			},
		},
		"cros_ssh": {
			Dependencies: []string{
				"dut_has_name",
				"dut_has_board_name",
				"dut_has_model_name",
				"Device is pingable",
			},
			RecoveryActions: []string{
				"cros_servo_power_reset_repair",
				"Trigger kernel panic to reset the whole board and try ssh to DUT",
				"cros_servo_cr50_reboot_repair",
				"Install OS in recovery mode by booting from servo USB-drive",
				"Update FW from fw-image by servo",
			},
		},
		"internal_storage": {
			Dependencies: []string{
				"cros_storage_writing",
				"cros_storage_file_system",
				"cros_storage_space_check",
				"cros_audit_storage_smart",
			},
			ExecName: "sample_pass",
		},
		"device_system_info": {
			Conditions: []string{
				"is_not_flex_board",
			},
			Dependencies: []string{
				"cros_default_boot",
				"cros_boot_in_normal_mode",
				"Match HWID",
				"Match serial-number",
				"cros_tpm_fwver_match",
				"cros_tpm_kernver_match",
			},
			ExecName: "sample_pass",
		},
		"has_python": {
			Docs: []string{
				"Verify that device has python on it.",
				"The Reven boards does not have python. TBD",
			},
			Conditions: []string{
				"is_not_flex_board",
			},
			Dependencies: []string{
				"cros_storage_writing",
			},
			ExecName: "cros_has_python_interpreter_working",
			RecoveryActions: []string{
				"Quick provision OS",
				"Install OS in recovery mode by booting from servo USB-drive",
				"Repair by powerwash",
			},
		},
		"last_provision_successful": {
			Dependencies: []string{
				"cros_storage_writing",
			},
			ExecName: "cros_is_last_provision_successful",
			RecoveryActions: []string{
				"Install OS in recovery mode by booting from servo USB-drive",
				"Quick provision OS",
				"Repair by powerwash",
			},
		},
		"device_enrollment": {
			Dependencies: []string{
				"cros_storage_writing",
			},
			RecoveryActions: []string{
				"tpm_enrollment_cleanup_and_reboot",
			},
			ExecName: "cros_is_enrollment_in_clean_state",
		},
		"tpm_enrollment_cleanup_and_reboot": {
			Docs: []string{
				"Cleanup the enrollment state.",
			},
			Dependencies: []string{
				"cros_ssh",
			},
			ExecExtraArgs: []string{
				"repair_timeout:120",
				"clear_tpm_owner_timeout:60",
				"file_deletion_timeout:120",
				"reboot_timeout:10",
				"tpm_timeout:150",
			},
			ExecTimeout: &durationpb.Duration{Seconds: 600},
			ExecName:    "cros_enrollment_cleanup",
		},
		"power_info": {
			Docs: []string{"Check for the AC power, and battery charging capability."},
			Conditions: []string{
				"is_not_flex_board",
				"cros_is_not_virtual_machine",
			},
			Dependencies: []string{
				"cros_storage_writing",
				"cros_is_ac_power_connected",
				"battery_is_good",
			},
			RecoveryActions: []string{
				"Power cycle DUT by RPM and wait",
				"cros_servo_power_reset_repair",
				"cros_servo_cr50_reboot_repair",
				"Restore AC detection by EC console",
				"Install OS in recovery mode by booting from servo USB-drive",
				"Quick provision OS",
			},
			ExecName: "sample_pass",
		},
		"tpm_info": {
			Conditions: []string{
				"is_not_flex_board",
				"cros_is_not_virtual_machine",
				"cros_is_tpm_present",
			},
			RecoveryActions: []string{
				"Install OS in recovery mode by booting from servo USB-drive",
				"Quick provision OS",
				"Repair by powerwash",
			},
			ExecName: "cros_is_tpm_in_good_status",
		},
		"tools_checks": {
			Dependencies: []string{
				"cros_gsctool",
			},
			ExecName: "sample_pass",
		},
		"hardware_audit": {
			Dependencies: []string{
				"wifi_audit",
				"bluetooth_audit",
			},
			ExecName: "sample_pass",
		},
		"cros is firmware in good state": {
			RecoveryActions: []string{
				"Install OS in recovery mode by booting from servo USB-drive",
				"Quick provision OS",
				"Update FW from fw-image by servo",
			},
			ExecName: "cros_is_firmware_in_good_state",
		},
		"firmware_check": {
			Conditions: []string{
				"is_not_flex_board",
			},
			Dependencies: []string{
				"cros_storage_writing",
				"cros is firmware in good state",
				"cros_rw_firmware_stable_verion",
			},
			ExecName: "sample_pass",
		},
		"stop_start_ui": {
			Docs: []string{
				"Check the command 'stop ui' won't crash the DUT.",
			},
			ExecTimeout: &durationpb.Duration{Seconds: 45},
			RecoveryActions: []string{
				"cros_servo_power_reset_repair",
				"cros_servo_cr50_reboot_repair",
				"Install OS in recovery mode by booting from servo USB-drive",
				"Quick provision OS",
				"Repair by powerwash",
			},
			ExecName: "cros_stop_start_ui",
		},
		"rw_vpd": {
			Docs: []string{
				"Verify that keys: 'should_send_rlz_ping', 'gbind_attribute', 'ubind_attribute' are present in vpd RW_VPD partition.",
			},
			Conditions: []string{
				"is_not_flex_board",
			},
			RecoveryActions: []string{
				"Install OS in recovery mode by booting from servo USB-drive",
				"Quick provision OS",
				"Repair by powerwash",
			},
			ExecName:               "cros_are_required_rw_vpd_keys_present",
			AllowFailAfterRecovery: true,
		},
		"servo_keyboard": {
			Conditions: []string{
				"dut_servo_host_present",
				"servo_state_is_working",
				"is_servo_keyboard_image_tool_present",
			},
			Dependencies: []string{
				"servo_init_usb_keyboard",
				"lufa_keyboard_found",
			},
			ExecName: "cros_run_shell_command",
			ExecExtraArgs: []string{
				"lsusb -vv -d 03eb:2042 |grep \"Remote Wakeup\"",
			},
			AllowFailAfterRecovery: true,
		},
		"servo_mac_address": {
			Conditions: []string{
				"dut_servo_host_present",
				"is_not_servo_v3",
				"servod_control_exist_for_mac_address",
			},
			ExecName:               "servo_audit_nic_mac_address",
			AllowFailAfterRecovery: true,
		},
		"is_not_servo_v3": {
			Conditions: []string{
				"is_servo_v3",
			},
			ExecName: "sample_fail",
		},
		"servod_control_exist_for_mac_address": {
			ExecName: "servo_check_servod_control",
			ExecExtraArgs: []string{
				"command:macaddr",
			},
		},
		"servo_init_usb_keyboard": {
			Docs: []string{
				"set servo's 'init_usb_keyboard' command to 'on' value.",
			},
			Dependencies: []string{
				"dut_servo_host_present",
			},
			ExecName: "servo_set",
			ExecExtraArgs: []string{
				"command:init_usb_keyboard",
				"string_value:on",
			},
		},
		"is_servo_keyboard_image_tool_present": {
			Docs: []string{
				"check if the servo keyboard image specified by the name of dfu-programmer can be found in DUT cli.",
			},
			Dependencies: []string{
				"dut_servo_host_present",
			},
			ExecName: "cros_is_tool_present",
			ExecExtraArgs: []string{
				"tools:dfu-programmer",
			},
		},
		"lufa_keyboard_found": {
			Docs: []string{
				"check if the lufa keyboard can be found by finding the match of the model information of it.",
			},
			ExecName: "cros_run_shell_command",
			ExecExtraArgs: []string{
				"lsusb -d 03eb:2042 |grep \"LUFA Keyboard Demo\"",
			},
		},
		"servo_state_is_working": {
			Docs: []string{
				"check the servo's state is WORKING.",
			},
			Dependencies: []string{
				"dut_servo_host_present",
			},
			ExecName: "servo_match_state",
			ExecExtraArgs: []string{
				"state:WORKING",
			},
		},
		"servo_state_is_not_working": {
			Docs: []string{
				"check the servo's state is not WORKING.",
			},
			Conditions: []string{
				"servo_state_is_working",
			},
			ExecName: "sample_fail",
		},
		"cros_rw_firmware_stable_verion": {
			Dependencies: []string{
				"cros_storage_writing",
				"cros_is_on_rw_firmware_stable_verion",
				"cros_is_rw_firmware_stable_version_available",
			},
			ExecName: "sample_pass",
		},
		"cros_gsctool": {
			Docs: []string{
				"Confirm that the GSC tool is function. This action ",
				"has been created so that it can be used to attach a ",
				"recovery action if the GSC tool is not functional on ",
				"the DUT.",
			},
			RecoveryActions: []string{
				"Install OS in recovery mode by booting from servo USB-drive",
				"Quick provision OS",
				"Repair by powerwash",
			},
			ExecName: "cros_is_gsc_tool_present",
		},
		"battery_is_good": {
			Docs: []string{
				"Check battery on the DUT is normal and update battery hardware state accordingly.",
			},
			Conditions: []string{
				"cros_is_battery_expected",
				"cros_is_not_virtual_machine",
				"Battery is expected on device",
				"Battery is present on device",
			},
			Dependencies: []string{
				"cros_storage_writing",
				"cros_is_battery_chargable_or_good_level",
			},
			ExecName: "cros_audit_battery",
		},
		"Battery is expected on device": {
			Docs: []string{
				"Verifies that device is expected to have battery based on DUT info.",
			},
			ExecName: "dut_has_battery",
		},
		"Battery is present on device": {
			Docs: []string{
				"Verifies if battery present is reported as present in power supply info.",
			},
			ExecName:   "cros_is_battery_present",
			RunControl: 1,
		},
		"No Battery is present on device": {
			Conditions: []string{
				"Battery is present on device",
			},
			ExecName: "sample_fail",
		},
		"wifi_audit": {
			Docs: []string{
				"Check wifi on the DUT is normal and update wifi hardware state accordingly.",
			},
			Dependencies: []string{
				"cros_ssh",
			},
			ExecName:               "cros_audit_wifi",
			AllowFailAfterRecovery: true,
		},
		"bluetooth_audit": {
			Docs: []string{
				"Check bluetooth on the DUT is normal and update bluetooth hardware state accordingly.",
			},
			Dependencies: []string{
				"cros_ssh",
			},
			ExecName:               "cros_audit_bluetooth",
			AllowFailAfterRecovery: true,
		},
		"cros_tpm_fwver_match": {
			Dependencies: []string{
				"cros_storage_writing",
			},
			Conditions: []string{
				"is_not_flex_board",
			},
			RecoveryActions: []string{
				"Install OS in recovery mode by booting from servo USB-drive",
			},
			ExecName: "cros_match_dev_tpm_firmware_version",
		},
		"cros_tpm_kernver_match": {
			Dependencies: []string{
				"cros_storage_writing",
			},
			Conditions: []string{
				"is_not_flex_board",
			},
			RecoveryActions: []string{
				"Install OS in recovery mode by booting from servo USB-drive",
			},
			ExecName: "cros_match_dev_tpm_kernel_version",
		},
		"cros_default_boot": {
			Docs: []string{
				"Check if the default boot drive is disk.",
			},
			Dependencies: []string{
				"cros_storage_writing",
			},
			Conditions: []string{
				"is_not_flex_board",
			},
			RecoveryActions: []string{
				"Set default boot as disk",
				"Quick provision OS",
				"Repair by powerwash",
				"Install OS in recovery mode by booting from servo USB-drive",
			},
			ExecName: "cros_is_default_boot_from_disk",
		},
		"cros_boot_in_normal_mode": {
			Conditions: []string{
				"is_not_flex_board",
				"Pools required to be in Secure mode",
			},
			Dependencies: []string{
				"cros_storage_writing",
			},
			RecoveryActions: []string{
				"Switch to secure-mode and reboot",
			},
			ExecName: "cros_is_not_in_dev_mode",
		},
		"Match HWID": {
			Conditions: []string{
				"is_not_flex_board",
				"Not Satlab device",
				"Is HWID known",
			},
			Dependencies: []string{
				"cros_storage_writing",
			},
			ExecName: "cros_match_hwid_to_inventory",
		},
		"Match serial-number": {
			Conditions: []string{
				"is_not_flex_board",
				"Not Satlab device",
				"Is serial-number known",
			},
			Dependencies: []string{
				"cros_storage_writing",
			},
			ExecName: "cros_match_serial_number_inventory",
		},
		"Is HWID known": {
			Docs: []string{
				"Check whether the DUT information includes its HWID.",
			},
			ExecName: "dut_has_hwid",
		},
		"Is serial-number known": {
			Docs: []string{
				"Check whether the DUT information includes its ",
				"serial number.",
			},
			ExecName: "dut_has_serial_number",
		},
		"Not Satlab device": {
			Docs: []string{
				"Verify that DUT name is not belong Satlab.",
			},
			Conditions: []string{
				"Is Satlab device",
			},
			ExecName: "sample_fail",
		},
		"Is Satlab device": {
			Docs: []string{
				"Verify that DUT name is belong Satlab.",
			},
			ExecName: "dut_regex_name_match",
			ExecExtraArgs: []string{
				"regex:^satlab",
			},
		},
		"Read DUT serial-number from DUT": {
			Conditions: []string{
				"Not Satlab device",
			},
			ExecName: "cros_update_hwid_to_inventory",
		},
		"Read DUT serial-number from DUT (Satlab)": {
			Conditions: []string{
				"Is Satlab device",
			},
			ExecName:               "cros_update_hwid_to_inventory",
			AllowFailAfterRecovery: true,
		},
		"Read HWID from DUT": {
			Conditions: []string{
				"Not Satlab device",
			},
			ExecName: "cros_update_hwid_to_inventory",
		},
		"Read HWID from DUT (Satlab)": {
			Conditions: []string{
				"Is Satlab device",
			},
			ExecName:               "cros_update_hwid_to_inventory",
			AllowFailAfterRecovery: true,
		},
		"cros_storage_writing": {
			Dependencies: []string{
				"cros_ssh",
			},
			RecoveryActions: []string{
				"Switch to secure-mode and reboot",
				"Install OS in recovery mode by booting from servo USB-drive",
				"Quick provision OS",
				"Repair by powerwash",
			},
			ExecName: "cros_is_file_system_writable",
		},
		"cros_storage_file_system": {
			Dependencies: []string{
				"cros_ssh",
			},
			RecoveryActions: []string{
				"Install OS in recovery mode by booting from servo USB-drive",
				"Quick provision OS",
				"Repair by powerwash",
			},
			ExecName: "cros_has_critical_kernel_error",
		},
		"cros_storage_space_check": {
			Dependencies: []string{
				"cros_stateful_partition_has_enough_inodes",
				"cros_stateful_partition_has_enough_storage_space",
				"cros_encrypted_stateful_partition_has_enough_storage_space",
			},
			ExecName: "sample_pass",
		},
		"cros_stateful_partition_has_enough_inodes": {
			Docs: []string{
				"check the stateful partition path has enough inodes",
			},
			ExecName: "cros_has_enough_inodes",
			ExecExtraArgs: []string{
				"/mnt/stateful_partition:100",
			},
		},
		"cros_stateful_partition_has_enough_storage_space": {
			Docs: []string{
				"check the stateful partition have enough disk space. The storage unit is in GB.",
			},
			ExecName: "cros_has_enough_storage_space",
			ExecExtraArgs: []string{
				"/mnt/stateful_partition:0.7",
			},
		},
		"cros_encrypted_stateful_partition_has_enough_storage_space": {
			Docs: []string{
				"check the encrypted stateful partition have enough disk space. The storage unit is in GB.",
			},
			ExecName: "cros_has_enough_storage_space",
			ExecExtraArgs: []string{
				"/mnt/stateful_partition/encrypted:0.1",
			},
		},
		"device_labels": {
			Dependencies: []string{
				"device_sku",
				"cr50_labels",
				"audio_loop_back_label",
			},
			ExecName: "sample_pass",
		},
		"audio_loop_back_label": {
			Docs: []string{
				"Update the audio_loop_back label on the cros Device.",
			},
			Conditions: []string{
				"dut_audio_loop_back_state_not_working",
			},
			ExecName:               "cros_update_audio_loopback_state_label",
			AllowFailAfterRecovery: true,
		},
		"dut_audio_loop_back_state_not_working": {
			Docs: []string{
				"Confirm that the DUT's audio loopback state is in not working state",
			},
			Conditions: []string{
				"cros_is_audio_loopback_state_working",
			},
			ExecName: "sample_fail",
		},
		"cr50_labels": {
			Docs: []string{
				"Update the cr50 label on the cros Device.",
			},
			Conditions: []string{
				"cros_is_cr50_firmware_exist",
			},
			Dependencies: []string{
				"cros_update_cr50_label",
				"cros_update_cr50_key_id_label",
			},
			ExecName:               "sample_pass",
			AllowFailAfterRecovery: true,
		},
		"cros_is_cr50_firmware_exist": {
			Docs: []string{
				"Checks if the cr 50 firmware exists on the DUT by running the gsctool version command.",
			},
			ExecName: "cros_run_shell_command",
			ExecExtraArgs: []string{
				"gsctool -a -f",
			},
		},
		"device_sku": {
			Docs: []string{
				"Update the device_sku label from the device if not present in inventory data.",
			},
			Conditions: []string{
				"dut_does_not_have_device_sku",
			},
			ExecName:               "cros_update_device_sku",
			AllowFailAfterRecovery: true,
		},
		"dut_does_not_have_device_sku": {
			Docs: []string{
				"Confirm that the DUT itself does not have device_sku label.",
			},
			Conditions: []string{
				"dut_has_device_sku",
			},
			ExecName: "sample_fail",
		},
		"Servo USB-Key needs to be reflashed": {
			Docs: []string{
				"Check if it is time to download image to servo usbkey.",
				"If so, then download the stable image to usbkey.",
			},
			Conditions: []string{
				"cros_is_time_to_force_download_image_to_usbkey",
			},
			Dependencies: []string{
				"Download stable image to USB-key",
			},
			ExecName: "sample_pass",
		},
		"Servo has USB-key with require image": {
			Docs: []string{
				"USB-drive contains stable image on it.",
			},
			Conditions: []string{
				"dut_servo_host_present",
				"servo_state_is_working",
			},
			ExecName:    "servo_usbkey_has_stable_image",
			ExecTimeout: &durationpb.Duration{Seconds: 120},
			RecoveryActions: []string{
				"Download stable image to USB-key",
			},
		},
		"Download stable image to USB-key": {
			Docs: []string{
				"Download lab stable image on servo USB-key",
				"Download the image can take longer if labstation download parallel a few images.",
			},
			Dependencies: []string{
				"dut_servo_host_present",
				"servo_state_is_working",
			},
			ExecName:    "servo_download_image_to_usb",
			ExecTimeout: &durationpb.Duration{Seconds: 3000},
			RunControl:  1,
		},
		"cros_is_time_to_force_download_image_to_usbkey": {
			Docs: []string{
				"Check if it is time to force download image to usbkey",
				"from the number of failed recoveries since last successful PARIS repair task.",
			},
			ExecExtraArgs: []string{
				"task_name:recovery",
				"repair_failed_count:1",
				"repair_failed_interval:10",
			},
		},
		"Match provision labels": {
			Docs: []string{
				"Verify that provision labels is correct.",
			},
			Dependencies: []string{
				"cros_match_cros_version_to_inventory",
				"cros_match_job_repo_url_version_to_inventory",
			},
			ExecName: "sample_pass",
		},
		"cros_match_cros_version_to_inventory": {
			Docs: []string{
				"Verify that cros-version match version on the host.",
			},
			Dependencies: []string{
				"cros_ssh",
			},
			RecoveryActions: []string{
				"cros_provisioning_labels_repair",
			},
		},
		"cros_match_job_repo_url_version_to_inventory": {
			Docs: []string{
				"Verify that job_repo_url matches the version on the host.",
			},
			Dependencies: []string{
				"cros_ssh",
			},
			RecoveryActions: []string{
				"cros_provisioning_labels_repair",
			},
		},
		"cros_provisioning_labels_repair": {
			Docs: []string{
				"Cleanup the labels and job-repo-url.",
			},
			Dependencies: []string{
				"cros_update_provision_os_version",
				"cros_update_job_repo_url",
			},
			ExecName: "sample_pass",
		},
		"Switch to secure-mode and reboot": {
			Docs: []string{
				"This repair action utilizes the dependent actions to set the",
				" GBB flags and disable booting into dev-mode. Then it reboots",
				" the DUT.",
			},
			Conditions: []string{"Pools required to be in Secure mode"},
			Dependencies: []string{
				"Reset GBB flags by host",
				"cros_switch_to_secure_mode",
				"Simple reboot",
				// Waiting to boot to tell if switch was successful.
				"Wait DUT to be pingable after reset",
			},
			ExecName: "sample_pass",
		},
		"Reset GBB flags by host": {
			Docs: []string{
				"This action sets the GBB flags to 0x0.",
			},
			ExecName:               "cros_set_gbb_flags",
			ExecTimeout:            &durationpb.Duration{Seconds: 3600},
			AllowFailAfterRecovery: true,
		},
		"cros_switch_to_secure_mode": {
			Docs: []string{
				"This action disables booting into dev-mode.",
			},
			ExecTimeout:            &durationpb.Duration{Seconds: 3600},
			AllowFailAfterRecovery: true,
		},
		"is_not_flex_board": {
			Docs: []string{
				"Verify that device is belong Reven models",
			},
			ExecExtraArgs: []string{
				"string_values:reven",
				"invert_result:true",
			},
			ExecName: "dut_check_board",
		},
		"Quick provision OS": {
			Docs: []string{
				"Install stable OS on the device.",
			},
			Conditions: []string{
				"has_stable_version_cros_image",
			},
			Dependencies: []string{
				"cros_ssh",
			},
			ExecName:    "cros_provision",
			ExecTimeout: &durationpb.Duration{Seconds: 3600},
		},
		"cros_servo_power_reset_repair": {
			Docs: []string{
				"This repair action will use servod command to reset power_state on the DUT.",
				"TODO: (blocked by: b/221083688) Collect logs from a successfully repaired DUT.",
			},
			ExecTimeout: &durationpb.Duration{Seconds: 200},
			Conditions: []string{
				"servod_echo",
			},
			Dependencies: []string{
				"servo_power_state_reset",
				"Wait DUT to be SSHable after reset",
			},
			ExecName: "sample_pass",
		},
		"Wait DUT to be SSHable after reset": {
			Docs: []string{
				"Try to wait device to be sshable after the device being rebooted.",
			},
			ExecTimeout: &durationpb.Duration{Seconds: 150},
			ExecName:    "cros_ssh",
		},
		"Wait DUT to be pingable after reset": {
			Docs: []string{
				"Wait DUT to be pingable after some action on it.",
			},
			ExecTimeout: &durationpb.Duration{Seconds: 150},
			RecoveryActions: []string{
				"Install OS in recovery mode by booting from servo USB-drive",
				"Repair by powerwash",
			},
			ExecName: "cros_ping",
		},
		"Trigger kernel panic to reset the whole board and try ssh to DUT": {
			Docs: []string{
				"This repair action repairs a Chrome device by sending a system request to the kernel.",
				"TODO: (blocked by: b/221083688) Collect logs from a successfully repaired DUT.",
			},
			Conditions: []string{
				"servod_echo",
			},
			Dependencies: []string{
				"Trigger kernel panic by servod",
				"Wait DUT to be SSHable after reset",
			},
			ExecName: "sample_pass",
		},
		"Trigger kernel panic by servod": {
			Docs: []string{
				"This repair action repairs a Chrome device by sending a system request to the kernel.",
			},
			Conditions: []string{
				"servod_echo",
			},
			ExecExtraArgs: []string{
				"count:3",
				"retry_interval:2",
			},
			ExecName: "servo_trigger_kernel_panic",
		},
		"cros_servo_cr50_reboot_repair": {
			Docs: []string{
				"Repair a Chrome Device by resetting cr50 by servo.",
				"Then, using servo to initialize dut again.",
				"TODO: (blocked by: b/221083688) Collect logs from a successfully repaired DUT.",
			},
			Conditions: []string{
				"dut_servo_host_present",
				"servod_echo",
				"servo_host_is_labstation",
				"servod_has_control_cr50_reboot",
			},
			Dependencies: []string{
				"servo_power_state_cr50_reset",
				"sleep_1_second",
				"init_dut_for_servo",
				"Wait DUT to be SSHable after reset",
			},
			ExecName: "sample_pass",
		},
		"servod_has_control_cr50_reboot": {
			Docs: []string{
				"Checks whether the servod has the command control: cr50_reboot.",
			},
			ExecExtraArgs: []string{
				"command:cr50_reboot",
			},
			ExecName: "servo_check_servod_control",
		},
		"servo_power_state_cr50_reset": {
			Docs: []string{
				"Repair a ChromeOS Device by resetting cr50 by servo.",
			},
			ExecExtraArgs: []string{
				"command:power_state",
				"string_value:cr50_reset",
			},
			ExecName:               "servo_set",
			AllowFailAfterRecovery: true,
		},
		"sleep_1_second": {
			ExecExtraArgs: []string{
				"sleep:1",
			},
			ExecName: "sample_sleep",
		},
		"Read BIOS from DUT by servo": {
			Docs: []string{
				"Read GBB flags from the DUT by servo.",
				"Set 40 minutes as some FW BIOS is too big and take time to flash it.",
			},
			Dependencies: []string{
				"dut_servo_host_present",
				"servo_state_is_working",
			},
			ExecName: "cros_read_gbb_by_servo",
			ExecExtraArgs: []string{
				"remove_file:false",
			},
			ExecTimeout: &durationpb.Duration{Seconds: 2400},
		},
		"Switch DUT to dev mode": {
			Docs: []string{
				"Force to set GBB flags to 0x18 to boot in DEV mode and enable to boot from USB-drive.",
				"Reboot and wait for device to be back.",
			},
			Dependencies: []string{
				"Set GBB flags to 0x18 by servo",
				"Wait DUT to be pingable after reset",
			},
			ExecName: "sample_pass",
		},
		"Set GBB flags to 0x18 by servo": {
			Docs: []string{
				"Force to set GBB flags to 0x18 to boot in DEV mode and enable to boot from USB-drive.",
				"Set 40 minutes as some FW BIOS is too big and take time to flash it.",
				"Allowed to fail as flags can applied but fail by some reason",
			},
			Dependencies: []string{
				"Read BIOS from DUT by servo",
			},
			ExecName: "cros_set_gbb_by_servo",
			ExecExtraArgs: []string{
				"gbb_flags:0x18",
			},
			ExecTimeout:            &durationpb.Duration{Seconds: 2400},
			AllowFailAfterRecovery: true,
		},
		"Power cycle DUT by RPM and wait": {
			Docs: []string{
				"Perfrom RPM cycle and wait to device to boot back.",
			},
			Conditions: []string{
				"has_rpm_info",
			},
			Dependencies: []string{
				"Power cycle DUT by RPM",
				"Wait DUT to be pingable after reset",
			},
			ExecName:   "sample_pass",
			RunControl: 1,
		},
		"Is not in audio box": {
			Docs: []string{
				"Verify that setup is not audio box",
			},
			Conditions: []string{
				"dut_is_in_audio_box",
			},
			ExecName: "sample_fail",
		},
		"Power cycle DUT by RPM": {
			Docs: []string{
				"Power cycle the DUT by RPM outlet.",
			},
			Conditions: []string{
				"has_rpm_info",
			},
			ExecName:   "rpm_power_cycle",
			RunControl: 1,
		},
		"Collect dmesg logs from DUT": {
			Docs: []string{
				"Collect the entire output of dmesg",
			},
			Conditions:             []string{},
			Dependencies:           []string{},
			ExecName:               "cros_dmesg",
			RecoveryActions:        []string{},
			AllowFailAfterRecovery: true,
		},
		"Restore AC detection by EC console": {
			Docs: []string{
				"Try to recover AC detection through servod's ec control",
			},
			Conditions: []string{
				"cros_is_battery_expected",
			},
			ExecExtraArgs: []string{
				"wait_timeout:120",
			},
			ExecTimeout: &durationpb.Duration{Seconds: 600},
			ExecName:    "servo_recover_ac_power",
		},
		"Disable software-controlled write-protect for 'host'": {
			Docs:                   []string{"Disable write-protect fprom 'host'."},
			Dependencies:           []string{"cros_ssh"},
			ExecName:               "cros_disable_fprom_write_protect",
			ExecExtraArgs:          []string{"fprom:host"},
			ExecTimeout:            &durationpb.Duration{Seconds: 300},
			AllowFailAfterRecovery: true,
			RunControl:             1,
		},
		"Disable software-controlled write-protect for 'ec'": {
			Docs:                   []string{"Disable write-protect fprom 'ec'."},
			Dependencies:           []string{"cros_ssh"},
			ExecName:               "cros_disable_fprom_write_protect",
			ExecExtraArgs:          []string{"fprom:ec"},
			ExecTimeout:            &durationpb.Duration{Seconds: 300},
			AllowFailAfterRecovery: true,
			RunControl:             1,
		},
		"Set needs_deploy state": {
			ExecName: "dut_state_needs_deploy",
		},
		"Install OS in recovery mode by booting from servo USB-drive": {
			Docs: []string{
				"This action installs the test image on DUT utilizing ",
				"the features of servo. DUT will be booted in recovery ",
				"mode. In some cases RO FW is not allowed to boot in ",
				"reovery mode with active PD, so we will change it to ",
				"sink-mode if required.",
			},
			Conditions: []string{
				"Pools required to be in Secure mode",
			},
			Dependencies: []string{
				"Servo USB-Key needs to be reflashed",
				"Servo has USB-key with require image",
				"cros_update_provision_os_version",
			},
			ExecName:      "os_install_repair",
			ExecExtraArgs: []string{"halt_timeout:120"},
			ExecTimeout:   &durationpb.Duration{Seconds: 3600},
		},
		"Install OS in dev mode by booting from servo USB-drive": {
			Docs: []string{
				"This action installs the test image on DUT after ",
				"booking the DUT in dev mode.",
			},
			Conditions: []string{
				"Pools allowed to stay in DEV mode",
			},
			Dependencies: []string{
				"Boot DUT from USB in DEV mode",
				"Device booted from USB-drive",
				"Run install after boot from USB-drive",
				"Cold reset DUT by servo and wait to boot",
				"Wait DUT to be SSHable after reset",
			},
			ExecName: "sample_pass",
		},
		"Cold reset DUT by servo and wait to boot": {
			Docs: []string{"Cold reset device by servo and wait for DUT to become ping-able."},
			Dependencies: []string{
				"dut_servo_host_present",
				"servo_state_is_working",
				"Cold reset DUT by servo",
				"Wait DUT to be pingable after reset",
			},
			ExecName:   "sample_pass",
			RunControl: 1,
		},
		"Cold reset DUT by servo": {
			Docs: []string{"Cold reset device by servo and do not wait."},
			Dependencies: []string{
				"dut_servo_host_present",
				"servo_state_is_working",
			},
			ExecName: "servo_set",
			ExecExtraArgs: []string{
				"command:power_state",
				"string_value:reset",
			},
			RunControl: 1,
		},
		"Pools allowed to stay in DEV mode": {
			Docs: []string{
				"Verify that pools are allowed to stay in DEV mode.",
			},
			ExecName: "dut_is_in_pool",
			ExecExtraArgs: []string{
				"crouton",
				"faft-test",
				"faft-test-au",
				"faft-test-tot",
				"nyc-meet-lab",
				"satlab_faft",
			},
		},
		"Pools required to be in Secure mode": {
			Docs: []string{
				"Verify that DUT need to be in Secure mode.",
			},
			Conditions: []string{
				"Pools allowed to stay in DEV mode",
			},
			ExecName: "sample_fail",
		},
		"Set default boot as disk and reboot": {
			Docs: []string{
				"Set default boot from disk and reboot.",
			},
			Dependencies: []string{
				"Set default boot as disk",
				"Simple reboot",
			},
			ExecName:   "sample_pass",
			RunControl: 1,
		},
		"Simple reboot": {
			Docs: []string{
				"Simple un-blocker reboot.",
			},
			ExecName: "cros_run_shell_command",
			ExecExtraArgs: []string{
				"reboot && exit",
			},
			RunControl: 1,
		},
		"Set default boot as disk": {
			Docs: []string{
				"Set default boot from disk by crossystem.",
			},
			ExecExtraArgs: []string{
				"command:dev_default_boot",
				"value:disk",
				"check_after_update:true",
			},
			ExecName: "cros_update_crossystem",
		},
		"Device NOT booted from USB-drive": {
			Docs: []string{
				"Verify that device was not booted from USB-drive.",
			},
			Conditions: []string{"Device booted from USB-drive"},
			RecoveryActions: []string{
				"Set default boot as disk and reboot",
			},
			ExecName:   "sample_fail",
			RunControl: 1,
		},
		"Device booted from USB-drive": {
			Docs: []string{
				"Verify that device was booted from USB-drive.",
			},
			ExecName:   "cros_booted_from_external_storage",
			RunControl: 1,
		},
		"Write factory-install-reset to file system": {
			ExecName: "cros_run_shell_command",
			ExecExtraArgs: []string{
				"echo \"fast safe\" > /mnt/stateful_partition/factory_install_reset",
			},
			AllowFailAfterRecovery: true,
		},
		"Repair by powerwash": {
			Docs: []string{
				"Install the stable test image designated for the DUT.",
			},
			Dependencies: []string{
				"Write factory-install-reset to file system",
				"Quick provision OS",
			},
			ExecName: "sample_pass",
		},
		"Update FW from fw-image by servo": {
			Docs: []string{
				"Download fw-image specified in stable version and flash EC/AP to the DUT by servo",
				"Set timeout for 100 minutes for now as = 10m(download)+ 7*3m(extraction-file)+10m(ec-update)+30m(ap-update).",
				"The time will be updated later based on collected metrics",
				"Each operation with extraction files can take up to a few minutes.",
				"Ap update on the DUT can take up to 30 minutes",
			},
			Conditions: []string{
				"dut_servo_host_present",
				"servo_state_is_working",
			},
			Dependencies: []string{
				"has_stable_version_fw_version",
			},
			ExecName: "cros_update_fw_with_fw_image_by_servo_from",
			ExecExtraArgs: []string{
				"update_ec:true",
				"update_ap:true",
				"download_timeout:600",
			},
			ExecTimeout: &durationpb.Duration{
				Seconds: 6000,
			},
		},
		"Boot DUT from USB in DEV mode": {
			Docs: []string{
				"Restart and try to boot from USB-drive",
				"First boot in dev mode can take time so set boot time to 10 minutes.",
			},
			ExecName: "cros_dev_mode_boot_from_servo_usb_drive",
			ExecExtraArgs: []string{
				"boot_timeout:600",
				"retry_interval:2",
			},
			ExecTimeout: &durationpb.Duration{Seconds: 900},
		},
		"Run install after boot from USB-drive": {
			Docs:        []string{"Perform install process"},
			ExecName:    "cros_run_chromeos_install_command_after_boot_usbdrive",
			ExecTimeout: &durationpb.Duration{Seconds: 1200},
		},
		"Perfrom RPM config verification": {
			Docs: []string{
				"Verify if RPM verification is required fo setup",
				"Setup with PD control temprarely excluded from testing.",
			},
			Conditions: []string{
				"dut_servo_host_present",
				"has_rpm_info",
				"servo_state_is_working",
				"Setup does't have Servo PD control",
			},
			ExecName: "sample_pass",
		},
		"Setup has Servo PD control": {
			Docs: []string{
				"Verify that servo has build in PD control.",
			},
			ExecName: "servo_build_in_pd_present",
		},
		"Setup does't have Servo PD control": {
			Docs: []string{
				"Verify that servo does not have build in PD control.",
			},
			Conditions: []string{
				"Setup has Servo PD control",
			},
			ExecName: "sample_fail",
		},
		"Verify RPM config (without battery)": {
			Docs: []string{
				"Verify RPM configs and set RPM state",
				"Not applicable for cr50 servos based on b/205728276",
				"Action is not critical as it updates own state.",
			},
			Conditions: []string{
				"Perfrom RPM config verification",
				"No Battery is present on device",
			},
			ExecName:               "rpm_audit_without_battery",
			ExecTimeout:            &durationpb.Duration{Seconds: 600},
			AllowFailAfterRecovery: true,
		},
		"Verify RPM config with battery": {
			Docs: []string{
				"Verify RPM when battery is present",
				"Not applicable for cr50 servos based on b/205728276",
				"Action is not critical as it updates own state.",
			},
			Conditions: []string{
				"Perfrom RPM config verification",
				"Battery is present on device",
			},
			ExecName:    "rpm_audit_with_battery",
			ExecTimeout: &durationpb.Duration{Seconds: 600},
			ExecExtraArgs: []string{
				"timeout:120",
				"wait_interval:5",
			},
			AllowFailAfterRecovery: true,
		},
	}
}
