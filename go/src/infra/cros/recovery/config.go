// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package recovery

import (
	"fmt"
	"io"
	"strings"
)

// DefaultConfig provides default config for recovery engine.
// Plan is planing to be deprecated with new flow.
func DefaultConfig() io.Reader {
	// Do not use set plans as plans created for this configuration on fly for backward compatibility.
	return strings.NewReader(`
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
	}`)
}

// configPlan holders for plans used for configuration.
// Order of the plan specified the execution order.
type configPlan struct {
	name      string
	body      string
	allowFail bool
}

// createConfiguration creates configuration plan based on provided plan data.
func createConfiguration(plans []configPlan) string {
	if len(plans) == 0 {
		return ""
	}
	planNames := ""
	planBodies := ""
	for i, p := range plans {
		if i > 0 {
			planNames += ","
			planBodies += ","
		}
		planNames += fmt.Sprintf("%q", p.name)
		planBodies += fmt.Sprintf(`%q:{%s, "allow_fail":%v}`, p.name, p.body, p.allowFail)
	}

	return fmt.Sprintf(`{"plan_names":[%s],"plans": {%s}}`, planNames, planBodies)
}
