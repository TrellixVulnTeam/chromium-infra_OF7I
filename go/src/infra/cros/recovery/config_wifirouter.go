// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package recovery

// galeOsName is used as stable version for gale.
// It is used until stableversion tool for ap/pcap is ready
// b:216192539
const galeOsName = "gale-test-ap-tryjob/R92-13982.81.0-b4959409"

const wifiRouterRepairPlanBody = `
"critical_actions": [
	"wifirouter_state_broken",
	"Device is pingable",
	"cros_ssh",
	"Device is on stable-version",
	"is_wifirouter_tools_present",
	"wifirouter_state_working"
],
"actions": {
	"is_wifirouter_tools_present":{
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
	},
	"Device is pingable": {
		"exec_timeout": {
			"seconds":15
		},
		"exec_name": "cros_ping"
	},
	"Device is on stable-version" : {
		"docs":[
			"TODO(b/216192539): extend stable version to peripheral routers",
			"This is intermittent solution for wifirouter until bug is resolved",
			"Currently lab only support one type of router device (board=gale,model=gale)"
		],
		"dependencies":[
			"is_wifirouter_gale_gale"
		],
		"exec_name":"cros_is_on_stable_version",
		"exec_extra_args": [
			"os_name:` + galeOsName + `"
		],
		"recovery_actions":[
			"install_wifirouter_gale_gale_os"
		]
	},
	"is_wifirouter_gale_gale" :{
		"docs": [
			"TODO: hardcoded to only accept model=gale, board=gale routers.",
			"Remove when stable version is ready"
		],
		"exec_name": "is_wifirouter_board_model_matching",
		"exec_extra_args": [
			"board:gale",
			"model:gale"
		]
	},
	"install_wifirouter_gale_gale_os":{
		"docs": [
			"Install wifirouter stable os. Currently only has one version"
		],
		"conditions": [
			"is_wifirouter_gale_gale"
		],
		"exec_name": "cros_provision",
		"exec_extra_args":[
			"os_name:` + galeOsName + `"
		],
		"exec_timeout":{
			"seconds": 3600
		}
	}
}
`
