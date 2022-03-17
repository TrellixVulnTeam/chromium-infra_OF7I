// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package config

import (
	"fmt"

	"google.golang.org/protobuf/types/known/durationpb"
)

// galeOsName is used as stable version for gale.
// It is used until stableversion tool for ap/pcap is ready
// TODO(b:216192539): need merge it to versioner.
const galeOsName = "gale-test-ap-tryjob/R92-13982.81.0-b4959409"

var osNameArg = fmt.Sprintf("os_name:%s", galeOsName)

func wifiRouterRepairPlan() *Plan {
	return &Plan{
		CriticalActions: []string{
			"wifirouter_state_broken",
			"Device is pingable",
			"cros_ssh",
			"Device is on stable-version",
			"is_wifirouter_tools_present",
			"Device has 50 percent tmp diskspace",
			"Device has 50 percent stateful partition diskspace",
			"wifirouter_state_working",
		},
		Actions: map[string]*Action{
			"is_wifirouter_tools_present": {
				Docs: []string{
					"check whether wifirouter critial tools present: ",
					"tcpdump, hostapd, dnsmasq, netperf, iperf",
				},
				Dependencies:  []string{"cros_ssh"},
				ExecName:      "cros_is_tool_present",
				ExecExtraArgs: []string{"tools:tcpdump,hostapd,dnsmasq,netperf,iperf"},
			},
			"Device is pingable": {
				ExecTimeout: &durationpb.Duration{Seconds: 15},
				ExecName:    "cros_ping",
			},
			"Device is on stable-version": {
				Docs: []string{
					"TODO(b/216192539): extend stable version to peripheral routers",
					"This is intermittent solution for wifirouter until bug is resolved",
					"Currently lab only support one type of router device (board=gale,model=gale)",
				},
				Dependencies:    []string{"is_wifirouter_gale_gale"},
				ExecName:        "cros_is_on_stable_version",
				ExecExtraArgs:   []string{osNameArg},
				RecoveryActions: []string{"install_wifirouter_gale_gale_os"},
			},
			"is_wifirouter_gale_gale": {
				Docs: []string{
					"TODO: hardcoded to only accept model=gale, board=gale routers.",
					"Remove when stable version is ready",
				},
				ExecName: "is_wifirouter_board_model_matching",
				ExecExtraArgs: []string{
					"board:gale",
					"model:gale",
				},
			},
			"install_wifirouter_gale_gale_os": {
				Docs:          []string{"Install wifirouter stable os. Currently only has one version"},
				Conditions:    []string{"is_wifirouter_gale_gale"},
				ExecName:      "cros_provision",
				ExecExtraArgs: []string{osNameArg},
				ExecTimeout:   &durationpb.Duration{Seconds: 3600},
			},
			"Device has 50 percent tmp diskspace": {
				Docs: []string{
					"Check if there are more than 50 percent of diskspace in /tmp",
				},
				Dependencies: []string{"Device is on stable-version"},
				ExecName:     "cros_has_enough_storage_space_percentage",
				ExecExtraArgs: []string{
					"path:/tmp",
					"expected:50"},
				RecoveryActions: []string{
					"Clean up tmp space",
					"install_wifirouter_gale_gale_os"},
			},
			"Clean up tmp space": {
				Docs: []string{
					"Clean up tmp space",
				},
				ExecName: "cros_run_shell_command",
				ExecExtraArgs: []string{
					"rm -Rf /tmp/*",
				},
			},
			"Device has 50 percent stateful partition diskspace": {
				Docs: []string{
					"Check if there are more than 50 percent of diskspace in /mnt/stateful_partition",
				},
				Dependencies: []string{"Device is on stable-version"},
				ExecName:     "cros_has_enough_storage_space_percentage",
				ExecExtraArgs: []string{
					"path:/mnt/stateful_partition",
					"expected:50"},
				RecoveryActions: []string{
					"Clean up stateful sub space",
					"install_wifirouter_gale_gale_os"},
			},
			"Clean up stateful sub space": {
				Docs: []string{
					"Clean up  /mnt/stateful_partition/home/.shadow ,/mnt/stateful_partition/dev_image/telemetry space",
				},
				ExecName: "cros_run_shell_command",
				ExecExtraArgs: []string{
					"rm -Rf /mnt/stateful_partition/home/.shadow /mnt/stateful_partition/dev_image/telemetry",
				},
			},
		},
	}
}
