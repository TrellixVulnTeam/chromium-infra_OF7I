// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package config

func androidPlan() *Plan {
	return &Plan{
		CriticalActions: []string{
			"dut_state_ready",
		},
	}
}

// AndroidRepairConfig provides config for repair android setup in the lab task.
func AndroidRepairConfig() *Configuration {
	return &Configuration{
		PlanNames: []string{
			PlanAndroid,
		},
		Plans: map[string]*Plan{
			PlanAndroid: setAllowFail(androidPlan(), false),
		}}
}

// AndroidDeployConfig provides config for deploy Android setup in the lab task.
func AndroidDeployConfig() *Configuration {
	return AndroidRepairConfig()
}
