// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package recovery

const btpeerRepairPlanBody = `
"critical_actions": [
	"btpeer_state_broken",
	"Device is pingable",
	"cros_ssh",
	"check_server",
	"btpeer_state_working"
],
"actions": {
	"check_server":{
		"docs":[
			"To check if devices is responsive we request not empty list of detected statuses."
		],
		"exec_name":"btpeer_get_detected_statuses"
	},
	"Device is pingable":{
		"exec_timeout": {
			"seconds":15
		},
		"exec_name":"cros_ping"
	}
}
`
