// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package recovery

import (
	"io"
	"strings"
)

// Default cobfiguration with all planes supported by engine.
// WHen you change or add new plan please be sure that is working
// and predictable.
// TODO(otabek@): Add plan for labstation.
// TODO(vkjoshi@): Add plans for Servo and DUT.
const defaultConfig = `
{
	"plans":{
		"labstation_repair":{
			` + labstationRepairPlanBody + `
		},
		"servo_repair":{
			` + servoRepairPlanBody + `
			,"allow_fail": true
		},
		"chameleon_repair":{
			` + chameleonPlanBody + `
			,"allow_fail": true
		},
		"bluetooth_peer_repair":{
			` + btpeerRepairPlanBody + `
			,"allow_fail": true
		},
		"cros_repair":{
			` + crosRepairPlanBody + `
		},
		"labstation_deploy":{
			` + labstationDeployPlanBody + `
		},
		"cros_deploy":{
			` + crosDeployPlanBody + `
		}
	}
}
 `

// DefaultConfig provides default config for recovery engine.
func DefaultConfig() io.Reader {
	return strings.NewReader(defaultConfig)
}
