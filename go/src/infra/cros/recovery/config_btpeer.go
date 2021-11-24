// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package recovery

const btpeerRepairPlanBody = `
"critical_actions": [
	"btpeer_state_broken",
	"cros_ping",
	"cros_ssh",
	"btpeer_state_working"
],
"actions": {}
`
