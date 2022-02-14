// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package recovery

// List of critical actions for deployment of the ChromeOS.
var crosDeployPlanCriticalActionList = []string{
	"Clean up",
	"Servo has USB-key with require image",
	"DUT is in dev-mode and allowed to boot from USB-key",
	"DUT has expected OS version",
	"DUT has expected test firmware",
	"Collect DUT labels",
	"DUT verify",
}

// List of actions configs for deployment of the ChromeOS.
const crosDeployPlanActions = `
"DUT is in dev-mode and allowed to boot from USB-key":{
	"docs":[
		"Verify that device is set to boot in DEV mode and enabled to boot from USB-drive."
	],
	"exec_timeout": {
		"seconds":2000
	},
	"exec_name":"cros_read_gbb_by_servo",
	"exec_extra_args":[
		"in_dev_mode:true",
		"usb_boot_enabled:true"
	],
	"recovery_actions":[
		"cros_set_gbb_by_servo"
	]
},
"DUT has expected OS version":{
	"docs":[
		"Verify that device has stable version OS on it and version is match."
	],
	"dependencies":[
		"cros_ssh",
		"has_stable_version_cros_image",
		"has_test_cros_image"
	],
	"exec_name":"cros_is_on_stable_version",
	"recovery_actions":[
		"Install OS in DEV mode"
	]
},
"DUT has expected test firmware":{
	"docs":[
		"verify that fw on the DUT is match stable version.",
		"TODO: Need implement",
		"Need verify that FW on the DUT has dev keys."
	],
	"dependencies":[
	],
	"exec_name":"sample_fail"
},
"DUT verify":{
	"docs":[
		"Run all repair critcal actions.",
		"TODO: Need add all critical actions"
	],
	"dependencies":[
	],
	"exec_name":"sample_fail"
},
"has_test_cros_image":{
	"docs":[
		"Verify that device has test OS image on it.",
		"TODO: Need implement"
	],
	"exec_name":"sample_fail"
},
"Install OS in DEV mode":{
	"docs":[
		"Install S on the device from USB-key when device is in DEV-mode.",
		"TODO: Need implement"
	],
	"exec_name":"sample_fail"
},
"Clean up":{
	"docs":[
		"Verify that device is set to boot in DEV mode and enabled to boot from USB-drive."
	],
	"dependencies":[
		"cros_remove_default_ap_file_servo_host"
	],
	"exec_name":"sample_pass"
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
"Collect DUT labels":{
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
},
`

// Represents the Chrome OS deployment plan for DUT.
var crosDeployPlanBody = `"critical_actions": [` + joinCriticalList(crosDeployPlanCriticalActionList) + `],
"actions": {` + crosDeployPlanActions + crosRepairPlanActions + `}`
