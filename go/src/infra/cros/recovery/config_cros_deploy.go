// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package recovery

const crosDeployPlanBody = `
"critical_actions": [
	"device_labels"
],
"actions": {
	"device_labels":{
		"dependencies":[
			"device_sku"
		 ],
		 "exec_name":"sample_pass"
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
			"has_dut_device_sku"
		],
		"exec_name":"sample_fail"
	}
}
`
