// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package recovery

import (
	"io"
	"strings"
)

// Default cobfiguration with all planes supported by engine.
// WHen you change or add new plan please be suer that is working
// and predictable.
// TODO(otabek@): Add plan for labstation.
// TODO(vkjoshi@): Add plans for Servo and DUT.
const defaultConfig = `
{
	"plans":{
		"labstation_repair":{
			"critical_actions": [
				"sample_pass"
			],
			"actions": {}
		},
		"servo_repair":{
			"critical_actions": [
				"sample_pass"
			],
			"actions": {},
			"allow_fail": true
		},
		"cros_repair":{
			"critical_actions": [
				"sample_pass"
			],
			"actions": {}
		},
		"labstation_deploy":{
			"critical_actions": [
				"sample_pass"
			],
			"actions": {}
		},
		"cros_deploy":{
			"critical_actions": [
				"sample_pass"
			],
			"actions": {}
		}
	}
}
 `

// DefaultConfig provides default config for recovery engine.
func DefaultConfig() io.Reader {
	return strings.NewReader(defaultConfig)
}
