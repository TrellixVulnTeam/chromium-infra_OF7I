// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package recovery

const wifiRouterRepairPlanBody = `
"critical_actions": [
	"wifirouter_state_broken",
	"cros_ping",
	"cros_ssh",
	"wifirouter_state_working"
]
`
