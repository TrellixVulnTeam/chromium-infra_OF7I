// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package recovery

const crosDeployPlanBody = `
"critical_actions": [
	"cros_remove_default_ap_file_servo_host",
	"cros_read_gbb_by_servo",
	"cros_ap_is_dev_signed_by_servo",
	"device_labels"
],
"actions": {
	"cros_read_gbb_by_servo":{
		"docs":[
			"Verify that device is set to boot in DEV mode and enabled to boot from USB-drive."
		],
		"exec_timeout": {
			"seconds":2000
		},
		"exec_extra_args":[
			"in_dev_mode:true",
			"usb_boot_enabled:true"
		],
		"recovery_actions":[
			"cros_set_gbb_by_servo"
		]
	},
	"cros_ap_is_dev_signed_by_servo":{
		"docs":[
			"Verify that AP has dev signed firmware."
		],
		"exec_timeout": {
			"seconds":2000
		}
	},
	"cros_set_gbb_by_servo":{
		"docs":[
			"Force to set GBB flags to 0x18 to boot in DEV mode and enable to boot from USB-drive."
		],
		"exec_timeout": {
			"seconds":2000
		},
		"exec_extra_args":[
			"gbb_flags:0x18"
		]
	},
	"device_labels":{
		"dependencies":[
			"device_sku",
			"servo_type_label"
		 ],
		 "exec_name":"sample_pass"
	},
	"servo_type_label":{
		"docs":[
			"Update the servo type label for the DUT info."
		],
		"exec_name":"servo_update_servo_type_label",
		"allow_fail_after_recovery": true
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
	}
}
`
