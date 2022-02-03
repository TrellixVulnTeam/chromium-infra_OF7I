// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package recovery

const wifiRouterRepairPlanBody = `
"critical_actions": [
	"wifirouter_state_broken",
	"cros_ping",
	"cros_ssh",
	"is_wifirouter_tools_present",
	"wifirouter_state_working"
],
"actions": {
	"is_wifirouter_tools_present": {
		"docs": [
			"check whether wifirouter critial tools present: ",
			"tcpdump, hostapd, dnsmasq, netperf, iperf"
		],
		"dependencies": [
			"cros_ssh"
		],
		"exec_name":"cros_is_tool_present",
		"exec_extra_args": [
			"tools:tcpdump,hostapd,dnsmasq,netperf,iperf"
		]
	}
}
`
