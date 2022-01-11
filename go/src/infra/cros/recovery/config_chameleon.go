// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package recovery

const chameleonPlanBody = `
"critical_actions": [
	"Mark as bad",
	"cros_ping",
	"cros_ssh",
	"Mark as good"
],
"actions": {
	"Mark as bad":{
		"exec_name":"chameleon_state_broken"
	},
	"Mark as good":{
		"exec_name":"chameleon_state_working"
	}
}
`
